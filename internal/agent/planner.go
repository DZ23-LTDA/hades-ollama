package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ollama/ollama/api"
)

type Planner interface {
	Plan(ctx context.Context, mission Mission) ([]Step, error)
}

type PlannerResolver interface {
	// ResolvePlanner must return an executable planner for the requested provider.
	// It must not silently substitute a different provider or a rule-only plan.
	ResolvePlanner(provider, model string) (Planner, error)
}

type PlannerResolution struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Reason   string `json:"reason"`
	CostTag  string `json:"cost_tag,omitempty"`
}

type RoutedPlannerResolver interface {
	// organizationID lets the resolver enforce per-organization policy (e.g. a
	// spend cap) when auto-routing to a provider.
	ResolvePlannerForMission(ctx context.Context, provider, model, organizationID string, capabilities []string) (Planner, PlannerResolution, error)
}

// plannerChatClient e' o nome historico do cliente de chat do planejador. E-666: passou a ser ALIAS da interface
// exportada `InferenceChatClient`, para que o runtime injete o MESMO tipo em vez de manter duas abstracoes
// paralelas para falar com modelo.
type plannerChatClient = InferenceChatClient

// PlanUsageSink permite que o runtime injete o destino do consumo do modelo. Fica em uma
// interface separada de Planner para não quebrar planejadores que não reportam consumo.
type PlanUsageSink interface {
	SetPlanUsageSink(*ModelUsage)
}

// PlanUsageReporter expõe o consumo da última chamada de planejamento.
type PlanUsageReporter interface {
	LastPlanUsage() (ModelUsage, bool)
}

type RulePlanner struct{}

func (RulePlanner) Plan(_ context.Context, mission Mission) ([]Step, error) {
	objective := strings.ToLower(mission.Objective)
	if strings.Contains(objective, "test") || strings.Contains(objective, "testar") || strings.Contains(objective, "coding") || strings.Contains(objective, "repair") || strings.Contains(objective, "worktree") || strings.Contains(objective, "merge") {
		steps := []Step{
			{
				ID:               "step_1",
				Kind:             "workspace.write",
				Title:            "Aplicar alterações de código no workspace isolado",
				Risk:             RiskWrite,
				RequiresApproval: true,
				State:            StepPending,
				Input:            map[string]any{"path": "main.go", "content": "package main\n\nfunc Answer() int { return 42 }\n"},
			},
			{
				ID:               "step_2",
				Kind:             "project.test.run",
				Title:            "Executar runner de testes do projeto no sandbox sem egress",
				Risk:             RiskRead,
				RequiresApproval: false,
				State:            StepPending,
				Input:            map[string]any{"framework": "go", "auto_repair": mission.AutoRepair},
			},
		}
		if mission.GitWorktreeActive || strings.Contains(objective, "merge") {
			steps = append(steps, Step{
				ID:               "step_3",
				Kind:             "git.merge.origin",
				Title:            "Mesclar alterações do branch da missão de volta ao repositório original",
				Risk:             RiskWrite,
				RequiresApproval: true,
				State:            StepPending,
				Input: map[string]any{
					"repo_root":    mission.GitRepoRoot,
					"branch_name":  mission.GitBranch,
					"worktree_dir": mission.GitWorktreePath,
				},
			})
		}
		return steps, nil
	}
	if strings.Contains(objective, "navegar") || strings.Contains(objective, "browser") {
		return []Step{
			{
				ID:               "step_1",
				Kind:             "browser.operator",
				Title:            "Navegar na página web e capturar frame visual",
				Risk:             RiskRead,
				RequiresApproval: false,
				State:            StepPending,
				Input:            map[string]any{"action": "navigate", "url": "https://example.com"},
			},
			{
				ID:               "step_2",
				Kind:             "workspace.write",
				Title:            "Gravar relatório e consolidar artefato da missão",
				Risk:             RiskWrite,
				RequiresApproval: true,
				State:            StepPending,
				Input:            map[string]any{"path": "relatorio-missao.md", "content": "# Relatório de Execução da Missão\n\nNavegação realizada com sucesso e frame visual espelhado em tempo real.\n"},
			},
		}, nil
	}
	if strings.Contains(objective, "escrever") || strings.Contains(objective, "criar arquivo") || strings.Contains(objective, "editar") {
		return []Step{{
			ID:               "step_1",
			Kind:             "workspace.write",
			Title:            "Escrever o resultado solicitado no workspace",
			Risk:             RiskWrite,
			RequiresApproval: true,
			State:            StepPending,
			Input:            map[string]any{"path": "agent-output.txt", "content": mission.Objective + "\n"},
		}}, nil
	}
	return []Step{{
		ID:    "step_1",
		Kind:  "workspace.list",
		Title: "Inspecionar o workspace autorizado",
		Risk:  RiskRead,
		State: StepPending,
		Input: map[string]any{"path": ".", "max_entries": 100},
	}}, nil
}

