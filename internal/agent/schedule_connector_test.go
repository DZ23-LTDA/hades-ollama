package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Nó de conector (estágio 13): chama o conector CONFIGURADO pelo caminho real
// (CallForOrganization), com escopo de organização e credencial do tenant.
// As regras que estes testes travam: conector ausente FALHA (não é pulado),
// conector de outra organização é recusado, e método que ESCREVE só sai com
// aprovação humana registrada para a mesma execução.
func connectorScheduleFixture(t *testing.T, server *httptest.Server, methods []string) (*Runtime, *ContextStore, Schedule) {
	t.Helper()
	runtime, contextStore, project := idempotencyRuntime(t)
	manager := NewConnectorManager()
	// O registrador exige HTTPS: o cliente do servidor de teste é o que confia no
	// certificado efêmero.
	manager.client = server.Client()
	if err := manager.RegisterForOrganization("org-a", ConnectorConfig{
		ID:         "crm",
		Provider:   "test",
		BaseURL:    server.URL,
		Operations: []ConnectorOperation{{Name: "contatos", Methods: methods, PathPrefixes: []string{"/contatos"}}},
	}); err != nil {
		t.Fatal(err)
	}
	configured, err := NewRuntime(RuntimeConfig{
		Store:         runtime.store,
		Context:       contextStore,
		Planner:       RulePlanner{},
		WorkspaceRoot: t.TempDir(),
		Connectors:    manager,
	})
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-conector",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		ProjectID:       project.ID,
		IntervalSeconds: 60,
		NextRunAt:       time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC),
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return configured.WithOrganization("org-a"), contextStore, schedule
}

// deadTLSServer devolve um servidor HTTPS já encerrado: serve de base_url válida
// quando o teste quer provar que a falha acontece ANTES de qualquer chamada.
func deadTLSServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Close()
	return server
}

func TestConnectorNodeCallsRealConnectorAndPublishesResponse(t *testing.T) {
	var sawAuthorization string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total":7}`))
	}))
	defer server.Close()

	runtime, _, schedule := connectorScheduleFixture(t, server, []string{"GET"})
	schedule.Steps = append(schedule.Steps, ScheduleStep{
		ID: "consultar", Kind: ScheduleStepActionConnector, DependsOn: []string{"gatilho"},
		Connector: "crm", ConnectorOperation: "contatos", ConnectorMethod: "GET",
		ConnectorPath: "/contatos", Target: "total",
	})
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr != nil {
		t.Fatalf("chamada GET deveria funcionar: %v", runErr)
	}
	var step ScheduleStepOutcome
	for _, outcome := range report.Steps {
		if outcome.ID == "consultar" {
			step = outcome
		}
	}
	if step.Status != ScheduleStepSucceeded || !strings.Contains(step.Reason, "HTTP 200") {
		t.Fatalf("o passo precisa registrar a resposta real: %+v", step)
	}
	if !strings.Contains(step.Reason, "total") {
		t.Fatalf("a resposta precisa ser publicada na variável de destino: %q", step.Reason)
	}
	// A chamada saiu pelo caminho do conector (aqui sem credencial configurada, o
	// cabeçalho fica vazio, mas a requisição CHEGOU ao destino).
	if sawAuthorization != "" {
		t.Fatalf("sem credencial configurada não pode haver Authorization: %q", sawAuthorization)
	}
}

func TestConnectorNodeFailsClosedWhenConnectorIsMissing(t *testing.T) {
	runtime, _, schedule := connectorScheduleFixture(t, deadTLSServer(t), []string{"GET"})
	schedule.Steps = append(schedule.Steps, ScheduleStep{
		ID: "ausente", Kind: ScheduleStepActionConnector, DependsOn: []string{"gatilho"},
		Connector: "nao-existe", ConnectorOperation: "contatos", ConnectorMethod: "GET", ConnectorPath: "/contatos",
	})
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr == nil {
		t.Fatal("conector não registrado precisa FALHAR, não ser pulado")
	}
	if !strings.Contains(runErr.Error(), "not registered") {
		t.Fatalf("o erro precisa dizer que o conector não existe: %v", runErr)
	}
	if len(report.DeadLetters) != 1 || report.DeadLetters[0].StepID != "ausente" {
		t.Fatalf("a falha precisa entrar na fila morta: %+v", report.DeadLetters)
	}
}

func TestConnectorNodeRefusesAnotherOrganizationConnector(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	runtime, _, schedule := connectorScheduleFixture(t, server, []string{"GET"})
	// O conector pertence a org-a; a execução se declara de outra organização.
	schedule.OrganizationID = "org-b"
	schedule.Steps = append(schedule.Steps, ScheduleStep{
		ID: "cruzado", Kind: ScheduleStepActionConnector, DependsOn: []string{"gatilho"},
		Connector: "crm", ConnectorOperation: "contatos", ConnectorMethod: "GET", ConnectorPath: "/contatos",
	})
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr == nil {
		t.Fatal("conector de OUTRA organização precisa ser recusado")
	}
	if len(report.DeadLetters) != 1 {
		t.Fatalf("a recusa de escopo precisa ficar registrada: %+v", report.DeadLetters)
	}
}

