package agent

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// ESTÁGIO 42 — LOCK ENTRE PROCESSOS NAS DEMAIS MUTAÇÕES (E-357).
//
// **MEDIDO antes**: só `IngestLiveCommerceEvent` e `RecordPlatformWarning` estavam travados. Criação, **aprovação**,
// avanço de estado e destaque seguiam apenas com mutex — e **uma aprovação perdida por gravação concorrente de outra
// instância é defeito de SEGURANÇA, não de conveniência**: a sessão entraria no ar sem o registro de quem aprovou.
//
// ***O mesmo defeito que o agendamento corrigiu em E-323 (double booking) reaparece aqui como "duas coisas ditas sim
// para o mesmo estado".***

func TestDuasInstanciasCriandoSessaoNaoPerdemNenhuma(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "live")
	instanciaA := NewLiveCommerceStore(raiz)
	instanciaB := NewLiveCommerceStore(raiz)
	agora := time.Now().UTC()

	var wg sync.WaitGroup
	wg.Add(2)
	erros := make([]error, 2)
	go func() {
		defer wg.Done()
		_, erros[0] = instanciaA.CreateLiveSession(LiveSession{
			OrganizationID: "org-a", ProjectID: "pA",
			AuthorizedUntil: agora.Add(time.Hour),
		})
	}()
	go func() {
		defer wg.Done()
		_, erros[1] = instanciaB.CreateLiveSession(LiveSession{
			OrganizationID: "org-a", ProjectID: "pB",
			AuthorizedUntil: agora.Add(time.Hour),
		})
	}()
	wg.Wait()
	for i, err := range erros {
		if err != nil {
			t.Fatalf("criação %d falhou: %v", i, err)
		}
	}

	// ARMADILHA: as DUAS sessões precisam existir. Sem lock + recarga, a segunda gravação apaga a primeira.
	sessoes, err := instanciaA.LiveSessions("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessoes) != 2 {
		t.Fatalf("ARMADILHA: perda de escrita entre instâncias — %d sessão(ões) em vez de 2", len(sessoes))
	}
	deB, err := instanciaB.LiveSessions("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(deB) != 2 {
		t.Fatalf("a segunda instância precisa ver as duas sessões: %d", len(deB))
	}
}

// A APROVAÇÃO é decisão de responsabilidade: perder o registro de quem aprovou é defeito de segurança.
func TestAprovacaoRegistradaNumaInstanciaNaoSePerde(t *testing.T) {
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
	// A instância A aprova; a instância B tenta avançar ao mesmo tempo.
	var wg sync.WaitGroup
	wg.Add(2)
	var aprovada LiveSession
	var erroAprovacao error
	go func() {
		defer wg.Done()
		aprovada, erroAprovacao = instanciaA.ApproveLiveSession("org-a", sessao.ID, "operador-responsavel", "ok")
	}()
	go func() { defer wg.Done(); _, _ = instanciaB.AdvanceLiveSession("org-a", sessao.ID, LiveLive, agora) }()
	wg.Wait()
	if erroAprovacao != nil {
		t.Fatalf("aprovação: %v", erroAprovacao)
	}
	// O retorno da aprovação já precisa identificar quem aprovou.
	if aprovada.Approval.Actor != "operador-responsavel" {
		t.Fatalf("o retorno da aprovação precisa registrar o aprovador: %+v", aprovada.Approval)
	}

	// ARMADILHA: depois da corrida, o REGISTRO de quem aprovou não pode ter sido perdido pela gravação da vizinha.
	sessoes, err := instanciaB.LiveSessions("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessoes) != 1 {
		t.Fatalf("esperava 1 sessão, veio %d", len(sessoes))
	}
	final := sessoes[0]
	if final.Approval.Actor == "" || final.Approval.Actor != "operador-responsavel" {
		t.Fatalf("ARMADILHA: o registro de quem aprovou se perdeu na corrida entre instâncias: %+v", final.Approval)
	}
	// E a sessão NUNCA pode estar no ar sem aprovação identificada.
	if final.State == LiveLive && final.Approval.Actor == "" {
		t.Fatalf("sessão no ar sem aprovador identificado: %+v", final)
	}
}

// Avanço de estado concorrente não pode produzir estado que nenhuma das instâncias pediu.
func TestAvancoConcorrenteMantemEstadoCoerente(t *testing.T) {
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
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = instanciaA.AdvanceLiveSession("org-a", sessao.ID, LiveReady, agora) }()
	go func() { defer wg.Done(); _, _ = instanciaB.AdvanceLiveSession("org-a", sessao.ID, LiveReady, agora) }()
	wg.Wait()

	sessoes, err := instanciaA.LiveSessions("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessoes) != 1 {
		t.Fatalf("a sessão precisa continuar existindo uma vez: %d", len(sessoes))
	}
	if sessoes[0].State != LiveReady {
		t.Fatalf("estado depois de dois avanços para READY precisa ser READY, veio %q", sessoes[0].State)
	}
	// E as duas instâncias concordam (nenhuma vê estado antigo).
	deB, err := instanciaB.LiveSessions("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if deB[0].State != sessoes[0].State {
		t.Fatalf("as duas instâncias divergem: %q vs %q", sessoes[0].State, deB[0].State)
	}
}
