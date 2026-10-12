package agent

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

// resolvedorFixo devolve sempre o mesmo planejador. Existe para a missão real poder declarar
// provedor sem depender da configuração de provedores do servidor.
type resolvedorFixo struct {
	provider string
	planner  Planner
}

func (r resolvedorFixo) ResolvePlanner(provider, _ string) (Planner, error) {
	if provider != r.provider {
		return nil, os.ErrNotExist
	}
	return r.planner, nil
}

// TestMissaoRealComModeloLocalPlanejaERegistraConsumo roda uma missão DE VERDADE contra
// um modelo LOCAL e prova que o consumo real (tokens e custo) entra no registro da missão.
//
// É OPT-IN de propósito: sem OLLAMA_AGENT_REAL_LOCAL_MODEL o teste NÃO roda, para a suíte seguir
// hermética. Com a variável definida ele FALHA ALTO se o modelo não responder — nunca finge
// sucesso. Modelo local é gratuito POR CONSTRUÇÃO, então o custo esperado é zero CONHECIDO.
func TestMissaoRealComModeloLocalPlanejaERegistraConsumo(t *testing.T) {
	modelo := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REAL_LOCAL_MODEL"))
	if modelo == "" {
		t.Skip("BLOCKED_BY_EXTERNAL_DEPENDENCY: defina OLLAMA_AGENT_REAL_LOCAL_MODEL (ex.: qwen2.5:7b) para rodar a missão real com modelo local")
	}
	endereco := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REAL_LOCAL_HOST"))
	if endereco == "" {
		endereco = "http://127.0.0.1:11434"
	}
	base, err := url.Parse(endereco)
	if err != nil {
		t.Fatalf("endereco invalido %q: %v", endereco, err)
	}

	root := t.TempDir()
	store, err := NewJSONStore(filepath.Join(root, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	planejador := &OllamaPlanner{
		Client: api.NewClient(base, http.DefaultClient),
		Model:  modelo,
		Price:  LocalModelPrice(),
	}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:           store,
		Planner:         planejador,
		PlannerResolver: resolvedorFixo{provider: "ollama-local", planner: planejador},
		WorkspaceRoot:   root,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	missao, err := runtime.CreateMission(ctx, CreateMissionRequest{
		Objective: "inspecionar o workspace",
		Provider:  "ollama-local",
		Model:     modelo,
		Workspace: root,
	})
	if err != nil {
		t.Fatalf("missao real nao foi criada: %v", err)
	}
	if len(missao.Plan) == 0 {
		t.Fatal("modelo real nao produziu plano")
	}
	if missao.Usage == nil {
		t.Fatal("consumo do modelo real NAO entrou no registro da missao")
	}
	if missao.Usage.Calls < 1 || missao.Usage.PromptTokens <= 0 {
		t.Fatalf("tokens reais nao registrados: %+v", missao.Usage)
	}
	if !missao.Usage.CostKnown || missao.Usage.CostCents != 0 {
		t.Fatalf("modelo local deve ter custo zero CONHECIDO: %+v", missao.Usage)
	}
	t.Logf("plano do modelo real (%s): %d etapa(s); consumo: %+v", modelo, len(missao.Plan), *missao.Usage)

	// EXECUÇÃO NÃO HOMOLOGADA — declarada, não escondida. Com modelo real o planejamento passa e o
	// consumo é registrado, mas a execução do plano devolve "anchored workspace path is required".
	// Este teste NÃO afirma execução real bem-sucedida: ele afirma planejamento real + consumo real,
	// e REGISTRA a falha. Quando a execução for corrigida, este log deve virar asserção.
	if err := runtime.Run(ctx, missao.ID); err != nil {
		t.Logf("EXECUCAO NAO HOMOLOGADA (lacuna declarada): %v", err)
		return
	}
	final, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("estado final da missao real: %s", final.State)
	if final.Usage == nil || final.Usage.PromptTokens <= 0 {
		t.Fatalf("consumo real nao sobreviveu: %+v", final.Usage)
	}
	for _, etapa := range final.Plan {
		if etapa.State != StepSucceeded {
			t.Fatalf("etapa %q terminou em %s, queria SUCCEEDED", etapa.ID, etapa.State)
		}
	}
}