func TestConnectorWriteMethodRequiresApprovalAndDenialBlocks(t *testing.T) {
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	runtime, contextStore, schedule := connectorScheduleFixture(t, server, []string{"POST"})
	schedule.Steps = append(schedule.Steps, ScheduleStep{
		ID: "criar", Kind: ScheduleStepActionConnector, DependsOn: []string{"gatilho"},
		Connector: "crm", ConnectorOperation: "contatos", ConnectorMethod: "POST",
		ConnectorPath: "/contatos", Body: `{"nome":"{{schedule.id}}"}`,
	})
	if _, err := contextStore.UpdateSchedule(schedule.ID, schedule); err != nil {
		t.Fatal(err)
	}

	// 1) Sem decisão: o passo PARA e NENHUMA chamada sai.
	paused, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr != nil {
		t.Fatalf("espera por aprovação não é falha: %v", runErr)
	}
	if calls != 0 {
		t.Fatalf("método que escreve não pode sair sem aprovação (chamadas=%d)", calls)
	}
	var pending bool
	for _, step := range paused.Steps {
		if step.ID == "criar" && step.Status == ScheduleStepPending {
			pending = true
		}
	}
	if !pending || len(paused.PendingApprovals) != 1 {
		t.Fatalf("o passo de escrita precisa ficar pendente: %+v / %+v", paused.Steps, paused.PendingApprovals)
	}
	recorded, err := runtime.RecordScheduleRun(schedule, paused, nil)
	if err != nil {
		t.Fatal(err)
	}

	// 2) Negativa: continua sem sair e sem virar falha técnica.
	denied, err := runtime.DecideScheduleApproval(recorded, "criar", paused.RunKey, false, "ana", "fora do horário")
	if err != nil {
		t.Fatal(err)
	}
	reportDenied, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), denied)
	if runErr != nil {
		t.Fatalf("negativa não é falha técnica: %v", runErr)
	}
	if calls != 0 || len(reportDenied.DeadLetters) != 0 {
		t.Fatalf("negativa não pode chamar nem entrar na fila morta (calls=%d)", calls)
	}

	// 3) Aprovação: agora a chamada sai de verdade, com o corpo expandido.
	approved, err := runtime.DecideScheduleApproval(recorded, "criar", paused.RunKey, true, "bia", "ok")
	if err != nil {
		t.Fatal(err)
	}
	reportApproved, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), approved)
	if runErr != nil {
		t.Fatalf("após aprovação a chamada deveria sair: %v", runErr)
	}
	if calls != 1 {
		t.Fatalf("esperava exatamente 1 chamada após aprovação, veio %d", calls)
	}
	if reportApproved.Missions != 0 {
		t.Fatalf("nó de conector não cria missão: %d", reportApproved.Missions)
	}
}

func TestConnectorNodeTreatsServerErrorAsFailure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	runtime, _, schedule := connectorScheduleFixture(t, server, []string{"GET"})
	schedule.Steps = append(schedule.Steps, ScheduleStep{
		ID: "quebrado", Kind: ScheduleStepActionConnector, DependsOn: []string{"gatilho"},
		Connector: "crm", ConnectorOperation: "contatos", ConnectorMethod: "GET", ConnectorPath: "/contatos",
	})
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr == nil {
		t.Fatal("HTTP 500 do conector não é sucesso do nosso nó")
	}
	if len(report.DeadLetters) != 1 || !strings.Contains(report.DeadLetters[0].Reason, "500") {
		t.Fatalf("a falha precisa registrar o status real: %+v", report.DeadLetters)
	}
}

func TestValidateScheduleConnectorRejectsBadShapes(t *testing.T) {
	base := func(step ScheduleStep) []ScheduleStep {
		return []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			step,
		}
	}
	valid := ScheduleStep{ID: "c", Kind: ScheduleStepActionConnector, DependsOn: []string{"gatilho"}, Connector: "crm", ConnectorOperation: "contatos", ConnectorMethod: "GET", ConnectorPath: "/contatos"}
	if err := ValidateScheduleSteps(base(valid)); err != nil {
		t.Fatalf("conector válido deveria ser aceito: %v", err)
	}
	semConector := valid
	semConector.Connector = ""
	if err := ValidateScheduleSteps(base(semConector)); err == nil {
		t.Fatal("conector sem id precisa ser recusado")
	}
	metodoInvalido := valid
	metodoInvalido.ConnectorMethod = "TRACE"
	if err := ValidateScheduleSteps(base(metodoInvalido)); err == nil {
		t.Fatal("método não suportado precisa ser recusado")
	}
	caminhoRelativo := valid
	caminhoRelativo.ConnectorPath = "contatos"
	if err := ValidateScheduleSteps(base(caminhoRelativo)); err == nil {
		t.Fatal("caminho sem / precisa ser recusado")
	}
}
