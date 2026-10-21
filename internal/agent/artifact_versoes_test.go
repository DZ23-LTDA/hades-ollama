package agent

import (
	"errors"
	"testing"
	"time"
)

// E-539 (Estagio 47) - a triade: versionar, comparar, restaurar.
//
// *O teste que importa e' o do restauro: **restaurar NAO pode apagar o que la' estava.***

func versao(id, tenant, project, path, sha string, n int, tamanho int64) ArtifactManifest {
	return ArtifactManifest{
		ID: id, TenantID: tenant, UserID: "ana", ProjectID: project, Path: path, Name: path,
		Version: n, SHA256: sha, Size: tamanho, MediaType: "text/plain",
	}
}

// ***A REGRA DO RESTAURO.*** Restaurar a 1 quando se esta' na 3 produz a **4** com o conteudo da 1 - e a 3 **fica**.
func TestRestaurarNaoApagaAHistoria(t *testing.T) {
	indice := []ArtifactManifest{
		versao("v1", "acme", "p1", "doc.md", "sha-A", 1, 100),
		versao("v2", "acme", "p1", "doc.md", "sha-B", 2, 120),
		versao("v3", "acme", "p1", "doc.md", "sha-C", 3, 140),
	}

	novo, err := ArtefactoRestaurarVersao(indice, "acme", "p1", "doc.md", 1, 3, time.Time{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// O novo manifesto: conteudo da 1, versao 4.
	if novo.SHA256 != "sha-A" {
		t.Fatalf("o restauro devia repor o conteudo da versao 1 (sha-A), veio %q", novo.SHA256)
	}
	if novo.Version != 4 {
		t.Fatalf("restaurar devia produzir a versao 4 (a seguir a' corrente 3), veio %d", novo.Version)
	}
	if novo.TenantID != "acme" || novo.ProjectID != "p1" || novo.Path != "doc.md" {
		t.Fatalf("o restauro mudou a identidade do artefacto: %+v", novo)
	}

	// ***E O INDICE NAO FOI TOCADO.*** *Esta e' a diferenca entre um sistema de versoes e um `UPDATE`.*
	if len(indice) != 3 {
		t.Fatalf("ARMADILHA: o restauro alterou o indice de entrada (%d itens) - *um restauro que muta a lista e' um "+
			"restauro que destroi precisamente o que se ia recuperar*", len(indice))
	}
	if _, ainda := versaoDeArtefacto(indice, "acme", "p1", "doc.md", 3); !ainda {
		t.Fatal("ARMADILHA: a versao 3 desapareceu - *o passado tem de permanecer consultavel*")
	}

	// E ao acrescentar o novo, a historia CRESCE (4 versoes), nunca encolhe.
	depois := append(append([]ArtifactManifest{}, indice...), novo)
	historico := ArtefactoHistorico(depois, "acme", "p1", "doc.md")
	if len(historico) != 4 {
		t.Fatalf("o historico devia ter 4 versoes, tem %d", len(historico))
	}
	if historico[0].Version != 4 || historico[3].Version != 1 {
		t.Fatalf("o historico devia vir da mais recente para a mais antiga: %d..%d", historico[0].Version, historico[3].Version)
	}
}

// Restaurar com um `esperado` desactualizado RECUSA - *duas restauracoes concorrentes perderiam uma.*
func TestRestaurarRecusaConflito(t *testing.T) {
	indice := []ArtifactManifest{
		versao("v1", "acme", "p1", "doc.md", "sha-A", 1, 100),
		versao("v3", "acme", "p1", "doc.md", "sha-C", 3, 140),
	}
	// Quem grava julga estar na 2 - mas ja' vai na 3.
	_, err := ArtefactoRestaurarVersao(indice, "acme", "p1", "doc.md", 1, 2, time.Time{})
	if !errors.Is(err, ErrRestauroConflitante) {
		t.Fatalf("ARMADILHA: restauro com `esperado` desactualizado foi aceite (err=%v)", err)
	}
}

// Restaurar uma versao que nao existe RECUSA.
func TestRestaurarVersaoInexistente(t *testing.T) {
	indice := []ArtifactManifest{versao("v1", "acme", "p1", "doc.md", "sha-A", 1, 100)}
	if _, err := ArtefactoRestaurarVersao(indice, "acme", "p1", "doc.md", 9, 1, time.Time{}); !errors.Is(err, ErrVersaoInexistente) {
		t.Fatalf("versao inexistente devia recusar, veio %v", err)
	}
}

// ***E O RESTAURO NAO ATRAVESSA TENANT.*** *A mesma fronteira do E-537, no terceiro sitio onde ela se afirma.*
func TestRestaurarNaoAtravessaTenant(t *testing.T) {
	indice := []ArtifactManifest{
		versao("a1", "acme", "p1", "doc.md", "sha-A", 1, 100),
		versao("b1", "globex", "p1", "doc.md", "sha-B", 1, 100),
	}
	// `acme` nao pode restaurar a versao que so' existe em `globex` - *mesmo com o mesmo projectID e o mesmo path.*
	if _, err := ArtefactoRestaurarVersao(indice, "acme", "p1", "doc.md", 1, 1, time.Time{}); err != nil {
		t.Fatalf("acme devia conseguir restaurar a SUA versao 1: %v", err)
	}
	novo, err := ArtefactoRestaurarVersao(indice, "acme", "p1", "doc.md", 1, 1, time.Time{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if novo.SHA256 != "sha-A" {
		t.Fatalf("ARMADILHA: o restauro trouxe o conteudo de OUTRO tenant: %q", novo.SHA256)
	}
}

// ***COMPARAR: `MesmoConteudo` SO' PELO HASH.*** *O tamanho nao decide conteudo.*
func TestCompararTamanhoIgualNaoEConteudoIgual(t *testing.T) {
	indice := []ArtifactManifest{
		versao("v1", "acme", "p1", "doc.md", "sha-A", 1, 1024),
		versao("v2", "acme", "p1", "doc.md", "sha-B", 2, 1024), // mesmo tamanho, hash diferente
	}
	d, err := ArtefactoCompararVersoes(indice, "acme", "p1", "doc.md", 1, 2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if d.MesmoConteudo {
		t.Fatal("ARMADILHA: `MesmoConteudo` foi verdadeiro com hashes DIFERENTES - *dois ficheiros de 1024 bytes podem " +
			"nao partilhar um unico byte, e dizer «e' igual» por tamanho e' um falso negativo a espera de acontecer*")
	}
	if !d.MesmoTamanho {
		t.Fatal("o tamanho e' igual e devia ser reportado como tal")
	}
	if d.Razao == "" {
		t.Fatal("a diferenca devia explicar-se em texto")
	}
}

func TestCompararConteudoIgualPeloHash(t *testing.T) {
	indice := []ArtifactManifest{
		versao("v1", "acme", "p1", "doc.md", "sha-A", 1, 100),
		versao("v2", "acme", "p1", "doc.md", "sha-A", 2, 100),
	}
	d, err := ArtefactoCompararVersoes(indice, "acme", "p1", "doc.md", 1, 2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !d.MesmoConteudo || d.Razao == "" {
		t.Fatalf("hashes iguais deviam dar MesmoConteudo com razao, veio %+v", d)
	}
}

// E comparar uma versao que nao existe RECUSA - *nao ha' diferenca contra o que nao existe.*
func TestCompararVersaoInexistente(t *testing.T) {
	indice := []ArtifactManifest{versao("v1", "acme", "p1", "doc.md", "sha-A", 1, 100)}
	if _, err := ArtefactoCompararVersoes(indice, "acme", "p1", "doc.md", 1, 7); !errors.Is(err, ErrVersaoInexistente) {
		t.Fatalf("versao inexistente devia recusar, veio %v", err)
	}
	// E sem tenant tambem.
	if _, err := ArtefactoCompararVersoes(indice, "", "p1", "doc.md", 1, 1); !errors.Is(err, ErrBuscaSemTenant) {
		t.Fatalf("sem tenant devia recusar, veio %v", err)
	}
}
