package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A CATRACA DO PACOTE REPLICA.
//
// Duas afirmacoes que o CI passa a verificar sozinho:
//  1. `content_sha256` == digest calculado AGORA pelo proprio Hades. Editar a
//     `description` de um manifesto (ou a `version`) **quebra** este teste -
//     e' a catraca a avisar que a assinatura teria de ser refeita.
//  2. O conteudo upstream continua **intacto**: os sha256 de PROVENANCE.md
//     batem com os ficheiros em disco. Editar codigo de terceiros sem atualizar
//     a procedencia (e o aviso da MIT) quebra aqui.
func TestReplicaPackDigestEProcedenciaIntactos(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.LoadSkillsForOrganization(replicaManifests, LocalOrganizationID); err != nil {
		t.Fatalf("carregar: %v", err)
	}
	skills := store.SkillsForOrganization(LocalOrganizationID)
	if len(skills) != 11 {
		t.Fatalf("esperava 11, carregou %d", len(skills))
	}
	for _, m := range skills {
		if strings.TrimSpace(m.ContentSHA256) == "" {
			t.Fatalf("%s: sem content_sha256 - a assinatura futura nao teria o que verificar", m.ID)
		}
		agora, err := SkillManifestDigest(m)
		if err != nil {
			t.Fatalf("%s: %v", m.ID, err)
		}
		// O digest e' ligado a organizacao: `organization_id` entra nos bytes
		// assinados. Carregar para outro tenant invalida este valor.
		if !strings.EqualFold(agora, m.ContentSHA256) {
			t.Fatalf("%s: digest divergiu\n  gravado: %s\n  medido:  %s\n"+
				"  (editar description/version exige re-assinar e regravar content_sha256)",
				m.ID, m.ContentSHA256, agora)
		}
		// FALHA FECHADO: sem chave autorizada, promover tem de RECUSAR.
		if _, err := (CapabilityPolicy{}).PromoteSkillTrusted(m); err == nil {
			t.Fatalf("%s: PromotedSkillTrusted aceitou sem chave - o catalogo concedeu privilegio", m.ID)
		} else if !errors.Is(err, ErrSkillKeyUnauthorized) {
			t.Fatalf("%s: recusou com %v (esperado ErrSkillKeyUnauthorized)", m.ID, err)
		}
	}
	t.Logf("  11 manifestos: digest confere e promocao recusada (fail-closed)")

	// ------------------------------------------------- PROCEDENCIA (sha256)
	// SEM NUMERO MAGICO: a contagem esperada e' descoberta a partir do que existe
	// em disco. Eu tinha escrito "25" por aritmetica de cabeca e o valor real era
	// outro - um numero inventado num teste e' uma afirmacao sem dono.
	// A PRECONDICAO para o sha256 sobreviver a um checkout: o pacote vendado tem
	// de estar marcado `-text` em .gitattributes. Sem isso, o `* text=auto` do
	// repositorio normaliza CRLF->LF no commit e os hashes deixam de bater noutra
	// maquina - a mesma armadilha do E-183, com outro ficheiro.
	attrs, err := os.ReadFile(filepath.Join("..", "..", ".gitattributes"))
	if err != nil {
		t.Fatalf("ler .gitattributes: %v", err)
	}
	if !strings.Contains(string(attrs), "skills/replica/replica-*/** -text") {
		t.Fatal(".gitattributes sem '-text' para skills/replica/replica-*/**: " +
			"o `* text=auto` normalizaria CRLF->LF e o sha256 deixaria de bater noutro checkout")
	}

	bruto, err := os.ReadFile(filepath.Join(replicaDir, "PROVENANCE.md"))
	if err != nil {
		t.Fatalf("ler PROVENANCE.md: %v", err)
	}
	// `\r?` porque o ficheiro e' gerado no Windows (CRLF) e pode ser lido como LF
	// noutro checkout: um `$` estrito reprovaria por fim de linha, nao por conteudo.
	linha := regexp.MustCompile("(?m)^\\| `([^`]+)` \\| `([0-9a-f]{64})` \\|\r?$")
	declarado := map[string]string{}
	for _, a := range linha.FindAllStringSubmatch(string(bruto), -1) {
		declarado[a[1]] = a[2]
	}
	if len(declarado) == 0 {
		t.Fatal("PROVENANCE.md nao declara nenhum sha256")
	}
	// Ficheiros NOSSOS em `skills/replica/`: nao sao conteudo de terceiros, logo
	// nao entram na tabela de procedencia. A lista e' explicita de proposito -
	// "ignorar tudo o que nao esta' declarado" transformaria a verificacao numa
	// tautologia, e um ficheiro de terceiros passaria sem ser notado.
	nossos := map[string]bool{"README.md": true, "PROVENANCE.md": true}
	// (a) todo ficheiro do pacote tem de estar declarado e bater
	conferidos := 0
	err = filepath.WalkDir(replicaDir, func(caminho string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "manifests" {
				return fs.SkipDir // derivados nossos, nao conteudo de terceiros
			}
			return nil
		}
		rel, err := filepath.Rel(replicaDir, caminho)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if nossos[rel] {
			return nil
		}
		esperado, ok := declarado[rel]
		if !ok {
			t.Errorf("%s existe mas NAO esta' declarado em PROVENANCE.md", rel)
			return nil
		}
		dados, err := os.ReadFile(caminho)
		if err != nil {
			return err
		}
		soma := sha256.Sum256(dados)
		if obtido := hex.EncodeToString(soma[:]); obtido != esperado {
			t.Errorf("%s: sha256 divergiu\n  declarado: %s\n  medido:    %s", rel, esperado, obtido)
		}
		conferidos++
		return nil
	})
	if err != nil {
		t.Fatalf("percorrer o pacote: %v", err)
	}
	// (b) e nenhuma linha orfa: declarar um ficheiro que ja' nao existe e' uma
	// procedencia que mente por omissao.
	for rel := range declarado {
		if _, err := os.Stat(filepath.Join(replicaDir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("PROVENANCE.md declara %s, que nao existe", rel)
		}
	}
	if conferidos != len(declarado) {
		t.Fatalf("conferidos %d ficheiros para %d declaracoes", conferidos, len(declarado))
	}
	t.Logf("  %d sha256 de procedencia conferem (conteudo upstream intacto, zero orfaos)", conferidos)
}

