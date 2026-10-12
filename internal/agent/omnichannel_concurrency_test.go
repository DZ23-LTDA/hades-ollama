package agent

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// CONCORRÊNCIA NO OMNICHANNEL (E-176).
//
// Em E-175 a medição ficou **declarada como isolada**: "não sob carga concorrente nem contenção do mutex". Aqui a
// ingestão é exercitada com VÁRIAS goroutines ao mesmo tempo, e os testes travam as propriedades que a
// concorrência pode quebrar:
//
//  1. entrega REPETIDA, vinda de várias goroutines em paralelo, é deduplicada (aceita **uma** vez);
//  2. uma organização **nunca** enxerga a mensagem da outra;
//  3. o rate limit **conta certo** sob concorrência (aceitas + recusadas = total);
//  4. o cache de eventos respeita o teto sem perder a consistência contábil.

const segredoConcorrencia = "segredo-de-teste-com-tamanho-suficiente"

func runtimeConcorrente(t *testing.T, limite int) (*OmnichannelRuntime, map[string]string) {
	t.Helper()
	secretEnv := "HADES_TEST_CONC_SECRET"
	t.Setenv(secretEnv, segredoConcorrencia)
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetRateLimit(limite)
	runtime.SetDedupeStore(NewOmnichannelDedupeStore(t.TempDir()))
	agora := time.Now().UTC()
	conectores := map[string]string{}
	for _, org := range []string{"org-conc-a", "org-conc-b"} {
		conector, err := runtime.RegisterConnector(OmnichannelConnector{
			OrganizationID: org, Channel: "whatsapp", Name: org, SecretEnv: secretEnv, Enabled: true,
		}, agora)
		if err != nil {
			t.Fatal(err)
		}
		conectores[org] = conector.ID
	}
	return runtime, conectores
}

func enviarConcorrente(t *testing.T, runtime *OmnichannelRuntime, org, connectorID, ident string) (OmnichannelEvent, error) {
	t.Helper()
	agora := time.Now().UTC()
	corpo := []byte(`{"external_message_id":"` + ident + `"}`)
	assinatura, err := runtime.SignWebhook(connectorID, agora.Unix(), corpo)
	if err != nil {
		return OmnichannelEvent{}, err
	}
	evento := OmnichannelEvent{ExternalMessageID: ident, ContactID: "+5511999998888", MessageType: "text"}
	return runtime.Ingest(org, "whatsapp", assinatura, agora.Unix(), corpo, evento, agora)
}

// A MESMA entrega chegando de várias goroutines ao mesmo tempo precisa ser aceita UMA única vez.
//
// Este teste é o que procura o TOCTOU: a checagem de dedupe e a gravação não acontecem no mesmo instante, então
// duas goroutines poderiam passar pela checagem antes de qualquer uma gravar.
func TestOmnichannelConcurrentDuplicateDeliveryIsAcceptedOnce(t *testing.T) {
	runtime, conectores := runtimeConcorrente(t, 1_000_000)
	const goroutines = 8
	var aceitas, duplicadas, falhas atomic.Int64
	var inicio sync.WaitGroup
	inicio.Add(1)
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inicio.Wait() // todas partem juntas, para maximizar a janela
			evento, err := enviarConcorrente(t, runtime, "org-conc-a", conectores["org-conc-a"], "wamid-corrida")
			switch {
			case err != nil:
				falhas.Add(1)
			case evento.Deduplicated:
				duplicadas.Add(1)
			default:
				aceitas.Add(1)
			}
		}()
	}
	inicio.Done()
	wg.Wait()

	if falhas.Load() != 0 {
		t.Fatalf("nenhuma entrega podia falhar: %d falhas", falhas.Load())
	}
	if aceitas.Load() != 1 {
		t.Fatalf("a MESMA entrega concorrente precisa ser aceita UMA vez: aceitas=%d duplicadas=%d",
			aceitas.Load(), duplicadas.Load())
	}
	if duplicadas.Load() != goroutines-1 {
		t.Fatalf("as demais precisam ser marcadas como duplicata: duplicadas=%d", duplicadas.Load())
	}
}

