package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ScheduleStepKind is the closed vocabulary of steps a persisted schedule flow
// may contain. The values mirror the node types published by the flow editor
// (app/ui/app/src/lib/flowGraph.ts), so a stored graph round-trips without a
// translation table and without vocabulary drift.
type ScheduleStepKind string

const (
	// ScheduleStepTriggerInterval starts the flow when the schedule interval
	// elapses. The interval itself stays in Schedule.IntervalSeconds/NextRunAt;
	// the step exists so the persisted graph keeps the editor's shape.
	ScheduleStepTriggerInterval ScheduleStepKind = "trigger.interval"
	// ScheduleStepTriggerWebhook starts the flow from an authenticated webhook
	// delivery instead of the interval.
	ScheduleStepTriggerWebhook ScheduleStepKind = "trigger.webhook"
	// ScheduleStepActionMission instantiates exactly one mission with the step
	// objective. It is the only step kind with an external side effect.
	ScheduleStepActionMission ScheduleStepKind = "action.mission"
	// ScheduleStepConditionIf gates its dependents on the recorded outcome of
	// one dependency, without any expression language to evaluate.
	ScheduleStepConditionIf ScheduleStepKind = "condition.if"
	// ScheduleStepActionTransform publica um valor no escopo da execução a partir
	// de um template com variáveis. É determinístico e puro: não fala com o mundo
	// externo, só transforma texto já disponível.
	ScheduleStepActionTransform ScheduleStepKind = "action.transform"
	// ScheduleStepLogicLoop instancia UMA missão por item de uma lista LIMITADA.
	// Cada iteração tem chave de idempotência própria, então repetir a execução
	// não duplica as missões do laço.
	ScheduleStepLogicLoop ScheduleStepKind = "logic.loop"
	// ScheduleStepApprovalRequest pausa o fluxo até uma decisão HUMANA registrada
	// para (passo, execução). Sem decisão o passo fica pendente e os dependentes
	// não rodam — o fluxo espera, não adivinha.
	ScheduleStepApprovalRequest ScheduleStepKind = "approval.request"
	// ScheduleStepActionConnector chama um conector CONFIGURADO pelo caminho real
	// (ConnectorManager.CallForOrganization): credencial do tenant, política de
	// egress e escopo de organização. Método que ESCREVE exige decisão humana
	// aprovada para a mesma execução antes de sair.
	ScheduleStepActionConnector ScheduleStepKind = "action.connector"
)

// Métodos de conector que MUDAM estado. Chamada de escrita não sai por conta
// própria: depende de aprovação registrada para a mesma execução.
var scheduleConnectorWriteMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// maxScheduleConnectorBody limita o corpo enviado e o trecho guardado como
// evidência: um conector não deve transformar o estado do motor em depósito de
// payload alheio.
const (
	maxScheduleConnectorBody       = 16 << 10
	maxScheduleConnectorBodyInVars = 2000
)

// Limites do escopo de variáveis e do laço. Um motor que roda a cada tick precisa
// de teto explícito: laço sem limite é gerador de missões, e valor sem limite é
// memória de estado sem fim.
const (
	maxScheduleVariables    = 32
	maxScheduleVariableSize = 4096
	maxScheduleLoopItems    = 10
	// maxScheduleMissionsPerRun impede que a combinação de passos de um grafo
	// crie missões sem controle numa única execução.
	maxScheduleMissionsPerRun = 24
)

// Outcomes a condition.if step can expect from its single dependency.
const (
	ScheduleExpectSucceeded = "succeeded"
	ScheduleExpectFailed    = "failed"
)

// maxScheduleFlowSteps bounds one persisted graph so a stored flow stays
// reviewable and a dispatch tick keeps a predictable cost.
const maxScheduleFlowSteps = 12

// Retry por passo. O padrão é 2 (uma repetição) porque a criação da missão é
// idempotente — repetir não cria missão duplicada — e uma falha transitória não
// deve abortar o fluxo inteiro. O teto impede laço longo num motor que roda a
// cada tick.
const (
	ScheduleStepDefaultAttempts = 2
	ScheduleStepMaxAttempts     = 5
	// scheduleStepRetryBaseDelay é o atraso base entre tentativas (linear e
	// limitado): o motor é local e roda em tick, então esperas longas travariam
	// o despacho.
	scheduleStepRetryBaseDelay = 50 * time.Millisecond
	scheduleStepRetryMaxDelay  = 500 * time.Millisecond
)

// scheduleStepAttempts resolve o número de tentativas declarado, aplicando o
// padrão e o teto. Valor acima do teto é recusado na validação, não silenciado.
func scheduleStepAttempts(step ScheduleStep) int {
	if step.MaxAttempts <= 0 {
		return ScheduleStepDefaultAttempts
	}
	if step.MaxAttempts > ScheduleStepMaxAttempts {
		return ScheduleStepMaxAttempts
	}
	return step.MaxAttempts
}

// runWithRetry executa a ação até maxAttempts e devolve quantas tentativas fez.
// O erro devolvido é o da ÚLTIMA tentativa: o relatório mostra o motivo real da
// falha final, e não o primeiro erro de uma sequência que se recuperou.
func runWithRetry(maxAttempts int, attempt func() error) (int, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	var err error
	for try := 1; try <= maxAttempts; try++ {
		err = attempt()
		if err == nil {
			return try, nil
		}
		if try < maxAttempts {
			delay := time.Duration(try) * scheduleStepRetryBaseDelay
			if delay > scheduleStepRetryMaxDelay {
				delay = scheduleStepRetryMaxDelay
			}
			time.Sleep(delay)
		}
	}
	return maxAttempts, err
}

// ErrUnknownScheduleStep reports a step kind outside the closed vocabulary. It
// is fail-closed: an unrecognised node never executes.
var ErrUnknownScheduleStep = errors.New("unknown schedule step kind")

// ScheduleStep is one node of a persisted schedule flow. An empty Steps slice on
// Schedule keeps the historical behaviour: one mission carrying the schedule
// objective.
type ScheduleStep struct {
	ID string `json:"id"`
	// Kind is validated against the closed vocabulary before the graph is
	// stored and again before it runs.
	Kind ScheduleStepKind `json:"kind"`
	// DependsOn lists the steps that must have succeeded before this one runs.
	DependsOn []string `json:"depends_on,omitempty"`
	// Objective is the mission objective for kind action.mission.
	Objective string `json:"objective,omitempty"`
	// Expect is the required dependency outcome for kind condition.if, either
	// ScheduleExpectSucceeded or ScheduleExpectFailed.
	Expect string `json:"expect,omitempty"`
	// MaxAttempts é quantas vezes um passo action.mission pode tentar antes de
	// falhar o fluxo. Vazio/0 usa o padrão do motor (ScheduleStepDefaultAttempts);
	// o teto ScheduleStepMaxAttempts é validado antes de persistir. Retry é seguro
	// porque a criação da missão é idempotente: repetir não duplica.
	MaxAttempts int `json:"max_attempts,omitempty"`
	// Source é o valor de entrada de action.transform, já com variáveis
	// ({{nome}}). Target é o NOME da variável que o passo publica no escopo da
	// execução; nomes válidos são limitados e um valor grande demais é recusado em
	// vez de truncado em silêncio.
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
	// Items é a lista de logic.loop. Cada item roda UMA vez, com chave de
	// idempotência própria, e a lista é limitada por maxScheduleLoopItems: um
	// laço sem teto num motor que roda em tick é um gerador de missões.
	Items []string `json:"items,omitempty"`
	// Connector identifica o conector REGISTRADO, Operation o nome declarado na
	// configuração, Method e Path a chamada. Body aceita variáveis. Nada aqui
	// inventa acesso: sem conector configurado para a organização, o passo falha.
	Connector          string `json:"connector,omitempty"`
	ConnectorOperation string `json:"connector_operation,omitempty"`
	ConnectorMethod    string `json:"connector_method,omitempty"`
	ConnectorPath      string `json:"connector_path,omitempty"`
	Body               string `json:"body,omitempty"`
}

