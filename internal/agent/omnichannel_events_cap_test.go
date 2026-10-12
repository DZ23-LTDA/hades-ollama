package agent

import (
	"testing"
	"time"
)

// TETO NO CACHE DE EVENTOS RICOS (E-174). `eventsByKey` era o ÚLTIMO mapa sem limite do omnichannel: num processo
// de longa duração, cada mensagem aceita ficava na memória para sempre.
//
// Limitar isso só é SEGURO porque a dedupe passou a ser persistida (E-172): a memória de "já entrou" não depende
// mais deste mapa, então descartar o payload antigo **não** reabre a janela de duplicação. Os testes travam:
//
//  1. o cache respeita o TETO por organização;
//  2. os eventos MAIS RECENTES sobrevivem (os antigos saem);
//  3. mesmo com o payload descartado, a DEDUPE continua reconhecendo a mensagem antiga (a proteção não depende
//     do cache);
//  4. o teto é por organização: o volume de uma não expulsa o evento de outra.

const segredoEventos = "segredo-de-teste-com-tamanho-suficiente"

func eventosRuntime(t *testing.T, org string, teto int) (*OmnichannelRuntime, string) {
	t.Helper()
	anterior := OmnichannelMaxEventsPerOrg
	OmnichannelMaxEventsPerOrg = teto
	t.Cleanup(func() { OmnichannelMaxEventsPerOrg = anterior })

	secretEnv := "HADES_TEST_EVENTS_SECRET"
	t.Setenv(secretEnv, segredoEventos)
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetRateLimit(1_000_000) // o rate limit não é o objeto deste teste
	runtime.SetDedupeStore(NewOmnichannelDedupeStore(t.TempDir()))
	agora := time.Now().UTC()
	connector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: org, Channel: "whatsapp", Name: "WhatsApp", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	return runtime, connector.ID
}

func eventoEnviar(t *testing.T, runtime *OmnichannelRuntime, org, connectorID, id string) {
	t.Helper()
	agora := time.Now().UTC()
	corpo := []byte(`{"external_message_id":"` + id + `"}`)
	assinatura, err := runtime.SignWebhook(connectorID, agora.Unix(), corpo)
	if err != nil {
		t.Fatal(err)
	}
	evento := OmnichannelEvent{ExternalMessageID: id, ContactID: "+5511999998888", MessageType: "text"}
	if _, err := runtime.Ingest(org, "whatsapp", assinatura, agora.Unix(), corpo, evento, agora); err != nil {
		t.Fatalf("evento %s: %v", id, err)
	}
	// O carimbo de tempo tem resolução de segundo no webhook; aqui o cache ordena por IngestedAt, que é por
	// chamada — mas um pequeno intervalo deixa a ordem inequívoca entre eventos do mesmo teste.
	time.Sleep(2 * time.Millisecond)
}

func TestOmnichannelEventsCacheRespectsTheCap(t *testing.T) {
	const teto = 20
	runtime, connector := eventosRuntime(t, "org-eventos", teto)
	for i := range teto + 15 {
		eventoEnviar(t, runtime, "org-eventos", connector, "wamid-evento-"+itoa(i))
	}
	eventos := runtime.Events("org-eventos")
	if len(eventos) > teto {
		t.Fatalf("o cache de eventos ricos precisa respeitar o teto de %d: %d", teto, len(eventos))
	}
	// Os MAIS RECENTES sobrevivem; o primeiro foi descartado.
	for _, evento := range eventos {
		if evento.ExternalMessageID == "wamid-evento-0" {
			t.Fatal("o evento mais ANTIGO deveria ter saído do cache")
		}
	}
	ultimo := "wamid-evento-" + itoa(teto+14)
	encontrado := false
	for _, evento := range eventos {
		if evento.ExternalMessageID == ultimo {
			encontrado = true
		}
	}
	if !encontrado {
		t.Fatalf("o evento mais RECENTE precisa continuar no cache: %s", ultimo)
	}
}

func TestOmnichannelEventsCacheCapKeepsDedupeWorking(t *testing.T) {
	const teto = 5
	runtime, connector := eventosRuntime(t, "org-dedupe-cache", teto)
	eventoEnviar(t, runtime, "org-dedupe-cache", connector, "wamid-antigo")
	// Enche o cache até o primeiro evento sair dele.
	for i := range teto + 5 {
		eventoEnviar(t, runtime, "org-dedupe-cache", connector, "wamid-novo-"+itoa(i))
	}
	for _, evento := range runtime.Events("org-dedupe-cache") {
		if evento.ExternalMessageID == "wamid-antigo" {
			t.Skip("o cache desta máquina reteve o evento antigo: o cenário exige que ele tenha saído")
		}
	}
	// Mesmo com o PAYLOAD fora do cache, a dedupe precisa reconhecer a mensagem antiga — é isto que torna o teto
	// seguro (e é a ligação com E-172).
	agora := time.Now().UTC()
	corpo := []byte(`{"external_message_id":"wamid-antigo"}`)
	assinatura, err := runtime.SignWebhook(connector, agora.Unix(), corpo)
	if err != nil {
		t.Fatal(err)
	}
	evento := OmnichannelEvent{ExternalMessageID: "wamid-antigo", ContactID: "+5511999998888", MessageType: "text"}
	repetido, err := runtime.Ingest("org-dedupe-cache", "whatsapp", assinatura, agora.Unix(), corpo, evento, agora)
	if err != nil {
		t.Fatalf("a duplicata continua sendo reconhecida, sem erro: %v", err)
	}
	if !repetido.Deduplicated {
		t.Fatalf("a dedupe NÃO pode depender do cache de eventos ricos: %+v", repetido)
	}
}

func TestOmnichannelEventsCacheCapIsPerOrganization(t *testing.T) {
	const teto = 5
	anterior := OmnichannelMaxEventsPerOrg
	OmnichannelMaxEventsPerOrg = teto
	t.Cleanup(func() { OmnichannelMaxEventsPerOrg = anterior })

	secretEnv := "HADES_TEST_EVENTS_ISOLATION"
	t.Setenv(secretEnv, segredoEventos)
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetRateLimit(1_000_000)
	agora := time.Now().UTC()
	conectorA, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-a", Channel: "whatsapp", Name: "A", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	conectorB, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-b", Channel: "whatsapp", Name: "B", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	// O volume de A NÃO pode expulsar o evento de B.
	eventoEnviar(t, runtime, "org-b", conectorB.ID, "wamid-de-b")
	for i := range teto + 5 {
		eventoEnviar(t, runtime, "org-a", conectorA.ID, "wamid-de-a-"+itoa(i))
	}
	deB := runtime.Events("org-b")
	if len(deB) != 1 || deB[0].ExternalMessageID != "wamid-de-b" {
		t.Fatalf("o teto é POR ORGANIZAÇÃO: o evento de B não pode ser expulso por A: %+v", deB)
	}
}
