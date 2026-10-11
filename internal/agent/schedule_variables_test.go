package agent

import (
	"context"
	"strings"
	"testing"
)

// Vocabulário de nós ampliado (estágio 13): variáveis/transformações e laços
// LIMITADOS. As duas regras que estes testes travam: variável desconhecida é
// ERRO (nunca string vazia) e laço sem teto não existe.
func TestExpandScheduleTemplateFailsClosedOnUnknownVariable(t *testing.T) {
	vars := map[string]string{"item": "caneta", "index": "0"}
	expanded, err := expandScheduleTemplate("Repor {{item}} (posição {{index}})", vars)
	if err != nil {
		t.Fatalf("variáveis definidas deveriam expandir: %v", err)
	}
	if expanded != "Repor caneta (posição 0)" {
		t.Fatalf("expansão errada: %q", expanded)
	}
	// Sem aspas, sem chaves aninhadas, sem expressão: só {{nome}}.
	if _, err := expandScheduleTemplate("olá {{desconhecida}}", vars); err == nil {
		t.Fatal("variável desconhecida precisa ser ERRO, não valor vazio")
	}
	plain, err := expandScheduleTemplate("texto sem variável", vars)
	if err != nil || plain != "texto sem variável" {
		t.Fatalf("texto sem variável não muda: %q %v", plain, err)
	}
}

func TestSetScheduleVariableEnforcesLimits(t *testing.T) {
	vars := map[string]string{}
	if err := setScheduleVariable(vars, "resumo", "ok"); err != nil {
		t.Fatalf("nome válido deveria ser aceito: %v", err)
	}
	if _, err := expandScheduleTemplate("{{resumo}}", vars); err != nil {
		t.Fatal(err)
	}
	if err := setScheduleVariable(vars, "nome inválido", "x"); err == nil {
		t.Fatal("nome com espaço precisa ser recusado")
	}
	if err := setScheduleVariable(vars, "grande", strings.Repeat("a", maxScheduleVariableSize+1)); err == nil {
		t.Fatal("valor acima do teto precisa ser recusado, não truncado")
	}
	full := map[string]string{}
	for i := range maxScheduleVariables {
		if err := setScheduleVariable(full, "v"+string(rune('a'+i%26))+string(rune('a'+i/26)), "x"); err != nil {
			t.Fatalf("entrada %d deveria caber: %v", i, err)
		}
	}
	if err := setScheduleVariable(full, "extra", "x"); err == nil {
		t.Fatal("escopo cheio precisa ser recusado")
	}
}

func TestScheduleTransformFeedsLaterStepObjective(t *testing.T) {
	runtime, contextStore, project := idempotencyRuntime(t)
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-vars",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		ProjectID:       project.ID,
		IntervalSeconds: 60,
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "resumo", Kind: ScheduleStepActionTransform, DependsOn: []string{"gatilho"}, Source: "rotina {{schedule.id}}", Target: "resumo"},
			{ID: "acao", Kind: ScheduleStepActionMission, DependsOn: []string{"resumo"}, Objective: "missão com {{resumo}}"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr != nil {
		t.Fatalf("fluxo com transformação deveria rodar: %v", runErr)
	}
	if report.Missions != 1 {
		t.Fatalf("esperava 1 missão, veio %d (%+v)", report.Missions, report.Steps)
	}
	// E a variável realmente chegou ao objetivo da missão criada.
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 1 || !strings.Contains(missions[0].Objective, "rotina sched-vars") {
		t.Fatalf("o objetivo da missão precisa conter a variável expandida: %+v", missions)
	}
	var transformReason string
	for _, step := range report.Steps {
		if step.ID == "resumo" {
			transformReason = step.Reason
		}
	}
	if !strings.Contains(transformReason, "resumo") {
		t.Fatalf("o passo de transformação precisa informar o que publicou: %q", transformReason)
	}
}

