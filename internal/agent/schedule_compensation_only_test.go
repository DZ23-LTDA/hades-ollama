package agent

import (
	"context"
	"strings"
	"testing"
)

// TestPassoSomenteCompensacaoNaoRodaNoFluxoNormal fecha o limite declarado em E-237: o passo de compensação
// precisa ser alcançável (o validador é fail-closed), mas NÃO pode fazer parte do fluxo normal. Sem a marca, ele
// rodaria duas vezes: uma no passe normal e outra no desfazer.
func TestPassoSomenteCompensacaoNaoRodaNoFluxoNormal(t *testing.T) {
	runtime, _, _ := setupTestSupervisorRuntime(t)

	passoA := flowStep("a", ScheduleStepActionMission, []string{"gatilho"}, "Passo A com efeito real", "")
	passoA.CompensateWith = "desfaz-a"
	desfazA := flowStep("desfaz-a", ScheduleStepActionMission, []string{"gatilho"}, "Desfazer o efeito do passo A", "")
	desfazA.CompensationOnly = true
	falha := flowStep("falha", ScheduleStepActionMission, []string{"a"}, "Publicar {{variavel-inexistente}}", "")

	schedule := Schedule{
		Objective:      "Fluxo com passo somente de compensação",
		OrganizationID: LocalOrganizationID,
		Steps: []ScheduleStep{
			flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
			passoA, desfazA, falha,
		},
	}

	report, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err == nil {
		t.Fatal("a falha do passo continua sendo falha")
	}

	var noPasseNormal string
	var noDesfazer string
	for _, resultado := range report.Steps {
		if resultado.ID != "desfaz-a" {
			continue
		}
		switch resultado.Status {
		case ScheduleStepSkipped:
			noPasseNormal = resultado.Reason
		case ScheduleStepCompensated:
			noDesfazer = resultado.Reason
		}
	}
	if !strings.Contains(noPasseNormal, "somente de compensação") {
		t.Fatalf("no fluxo normal o passo precisa ser registrado como ignorado por ser somente de compensação, veio %q", noPasseNormal)
	}
	if noDesfazer == "" {
		t.Fatalf("no desfazer o passo precisa ter rodado de verdade: %+v", report.Steps)
	}

	// A PROVA do efeito único: a missão de desfazer existe UMA vez. Sem a marca seriam duas.
	criadas, err := runtime.ListMissions()
	if err != nil {
		t.Fatalf("listar missões: %v", err)
	}
	quantos := 0
	for _, missao := range criadas {
		if missao.Objective == "Desfazer o efeito do passo A" {
			quantos++
		}
	}
	if quantos != 1 {
		t.Fatalf("o desfazer precisa acontecer EXATAMENTE uma vez (só no passe de compensação), veio %d", quantos)
	}
	t.Logf("SOMENTE COMPENSAÇÃO: passe_normal=%q · desfazer=%q · missões_de_desfazer=%d", noPasseNormal, noDesfazer, quantos)
}

// TestValidacaoRecusaSomenteCompensacaoInvalido mantém o validador FAIL-CLOSED: a marca nova não pode virar
// brecha. Passo marcado sem quem o use, dependência nele, e gatilho marcado são todos recusados.
func TestValidacaoRecusaSomenteCompensacaoInvalido(t *testing.T) {
	casos := []struct {
		nome     string
		esperado string
		montar   func() []ScheduleStep
	}{
		{
			nome:     "marcado sem quem o use",
			esperado: "no step compensates with it",
			montar: func() []ScheduleStep {
				desfaz := flowStep("desfaz", ScheduleStepActionMission, []string{"gatilho"}, "Desfazer", "")
				desfaz.CompensationOnly = true
				return []ScheduleStep{
					flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
					flowStep("a", ScheduleStepActionMission, []string{"gatilho"}, "Passo A", ""),
					desfaz,
				}
			},
		},
		{
			nome:     "outro passo depende dele",
			esperado: "cannot depend on compensation-only",
			montar: func() []ScheduleStep {
				desfaz := flowStep("desfaz", ScheduleStepActionMission, []string{"gatilho"}, "Desfazer", "")
				desfaz.CompensationOnly = true
				passoA := flowStep("a", ScheduleStepActionMission, []string{"desfaz"}, "Passo A", "")
				passoA.CompensateWith = "desfaz"
				return []ScheduleStep{
					flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
					desfaz, passoA,
				}
			},
		},
		{
			nome:     "gatilho marcado",
			esperado: "cannot be compensation-only",
			montar: func() []ScheduleStep {
				gatilho := flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", "")
				gatilho.CompensationOnly = true
				passoA := flowStep("a", ScheduleStepActionMission, []string{"gatilho"}, "Passo A", "")
				passoA.CompensateWith = "gatilho"
				return []ScheduleStep{gatilho, passoA}
			},
		},
	}

	for _, caso := range casos {
		err := ValidateScheduleSteps(caso.montar())
		if err == nil {
			t.Fatalf("%s: esperava recusa, veio nil", caso.nome)
		}
		if !strings.Contains(err.Error(), caso.esperado) {
			t.Fatalf("%s: esperava motivo contendo %q, veio %q", caso.nome, caso.esperado, err.Error())
		}
	}
	t.Logf("VALIDAÇÃO FAIL-CLOSED: %d casos de somente-compensação inválido recusados", len(casos))
}
