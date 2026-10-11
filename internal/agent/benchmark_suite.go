package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Estágio 21 — benchmark comparável, feito do jeito honesto.
//
// O estágio pede um relatório de benchmark com metodologia, tarefas e métricas. A parte que
// NÃO se pode inventar é a comparação: executar Manus, Claude Code, Codex, Devin, Cursor,
// Lovable, n8n, Dify, CrewAI ou LangGraph exigiria contas, credenciais e custo que este
// ambiente não tem — e afirmar superioridade sem executá-los seria exatamente o que o
// estágio proíbe. Então esta suíte mede o que é medível aqui:
//
//   - **tarefas herméticas de ponta a ponta do PRÓPRIO Hades**, cada uma com critério de
//     sucesso objetivo;
//   - por tarefa: se concluiu, quantos **passos** (chamadas de API), **duração real**,
//     quantas **aprovações humanas** exigiu, **artefatos** gerados, se houve **prova de
//     isolamento** e se a **retomada/recuperação** foi exercitada;
//   - **comparação externa: NÃO EXECUTADA**, com o motivo declarado — nunca "somos melhores".
//
// Isso dá números reproduzíveis para acompanhar regressão de desempenho e de capacidade, e
// mantém a reivindicação competitiva exatamente onde ela deve estar: não feita.

// BenchmarkTaskResult é o resultado medido de UMA tarefa.
type BenchmarkTaskResult struct {
	Name            string   `json:"name"`
	Completed       bool     `json:"completed"`
	Steps           int      `json:"steps"`
	DurationMillis  int64    `json:"duration_millis"`
	HumanApprovals  int      `json:"human_approvals"`
	Artifacts       []string `json:"artifacts,omitempty"`
	IsolationProven bool     `json:"isolation_proven"`
	RecoveryProven  bool     `json:"recovery_proven"`
	Detail          string   `json:"detail"`
}

// BenchmarkSuiteReport é o relatório da suíte.
type BenchmarkSuiteReport struct {
	Suite               string                `json:"suite"`
	MeasuredAt          time.Time             `json:"measured_at"`
	Environment         string                `json:"environment"`
	Methodology         string                `json:"methodology"`
	Tasks               []BenchmarkTaskResult `json:"tasks"`
	CompletedTasks      int                   `json:"completed_tasks"`
	TotalTasks          int                   `json:"total_tasks"`
	TotalDurationMillis int64                 `json:"total_duration_millis"`
	ExternalComparison  string                `json:"external_comparison"`
	ExternalReason      string                `json:"external_reason"`
	Verdict             string                `json:"verdict"`
}

const benchmarkMethodology = "Tarefas herméticas de ponta a ponta executadas no próprio Hades, cada uma com " +
	"critério de sucesso objetivo; mede-se conclusão, passos de API, duração real, aprovações humanas exigidas, " +
	"artefatos gerados, prova de isolamento entre organizações e prova de retomada/recuperação. Ambiente declarado " +
	"no relatório; nada é estimado."

// RunBenchmarkSuite executa a suíte de tarefas e devolve o relatório MEDIDO.
func (r *Runtime) RunBenchmarkSuite(now time.Time) BenchmarkSuiteReport {
	report := BenchmarkSuiteReport{
		Suite:              "hades-hermetic-tasks",
		MeasuredAt:         now,
		Environment:        fmt.Sprintf("%s/%s · go %s", runtime.GOOS, runtime.GOARCH, runtime.Version()),
		Methodology:        benchmarkMethodology,
		ExternalComparison: "NAO_EXECUTADA",
		ExternalReason: "executar Manus, Claude Code, Codex, Devin, Cursor, Lovable, n8n, Dify, CrewAI ou LangGraph " +
			"exigiria contas, credenciais e custo que este ambiente não possui; sem executá-los, qualquer comparação " +
			"seria invenção, então NÃO se afirma superioridade",
		Verdict: "MEDIDO",
	}
	for _, task := range []func(time.Time) BenchmarkTaskResult{
		benchmarkBusinessJourney,
		benchmarkIdempotentOrder,
		benchmarkInboxSLAAndHandoff,
		benchmarkOrganizationIsolation,
		benchmarkRestartRecovery,
	} {
		report.Tasks = append(report.Tasks, task(now))
	}
	for _, task := range report.Tasks {
		if task.Completed {
			report.CompletedTasks++
		} else {
			report.Verdict = "PARCIAL"
		}
		report.TotalDurationMillis += task.DurationMillis
	}
	report.TotalTasks = len(report.Tasks)
	return report
}

