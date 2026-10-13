package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// ESTÁGIO 42 — NÚCLEO DE SEGURANÇA: TESTES (E-332).
//
// O complemento proíbe evasão, spam, falsa compra e "live infinita". Cada proibição está aqui como direção negativa —
// se alguém afrouxar a regra, o teste falha.

func liveFixture(t *testing.T) (*LiveCommerceStore, LiveSession) {
	t.Helper()
	store := NewLiveCommerceStore(filepath.Join(t.TempDir(), "live"))
	sessao, err := store.CreateLiveSession(LiveSession{
		OrganizationID:  "org-a",
		ProjectID:       "proj-1",
		AuthorizedUntil: time.Now().UTC().Add(90 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return store, sessao
}

func TestNaoEntraEmLiveSemAprovacaoHumana(t *testing.T) {
	store, sessao := liveFixture(t)
	agora := time.Now().UTC()
	if _, err := store.AdvanceLiveSession("org-a", sessao.ID, LiveReady, agora); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AdvanceLiveSession("org-a", sessao.ID, LiveWaitingApproval, agora); err != nil {
		t.Fatal(err)
	}
	// ARMADILHA: sem aprovação registrada, entrar no ar é RECUSADO.
	if _, err := store.AdvanceLiveSession("org-a", sessao.ID, LiveLive, agora); !errors.Is(err, ErrLiveApprovalRequired) {
		t.Fatalf("LIVE sem aprovação do operador precisa ser recusado: %v", err)
	}
	sessoes, _ := store.LiveSessions("org-a")
	if sessoes[0].State != LiveWaitingApproval {
		t.Fatalf("a sessão precisa permanecer em WAITING_APPROVAL: %q", sessoes[0].State)
	}
	// Com aprovação humana identificada, entra.
	if _, err := store.ApproveLiveSession("org-a", sessao.ID, "operador.responsavel", "pré-voo conferido"); err != nil {
		t.Fatal(err)
	}
	noAr, err := store.AdvanceLiveSession("org-a", sessao.ID, LiveLive, agora)
	if err != nil {
		t.Fatalf("com aprovação precisa entrar no ar: %v", err)
	}
	if noAr.State != LiveLive || noAr.StartedAt == nil {
		t.Fatalf("entrar no ar precisa registrar o início: %+v", noAr)
	}
	// Aprovação exige responsável identificado (não existe "aprovado pelo sistema").
	if _, err := store.ApproveLiveSession("org-a", sessao.ID, "  ", "sem nome"); !errors.Is(err, ErrLiveCommerceInvalid) {
		t.Fatalf("aprovação anônima precisa ser recusada: %v", err)
	}
}

func TestAvisoDaPlataformaRecuaENaoEvade(t *testing.T) {
	store, sessao := liveFixture(t)
	agora := time.Now().UTC()
	for _, passo := range []string{LiveReady, LiveWaitingApproval} {
		if _, err := store.AdvanceLiveSession("org-a", sessao.ID, passo, agora); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.ApproveLiveSession("org-a", sessao.ID, "operador", "ok"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AdvanceLiveSession("org-a", sessao.ID, LiveLive, agora); err != nil {
		t.Fatal(err)
	}

	// AVISO DA PLATAFORMA: recuar, registrar e escalar — nunca ocultar.
	recuada, err := store.RecordPlatformWarning("org-a", sessao.ID, "aviso de diretriz",
		"captura e identificador do aviso", agora)
	if err != nil {
		t.Fatal(err)
	}
	if recuada.State != LiveExternallyBlocked {
		t.Fatalf("com aviso da plataforma a sessão NÃO pode continuar no ar: %q", recuada.State)
	}
	if !recuada.HumanReview || len(recuada.Incidents) == 0 || len(recuada.Moderation) == 0 {
		t.Fatalf("o aviso precisa gerar incidente, evidência e exigir revisão humana: %+v", recuada)
	}
	// ARMADILHA: não existe caminho de volta ao ar sem passar por decisão humana explícita.
	if _, err := store.AdvanceLiveSession("org-a", sessao.ID, LiveLive, agora); !errors.Is(err, ErrLiveInvalidTransition) {
		t.Fatalf("sessão bloqueada pela plataforma não pode voltar ao ar sozinha: %v", err)
	}
	// O caminho permitido é ENCERRAR (recuo), e ele funciona.
	encerrada, err := store.AdvanceLiveSession("org-a", sessao.ID, LiveEnded, agora)
	if err != nil {
		t.Fatalf("encerrar precisa ser permitido: %v", err)
	}
	if encerrada.State != LiveEnded || encerrada.EndedAt == nil {
		t.Fatalf("encerrar precisa registrar o fim: %+v", encerrada)
	}
}

func TestFimAutorizadoEncerraSozinhoENaoDeixaEntrarDepois(t *testing.T) {
	agora := time.Now().UTC()
	store := NewLiveCommerceStore(filepath.Join(t.TempDir(), "live"))
	// Prazo curto, deliberadamente.
	sessao, err := store.CreateLiveSession(LiveSession{OrganizationID: "org-a", ProjectID: "p1",
		AuthorizedUntil: agora.Add(2 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	for _, passo := range []string{LiveReady, LiveWaitingApproval} {
		if _, err := store.AdvanceLiveSession("org-a", sessao.ID, passo, agora); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.ApproveLiveSession("org-a", sessao.ID, "operador", "ok"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AdvanceLiveSession("org-a", sessao.ID, LiveLive, agora); err != nil {
		t.Fatal(err)
	}
	// Passou do fim autorizado: a varredura ENCERRA (fail-closed no tempo).
	depois := agora.Add(5 * time.Second)
	encerradas, err := store.SweepLiveSessions("org-a", depois)
	if err != nil {
		t.Fatal(err)
	}
	if len(encerradas) != 1 || encerradas[0].State != LiveEnded {
		t.Fatalf("o fim autorizado precisa encerrar a sessão automaticamente: %+v", encerradas)
	}
	// ARMADILHA (corrigida: minha asserção chamava LiveReady, que É legal em DRAFT — o que precisa ser provado é o
	// caminho ATÉ o ar): com o prazo já vencido, percorrer o caminho legal até LIVE NÃO pode colocar a sessão no ar —
	// ela tem de ENCERRAR sozinha, fail-closed no tempo.
	outra, err := store.CreateLiveSession(LiveSession{OrganizationID: "org-a", ProjectID: "p2",
		AuthorizedUntil: agora.Add(1 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	vencido := depois.Add(10 * time.Second)
	for _, passo := range []string{LiveReady, LiveWaitingApproval} {
		if _, err := store.AdvanceLiveSession("org-a", outra.ID, passo, vencido); err != nil {
			t.Fatalf("passo %s: %v", passo, err)
		}
	}
	if _, err := store.ApproveLiveSession("org-a", outra.ID, "operador", "aprovado antes do prazo"); err != nil {
		t.Fatal(err)
	}
	tardia, err := store.AdvanceLiveSession("org-a", outra.ID, LiveLive, vencido)
	if err != nil {
		t.Fatalf("entrar no ar após o prazo não é erro de transição: o motor deve ENCERRAR: %v", err)
	}
	if tardia.State != LiveEnded {
		t.Fatalf("ARMADILHA: entrou no ar depois do fim autorizado (estado %q) — o tempo é limite, não sugestão", tardia.State)
	}
	if tardia.EndReason == "" {
		t.Fatal("o encerramento automático precisa registrar o motivo")
	}
	// E prazo no passado nem é aceito na criação: NÃO se promete live infinita nem prazo vencido.
	if _, err := store.CreateLiveSession(LiveSession{OrganizationID: "org-a", ProjectID: "p3",
		AuthorizedUntil: agora.Add(-time.Minute)}); !errors.Is(err, ErrLiveCommerceInvalid) {
		t.Fatalf("criar sessão com fim autorizado no passado precisa ser recusado: %v", err)
	}
	if _, err := store.CreateLiveSession(LiveSession{OrganizationID: "org-a", ProjectID: "p4"}); !errors.Is(err, ErrLiveCommerceInvalid) {
		t.Fatalf("criar sessão SEM fim autorizado precisa ser recusado (nada de live infinita): %v", err)
	}
}

func TestEventoDeCarrinhoNaoEhVendaConfirmada(t *testing.T) {
	store, _ := liveFixture(t)
	// Evento de carrinho: legítimo, mas NÃO é venda.
	if _, _, err := store.IngestLiveCommerceEvent(CommerceEvent{OrganizationID: "org-a", ExternalID: "carrinho-1",
		Kind: "cart_added", SKU: "SKU-1", SignatureVerified: true}); err != nil {
		t.Fatal(err)
	}
	resumo, err := store.ConfirmedSales("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if resumo.Orders != 0 || resumo.Revenue != 0 {
		t.Fatalf("ARMADILHA: evento de carrinho foi contado como venda (receita %d, pedidos %d)", resumo.Revenue, resumo.Orders)
	}
	if resumo.CartAdds != 1 {
		t.Fatalf("o carrinho precisa ser contado à parte, como informação: %+v", resumo)
	}
	// Compra confirmada SEM pedido identificado é recusada: não se inventa venda.
	if _, _, err := store.IngestLiveCommerceEvent(CommerceEvent{OrganizationID: "org-a", ExternalID: "compra-sem-pedido",
		Kind: "purchase_confirmed", SKU: "SKU-1", AmountCents: 9900, SignatureVerified: true}); !errors.Is(err, ErrLiveCommerceInvalid) {
		t.Fatalf("compra confirmada sem OrderID precisa ser recusada: %v", err)
	}
	// Compra confirmada com pedido conta UMA vez.
	if _, _, err := store.IngestLiveCommerceEvent(CommerceEvent{OrganizationID: "org-a", ExternalID: "compra-1",
		Kind: "purchase_confirmed", SKU: "SKU-1", OrderID: "PED-1", AmountCents: 9900, SignatureVerified: true}); err != nil {
		t.Fatal(err)
	}
	resumo, _ = store.ConfirmedSales("org-a")
	if resumo.Orders != 1 || resumo.Revenue != 9900 {
		t.Fatalf("a venda confirmada precisa contar: %+v", resumo)
	}
}

func TestEventoRepetidoNaoInflaReceitaNemDuplicaEfeito(t *testing.T) {
	store, _ := liveFixture(t)
	evento := CommerceEvent{OrganizationID: "org-a", ExternalID: "compra-replay", Kind: "purchase_confirmed",
		SKU: "SKU-9", OrderID: "PED-9", AmountCents: 5000, SignatureVerified: true}
	primeiro, replay, err := store.IngestLiveCommerceEvent(evento)
	if err != nil {
		t.Fatal(err)
	}
	if replay {
		t.Fatal("a primeira ingestão não é replay")
	}
	segundo, replay2, err := store.IngestLiveCommerceEvent(evento)
	if err != nil {
		t.Fatal(err)
	}
	if !replay2 || segundo.ID != primeiro.ID {
		t.Fatalf("o replay precisa devolver o MESMO evento: %+v", segundo)
	}
	// ARMADILHA: receita contada UMA vez, apesar de duas entregas.
	resumo, _ := store.ConfirmedSales("org-a")
	if resumo.Orders != 1 || resumo.Revenue != 5000 {
		t.Fatalf("ARMADILHA: evento repetido inflou a receita: %+v", resumo)
	}
	// O mesmo pedido com OUTRO external_id também não soma duas vezes (dedupe por pedido).
	if _, _, err := store.IngestLiveCommerceEvent(CommerceEvent{OrganizationID: "org-a", ExternalID: "compra-outro-id",
		Kind: "purchase_confirmed", SKU: "SKU-9", OrderID: "PED-9", AmountCents: 5000, SignatureVerified: true}); err != nil {
		t.Fatal(err)
	}
	resumo, _ = store.ConfirmedSales("org-a")
	if resumo.Orders != 1 || resumo.Revenue != 5000 {
		t.Fatalf("o mesmo PEDIDO não pode somar duas vezes: %+v", resumo)
	}
}

func TestEventoSemAssinaturaVerificadaEhRecusado(t *testing.T) {
	store, _ := liveFixture(t)
	if _, _, err := store.IngestLiveCommerceEvent(CommerceEvent{OrganizationID: "org-a", ExternalID: "sem-assinatura",
		Kind: "purchase_confirmed", OrderID: "PED-X", AmountCents: 100}); !errors.Is(err, ErrLiveEventUnverified) {
		t.Fatalf("evento sem assinatura verificada precisa ser recusado: %v", err)
	}
	if _, _, err := store.IngestLiveCommerceEvent(CommerceEvent{OrganizationID: "org-a", ExternalID: "tipo-estranho",
		Kind: "presente_misterioso", SignatureVerified: true}); !errors.Is(err, ErrLiveCommerceInvalid) {
		t.Fatalf("tipo de evento desconhecido precisa ser recusado: %v", err)
	}
}

func TestLiveNaoMisturaOrganizacoesENaoDuplicaModulos(t *testing.T) {
	store, sessao := liveFixture(t)
	// A vizinha não vê a sessão nem consegue operá-la.
	vizinhas, err := store.LiveSessions("org-vizinha")
	if err != nil {
		t.Fatal(err)
	}
	if len(vizinhas) != 0 {
		t.Fatalf("a organização vizinha não pode ver sessões alheias: %+v", vizinhas)
	}
	if _, err := store.AdvanceLiveSession("org-vizinha", sessao.ID, LiveReady, time.Now().UTC()); !errors.Is(err, ErrLiveSessionNotFound) {
		t.Fatalf("a vizinha não pode operar sessão de outra organização: %v", err)
	}
	// Destaque referencia SKU do catálogo existente — sem SKU, recusa.
	if _, err := store.ShowcaseForProduct("org-a", sessao.ID, ProductShowcase{Position: 1}, time.Now().UTC()); !errors.Is(err, ErrLiveCommerceInvalid) {
		t.Fatalf("destaque sem SKU precisa ser recusado (não se inventa produto): %v", err)
	}
	// Destaque como SUGESTÃO ao operador (não automatiza clique na plataforma).
	comDestaque, err := store.ShowcaseForProduct("org-a", sessao.ID,
		ProductShowcase{SKU: "SKU-1", Position: 1, PinnedByPlatform: false}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(comDestaque.Showcase) != 1 || comDestaque.Showcase[0].PinnedByPlatform {
		t.Fatalf("o destaque precisa ficar registrado como sugestão ao operador: %+v", comDestaque.Showcase)
	}
}