// ValidateScheduleSteps accepts an empty graph (legacy single-objective
// schedule) and otherwise rejects every malformed graph: unknown kinds, missing
// or duplicated identifiers, more than one trigger, unreachable steps and
// cycles. Validation is fail-closed so nothing partial is ever persisted.
func ValidateScheduleSteps(steps []ScheduleStep) error {
	if len(steps) == 0 {
		return nil
	}
	if len(steps) > maxScheduleFlowSteps {
		return fmt.Errorf("schedule flow accepts at most %d steps, got %d", maxScheduleFlowSteps, len(steps))
	}
	byID := make(map[string]ScheduleStep, len(steps))
	for _, step := range steps {
		id := strings.TrimSpace(step.ID)
		if id == "" {
			return errors.New("schedule step id is required")
		}
		if _, exists := byID[id]; exists {
			return fmt.Errorf("schedule step id %q is duplicated", id)
		}
		if err := validateScheduleStepShape(step); err != nil {
			return err
		}
		byID[id] = step
	}
	triggers := 0
	for _, step := range steps {
		if isScheduleTrigger(step.Kind) {
			triggers++
		}
	}
	if triggers != 1 {
		return fmt.Errorf("schedule flow requires exactly one trigger, got %d", triggers)
	}
	for _, step := range steps {
		for _, dependency := range step.DependsOn {
			dep := strings.TrimSpace(dependency)
			if dep == strings.TrimSpace(step.ID) {
				return fmt.Errorf("schedule step %q depends on itself", strings.TrimSpace(step.ID))
			}
			if _, exists := byID[dep]; !exists {
				return fmt.Errorf("schedule step %q depends on unknown step %q", strings.TrimSpace(step.ID), dep)
			}
		}
	}
	if _, err := orderScheduleSteps(steps); err != nil {
		return err
	}
	return nil
}

func validateScheduleStepShape(step ScheduleStep) error {
	id := strings.TrimSpace(step.ID)
	switch step.Kind {
	case ScheduleStepTriggerInterval, ScheduleStepTriggerWebhook:
		if len(step.DependsOn) != 0 {
			return fmt.Errorf("schedule trigger %q cannot depend on another step", id)
		}
		if strings.TrimSpace(step.Objective) != "" || strings.TrimSpace(step.Expect) != "" {
			return fmt.Errorf("schedule trigger %q cannot carry an objective or an expectation", id)
		}
	case ScheduleStepActionMission:
		if strings.TrimSpace(step.Objective) == "" {
			return fmt.Errorf("schedule action %q requires an objective", id)
		}
		if len(step.DependsOn) == 0 {
			return fmt.Errorf("schedule action %q requires at least one dependency", id)
		}
		if strings.TrimSpace(step.Expect) != "" {
			return fmt.Errorf("schedule action %q cannot carry an expectation", id)
		}
		if step.MaxAttempts < 0 || step.MaxAttempts > ScheduleStepMaxAttempts {
			return fmt.Errorf("schedule action %q accepts at most %d attempts, got %d", id, ScheduleStepMaxAttempts, step.MaxAttempts)
		}
	case ScheduleStepActionTransform:
		if len(step.DependsOn) == 0 {
			return fmt.Errorf("schedule transform %q requires at least one dependency", id)
		}
		if strings.TrimSpace(step.Source) == "" {
			return fmt.Errorf("schedule transform %q requires a source", id)
		}
		if !validScheduleVariableName(strings.TrimSpace(step.Target)) {
			return fmt.Errorf("schedule transform %q requires a valid target variable name", id)
		}
		if len(step.Source) > maxScheduleVariableSize {
			return fmt.Errorf("schedule transform %q source exceeds %d characters", id, maxScheduleVariableSize)
		}
		if strings.TrimSpace(step.Objective) != "" || strings.TrimSpace(step.Expect) != "" {
			return fmt.Errorf("schedule transform %q cannot carry an objective or an expectation", id)
		}
		if step.MaxAttempts != 0 {
			return fmt.Errorf("schedule transform %q cannot declare attempts", id)
		}
		if len(step.Items) != 0 {
			return fmt.Errorf("schedule transform %q cannot carry loop items", id)
		}
	case ScheduleStepLogicLoop:
		if len(step.DependsOn) == 0 {
			return fmt.Errorf("schedule loop %q requires at least one dependency", id)
		}
		if len(step.Items) == 0 {
			return fmt.Errorf("schedule loop %q requires at least one item", id)
		}
		if len(step.Items) > maxScheduleLoopItems {
			return fmt.Errorf("schedule loop %q accepts at most %d items, got %d", id, maxScheduleLoopItems, len(step.Items))
		}
		if strings.TrimSpace(step.Objective) == "" {
			return fmt.Errorf("schedule loop %q requires an objective template", id)
		}
		if strings.TrimSpace(step.Expect) != "" {
			return fmt.Errorf("schedule loop %q cannot carry an expectation", id)
		}
		if step.MaxAttempts != 0 {
			return fmt.Errorf("schedule loop %q cannot declare attempts", id)
		}
		for index, item := range step.Items {
			if strings.TrimSpace(item) == "" {
				return fmt.Errorf("schedule loop %q item %d is empty", id, index+1)
			}
			if len(item) > maxScheduleVariableSize {
				return fmt.Errorf("schedule loop %q item %d exceeds %d characters", id, index+1, maxScheduleVariableSize)
			}
		}
	case ScheduleStepApprovalRequest:
		if len(step.DependsOn) == 0 {
			return fmt.Errorf("schedule approval %q requires at least one dependency", id)
		}
		if strings.TrimSpace(step.Objective) == "" {
			return fmt.Errorf("schedule approval %q requires an objective describing the decision", id)
		}
		if strings.TrimSpace(step.Expect) != "" {
			return fmt.Errorf("schedule approval %q cannot carry an expectation", id)
		}
		if step.MaxAttempts != 0 {
			return fmt.Errorf("schedule approval %q cannot declare attempts", id)
		}
		if len(step.Items) != 0 {
			return fmt.Errorf("schedule approval %q cannot carry loop items", id)
		}
		if strings.TrimSpace(step.Source) != "" || strings.TrimSpace(step.Target) != "" {
			return fmt.Errorf("schedule approval %q cannot carry a transform source or target", id)
		}
	case ScheduleStepActionConnector:
		if len(step.DependsOn) == 0 {
			return fmt.Errorf("schedule connector %q requires at least one dependency", id)
		}
		if strings.TrimSpace(step.Connector) == "" {
			return fmt.Errorf("schedule connector %q requires a connector id", id)
		}
		if strings.TrimSpace(step.ConnectorOperation) == "" {
			return fmt.Errorf("schedule connector %q requires an operation name", id)
		}
		method := strings.ToUpper(strings.TrimSpace(step.ConnectorMethod))
		switch method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			return fmt.Errorf("schedule connector %q requires a supported HTTP method, got %q", id, step.ConnectorMethod)
		}
		if !strings.HasPrefix(strings.TrimSpace(step.ConnectorPath), "/") {
			return fmt.Errorf("schedule connector %q requires a path starting with /", id)
		}
		if len(step.Body) > maxScheduleConnectorBody {
			return fmt.Errorf("schedule connector %q body exceeds %d bytes", id, maxScheduleConnectorBody)
		}
		if strings.TrimSpace(step.Expect) != "" {
			return fmt.Errorf("schedule connector %q cannot carry an expectation", id)
		}
		if step.MaxAttempts < 0 || step.MaxAttempts > ScheduleStepMaxAttempts {
			return fmt.Errorf("schedule connector %q accepts at most %d attempts, got %d", id, ScheduleStepMaxAttempts, step.MaxAttempts)
		}
		if len(step.Items) != 0 {
			return fmt.Errorf("schedule connector %q cannot carry loop items", id)
		}
		if strings.TrimSpace(step.Objective) != "" {
			return fmt.Errorf("schedule connector %q cannot carry an objective", id)
		}
	case ScheduleStepConditionIf:
		if len(step.DependsOn) != 1 {
			return fmt.Errorf("schedule condition %q requires exactly one dependency, got %d", id, len(step.DependsOn))
		}
		if strings.TrimSpace(step.Objective) != "" {
			return fmt.Errorf("schedule condition %q cannot carry an objective", id)
		}
		switch strings.TrimSpace(step.Expect) {
		case ScheduleExpectSucceeded, ScheduleExpectFailed:
		default:
			return fmt.Errorf("schedule condition %q requires expect %q or %q", id, ScheduleExpectSucceeded, ScheduleExpectFailed)
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnknownScheduleStep, step.Kind)
	}
	return nil
}

