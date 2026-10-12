package agent

import (
	"strconv"
	"testing"
	"time"
)

// CUSTO DA PODA DO CACHE DE EVENTOS (E-175).
//
// Em E-174 o teto do cache de eventos ricos ficou **declarado como não medido**: "a poda percorre os eventos da
// organização a cada mensagem aceita e a latência dessa varredura não foi medida sob carga". Este benchmark
// fecha essa lacuna medindo o **caminho real de ingestão** em dois estados:
//
//   - **cache abaixo do teto**: a poda sai cedo (só conta), então mede o custo normal da ingestão;
//   - **cache no teto**: cada mensagem aceita descarta a mais antiga, então a DIFERENÇA entre os dois mede o
//     custo da poda.
//
// O store de dedupe fica NIL de propósito: o objetivo é isolar a poda do cache, não medir I/O de disco — isso
// mantém o número interpretável (um benchmark que mede duas coisas não explica nenhuma).

const tetoDoBenchmark = 2000

func benchmarkIngestao(b *testing.B, cacheCheio bool) {
	anterior := OmnichannelMaxEventsPerOrg
	OmnichannelMaxEventsPerOrg = tetoDoBenchmark
	b.Cleanup(func() { OmnichannelMaxEventsPerOrg = anterior })

	const secretEnv = "HADES_BENCH_EVENTS_SECRET"
	b.Setenv(secretEnv, "segredo-de-benchmark-com-tamanho-suficiente")

	runtime := NewOmnichannelRuntime(nil)
	runtime.SetRateLimit(1 << 30) // o rate limit não é o objeto deste benchmark
	agora := time.Now().UTC()
	connector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-bench-eventos", Channel: "whatsapp", Name: "Bench", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		b.Fatal(err)
	}

	enviar := func(ident string) {
		corpo := []byte(`{"external_message_id":"` + ident + `"}`)
		assinatura, err := runtime.SignWebhook(connector.ID, agora.Unix(), corpo)
		if err != nil {
			b.Fatal(err)
		}
		evento := OmnichannelEvent{ExternalMessageID: ident, ContactID: "+5511999998888", MessageType: "text"}
		if _, err := runtime.Ingest("org-bench-eventos", "whatsapp", assinatura, agora.Unix(), corpo, evento, agora); err != nil {
			b.Fatal(err)
		}
	}

	if cacheCheio {
		// Enche o cache ATÉ o teto: a partir daqui, toda mensagem aceita dispara a poda de verdade.
		for i := range tetoDoBenchmark {
			enviar("wamid-cheio-" + strconv.Itoa(i))
		}
		if total := len(runtime.Events("org-bench-eventos")); total != tetoDoBenchmark {
			b.Fatalf("o cache precisa estar exatamente no teto: %d", total)
		}
	}

	b.ResetTimer()
	for i := range b.N {
		enviar("wamid-medido-" + strconv.Itoa(i))
	}
}

// BenchmarkOmnichannelIngestCacheAbaixoDoTeto mede a ingestão com o cache folgado (a poda sai cedo).
func BenchmarkOmnichannelIngestCacheAbaixoDoTeto(b *testing.B) {
	benchmarkIngestao(b, false)
}

// BenchmarkOmnichannelIngestCacheNoTeto mede a ingestão com o cache CHEIO: a diferença para o benchmark acima é o
// custo da poda por mensagem aceita.
func BenchmarkOmnichannelIngestCacheNoTeto(b *testing.B) {
	benchmarkIngestao(b, true)
}

// BenchmarkOmnichannelIngestMapaGrandeDeOutraOrg é o CONTROLE que separa as duas explicações possíveis: aqui o
// mapa tem o MESMO tamanho do caso "no teto" (2000 eventos), mas de OUTRA organização — então a poda desta
// organização nunca roda. Se o número se parecer com o caso "no teto", o custo é o TAMANHO DO MAPA VIVO e não a
// poda; se se parecer com o caso folgado, a causa está na poda. É a medição que decide, não a intuição.
func BenchmarkOmnichannelIngestMapaGrandeDeOutraOrg(b *testing.B) {
	anterior := OmnichannelMaxEventsPerOrg
	OmnichannelMaxEventsPerOrg = tetoDoBenchmark
	b.Cleanup(func() { OmnichannelMaxEventsPerOrg = anterior })

	const secretEnv = "HADES_BENCH_EVENTS_SECRET"
	b.Setenv(secretEnv, "segredo-de-benchmark-com-tamanho-suficiente")

	runtime := NewOmnichannelRuntime(nil)
	runtime.SetRateLimit(1 << 30)
	agora := time.Now().UTC()
	outra, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-bench-outra", Channel: "telegram", Name: "Outra", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		b.Fatal(err)
	}
	medida, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-bench-eventos", Channel: "whatsapp", Name: "Bench", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		b.Fatal(err)
	}

	enviar := func(org, connectorID, ident string) {
		corpo := []byte(`{"external_message_id":"` + ident + `"}`)
		assinatura, err := runtime.SignWebhook(connectorID, agora.Unix(), corpo)
		if err != nil {
			b.Fatal(err)
		}
		evento := OmnichannelEvent{ExternalMessageID: ident, ContactID: "+5511999998888", MessageType: "text"}
		if _, err := runtime.Ingest(org, "whatsapp", assinatura, agora.Unix(), corpo, evento, agora); err != nil {
			b.Fatal(err)
		}
	}

	// Enche o mapa com 2000 eventos de OUTRA organização (canal telegram, para não colidir).
	for i := range tetoDoBenchmark {
		ident := "wamid-outra-" + strconv.Itoa(i)
		corpo := []byte(`{"external_message_id":"` + ident + `"}`)
		assinatura, err := runtime.SignWebhook(outra.ID, agora.Unix(), corpo)
		if err != nil {
			b.Fatal(err)
		}
		evento := OmnichannelEvent{ExternalMessageID: ident, ContactID: "+5511999998888", MessageType: "text"}
		if _, err := runtime.Ingest("org-bench-outra", "telegram", assinatura, agora.Unix(), corpo, evento, agora); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := range b.N {
		enviar("org-bench-eventos", medida.ID, "wamid-controle-"+strconv.Itoa(i))
	}
}
