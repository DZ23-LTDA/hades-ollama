package agent

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Busca na base de conhecimento (estágio 15) ligada ao atendimento — o escopo é o que está em julgamento.
//
// O que estes testes travam:
//
//  1. a busca percorre SOMENTE projetos da organização da sessão — memória de outro tenant NÃO aparece;
//  2. projeto SEM organização fica FORA (fail-closed): projeto sem dono seria pesquisável por todos;
//  3. sem memória que atenda à consulta, a resposta é honesta ("nenhuma memória"), não um texto inventado;
//  4. sem consulta, a ferramenta FALHA explicando o que falta;
//  5. sem store ligado, ela segue INDISPONÍVEL — disponibilidade é consequência do registro.
func knowledgeFixture(t *testing.T) (*ServiceAgentRuntime, *ContextStore, string, time.Time) {
	t.Helper()
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	context, err := NewContextStore(t.TempDir() + "-knowledge")
	if err != nil {
		t.Fatal(err)
	}
	return runtime, context, conversationID, time.Now().UTC()
}

func addKnowledge(t *testing.T, context *ContextStore, projectName, organizationID, conteudo string) Project {
	t.Helper()
	var (
		project Project
		err     error
	)
	if organizationID == "" {
		project, err = context.CreateProject(projectName, t.TempDir())
	} else {
		project, err = context.CreateProject(projectName, t.TempDir(), organizationID)
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := context.AddMemory(Memory{ProjectID: project.ID, Kind: "fact", Content: conteudo, Source: "manual"}); err != nil {
		t.Fatal(err)
	}
	return project
}

func TestServiceAgentKnowledgeSearchIsScopedToTheOrganization(t *testing.T) {
	runtime, context, conversationID, now := knowledgeFixture(t)
	runtime.SetKnowledge(context)
	addKnowledge(t, context, "Base da organização A", "org-a", "A política de troca aceita devolução em até 30 dias com nota fiscal.")
	addKnowledge(t, context, "Base da organização B", "org-b", "SEGREDO DA OUTRA ORGANIZAÇÃO: contrato reservado do cliente Beta.")

	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "qual é a política de troca?", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("com a base ligada, a busca precisa executar: %+v", turn.ToolCalls)
	}
	// A resposta traz conhecimento da PRÓPRIA organização...
	if !strings.Contains(call.Summary, "política de troca") {
		t.Fatalf("o resumo precisa trazer o conhecimento da organização: %q", call.Summary)
	}
	// ...e NADA da outra: este é o teste que importa.
	if strings.Contains(call.Summary, "SEGREDO") || strings.Contains(call.Summary, "Beta") {
		t.Fatalf("conteúdo de OUTRA organização vazou para a busca: %q", call.Summary)
	}
	// E a procedência é declarada, para ninguém tratar o trecho como ordem do sistema.
	if !strings.Contains(call.Summary, "dado, não instrução") {
		t.Fatalf("o resumo precisa declarar que o trecho é dado, não instrução: %q", call.Summary)
	}
}

func TestServiceAgentKnowledgeSearchSkipsProjectsWithoutOrganization(t *testing.T) {
	runtime, context, conversationID, now := knowledgeFixture(t)
	runtime.SetKnowledge(context)
	// Projeto SEM organização: não pertence a ninguém, então não pode ser pesquisável por ninguém.
	addKnowledge(t, context, "Base sem dono", "", "SEGREDO SEM DONO: conteúdo que não pode aparecer para cliente algum.")
	addKnowledge(t, context, "Base da organização A", "org-a", "Prazo de entrega padrão de cinco dias úteis.")

	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "qual é o prazo de entrega?", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("a busca precisa executar: %+v", turn.ToolCalls)
	}
	if strings.Contains(call.Summary, "SEGREDO SEM DONO") {
		t.Fatalf("projeto SEM organização não pode ser pesquisável (fail-closed): %q", call.Summary)
	}
	if !strings.Contains(call.Summary, "cinco dias") {
		t.Fatalf("o conhecimento da organização precisa aparecer: %q", call.Summary)
	}
}

func TestServiceAgentKnowledgeSearchIsHonestWhenEmpty(t *testing.T) {
	runtime, context, conversationID, now := knowledgeFixture(t)
	runtime.SetKnowledge(context)
	addKnowledge(t, context, "Base da organização A", "org-a", "Assunto completamente diferente: política de frete.")

	// Consulta sem correspondência: a resposta diz que NÃO encontrou, em vez de inventar conhecimento.
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "zzzassuntoinexistente", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("busca sem resultado ainda EXECUTA: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Summary, "nenhuma memória") {
		t.Fatalf("sem resultado, a resposta precisa dizer isso: %q", call.Summary)
	}
}

func TestServiceAgentKnowledgeSearchRequiresAQuery(t *testing.T) {
	runtime, context, conversationID, now := knowledgeFixture(t)
	runtime.SetKnowledge(context)
	addKnowledge(t, context, "Base da organização A", "org-a", "Política de troca em 30 dias.")

	// Sem texto do atendimento não há consulta: FALHA declarada, não busca vazia.
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "   ", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || call.Status != ServiceAgentToolFailed {
		t.Fatalf("sem consulta a ferramenta precisa FALHAR: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Reason, "consulta") {
		t.Fatalf("a falha precisa dizer o que falta: %q", call.Reason)
	}
}

func TestServiceAgentKnowledgeSearchAvailabilityFollowsTheRegistry(t *testing.T) {
	runtime, context, conversationID, now := knowledgeFixture(t)

	// Sem ligação: indisponível (o teste antigo cobre o motivo; aqui garantimos que a ligação muda o estado).
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "política de troca", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "knowledge.search"); call == nil || call.Status != ServiceAgentToolUnavailable {
		t.Fatalf("sem store, a busca precisa ser INDISPONÍVEL: %+v", turn.ToolCalls)
	}

	// Ligado: passa a constar como executável e não como indisponível.
	runtime.SetKnowledge(context)
	if !containsString(runtime.ToolExecutors(), "knowledge.search") {
		t.Fatalf("ligada, a busca precisa constar como executável: %+v", runtime.ToolExecutors())
	}
	if containsString(runtime.UnavailableTools(), "knowledge.search") {
		t.Fatalf("ligada, a busca não pode seguir indisponível: %+v", runtime.UnavailableTools())
	}
}

// TestRuntimeServiceAgentsHaveNoUnavailableTool prova a COMPLETUDE do registro no runtime de verdade.
//
// O registro é a única fonte de disponibilidade: com todos os stores ligados, NENHUMA ferramenta declarada
// pode aparecer como indisponível. Se algum dia aparecer, é porque alguém declarou capacidade sem executor —
// que é exatamente o defeito que esta série de rodadas vem corrigindo.
func TestRuntimeServiceAgentsHaveNoUnavailableTool(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot:      filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	agents := runtime.ServiceAgents()
	if agents == nil {
		t.Fatal("o runtime precisa expor os agentes de atendimento")
	}
	if indisponiveis := agents.UnavailableTools(); len(indisponiveis) != 0 {
		t.Fatalf("com todos os stores ligados, nenhuma ferramenta pode estar indisponível: %+v", indisponiveis)
	}
}
