package agent

import (
	"context"
	"testing"
	"time"
)

// Pré-requisito do retry de automação (estágio 13): sem idempotência na criação
// de missão, repetir um passo `action.mission` que falhou DEPOIS de persistir
// criaria missões fantasma. Estes testes travam a propriedade no nível do núcleo
// e no nível do fluxo.
func idempotencyRuntime(t *testing.T) (*Runtime, *ContextStore, Project) {
	t.Helper()
	store := NewMemoryStore()
	contextStore, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// O projeto precisa viver DENTRO do workspace do runtime: a criação valida
	// que a raiz do projeto não escapa do workspace.
	workspace := t.TempDir()
	project, err := contextStore.CreateProject("projeto idempotente", workspace, "org-a")
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         store,
		Context:       contextStore,
		Planner:       RulePlanner{},
		WorkspaceRoot: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime.WithOrganization("org-a"), contextStore, project
}

func TestCreateMissionIdempotencyKeyDoesNotDuplicateMission(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	request := CreateMissionRequest{
		Objective:      "tarefa idempotente",
		ProjectID:      project.ID,
		OrganizationID: "org-a",
		IdempotencyKey: "sched-1@run-1/acao",
	}
	first, err := runtime.CreateMission(context.Background(), request)
	if err != nil {
		t.Fatalf("primeira criação: %v", err)
	}
	second, err := runtime.CreateMission(context.Background(), request)
	if err != nil {
		t.Fatalf("segunda criação (mesma chave) deveria ser idempotente: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("mesma chave gerou missões diferentes: %s e %s", first.ID, second.ID)
	}
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 {
		t.Fatalf("a mesma chave não pode duplicar missão, existem %d", len(missions))
	}
}

func TestCreateMissionWithoutKeyStillCreatesDistinctMissions(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	request := CreateMissionRequest{Objective: "sem chave", ProjectID: project.ID, OrganizationID: "org-a"}
	first, err := runtime.CreateMission(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := runtime.CreateMission(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID {
		t.Fatal("sem chave de idempotência cada criação deve gerar missão própria")
	}
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 2 {
		t.Fatalf("esperava 2 missões, veio %d", len(missions))
	}
}

func TestIdempotentMissionIDIsStableAndOrganizationScoped(t *testing.T) {
	key := "sched-1@run-1/acao"
	first := idempotentMissionID("org-a", key)
	if first != idempotentMissionID("org-a", key) {
		t.Fatal("a mesma chave precisa derivar o mesmo id")
	}
	if first == idempotentMissionID("org-b", key) {
		t.Fatal("organizações diferentes não podem colidir na mesma chave")
	}
	if !validSnapshotID(first) {
		t.Fatalf("id derivado precisa ser um id válido, veio %q", first)
	}
	if first == idempotentMissionID("org-a", key+"-outra") {
		t.Fatal("chaves diferentes precisam derivar ids diferentes")
	}
}

func TestScheduleFlowRunKeyIsStableWithinRunAndChangesBetweenRuns(t *testing.T) {
	due := time.Date(2026, 10, 11, 1, 0, 0, 0, time.UTC)
	schedule := Schedule{ID: "sched-1", NextRunAt: due}
	first := ScheduleFlowRunKey(schedule)
	if first != ScheduleFlowRunKey(schedule) {
		t.Fatal("a chave da execução precisa ser estável para o mesmo NextRunAt (retry dentro da execução)")
	}
	schedule.NextRunAt = due.Add(time.Hour)
	if first == ScheduleFlowRunKey(schedule) {
		t.Fatal("execuções diferentes precisam ter chaves diferentes")
	}
	if scheduleStepIdempotencyKey("run", "acao") == scheduleStepIdempotencyKey("run", "outra") {
		t.Fatal("passos diferentes precisam ter chaves diferentes")
	}
}

func TestScheduleFlowRepeatDoesNotCreateDuplicateMission(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	schedule := Schedule{
		ID:             "sched-idem",
		OrganizationID: "org-a",
		ProjectID:      project.ID,
		Objective:      "corpo",
		NextRunAt:      time.Date(2026, 10, 11, 1, 0, 0, 0, time.UTC),
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"gatilho"}, Objective: "missão do passo"},
		},
	}
	first, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err != nil {
		t.Fatalf("primeira execução: %v", err)
	}
	if first.Missions != 1 {
		t.Fatalf("primeira execução deveria criar 1 missão, veio %d (%+v)", first.Missions, first.Steps)
	}

	// Repetir a MESMA execução (mesmo NextRunAt) não pode criar outra missão.
	second, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err != nil {
		t.Fatalf("segunda execução: %v", err)
	}
	if second.Missions != 0 {
		t.Fatalf("repetição não pode criar missão nova, veio %d", second.Missions)
	}
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 {
		t.Fatalf("o fluxo repetido duplicou missão: %d", len(missions))
	}
	var reason string
	for _, step := range second.Steps {
		if step.ID == "acao" {
			reason = step.Reason
			if step.Status != ScheduleStepSucceeded {
				t.Fatalf("passo idempotente deveria constar como executado: %+v", step)
			}
		}
	}
	if reason == "" {
		t.Fatal("o relatório precisa dizer que a missão já existia, em vez de contá-la como criada")
	}
}
