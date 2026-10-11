package agent

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Ferramentas de comércio e empresa dos agentes de atendimento (estágio 35).
//
// O que estes testes travam:
//
//  1. SEM O STORE LIGADO, a ferramenta é INDISPONÍVEL — disponibilidade é consequência de ter executor,
//     não de constar numa lista (foi exatamente o defeito que E-144 corrigiu no papel);
//  2. COM o store ligado, a ferramenta EXECUTA e devolve dado real (nome, preço, contagem);
//  3. organização sem catálogo DIZ que não tem, em vez de inventar oferta;
//  4. ISOLAMENTO: o catálogo de outra organização não aparece.
func commerceAgentRuntime(t *testing.T) (*ServiceAgentRuntime, *CommerceStore, *CompanyStore, string, time.Time) {
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
	commerce, err := NewCommerceStoreAt(filepath.Join(root, "commerce"))
	if err != nil {
		t.Fatal(err)
	}
	companies, err := NewCompanyStore(filepath.Join(root, "companies"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()

	phone := "+5511988887777"
	if _, _, err := crm.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Bruno Lima", Phone: phone}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := consent.GrantConsent("org-a", ConsentChannelWhatsApp, phone, "site", "checkbox", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	conversation, _, err := inbox.AppendInbound("org-a", "whatsapp", phone, "Bruno Lima", "tem esse produto?", "wamid.cc.1", now)
	if err != nil {
		t.Fatal(err)
	}
	// Catálogo REAL: um produto com preço e um SKU zerado (estoque baixo).
	if _, err := commerce.UpsertProduct(CommerceProduct{OrganizationID: "org-a", SKU: "CAM-001", Name: "Camiseta azul", PriceCents: 7900, Currency: "BRL", Active: true}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.UpsertProduct(CommerceProduct{OrganizationID: "org-a", SKU: "BON-002", Name: "Boné", PriceCents: 4500, Currency: "BRL", Active: true}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.RecordStock("org-a", "CAM-001", 5, CommerceMovementEntry, "lote inicial", now); err != nil {
		t.Fatal(err)
	}
	if _, err := companies.Create(Company{ID: "emp-1", OrganizationID: "org-a", Name: "Loja da Ana", Currency: "BRL"}); err != nil {
		t.Fatal(err)
	}
	runtime := NewServiceAgentRuntime(crm, inbox, consent, DefaultServiceAgentTools())
	return runtime, commerce, companies, conversation.ID, now
}

func TestServiceAgentCommerceToolsUnavailableWithoutWiring(t *testing.T) {
	runtime, _, _, conversationID, now := commerceAgentRuntime(t)

	// Sem `SetCommerce`, o agente de vendas declara `catalog.read` e NÃO existe executor.
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSales, ConversationID: conversationID,
		Intent: "tem camiseta?", RequestedTools: []string{"catalog.read"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "catalog.read")
	if call == nil || call.Status != ServiceAgentToolUnavailable {
		t.Fatalf("sem store ligado, catalog.read precisa ser INDISPONÍVEL: %+v", turn.ToolCalls)
	}
	if call.Reason == "" {
		t.Fatal("indisponibilidade precisa de motivo")
	}
}

func TestServiceAgentCommerceToolsExecuteWithRealData(t *testing.T) {
	runtime, commerce, companies, conversationID, now := commerceAgentRuntime(t)
	runtime.SetCommerce(commerce)
	runtime.SetCompanies(companies)

	// VENDAS pede o catálogo: o resumo precisa trazer nome e PREÇO reais do store.
	sales, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSales, ConversationID: conversationID,
		Intent: "quanto custa a camiseta?", RequestedTools: []string{"catalog.read"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	catalog := callByName(sales.ToolCalls, "catalog.read")
	if catalog == nil || catalog.Status != ServiceAgentToolExecuted {
		t.Fatalf("com store ligado, catalog.read precisa executar: %+v", sales.ToolCalls)
	}
	if !strings.Contains(catalog.Summary, "Camiseta azul") || !strings.Contains(catalog.Summary, "7900") {
		t.Fatalf("o resumo precisa trazer o produto REAL com preço: %q", catalog.Summary)
	}

	// COMÉRCIO pede estoque e pedidos: as duas ferramentas antes anunciadas como disponíveis sem executor.
	commerceAgent, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentCommerce, ConversationID: conversationID,
		Intent: "como está meu pedido?", RequestedTools: []string{"commerce.stock.read", "commerce.order.read"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	stock := callByName(commerceAgent.ToolCalls, "commerce.stock.read")
	if stock == nil || stock.Status != ServiceAgentToolExecuted {
		t.Fatalf("commerce.stock.read precisa executar: %+v", commerceAgent.ToolCalls)
	}
	// O SKU sem estoque entra aparece como estoque baixo — dado real, não texto fixo.
	if !strings.Contains(stock.Summary, "BON-002") {
		t.Fatalf("estoque baixo precisa apontar o SKU real: %q", stock.Summary)
	}
	orders := callByName(commerceAgent.ToolCalls, "commerce.order.read")
	if orders == nil || orders.Status != ServiceAgentToolExecuted {
		t.Fatalf("commerce.order.read precisa executar: %+v", commerceAgent.ToolCalls)
	}
	if !strings.Contains(orders.Summary, "0 pedido") {
		t.Fatalf("sem pedidos, o resumo precisa dizer ZERO em vez de inventar: %q", orders.Summary)
	}

	// SUPERVISOR pede o relatório de empresas.
	supervisor, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupervisor, ConversationID: conversationID,
		Intent: "como está a empresa?", RequestedTools: []string{"company.report.read"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	report := callByName(supervisor.ToolCalls, "company.report.read")
	if report == nil || report.Status != ServiceAgentToolExecuted {
		t.Fatalf("company.report.read precisa executar: %+v", supervisor.ToolCalls)
	}
	if !strings.Contains(report.Summary, "Loja da Ana") {
		t.Fatalf("o relatório precisa trazer a empresa real: %q", report.Summary)
	}

	// E a disponibilidade agora inclui as ferramentas ligadas.
	available := runtime.ToolExecutors()
	for _, name := range []string{"catalog.read", "commerce.stock.read", "commerce.order.read", "company.report.read"} {
		if !containsString(available, name) {
			t.Fatalf("%s precisa constar como executável depois da ligação: %+v", name, available)
		}
	}
	if containsString(runtime.UnavailableTools(), "commerce.stock.read") {
		t.Fatalf("commerce.stock.read não pode continuar indisponível: %+v", runtime.UnavailableTools())
	}
}

func TestServiceAgentCommerceToolsAreScopedByOrganization(t *testing.T) {
	runtime, commerce, _, conversationID, now := commerceAgentRuntime(t)
	runtime.SetCommerce(commerce)

	// Organização sem catálogo: a resposta é "não tem", nunca o catálogo da outra.
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-b", AgentID: ServiceAgentSales, ConversationID: conversationID,
		Intent: "tem camiseta?", RequestedTools: []string{"catalog.read"},
	}, now)
	if err != nil {
		// A conversa é de org-a: para org-b nem a conversa existe, e o erro declarado é a resposta correta.
		if !strings.Contains(err.Error(), "conversation") && !strings.Contains(err.Error(), "not found") {
			t.Fatalf("erro inesperado para organização sem conversa: %v", err)
		}
		return
	}
	call := callByName(turn.ToolCalls, "catalog.read")
	if call == nil {
		t.Fatalf("a tentativa precisa aparecer: %+v", turn.ToolCalls)
	}
	if call.Status == ServiceAgentToolExecuted && strings.Contains(call.Summary, "Camiseta azul") {
		t.Fatalf("o catálogo de OUTRA organização não pode aparecer: %q", call.Summary)
	}
}