func isScheduleTrigger(kind ScheduleStepKind) bool {
	return kind == ScheduleStepTriggerInterval || kind == ScheduleStepTriggerWebhook
}

// orderScheduleSteps returns a deterministic topological order of the graph and
// fails when the graph contains a cycle or a step unreachable from the trigger.
func orderScheduleSteps(steps []ScheduleStep) ([]ScheduleStep, error) {
	byID := make(map[string]ScheduleStep, len(steps))
	pending := make(map[string]int, len(steps))
	dependents := make(map[string][]string, len(steps))
	for _, step := range steps {
		id := strings.TrimSpace(step.ID)
		byID[id] = step
		pending[id] = 0
	}
	for _, step := range steps {
		id := strings.TrimSpace(step.ID)
		for _, dependency := range step.DependsOn {
			dep := strings.TrimSpace(dependency)
			if _, exists := byID[dep]; !exists {
				return nil, fmt.Errorf("schedule step %q depends on unknown step %q", id, dep)
			}
			pending[id]++
			dependents[dep] = append(dependents[dep], id)
		}
	}
	ready := make([]string, 0, len(steps))
	for id, remaining := range pending {
		if remaining == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	order := make([]ScheduleStep, 0, len(steps))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, byID[id])
		for _, dependent := range dependents[id] {
			pending[dependent]--
			if pending[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
		sort.Strings(ready)
	}
	if len(order) != len(steps) {
		return nil, errors.New("schedule flow must be acyclic and every step must depend on the flow trigger")
	}
	return order, nil
}

// ScheduleStepSucceeded/Skipped/Failed classificam o passo. "pending" existe para
// a espera por decisão humana: um passo aguardando aprovação não é sucesso nem
// falha, e tratar como qualquer um dos dois mentiria sobre o estado do fluxo.
const (
	ScheduleStepSucceeded = "succeeded"
	ScheduleStepSkipped   = "skipped"
	ScheduleStepFailed    = "failed"
	ScheduleStepPending   = "pending"
)

// ScheduleApproval é uma decisão humana sobre UM passo de UMA execução. A chave
// inclui a execução (RunKey) de propósito: uma aprovação concedida durante uma
// pausa NÃO autoriza a mesma etapa numa execução futura.
type ScheduleApproval struct {
	StepID    string    `json:"step_id"`
	RunKey    string    `json:"run_key"`
	Approved  bool      `json:"approved"`
	ActorID   string    `json:"actor_id,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	DecidedAt time.Time `json:"decided_at"`
}

// SchedulePendingApproval registra que a execução está PARADA esperando decisão.
type SchedulePendingApproval struct {
	StepID      string    `json:"step_id"`
	RunKey      string    `json:"run_key"`
	Objective   string    `json:"objective,omitempty"`
	RequestedAt time.Time `json:"requested_at"`
}

// maxScheduleApprovals limita o histórico de decisões e a lista de pendências.
const maxScheduleApprovals = 20

// ScheduleStepOutcome é o resultado de UM passo da execução.
type ScheduleStepOutcome struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	// Attempts é quantas tentativas o passo consumiu (retry por nó). 0 significa
	// "não se aplica" (gatilho/condição, que não tentam de novo).
	Attempts int `json:"attempts,omitempty"`
	// Children detalha as iterações de logic.loop: cada item é uma unidade de
	// evidência própria, e não um contador agregado.
	Children []ScheduleStepOutcome `json:"children,omitempty"`
}

// ScheduleRunReport é o resultado auditável de uma execução de fluxo. Ele é
// persistido junto do schedule (Schedule.LastRun) tanto em sucesso quanto em
// falha, para que uma execução parcial seja visível.
type ScheduleRunReport struct {
	FinishedAt time.Time             `json:"finished_at"`
	Missions   int                   `json:"missions"`
	Steps      []ScheduleStepOutcome `json:"steps,omitempty"`
	// RunKey identifica a execução que produziu este relatório. É o que permite
	// reprocessar depois SEM adivinhar: o reprocessamento reusa a mesma chave, e
	// assim os passos que já criaram missão continuam apontando para ela.
	RunKey string `json:"run_key,omitempty"`
	// DeadLetters são os passos que esgotaram as tentativas NESTA execução. O
	// relatório é quem sabe quais falharam; o gravador persiste no schedule.
	DeadLetters []ScheduleStepDeadLetter `json:"dead_letters,omitempty"`
	// FlowVersion é a versão do grafo que ESTA execução rodou. Sem ela, comparar
	// execuções de grafos diferentes seria comparação inválida.
	FlowVersion int `json:"flow_version,omitempty"`
	// PendingApprovals são os passos que PARARAM esta execução esperando decisão
	// humana. O gravador persiste no schedule para que a pausa sobreviva ao tick.
	PendingApprovals []SchedulePendingApproval `json:"pending_approvals,omitempty"`
}

// SkippedSteps devolve os identificadores ignorados na execução. Vazio significa
// que todo passo do grafo executou.
func (r ScheduleRunReport) SkippedSteps() []string {
	skipped := make([]string, 0)
	for _, step := range r.Steps {
		if step.Status == ScheduleStepSkipped {
			skipped = append(skipped, step.ID)
		}
	}
	sort.Strings(skipped)
	return skipped
}

// FailedStep devolve o identificador do passo que falhou, se houve.
func (r ScheduleRunReport) FailedStep() string {
	for _, step := range r.Steps {
		if step.Status == ScheduleStepFailed {
			return step.ID
		}
	}
	return ""
}

// ExecuteScheduleFlow runs a persisted schedule flow and returns how many
// missions it created. An empty graph keeps the legacy behaviour of a single
// mission carrying the schedule objective.
//
// Every mission created here runs with the baseline read-only capabilities that
// CreateMission applies by default: a stored flow can never grant an elevated
// scope, matching the autorun restriction enforced by the HTTP layer. The flow
// stops at the first action that cannot create its mission and reports that
// error so the caller can record the schedule failure.
func (r *Runtime) ExecuteScheduleFlow(ctx context.Context, schedule Schedule) (int, error) {
	report, err := r.ExecuteScheduleFlowReport(ctx, schedule)
	return report.Missions, err
}

// ExecuteScheduleFlowReport executa o grafo e devolve o resultado POR NÓ, mesmo
// quando falha — o relatório parcial é o que permite registrar quais passos
// rodaram e quais foram ignorados. Erro continua sendo erro: o relatório não
// transforma falha em sucesso.
func (r *Runtime) ExecuteScheduleFlowReport(ctx context.Context, schedule Schedule) (ScheduleRunReport, error) {
	return r.executeScheduleFlow(ctx, schedule, ScheduleFlowRunKey(schedule), nil, nil, nil)
}

// executeScheduleFlow é o núcleo compartilhado entre execução normal, execução
// manual e reprocessamento. Existe para as três não divergirem:
//   - runKey identifica a execução (reprocessamento reusa a chave anterior);
//   - seedOutcomes entra com o resultado já conhecido dos passos que NÃO serão
//     reexecutados, para que as dependências deles continuem satisfeitas;
//   - carryOver preserva no relatório novo o resultado dos passos pulados;
//   - execute limita quais passos rodam agora (nil = todos).
func (r *Runtime) executeScheduleFlow(ctx context.Context, schedule Schedule, runKey string, seedOutcomes map[string]bool, carryOver map[string]ScheduleStepOutcome, execute map[string]bool) (ScheduleRunReport, error) {
	report := ScheduleRunReport{Steps: make([]ScheduleStepOutcome, 0, len(schedule.Steps)), RunKey: runKey, FlowVersion: schedule.FlowVersion}
	if len(schedule.Steps) == 0 {
		if _, err := r.createScheduledMission(ctx, schedule, schedule.Objective); err != nil {
			report.FinishedAt = time.Now().UTC()
			report.Steps = append(report.Steps, ScheduleStepOutcome{
				ID:     "legacy-objective",
				Kind:   string(ScheduleStepActionMission),
				Status: ScheduleStepFailed,
				Reason: err.Error(),
			})
			return report, err
		}
		report.Missions = 1
		report.FinishedAt = time.Now().UTC()
		report.Steps = append(report.Steps, ScheduleStepOutcome{
			ID:     "legacy-objective",
			Kind:   string(ScheduleStepActionMission),
			Status: ScheduleStepSucceeded,
		})
		return report, nil
	}
	if err := ValidateScheduleSteps(schedule.Steps); err != nil {
		return report, err
	}
	order, err := orderScheduleSteps(schedule.Steps)
	if err != nil {
		return report, err
	}
	outcomes := make(map[string]bool, len(order))
	for id, result := range seedOutcomes {
		outcomes[id] = result
	}
	// Escopo de variáveis da execução: começa com os valores que o servidor
	// conhece (todos derivados do schedule autenticado) e cresce com os passos
	// action.transform e logic.loop.
	vars := scheduleBuiltinVariables(schedule, runKey)
	// pendingSteps distingue "concluiu sem sucesso" de "aguarda decisão humana":
	// só o primeiro satisfaz um ramo alternativo (condition.if com expect failed).
	pendingSteps := make(map[string]bool, len(order))
	for _, step := range order {
		id := strings.TrimSpace(step.ID)
		if execute != nil && !execute[id] {
			// Passo preservado: não roda de novo, mas o resultado anterior entra no
			// relatório e nas dependências (senão o dependente ficaria bloqueado).
			if previous, ok := carryOver[id]; ok {
				report.Steps = append(report.Steps, previous)
				outcomes[id] = previous.Status == ScheduleStepSucceeded
			} else {
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepSkipped,
					Reason: "passo não incluído neste reprocessamento",
				})
				outcomes[id] = false
			}
			continue
		}
		if !scheduleStepDependenciesReady(step, outcomes, pendingSteps) {
			outcomes[id] = false
			// Um passo barrado por dependência PENDENTE também fica pendente: não
			// concluiu nada, e o ramo alternativo não pode tratá-lo como falha.
			if stepHasPendingDependency(step, pendingSteps) {
				pendingSteps[id] = true
			}
			report.Steps = append(report.Steps, ScheduleStepOutcome{
				ID:     id,
				Kind:   string(step.Kind),
				Status: ScheduleStepSkipped,
				Reason: "dependência não teve sucesso",
			})
			continue
		}
		switch step.Kind {
		case ScheduleStepTriggerInterval, ScheduleStepTriggerWebhook:
			outcomes[id] = true
			report.Steps = append(report.Steps, ScheduleStepOutcome{ID: id, Kind: string(step.Kind), Status: ScheduleStepSucceeded})
		case ScheduleStepActionMission:
			// O objetivo também usa o escopo de variáveis: um objetivo com
			// {{nome}} não resolvido seria missão pedindo algo que ninguém definiu.
			objective, expandErr := expandScheduleTemplate(step.Objective, vars)
			if expandErr != nil {
				outcomes[id] = false
				report.FinishedAt = time.Now().UTC()
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepFailed,
					Reason: expandErr.Error(),
				})
				report.DeadLetters = append(report.DeadLetters, ScheduleStepDeadLetter{
					StepID:   id,
					Kind:     string(step.Kind),
					Reason:   expandErr.Error(),
					FailedAt: report.FinishedAt,
				})
				return report, fmt.Errorf("schedule step %q: %w", id, expandErr)
			}
			// Retry por nó: a chave de idempotência é a MESMA em todas as
			// tentativas, então uma repetição que encontra a missão já criada não
			// duplica nem conta como missão nova.
			created := false
			attempts, err := runWithRetry(scheduleStepAttempts(step), func() error {
				_, wasCreated, createErr := r.createScheduledMissionIdempotent(ctx, schedule, strings.TrimSpace(objective), scheduleStepIdempotencyKey(runKey, id))
				if createErr == nil {
					created = wasCreated
				}
				return createErr
			})
			if err != nil {
				outcomes[id] = false
				report.FinishedAt = time.Now().UTC()
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:       id,
					Kind:     string(step.Kind),
					Status:   ScheduleStepFailed,
					Reason:   err.Error(),
					Attempts: attempts,
				})
				// DLQ por nó: o passo que esgotou as tentativas é registrado com o
				// motivo final, para não desaparecer no próximo tick.
				report.DeadLetters = append(report.DeadLetters, ScheduleStepDeadLetter{
					StepID:    id,
					Kind:      string(step.Kind),
					Reason:    err.Error(),
					Attempts:  attempts,
					Objective: strings.TrimSpace(step.Objective),
					FailedAt:  report.FinishedAt,
				})
				return report, fmt.Errorf("schedule step %q: %w", id, err)
			}
			outcome := ScheduleStepOutcome{ID: id, Kind: string(step.Kind), Status: ScheduleStepSucceeded, Attempts: attempts}
			if created {
				report.Missions++
			} else {
				// Repetição idempotente: nada foi criado agora e o relatório diz
				// isso, em vez de contar uma missão que já existia.
				outcome.Reason = "missão já existia para esta execução (idempotente)"
			}
			outcomes[id] = true
			report.Steps = append(report.Steps, outcome)
		case ScheduleStepActionTransform:
			// Puro e determinístico: expande o template e publica a variável. Falha
			// de variável desconhecida ou de limite é ERRO, não valor vazio.
			expanded, expandErr := expandScheduleTemplate(step.Source, vars)
			if expandErr == nil {
				expandErr = setScheduleVariable(vars, strings.TrimSpace(step.Target), expanded)
			}
			if expandErr != nil {
				outcomes[id] = false
				report.FinishedAt = time.Now().UTC()
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepFailed,
					Reason: expandErr.Error(),
				})
				report.DeadLetters = append(report.DeadLetters, ScheduleStepDeadLetter{
					StepID:   id,
					Kind:     string(step.Kind),
					Reason:   expandErr.Error(),
					FailedAt: report.FinishedAt,
				})
				return report, fmt.Errorf("schedule step %q: %w", id, expandErr)
			}
			outcomes[id] = true
			report.Steps = append(report.Steps, ScheduleStepOutcome{
				ID:     id,
				Kind:   string(step.Kind),
				Status: ScheduleStepSucceeded,
				Reason: fmt.Sprintf("variável %q publicada", strings.TrimSpace(step.Target)),
			})
		case ScheduleStepLogicLoop:
			// Laço LIMITADO: uma missão por item, com chave de idempotência que
			// inclui o índice. O teto por execução é conferido ANTES de criar
			// qualquer missão, para não deixar trabalho pela metade.
			if report.Missions+len(step.Items) > maxScheduleMissionsPerRun {
				outcomes[id] = false
				report.FinishedAt = time.Now().UTC()
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepFailed,
					Reason: fmt.Sprintf("o laço criaria %d missões e excede o teto de %d por execução", report.Missions+len(step.Items), maxScheduleMissionsPerRun),
				})
				return report, fmt.Errorf("schedule step %q: mission budget exceeded", id)
			}
			loopOutcome := ScheduleStepOutcome{ID: id, Kind: string(step.Kind), Status: ScheduleStepSucceeded}
			for index, item := range step.Items {
				iterationVars := make(map[string]string, len(vars)+2)
				for name, value := range vars {
					iterationVars[name] = value
				}
				iterationVars["item"] = item
				iterationVars["index"] = strconv.Itoa(index)
				objective, expandErr := expandScheduleTemplate(step.Objective, iterationVars)
				child := ScheduleStepOutcome{
					ID:     fmt.Sprintf("%s[%d]", id, index),
					Kind:   string(step.Kind),
					Status: ScheduleStepSucceeded,
				}
				if expandErr != nil {
					child.Status = ScheduleStepFailed
					child.Reason = expandErr.Error()
					loopOutcome.Children = append(loopOutcome.Children, child)
					loopOutcome.Status = ScheduleStepFailed
					loopOutcome.Reason = expandErr.Error()
					outcomes[id] = false
					report.FinishedAt = time.Now().UTC()
					report.Steps = append(report.Steps, loopOutcome)
					return report, fmt.Errorf("schedule step %q: %w", id, expandErr)
				}
				key := fmt.Sprintf("%s/%s#%d", scheduleStepIdempotencyKey(runKey, id), item, index)
				_, wasCreated, createErr := r.createScheduledMissionIdempotent(ctx, schedule, strings.TrimSpace(objective), key)
				if createErr != nil {
					child.Status = ScheduleStepFailed
					child.Reason = createErr.Error()
					loopOutcome.Children = append(loopOutcome.Children, child)
					loopOutcome.Status = ScheduleStepFailed
					loopOutcome.Reason = createErr.Error()
					outcomes[id] = false
					report.FinishedAt = time.Now().UTC()
					report.Steps = append(report.Steps, loopOutcome)
					report.DeadLetters = append(report.DeadLetters, ScheduleStepDeadLetter{
						StepID:    id,
						Kind:      string(step.Kind),
						Reason:    createErr.Error(),
						Objective: strings.TrimSpace(objective),
						FailedAt:  report.FinishedAt,
					})
					return report, fmt.Errorf("schedule step %q: %w", id, createErr)
				}
				if wasCreated {
					report.Missions++
					child.Reason = "missão criada"
				} else {
					child.Reason = "missão já existia (idempotente)"
				}
				loopOutcome.Children = append(loopOutcome.Children, child)
			}
			loopOutcome.Reason = fmt.Sprintf("%d itens processados", len(step.Items))
			outcomes[id] = true
			report.Steps = append(report.Steps, loopOutcome)
		case ScheduleStepApprovalRequest:
			// Espera por decisão humana. Sem decisão o passo fica PENDENTE e os
			// dependentes não rodam: o fluxo para, em vez de adivinhar. A decisão é
			// buscada por (passo, execução), então uma aprovação não vaza para uma
			// execução futura.
			objective, expandErr := expandScheduleTemplate(step.Objective, vars)
			if expandErr != nil {
				outcomes[id] = false
				report.FinishedAt = time.Now().UTC()
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepFailed,
					Reason: expandErr.Error(),
				})
				return report, fmt.Errorf("schedule step %q: %w", id, expandErr)
			}
			approval, decided := scheduleApprovalFor(schedule, id, runKey)
			switch {
			case !decided:
				outcomes[id] = false
				pendingSteps[id] = true
				report.PendingApprovals = append(report.PendingApprovals, SchedulePendingApproval{
					StepID:      id,
					RunKey:      runKey,
					Objective:   strings.TrimSpace(objective),
					RequestedAt: time.Now().UTC(),
				})
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepPending,
					Reason: "aguardando aprovação humana para esta execução",
				})
			case approval.Approved:
				outcomes[id] = true
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepSucceeded,
					Reason: "aprovado" + approvalActorSuffix(approval),
				})
			default:
				// Negativa é decisão humana, não falha técnica: não entra na fila
				// morta, mas também não libera o dependente.
				outcomes[id] = false
				reason := "aprovação negada" + approvalActorSuffix(approval)
				if strings.TrimSpace(approval.Reason) != "" {
					reason = reason + ": " + strings.TrimSpace(approval.Reason)
				}
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepSkipped,
					Reason: reason,
				})
			}
		case ScheduleStepActionConnector:
			// Conector REAL: a chamada passa por CallForOrganization, com
			// credencial do tenant, política de egress e escopo de organização. Sem
			// conector configurado o passo FALHA — nunca é silenciosamente pulado.
			method := strings.ToUpper(strings.TrimSpace(step.ConnectorMethod))
			body, expandErr := expandScheduleTemplate(step.Body, vars)
			if expandErr != nil {
				outcomes[id] = false
				report.FinishedAt = time.Now().UTC()
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepFailed,
					Reason: expandErr.Error(),
				})
				return report, fmt.Errorf("schedule step %q: %w", id, expandErr)
			}
			if scheduleConnectorWriteMethods[method] {
				// Método que ESCREVE exige decisão humana aprovada para esta
				// execução. Sem isso, o passo fica pendente como o nó de aprovação:
				// o motor não dispara efeito externo por conta própria.
				approval, decided := scheduleApprovalFor(schedule, id, runKey)
				if !decided {
					outcomes[id] = false
					pendingSteps[id] = true
					report.PendingApprovals = append(report.PendingApprovals, SchedulePendingApproval{
						StepID:      id,
						RunKey:      runKey,
						Objective:   fmt.Sprintf("%s %s (conector %s)", method, strings.TrimSpace(step.ConnectorPath), strings.TrimSpace(step.Connector)),
						RequestedAt: time.Now().UTC(),
					})
					report.Steps = append(report.Steps, ScheduleStepOutcome{
						ID:     id,
						Kind:   string(step.Kind),
						Status: ScheduleStepPending,
						Reason: fmt.Sprintf("método %s altera estado e exige aprovação humana para esta execução", method),
					})
					continue
				}
				if !approval.Approved {
					outcomes[id] = false
					reason := fmt.Sprintf("%s não autorizado%s", method, approvalActorSuffix(approval))
					if strings.TrimSpace(approval.Reason) != "" {
						reason = reason + ": " + strings.TrimSpace(approval.Reason)
					}
					report.Steps = append(report.Steps, ScheduleStepOutcome{
						ID:     id,
						Kind:   string(step.Kind),
						Status: ScheduleStepSkipped,
						Reason: reason,
					})
					continue
				}
			}
			if r.connectors == nil {
				outcomes[id] = false
				report.FinishedAt = time.Now().UTC()
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepFailed,
					Reason: "nenhum gerenciador de conectores configurado no runtime",
				})
				return report, fmt.Errorf("schedule step %q: connector manager is not configured", id)
			}
			scope := strings.TrimSpace(schedule.OrganizationID)
			if scope == "" {
				scope = strings.TrimSpace(r.organizationScope)
			}
			status, response, callErr := r.connectors.CallForOrganization(
				ctx, scope, strings.TrimSpace(step.Connector), strings.TrimSpace(step.ConnectorOperation),
				method, strings.TrimSpace(step.ConnectorPath), []byte(body),
			)
			if callErr != nil {
				outcomes[id] = false
				report.FinishedAt = time.Now().UTC()
				report.Steps = append(report.Steps, ScheduleStepOutcome{
					ID:     id,
					Kind:   string(step.Kind),
					Status: ScheduleStepFailed,
					Reason: callErr.Error(),
				})
				report.DeadLetters = append(report.DeadLetters, ScheduleStepDeadLetter{
					StepID:   id,
					Kind:     string(step.Kind),
					Reason:   callErr.Error(),
					FailedAt: report.FinishedAt,
				})
				return report, fmt.Errorf("schedule step %q: %w", id, callErr)
			}
			outcome := ScheduleStepOutcome{
				ID:     id,
				Kind:   string(step.Kind),
				Status: ScheduleStepSucceeded,
				Reason: fmt.Sprintf("%s HTTP %d", method, status),
			}
			if status >= http.StatusBadRequest {
				// Resposta de erro do outro lado não é sucesso do nosso nó.
				outcome.Status = ScheduleStepFailed
				outcome.Reason = fmt.Sprintf("%s HTTP %d", method, status)
				outcomes[id] = false
				report.FinishedAt = time.Now().UTC()
				report.Steps = append(report.Steps, outcome)
				report.DeadLetters = append(report.DeadLetters, ScheduleStepDeadLetter{
					StepID:   id,
					Kind:     string(step.Kind),
					Reason:   outcome.Reason,
					FailedAt: report.FinishedAt,
				})
				return report, fmt.Errorf("schedule step %q: connector returned HTTP %d", id, status)
			}
			if target := strings.TrimSpace(step.Target); target != "" {
				value := response
				if len(value) > maxScheduleConnectorBodyInVars {
					value = value[:maxScheduleConnectorBodyInVars]
				}
				if setErr := setScheduleVariable(vars, target, value); setErr != nil {
					outcomEsReason := setErr.Error()
					outcome.Status = ScheduleStepFailed
					outcome.Reason = outcomEsReason
					outcomes[id] = false
					report.FinishedAt = time.Now().UTC()
					report.Steps = append(report.Steps, outcome)
					return report, fmt.Errorf("schedule step %q: %w", id, setErr)
				}
				outcome.Reason = outcome.Reason + fmt.Sprintf(" · resposta publicada em %q", target)
			}
			outcomes[id] = true
			report.Steps = append(report.Steps, outcome)
		case ScheduleStepConditionIf:
			matched := scheduleConditionMatches(step, outcomes)
			outcomes[id] = matched
			outcome := ScheduleStepOutcome{ID: id, Kind: string(step.Kind), Status: ScheduleStepSucceeded}
			if !matched {
				outcome.Status = ScheduleStepSkipped
				outcome.Reason = "condição não satisfeita pelos dependentes"
			}
			report.Steps = append(report.Steps, outcome)
		default:
			return report, fmt.Errorf("%w: %s", ErrUnknownScheduleStep, step.Kind)
		}
	}
	report.FinishedAt = time.Now().UTC()
	return report, nil
}

// ScheduleStepDeadLetter é a entrada de DLQ de UM passo que esgotou as
// tentativas. Fica no relatório (motor) e é persistida no schedule (gravador),
// para que uma falha não desapareça no próximo tick e o operador possa
// reprocessar exatamente aquele passo.
type ScheduleStepDeadLetter struct {
	StepID    string    `json:"step_id"`
	Kind      string    `json:"kind"`
	Reason    string    `json:"reason"`
	Attempts  int       `json:"attempts,omitempty"`
	Objective string    `json:"objective,omitempty"`
	FailedAt  time.Time `json:"failed_at"`
}

// maxScheduleDeadLetters limita a DLQ por schedule: sem teto, um fluxo que falha
// a cada tick cresceria sem fim no arquivo de estado.
const maxScheduleDeadLetters = 20

// mergeScheduleDeadLetters acrescenta as novas falhas e REMOVE as entradas dos
// passos que tiveram sucesso nesta execução: um passo recuperado não pode
// continuar listado como morto.
func mergeScheduleDeadLetters(existing, added []ScheduleStepDeadLetter, succeeded map[string]bool) []ScheduleStepDeadLetter {
	// Mais recentes primeiro: a última falha do passo é a que importa para agir.
	merged := make([]ScheduleStepDeadLetter, 0, len(existing)+len(added))
	for _, entry := range added {
		if succeeded[entry.StepID] {
			continue
		}
		replaced := false
		for i, current := range merged {
			if current.StepID == entry.StepID {
				merged[i] = entry
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, entry)
		}
	}
	for _, entry := range existing {
		// Passo recuperado sai da DLQ; passo ainda falho mantém a entrada antiga
		// somente se não houver entrada nova mais recente para ele.
		if succeeded[entry.StepID] {
			continue
		}
		duplicate := false
		for _, current := range merged {
			if current.StepID == entry.StepID {
				duplicate = true
				break
			}
		}
		if !duplicate {
			merged = append(merged, entry)
		}
	}
	if len(merged) > maxScheduleDeadLetters {
		merged = merged[:maxScheduleDeadLetters]
	}
	return merged
}

// scheduleVariablePattern casa {{nome}} nos templates de passo. O conjunto de
// caracteres é restrito para que o template não vire uma linguagem de expressão.
var scheduleVariablePattern = regexp.MustCompile(`\{\{([a-zA-Z0-9_.-]{1,64})\}\}`)

// validScheduleVariableName aceita apenas nomes simples: sem espaço, sem barra,
// sem cifrão.
func validScheduleVariableName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '_' || char == '.' || char == '-' {
			continue
		}
		return false
	}
	return true
}

// expandScheduleTemplate substitui {{nome}} pelo valor da variável. Variável
// DESCONHECIDA é erro, não string vazia: um objetivo de missão que perde o valor
// por causa de um nome errado seria pior do que uma falha explícita.
func expandScheduleTemplate(text string, vars map[string]string) (string, error) {
	var failure error
	expanded := scheduleVariablePattern.ReplaceAllStringFunc(text, func(match string) string {
		name := strings.TrimSpace(match[2 : len(match)-2])
		value, ok := vars[name]
		if !ok {
			if failure == nil {
				failure = fmt.Errorf("variável %q não definida no escopo da execução", name)
			}
			return match
		}
		return value
	})
	if failure != nil {
		return "", failure
	}
	return expanded, nil
}

// setScheduleVariable publica um valor no escopo respeitando os limites: nome
// inválido, valor grande demais ou escopo cheio são ERROS, nunca truncamento
// silencioso.
func setScheduleVariable(vars map[string]string, name, value string) error {
	if !validScheduleVariableName(name) {
		return fmt.Errorf("nome de variável inválido: %q", name)
	}
	if len(value) > maxScheduleVariableSize {
		return fmt.Errorf("variável %q excede %d caracteres", name, maxScheduleVariableSize)
	}
	if _, exists := vars[name]; !exists && len(vars) >= maxScheduleVariables {
		return fmt.Errorf("escopo de variáveis excede %d entradas", maxScheduleVariables)
	}
	vars[name] = value
	return nil
}

// scheduleBuiltinVariables são os valores que o servidor conhece sem depender do
// conteúdo dos passos. Tudo aqui vem do schedule autenticado, nunca de texto
// externo não confiável.
func scheduleBuiltinVariables(schedule Schedule, runKey string) map[string]string {
	return map[string]string{
		"schedule.id":        schedule.ID,
		"schedule.objective": schedule.Objective,
		"run.key":            runKey,
	}
}

// approvalActorSuffix mostra QUEM decidiu. A trilha importa: uma aprovação sem
// identidade de quem decidiu não serve de auditoria.
func approvalActorSuffix(approval ScheduleApproval) string {
	actor := strings.TrimSpace(approval.ActorID)
	if actor == "" {
		return ""
	}
	return " por " + actor
}

// scheduleApprovalFor procura a decisão humana de UM passo em UMA execução. A
// busca é por (passo, execução) de propósito: aprovação de uma execução não
// autoriza a mesma etapa numa execução futura.
func scheduleApprovalFor(schedule Schedule, stepID, runKey string) (ScheduleApproval, bool) {
	stepID = strings.TrimSpace(stepID)
	for index := len(schedule.Approvals) - 1; index >= 0; index-- {
		approval := schedule.Approvals[index]
		if strings.TrimSpace(approval.StepID) == stepID && strings.TrimSpace(approval.RunKey) == strings.TrimSpace(runKey) {
			return approval, true
		}
	}
	return ScheduleApproval{}, false
}

// mergeScheduleApprovals acrescenta decisões novas (mais recentes primeiro) e
// aplica o teto, para o histórico não crescer sem fim.
func mergeScheduleApprovals(existing, added []ScheduleApproval) []ScheduleApproval {
	merged := make([]ScheduleApproval, 0, len(existing)+len(added))
	for _, entry := range added {
		replaced := false
		for i, current := range merged {
			if current.StepID == entry.StepID && current.RunKey == entry.RunKey {
				merged[i] = entry
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, entry)
		}
	}
	for _, entry := range existing {
		duplicate := false
		for _, current := range merged {
			if current.StepID == entry.StepID && current.RunKey == entry.RunKey {
				duplicate = true
				break
			}
		}
		if !duplicate {
			merged = append(merged, entry)
		}
	}
	if len(merged) > maxScheduleApprovals {
		merged = merged[:maxScheduleApprovals]
	}
	return merged
}

// scheduleStepsEqual compara grafos por conteúdo. A ordem importa: o grafo é
// executado na ordem de dependência declarada, e reordenar os passos é mudança de
// fluxo.
func scheduleStepsEqual(a, b []ScheduleStep) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID ||
			a[i].Kind != b[i].Kind ||
			a[i].Objective != b[i].Objective ||
			a[i].Expect != b[i].Expect ||
			a[i].MaxAttempts != b[i].MaxAttempts ||
			a[i].Source != b[i].Source ||
			a[i].Target != b[i].Target ||
			len(a[i].Items) != len(b[i].Items) ||
			len(a[i].DependsOn) != len(b[i].DependsOn) {
			return false
		}
		for j := range a[i].Items {
			if a[i].Items[j] != b[i].Items[j] {
				return false
			}
		}
		for j := range a[i].DependsOn {
			if a[i].DependsOn[j] != b[i].DependsOn[j] {
				return false
			}
		}
	}
	return true
}

// ErrScheduleReprocessUnavailable recusa o reprocessamento quando não há execução
// anterior registrada: sem a chave da execução não é possível reusar a mesma
// identidade, e adivinhar criaria missão duplicada.
var ErrScheduleReprocessUnavailable = errors.New("schedule has no previous run to reprocess")

// ReprocessScheduleFlow reexecuta SOMENTE os passos que ficaram ignorados ou
// falharam, reusando a chave da execução anterior. Os passos que já tiveram
// sucesso são preservados com o resultado original e não criam missão nova — a
// idempotência da criação é o que garante isso, mesmo se um deles for reexecutado
// depois de ter criado a missão.
func (r *Runtime) ReprocessScheduleFlow(ctx context.Context, schedule Schedule) (ScheduleRunReport, error) {
	previous := schedule.LastRun
	if previous == nil || strings.TrimSpace(previous.RunKey) == "" {
		return ScheduleRunReport{}, ErrScheduleReprocessUnavailable
	}
	if len(schedule.Steps) == 0 {
		// Fluxo legado (uma missão com o objetivo): reprocessar é reexecutar o
		// objetivo, e a linha de base é o próprio passo sintético.
		if _, err := r.createScheduledMission(ctx, schedule, schedule.Objective); err != nil {
			return ScheduleRunReport{
				RunKey:     previous.RunKey,
				FinishedAt: time.Now().UTC(),
				Steps: []ScheduleStepOutcome{{
					ID:     "legacy-objective",
					Kind:   string(ScheduleStepActionMission),
					Status: ScheduleStepFailed,
					Reason: err.Error(),
				}},
			}, err
		}
		return ScheduleRunReport{
			RunKey:     previous.RunKey,
			FinishedAt: time.Now().UTC(),
			Missions:   1,
			Steps: []ScheduleStepOutcome{{
				ID:       "legacy-objective",
				Kind:     string(ScheduleStepActionMission),
				Status:   ScheduleStepSucceeded,
				Attempts: 1,
			}},
		}, nil
	}

	execute := make(map[string]bool, len(schedule.Steps))
	seed := make(map[string]bool, len(schedule.Steps))
	carry := make(map[string]ScheduleStepOutcome, len(schedule.Steps))
	for _, outcome := range previous.Steps {
		switch outcome.Status {
		case ScheduleStepSucceeded:
			// Preservado: não roda de novo e continua satisfazendo dependentes.
			carry[outcome.ID] = outcome
			seed[outcome.ID] = true
		default:
			// Skipped ou failed: é exatamente o que o reprocessamento existe para
			// tentar de novo.
			execute[outcome.ID] = true
			seed[outcome.ID] = false
		}
	}
	// Passo NOVO no grafo (adicionado depois da execução anterior) também roda:
	// não tem resultado anterior para preservar.
	for _, step := range schedule.Steps {
		id := strings.TrimSpace(step.ID)
		if _, known := carry[id]; known {
			continue
		}
		if _, known := execute[id]; known {
			continue
		}
		execute[id] = true
	}
	return r.executeScheduleFlow(ctx, schedule, previous.RunKey, seed, carry, execute)
}

// scheduleDependenciesSatisfied reports whether every dependency already ran and
// succeeded. A skipped dependency therefore blocks its dependents instead of
// letting them run on a partial flow.
func scheduleDependenciesSatisfied(step ScheduleStep, outcomes map[string]bool) bool {
	for _, dependency := range step.DependsOn {
		outcome, ran := outcomes[strings.TrimSpace(dependency)]
		if !ran || !outcome {
			return false
		}
	}
	return true
}

// stepHasPendingDependency informa se algum dependente ainda aguarda decisão
// humana. Nesse caso o passo não "concluiu sem sucesso": ele nem começou.
func stepHasPendingDependency(step ScheduleStep, pending map[string]bool) bool {
	for _, dependency := range step.DependsOn {
		if pending[strings.TrimSpace(dependency)] {
			return true
		}
	}
	return false
}

// scheduleStepDependenciesReady decide se um passo pode ser AVALIADO agora.
//
// O que muda frente a scheduleDependenciesSatisfied é o ramo alternativo: um
// condition.if que espera FALHA observa uma dependência que CONCLUIU sem sucesso.
// Exigir sucesso dela incondicionalmente — como era antes — tornava o ramo
// alternativo inalcançável, porque a checagem barrava o passo antes de a condição
// ser avaliada.
//
// O conjunto `pending` existe porque "não teve sucesso" tem DUAS causas distintas:
// concluiu sem sucesso (o ramo alternativo deve rodar) e aguarda decisão humana (o
// ramo alternativo NÃO pode rodar: ninguém decidiu nada ainda).
func scheduleStepDependenciesReady(step ScheduleStep, outcomes map[string]bool, pending map[string]bool) bool {
	if step.Kind == ScheduleStepConditionIf && strings.TrimSpace(step.Expect) == ScheduleExpectFailed {
		if len(step.DependsOn) != 1 {
			return false
		}
		dependency := strings.TrimSpace(step.DependsOn[0])
		if pending[dependency] {
			return false
		}
		outcome, ran := outcomes[dependency]
		return ran && !outcome
	}
	return scheduleDependenciesSatisfied(step, outcomes)
}

// scheduleConditionMatches evaluates a condition.if step against the recorded
// outcome of its single dependency.
func scheduleConditionMatches(step ScheduleStep, outcomes map[string]bool) bool {
	if len(step.DependsOn) != 1 {
		return false
	}
	outcome, ran := outcomes[strings.TrimSpace(step.DependsOn[0])]
	if !ran {
		return false
	}
	if strings.TrimSpace(step.Expect) == ScheduleExpectFailed {
		return !outcome
	}
	return outcome
}

func (r *Runtime) createScheduledMission(ctx context.Context, schedule Schedule, objective string) (Mission, error) {
	return r.CreateMission(ctx, CreateMissionRequest{
		Objective:      objective,
		Model:          schedule.Model,
		Workspace:      schedule.Workspace,
		ProjectID:      schedule.ProjectID,
		OrganizationID: schedule.OrganizationID,
		AutoRun:        true,
	})
}

// ScheduleFlowRunKey identifica UMA execução do fluxo. Ele é estável durante a
// execução (o NextRunAt só é reescrito depois, ao registrar o resultado) e muda
// entre execuções — é o que permite repetir um passo com segurança: a chave de
// idempotência do passo é derivada daqui, então uma repetição DENTRO da mesma
// execução aponta para a mesma missão em vez de criar outra.
func ScheduleFlowRunKey(schedule Schedule) string {
	return strings.TrimSpace(schedule.ID) + "@" + schedule.NextRunAt.UTC().Format(time.RFC3339Nano)
}

// scheduleStepIdempotencyKey isola cada passo dentro da execução.
func scheduleStepIdempotencyKey(runKey, stepID string) string {
	return runKey + "/" + strings.TrimSpace(stepID)
}

// createScheduledMissionIdempotent cria a missão do passo sem duplicar quando a
// mesma execução rodar de novo. Devolve created=false quando a missão já existia,
// para o relatório não afirmar que criou o que não criou.
func (r *Runtime) createScheduledMissionIdempotent(ctx context.Context, schedule Schedule, objective, key string) (Mission, bool, error) {
	if existing, err := r.getMission(idempotentMissionID(schedule.OrganizationID, key)); err == nil {
		if missionScopeMatches(existing, schedule.OrganizationID, r.organizationScope) {
			return existing, false, nil
		}
	}
	mission, err := r.CreateMission(ctx, CreateMissionRequest{
		Objective:      objective,
		Model:          schedule.Model,
		Workspace:      schedule.Workspace,
		ProjectID:      schedule.ProjectID,
		OrganizationID: schedule.OrganizationID,
		AutoRun:        true,
		IdempotencyKey: key,
	})
	if err != nil {
		return Mission{}, false, err
	}
	return mission, true, nil
}
