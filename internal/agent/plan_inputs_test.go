package agent

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePlanStepInputs(t *testing.T) {
	descritor := ToolDescriptor{Name: "workspace.read", RequiredInputs: []string{"path"}}
	casos := []struct {
		nome       string
		step       Step
		desc       ToolDescriptor
		querErro   bool
		querTrecho string
	}{
		{
			nome:     "input vazio e recusado",
			step:     Step{ID: "step_1", Kind: "workspace.read", Input: map[string]any{}},
			desc:     descritor,
			querErro: true, querTrecho: "path",
		},
		{
			nome:     "campo ausente e recusado",
			step:     Step{ID: "step_2", Kind: "workspace.read", Input: map[string]any{"outro": "x"}},
			desc:     descritor,
			querErro: true, querTrecho: "step_2",
		},
		{
			nome:     "string em branco nao ancora caminho e e recusada",
			step:     Step{ID: "step_3", Kind: "workspace.read", Input: map[string]any{"path": "   "}},
			desc:     descritor,
			querErro: true, querTrecho: "path",
		},
		{
			nome:     "campo nulo e recusado",
			step:     Step{ID: "step_4", Kind: "workspace.read", Input: map[string]any{"path": nil}},
			desc:     descritor,
			querErro: true, querTrecho: "path",
		},
		{
			nome:     "campo presente e aceito",
			step:     Step{ID: "step_5", Kind: "workspace.read", Input: map[string]any{"path": "README.md"}},
			desc:     descritor,
			querErro: false,
		},
		{
			nome:     "descritor sem exigencia aceita input vazio",
			step:     Step{ID: "step_6", Kind: "workspace.list", Input: map[string]any{}},
			desc:     ToolDescriptor{Name: "workspace.list"},
			querErro: false,
		},
	}
	for _, caso := range casos {
		err := ValidatePlanStepInputs(caso.step, caso.desc)
		if caso.querErro && err == nil {
			t.Fatalf("%s: esperava erro e nao houve", caso.nome)
		}
		if !caso.querErro && err != nil {
			t.Fatalf("%s: erro inesperado: %v", caso.nome, err)
		}
		if caso.querErro && err != nil && !strings.Contains(err.Error(), caso.querTrecho) {
			t.Fatalf("%s: erro %q nao contem %q", caso.nome, err.Error(), caso.querTrecho)
		}
	}
}

// planejadorComInputFaltando devolve um plano plausível e inútil: exatamente o que um modelo real
// pode devolver (observado no teste opt-in com qwen2.5:7b).
type planejadorComInputFaltando struct{}

func (planejadorComInputFaltando) Plan(context.Context, Mission) ([]Step, error) {
	return []Step{{
		ID:    "step_1",
		Kind:  "workspace.read",
		Title: "Ler arquivo do workspace",
		Risk:  RiskRead,
		State: StepPending,
		Input: map[string]any{},
	}}, nil
}

func TestRuntimeRecusaNoPlanejamentoPlanoSemInputObrigatorio(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(filepath.Join(root, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Planner: planejadorComInputFaltando{}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	_, err = runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "ler um arquivo"})
	if err == nil {
		t.Fatal("plano sem o campo obrigatorio NAO pode ser aceito")
	}
	if !strings.Contains(err.Error(), "step_1") || !strings.Contains(err.Error(), "path") {
		t.Fatalf("o erro precisa nomear a ETAPA e o CAMPO para ser acionavel: %v", err)
	}
	if !strings.Contains(err.Error(), "workspace.read") {
		t.Fatalf("o erro precisa nomear a FERRAMENTA: %v", err)
	}
}
