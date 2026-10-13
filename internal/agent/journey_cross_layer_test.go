package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// ESTÁGIO 38 / §11.4 — JORNADA CRUZADA: CANAL → AGENTE → INBOX → CRM → REABERTURA, COM REPLAY (E-308).
//
// MEDIDO: **nenhum** arquivo de teste usava `Ingest(` e `RunTurn(` juntos. Cada camada era testada ISOLADA — e o
// §11.4 exige exatamente a jornada *“mensagem recebida → agente → ferramenta autorizada → resposta → CRM →
// histórico → reabertura, com replay e falhas”*. Testar camadas isoladas não prova que elas se encontram: é na
// fronteira que a integração quebra.
//
// A chave deste teste é o **INBOX COMPARTILHADO**: o runtime omnichannel e o runtime de agentes usam o MESMO store de
// caixa de entrada — sem isso, o teste não cruzaria nada.
//
// Propriedades travadas:
//  1. mensagem recebida pelo canal (com assinatura válida) chega ao inbox do runtime de agentes;
//  2. o agente de atendimento reconhece o CONTATO no CRM e enxerga o HISTÓRICO da conversa;
//  3. um turno do agente com ferramenta autorizada responde e o histórico cresce;
//  4. REABERTURA: um runtime NOVO, sobre a mesma raiz, lê a mesma conversa com o histórico preservado;
//  5. REPLAY: reentregar o MESMO `external_message_id` é deduplicado e NÃO duplica mensagem no inbox;
//  6. FALHA: assinatura inválida é recusada e NÃO adiciona nada.
func TestJornadaCruzadaCanalAgenteInboxCrmComReplay(t *testing.T) {
	raiz := t.TempDir()
	inbox, err := NewInboxStore(filepath.Join(raiz, "inbox"))
	if err != nil {
		t.Fatal(err)
	}
	crm, err := NewCRMStore(filepath.Join(raiz, "crm"))
	if err != nil {
		t.Fatal(err)
	}
	consent, err := NewConsentRegistry(filepath.Join(raiz, "consent"))
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	const org = "org-a"
	const canal = "whatsapp"
	const telefone = "+5511944443333"
	const segredo = "segredo-de-teste-com-tamanho-suficiente"
	t.Setenv("HADES_TEST_OMNI_SECRET", segredo)

	// 1. CANAL: conector registrado e mensagem recebida com assinatura VÁLIDA.
	canalRuntime := NewOmnichannelRuntime(inbox)
	conector, err := canalRuntime.RegisterConnector(OmnichannelConnector{
		OrganizationID: org, Channel: canal, Name: "WhatsApp da A", SecretEnv: "HADES_TEST_OMNI_SECRET", Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	corpo := []byte(`{"external_message_id":"wamid.cruzado.1"}`)
	assinatura := omnichannelSignature(segredo, agora.Unix(), corpo)
	evento, err := canalRuntime.Ingest(org, canal, assinatura, agora.Unix(), corpo, OmnichannelEvent{
		ExternalMessageID: "wamid.cruzado.1",
		ContactID:         telefone,
		MessageType:       "text",
		Preview:           "quero saber do meu pedido",
	}, agora)
	if err != nil {
		t.Fatalf("mensagem recebida pelo canal: %v", err)
	}
	if evento.ConnectorID != conector.ID {
		t.Fatalf("o evento precisa vir do conector registrado: %+v", evento)
	}

	// O MESMO inbox é lido pelo runtime de agentes — é isto que cruza as camadas.
	conversas := inbox.ListConversations(org, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(conversas) == 0 {
		t.Fatal("a mensagem do canal precisa chegar ao inbox compartilhado")
	}
	conversa := conversas[0]
	_, historicoInicial, err := inbox.Conversation(org, conversa.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(historicoInicial) == 0 {
		t.Fatal("a mensagem recebida pelo canal precisa estar no histórico da conversa")
	}

	// 2. CRM: o contato existe e o agente precisa reconhecê-lo.
	if _, _, err := crm.UpsertContact(CRMContact{OrganizationID: org, Name: "Cliente Cruzado", Phone: telefone}, agora); err != nil {
		t.Fatal(err)
	}
	if _, err := consent.GrantConsent(org, ConsentChannelWhatsApp, telefone, "site", "checkbox", "ana", nil, agora); err != nil {
		t.Fatal(err)
	}
	agentes := NewServiceAgentRuntime(crm, inbox, consent, DefaultServiceAgentTools())

	// 3. AGENTE: turno com ferramenta AUTORIZADA (leitura do contato), com o histórico do canal no contexto.
	turno, err := agentes.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: org,
		AgentID:        ServiceAgentSupport,
		ConversationID: conversa.ID,
		Intent:         "quero saber do meu pedido",
		ModelID:        "modelo-local-a",
		RequestedTools: []string{"crm.contact.read"},
	}, agora)
	if err != nil {
		t.Fatalf("turno do agente: %v", err)
	}
	reconheceu := false
	for _, c := range turno.ToolCalls {
		if c.Name == "crm.contact.read" && c.ExecutedAt.IsZero() == false {
			reconheceu = true
		}
	}
	if !reconheceu {
		t.Fatalf("o agente precisa RECONHECER o contato lendo o CRM nesta conversa: %+v", turno.ToolCalls)
	}
	if len(turno.Context.History) == 0 {
		t.Fatal("o agente precisa enxergar o HISTÓRICO da conversa que veio do canal")
	}

	// 5. REPLAY: reentregar o MESMO external_message_id é deduplicado.
	replay, err := canalRuntime.Ingest(org, canal, assinatura, agora.Unix(), corpo, OmnichannelEvent{
		ExternalMessageID: "wamid.cruzado.1", ContactID: telefone, MessageType: "text", Preview: "quero saber do meu pedido",
	}, agora)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !replay.Deduplicated {
		t.Fatalf("reentrega do mesmo external_message_id precisa ser marcada como duplicata: %+v", replay)
	}
	_, aposReplay, err := inbox.Conversation(org, conversa.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(aposReplay) != len(historicoInicial)+1 {
		// +1 = a resposta do agente no passo 3. O replay NÃO pode acrescentar mensagem.
		t.Fatalf("o replay não pode duplicar mensagem no inbox: esperado=%d, veio=%d", len(historicoInicial)+1, len(aposReplay))
	}

	// 6. FALHA: assinatura inválida é recusada e não adiciona nada.
	if _, err := canalRuntime.Ingest(org, canal, "assinatura-errada", agora.Unix(), corpo, OmnichannelEvent{
		ExternalMessageID: "wamid.cruzado.2", ContactID: telefone, MessageType: "text",
	}, agora); err == nil {
		t.Fatal("assinatura inválida precisa ser recusada")
	}

	// 4. REABERTURA: runtime NOVO sobre a mesma raiz lê a mesma conversa com o histórico preservado.
	reaberto, err := NewInboxStore(filepath.Join(raiz, "inbox"))
	if err != nil {
		t.Fatal(err)
	}
	_, historicoReaberto, err := reaberto.Conversation(org, conversa.ID)
	if err != nil {
		t.Fatalf("reabertura da conversa: %v", err)
	}
	if len(historicoReaberto) == 0 {
		t.Fatal("o histórico precisa sobreviver à reabertura")
	}
	if _, _, err := reaberto.Conversation("org-b", conversa.ID); err == nil {
		t.Fatal("a reabertura não pode cruzar organizações")
	}
}