func TestScheduleTransformFailsWhenVariableIsUnknown(t *testing.T) {
	runtime, contextStore, project := idempotencyRuntime(t)
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-var-erro",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		ProjectID:       project.ID,
		IntervalSeconds: 60,
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{ID: "quebra", Kind: ScheduleStepActionTransform, DependsOn: []string{"gatilho"}, Source: "{{nao_existe}}", Target: "x"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr == nil {
		t.Fatal("variável desconhecida precisa falhar o fluxo")
	}
	if len(report.DeadLetters) != 1 || report.DeadLetters[0].StepID != "quebra" {
		t.Fatalf("a falha de transformação precisa entrar na fila morta: %+v", report.DeadLetters)
	}
	if !strings.Contains(runErr.Error(), "não definida") {
		t.Fatalf("o erro precisa dizer o que faltou: %v", runErr)
	}
}

func TestScheduleLoopCreatesOneMissionPerItemAndIsIdempotent(t *testing.T) {
	runtime, contextStore, project := idempotencyRuntime(t)
	schedule, err := contextStore.CreateSchedule(Schedule{
		ID:              "sched-loop",
		Objective:       "corpo",
		OrganizationID:  "org-a",
		ProjectID:       project.ID,
		IntervalSeconds: 60,
		Steps: []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			{
				ID: "lote", Kind: ScheduleStepLogicLoop, DependsOn: []string{"gatilho"},
				Objective: "processar {{item}} na posição {{index}}",
				Items:     []string{"alfa", "beta", "gama"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	report, runErr := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if runErr != nil {
		t.Fatalf("laço limitado deveria rodar: %v", runErr)
	}
	if report.Missions != 3 {
		t.Fatalf("esperava 1 missão por item, veio %d", report.Missions)
	}
	var children int
	for _, step := range report.Steps {
		if step.ID == "lote" {
			children = len(step.Children)
		}
	}
	if children != 3 {
		t.Fatalf("a evidência precisa ter uma entrada POR ITERAÇÃO, veio %d", children)
	}
	missions, err := runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 3 {
		t.Fatalf("esperava 3 missões, veio %d", len(missions))
	}
	objectives := map[string]bool{}
	for _, mission := range missions {
		objectives[mission.Objective] = true
	}
	for _, expected := range []string{"processar alfa na posição 0", "processar beta na posição 1", "processar gama na posição 2"} {
		if !objectives[expected] {
			t.Fatalf("objetivo %q não foi criado; veio %v", expected, objectives)
		}
	}

	// Repetir a MESMA execução não duplica nenhuma iteração do laço.
	schedule.LastRun = &report
	second, err := runtime.ReprocessScheduleFlow(context.Background(), schedule)
	if err != nil {
		t.Fatalf("reprocessamento do laço: %v", err)
	}
	if second.Missions != 0 {
		t.Fatalf("reprocessar não pode recriar itens do laço, veio %d", second.Missions)
	}
	missions, err = runtime.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 3 {
		t.Fatalf("o laço duplicou missões no reprocessamento: %d", len(missions))
	}
}

func TestValidateScheduleLoopRejectsUnboundedLists(t *testing.T) {
	items := make([]string, maxScheduleLoopItems+1)
	for i := range items {
		items[i] = "item"
	}
	steps := []ScheduleStep{
		{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
		{ID: "lote", Kind: ScheduleStepLogicLoop, DependsOn: []string{"gatilho"}, Objective: "{{item}}", Items: items},
	}
	if err := ValidateScheduleSteps(steps); err == nil {
		t.Fatalf("lista acima de %d itens precisa ser recusada antes de persistir", maxScheduleLoopItems)
	}
	// Lista vazia também é recusada: laço sem item é laço que não faz nada.
	steps[1].Items = nil
	if err := ValidateScheduleSteps(steps); err == nil {
		t.Fatal("laço sem item precisa ser recusado")
	}
	// Item vazio é recusado.
	steps[1].Items = []string{"ok", "  "}
	if err := ValidateScheduleSteps(steps); err == nil {
		t.Fatal("item vazio precisa ser recusado")
	}
}

func TestValidateScheduleTransformRejectsBadShapes(t *testing.T) {
	base := func(step ScheduleStep) []ScheduleStep {
		return []ScheduleStep{
			{ID: "gatilho", Kind: ScheduleStepTriggerInterval},
			step,
		}
	}
	valid := ScheduleStep{ID: "t", Kind: ScheduleStepActionTransform, DependsOn: []string{"gatilho"}, Source: "{{schedule.id}}", Target: "resumo"}
	if err := ValidateScheduleSteps(base(valid)); err != nil {
		t.Fatalf("transformação válida deveria ser aceita: %v", err)
	}
	semSource := valid
	semSource.Source = ""
	if err := ValidateScheduleSteps(base(semSource)); err == nil {
		t.Fatal("transformação sem source precisa ser recusada")
	}
	alvoInvalido := valid
	alvoInvalido.Target = "nome inválido"
	if err := ValidateScheduleSteps(base(alvoInvalido)); err == nil {
		t.Fatal("alvo inválido precisa ser recusado")
	}
	comObjetivo := valid
	comObjetivo.Objective = "não pode"
	if err := ValidateScheduleSteps(base(comObjetivo)); err == nil {
		t.Fatal("transformação não carrega objetivo")
	}
}