// Uma organização NUNCA enxerga a mensagem da outra, mesmo com as duas ingerindo ao mesmo tempo.
func TestOmnichannelConcurrentKeepsTenantsApart(t *testing.T) {
	runtime, conectores := runtimeConcorrente(t, 1_000_000)
	const porOrg = 20
	var wg sync.WaitGroup
	for _, org := range []string{"org-conc-a", "org-conc-b"} {
		for i := range porOrg {
			wg.Add(1)
			go func(org string, i int) {
				defer wg.Done()
				if _, err := enviarConcorrente(t, runtime, org, conectores[org], fmt.Sprintf("wamid-%s-%d", org, i)); err != nil {
					t.Errorf("org %s: %v", org, err)
				}
			}(org, i)
		}
	}
	wg.Wait()

	for _, org := range []string{"org-conc-a", "org-conc-b"} {
		eventos := runtime.Events(org)
		if len(eventos) == 0 {
			t.Fatalf("org %s precisa ter eventos", org)
		}
		for _, evento := range eventos {
			if evento.OrganizationID != org {
				t.Fatalf("VAZAMENTO entre tenants: org %s recebeu evento de %s", org, evento.OrganizationID)
			}
		}
	}
}

// O rate limit precisa CONTAR CERTO sob concorrência: aceitas + recusadas = total enviado.
func TestOmnichannelConcurrentRateLimitCountsCorrectly(t *testing.T) {
	const limite = 10
	runtime, conectores := runtimeConcorrente(t, limite)
	const goroutines, porGoroutine = 8, 5
	var aceitas, recusadas, inesperadas atomic.Int64
	var wg sync.WaitGroup
	inicio := make(chan struct{})
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			<-inicio
			for i := range porGoroutine {
				ident := fmt.Sprintf("wamid-rl-%d-%d", g, i)
				if _, err := enviarConcorrente(t, runtime, "org-conc-a", conectores["org-conc-a"], ident); err != nil {
					if strings.Contains(err.Error(), "rate limit") {
						recusadas.Add(1)
						continue
					}
					inesperadas.Add(1)
					continue
				}
				aceitas.Add(1)
			}
		}(g)
	}
	close(inicio)
	wg.Wait()

	total := int64(goroutines * porGoroutine)
	if inesperadas.Load() != 0 {
		t.Fatalf("só rate limit podia recusar: %d erros inesperados", inesperadas.Load())
	}
	if soma := aceitas.Load() + recusadas.Load(); soma != total {
		t.Fatalf("aceitas+recusadas precisa ser o total: %d+%d != %d", aceitas.Load(), recusadas.Load(), total)
	}
	if aceitas.Load() > limite {
		t.Fatalf("o rate limit de %d foi ultrapassado sob concorrência: %d aceitas", limite, aceitas.Load())
	}
}

// O cache de eventos respeita o teto também sob concorrência, e a contagem fica consistente.
func TestOmnichannelConcurrentEventsCacheRespectsTheCap(t *testing.T) {
	const teto = 25
	anterior := OmnichannelMaxEventsPerOrg
	OmnichannelMaxEventsPerOrg = teto
	t.Cleanup(func() { OmnichannelMaxEventsPerOrg = anterior })

	runtime, conectores := runtimeConcorrente(t, 1_000_000)
	const goroutines, porGoroutine = 6, 12
	var wg sync.WaitGroup
	inicio := make(chan struct{})
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			<-inicio
			for i := range porGoroutine {
				ident := fmt.Sprintf("wamid-cache-%d-%d", g, i)
				if _, err := enviarConcorrente(t, runtime, "org-conc-a", conectores["org-conc-a"], ident); err != nil {
					t.Errorf("envio %s: %v", ident, err)
				}
			}
		}(g)
	}
	close(inicio)
	wg.Wait()

	if eventos := runtime.Events("org-conc-a"); len(eventos) > teto {
		t.Fatalf("o teto de %d precisa valer sob concorrência: %d eventos", teto, len(eventos))
	}
}
