package agent

import (
	"context"
	"path/filepath"
	"testing"
)

// TestRuntimeDeclaraAOrigemDoConsumo prova a opcao (a) medida em E-241: em vez de fingir custo POR ETAPA (rateio
// arbitrario), o registro da missao DECLARA de onde o consumo veio. Hoje existe uma unica origem: o planejamento,
// a chamada que produz o plano inteiro.
//
// O teste cobre os DOIS sentidos: quando ha consumo, a origem e declarada; quando NAO ha consumo, nenhuma origem e
// inventada — declarar origem para custo inexistente seria o mesmo defeito, ao contrario.
func TestRuntimeDeclaraAOrigemDoConsumo(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(filepath.Join(root, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	planner := &planejadorComConsumo{price: ModelPrice{InputCentsPer1K: 6, OutputCentsPer1K: 12, Known: true}}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Planner: planner, WorkspaceRoot: root, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:    "medir a origem do consumo",
		Capabilities: []string{"workspace:read"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if missao.Usage == nil || missao.Usage.Calls < 1 {
		t.Fatalf("o consumo do planejamento deveria estar registrado: %+v", missao.Usage)
	}
	if missao.UsageOrigin != UsageOriginPlanning {
		t.Fatalf("origem do consumo deveria ser %q, veio %q", UsageOriginPlanning, missao.UsageOrigin)
	}

	// Persistido: a origem precisa sobreviver ao registro, nao apenas existir na resposta.
	recarregada, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recarregada.UsageOrigin != UsageOriginPlanning {
		t.Fatalf("origem do consumo nao sobreviveu ao registro: %q", recarregada.UsageOrigin)
	}
	t.Logf("ORIGEM DO CONSUMO: origem=%q chamadas=%d tokens=%d custo=%d",
		recarregada.UsageOrigin, recarregada.Usage.Calls, recarregada.Usage.PromptTokens, recarregada.Usage.CostCents)
}

// TestRuntimeNaoDeclaraOrigemSemConsumo garante o outro lado: sem consumo reportado, nada de origem declarada.
func TestRuntimeNaoDeclaraOrigemSemConsumo(t *testing.T) {
	root := t.TempDir()
	planner := fixedPlanner{steps: []Step{{
		ID: "step_1", Kind: "workspace.list", Title: "listar", Risk: RiskRead, State: StepPending,
		Input: map[string]any{"path": ".", "max_entries": 5},
	}}}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: planner, WorkspaceRoot: root, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:    "sem consumo reportado",
		Capabilities: []string{"workspace:read"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if missao.Usage != nil {
		t.Fatalf("planejador que nao reporta consumo nao pode gerar Usage: %+v", missao.Usage)
	}
	if missao.UsageOrigin != "" {
		t.Fatalf("sem consumo nao pode haver origem declarada, veio %q", missao.UsageOrigin)
	}
	t.Logf("SEM CONSUMO: origem=%q usage=%v", missao.UsageOrigin, missao.Usage)
}
