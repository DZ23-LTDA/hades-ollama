package agent

import (
	"context"
	"testing"
	"time"
)

// E-414 / ADV-62 — UM CICLO COM `NextRunAt` NO FUTURO NÃO PODE AVANÇAR NO TICK.
//
// **MEDIDO (6ª auditoria independente)**: `supervisor.go:400-403` tem o portão
// `if !cycle.Enabled || cycle.NextRunAt.After(now)`. O **único** teste de ciclo
// (`supervisor_test.go:106 TestSupervisorCompanyCycleAdvances`) usa `NextRunAt: past` ⇒ *remover a metade
// `NextRunAt.After(now)` da condição **não fazia nenhum teste falhar*** ✗. A sonda do auditor mostrou o
// defeito real: com a mutação, um ciclo com `NextRunAt=+24h` produzia `CyclesAdvanced=1` e **1 missão criada**
// (no original: 0 e 0).
//
// *O contraste é o que condena:* ignorar `NextRunAt` no **claim de workflow** (`context.go:1196`) **falha dois
// testes** — *o mesmo conceito, defendido num caminho e não no outro.*
//
// Este teste é a sonda transformada em prova: **a remoção do portão tem de doer.**
func TestCicloComProximaExecucaoNoFuturoNaoAvancaNoTick(t *testing.T) {
	runtime, compStore, _ := setupTestSupervisorRuntime(t)
	ctx := context.Background()

	now := time.Now().UTC()
	futuro := now.Add(24 * time.Hour)

	comp, err := compStore.Create(Company{Name: "Tech Corp", OrganizationID: LocalOrganizationID})
	if err != nil {
		t.Fatalf("create company: %v", err)
	}
	_, err = compStore.AddCycle(comp.ID, CompanyCycle{
		ID:              "cycle_futuro_1",
		Name:            "Ciclo agendado para amanhã",
		Objective:       "Construir módulo de relatórios e executar testes unitários",
		Frequency:       "daily",
		IntervalSeconds: 86400,
		Enabled:         true,
		NextRunAt:       futuro, // NÃO vencido
	})
	if err != nil {
		t.Fatalf("add cycle: %v", err)
	}

	sup := runtime.Supervisor()
	res, err := sup.Tick(ctx, now)
	if err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	// (a) O CICLO NÃO PODE AVANÇAR. Sem esta asserção, remover `NextRunAt.After(now)` de supervisor.go:401
	// passa em silêncio — foi o que o auditor mediu.
	if res.CyclesAdvanced != 0 {
		t.Fatalf("ARMADILHA: um ciclo com NextRunAt no FUTURO avançou %d vez(es) — o portão por vencimento "+
			"(supervisor.go:401 `NextRunAt.After(now)`) está sem defesa.", res.CyclesAdvanced)
	}

	// (b) E o ESTADO não pode ter sido tocado: nem relógio nem marca de última execução.
	depois, err := compStore.Get(comp.ID)
	if err != nil {
		t.Fatalf("get company: %v", err)
	}
	if len(depois.Cycles) != 1 {
		t.Fatalf("esperava 1 ciclo, veio %d", len(depois.Cycles))
	}
	ciclo := depois.Cycles[0]
	if !ciclo.NextRunAt.Equal(futuro) {
		t.Fatalf("ARMADILHA: o NextRunAt de um ciclo NÃO vencido mudou: era %v, ficou %v — *o agendamento foi "+
			"consumido antes da hora*", futuro, ciclo.NextRunAt)
	}
	if ciclo.LastRunAt != nil {
		t.Fatalf("ARMADILHA: um ciclo não vencido registrou LastRunAt=%v — ele rodou antes do tempo", ciclo.LastRunAt)
	}

	// (c) O CONTROLE NÃO É AQUI. *Tentei acrescentar um ciclo vencido **depois** do primeiro tick e ele não
	// avançou* ✗ — a lista de empresas é lida/cacheada por tick, então um ciclo criado no meio não é visto.
	// Em vez de forçar o cenário, aponto o controle que **já existe e está verde**:
	// `TestSupervisorCompanyCycleAdvances` (`supervisor_test.go:106`) usa `NextRunAt: past` e exige
	// `CyclesAdvanced >= 1` ⇒ *a asserção (a) acima distingue mesmo os dois casos, e o controle é um teste
	// próprio — com um propósito, não com dois.*
}
