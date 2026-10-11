package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// PERSISTÊNCIA DA JANELA DE RATE LIMIT (E-170). O limite declarado era "rate-limit/replay em memória" — e vale
// separar: a proteção de REPLAY de webhook já era durável; o que estava em memória era a JANELA DE RATE LIMIT.
//
// O defeito concreto: reiniciar o processo ZERA a cota. Quem estourou o limite e reinicia o serviço (ou explora
// um reinício) volta a ter janela cheia. Os testes travam:
//
//  1. com store, a janela SOBREVIVE ao reinício (o evento seguinte continua limitado);
//  2. sem store, o estado DIZ que não há persistência (não finge durabilidade);
//  3. falha de leitura é REPORTADA e uma gravação bem-sucedida NÃO apaga esse registro;
//  4. janela vencida não é restaurada (não se herda uma cota velha).

// rateRuntime monta um runtime com o conector registrado e devolve o runtime e o ID do conector — que é NOVO a
// cada chamada, porque os CONECTORES também vivem em memória (limite declarado separadamente).
func rateRuntime(t *testing.T, org, secretEnv string, limite int, store *OmnichannelRateLimitStore) (*OmnichannelRuntime, string) {
	t.Helper()
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetRateLimit(limite)
	if store != nil {
		runtime.SetRateLimitStore(store)
	}
	t.Setenv(secretEnv, "segredo-de-teste-com-tamanho-suficiente")
	connector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: org, Channel: "whatsapp", Name: "WhatsApp", SecretEnv: secretEnv, Enabled: true,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return runtime, connector.ID
}

// rateEnviar ingere um evento assinado com o conector do PRÓPRIO runtime.
func rateEnviar(t *testing.T, runtime *OmnichannelRuntime, org, connectorID, id string) error {
	t.Helper()
	agora := time.Now().UTC()
	corpo := []byte(`{"external_message_id":"` + id + `"}`)
	assinatura, err := runtime.SignWebhook(connectorID, agora.Unix(), corpo)
	if err != nil {
		t.Fatal(err)
	}
	evento := OmnichannelEvent{ExternalMessageID: id, ContactID: "+5511999998888", MessageType: "text"}
	_, err = runtime.Ingest(org, "whatsapp", assinatura, agora.Unix(), corpo, evento, agora)
	return err
}

func TestOmnichannelRateLimitWindowSurvivesRestart(t *testing.T) {
	store := NewOmnichannelRateLimitStore(filepath.Join(t.TempDir(), "rate"))
	const segredo = "HADES_TEST_OMNI_RATE_SECRET"

	primeiro, conector := rateRuntime(t, "org-rate", segredo, 2, store)
	if !primeiro.RateLimitPersisted() {
		t.Fatal("com store configurado, o estado precisa declarar persistência")
	}
	aceitos := 0
	for i := range 3 {
		id := "wamid-rate-" + string(rune('a'+i))
		err := rateEnviar(t, primeiro, "org-rate", conector, id)
		switch {
		case err == nil:
			aceitos++
		case errors.Is(err, ErrOmnichannelRateLimited):
		default:
			t.Fatalf("erro inesperado: %v", err)
		}
	}
	if aceitos != 2 {
		t.Fatalf("a cota de 2 precisava valer no primeiro runtime (aceitos=%d)", aceitos)
	}
	if erro := primeiro.LastRateLimitError(); erro != "" {
		t.Fatalf("não podia haver erro de persistência: %s", erro)
	}

	// SEGUNDO runtime: é o "reinício do processo". A janela precisa continuar consumida.
	depois, conectorDepois := rateRuntime(t, "org-rate", segredo, 2, store)
	err := rateEnviar(t, depois, "org-rate", conectorDepois, "wamid-rate-depois")
	if !errors.Is(err, ErrOmnichannelRateLimited) {
		t.Fatalf("a janela precisa SOBREVIVER ao reinício (limite persistido), veio %v", err)
	}
}

func TestOmnichannelRateLimitWithoutStoreSaysSoInsteadOfPretending(t *testing.T) {
	runtime := NewOmnichannelRuntime(nil)
	if runtime.RateLimitPersisted() {
		t.Fatal("sem store, o estado NÃO pode declarar persistência")
	}
	if erro := runtime.LastRateLimitError(); erro != "" {
		t.Fatalf("sem store não há erro a reportar: %s", erro)
	}
}

func TestOmnichannelRateLimitReportsUnreadableWindow(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "rate-quebrado")
	if err := os.MkdirAll(raiz, 0o700); err != nil {
		t.Fatal(err)
	}
	// Arquivo corrompido: a janela não pode ser lida, e isso precisa APARECER — mesmo depois de uma gravação
	// bem-sucedida (o registro de leitura não pode ser apagado por um save que deu certo).
	if err := os.WriteFile(filepath.Join(raiz, "org-quebrada.json"), []byte("{isso não é json"), 0o600); err != nil {
		t.Fatal(err)
	}
	const segredo = "HADES_TEST_OMNI_BROKEN_SECRET"
	runtime, conector := rateRuntime(t, "org-quebrada", segredo, 5, NewOmnichannelRateLimitStore(raiz))
	if err := rateEnviar(t, runtime, "org-quebrada", conector, "wamid-quebrado"); err != nil {
		t.Fatalf("o evento continua sendo processado (a falha é da persistência, não do evento): %v", err)
	}
	if erro := runtime.LastRateLimitError(); erro == "" {
		t.Fatal("janela ilegível precisa ser REPORTADA, não escondida")
	} else if !strings.Contains(erro, "carregar janela") {
		t.Fatalf("o erro precisa identificar a LEITURA: %s", erro)
	}
}

func TestOmnichannelRateLimitDoesNotRestoreExpiredWindow(t *testing.T) {
	store := NewOmnichannelRateLimitStore(filepath.Join(t.TempDir(), "rate-vencido"))
	// Janela VENCIDA gravada no disco: não pode ser herdada como se fosse atual.
	velho := time.Now().UTC().Add(-2 * OmnichannelRateWindow)
	if err := store.Save("org-vencida", map[string]omnichannelRateLimitBucket{
		"whatsapp": {WindowStart: velho, Count: 99},
	}); err != nil {
		t.Fatal(err)
	}
	const segredo = "HADES_TEST_OMNI_EXPIRED_SECRET"
	runtime, conector := rateRuntime(t, "org-vencida", segredo, 2, store)
	if err := rateEnviar(t, runtime, "org-vencida", conector, "wamid-vencido"); err != nil {
		t.Fatalf("janela vencida não pode bloquear evento novo: %v", err)
	}
}