type UnconfiguredPlanner struct{}

func (UnconfiguredPlanner) Plan(_ context.Context, _ Mission) ([]Step, error) {
	return nil, errors.New("planner model is not configured")
}

// missionContextPreamble monta o preâmbulo do Context Bootstrap para uma missão.
// Devolve string vazia quando a missão não tem escopo de organização/projeto:
// nesse caso o planejador segue apenas com o prompt de sistema original.
// Capacidade inválida NÃO vira permissão: quando a política recusa a lista, as
// capacidades cruas continuam no preâmbulo e aparecem como INDISPONIVEL, o que
// é mais honesto do que esconder o pedido.
func missionContextPreamble(mission Mission) string {
	if strings.TrimSpace(mission.OrganizationID) == "" && strings.TrimSpace(mission.ProjectID) == "" {
		return ""
	}
	policy := DefaultCapabilityPolicy()
	granted := mission.Capabilities
	if validated, err := policy.ValidateMissionCapabilities(mission.Capabilities); err == nil {
		granted = validated
	}
	bootstrap, err := BuildContextBootstrap(ContextBootstrapInput{
		OrganizationID:      mission.OrganizationID,
		ProjectID:           mission.ProjectID,
		MissionID:           mission.ID,
		Workspace:           mission.Workspace,
		Objective:           mission.Objective,
		GrantedCapabilities: granted,
		Policy:              policy,
		PlanTitles:          planTitles(mission.Plan),
	})
	if err != nil {
		return ""
	}
	return bootstrap.SystemMessage()
}

func planTitles(steps []Step) []string {
	titles := make([]string, 0, len(steps))
	for _, step := range steps {
		if title := strings.TrimSpace(step.Title); title != "" {
			titles = append(titles, title)
		}
	}
	return titles
}

type OllamaPlanner struct {
	Client plannerChatClient
	Model  string
	// Usage, quando não nulo, recebe o consumo da última chamada de planejamento. O runtime
	// injeta o destino via SetPlanUsageSink.
	Usage *ModelUsage
	// Price é o preço declarado do modelo. Sem preço, os tokens continuam sendo registrados e
	// o custo fica declarado como DESCONHECIDO — nunca como zero.
	Price ModelPrice
	// Tools é o contrato REAL das ferramentas registradas, injetado pelo runtime. Vazio
	// significa prompt sem contrato (comportamento anterior à correção).
	Tools []ToolDescriptor
}

// SetPlanUsageSink registra o destino do consumo. Receiver por ponteiro de propósito: sem
// ponteiro o campo não seria gravável e o consumo se perderia.
func (p *OllamaPlanner) SetPlanUsageSink(sink *ModelUsage) {
	p.Usage = sink
}

// LastPlanUsage devolve o consumo acumulado, se houve alguma chamada.
func (p OllamaPlanner) LastPlanUsage() (ModelUsage, bool) {
	if p.Usage == nil || p.Usage.Calls <= 0 {
		return ModelUsage{}, false
	}
	return *p.Usage, true
}

// PlanToolContractSink permite que o runtime entregue ao planejador o contrato REAL das
// ferramentas registradas.
//
// Por que existe: o prompt de sistema só listava os kinds permitidos, sem dizer quais campos
// de input cada um exige. Um modelo local (qwen2.5:7b) devolveu então `workspace.read` com
// input VAZIO — plano sintaticamente válido e inutilizável. Repetir a lista à mão no prompt
// divergiria do código; aqui o prompt é gerado dos próprios descritores.
type PlanToolContractSink interface {
	SetPlanToolContract([]ToolDescriptor)
}

