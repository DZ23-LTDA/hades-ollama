package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// HISTÓRICO DE VERSÕES DE SKILL (E-629).
//
// O QUE ESTE TESTE IMPEDE: que registar uma versão nova continue a APAGAR a anterior. O store
// guardava uma versão por ID (`s.skills[id]`), e a perda era silenciosa — só se descobria quando
// alguém precisava da versão antiga.
//
// O teste olha para o DISCO, e não para memória: um histórico que não sobrevive ao processo não é
// histórico.

func manifestoDeSkill(versao string) SkillManifest {
	return SkillManifest{ID: "skill-x", Version: versao, Description: "skill de teste"}
}

func TestRegistarVersaoNovaArquivaAAnterior(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkill(manifestoDeSkill("1.0")); err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkill(manifestoDeSkill("2.0")); err != nil {
		t.Fatal(err)
	}

	// (1) A VERSÃO ACTIVA é a nova.
	activas := store.Skills()
	if len(activas) != 1 || activas[0].Version != "2.0" {
		t.Fatalf("a versão activa devia ser a 2.0: %+v", activas)
	}

	// (2) E A ANTERIOR FICOU EM DISCO — é isto que faltava.
	arquivada := filepath.Join(store.root, "skills", "skill-x.versions", "1.0.json")
	if _, err := os.Stat(arquivada); err != nil {
		t.Fatalf("*** a versão anterior NÃO foi arquivada ***: %v", err)
	}
}

// Registar a MESMA versão não é uma versão nova: não deve poluir o histórico com repetições.
func TestRegistarAMesmaVersaoNaoArquivaNada(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkill(manifestoDeSkill("1.0")); err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkill(manifestoDeSkill("1.0")); err != nil {
		t.Fatal(err)
	}
	pasta := filepath.Join(store.root, "skills", "skill-x.versions")
	if entradas, err := os.ReadDir(pasta); err == nil && len(entradas) != 0 {
		t.Fatalf("registar a mesma versão arquivou %d ficheiro(s)", len(entradas))
	}
}

// E o histórico NÃO é criado para a PRIMEIRA versão (não havia anterior a preservar).
func TestPrimeiraVersaoNaoCriaHistorico(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkill(manifestoDeSkill("1.0")); err != nil {
		t.Fatal(err)
	}
	pasta := filepath.Join(store.root, "skills", "skill-x.versions")
	if entradas, err := os.ReadDir(pasta); err == nil && len(entradas) != 0 {
		t.Fatalf("a primeira versão não devia criar histórico: %d ficheiro(s)", len(entradas))
	}
}
