package agent

import (
	"strings"
	"testing"
	"time"
)

// Execução REAL de ferramentas dos agentes de atendimento (estágio 35). O que estes testes travam:
//
//  1. ferramenta com executor DEVOLVE DADO DO STORE — não texto inventado;
//  2. pedir NÃO É PODER: ferramenta que o agente não declara é bloqueada, mesmo com executor existente;
//  3. declarada e SEM executor aparece como INDISPONÍVEL, com motivo (antes era anunciada como
//     "disponível" por um mapa digitado à mão);
//  4. ferramenta de SAÍDA exige consentimento concedido;
//  5. falha do executor é registrada como FALHA, com o erro real — nada de sucesso fingido;
//  6. o ESCOPO vem da sessão: `ToolArguments` não consegue apontar para outra organização.
func callByName(calls []ServiceAgentToolCall, name string) *ServiceAgentToolCall {
	for index := range calls {
		if calls[index].Name == name {
			return &calls[index]
		}
	}
	return nil
}

func TestServiceAgentToolExecutesAgainstRealStore(t *testing.T) {
	runtime, _, crm, _, conversationID := serviceAgentRuntime(t, true)
	now := time.Now().UTC()

	// `crm.opportunity.write` é declarada pelo agente de VENDAS — e é EFEITO: exige aprovação humana.
	pedido := ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSales, ConversationID: conversationID,
		Intent: "quero o plano avançado", ModelID: "modelo-local",
		RequestedTools: []string{"crm.contact.read", "crm.opportunity.write"},
		ToolArguments:  ServiceAgentToolRequest{Title: "Plano avançado — turno", ValueCents: 79900, OwnerID: "ana"},
	}
	oportunidadesAntes := len(crm.ListOpportunities("org-a", ""))

	turn, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	read := callByName(turn.ToolCalls, "crm.contact.read")
	if read == nil || read.Status != ServiceAgentToolExecuted {
		t.Fatalf("crm.contact.read precisa executar: %+v", turn.ToolCalls)
	}
	// O resumo traz dado REAL do CRM (o contato cadastrado no fixture).
	if !strings.Contains(read.Summary, "Ana Souza") {
		t.Fatalf("o resumo precisa vir do store, não de texto genérico: %q", read.Summary)
	}
	// LEITURA executa; ESCRITA aguarda decisão humana — e NADA foi gravado enquanto isso.
	write := callByName(turn.ToolCalls, "crm.opportunity.write")
	if write == nil || write.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("escrita sem aprovação precisa ficar PENDENTE: %+v", turn.ToolCalls)
	}
	if write.ApprovalID == "" {
		t.Fatal("o pedido de aprovação precisa ser identificável")
	}
	if write.Effect != ServiceAgentEffectWrite {
		t.Fatalf("o efeito da ferramenta precisa ser declarado: %q", write.Effect)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != oportunidadesAntes {
		t.Fatalf("nada pode ser escrito antes da aprovação: antes=%d depois=%d", oportunidadesAntes, depois)
	}

	// Decisão humana e repetição do MESMO turno: agora executa e o resumo declara o que foi criado.
	approvePendingForTool(t, runtime, "org-a", "crm.opportunity.write", "ana")
	turn, err = runtime.RunTurn(pedido, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	write = callByName(turn.ToolCalls, "crm.opportunity.write")
	if write == nil || write.Status != ServiceAgentToolExecuted {
		t.Fatalf("com aprovação, crm.opportunity.write precisa executar: %+v", turn.ToolCalls)
	}
	if !strings.Contains(write.Summary, "oportunidade") || !strings.Contains(write.Summary, "79900") {
		t.Fatalf("o resumo da escrita precisa declarar o que foi criado: %q", write.Summary)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != oportunidadesAntes+1 {
		t.Fatalf("a oportunidade aprovada precisa existir no CRM: antes=%d depois=%d", oportunidadesAntes, depois)
	}
}

func TestServiceAgentTurnBlockedToolNotDeclaredByAgent(t *testing.T) {
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	now := time.Now().UTC()

	// `inbox.assign` existe (tem executor) e é declarada pelo SUPERVISOR — mas o RECEPCIONISTA não a
	// declara. Pedir não pode virar poder.
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi", RequestedTools: []string{"inbox.assign"},
		ToolArguments: ServiceAgentToolRequest{Assignee: "equipe-vendas"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "inbox.assign")
	if call == nil {
		t.Fatalf("a tentativa precisa aparecer na trilha: %+v", turn.ToolCalls)
	}
	if call.Status != ServiceAgentToolBlocked {
		t.Fatalf("ferramenta não declarada pelo agente precisa ser BLOQUEADA: %+v", call)
	}
	if !strings.Contains(call.Reason, "não declara") {
		t.Fatalf("o motivo precisa dizer que o agente não declara: %q", call.Reason)
	}
	// E o supervisor, que DECLARA a mesma ferramenta, executa — depois da aprovação humana, porque
	// atribuição de conversa é EFEITO.
	supervisor, _, _, _, supervisorConversation := serviceAgentRuntime(t, true)
	pedidoSupervisor := ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupervisor, ConversationID: supervisorConversation,
		Intent: "organizar a fila", RequestedTools: []string{"inbox.assign"},
		ToolArguments: ServiceAgentToolRequest{Assignee: "equipe-vendas"},
	}
	supervisorTurn, err := supervisor.RunTurn(pedidoSupervisor, now)
	if err != nil {
		t.Fatal(err)
	}
	supervisorCall := callByName(supervisorTurn.ToolCalls, "inbox.assign")
	if supervisorCall == nil || supervisorCall.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("o supervisor declara inbox.assign, mas atribuir exige aprovação: %+v", supervisorTurn.ToolCalls)
	}
	approvePendingForTool(t, supervisor, "org-a", "inbox.assign", "supervisora-humana")
	supervisorTurn, err = supervisor.RunTurn(pedidoSupervisor, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	supervisorCall = callByName(supervisorTurn.ToolCalls, "inbox.assign")
	if supervisorCall == nil || supervisorCall.Status != ServiceAgentToolExecuted {
		t.Fatalf("com aprovação, o supervisor precisa executar: %+v", supervisorTurn.ToolCalls)
	}
}

func TestServiceAgentToolWithoutExecutorIsUnavailableNotAvailable(t *testing.T) {
	_, inbox, crm, consent, conversationID := serviceAgentRuntime(t, true)
	// Catálogo de PRODUÇÃO: é ele que listava `commerce.stock.read` como disponível sem executor.
	runtime := NewServiceAgentRuntime(crm, inbox, consent, DefaultServiceAgentTools())
	now := time.Now().UTC()

	// `commerce.stock.read` é declarada pelo agente de COMÉRCIO — e NÃO tem executor neste runtime.
	// Antes, `DefaultServiceAgentTools` a anunciava como disponível: era um booleano sem implementação.
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentCommerce, ConversationID: conversationID,
		Intent: "tem em estoque?", RequestedTools: []string{"commerce.stock.read"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "commerce.stock.read")
	if call == nil || call.Status != ServiceAgentToolUnavailable {
		t.Fatalf("sem executor, a ferramenta precisa ser INDISPONÍVEL: %+v", turn.ToolCalls)
	}
	if call.Reason == "" {
		t.Fatal("indisponibilidade precisa de motivo explícito")
	}
	// A disponibilidade agora é DERIVADA: a ferramenta não aparece como disponível...
	for _, name := range runtime.ToolExecutors() {
		if name == "commerce.stock.read" {
			t.Fatal("commerce.stock.read não pode constar como executável")
		}
	}
	// ...mas aparece na lista de indisponíveis, com o motivo visível para quem opera.
	unavailable := runtime.UnavailableTools()
	if !containsString(unavailable, "commerce.stock.read") || !containsString(unavailable, "company.report.read") {
		t.Fatalf("as declaradas sem executor precisam ser listadas: %+v", unavailable)
	}
}

func TestServiceAgentOutboundToolRequiresConsent(t *testing.T) {
	// SEM consentimento: a ferramenta de saída não roda, e o motivo é a política.
	runtime, inbox, _, _, conversationID := serviceAgentRuntime(t, false)
	now := time.Now().UTC()
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi, tudo bem?", RequestedTools: []string{"crm.contact.read"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !turn.Escalated {
		t.Fatal("sem consentimento o turno precisa escalar")
	}
	// O caminho único de resposta é recusado para `inbox.reply` como ferramenta (evita segunda mensagem).
	turnReply, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi", RequestedTools: []string{"inbox.reply"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	replyCall := callByName(turnReply.ToolCalls, "inbox.reply")
	if replyCall == nil || replyCall.Status != ServiceAgentToolBlocked {
		t.Fatalf("inbox.reply como ferramenta precisa ser recusada: %+v", turnReply.ToolCalls)
	}
	if !strings.Contains(replyCall.Reason, "caminho único") {
		t.Fatalf("o motivo precisa explicar a recusa: %q", replyCall.Reason)
	}
	// E a caixa NÃO ganhou mensagem de saída por causa disso.
	_, history, err := inbox.Conversation("org-a", conversationID)
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range history {
		if message.Direction == "outbound" {
			t.Fatalf("nenhuma resposta podia sair sem consentimento: %+v", message)
		}
	}
}

