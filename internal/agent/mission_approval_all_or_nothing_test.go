package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestRuntimeNaoProduzEfeitoComAprovacaoPendenteNaNada cobre o caso que os testes de missão multi-etapa NÃO
// cobriam: a cadeia de aprovações era sempre feliz (todas aprovadas de uma vez), o que deixava em aberto uma
// pergunta perigosa — *e se o humano aprovar uma escrita e só depois negar a outra?*
//
// MEDIÇÃO (que definiu este teste): o runtime só executa quando NÃO há decisão pendente. A cadeia é TUDO-OU-NADA.
// Portanto:
//
//  1. com a primeira escrita aprovada e a segunda PENDENTE, executar NÃO escreve nada — o plano inteiro espera a
//     autorização completa (nada de efeito parcial escapando pelo meio);
//  2. negada a segunda escrita, a missão fica terminal e NENHUM arquivo é criado e NENHUM artefato é registrado;
//  3. a etapa aprovada não aparece como concluída sem ter executado — o estado não mente.
//
// Notas de medição que mudaram o desenho (registradas para não se perderem):
//   - travessia de caminho (`../fora.txt`) é recusada no PREPARO DA APROVAÇÃO, antes de a missão existir;
//   - decidir as duas etapas na MESMA rodada (uma aprovada, outra negada) também deixa a primeira em PENDING.
func TestRuntimeNaoProduzEfeitoComAprovacaoPendenteNaNada(t *testing.T) {
	root := t.TempDir()
	planner := fixedPlanner{steps: []Step{
		{
			ID: "step_1", Kind: "workspace.write", Title: "escrever a.txt", Risk: RiskWrite, RequiresApproval: true, State: StepPending,
			Input: map[string]any{"path": "a.txt", "content": "primeiro\n"},
		},
		{
			ID: "step_2", Kind: "workspace.write", Title: "escrever b.txt", Risk: RiskWrite, RequiresApproval: true, State: StepPending,
			Input: map[string]any{"path": "b.txt", "content": "segundo\n"},
		},
	}}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: planner, WorkspaceRoot: root, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	missao, err := runtime.CreateMission(ctx, CreateMissionRequest{
		Objective:    "escrever dois arquivos, aprovando o primeiro e negando o segundo",
		Capabilities: []string{"workspace:read", "workspace:write"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(missao.Approvals) != 2 {
		t.Fatalf("esperava 2 aprovações (uma por escrita), veio %d", len(missao.Approvals))
	}

	decidir := func(stepID string, aprovar bool) {
		t.Helper()
		atual, err := runtime.GetMission(missao.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, aprovacao := range atual.Approvals {
			if aprovacao.StepID != stepID || aprovacao.Status != ApprovalPending {
				continue
			}
			if _, err := runtime.DecideApproval(missao.ID, aprovacao.ID, aprovar, "decisão do teste de cadeia tudo-ou-nada"); err != nil {
				t.Fatalf("decisão de %s falhou: %v", stepID, err)
			}
			return
		}
		t.Fatalf("não encontrei aprovação pendente para %s", stepID)
	}

	// 1. Só a primeira escrita aprovada: executar NÃO pode produzir efeito nenhum enquanto a segunda decide.
	decidir("step_1", true)
	if err := runtime.Run(ctx, missao.ID); err != nil {
		t.Fatalf("execução com decisão pendente falhou: %v", err)
	}
	parcial, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); err == nil {
		t.Fatal("com a segunda escrita PENDENTE, nada podia ter sido escrito — a cadeia é tudo-ou-nada")
	}
	if len(parcial.Artifacts) != 0 {
		t.Fatalf("com decisão pendente não pode existir artefato: %+v", parcial.Artifacts)
	}
	for _, etapa := range parcial.Plan {
		if etapa.State == StepSucceeded {
			t.Fatalf("nenhuma etapa pode aparecer concluída antes da cadeia fechar: %+v", etapa)
		}
	}

	// 2. Segunda escrita NEGADA: missão terminal, nada escrito, nada registrado.
	decidir("step_2", false)
	_ = runtime.Run(ctx, missao.ID) // terminal é esperado; o estado real é medido abaixo.
	final, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.State == MissionCompleted {
		t.Fatalf("missão com escrita NEGADA não pode concluir: %+v", final)
	}
	for _, nome := range []string{"a.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(root, nome)); err == nil {
			t.Fatalf("escrita negada (ou pendente) não pode criar %s", nome)
		}
	}
	if len(final.Artifacts) != 0 {
		t.Fatalf("escrita negada não pode registrar artefato: %+v", final.Artifacts)
	}
	porID := make(map[string]Step, len(final.Plan))
	for _, etapa := range final.Plan {
		porID[etapa.ID] = etapa
	}
	if porID["step_1"].State == StepSucceeded {
		t.Fatalf("etapa que nunca executou não pode constar como concluída: %+v", porID["step_1"])
	}

	t.Logf("CADEIA TUDO-OU-NADA: missao=%s etapa1=%s etapa2=%s arquivos=0 artefatos=%d",
		final.State, porID["step_1"].State, porID["step_2"].State, len(final.Artifacts))
}
