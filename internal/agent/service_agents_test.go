package agent

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Estágio 35 — agentes de atendimento e vendas.
//
// As propriedades travadas aqui:
//  1. os SETE agentes do estágio existem, com ferramentas e regras de escalada;
//  2. o turno usa CONTEXTO REAL: contato do CRM, oportunidades e HISTÓRICO da conversa
//     (continuidade humano/IA);
//  3. sem consentimento o agente **ESCALA** em vez de responder;
//  4. pedido de humano escala imediatamente;
//  5. a PROCEDÊNCIA do texto é declarada — nunca fingir que veio de modelo;
//  6. **trocar de modelo NÃO remove política** (consentimento e DLP continuam valendo);
//  7. ferramenta que o agente lista e o runtime não tem aparece como BLOQUEADA;
//  8. escopo por organização.
func serviceAgentRuntime(t *testing.T, grantConsent bool) (*ServiceAgentRuntime, *InboxStore, *CRMStore, *ConsentRegistry, string) {
	t.Helper()
	root := t.TempDir()
	inbox, err := NewInboxStore(filepath.Join(root, "inbox"))
	if err != nil {
		t.Fatal(err)
	}
	crm, err := NewCRMStore(filepath.Join(root, "crm"))
	if err != nil {
		t.Fatal(err)
	}
	consent, err := NewConsentRegistry(filepath.Join(root, "consent"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	phone := "+5511999998888"
	if _, _, err := crm.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Ana Souza", Phone: phone, Tags: []string{"cliente"}}, now); err != nil {
		t.Fatal(err)
	}
	contact := crm.ListContacts("org-a", "")[0]
	if _, err := crm.CreateOpportunity("org-a", contact.ID, "Plano avançado", 49900, "ana", now); err != nil {
		t.Fatal(err)
	}
	if grantConsent {
		if _, err := consent.GrantConsent("org-a", ConsentChannelWhatsApp, phone, "site", "checkbox", "ana", nil, now); err != nil {
			t.Fatal(err)
		}
	}
	conversation, _, err := inbox.AppendInbound("org-a", "whatsapp", phone, "Ana Souza", "quero saber do plano", "wamid.sa.1", now)
	if err != nil {
		t.Fatal(err)
	}
	// Uma mensagem anterior no histórico: a continuidade precisa aparecer no contexto.
	if _, _, err := inbox.AppendInbound("org-a", "whatsapp", phone, "Ana Souza", "bom dia", "wamid.sa.0", now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	runtime := NewServiceAgentRuntime(crm, inbox, consent, map[string]bool{
		"crm.contact.read": true, "crm.opportunity.write": true, "catalog.read": true,
		"inbox.reply": true, "inbox.handoff": true, "inbox.conversation.read": true,
	})
	// O store de APROVAÇÃO HUMANA entra no fixture porque escrever sem aprovação é RECUSADO — os testes de
	// escrita precisam do caminho real: pedir, decidir e só então executar.
	runtime.SetApprovalStore(NewServiceAgentApprovalStore(filepath.Join(root, "approvals")))
	return runtime, inbox, crm, consent, conversation.ID
}

// approvePendingForTool aprova o pedido PENDENTE da ferramenta indicada, como faria um humano.
//
// Devolve a aprovação decidida. Falha o teste se não houver pedido pendente — pedir antes de decidir é a
// ordem que o sistema impõe, e o teste não deve contorná-la.
func approvePendingForTool(t *testing.T, runtime *ServiceAgentRuntime, organizationID, toolName, actor string) ServiceAgentApproval {
	t.Helper()
	store := runtime.ApprovalStore()
	if store == nil {
		t.Fatal("o runtime precisa ter store de aprovações para este caminho")
	}
	pending, err := store.ListApprovals(organizationID, ServiceAgentApprovalPending, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	for _, approval := range pending {
		if approval.ToolName == toolName {
			decided, err := store.Decide(organizationID, approval.ID, true, actor, "aprovado no teste", time.Now().UTC())
			if err != nil {
				t.Fatal(err)
			}
			return decided
		}
	}
	t.Fatalf("não há pedido pendente para %s: %+v", toolName, pending)
	return ServiceAgentApproval{}
}

func TestServiceAgentCatalogCoversTheStageAndDeclaresTools(t *testing.T) {
	catalog := ServiceAgentCatalog()
	if len(catalog) != 7 {
		t.Fatalf("o estágio pede sete agentes: %d", len(catalog))
	}
	expected := []string{ServiceAgentCommerce, ServiceAgentReceptionist, ServiceAgentSales, ServiceAgentSocial, ServiceAgentSupervisor, ServiceAgentSupport, ServiceAgentVoice}
	for index, spec := range catalog {
		if spec.ID != expected[index] {
			t.Fatalf("catálogo fora de ordem/identificador: %+v", catalog)
		}
		if spec.Objective == "" || len(spec.Channels) == 0 || len(spec.Tools) == 0 {
			t.Fatalf("agente incompleto: %+v", spec)
		}
		if len(spec.EscalationRules) == 0 {
			t.Fatalf("agente sem regra de escalada: %+v", spec)
		}
	}
	// O agente de vendas existe e escreve oportunidade; o de voz passa a chamada para humano.
	sales, ok := ServiceAgentSpecByID(ServiceAgentSales)
	if !ok || !containsString(sales.Tools, "crm.opportunity.write") {
		t.Fatalf("vendas precisa escrever oportunidade: %+v", sales)
	}
	voice, ok := ServiceAgentSpecByID(ServiceAgentVoice)
	if !ok || !containsString(voice.Tools, "voice.transfer") {
		t.Fatalf("voz precisa transferir: %+v", voice)
	}
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func TestServiceAgentTurnUsesRealCRMContextAndHistory(t *testing.T) {
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	now := time.Now().UTC()
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSales, ConversationID: conversationID,
		Intent: "quero saber do plano", ModelID: "qwen2.5:7b",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if turn.Decision != "responder" || turn.Escalated {
		t.Fatalf("com consentimento o agente precisa responder: %+v", turn)
	}
	// Contexto REAL: contato do CRM, oportunidade aberta e histórico da conversa.
	if turn.Context.Contact == nil || turn.Context.Contact.Name != "Ana Souza" {
		t.Fatalf("o contato precisa vir do CRM: %+v", turn.Context.Contact)
	}
	if len(turn.Context.Opportunities) != 1 || turn.Context.Opportunities[0].Title != "Plano avançado" {
		t.Fatalf("a oportunidade precisa vir do CRM: %+v", turn.Context.Opportunities)
	}
	if len(turn.Context.History) < 2 {
		t.Fatalf("o histórico da conversa precisa estar no contexto: %d", len(turn.Context.History))
	}
	if turn.Context.ConsentState != ConsentGranted {
		t.Fatalf("o estado de consentimento precisa ser declarado: %q", turn.Context.ConsentState)
	}
	// PROCEDÊNCIA declarada: sem modelo conectado, o texto é template — e o turno diz.
	if turn.ReplySource != ReplySourceTemplate {
		t.Fatalf("a procedência do texto precisa ser declarada: %q", turn.ReplySource)
	}
	if !strings.Contains(turn.Reply, "Ana") {
		t.Fatalf("a resposta precisa usar o nome real do contato: %q", turn.Reply)
	}
	if turn.Message == nil || turn.Message.ConversationID != conversationID {
		t.Fatalf("a resposta precisa entrar na MESMA conversa: %+v", turn.Message)
	}
	// A mensagem está de fato no histórico da conversa.
	_, history, err := runtime.inbox.Conversation("org-a", conversationID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, message := range history {
		if message.ID == turn.Message.ID && message.Direction == InboxDirectionOutbound {
			found = true
		}
	}
	if !found {
		t.Fatal("a resposta precisa aparecer no histórico da conversa")
	}
}

func TestServiceAgentEscalatesWithoutConsentInsteadOfReplying(t *testing.T) {
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, false)
	now := time.Now().UTC()
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "meu pedido não chegou",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !turn.Escalated || turn.Decision != "escalar" {
		t.Fatalf("sem consentimento o agente precisa ESCALAR: %+v", turn)
	}
	if turn.Escalation != ServiceAgentEscalateConsent {
		t.Fatalf("o motivo da escalada precisa ser o consentimento: %+v", turn)
	}
	if turn.Message != nil || turn.Reply != "" {
		t.Fatalf("escalar não pode responder: %+v", turn)
	}
	// E NADA foi enviado para a conversa.
	_, history, err := runtime.inbox.Conversation("org-a", conversationID)
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range history {
		if message.Direction == InboxDirectionOutbound {
			t.Fatalf("nenhuma mensagem podia sair sem consentimento: %+v", message)
		}
	}
}

func TestServiceAgentEscalatesWhenHumanIsRequested(t *testing.T) {
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	now := time.Now().UTC()
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "quero falar com uma pessoa de verdade",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !turn.Escalated || turn.Escalation != ServiceAgentEscalateHuman {
		t.Fatalf("pedido de humano precisa escalar imediatamente: %+v", turn)
	}
}

func TestServiceAgentModelSwitchKeepsPolicies(t *testing.T) {
	runtime, _, _, consent, conversationID := serviceAgentRuntime(t, true)
	now := time.Now().UTC()
	// Mesmo turno, modelos diferentes: a política é a mesma nos dois.
	for _, model := range []string{"qwen2.5:7b", "llama3.2:3b", "gpt-externo-homologado"} {
		turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
			OrganizationID: "org-a", AgentID: ServiceAgentSales, ConversationID: conversationID,
			Intent: "quero o plano", ModelID: model,
		}, now)
		if err != nil {
			t.Fatal(err)
		}
		if turn.Context.ModelID != model {
			t.Fatalf("o modelo escolhido precisa aparecer no contexto: %+v", turn.Context)
		}
		if len(turn.Context.Policies) < 3 {
			t.Fatalf("as políticas precisam estar declaradas: %+v", turn.Context.Policies)
		}
		if turn.Context.ConsentState != ConsentGranted {
			t.Fatalf("consentimento precisa valer em qualquer modelo: %+v", turn.Context)
		}
	}
	// Revogando o opt-out, o agente escala com QUALQUER modelo — trocar de modelo não
	// é atalho para contornar consentimento.
	if _, err := consent.RevokeConsent("org-a", ConsentChannelWhatsApp, "+5511999998888", "pedido do titular", "ana", now); err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{"qwen2.5:7b", "gpt-externo-homologado"} {
		turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
			OrganizationID: "org-a", AgentID: ServiceAgentSales, ConversationID: conversationID,
			Intent: "quero o plano", ModelID: model,
		}, now)
		if err != nil {
			t.Fatal(err)
		}
		if !turn.Escalated || turn.Escalation != ServiceAgentEscalateConsent {
			t.Fatalf("após opt-out o agente precisa escalar em qualquer modelo (%s): %+v", model, turn)
		}
	}
}

