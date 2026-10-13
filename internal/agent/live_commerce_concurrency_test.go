package agent

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// SONDA DE CONCORRÊNCIA — ESTÁGIO 42 (E-355).
//
// O núcleo de segurança do estágio 42 foi provado com testes **sequenciais**. Concorrência é outra pergunta: **duas
// operações legítimas chegando ao mesmo tempo** podem deixar a sessão num estado que nenhuma delas pediu.
//
// Aqui a invariante é: depois de qualquer corrida entre **varredura de fim autorizado**, **encerramento manual**,
// **aviso da plataforma** e **ingestão de evento**, a sessão continua **consistente** — terminal quando deve ser,
// nunca "no ar" sem aprovação, nunca com dois fins.

func sessaoNoAr(t *testing.T, store *LiveCommerceStore, id string, agora time.Time) LiveSession {
	t.Helper()
	for _, passo := range []string{LiveReady, LiveWaitingApproval} {
		if _, err := store.AdvanceLiveSession("org-a", id, passo, agora); err != nil {
			t.Fatalf("passo %s: %v", passo, err)
		}
	}
	if _, err := store.ApproveLiveSession("org-a", id, "operador", "ok"); err != nil {
		t.Fatal(err)
	}
	sessao, err := store.AdvanceLiveSession("org-a", id, LiveLive, agora)
	if err != nil {
		t.Fatalf("entrar no ar: %v", err)
	}
	return sessao
}

func TestCorridaEntreVarreduraEncerramentoEAvisoMantemSessaoConsistente(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "live")
	store := NewLiveCommerceStore(raiz)
	agora := time.Now().UTC()
	sessao, err := store.CreateLiveSession(LiveSession{
		OrganizationID: "org-a", ProjectID: "p1",
		AuthorizedUntil: agora.Add(1 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	sessaoNoAr(t, store, sessao.ID, agora)

	// Depois do fim autorizado, três operações LEGÍTIMAS competem pela mesma sessão.
	depois := agora.Add(3 * time.Second)

	var wg sync.WaitGroup
	resultados := make([]error, 3)
	wg.Add(3)
	go func() {
		defer wg.Done()
		_, resultados[0] = store.SweepLiveSessions("org-a", depois)
	}()
	go func() {
		defer wg.Done()
		_, resultados[1] = store.AdvanceLiveSession("org-a", sessao.ID, LiveEnded, depois)
	}()
	go func() {
		defer wg.Done()
		_, resultados[2] = store.RecordPlatformWarning("org-a", sessao.ID, "aviso", "evidência", depois)
	}()
	wg.Wait()

	sessoes, err := store.LiveSessions("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessoes) != 1 {
		t.Fatalf("a sessão precisa continuar existindo UMA vez: %d", len(sessoes))
	}
	final := sessoes[0]
	// INVARIANTE 1: nunca fica "no ar" depois do fim autorizado.
	if final.State == LiveLive || final.State == LivePaused {
		t.Fatalf("ARMADILHA: sessão continuou no ar (%q) depois do fim autorizado", final.State)
	}
	// INVARIANTE 2: o fim, quando registrado, tem motivo e data — e não é sobrescrito por valor vazio.
	if final.State == LiveEnded {
		if final.EndedAt == nil || final.EndReason == "" {
			t.Fatalf("sessão encerrada precisa de data e motivo: %+v", final)
		}
	}
	// INVARIANTE 3: aviso da plataforma gera incidente + revisão humana, em qualquer ordem.
	if len(final.Moderation) > 0 && (!final.HumanReview || len(final.Incidents) == 0) {
		t.Fatalf("aviso registrado exige incidente e revisão humana: %+v", final)
	}
	// E a leitura seguinte precisa ser estável (estado não "volta" entre leituras).
	novamente, err := store.LiveSessions("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if novamente[0].State != final.State {
		t.Fatalf("o estado oscilou entre leituras: %q -> %q", final.State, novamente[0].State)
	}
	t.Logf("corrida medida: estado final=%q encerrada=%v avisos=%d incidentes=%d revisao=%v erros=%v",
		final.State, final.EndedAt != nil, len(final.Moderation), len(final.Incidents), final.HumanReview, resultados)
}

// Ingestão concorrente do MESMO evento: exatamente um efeito (idempotência sob corrida).
func TestIngestaoConcorrenteDoMesmoEventoNaoDuplicaReceita(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "live")
	store := NewLiveCommerceStore(raiz)
	evento := CommerceEvent{
		OrganizationID: "org-a", ExternalID: "compra-corrida", Kind: "purchase_confirmed",
		SKU: "SKU-1", OrderID: "PED-1", AmountCents: 5000, SignatureVerified: true,
	}

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	erros := make([]error, goroutines)
	replays := make([]bool, goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, replay, err := store.IngestLiveCommerceEvent(evento)
			erros[i], replays[i] = err, replay
		}()
	}
	wg.Wait()

	for i, err := range erros {
		if err != nil {
			t.Fatalf("ingestão %d falhou: %v", i, err)
		}
	}
	resumo, err := store.ConfirmedSales("org-a")
	if err != nil {
		t.Fatal(err)
	}
	// ARMADILHA: 8 entregas do MESMO pedido não podem virar 8 vendas nem 8x receita.
	if resumo.Orders != 1 || resumo.Revenue != 5000 {
		t.Fatalf("ARMADILHA: ingestão concorrente inflou a receita: %+v", resumo)
	}
	primeiros := 0
	for _, replay := range replays {
		if !replay {
			primeiros++
		}
	}
	if primeiros != 1 {
		t.Fatalf("exatamente UMA entrega é a primeira; as demais são replay: %d", primeiros)
	}
}

