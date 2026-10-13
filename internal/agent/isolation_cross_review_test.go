package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// REVISÃO CRUZADA DE PERMISSÃO E ISOLAMENTO — ESTÁGIO 42 (E-336).
//
// Ataque declarado: *revisão cruzada de permissões/isolamento nos módulos novos*. Duas brechas confirmadas:
//  1. o destaque de produto aceitava **qualquer** SKU não vazio — dava para "destacar" produto inexistente, ou de
//     outra empresa, se o chamador passasse SKU alheia;
//  2. o upload de mídia não tinha **escopo de organização** — o destino era um caminho cru.
//
// ***Isolar por convenção não é isolar.***

// --- 1. destaque de produto não cruza organização e não inventa SKU ---

func TestDestaqueRecusaSkuDeOutraOrganizacaoEInexistente(t *testing.T) {
	store := NewLiveCommerceStore(filepath.Join(t.TempDir(), "live"))
	sessao, err := store.CreateLiveSession(LiveSession{OrganizationID: "org-a", ProjectID: "p1",
		AuthorizedUntil: time.Now().UTC().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	catalogo := &catalogoDeTeste{produtos: map[string][]string{
		"org-a": {"SKU-PROPRIO"},
		"org-b": {"SKU-DA-VIZINHA"},
	}}
	store.SetCatalog(catalogo)

	// SKU inexistente na própria organização ⇒ recusa.
	if _, err := store.ShowcaseForProduct("org-a", sessao.ID,
		ProductShowcase{SKU: "SKU-QUE-NAO-EXISTE", Position: 1}, time.Now().UTC()); !errors.Is(err, ErrLiveCommerceInvalid) {
		t.Fatalf("SKU inexistente precisa ser recusado: %v", err)
	}
	// ARMADILHA: SKU que existe — mas em OUTRA organização ⇒ recusa (a consulta é feita com o tenant do chamador).
	if _, err := store.ShowcaseForProduct("org-a", sessao.ID,
		ProductShowcase{SKU: "SKU-DA-VIZINHA", Position: 1}, time.Now().UTC()); !errors.Is(err, ErrLiveCommerceInvalid) {
		t.Fatalf("ARMADILHA: destaque cruzou organização — SKU de outra empresa foi aceito (o catálogo é por tenant)")
	}
	// SKU próprio e existente ⇒ aceito.
	aceito, err := store.ShowcaseForProduct("org-a", sessao.ID,
		ProductShowcase{SKU: "SKU-PROPRIO", Position: 1}, time.Now().UTC())
	if err != nil {
		t.Fatalf("SKU da própria organização precisa ser aceito: %v", err)
	}
	if len(aceito.Showcase) != 1 || aceito.Showcase[0].SKU != "SKU-PROPRIO" {
		t.Fatalf("o destaque precisa registrar o SKU verificado: %+v", aceito.Showcase)
	}
}

// --- 2. upload de mídia com escopo de organização ---

func TestUploadDeMidiaFicaDentroDaRaizDaOrganizacao(t *testing.T) {
	base := t.TempDir()
	origem := arquivoComConteudo(t, t.TempDir(), "video.mp4", cabecalhoMP4())

	// A raiz é derivada da organização: cada tenant tem a sua pasta.
	raizA, err := MediaOrgRoot(base, "org-a")
	if err != nil {
		t.Fatal(err)
	}
	raizB, err := MediaOrgRoot(base, "org-b")
	if err != nil {
		t.Fatal(err)
	}
	if raizA == raizB {
		t.Fatalf("duas organizações não podem compartilhar a mesma raiz de mídia: %q", raizA)
	}
	if filepath.Dir(raizA) != base {
		t.Fatalf("a raiz da organização precisa ficar sob a base: %q", raizA)
	}

	validado, err := ValidateMediaUploadInOrg(origem, base, "org-a", "video.mp4", "video/mp4", DefaultMediaLimits())
	if err != nil {
		t.Fatalf("upload válido: %v", err)
	}
	// ARMADILHA: o arquivo precisa estar DENTRO da raiz do próprio tenant — jamais na de outro.
	if !pathDentroDe(validado.StoredPath, raizA) {
		t.Fatalf("ARMADILHA: arquivo de org-a gravado fora da raiz dela: %q", validado.StoredPath)
	}
	if pathDentroDe(validado.StoredPath, raizB) {
		t.Fatalf("ARMADILHA: arquivo de org-a caiu na raiz de org-b: %q", validado.StoredPath)
	}
	// E o upload de outra organização vai para a pasta dela — isolamento nos dois sentidos.
	// MEDIDO por `go vet`: a função devolve DOIS valores. Medir a assinatura em vez de supor.
	outro, err := ValidateMediaUploadInOrg(origem, base, "org-b", "video.mp4", "video/mp4", DefaultMediaLimits())
	if err != nil {
		t.Fatalf("upload da organização vizinha: %v", err)
	}
	if outro.StoredPath == validado.StoredPath {
		t.Fatal("uploads de organizações diferentes não podem compartilhar caminho")
	}
	// BASE VAZIA ou organização vazia ⇒ recusa explícita (não existe "pasta global" implícita).
	if _, err := MediaOrgRoot("", "org-a"); !errors.Is(err, ErrMediaTenantScope) {
		t.Fatalf("base vazia precisa ser recusada: %v", err)
	}
	if _, err := MediaOrgRoot(base, "   "); !errors.Is(err, ErrMediaTenantScope) {
		t.Fatalf("organização vazia precisa ser recusada: %v", err)
	}
}

// A validação por conteúdo continua valendo no caminho com escopo (a correção de isolamento não afrouxou a anterior).
func TestUploadComEscopoContinuaValidandoConteudo(t *testing.T) {
	base := t.TempDir()
	// .mp4 que na verdade é PNG: recusado, mesmo com escopo correto.
	falso := arquivoComConteudo(t, t.TempDir(), "video.mp4", cabecalhoPNG())
	if _, err := ValidateMediaUploadInOrg(falso, base, "org-a", "video.mp4", "video/mp4", DefaultMediaLimits()); !errors.Is(err, ErrMediaContentMismatch) {
		t.Fatalf("a checagem de conteúdo precisa continuar valendo: %v", err)
	}
	// Nome perigoso continua recusado no caminho com escopo.
	bom := arquivoComConteudo(t, t.TempDir(), "ok.mp4", cabecalhoMP4())
	if _, err := ValidateMediaUploadInOrg(bom, base, "org-a", "../fuga.mp4", "video/mp4", DefaultMediaLimits()); !errors.Is(err, ErrMediaPathUnsafe) {
		t.Fatalf("nome com traversal precisa continuar recusado: %v", err)
	}
	// E nada foi gravado fora da raiz da organização.
	if _, err := os.Stat(filepath.Join(base, "fuga.mp4")); err == nil {
		t.Fatal("ARMADILHA: arquivo com traversal escapou para a base")
	}
}

// pathDentroDe diz se `caminho` está dentro de `raiz` (comparação de caminhos resolvidos).
func pathDentroDe(caminho, raiz string) bool {
	absCaminho, err1 := filepath.Abs(caminho)
	absRaiz, err2 := filepath.Abs(raiz)
	if err1 != nil || err2 != nil {
		return false
	}
	relativo, err := filepath.Rel(absRaiz, absCaminho)
	if err != nil {
		return false
	}
	return relativo != ".." && !stringsHasPrefix(relativo, ".."+string(filepath.Separator))
}

func stringsHasPrefix(texto, prefixo string) bool {
	return len(texto) >= len(prefixo) && texto[:len(prefixo)] == prefixo
}