func TestServiceAgentBlocksToolsItDoesNotHave(t *testing.T) {
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	now := time.Now().UTC()
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSocial, ConversationID: conversationID,
		Intent: "oi",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// O agente social lista inbox.handoff e social.draft, que o runtime NÃO registrou:
	// elas precisam aparecer como bloqueadas, não ser ignoradas em silêncio.
	if len(turn.Context.BlockedTools) == 0 {
		t.Fatalf("ferramenta não registrada precisa aparecer como bloqueada: %+v", turn.Context)
	}
	if !containsString(turn.Context.BlockedTools, "social.draft") {
		t.Fatalf("social.draft precisa estar bloqueada: %+v", turn.Context.BlockedTools)
	}
	for _, tool := range turn.Context.AllowedTools {
		if !runtime.availableTools()[tool] {
			t.Fatalf("ferramenta permitida precisa existir no runtime: %s", tool)
		}
	}
}

func TestServiceAgentIsScopedByOrganization(t *testing.T) {
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	now := time.Now().UTC()
	// A conversa é da org-a: pedir em nome da org-b não pode funcionar.
	if _, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-b", AgentID: ServiceAgentSales, ConversationID: conversationID,
		Intent: "quero o plano",
	}, now); err == nil {
		t.Fatal("conversa de outra organização não pode ser usada")
	}
	// Agente inexistente falha explicitamente.
	if _, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: "agente-fantasma", ConversationID: conversationID,
	}, now); err == nil {
		t.Fatal("agente inexistente precisa falhar")
	}
}
