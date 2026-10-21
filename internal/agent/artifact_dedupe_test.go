package agent

import (
	"testing"
)

// E-542 (Estagio 47) - a deduplicacao: agrupa pelo hash, **dentro do tenant**, e recusa atravessar a fronteira.

func dup(id, tenant, user, path, sha string, n int, tamanho int64, acl []string) ArtifactManifest {
	return ArtifactManifest{
		ID: id, TenantID: tenant, UserID: user, ProjectID: "p1", Path: path, Name: path,
		Version: n, SHA256: sha, Size: tamanho, MediaType: "text/plain", ACL: acl,
	}
}

// ***O TESTE QUE IMPORTA.*** Dois tenants com o mesmo conteudo: **cada um ve' o seu grupo, e so' o seu.**
func TestDedupeNaoRevelaConteudoDeOutroTenant(t *testing.T) {
	indice := []ArtifactManifest{
		// `acme` tem o mesmo ficheiro em dois sitios.
		dup("a1", "acme", "ana", "docs/contrato.md", "sha-IGUAL", 1, 100, nil),
		dup("a2", "acme", "ana", "backup/contrato.md", "sha-IGUAL", 1, 100, nil),
		// `globex` tem o MESMO conteudo, num sitio.
		dup("b1", "globex", "bob", "docs/contrato.md", "sha-IGUAL", 1, 100, nil),
	}

	doAcme := ArtefactoDetetarDuplicados(indice, "acme", "ana")
	if len(doAcme) != 1 {
		t.Fatalf("acme devia ver 1 grupo, viu %d", len(doAcme))
	}
	// ***A PROVA:*** *o grupo de `acme` tem **exactamente 2** caminhos - o `globex` **nao** conta para o grupo dele.*
	if len(doAcme[0].Caminhos) != 2 {
		t.Fatalf("ARMADILHA: o grupo de `acme` tem %d caminhos - *se contasse o de `globex`, a dedup seria um canal "+
			"lateral que revela que OUTRA organizacao tem o mesmo ficheiro*", len(doAcme[0].Caminhos))
	}
	// E o desperdicio e' de UMA copia, nao de duas.
	if doAcme[0].DesperdicioBytes != 100 {
		t.Fatalf("desperdicio devia ser 100 (uma copia a mais), veio %d", doAcme[0].DesperdicioBytes)
	}

	// E o `globex` sozinho **nao tem duplicado nenhum** - *tem uma copia, e um so' ficheiro nao e' duplicado.*
	doGlobex := ArtefactoDetetarDuplicados(indice, "globex", "bob")
	if len(doGlobex) != 0 {
		t.Fatalf("ARMADILHA: `globex` viu %d grupos - *ele tem UMA copia daquele conteudo; agrupa-lo com a de `acme` "+
			"seria exactamente o canal lateral que este ficheiro recusa*", len(doGlobex))
	}
}

// Um ficheiro SEM sha256 e' ignorado: *dois desconhecidos nao sao o mesmo.*
func TestDedupeIgnoraSemHash(t *testing.T) {
	indice := []ArtifactManifest{
		dup("a1", "acme", "ana", "x.md", "", 1, 100, nil),
		dup("a2", "acme", "ana", "y.md", "", 1, 100, nil),
		dup("a3", "acme", "ana", "z.md", "   ", 1, 100, nil),
	}
	if grupos := ArtefactoDetetarDuplicados(indice, "acme", "ana"); len(grupos) != 0 {
		t.Fatalf("ARMADILHA: agrupou %d grupos sem hash - *nao se agrupa por «nao sei»*", len(grupos))
	}
}

// A visibilidade manda: *um estranho do mesmo tenant nao ve' os duplicados dos outros.*
func TestDedupeRespeitaACLePropriedade(t *testing.T) {
	indice := []ArtifactManifest{
		dup("a1", "acme", "ana", "docs/a.md", "sha-X", 1, 50, nil),
		dup("a2", "acme", "ana", "docs/b.md", "sha-X", 1, 50, nil),
	}
	// O dono ve' o grupo.
	if g := ArtefactoDetetarDuplicados(indice, "acme", "ana"); len(g) != 1 {
		t.Fatalf("o dono devia ver 1 grupo, viu %d", len(g))
	}
	// Um estranho do MESMO tenant nao ve' nada - *a ACL vazia significa «ninguem alem do dono», nunca «todos».*
	if g := ArtefactoDetetarDuplicados(indice, "acme", "estranho"); len(g) != 0 {
		t.Fatalf("ARMADILHA: um estranho viu %d grupos", len(g))
	}
}

// O conteudo e' provado pelo HASH, nao pelo tamanho - *e o inverso do falso positivo tambem conta.*
func TestDedupeNaoAgrupaTamanhoIgualComHashDiferente(t *testing.T) {
	indice := []ArtifactManifest{
		dup("a1", "acme", "ana", "docs/a.md", "sha-A", 1, 1024, nil),
		dup("a2", "acme", "ana", "docs/b.md", "sha-B", 1, 1024, nil), // mesmo tamanho, conteudo diferente
	}
	if g := ArtefactoDetetarDuplicados(indice, "acme", "ana"); len(g) != 0 {
		t.Fatalf("ARMADILHA: agrupou %d grupos com hashes DIFERENTES e tamanhos iguais - *o tamanho nao decide conteudo*", len(g))
	}
}

// A ordem e' determinista: *maior desperdicio primeiro, e o hash como desempate.*
func TestDedupeOrdemDeterminista(t *testing.T) {
	indice := []ArtifactManifest{
		dup("p1", "acme", "ana", "a/x.md", "sha-PEQUENO", 1, 10, nil),
		dup("p2", "acme", "ana", "b/x.md", "sha-PEQUENO", 1, 10, nil),
		dup("g1", "acme", "ana", "a/grande.bin", "sha-GRANDE", 1, 1000, nil),
		dup("g2", "acme", "ana", "b/grande.bin", "sha-GRANDE", 1, 1000, nil),
		dup("g3", "acme", "ana", "c/grande.bin", "sha-GRANDE", 1, 1000, nil),
	}
	g := ArtefactoDetetarDuplicados(indice, "acme", "ana")
	if len(g) != 2 {
		t.Fatalf("devia haver 2 grupos, houve %d", len(g))
	}
	// O grande desperdica 2000 (duas copias a mais); o pequeno, 10.
	if g[0].SHA256 != "sha-GRANDE" || g[0].DesperdicioBytes != 2000 {
		t.Fatalf("o primeiro grupo devia ser o de maior desperdicio (sha-GRANDE, 2000), veio %+v", g[0])
	}
	if g[1].SHA256 != "sha-PEQUENO" || g[1].DesperdicioBytes != 10 {
		t.Fatalf("o segundo devia ser sha-PEQUENO (10), veio %+v", g[1])
	}
	// E os caminhos vem ordenados dentro do grupo.
	for i := 1; i < len(g[0].Caminhos); i++ {
		if g[0].Caminhos[i-1] > g[0].Caminhos[i] {
			t.Fatalf("os caminhos deviam vir ordenados: %v", g[0].Caminhos)
		}
	}
}
