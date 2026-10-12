package agent

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// TestStagingPromocaoEUrollbackDoAppGerado cobre a lacuna declarada em E-246: o Builder publicava arquivos e nunca
// SUBIA o que publicou, entao nao havia como verificar saude antes de promover uma versao.
//
// O teste exige o comportamento fail-closed: promover o que NAO responde saudavel trocaria uma versao boa por uma
// quebrada, e o publicado tem de continuar sendo o que ja funcionava.
func TestStagingPromocaoEUrollbackDoAppGerado(t *testing.T) {
	servico, err := NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := servico.Create(context.Background(), BuilderSpec{Name: "Gestao", Kind: BuilderFullstack})
	if err != nil {
		t.Fatal(err)
	}
	org := projeto.OrganizationID
	senha := "senha-de-staging-forte"

	// Sem publicar nao ha o que subir; sem senha nao existe credencial padrao.
	if _, err := servico.StageFullstack(context.Background(), projeto.ID, senha); err == nil {
		t.Fatal("staging de projeto NAO publicado deveria ser recusado")
	}
	if _, err := servico.StageFullstack(context.Background(), projeto.ID, "  "); err == nil {
		t.Fatal("staging sem senha deveria ser recusado (nao existe credencial padrao)")
	}
	// Kind sem backend proprio tambem e recusado com erro identificavel.
	outro, err := servico.Create(context.Background(), BuilderSpec{Name: "Jogo", Kind: BuilderGame})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servico.StageFullstack(context.Background(), outro.ID, senha); !errors.Is(err, ErrBuilderStagingUnsupported) {
		t.Fatalf("kind sem backend deveria recusar staging, veio: %v", err)
	}

	// v1 publicada e SUBIDA de verdade.
	publicadoV1, caminhoV1, err := servico.PublishLocal(context.Background(), projeto.ID)
	if err != nil {
		t.Fatal(err)
	}
	palco, err := servico.StageFullstack(context.Background(), projeto.ID, senha)
	if err != nil {
		t.Fatalf("staging falhou: %v", err)
	}
	porta := strings.TrimPrefix(palco.URL, "http://127.0.0.1:")
	if err := palco.Health(context.Background()); err != nil {
		t.Fatalf("palco deveria estar saudavel: %v", err)
	}
	// O app no palco ATENDE de verdade: login e CRUD por HTTP real.
	token := tokenDoLogin(t, porta, senha)
	if _, status := chamarApp(t, porta, http.MethodPost, "/api/items", token, map[string]any{"name": "Parafuso", "quantity": 2}); status != http.StatusCreated {
		t.Fatalf("criar item no palco deveria dar 201, veio %d", status)
	}

	// FAIL-CLOSED: palco derrubado NAO pode ser promovido, e o publicado nao muda.
	palco.Stop()
	if _, err := servico.PromoteIfHealthy(context.Background(), projeto.ID, org, palco); !errors.Is(err, ErrBuilderPromotionUnhealthy) {
		t.Fatalf("promocao de palco derrubado deveria ser recusada, veio: %v", err)
	}
	aposRecusa, err := servico.Get(projeto.ID)
	if err != nil {
		t.Fatal(err)
	}
	if aposRecusa.PublishedPath != publicadoV1.PublishedPath {
		t.Fatalf("recusa de promocao NAO pode mudar o publicado: antes=%s depois=%s", publicadoV1.PublishedPath, aposRecusa.PublishedPath)
	}

	// Palco saudavel: promocao acontece e a versao promovida fica registrada.
	saudavel, err := servico.StageFullstack(context.Background(), projeto.ID, senha)
	if err != nil {
		t.Fatal(err)
	}
	defer saudavel.Stop()
	promovido, err := servico.PromoteIfHealthy(context.Background(), projeto.ID, org, saudavel)
	if err != nil {
		t.Fatalf("promocao de palco saudavel falhou: %v", err)
	}
	if promovido.PublishedPath != saudavel.PublishedPath {
		t.Fatalf("publicado deveria ser a versao do palco: %s", promovido.PublishedPath)
	}

	// v2: nova versao publicada, depois ROLLBACK para v1 — e a v1 precisa CONTINUAR funcionando.
	if _, err := servico.ApplyVisualComponents(context.Background(), projeto.ID, []VisualComponent{{ID: "c1", Type: "heading"}}); err != nil {
		t.Fatalf("atualizar para v2 falhou: %v", err)
	}
	publicadoV2, caminhoV2, err := servico.PublishLocal(context.Background(), projeto.ID)
	if err != nil {
		t.Fatal(err)
	}
	if caminhoV2 == caminhoV1 {
		t.Fatalf("v2 deveria publicar em caminho proprio: %s", caminhoV2)
	}
	revertido, caminhoRevertido, err := servico.RollbackPublish(context.Background(), projeto.ID, org, 1)
	if err != nil {
		t.Fatalf("rollback falhou: %v", err)
	}
	if !strings.HasSuffix(caminhoRevertido, "v1") || revertido.PublishedPath != caminhoV1 {
		t.Fatalf("rollback deveria voltar para v1: caminho=%s published=%s", caminhoRevertido, revertido.PublishedPath)
	}
	if publicadoV2.Version < 2 {
		t.Fatalf("a segunda publicacao deveria ser versao >= 2, veio %d", publicadoV2.Version)
	}
	// A versao revertida sobe e atende: rollback nao pode devolver um ponteiro para algo quebrado.
	palcoRevertido, err := servico.StageFullstack(context.Background(), projeto.ID, senha)
	if err != nil {
		t.Fatalf("staging apos rollback falhou: %v", err)
	}
	defer palcoRevertido.Stop()
	if err := palcoRevertido.Health(context.Background()); err != nil {
		t.Fatalf("versao revertida deveria estar saudavel: %v", err)
	}
	if palcoRevertido.Version != 1 {
		t.Fatalf("palco apos rollback deveria ser a v1, veio v%d", palcoRevertido.Version)
	}
	tokenRevertido := tokenDoLogin(t, strings.TrimPrefix(palcoRevertido.URL, "http://127.0.0.1:"), senha)
	if tokenRevertido == "" {
		t.Fatal("versao revertida nao autenticou: rollback devolveu algo quebrado")
	}

	t.Logf("STAGING E ROLLBACK: staging=v%d saudavel=ok promocao=ok recusa_sem_saude=ok v1=%s v2=%s rollback=v1",
		palco.Version, caminhoV1, caminhoV2)
}
