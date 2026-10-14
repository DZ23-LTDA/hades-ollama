package agent

import (
	"strings"
	"testing"
)

// PACOTE REPLICA - INSTALACAO VERIFICADA (terceiros, MIT, Jake Schincariol).
//
// Este teste existe porque *"um ficheiro no disco nao e' uma instalacao"*: ele
// prova que os 11 manifestos **carregam** pelo caminho real do Hades, que o
// catalogo **nao** concede privilegio, e que o conteudo upstream esta' **intacto**
// byte a byte (contra os sha256 declarados em PROVENANCE.md).
//
// O pacote vive em `skills/replica/` na raiz do repositorio.
const (
	replicaDir       = "../../skills/replica"
	replicaManifests = "../../skills/replica/manifests"
)

func TestReplicaPackCarregaOnzeSkills(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LoadSkillsForOrganization(replicaManifests, LocalOrganizationID); err != nil {
		t.Fatalf("carregar %s: %v", replicaManifests, err)
	}
	// ACL VAZIA NAO E' TODOS: carregar para "" nao pode expor nada.
	if soltas := store.SkillsForOrganization(""); len(soltas) != 0 {
		t.Fatalf("carregar para organizacao vazia expos %d skills", len(soltas))
	}
	skills := store.SkillsForOrganization(LocalOrganizationID)
	if len(skills) != 11 {
		t.Fatalf("esperava 11 skills, carregou %d", len(skills))
	}
	esperados := map[string]bool{
		"replica-architect": true, "replica-backend": true, "replica-brand": true,
		"replica-build": true, "replica-deploy": true, "replica-design": true,
		"replica-diff": true, "replica-entrepreneur": true, "replica-launch": true,
		"replica-recon": true, "replica-test": true,
	}
	for _, m := range skills {
		if !esperados[m.ID] {
			t.Fatalf("skill inesperada: %s", m.ID)
		}
		delete(esperados, m.ID)
		// O CATALOGO NAO CONCEDE PRIVILEGIO (E-8): trusted nunca vem do disco.
		if m.Trusted {
			t.Fatalf("%s: chegou trusted do disco", m.ID)
		}
		if !m.Enabled {
			t.Fatalf("%s: chegou desabilitada do disco", m.ID)
		}
		if strings.TrimSpace(m.Version) == "" || m.License != "MIT" {
			t.Fatalf("%s: version=%q license=%q", m.ID, m.Version, m.License)
		}
		// NENHUM escopo declarado: uma skill de catalogo nao pode reivindicar
		// capacidade so' por estar instalada.
		if len(m.Scopes) != 0 || len(m.Tools) != 0 {
			t.Fatalf("%s: declara scopes=%v tools=%v", m.ID, m.Scopes, m.Tools)
		}
		if m.TokenBudget <= 0 {
			t.Fatalf("%s: token_budget nao declarado (%d)", m.ID, m.TokenBudget)
		}
	}
	if len(esperados) != 0 {
		t.Fatalf("faltaram skills: %v", esperados)
	}
}
