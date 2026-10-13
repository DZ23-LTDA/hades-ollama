package agent

import (
	"fmt"
	"testing"
	"time"
)

// Estágio 34 — ORDENAÇÃO POR CONVERSA (E-298): MARCAR, NUNCA DESCARTAR NEM REORDENAR EM SILÊNCIO.
//
// O runtime omnichannel não tinha NENHUM conceito de ordem. A ordem de chegada não é a ordem do provedor:
// reentregas, atrasos de rede e canais paralelos reordenam. Reordenar de verdade exigiria buffer com espera e
// descartar perderia dado; o honesto é MANTER o último instante aceito, MARCAR quem chega atrás e CONTAR.
//
// Propriedades travadas (ponta a ponta, com HMAC e conector de verdade):
//  1. em ordem ⇒ não marcado;
//  2. depois de um instante maior, um instante MENOR ⇒ marcado como fora de ordem E AINDA ENTREGUE (não descartado);
//  3. instante IGUAL ⇒ não marcado (empate de relógio não pode virar alarme falso);
//  4. conversa diferente ⇒ linha do tempo própria (não contamina);
//  5. organização diferente ⇒ linha do tempo própria (isolamento entre tenants);
//  6. o contador mostra o fenômeno em vez de escondê-lo.
func TestRuntimeOmnichannelMarcaEventoForaDeOrdemSemDescartar(t *testing.T) {
	runtime, _, _ := omnichannelRuntime(t)
	const segredo = "segredo-de-teste-com-tamanho-suficiente"
	agora := time.Now().UTC()

	enviar := func(org, canal, conversa, id string, instante time.Time) OmnichannelEvent {
		t.Helper()
		corpo := []byte(fmt.Sprintf(`{"external_message_id":%q}`, id))
		assinatura := omnichannelSignature(segredo, instante.Unix(), corpo)
		evento := omnichannelEvent(t)
		evento.ExternalMessageID = id
		evento.ConversationID = conversa
		aceito, err := runtime.Ingest(org, canal, assinatura, instante.Unix(), corpo, evento, agora)
		if err != nil {
			t.Fatalf("ingest %s: %v", id, err)
		}
		return aceito
	}

	// 1. Em ordem: não marcado.
	if e := enviar("org-a", "whatsapp", "conv-1", "m1", agora.Add(-40*time.Second)); e.OutOfOrder {
		t.Fatalf("primeiro evento não pode ser fora de ordem: %+v", e)
	}
	if e := enviar("org-a", "whatsapp", "conv-1", "m2", agora.Add(-30*time.Second)); e.OutOfOrder {
		t.Fatalf("evento em ordem não pode ser marcado: %+v", e)
	}

	// 2. Instante MENOR depois de um maior: marcado E AINDA ENTREGUE.
	atrasado := enviar("org-a", "whatsapp", "conv-1", "m3", agora.Add(-35*time.Second))
	if !atrasado.OutOfOrder {
		t.Fatalf("evento que chegou atrás precisa ser MARCADO: %+v", atrasado)
	}
	if atrasado.ExternalMessageID != "m3" || atrasado.ID == "" {
		t.Fatalf("o evento fora de ordem NÃO pode ser descartado nem perder identidade: %+v", atrasado)
	}
	if atrasado.Deduplicated {
		t.Fatalf("fora de ordem não é duplicata: %+v", atrasado)
	}

	// 3. Instante IGUAL não é fora de ordem (empate não vira alarme falso).
	if e := enviar("org-a", "whatsapp", "conv-1", "m4", agora.Add(-30*time.Second)); e.OutOfOrder {
		t.Fatalf("instante igual não pode ser marcado como fora de ordem: %+v", e)
	}

	// 4. Conversa diferente tem linha do tempo própria: um instante antigo NELA não é atraso.
	if e := enviar("org-a", "whatsapp", "conv-2", "m5", agora.Add(-90*time.Second)); e.OutOfOrder {
		t.Fatalf("outra conversa não pode herdar a linha do tempo da primeira: %+v", e)
	}

	// 5. Organização diferente: mesma conversa, outra linha do tempo (isolamento entre tenants). O tenant precisa
	// do PRÓPRIO conector — o runtime não inventa organização.
	if _, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-b", Channel: "whatsapp", Name: "WhatsApp da B",
		SecretEnv: "HADES_TEST_OMNI_SECRET", Enabled: true,
	}, agora); err != nil {
		t.Fatal(err)
	}
	if e := enviar("org-b", "whatsapp", "conv-1", "m6", agora.Add(-90*time.Second)); e.OutOfOrder {
		t.Fatalf("outra organização não pode ser afetada pela primeira: %+v", e)
	}

	// 6. O contador mostra o fenômeno.
	if contagem := runtime.OutOfOrderCount(); contagem != 1 {
		t.Fatalf("contador de fora de ordem = %d, quer 1", contagem)
	}
}
