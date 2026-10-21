package agent

import (
	"errors"
	"testing"
)

// E-537 (Estagio 47) - o teste decisivo: **a busca NAO atravessa tenant.**

func artefacto(id, tenant, user, project, path, nome string, versao int, acl []string) ArtifactManifest {
	return ArtifactManifest{
		ID: id, TenantID: tenant, UserID: user, ProjectID: project, Path: path, Name: nome,
		Version: versao, SHA256: "sha-" + id, ACL: acl,
	}
}

// ***O TESTE QUE IMPORTA.*** Dois tenants com o **mesmo nome de ficheiro** - o caso em que uma busca descuidada devolve
// o relatorio de outra organizacao.
func TestBuscaNaoAtravessaTenant(t *testing.T) {
	indice := []ArtifactManifest{
		artefacto("a1", "acme", "ana", "p1", "out/relatorio.pdf", "relatorio.pdf", 1, nil),
		artefacto("b1", "globex", "bob", "p1", "out/relatorio.pdf", "relatorio.pdf", 1, nil),
	}

	// Ana procura «relatorio» em `acme`: tem de ver UM, e o dela.
	achados, err := BuscarArtefactos(indice, ArtifactQuery{TenantID: "acme", Sujeito: "ana", Texto: "relatorio"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(achados) != 1 {
		t.Fatalf("ARMADILHA: a busca de `acme` devolveu %d artefactos - *o `relatorio.pdf` de `globex` nao pode "+
			"aparecer, nem por acidente de nome igual*", len(achados))
	}
	if achados[0].ID != "a1" || achados[0].TenantID != "acme" {
		t.Fatalf("ARMADILHA: a busca devolveu o artefacto ERRADO: %+v", achados[0])
	}

	// E o inverso: Bob, em `globex`, ve o dele e so' o dele.
	doBob, err := BuscarArtefactos(indice, ArtifactQuery{TenantID: "globex", Sujeito: "bob"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(doBob) != 1 || doBob[0].ID != "b1" {
		t.Fatalf("ARMADILHA: `globex` viu %d artefactos: %+v", len(doBob), doBob)
	}
}

// E uma busca SEM tenant recusa: *«sem tenant» nao e' uma busca ampla, e' a ausencia de fronteira.*
func TestBuscaRecusaSemTenantESemSujeito(t *testing.T) {
	indice := []ArtifactManifest{artefacto("a1", "acme", "ana", "p1", "x.txt", "x.txt", 1, nil)}

	if _, err := BuscarArtefactos(indice, ArtifactQuery{Sujeito: "ana"}); !errors.Is(err, ErrBuscaSemTenant) {
		t.Fatalf("busca sem tenant devia recusar, veio %v", err)
	}
	if _, err := BuscarArtefactos(indice, ArtifactQuery{TenantID: "acme"}); !errors.Is(err, ErrBuscaSemSujeito) {
		t.Fatalf("busca sem sujeito devia recusar, veio %v", err)
	}
}

// A ACL e a propriedade continuam a mandar - *a busca nao e' uma porta lateral para o que `ArtifactVisivelPara` nega.*
func TestBuscaRespeitaACLePropriedade(t *testing.T) {
	indice := []ArtifactManifest{
		artefacto("a1", "acme", "ana", "p1", "meu.txt", "meu.txt", 1, []string{"group:qa"}),
	}
	// O dono ve'.
	if achados, _ := BuscarArtefactos(indice, ArtifactQuery{TenantID: "acme", Sujeito: "ana"}); len(achados) != 1 {
		t.Fatalf("o dono devia ver o seu artefacto, viu %d", len(achados))
	}
	// A ACL acrescenta.
	if achados, _ := BuscarArtefactos(indice, ArtifactQuery{TenantID: "acme", Sujeito: "qa"}); len(achados) != 1 {
		t.Fatalf("a ACL devia deixar `qa` ver, viu %d", len(achados))
	}
	// Um estranho do MESMO tenant nao ve' - *e isto prova que a busca nao e' mais permissiva do que o acesso.*
	if achados, _ := BuscarArtefactos(indice, ArtifactQuery{TenantID: "acme", Sujeito: "estranho"}); len(achados) != 0 {
		t.Fatalf("ARMADILHA: um estranho do mesmo tenant viu %d artefactos - *a busca seria uma porta lateral*", len(achados))
	}
}

// Sem pedir historico, a busca devolve a VERSAO CORRENTE - e nao todas.
func TestBuscaDevolveAVersaoCorrentePorOmissao(t *testing.T) {
	indice := []ArtifactManifest{
		artefacto("v1", "acme", "ana", "p1", "doc.md", "doc.md", 1, nil),
		artefacto("v2", "acme", "ana", "p1", "doc.md", "doc.md", 2, nil),
		artefacto("v3", "acme", "ana", "p1", "doc.md", "doc.md", 3, nil),
	}
	achados, err := BuscarArtefactos(indice, ArtifactQuery{TenantID: "acme", Sujeito: "ana"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(achados) != 1 {
		t.Fatalf("por omissao devia haver 1 (a corrente), houve %d", len(achados))
	}
	if achados[0].Version != 3 || achados[0].ID != "v3" {
		t.Fatalf("a versao corrente devia ser a 3, veio %+v", achados[0])
	}
	// E pedindo o historico, veem-se as tres - *em ordem determinista.*
	todas, _ := BuscarArtefactos(indice, ArtifactQuery{TenantID: "acme", Sujeito: "ana", IncluirVersoesAntigas: true})
	if len(todas) != 3 {
		t.Fatalf("com historico deviam ser 3, foram %d", len(todas))
	}
	if todas[0].Version != 3 || todas[2].Version != 1 {
		t.Fatalf("a ordem devia ser por versao decrescente num empate de nome: %d..%d", todas[0].Version, todas[2].Version)
	}
}

// O CAS do artefacto: *quem grava diz sobre que versao grava, e um `esperado` errado RECUSA em vez de sobrescrever.*
func TestProximaVersaoDeArtefactoRecusaConflito(t *testing.T) {
	indice := []ArtifactManifest{
		artefacto("v1", "acme", "ana", "p1", "doc.md", "doc.md", 1, nil),
		artefacto("v2", "acme", "ana", "p1", "doc.md", "doc.md", 2, nil),
	}
	// Criar: esperado 0, corrente 0 -> 1.
	if v, err := ProximaVersaoDeArtefacto(indice, "acme", "p1", "novo.md", 0); err != nil || v != 1 {
		t.Fatalf("criacao devia dar 1, deu %d (err=%v)", v, err)
	}
	// Gravar sobre a corrente: esperado 2 -> 3.
	if v, err := ProximaVersaoDeArtefacto(indice, "acme", "p1", "doc.md", 2); err != nil || v != 3 {
		t.Fatalf("esperado 2 devia dar 3, deu %d (err=%v)", v, err)
	}
	// ***E O CONFLITO: esperado 1, mas ja' vai na 2 -> RECUSA.*** *Sem isto, duas gravacoes concorrentes perdem uma.*
	if _, err := ProximaVersaoDeArtefacto(indice, "acme", "p1", "doc.md", 1); err == nil {
		t.Fatal("ARMADILHA: um `esperado` desactualizado foi aceite - *e' o padrao CAS do `builder`, e sem ele a " +
			"gravacao sobrescreve em silencio*")
	}
	// E o mesmo ficheiro NOUTRO tenant nao interfere: a versao e' por tenant.
	if v, err := ProximaVersaoDeArtefacto(indice, "globex", "p1", "doc.md", 0); err != nil || v != 1 {
		t.Fatalf("noutro tenant devia comecar em 1, deu %d (err=%v)", v, err)
	}
}
