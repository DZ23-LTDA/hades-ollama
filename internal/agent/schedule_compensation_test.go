package agent

import (
	"context"
	"strings"
	"testing"
)

// TestCompensacaoDesfazEmOrdemInversaQuandoPassoPosteriorFalha prova a compensação (saga) no motor de fluxo:
// quando um passo falha de forma DEFINITIVA, o que já teve efeito é DESFEITO, e em ORDEM INVERSA de execução —
// desfazer na ordem em que executou deixaria o estado no meio do caminho. A falha continua sendo falha.
func TestCompensacaoDesfazEmOrdemInversaQuandoPassoPosteriorFalha(t *testing.T) {
	runtime, _, _ := setupTestSupervisorRuntime(t)

	compensaA := flowStep("compensa-a", ScheduleStepActionMission, []string{"gatilho"}, "Desfazer efeito do passo A", "")
	compensaB := flowStep("compensa-b", ScheduleStepActionMission, []string{"gatilho"}, "Desfazer efeito do passo B", "")
	passoA := flowStep("a", ScheduleStepActionMission, []string{"gatilho"}, "Passo A com efeito real", "")
	passoA.CompensateWith = "compensa-a"
	passoB := flowStep("b", ScheduleStepActionMission, []string{"a"}, "Passo B com efeito real", "")
	passoB.CompensateWith = "compensa-b"
	// Falha DETERMINÍSTICA: objetivo com variável que ninguém definiu.
	falha := flowStep("falha", ScheduleStepActionMission, []string{"b"}, "Publicar {{variavel-inexistente}}", "")

	schedule := Schedule{
		Objective:      "Fluxo com compensação",
		OrganizationID: LocalOrganizationID,
		Steps: []ScheduleStep{
			flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
			passoA, passoB, compensaA, compensaB, falha,
		},
	}

	report, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err == nil {
		t.Fatal("falha definitiva de um passo precisa continuar sendo falha, mesmo com compensação")
	}

	posicao := make(map[string]int, len(report.Steps))
	status := make(map[string]string, len(report.Steps))
	motivo := make(map[string]string, len(report.Steps))
	for i, resultado := range report.Steps {
		if _, repetido := posicao[resultado.ID]; !repetido {
			posicao[resultado.ID] = i
		}
		status[resultado.ID] = resultado.Status
		motivo[resultado.ID] = resultado.Reason
	}

	if status["falha"] != ScheduleStepFailed {
		t.Fatalf("o passo que falhou precisa estar marcado como falha: %+v", report.Steps)
	}
	if status["a"] != ScheduleStepSucceeded || status["b"] != ScheduleStepSucceeded {
		t.Fatalf("os passos com efeito precisam ter sucesso antes da compensação: %+v", report.Steps)
	}
	// A ORDEM INVERSA é o que prova a compensação: B (executado depois) é desfeito ANTES de A.
	for _, id := range []string{"compensa-a", "compensa-b"} {
		if status[id] != ScheduleStepCompensated {
			t.Fatalf("compensação %q precisa aparecer como aplicada, veio status=%q: %+v", id, status[id], report.Steps)
		}
		if !strings.Contains(motivo[id], "compensação de") {
			t.Fatalf("compensação %q precisa registrar de QUEM é o desfazer, veio %q", id, motivo[id])
		}
	}
	// A ORDEM que prova a compensação é a do PASSE DE COMPENSAÇÃO (status "compensated"), não a do passe
	// normal: os passos de compensação são alcançáveis no grafo e por isso também aparecem antes da falha.
	passeCompensacao := make(map[string]int, 2)
	for i, resultado := range report.Steps {
		if resultado.Status == ScheduleStepCompensated {
			passeCompensacao[resultado.ID] = i
		}
	}
	if len(passeCompensacao) != 2 {
		t.Fatalf("esperava 2 registros de compensação, veio %v (passos=%+v)", passeCompensacao, report.Steps)
	}
	if passeCompensacao["compensa-b"] >= passeCompensacao["compensa-a"] {
		t.Fatalf("compensação precisa rodar em ORDEM INVERSA: compensa-b@%d deveria vir antes de compensa-a@%d",
			passeCompensacao["compensa-b"], passeCompensacao["compensa-a"])
	}
	_ = posicao

	// O efeito do desfazer é REAL: as missões de compensação existem no runtime.
	criadas, err := runtime.ListMissions()
	if err != nil {
		t.Fatalf("listar missões: %v", err)
	}
	objetivos := make(map[string]bool, len(criadas))
	for _, missao := range criadas {
		objetivos[missao.Objective] = true
	}
	for _, esperado := range []string{"Desfazer efeito do passo A", "Desfazer efeito do passo B"} {
		if !objetivos[esperado] {
			t.Fatalf("missão de compensação %q não foi criada; criadas: %v", esperado, objetivos)
		}
	}

	t.Logf("COMPENSAÇÃO: ordem=%s → %s · falha=%s · missões_de_compensação=2",
		"compensa-b", "compensa-a", status["falha"])
}

// TestCompensacaoAusenteViraIncidente prova que compensação que NÃO consegue rodar não desaparece: vira
// registro de falha de compensação + dead letter. Silêncio aqui esconderia dano residual.
func TestCompensacaoAusenteViraIncidente(t *testing.T) {
	runtime, _, _ := setupTestSupervisorRuntime(t)

	passoA := flowStep("a", ScheduleStepActionMission, []string{"gatilho"}, "Passo A com efeito real", "")
	passoA.CompensateWith = "compensacao-que-nao-existe"
	falha := flowStep("falha", ScheduleStepActionMission, []string{"a"}, "Publicar {{variavel-inexistente}}", "")

	schedule := Schedule{
		Objective:      "Fluxo com compensação inválida",
		OrganizationID: LocalOrganizationID,
		Steps: []ScheduleStep{
			flowStep("gatilho", ScheduleStepTriggerInterval, nil, "", ""),
			passoA, falha,
		},
	}

	report, err := runtime.ExecuteScheduleFlowReport(context.Background(), schedule)
	if err == nil {
		t.Fatal("a falha do passo continua sendo falha")
	}

	var encontrado bool
	for _, resultado := range report.Steps {
		if resultado.ID == "compensacao-que-nao-existe" && resultado.Status == ScheduleStepCompensationFailed {
			encontrado = true
			if !strings.Contains(resultado.Reason, "inexistente") {
				t.Fatalf("motivo precisa explicar a referência inválida, veio %q", resultado.Reason)
			}
		}
	}
	if !encontrado {
		t.Fatalf("compensação inválida precisa virar falha de compensação registrada: %+v", report.Steps)
	}
	if len(report.DeadLetters) == 0 {
		t.Fatal("compensação que não rodou precisa virar incidente (dead letter), não silêncio")
	}
	t.Logf("COMPENSAÇÃO AUSENTE: incidentes=%d motivo=%q", len(report.DeadLetters), report.DeadLetters[0].Reason)
}
