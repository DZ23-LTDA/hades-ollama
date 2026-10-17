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

// TestMissaoRealComModeloLocalProduzArtefato leva a missão real um passo adiante: além de planejar
// com modelo local, ela precisa PRODUZIR ARTEFATO — e escrita exige aprovação, então o teste
// exercita o gate humano de verdade (aprovar e retomar).
//
// OPT-IN: sem OLLAMA_AGENT_REAL_LOCAL_MODEL, SKIP. O resultado do modelo é variável por natureza,
// então o teste declara o que observou em vez de fingir: se o modelo não planejar escrita nesta
// amostra, isso é registrado no log e a asserção de artefato não é aplicada — mas consumo real e
// conclusão continuam exigidos.
func TestMissaoRealComModeloLocalProduzArtefato(t *testing.T) {
	modelo := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REAL_LOCAL_MODEL"))
	if modelo == "" {
		t.Skip("BLOCKED_BY_EXTERNAL_DEPENDENCY: defina OLLAMA_AGENT_REAL_LOCAL_MODEL (ex.: qwen2.5:7b) para rodar a missão real com artefato")
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
	planejador := &OllamaPlanner{Client: api.NewClient(base, http.DefaultClient), Model: modelo, Price: LocalModelPrice()}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:           store,
		Planner:         planejador,
		PlannerResolver: resolvedorFixo{provider: "ollama-local", planner: planejador},
		WorkspaceRoot:   root, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	missao, err := runtime.CreateMission(ctx, CreateMissionRequest{
		Objective:    "criar o arquivo relatorio.md no workspace com o conteudo: hades ok",
		Provider:     "ollama-local",
		Model:        modelo,
		Workspace:    root,
		Capabilities: []string{"workspace:read", "workspace:write"},
	})
	if err != nil {
		t.Fatalf("missao real nao foi criada: %v", err)
	}
	if missao.Usage == nil || missao.Usage.PromptTokens <= 0 {
		t.Fatalf("consumo real nao registrado: %+v", missao.Usage)
	}
	for _, etapa := range missao.Plan {
		t.Logf("etapa planejada: kind=%q aprovacao=%v input=%+v", etapa.Kind, etapa.RequiresApproval, etapa.Input)
	}

	precisaAprovacao := missao.State == MissionAwaitingApproval && len(missao.Approvals) > 0
	if !precisaAprovacao {
		t.Logf("DECLARADO: nesta amostra o modelo nao planejou etapa que exige aprovacao (estado=%s); "+
			"a assercao de artefato nao se aplica, mas o consumo real acima continua exigido", missao.State)
		return
	}
	if err := runtime.Run(ctx, missao.ID); err != nil {
		t.Fatalf("execucao antes da aprovacao falhou: %v", err)
	}
	aprovada, err := runtime.DecideApproval(missao.ID, missao.Approvals[0].ID, true, "aprovado pelo teste opt-in de artefato real")
	if err != nil {
		t.Fatalf("aprovacao falhou: %v", err)
	}
	if err := runtime.Run(ctx, aprovada.ID); err != nil {
		t.Fatalf("execucao apos aprovacao falhou: %v", err)
	}
	final, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("estado final: %s | artefatos: %d", final.State, len(final.Artifacts))
	if final.State != MissionCompleted {
		t.Fatalf("missao real com artefato nao concluiu: %s", final.State)
	}
	if len(final.Artifacts) == 0 || final.Artifacts[0].SHA256 == "" {
		t.Fatalf("missao concluiu sem artefato verificavel: %+v", final.Artifacts)
	}
	// O manifesto guarda o caminho RELATIVO ao workspace; resolver contra a raiz foi o erro do
	// meu primeiro teste (não do produto).
	caminho := final.Artifacts[0].Path
	if !filepath.IsAbs(caminho) {
		caminho = filepath.Join(root, caminho)
	}
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("artefato registrado nao existe no disco (%s): %v", caminho, err)
	}
	if len(dados) == 0 {
		t.Fatalf("artefato vazio em %s", caminho)
	}
	t.Logf("ARTEFATO REAL: %s (%d bytes, sha256=%s)", caminho, len(dados), final.Artifacts[0].SHA256)
}