func (p *OllamaPlanner) SetPlanToolContract(descriptors []ToolDescriptor) {
	p.Tools = descriptors
}

// PlanningToolContract renderiza o contrato das ferramentas a partir dos descritores reais.
// Ordena por nome para o prompt ser determinístico (o registro não garante ordem estável).
func PlanningToolContract(descriptors []ToolDescriptor) string {
	if len(descriptors) == 0 {
		return ""
	}
	linhas := make([]string, 0, len(descriptors))
	for _, descritor := range descriptors {
		exigidos := "nenhum"
		if len(descritor.RequiredInputs) > 0 {
			exigidos = strings.Join(descritor.RequiredInputs, ", ")
		}
		aprovacao := "nao"
		if descritor.RequiresApproval {
			aprovacao = "sim"
		}
		linhas = append(linhas, fmt.Sprintf("- %s (risco %s, aprovacao %s) exige input: %s", descritor.Name, descritor.Risk, aprovacao, exigidos))
	}
	sort.Strings(linhas)
	return "Contrato das ferramentas registradas. O input de cada passo DEVE trazer os campos exigidos; passo sem campo exigido e recusado antes de executar:\n" + strings.Join(linhas, "\n")
}

func (p OllamaPlanner) Plan(ctx context.Context, mission Mission) ([]Step, error) {
	model := strings.TrimSpace(mission.Model)
	if model == "" {
		model = strings.TrimSpace(p.Model)
	}
	if p.Client == nil {
		return nil, errors.New("planner provider client is unavailable")
	}
	if model == "" {
		return nil, errors.New("planner provider model is required")
	}
	stream := false
	format := json.RawMessage(`"json"`)
	// Context Bootstrap (estágio 5): quando a missão pertence a uma organização
	// ou projeto, o contexto canônico do Hades entra como mensagem de sistema
	// ANTES do objetivo, declarando identidade, escopo e permissões. Sem escopo
	// de tenant nada é injetado — na dúvida, não se vaza contexto de outro dono.
	instrucaoSistema := "You are a mission planner. Return only JSON with a top-level steps array. Each step must have kind, title, risk, requires_approval, and input. Allowed kinds are workspace.list, workspace.read, git.repo.inspect, workspace.write, terminal.exec, sandbox.exec, browser.operator, desktop.companion, mcp.call, mcp.remote.call, connector.http, and media.process only when a media provider is configured, plus model.infer for a step that must actually run one inference call on the model routed for that step. Never invent completed results. Use read risk for inspection, write risk for filesystem changes, external_side_effect for browser, desktop, MCP, connectors, and media provider actions, and require approval for write, terminal, sandbox, browser, desktop, MCP, connector, and media steps. Use workspace.list to inspect a DIRECTORY and workspace.read to read a specific FILE: never use workspace.read to inspect a directory. Workspace paths MUST be RELATIVE to the workspace root (for example \"README.md\" or \"cmd/app\" or \".\" for the root directory): an absolute path such as \"/\" is refused before execution."
	// O contrato vem dos DESCRITORES reais: dizer só os kinds permitidos deixava o modelo
	// devolver passos sem os campos exigidos.
	if contrato := PlanningToolContract(p.Tools); contrato != "" {
		instrucaoSistema += "\n" + contrato
	}
	messages := []api.Message{
		{Role: "system", Content: instrucaoSistema},
	}
	if preamble := missionContextPreamble(mission); preamble != "" {
		messages = append(messages, api.Message{Role: "system", Content: preamble})
	}
	messages = append(messages, api.Message{Role: "user", Content: fmt.Sprintf("Objective: %s", mission.Objective)})
	request := &api.ChatRequest{
		Model:    model,
		Stream:   &stream,
		Format:   format,
		Messages: messages,
	}
	var response string
	var consumo ModelUsage
	err := p.Client.Chat(ctx, request, func(chatResponse api.ChatResponse) error {
		response = chatResponse.Message.Content
		// Em streaming os contadores chegam no último pedaço: guardar o MAIOR valor visto
		// evita contar o mesmo consumo mais de uma vez.
		if atual := UsageFromMetrics(chatResponse.Metrics, mission.Provider, model, p.Price); atual.PromptTokens+atual.CompletionTokens > consumo.PromptTokens+consumo.CompletionTokens {
			consumo = atual
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("planner provider request failed: %w", err)
	}
	if consumo.Calls == 0 {
		// A chamada ACONTECEU, mesmo sem contadores: registrar uma chamada sem tokens é mais
		// honesto do que esconder que houve consumo não medido.
		consumo = UsageFromMetrics(api.Metrics{}, mission.Provider, model, p.Price)
	}
	if p.Usage != nil {
		*p.Usage = consumo
	}
	steps, err := parsePlan(response)
	if err != nil {
		return nil, fmt.Errorf("planner returned invalid plan: %w", err)
	}
	return normalizeSteps(steps)
}

func parsePlan(content string) ([]Step, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	var payload struct {
		Steps []map[string]any `json:"steps"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, err
	}
	if len(payload.Steps) == 0 {
		return nil, errors.New("planner returned no steps")
	}
	steps := make([]Step, 0, len(payload.Steps))
	for i, raw := range payload.Steps {
		// Smaller local models often return "input" as a plain string; keep
		// it as text instead of rejecting the whole plan. Validation of kinds
		// and risks still happens in normalizeSteps.
		switch input := raw["input"].(type) {
		case string:
			raw["input"] = map[string]any{"text": input}
		case nil, map[string]any:
		default:
			raw["input"] = map[string]any{"value": input}
		}
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		var step Step
		if err := json.Unmarshal(data, &step); err != nil {
			return nil, fmt.Errorf("step %d: %w", i+1, err)
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func normalizeSteps(steps []Step) ([]Step, error) {
	if len(steps) == 0 || len(steps) > 32 {
		return nil, errors.New("planner step count is outside the allowed range")
	}
	allowed := map[string]RiskClass{
		"workspace.list":    RiskRead,
		"workspace.read":    RiskRead,
		"git.repo.inspect":  RiskRead,
		"workspace.write":   RiskWrite,
		"terminal.exec":     RiskWrite,
		"sandbox.exec":      RiskWrite,
		"browser.operator":  RiskExternalSideEffect,
		"desktop.companion": RiskExternalSideEffect,
		"mcp.call":          RiskExternalSideEffect,
		"mcp.remote.call":   RiskExternalSideEffect,
		"connector.http":    RiskExternalSideEffect,
		"media.process":     RiskExternalSideEffect,
		// SEC-01: this registry is the single source of truth for step risk, so it
		// must declare the same class as the tool descriptor: project.test.run
		// executes project code on the host (external side effect).
		"project.test.run": RiskExternalSideEffect,
		"git.merge.origin": RiskWrite,
		// E-666: inferencia por passo (`model.infer`). E' EFEITO EXTERNO - sai da maquina, custa e pode enviar dado -
		// e portanto exige aprovacao, como qualquer ferramenta dessa classe. *A classe aqui tem de ser a MESMA do
		// descritor da ferramenta: este mapa e' a fonte unica do risco (SEC-01), e uma divergencia entre os dois
		// deixaria o plano rebaixar o risco.* Sem esta linha o kind nem e' planejavel: `normalizeSteps` recusa
		// qualquer ferramenta fora deste mapa ("planner returned unsupported tool") - *e um executor inalcancavel e'
		// so' uma promessa.*
		KindDeInferencia: RiskExternalSideEffect,
	}
	for i := range steps {
		if _, ok := allowed[steps[i].Kind]; !ok {
			return nil, fmt.Errorf("planner returned unsupported tool %q", steps[i].Kind)
		}
		steps[i].ID = fmt.Sprintf("step_%d", i+1)
		if steps[i].Title == "" {
			steps[i].Title = steps[i].Kind
		}
		// O risco é sempre o do registry: o planner (saída de LLM) não pode
		// rebaixar o risco de uma ferramenta para escapar do approval.
		steps[i].Risk = allowed[steps[i].Kind]
		if steps[i].Risk != RiskRead {
			steps[i].RequiresApproval = true
		}
		steps[i].State = StepPending
	}
	return steps, nil
}