// benchmarkRuntime cria um runtime isolado para uma tarefa, com raiz própria.
func benchmarkRuntime(taskName string) (*Runtime, string, error) {
	root, err := os.MkdirTemp("", "hades-bench-")
	if err != nil {
		return nil, "", err
	}
	runtimeInstance, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot:      filepath.Join(root, "data-"+sanitizeIDComponent(taskName)),
	})
	if err != nil {
		_ = os.RemoveAll(root)
		return nil, "", err
	}
	return runtimeInstance, root, nil
}

// benchmarkBusinessJourney mede a certificação da jornada empresarial em sandbox.
func benchmarkBusinessJourney(now time.Time) BenchmarkTaskResult {
	result := BenchmarkTaskResult{Name: "jornada_empresa_sandbox"}
	started := time.Now()
	runtimeInstance, root, err := benchmarkRuntime(result.Name)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	defer os.RemoveAll(root)

	input := DefaultBusinessJourneyInput(LocalOrganizationID, LocalOrganizationID+"-vizinho")
	journey, err := runtimeInstance.CertifyBusinessJourney(input, now)
	result.Steps = 1
	result.DurationMillis = time.Since(started).Milliseconds()
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	result.IsolationProven = journey.IsolationProven
	result.RecoveryProven = journey.RecoveryProven
	result.HumanApprovals = journey.Supervisor.AwaitingApproval
	for _, step := range journey.Steps {
		if step.Evidence != "" {
			result.Artifacts = append(result.Artifacts, step.Step)
		}
	}
	result.Completed = journey.Verdict == "SANDBOX_APROVADO" && journey.IsolationProven && journey.RecoveryProven
	result.Detail = fmt.Sprintf("veredito=%s passos=%d nao_feito=%d", journey.Verdict, len(journey.Steps), len(journey.NotDone))
	return result
}

// benchmarkIdempotentOrder mede o caminho de comércio com idempotência e recusa de dado
// parecido com cartão.
func benchmarkIdempotentOrder(now time.Time) BenchmarkTaskResult {
	result := BenchmarkTaskResult{Name: "pedido_idempotente"}
	started := time.Now()
	runtimeInstance, root, err := benchmarkRuntime(result.Name)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	defer os.RemoveAll(root)
	organization := LocalOrganizationID

	if _, err := runtimeInstance.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: organization, SKU: "BENCH-1", Name: "Produto de benchmark",
		PriceCents: 5000, Currency: "BRL", Active: true,
	}, now); err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	if _, err := runtimeInstance.Commerce().RecordStock(organization, "BENCH-1", 5, CommerceMovementEntry, "benchmark", now); err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++

	request := CommerceOrderRequest{
		OrganizationID: organization, ContactID: "contato-bench", Currency: "BRL",
		IdempotencyKey: "bench-pedido-1",
		Items:          []CommerceOrderItem{{SKU: "BENCH-1", Quantity: 2}},
	}
	first, err := runtimeInstance.Commerce().CreateOrder(request, now)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	// MESMA chave de idempotência: o efeito NÃO pode duplicar.
	second, err := runtimeInstance.Commerce().CreateOrder(request, now.Add(time.Minute))
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	idempotent := first.ID == second.ID && first.TotalCents == second.TotalCents

	// Referência com aparência de cartão precisa ser RECUSADA.
	cardLike := CommercePayment{Method: "pix", Reference: strings.Repeat("4", 16), AmountCents: first.TotalCents, RecordedAt: now}
	if _, err := runtimeInstance.Commerce().ConfirmPayment(organization, first.ID, cardLike, "benchmark", now); err == nil {
		result.Detail = "referência parecida com cartão foi ACEITA (deveria ser recusada)"
		return result
	}
	result.Steps++

	// Pagamento por referência válida é aceito.
	valid := CommercePayment{Method: "pix", Reference: "pix-bench-1", AmountCents: first.TotalCents, RecordedAt: now}
	paid, err := runtimeInstance.Commerce().ConfirmPayment(organization, first.ID, valid, "benchmark", now)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	summary := runtimeInstance.Commerce().Summary(organization)
	result.Steps++

	result.RecoveryProven = false
	result.Completed = idempotent && paid.Status != "" && summary.Orders == 1 && summary.Paid == 1
	result.DurationMillis = time.Since(started).Milliseconds()
	result.Detail = fmt.Sprintf("pedidos=%d pagos=%d idempotente=%t receita=%d",
		summary.Orders, summary.Paid, idempotent, summary.RevenueCents)
	return result
}