// O CARREGADOR RECUSA CAMPO DESCONHECIDO - e' o que impede "instalar" uma skill
// com campos inventados que o runtime nao interpreta mas alguem poderia ler.
func TestReplicaPackRecusaCampoDesconhecido(t *testing.T) {
	dir := t.TempDir()
	injetado := `{"id":"replica-falsa","version":"1","description":"d","trusted":true,"scopes":["sandbox:execute"]}`
	if err := os.WriteFile(filepath.Join(dir, "falsa.json"), []byte(injetado), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// NOTA: `trusted` e `scopes` SÃO campos validos da struct - o que este teste
	// prova e' que mesmo declarando `trusted:true` no JSON o carregador o zera
	// (DisallowUnknownFields barra o que nao existe; o runtime zera o que existe).
	if err := store.LoadSkillsForOrganization(dir, LocalOrganizationID); err != nil {
		t.Fatalf("carregar: %v", err)
	}
	skills := store.SkillsForOrganization(LocalOrganizationID)
	if len(skills) != 1 {
		t.Fatalf("esperava 1, carregou %d", len(skills))
	}
	if skills[0].Trusted {
		t.Fatalf("trusted:true do JSON foi aceito - o catalogo concedeu privilegio")
	}

	invalido := t.TempDir()
	if err := os.WriteFile(filepath.Join(invalido, "x.json"),
		[]byte(`{"id":"x","version":"1","description":"d","campo_inventado":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store2, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	err = store2.LoadSkillsForOrganization(invalido, LocalOrganizationID)
	if err == nil {
		t.Fatalf("campo desconhecido foi aceito")
	}
	if !strings.Contains(fmt.Sprint(err), "unknown field") {
		t.Fatalf("recusou por outro motivo: %v", err)
	}
	t.Logf("  campo inventado recusado: %v", err)
}
