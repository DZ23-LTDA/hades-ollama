package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

// Achado do estágio 13: o botão "Executar imediatamente" criava uma missão
// avulsa (`createMission` com o objetivo do schedule) e IGNORAVA os passos
// persistidos — um fluxo com condição e várias ações rodava como uma única missão,
// e o operador não recebia evidência por nó. A rota de execução manual passou a
// rodar o mesmo caminho do despacho automático.
func manualRunAPI(t *testing.T) (*agentAPI, agent.Schedule) {
	t.Helper()
	contextStore, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	project, err := contextStore.CreateProject("projeto do schedule", workspace, "org-a")
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := contextStore.CreateSchedule(agent.Schedule{
		ID:              "sched-manual",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		ProjectID:       project.ID,
		IntervalSeconds: 60,
		Enabled:         true,
		Steps: []agent.ScheduleStep{
			{ID: "gatilho", Kind: agent.ScheduleStepTriggerInterval},
			{ID: "acao", Kind: agent.ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "missão do passo"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Context:       contextStore,
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: workspace, DataRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return &agentAPI{context: contextStore, runtime: runtime, authRequired: true}, schedule
}

func TestRunScheduleNowExecutesThePersistedFlowAndRecordsEvidence(t *testing.T) {
	api, schedule := manualRunAPI(t)
	ctx, recorder := skillRequest(t, http.MethodPost, "/api/agent/v1/schedules/"+schedule.ID+"/run", "{}", schedule.ID, "org-a")

	api.runScheduleNow(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("execução manual deveria responder 200, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Succeeded bool                    `json:"succeeded"`
		Report    agent.ScheduleRunReport `json:"report"`
		Schedule  agent.Schedule          `json:"schedule"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Succeeded {
		t.Fatalf("fluxo válido deveria ter sucesso: %s", recorder.Body.String())
	}
	// O grafo foi executado de verdade: dois passos, um deles criando missão.
	if len(payload.Report.Steps) != 2 {
		t.Fatalf("relatório precisa ter os passos do GRAFO, veio %+v", payload.Report.Steps)
	}
	if payload.Report.Missions != 1 {
		t.Fatalf("o passo action.mission deveria criar 1 missão, veio %d", payload.Report.Missions)
	}
	if payload.Report.RunKey == "" {
		t.Fatal("a execução precisa registrar a chave da execução para permitir reprocessamento")
	}
	// E a evidência ficou PERSISTIDA no schedule.
	if payload.Schedule.LastRun == nil || payload.Schedule.LastRunAt == nil {
		t.Fatalf("execução manual precisa registrar last_run: %+v", payload.Schedule)
	}
	stored, err := api.context.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LastRun == nil || len(stored.LastRun.Steps) != 2 {
		t.Fatalf("evidência não persistiu no store: %+v", stored.LastRun)
	}
	if stored.LastRun.RunKey == "" {
		t.Fatal("a chave da execução precisa ser persistida junto do relatório")
	}
}

func TestRunScheduleNowIsScopedToTheActiveOrganization(t *testing.T) {
	api, schedule := manualRunAPI(t)
	ctx, recorder := skillRequest(t, http.MethodPost, "/api/agent/v1/schedules/"+schedule.ID+"/run", "{}", schedule.ID, "org-b")
	api.runScheduleNow(ctx)
	if recorder.Code == http.StatusOK {
		t.Fatalf("outra organização não pode executar o schedule alheio: %s", recorder.Body.String())
	}
	if recorder.Code != http.StatusNotFound && recorder.Code != http.StatusForbidden {
		t.Fatalf("recusa deveria ser 404/403, veio %d", recorder.Code)
	}
	stored, err := api.context.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LastRun != nil {
		t.Fatalf("execução recusada não pode deixar evidência: %+v", stored.LastRun)
	}
}

func TestRunScheduleNowReportsFailureWithTheStepNamed(t *testing.T) {
	api, schedule := manualRunAPI(t)
	// Remove o projeto para que a criação da missão falhe de forma determinística.
	schedule.ProjectID = ""
	if _, err := api.context.UpdateSchedule(schedule.ID, schedule); err != nil {
		t.Fatal(err)
	}
	ctx, recorder := skillRequest(t, http.MethodPost, "/api/agent/v1/schedules/"+schedule.ID+"/run", "{}", schedule.ID, "org-a")
	api.runScheduleNow(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("execução que FALHA ainda precisa devolver o relatório, veio %d", recorder.Code)
	}
	var payload struct {
		Succeeded bool                    `json:"succeeded"`
		Error     string                  `json:"error"`
		Report    agent.ScheduleRunReport `json:"report"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Succeeded {
		t.Fatal("fluxo que falha não pode ser reportado como sucesso")
	}
	if payload.Error == "" || payload.Report.FailedStep() != "acao" {
		t.Fatalf("falha precisa nomear o passo: erro=%q relatório=%+v", payload.Error, payload.Report.Steps)
	}
	stored, err := api.context.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LastFailureCode != "mission_creation_failed:acao" {
		t.Fatalf("código de falha deveria nomear o passo, veio %q", stored.LastFailureCode)
	}
}

// Reprocessamento pela rota: reexecuta o que ficou pendente reusando a execução
// anterior, e recusa explicitamente quando não há execução para reprocessar.
func TestReprocessScheduleNowRefusesWithoutPreviousRun(t *testing.T) {
	api, schedule := manualRunAPI(t)
	ctx, recorder := skillRequest(t, http.MethodPost, "/api/agent/v1/schedules/"+schedule.ID+"/reprocess", "{}", schedule.ID, "org-a")
	api.reprocessScheduleNow(ctx)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("sem execução anterior deveria ser 409, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "no previous run") {
		t.Fatalf("a recusa precisa explicar o motivo: %s", recorder.Body.String())
	}
}

func TestReprocessScheduleNowAfterRunPreservesMissions(t *testing.T) {
	api, schedule := manualRunAPI(t)
	// 1) executa uma vez (registra a execução e cria a missão)
	runCtx, runRecorder := skillRequest(t, http.MethodPost, "/api/agent/v1/schedules/"+schedule.ID+"/run", "{}", schedule.ID, "org-a")
	api.runScheduleNow(runCtx)
	if runRecorder.Code != http.StatusOK {
		t.Fatalf("execução inicial: %d %s", runRecorder.Code, runRecorder.Body.String())
	}

	// 2) reprocessa: nada pendente, então não cria missão nova
	reprocessCtx, reprocessRecorder := skillRequest(t, http.MethodPost, "/api/agent/v1/schedules/"+schedule.ID+"/reprocess", "{}", schedule.ID, "org-a")
	api.reprocessScheduleNow(reprocessCtx)
	if reprocessRecorder.Code != http.StatusOK {
		t.Fatalf("reprocessamento: %d %s", reprocessRecorder.Code, reprocessRecorder.Body.String())
	}
	var payload struct {
		Succeeded bool                    `json:"succeeded"`
		Report    agent.ScheduleRunReport `json:"report"`
	}
	if err := json.Unmarshal(reprocessRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Succeeded {
		t.Fatalf("reprocessamento sem pendências deveria ter sucesso: %s", reprocessRecorder.Body.String())
	}
	if payload.Report.Missions != 0 {
		t.Fatalf("passos já bem-sucedidos não podem criar missão no reprocessamento, veio %d", payload.Report.Missions)
	}
	if payload.Report.RunKey == "" {
		t.Fatal("o reprocessamento precisa manter a chave da execução")
	}
}

func TestReprocessScheduleNowIsScopedToTheActiveOrganization(t *testing.T) {
	api, schedule := manualRunAPI(t)
	runCtx, runRecorder := skillRequest(t, http.MethodPost, "/api/agent/v1/schedules/"+schedule.ID+"/run", "{}", schedule.ID, "org-a")
	api.runScheduleNow(runCtx)
	if runRecorder.Code != http.StatusOK {
		t.Fatalf("execução inicial: %d", runRecorder.Code)
	}
	ctx, recorder := skillRequest(t, http.MethodPost, "/api/agent/v1/schedules/"+schedule.ID+"/reprocess", "{}", schedule.ID, "org-b")
	api.reprocessScheduleNow(ctx)
	if recorder.Code == http.StatusOK {
		t.Fatalf("outra organização não pode reprocessar o schedule alheio: %s", recorder.Body.String())
	}
}

// Nó de aprovação pela rota: a decisão humana destrava a retomada, exige uma
// execução em espera e respeita o isolamento por organização.
func TestDecideScheduleApprovalUnblocksThePausedRun(t *testing.T) {
	api, schedule := manualRunAPI(t)
	// Um passo de aprovação entre o gatilho e a ação.
	schedule.Steps = []agent.ScheduleStep{
		{ID: "gatilho", Kind: agent.ScheduleStepTriggerInterval},
		{ID: "liberar", Kind: agent.ScheduleStepApprovalRequest, DependsOn: []string{"gatilho"}, Objective: "liberar envio"},
		{ID: "acao", Kind: agent.ScheduleStepActionMission, DependsOn: []string{"liberar"}, Objective: "enviar"},
	}
	if _, err := api.context.UpdateSchedule(schedule.ID, schedule); err != nil {
		t.Fatal(err)
	}
	// 1) A execução PAUSA: nada criado e uma pendência registrada.
	runCtx, runRecorder := skillRequest(t, http.MethodPost, "/api/agent/v1/schedules/"+schedule.ID+"/run", "{}", schedule.ID, "org-a")
	api.runScheduleNow(runCtx)
	if runRecorder.Code != http.StatusOK {
		t.Fatalf("execução: %d %s", runRecorder.Code, runRecorder.Body.String())
	}
	paused, err := api.context.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(paused.PendingApprovals) != 1 || paused.PendingApprovals[0].StepID != "liberar" {
		t.Fatalf("a pausa precisa ficar registrada: %+v", paused.PendingApprovals)
	}

	// 2) Decidir sem informar a execução usa a que está PARADA (conveniência real
	// do operador, sem adivinhar passo).
	decisionCtx, decisionRecorder := skillRequest(t, http.MethodPost,
		"/api/agent/v1/schedules/"+schedule.ID+"/approvals",
		`{"step_id":"liberar","approved":true,"reason":"ok"}`, schedule.ID, "org-a")
	api.decideScheduleApproval(decisionCtx)
	if decisionRecorder.Code != http.StatusOK {
		t.Fatalf("decisão: %d %s", decisionRecorder.Code, decisionRecorder.Body.String())
	}
	decided, err := api.context.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(decided.PendingApprovals) != 0 || len(decided.Approvals) != 1 || !decided.Approvals[0].Approved {
		t.Fatalf("decisão não foi registrada corretamente: %+v / %+v", decided.PendingApprovals, decided.Approvals)
	}

	// 3) Retomar pelo reprocessamento executa o passo que estava barrado.
	resumeCtx, resumeRecorder := skillRequest(t, http.MethodPost, "/api/agent/v1/schedules/"+schedule.ID+"/reprocess", "{}", schedule.ID, "org-a")
	api.reprocessScheduleNow(resumeCtx)
	if resumeRecorder.Code != http.StatusOK {
		t.Fatalf("retomada: %d %s", resumeRecorder.Code, resumeRecorder.Body.String())
	}
	var payload struct {
		Succeeded bool                    `json:"succeeded"`
		Report    agent.ScheduleRunReport `json:"report"`
	}
	if err := json.Unmarshal(resumeRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Succeeded || payload.Report.Missions != 1 {
		t.Fatalf("após a aprovação a ação deveria criar 1 missão: %s", resumeRecorder.Body.String())
	}
}

func TestDecideScheduleApprovalRefusesWithoutPendingApproval(t *testing.T) {
	api, schedule := manualRunAPI(t)
	ctx, recorder := skillRequest(t, http.MethodPost,
		"/api/agent/v1/schedules/"+schedule.ID+"/approvals",
		`{"step_id":"liberar","approved":true}`, schedule.ID, "org-a")
	api.decideScheduleApproval(ctx)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("sem execução em espera a decisão precisa ser recusada (409), veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	// E nada foi registrado.
	stored, err := api.context.GetSchedule(schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Approvals) != 0 {
		t.Fatalf("decisão recusada não pode ficar registrada: %+v", stored.Approvals)
	}
}

func TestDecideScheduleApprovalIsScopedToTheActiveOrganization(t *testing.T) {
	api, schedule := manualRunAPI(t)
	ctx, recorder := skillRequest(t, http.MethodPost,
		"/api/agent/v1/schedules/"+schedule.ID+"/approvals",
		`{"step_id":"liberar","approved":true}`, schedule.ID, "org-b")
	api.decideScheduleApproval(ctx)
	if recorder.Code == http.StatusOK {
		t.Fatalf("outra organização não pode decidir sobre o schedule alheio: %s", recorder.Body.String())
	}
}