// benchmarkInboxSLAAndHandoff mede o SLA, a transferência para humano e a exigência de motivo.
func benchmarkInboxSLAAndHandoff(now time.Time) BenchmarkTaskResult {
	result := BenchmarkTaskResult{Name: "atendimento_sla_handoff"}
	started := time.Now()
	runtimeInstance, root, err := benchmarkRuntime(result.Name)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	defer os.RemoveAll(root)
	store := runtimeInstance.Inbox()
	organization := LocalOrganizationID

	// Dentro do SLA: primeira resposta em 2 minutos.
	prompt, _, err := store.AppendInbound(organization, "whatsapp", "contato-1", "Cliente 1", "olá", "ext-1", now)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	if _, _, err := store.AppendOutbound(organization, prompt.ID, "agent", "recepcao", "já vou ajudar", nil, now.Add(2*time.Minute)); err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++

	// Fora do SLA: sem resposta, a violação precisa ser MEDIDA.
	if _, _, err := store.AppendInbound(organization, "telegram", "contato-2", "Cliente 2", "preciso de ajuda", "ext-2", now); err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	metrics := store.Metrics(organization, now.Add(30*time.Minute))
	result.Steps++
	if metrics.SLABreached < 1 {
		result.Detail = fmt.Sprintf("violação de SLA não foi medida: %+v", metrics)
		return result
	}

	// Transferência SEM motivo precisa ser recusada.
	if _, _, err := store.HandoffToHuman(organization, prompt.ID, "agente-1", "humano-1", "", now.Add(3*time.Minute)); err == nil {
		result.Detail = "transferência sem motivo foi aceita (deveria ser recusada)"
		return result
	}
	result.Steps++
	// Com motivo, a conversa passa a aguardar humano.
	handed, _, err := store.HandoffToHuman(organization, prompt.ID, "agente-1", "humano-1", "cliente pediu pessoa", now.Add(4*time.Minute))
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	result.HumanApprovals = 1
	result.Completed = handed.Status == "pending_human" && metrics.MessagesInbound == 2
	result.DurationMillis = time.Since(started).Milliseconds()
	result.Detail = fmt.Sprintf("conversas=%d violacoes_sla=%d inbound=%d outbound=%d",
		metrics.TotalConversations, metrics.SLABreached, metrics.MessagesInbound, metrics.MessagesOutbound)
	return result
}

