package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// flowStep is a small helper keeping the validation table readable.
func flowStep(id string, kind ScheduleStepKind, dependsOn []string, objective, expect string) ScheduleStep {
	return ScheduleStep{ID: id, Kind: kind, DependsOn: dependsOn, Objective: objective, Expect: expect}
}

func TestValidateScheduleStepsAcceptsThreeNodeFlow(t *testing.T) {
	steps := []ScheduleStep{
		flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
		flowStep("coletar", ScheduleStepActionMission, []string{"gatilho"}, "Coletar métricas", ""),
		flowStep("decidir", ScheduleStepConditionIf, []string{"coletar"}, "", ScheduleExpectSucceeded),
	}
	if err := ValidateScheduleSteps(steps); err != nil {
		t.Fatalf("esperava fluxo válido, obteve: %v", err)
	}
}

func TestValidateScheduleStepsAcceptsLegacyEmptyGraph(t *testing.T) {
	if err := ValidateScheduleSteps(nil); err != nil {
		t.Fatalf("grafo vazio deve continuar válido, obteve: %v", err)
	}
}

func TestValidateScheduleStepsRejectsMalformedGraphs(t *testing.T) {
	tooMany := make([]ScheduleStep, 0, maxScheduleFlowSteps+1)
	tooMany = append(tooMany, flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""))
	for i := 0; i <= maxScheduleFlowSteps; i++ {
		tooMany = append(tooMany, flowStep(
			"passo"+strings.Repeat("x", i%3)+string(rune('a'+i%26))+string(rune('0'+i/26)),
			ScheduleStepConditionIf, []string{"gatilho"}, "", ScheduleExpectSucceeded,
		))
	}

	cases := []struct {
		name  string
		steps []ScheduleStep
		want  string
	}{
		{
			name: "tipo fora do vocabulário",
			steps: []ScheduleStep{
				flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
				flowStep("shell", ScheduleStepKind("action.shell"), []string{"gatilho"}, "rodar", ""),
			},
			want: "unknown schedule step kind",
		},
		{
			name: "sem gatilho",
			steps: []ScheduleStep{
				flowStep("a", ScheduleStepActionMission, []string{"b"}, "objetivo", ""),
				flowStep("b", ScheduleStepActionMission, []string{"a"}, "objetivo", ""),
			},
			want: "exactly one trigger",
		},
		{
			name: "dois gatilhos",
			steps: []ScheduleStep{
				flowStep("t1", ScheduleStepTriggerInterval, nil, "", ""),
				flowStep("t2", ScheduleStepTriggerWebhook, nil, "", ""),
			},
			want: "exactly one trigger",
		},
		{
			name: "ciclo entre ações",
			steps: []ScheduleStep{
				flowStep("t", ScheduleStepTriggerInterval, nil, "", ""),
				flowStep("a", ScheduleStepActionMission, []string{"b"}, "objetivo a", ""),
				flowStep("b", ScheduleStepActionMission, []string{"a"}, "objetivo b", ""),
			},
			want: "acyclic",
		},
		{
			name: "ação sem objetivo",
			steps: []ScheduleStep{
				flowStep("t", ScheduleStepTriggerInterval, nil, "", ""),
				flowStep("a", ScheduleStepActionMission, []string{"t"}, "   ", ""),
			},
			want: "requires an objective",
		},
		{
			name: "ação sem dependência",
			steps: []ScheduleStep{
				flowStep("t", ScheduleStepTriggerInterval, nil, "", ""),
				flowStep("a", ScheduleStepActionMission, nil, "objetivo", ""),
			},
			want: "requires at least one dependency",
		},
		{
			name: "gatilho com dependência",
			steps: []ScheduleStep{
				flowStep("t", ScheduleStepTriggerInterval, []string{"a"}, "", ""),
				flowStep("a", ScheduleStepActionMission, []string{"t"}, "objetivo", ""),
			},
			want: "cannot depend on another step",
		},
		{
			name: "condição com espera inválida",
			steps: []ScheduleStep{
				flowStep("t", ScheduleStepTriggerInterval, nil, "", ""),
				flowStep("a", ScheduleStepActionMission, []string{"t"}, "objetivo", ""),
				flowStep("c", ScheduleStepConditionIf, []string{"a"}, "", "talvez"),
			},
			want: "requires expect",
		},
		{
			name: "condição com duas dependências",
			steps: []ScheduleStep{
				flowStep("t", ScheduleStepTriggerInterval, nil, "", ""),
				flowStep("a", ScheduleStepActionMission, []string{"t"}, "objetivo", ""),
				flowStep("c", ScheduleStepConditionIf, []string{"t", "a"}, "", ScheduleExpectSucceeded),
			},
			want: "exactly one dependency",
		},
		{
			name: "dependência inexistente",
			steps: []ScheduleStep{
				flowStep("t", ScheduleStepTriggerInterval, nil, "", ""),
				flowStep("a", ScheduleStepActionMission, []string{"fantasma"}, "objetivo", ""),
			},
			want: "unknown step",
		},
		{
			name: "identificador duplicado",
			steps: []ScheduleStep{
				flowStep("t", ScheduleStepTriggerInterval, nil, "", ""),
				flowStep("t", ScheduleStepActionMission, []string{"t"}, "objetivo", ""),
			},
			want: "duplicated",
		},
		{
			name: "identificador vazio",
			steps: []ScheduleStep{
				flowStep("", ScheduleStepTriggerInterval, nil, "", ""),
			},
			want: "id is required",
		},
		{
			name:  "passos demais",
			steps: tooMany,
			want:  "at most",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateScheduleSteps(tc.steps)
			if err == nil {
				t.Fatalf("esperava erro contendo %q, obteve nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("esperava erro contendo %q, obteve %q", tc.want, err.Error())
			}
		})
	}
}

