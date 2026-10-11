package agent

import (
	"strings"
	"testing"
	"time"
)

// Ferramentas de EMPRESA, SOCIAL e VOZ ligadas aos módulos que já existiam (estágio 35).
//
// O que estes testes travam:
//
//  1. `social.draft` é ESCRITA: sem aprovação NÃO cria rascunho nenhum — e o rascunho nasce pedindo aprovação
//     da própria empresa e em modo sandbox;
//  2. sem empresa definida (ou com várias), o pedido FALHA pedindo o que falta, em vez de escolher sozinho;
//  3. `voice.transfer` exige `session_id` e encaminha de verdade uma sessão existente — também sob aprovação;
//  4. a aprovação é amarrada à SESSÃO: autorizar uma não autoriza outra;
//  5. `knowledge.search` continua INDISPONÍVEL com motivo declarado: não há mapeamento organização→projeto, e
//     inventar um escopo de busca seria pior que dizer a verdade.
func TestServiceAgentSocialDraftRequiresApprovalAndCreatesInSandbox(t *testing.T) {
	runtime, _, companies, conversationID, now := commerceAgentRuntime(t)
	runtime.SetCompanies(companies)
	runtime.SetApprovalStore(NewServiceAgentApprovalStore(serviceAgentApprovalRoot(t.TempDir())))

	company := companies.List("org-a")[0]
	pedido := ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSocial, ConversationID: conversationID,
		Intent: "quero divulgar a promoção", RequestedTools: []string{"social.draft"},
		ToolArguments: ServiceAgentToolRequest{
			CompanyID: company.ID, Provider: "instagram",
			Title: "Promoção de camisetas", Body: "Camiseta azul com 20% de desconto nesta semana.",
		},
	}
	turn, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "social.draft")
	if call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("publicar rascunho social é escrita e exige aprovação: %+v", turn.ToolCalls)
	}
	// NADA foi criado enquanto a aprovação não veio.
	if drafts := companies.List("org-a")[0].SocialDrafts; len(drafts) != 0 {
		t.Fatalf("nenhum rascunho pode existir antes da aprovação: %+v", drafts)
	}

	approvePendingForTool(t, runtime, "org-a", "social.draft", "ana")
	turn, err = runtime.RunTurn(pedido, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call = callByName(turn.ToolCalls, "social.draft")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("com aprovação o rascunho precisa ser criado: %+v", turn.ToolCalls)
	}
	drafts := companies.List("org-a")[0].SocialDrafts
	if len(drafts) != 1 {
		t.Fatalf("exatamente um rascunho: %+v", drafts)
	}
	// O rascunho nasce em SANDBOX e exigindo aprovação da empresa: nada é publicado por conta do agente.
	if drafts[0].Mode != "sandbox" || !drafts[0].ApprovalRequired || drafts[0].Approved {
		t.Fatalf("rascunho precisa nascer em sandbox e não aprovado: %+v", drafts[0])
	}
	if drafts[0].PublishedAt != nil {
		t.Fatal("rascunho NÃO pode ser publicado")
	}
	if !strings.Contains(call.Summary, "sem publicação") {
		t.Fatalf("o resumo precisa declarar que não houve publicação: %q", call.Summary)
	}
}

func TestServiceAgentSocialDraftRefusesToGuessTheCompany(t *testing.T) {
	runtime, _, companies, conversationID, now := commerceAgentRuntime(t)
	runtime.SetCompanies(companies)
	runtime.SetApprovalStore(NewServiceAgentApprovalStore(serviceAgentApprovalRoot(t.TempDir())))

	// Duas empresas: escolher uma por conta própria seria decidir pelo humano.
	if _, err := companies.Create(Company{ID: "emp-2", OrganizationID: "org-a", Name: "Segunda Loja", Currency: "BRL"}); err != nil {
		t.Fatal(err)
	}
	pedido := ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSocial, ConversationID: conversationID,
		Intent: "divulgar", RequestedTools: []string{"social.draft"},
		ToolArguments: ServiceAgentToolRequest{Provider: "instagram", Title: "T", Body: "B"},
	}
	turn, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	// O caminho é: pendente de aprovação, e o executor — quando rodar — recusa sem company_id. Aqui basta
	// provar que NADA foi criado e que a ferramenta não escolheu sozinha.
	if call := callByName(turn.ToolCalls, "social.draft"); call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("a ferramenta precisa ficar pendente: %+v", turn.ToolCalls)
	}
	for _, company := range companies.List("org-a") {
		if len(company.SocialDrafts) != 0 {
			t.Fatalf("nenhum rascunho pode ser criado sem dizer a empresa: %+v", company.SocialDrafts)
		}
	}

	// Aprovado, o executor roda e FALHA pedindo o que falta — não inventa a empresa.
	approvePendingForTool(t, runtime, "org-a", "social.draft", "ana")
	turn, err = runtime.RunTurn(pedido, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "social.draft")
	if call == nil || call.Status != ServiceAgentToolFailed {
		t.Fatalf("sem company_id o pedido precisa FALHAR: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Reason, "company_id") {
		t.Fatalf("a falha precisa dizer o que falta: %q", call.Reason)
	}
	for _, company := range companies.List("org-a") {
		if len(company.SocialDrafts) != 0 {
			t.Fatalf("nenhum rascunho pode ser criado em falha: %+v", company.SocialDrafts)
		}
	}
}

