package agent

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// ESTÁGIO 42 — DUAS INSTÂNCIAS, MESMO DIRETÓRIO (E-356).
//
// Duas instâncias do store sobre a **mesma raiz** é o equivalente em teste de **dois processos** do servidor: cada uma
// tem o seu cache em memória e a sua própria trava. **MEDIDO antes**: o store não usava lock de arquivo nem
// invalidação de cache — a mesma lacuna que o agendamento declarou e corrigiu em E-323.
//
// ***Um evento aceito numa instância e apagado pela outra é o mesmo defeito de um cliente confirmado em horário já
// ocupado: o sistema disse "sim" para duas coisas que não cabem juntas.***

func TestSegundaInstanciaVeSessaoCriadaPelaPrimeira(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "live")
	instanciaA := NewLiveCommerceStore(raiz)
	instanciaB := NewLiveCommerceStore(raiz)

	sessao, err := instanciaA.CreateLiveSession(LiveSession{
		OrganizationID: "org-a", ProjectID: "p1",
		AuthorizedUntil: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	// ARMADILHA: sem invalidação de cache, a instância B responde do cache vazio e "não vê" a sessão.
	sessoes, err := instanciaB.LiveSessions("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessoes) != 1 || sessoes[0].ID != sessao.ID {
		t.Fatalf("ARMADILHA: a segunda instância não viu a sessão criada pela primeira: %+v", sessoes)
	}
}

func TestDuasInstanciasNaoPerdemEventoAceito(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "live")
	instanciaA := NewLiveCommerceStore(raiz)
	instanciaB := NewLiveCommerceStore(raiz)

	// Dois eventos DIFERENTES, aceitos ao mesmo tempo por instâncias diferentes.
	eventoA := CommerceEvent{
		OrganizationID: "org-a", ExternalID: "compra-A", Kind: "purchase_confirmed",
		SKU: "SKU-1", OrderID: "PED-A", AmountCents: 1000, SignatureVerified: true,
	}
	eventoB := CommerceEvent{
		OrganizationID: "org-a", ExternalID: "compra-B", Kind: "purchase_confirmed",
		SKU: "SKU-2", OrderID: "PED-B", AmountCents: 2000, SignatureVerified: true,
	}

	var wg sync.WaitGroup
	wg.Add(2)
	erros := make([]error, 2)
	go func() { defer wg.Done(); _, _, erros[0] = instanciaA.IngestLiveCommerceEvent(eventoA) }()
	go func() { defer wg.Done(); _, _, erros[1] = instanciaB.IngestLiveCommerceEvent(eventoB) }()
	wg.Wait()
	for i, err := range erros {
		if err != nil {
			t.Fatalf("ingestão %d falhou: %v", i, err)
		}
	}

	// ARMADILHA: os DOIS eventos precisam sobreviver. Sem lock + recarga, cada instância grava a sua versão e uma
	// apaga o evento da outra — a receita apareceria com apenas um pedido.
	resumo, err := instanciaA.ConfirmedSales("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if resumo.Orders != 2 || resumo.Revenue != 3000 {
		t.Fatalf("ARMADILHA: perda de escrita entre instâncias — pedidos=%d receita=%d (esperado 2 e 3000)",
			resumo.Orders, resumo.Revenue)
	}
	// E o estado final precisa ser o mesmo visto das DUAS instâncias.
	deB, err := instanciaB.ConfirmedSales("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if deB.Orders != resumo.Orders || deB.Revenue != resumo.Revenue {
		t.Fatalf("as duas instâncias precisam concordar: %+v vs %+v", resumo, deB)
	}
}

func TestAvisoRegistradoNumaInstanciaNaoEhPerdidoPelaOutra(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "live")
	instanciaA := NewLiveCommerceStore(raiz)
	instanciaB := NewLiveCommerceStore(raiz)
	agora := time.Now().UTC()

	sessao, err := instanciaA.CreateLiveSession(LiveSession{
		OrganizationID: "org-a", ProjectID: "p1",
		AuthorizedUntil: agora.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, passo := range []string{LiveReady, LiveWaitingApproval} {
		if _, err := instanciaA.AdvanceLiveSession("org-a", sessao.ID, passo, agora); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := instanciaA.ApproveLiveSession("org-a", sessao.ID, "operador", "ok"); err != nil {
		t.Fatal(err)
	}
	if _, err := instanciaA.AdvanceLiveSession("org-a", sessao.ID, LiveLive, agora); err != nil {
		t.Fatal(err)
	}
	// Aviso da plataforma chega na instância A; a instância B precisa VER o incidente e a exigência de revisão.
	if _, err := instanciaA.RecordPlatformWarning("org-a", sessao.ID, "diretriz", "evidência", agora); err != nil {
		t.Fatal(err)
	}
	sessoes, err := instanciaB.LiveSessions("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessoes) != 1 {
		t.Fatalf("a segunda instância precisa ver a sessão: %d", len(sessoes))
	}
	final := sessoes[0]
	if final.State == LiveLive {
		t.Fatalf("ARMADILHA: sessão no ar apesar do aviso registrado na outra instância: %+v", final)
	}
	if !final.HumanReview || len(final.Incidents) == 0 {
		t.Fatalf("o incidente e a revisão humana precisam estar visíveis na outra instância: %+v", final)
	}
}