func TestValidateScheduleStepsRejectsUnknownKindWithSentinel(t *testing.T) {
	err := ValidateScheduleSteps([]ScheduleStep{
		flowStep("x", ScheduleStepKind("action.desconhecida"), nil, "", ""),
	})
	if !errors.Is(err, ErrUnknownScheduleStep) {
		t.Fatalf("esperava ErrUnknownScheduleStep, obteve: %v", err)
	}
}

func TestOrderScheduleStepsRespectsDependencies(t *testing.T) {
	steps := []ScheduleStep{
		flowStep("terceiro", ScheduleStepActionMission, []string{"segundo"}, "objetivo 3", ""),
		flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
		flowStep("segundo", ScheduleStepActionMission, []string{"primeiro"}, "objetivo 2", ""),
		flowStep("primeiro", ScheduleStepActionMission, []string{"gatilho"}, "objetivo 1", ""),
	}
	order, err := orderScheduleSteps(steps)
	if err != nil {
		t.Fatalf("esperava ordem válida, obteve: %v", err)
	}
	got := make([]string, 0, len(order))
	for _, step := range order {
		got = append(got, step.ID)
	}
	want := []string{"gatilho", "primeiro", "segundo", "terceiro"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ordem topológica inesperada: %v (esperado %v)", got, want)
	}
}

func TestExecuteScheduleFlowLegacyScheduleRunsSingleMission(t *testing.T) {
	runtime, _, _ := setupTestSupervisorRuntime(t)

	missions, err := runtime.ExecuteScheduleFlow(context.Background(), Schedule{
		Objective:      "Objetivo legado",
		Workspace:      "",
		OrganizationID: LocalOrganizationID,
	})
	if err != nil {
		t.Fatalf("esperava execução legada sem erro, obteve: %v", err)
	}
	if missions != 1 {
		t.Fatalf("esperava 1 missão no caminho legado, obteve %d", missions)
	}
}

func TestExecuteScheduleFlowRunsThreeNodeGraphInOrder(t *testing.T) {
	runtime, _, _ := setupTestSupervisorRuntime(t)

	schedule := Schedule{
		Objective:      "Fluxo de três nós",
		OrganizationID: LocalOrganizationID,
		Steps: []ScheduleStep{
			flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
			flowStep("coletar", ScheduleStepActionMission, []string{"gatilho"}, "Coletar métricas do dia", ""),
			flowStep("decidir", ScheduleStepConditionIf, []string{"coletar"}, "", ScheduleExpectSucceeded),
			flowStep("publicar", ScheduleStepActionMission, []string{"decidir"}, "Publicar relatório diário", ""),
		},
	}

	missions, err := runtime.ExecuteScheduleFlow(context.Background(), schedule)
	if err != nil {
		t.Fatalf("esperava fluxo válido sem erro, obteve: %v", err)
	}
	if missions != 2 {
		t.Fatalf("esperava 2 missões (coletar e publicar), obteve %d", missions)
	}

	created, err := runtime.ListMissions()
	if err != nil {
		t.Fatalf("listar missões: %v", err)
	}
	objectives := make(map[string]bool, len(created))
	for _, mission := range created {
		objectives[mission.Objective] = true
	}
	for _, want := range []string{"Coletar métricas do dia", "Publicar relatório diário"} {
		if !objectives[want] {
			t.Fatalf("missão %q não foi criada; criadas: %v", want, objectives)
		}
	}
}

