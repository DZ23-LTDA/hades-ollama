package agent

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MEDIÇÃO SOB CONCORRÊNCIA (E-176).
//
// Em E-175 a medição ficou declarada como **isolada**: "não sob carga concorrente nem contenção do mutex". Aqui a
// ingestão é medida com VÁRIAS goroutines, e a diferença para o benchmark isolado é o preço da contenção.

// BenchmarkOmnichannelIngestParalelo mede a ingestão aceita com goroutines em paralelo (b.RunParallel).
func BenchmarkOmnichannelIngestParalelo(b *testing.B) {
	anterior := OmnichannelMaxEventsPerOrg
	OmnichannelMaxEventsPerOrg = 1 << 30 // sem poda: aqui o objeto é a CONTENÇÃO (a poda foi medida em E-175)
	b.Cleanup(func() { OmnichannelMaxEventsPerOrg = anterior })

	const secretEnv = "HADES_BENCH_CONC_SECRET"
	b.Setenv(secretEnv, "segredo-de-benchmark-com-tamanho-suficiente")

	runtime := NewOmnichannelRuntime(nil)
	runtime.SetRateLimit(1 << 30)
	agora := time.Now().UTC()
	conector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-bench-conc", Channel: "whatsapp", Name: "Conc", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		b.Fatal(err)
	}

	// Cada goroutine usa identificadores próprios: mede-se a contenção do runtime, não a dedupe.
	var contador atomic.Int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ident := "wamid-conc-" + strconv.Itoa(int(contador.Add(1)))
			corpo := []byte(`{"external_message_id":"` + ident + `"}`)
			assinatura, err := runtime.SignWebhook(conector.ID, agora.Unix(), corpo)
			if err != nil {
				b.Fatal(err)
			}
			evento := OmnichannelEvent{ExternalMessageID: ident, ContactID: "+5511999998888", MessageType: "text"}
			if _, err := runtime.Ingest("org-bench-conc", "whatsapp", assinatura, agora.Unix(), corpo, evento, agora); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestOmnichannelInFlightReservationIsReleased trava a LIBERAÇÃO da reserva (E-176): se a marca de "em voo" não
// saísse ao final, uma entrega legítima seria classificada como duplicata para sempre — o mesmo ID nunca mais
// entraria.
func TestOmnichannelInFlightReservationIsReleased(t *testing.T) {
	runtime, conectores := runtimeConcorrente(t, 1_000_000)
	// Várias entregas DISTINTAS em paralelo: todas precisam ser aceitas e nenhuma pode ficar "presa".
	const goroutines, porGoroutine = 4, 5
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := range porGoroutine {
				ident := fmt.Sprintf("wamid-voo-%d-%d", g, i)
				evento, err := enviarConcorrente(t, runtime, "org-conc-a", conectores["org-conc-a"], ident)
				if err != nil {
					t.Errorf("envio %s: %v", ident, err)
					continue
				}
				if evento.Deduplicated {
					t.Errorf("entrega NOVA marcada como duplicata: %s", ident)
				}
			}
		}(g)
	}
	wg.Wait()

	runtime.mu.Lock()
	presas := len(runtime.eventosEmVoo)
	runtime.mu.Unlock()
	if presas != 0 {
		t.Fatalf("a reserva precisa ser LIBERADA ao final: %d chaves presas", presas)
	}
	// O MESMO identificador: a primeira entra, a segunda é duplicata.
	if evento, err := enviarConcorrente(t, runtime, "org-conc-a", conectores["org-conc-a"], "wamid-voo-repetido"); err != nil {
		t.Fatal(err)
	} else if evento.Deduplicated {
		t.Fatal("a primeira entrega deste id precisa ser aceita")
	}
	if evento, err := enviarConcorrente(t, runtime, "org-conc-a", conectores["org-conc-a"], "wamid-voo-repetido"); err != nil {
		t.Fatal(err)
	} else if !evento.Deduplicated {
		t.Fatal("a segunda entrega do MESMO id precisa ser duplicata")
	}
}