// Aviso da plataforma concorrente com a varredura: nunca pode terminar "no ar" com incidente pendente.
func TestAvisoDuranteVarreduraNaoDeixaSessaoNoAr(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "live")
	store := NewLiveCommerceStore(raiz)
	agora := time.Now().UTC()
	sessao, err := store.CreateLiveSession(LiveSession{
		OrganizationID: "org-a", ProjectID: "p1",
		AuthorizedUntil: agora.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	sessaoNoAr(t, store, sessao.ID, agora)

	var wg sync.WaitGroup
	wg.Add(2)
	var erroAviso, erroVarredura error
	go func() {
		defer wg.Done()
		_, erroAviso = store.RecordPlatformWarning("org-a", sessao.ID, "diretriz", "id", agora)
	}()
	go func() {
		defer wg.Done()
		// 🚨 ADV-54: a varredura era um NO-OP — a sessão estava autorizada por 30 min e a varredura era chamada com
		// `agora+1s`, então o guard `now.Before(AuthorizedUntil)` fazia `continue` e NADA era encerrado. *Uma "corrida"
		// em que um dos lados não faz nada não é corrida: é rótulo.* Aqui a varredura age de verdade (depois da
		// autorização), para haver disputa real pelo mesmo estado.
		_, erroVarredura = store.SweepLiveSessions("org-a", agora.Add(31*time.Minute))
	}()
	wg.Wait()
	if erroAviso != nil {
		t.Fatalf("registrar o aviso não pode falhar: %v", erroAviso)
	}
	if erroVarredura != nil {
		t.Fatalf("a varredura não pode falhar: %v", erroVarredura)
	}

	sessoes, err := store.LiveSessions("org-a")
	if err != nil {
		t.Fatalf("listar as sessões: %v", err)
	}
	if len(sessoes) == 0 {
		t.Fatal("ARMADILHA: a sessão desapareceu — verificação vácua (lista vazia) em vez de propriedade")
	}
	final := sessoes[0]
	// 🚨 ADV-54 (a asserção que NÃO disparava): a versão anterior exigia `HumanReview && State == LiveLive`, e quando
	// o aviso era **PERDIDO** — exatamente o defeito que este teste promete guardar — o resultado era
	// `HumanReview=false, State=LiveLive`, deixando a conjunção **FALSA** e o teste **VERDE** com o mecanismo de
	// segurança desligado. *Asserção que não dispara no modo de falha prometido não guarda nada.*
	if !final.HumanReview {
		t.Fatalf("ARMADILHA: o aviso da plataforma foi PERDIDO — a sessão ficou sem revisão humana exigida: %+v", final)
	}
	if len(final.Incidents) == 0 {
		t.Fatalf("ARMADILHA: o incidente do aviso não foi registrado (aviso perdido na corrida): %+v", final)
	}
}

var _ = context.Background