func TestExecuteScheduleFlowConditionFalseBlocksDependents(t *testing.T) {
	runtime, _, _ := setupTestSupervisorRuntime(t)

	schedule := Schedule{
		Objective:      "Fluxo bloqueado",
		OrganizationID: LocalOrganizationID,
		Steps: []ScheduleStep{
			flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
			flowStep("coletar", ScheduleStepActionMission, []string{"gatilho"}, "Coletar métricas do dia", ""),
			flowStep("decidir", ScheduleStepConditionIf, []string{"coletar"}, "", ScheduleExpectFailed),
			flowStep("publicar", ScheduleStepActionMission, []string{"decidir"}, "Publicar relatório diário", ""),
		},
	}

	missions, err := runtime.ExecuteScheduleFlow(context.Background(), schedule)
	if err != nil {
		t.Fatalf("esperava fluxo sem erro, obteve: %v", err)
	}
	if missions != 1 {
		t.Fatalf("esperava apenas 1 missão com condição falsa, obteve %d", missions)
	}

	created, err := runtime.ListMissions()
	if err != nil {
		t.Fatalf("listar missões: %v", err)
	}
	for _, mission := range created {
		if mission.Objective == "Publicar relatório diário" {
			t.Fatal("passo dependente executou apesar da condição falsa")
		}
	}
}

func TestCreateScheduleRejectsMalformedFlowBeforePersistence(t *testing.T) {
	_, _, store := setupTestSupervisorRuntime(t)

	_, err := store.CreateSchedule(Schedule{
		Objective:       "Fluxo inválido",
		IntervalSeconds: 3600,
		OrganizationID:  LocalOrganizationID,
		Steps: []ScheduleStep{
			flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
			flowStep("nó", ScheduleStepKind("action.shell"), []string{"gatilho"}, "rodar", ""),
		},
	})
	if err == nil {
		t.Fatal("esperava rejeição de fluxo com tipo desconhecido")
	}
	if !errors.Is(err, ErrUnknownScheduleStep) {
		t.Fatalf("esperava ErrUnknownScheduleStep, obteve: %v", err)
	}
	if schedules := store.ListSchedules(); len(schedules) != 0 {
		t.Fatalf("nada deveria ter sido persistido, obteve %d agendamento(s)", len(schedules))
	}
}

func TestCreateAndUpdateSchedulePersistFlowGraph(t *testing.T) {
	_, _, store := setupTestSupervisorRuntime(t)

	steps := []ScheduleStep{
		flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
		flowStep("coletar", ScheduleStepActionMission, []string{"gatilho"}, "Coletar métricas", ""),
	}
	created, err := store.CreateSchedule(Schedule{
		Objective:       "Fluxo persistido",
		IntervalSeconds: 600,
		OrganizationID:  LocalOrganizationID,
		Steps:           steps,
	})
	if err != nil {
		t.Fatalf("criar agendamento com fluxo: %v", err)
	}
	if len(created.Steps) != 2 {
		t.Fatalf("esperava 2 passos persistidos, obteve %d", len(created.Steps))
	}
	reloaded, err := store.GetSchedule(created.ID)
	if err != nil {
		t.Fatalf("recarregar agendamento: %v", err)
	}
	if len(reloaded.Steps) != 2 || reloaded.Steps[1].Objective != "Coletar métricas" {
		t.Fatalf("fluxo não sobreviveu à persistência: %+v", reloaded.Steps)
	}

	reloaded.Steps = append(reloaded.Steps, flowStep("decidir", ScheduleStepConditionIf, []string{"coletar"}, "", ScheduleExpectSucceeded))
	updated, err := store.UpdateSchedule(created.ID, reloaded)
	if err != nil {
		t.Fatalf("atualizar agendamento com fluxo: %v", err)
	}
	if len(updated.Steps) != 3 {
		t.Fatalf("esperava 3 passos após update, obteve %d", len(updated.Steps))
	}

	broken := updated
	broken.Steps = []ScheduleStep{
		flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
		flowStep("ciclo-a", ScheduleStepActionMission, []string{"ciclo-b"}, "objetivo a", ""),
		flowStep("ciclo-b", ScheduleStepActionMission, []string{"ciclo-a"}, "objetivo b", ""),
	}
	if _, err := store.UpdateSchedule(created.ID, broken); err == nil {
		t.Fatal("update com ciclo deveria ser rejeitado")
	}
	stillThere, err := store.GetSchedule(created.ID)
	if err != nil {
		t.Fatalf("recarregar após rejeição: %v", err)
	}
	if len(stillThere.Steps) != 3 {
		t.Fatalf("grafo válido não deveria ser sobrescrito por um inválido: %+v", stillThere.Steps)
	}
}