func TestServiceAgentToolFailureIsRecordedWithRealError(t *testing.T) {
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	now := time.Now().UTC()

	// `crm.opportunity.write` sem título: o executor recusa. Com a aprovação humana no caminho, primeiro o
	// pedido fica PENDENTE; depois de aprovado, o executor roda e FALHA — e a falha precisa aparecer com o
	// motivo real, em vez de ser tratada como sucesso.
	pedido := ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSales, ConversationID: conversationID,
		Intent: "quero comprar", RequestedTools: []string{"crm.opportunity.write"},
	}
	turn, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "crm.opportunity.write"); call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("sem aprovação a escrita precisa ficar pendente: %+v", turn.ToolCalls)
	}
	approvePendingForTool(t, runtime, "org-a", "crm.opportunity.write", "ana")

	turn, err = runtime.RunTurn(pedido, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolFailed {
		t.Fatalf("falha do executor precisa ser registrada como falha: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Reason, "title") {
		t.Fatalf("o erro REAL precisa ser preservado: %q", call.Reason)
	}
	if call.Summary != "" {
		t.Fatalf("falha não pode trazer resumo de sucesso: %q", call.Summary)
	}
}

func TestServiceAgentToolScopeComesFromTheSession(t *testing.T) {
	runtime, _, crm, _, conversationID := serviceAgentRuntime(t, true)
	now := time.Now().UTC()

	// O pedido TENTA apontar para outra organização em `ToolArguments`: o runtime precisa ignorar e usar a
	// organização da sessão, senão o pedido escolheria o tenant.
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi", RequestedTools: []string{"crm.contact.read"},
		ToolArguments: ServiceAgentToolRequest{OrganizationID: "org-invasora"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.contact.read")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("a leitura precisa executar no escopo da sessão: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Summary, "Ana Souza") {
		t.Fatalf("a leitura precisa trazer o contato da SESSÃO: %q", call.Summary)
	}
	// E a outra organização continua vazia: nada foi criado lá.
	if contacts := crm.ListContacts("org-invasora", ""); len(contacts) != 0 {
		t.Fatalf("organização alheia não pode ganhar dado: %+v", contacts)
	}
}

func TestServiceAgentRepeatedRequestDoesNotExecuteTwice(t *testing.T) {
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	now := time.Now().UTC()

	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSales, ConversationID: conversationID,
		Intent: "quero o plano", RequestedTools: []string{"crm.opportunity.write", "crm.opportunity.write"},
		ToolArguments: ServiceAgentToolRequest{Title: "Plano duplicado"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Executar o efeito duas vezes por engano é pior que ignorar a repetição.
	occurrences := 0
	for _, call := range turn.ToolCalls {
		if call.Name == "crm.opportunity.write" {
			occurrences++
		}
	}
	if occurrences != 1 {
		t.Fatalf("pedido repetido não pode executar duas vezes: %d ocorrências", occurrences)
	}
}
