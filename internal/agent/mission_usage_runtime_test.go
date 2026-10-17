package agent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ollama/ollama/api"
)

// planejadorComConsumo simula um planejador real: recebe o destino injetado pelo runtime e
// publica os contadores da resposta do modelo. Existe para provar que o consumo CHEGA ao
// registro da missão, e não apenas que a struct tem o campo.
type planejadorComConsumo struct {
	sink  *ModelUsage
	price ModelPrice
}

func (p *planejadorComConsumo) SetPlanUsageSink(sink *ModelUsage) { p.sink = sink }

func (p *planejadorComConsumo) Plan(_ context.Context, mission Mission) ([]Step, error) {
	if p.sink != nil {
		// O planejador real conhece o provedor/modelo que chamou: a identidade vem daqui,
		// nao da requisicao da missao.
		*p.sink = UsageFromMetrics(api.Metrics{PromptEvalCount: 1200, EvalCount: 300}, "groq", "llama-3.1-8b", p.price)
	}
	return []Step{{
		ID:    "step_1",
		Kind:  "workspace.list",
		Title: "Inspecionar o workspace autorizado",
		Risk:  RiskRead,
		State: StepPending,
		Input: map[string]any{"path": ".", "max_entries": 10},
	}}, nil
}

func TestRuntimePersisteConsumoDoModeloNaMissao(t *testing.T) {
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
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		// Sem Provider: o runtime valida provedor declarado contra os provedores configurados, e
		// este teste prova a PERSISTENCIA do consumo, nao o roteamento de provedor.
		Objective: "inspecionar o workspace",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mission.Usage == nil {
		t.Fatal("consumo do modelo nao chegou ao registro da missao")
	}
	// 1200 tokens x 6c/1K = 8 centavos (ceil de 7,2) e 300 x 12c/1K = 4 centavos (ceil de 3,6).
	if mission.Usage.PromptTokens != 1200 || mission.Usage.CompletionTokens != 300 || mission.Usage.Calls != 1 {
		t.Fatalf("contadores incorretos: %+v", mission.Usage)
	}
	if !mission.Usage.CostKnown || mission.Usage.CostCents != 12 {
		t.Fatalf("custo incorreto: %+v", mission.Usage)
	}
	if mission.Usage.Provider != "groq" || mission.Usage.Model != "llama-3.1-8b" {
		t.Fatalf("identidade do consumo incorreta: %+v", mission.Usage)
	}

	// A prova que importa: o consumo sobrevive à releitura do store.
	recarregada, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recarregada.Usage == nil {
		t.Fatal("consumo NAO foi persistido com a missao")
	}
	if recarregada.Usage.PromptTokens != 1200 || recarregada.Usage.CompletionTokens != 300 || recarregada.Usage.CostCents != 12 {
		t.Fatalf("consumo persistido diferente do observado: %+v", recarregada.Usage)
	}
}

func TestRuntimeNaoInventaConsumoQuandoOPlanejadorNaoReporta(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(filepath.Join(root, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	// RulePlanner nao reporta consumo: a missao NAO pode ganhar um registro de custo inventado.
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Planner: RulePlanner{}, WorkspaceRoot: root, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "inspecionar o workspace"})
	if err != nil {
		t.Fatal(err)
	}
	if mission.Usage != nil {
		t.Fatalf("planejador sem reporte nao pode produzir consumo: %+v", mission.Usage)
	}
	recarregada, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recarregada.Usage != nil {
		t.Fatalf("consumo inventado sobreviveu a persistencia: %+v", recarregada.Usage)
	}
}