// benchmarkOrganizationIsolation mede o isolamento entre organizações em comércio e inbox.
func benchmarkOrganizationIsolation(now time.Time) BenchmarkTaskResult {
	result := BenchmarkTaskResult{Name: "isolamento_entre_organizacoes"}
	started := time.Now()
	runtimeInstance, root, err := benchmarkRuntime(result.Name)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	defer os.RemoveAll(root)
	const organizationA = "org-bench-a"
	const organizationB = "org-bench-b"

	if _, err := runtimeInstance.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: organizationA, SKU: "ISO-1", Name: "Só da A", PriceCents: 1000, Currency: "BRL", Active: true,
	}, now); err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	if _, _, err := runtimeInstance.Inbox().AppendInbound(organizationA, "whatsapp", "contato-a", "Cliente A", "oi", "iso-1", now); err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++

	// A organização B NÃO pode ver nada da A.
	productsB := runtimeInstance.Commerce().Products(organizationB)
	conversationsB := runtimeInstance.Inbox().ListConversations(organizationB, "")
	metricsB := runtimeInstance.Inbox().Metrics(organizationB, now)
	result.Steps += 3
	result.IsolationProven = len(productsB) == 0 && len(conversationsB) == 0 && metricsB.TotalConversations == 0
	result.Completed = result.IsolationProven && len(runtimeInstance.Commerce().Products(organizationA)) == 1
	result.DurationMillis = time.Since(started).Milliseconds()
	result.Detail = fmt.Sprintf("produtos_b=%d conversas_b=%d produtos_a=%d", len(productsB), len(conversationsB),
		len(runtimeInstance.Commerce().Products(organizationA)))
	return result
}

// benchmarkRestartRecovery mede a persistência e a retomada: um runtime NOVO, a partir do
// mesmo diretório de dados, precisa enxergar o estado e NÃO duplicar o efeito já aplicado.
func benchmarkRestartRecovery(now time.Time) BenchmarkTaskResult {
	result := BenchmarkTaskResult{Name: "retomada_apos_reinicio"}
	started := time.Now()
	root, err := os.MkdirTemp("", "hades-bench-restart-")
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	defer os.RemoveAll(root)
	dataRoot := filepath.Join(root, "data")
	organization := LocalOrganizationID

	before, err := NewRuntime(RuntimeConfig{
		Store: NewMemoryStore(), Planner: RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "ws"), DataRoot: dataRoot,
	})
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	if _, err := before.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: organization, SKU: "RET-1", Name: "Persistente", PriceCents: 2500, Currency: "BRL", Active: true,
	}, now); err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	// Sem entrada de estoque o pedido é (corretamente) recusado; a tarefa mede PERSISTÊNCIA,
	// então a carga inicial faz parte da preparação.
	if _, err := before.Commerce().RecordStock(organization, "RET-1", 3, CommerceMovementEntry, "benchmark", now); err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	request := CommerceOrderRequest{
		OrganizationID: organization, ContactID: "contato-ret", Currency: "BRL",
		IdempotencyKey: "ret-pedido-1",
		Items:          []CommerceOrderItem{{SKU: "RET-1", Quantity: 1}},
	}
	created, err := before.Commerce().CreateOrder(request, now)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++

	// REINÍCIO: runtime novo, mesmo diretório de dados.
	after, err := NewRuntime(RuntimeConfig{
		Store: NewMemoryStore(), Planner: RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "ws"), DataRoot: dataRoot,
	})
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	products := after.Commerce().Products(organization)
	orders := after.Commerce().Orders(organization)
	result.Steps += 2
	if len(products) != 1 || len(orders) != 1 {
		result.Detail = fmt.Sprintf("estado não sobreviveu ao reinício: produtos=%d pedidos=%d", len(products), len(orders))
		return result
	}
	// O MESMO efeito não pode ocorrer duas vezes: repetir a chave após o reinício devolve o
	// MESMO pedido, sem duplicar.
	replayed, err := after.Commerce().CreateOrder(request, now.Add(time.Minute))
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	result.Steps++
	afterOrders := after.Commerce().Orders(organization)
	result.Steps++
	result.RecoveryProven = replayed.ID == created.ID && len(afterOrders) == 1
	result.Completed = result.RecoveryProven && len(products) == 1
	result.DurationMillis = time.Since(started).Milliseconds()
	result.Detail = fmt.Sprintf("produtos=%d pedidos=%d repetido_mesmo_pedido=%t", len(products), len(afterOrders), replayed.ID == created.ID)
	return result
}