func TestServiceAgentVoiceTransferUsesRealSessionUnderApproval(t *testing.T) {
	runtime, _, companies, conversationID, now := commerceAgentRuntime(t)
	runtime.SetCompanies(companies)
	runtime.SetApprovalStore(NewServiceAgentApprovalStore(serviceAgentApprovalRoot(t.TempDir())))
	company := companies.List("org-a")[0]

	// Sessão de voz REAL, criada pelo módulo Tel-Agent que já existe.
	if _, _, err := companies.StartTelAgentSession(company.ID, "org-a", "cliente-voz", "phone"); err != nil {
		t.Fatal(err)
	}
	sessao := companies.List("org-a")[0].TelAgentSessions[0]

	// Sem session_id, a ferramenta FALHA dizendo o que falta (não encaminha "a sessão mais recente").
	semSessao := ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentVoice, ConversationID: conversationID,
		Intent: "quero falar com um atendente", RequestedTools: []string{"voice.transfer"},
		ToolArguments: ServiceAgentToolRequest{CompanyID: company.ID},
	}
	turn, err := runtime.RunTurn(semSessao, now)
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "voice.transfer"); call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("transferência é escrita e exige aprovação: %+v", turn.ToolCalls)
	}
	approvePendingForTool(t, runtime, "org-a", "voice.transfer", "supervisora")
	turn, err = runtime.RunTurn(semSessao, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "voice.transfer")
	if call == nil || call.Status != ServiceAgentToolFailed || !strings.Contains(call.Reason, "session_id") {
		t.Fatalf("sem session_id precisa falhar explicando: %+v", turn.ToolCalls)
	}

	// Com a sessão REAL: outra aprovação (a anterior foi consumida) e transferência de verdade.
	comSessao := semSessao
	comSessao.ToolArguments.SessionID = sessao.ID
	turn, err = runtime.RunTurn(comSessao, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "voice.transfer"); call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("cada sessão exige aprovação própria: %+v", turn.ToolCalls)
	}
	approvePendingForTool(t, runtime, "org-a", "voice.transfer", "supervisora")
	turn, err = runtime.RunTurn(comSessao, now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call = callByName(turn.ToolCalls, "voice.transfer")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("com sessão e aprovação a transferência precisa executar: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Summary, sessao.ID) {
		t.Fatalf("o resumo precisa nomear a sessão transferida: %q", call.Summary)
	}
	// A sessão no store conta a verdade: houve pedido de transferência humana e ela está aguardando humano.
	atualizada := companies.List("org-a")[0].TelAgentSessions[0]
	if atualizada.State != TelSessionWaitingHuman {
		t.Fatalf("a sessão precisa aguardar humano após a transferência: %+v", atualizada)
	}
}

func TestServiceAgentVoiceApprovalIsBoundToTheSession(t *testing.T) {
	runtime, _, companies, conversationID, now := commerceAgentRuntime(t)
	runtime.SetCompanies(companies)
	runtime.SetApprovalStore(NewServiceAgentApprovalStore(serviceAgentApprovalRoot(t.TempDir())))
	company := companies.List("org-a")[0]
	for _, actor := range []string{"cliente-1", "cliente-2"} {
		if _, _, err := companies.StartTelAgentSession(company.ID, "org-a", actor, "phone"); err != nil {
			t.Fatal(err)
		}
	}
	sessoes := companies.List("org-a")[0].TelAgentSessions
	if len(sessoes) != 2 {
		t.Fatalf("esperava duas sessões: %+v", sessoes)
	}

	// Aprova a transferência da PRIMEIRA sessão...
	primeira := ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentVoice, ConversationID: conversationID,
		Intent: "quero humano", RequestedTools: []string{"voice.transfer"},
		ToolArguments: ServiceAgentToolRequest{CompanyID: company.ID, SessionID: sessoes[0].ID},
	}
	if _, err := runtime.RunTurn(primeira, now); err != nil {
		t.Fatal(err)
	}
	approvePendingForTool(t, runtime, "org-a", "voice.transfer", "supervisora")

	// ...e tenta usar essa aprovação para a SEGUNDA sessão: precisa exigir decisão nova.
	segunda := primeira
	segunda.ToolArguments.SessionID = sessoes[1].ID
	turn, err := runtime.RunTurn(segunda, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "voice.transfer")
	if call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("aprovação de uma sessão não pode autorizar outra: %+v", turn.ToolCalls)
	}
}

func TestServiceAgentKnowledgeSearchStaysUnavailableWithReason(t *testing.T) {
	runtime, _, companies, conversationID, now := commerceAgentRuntime(t)
	runtime.SetCompanies(companies)
	runtime.SetApprovalStore(NewServiceAgentApprovalStore(serviceAgentApprovalRoot(t.TempDir())))

	// O agente de SUPORTE declara `knowledge.search`, mas não há mapeamento organização→projeto no runtime —
	// e inventar um escopo de busca seria pior que declarar a indisponibilidade.
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "como faço a troca?", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || call.Status != ServiceAgentToolUnavailable {
		t.Fatalf("knowledge.search precisa seguir INDISPONÍVEL: %+v", turn.ToolCalls)
	}
	if call.Reason == "" {
		t.Fatal("a indisponibilidade precisa vir com motivo")
	}
	if containsString(runtime.ToolExecutors(), "knowledge.search") {
		t.Fatalf("knowledge.search não pode constar como executável: %+v", runtime.ToolExecutors())
	}
}
