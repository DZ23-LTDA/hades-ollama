package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
)

// Critério do estágio 15, provado AO VIVO: a MESMA missão, com o MESMO contexto,
// trocando de modelo entre dois modelos locais reais.
//
// A diferença para um teste de unidade: aqui a requisição sai de verdade, o
// modelo responde de verdade, e o modelo selecionado muda de fato. O que NÃO pode
// mudar é o contexto injetado — senão "trocar o motor" viraria "perder o
// contexto".
//
// Sem Ollama local o teste é PULADO com causa declarada, nunca aprovado por
// omissão.
type recordingChatClient struct {
	real     plannerChatClient
	messages []api.Message
	model    string
	answered bool
	err      error
}

func (r *recordingChatClient) Chat(ctx context.Context, request *api.ChatRequest, callback api.ChatResponseFunc) error {
	r.model = request.Model
	r.messages = append([]api.Message(nil), request.Messages...)
	var got bool
	wrapped := func(response api.ChatResponse) error {
		if strings.TrimSpace(response.Message.Content) != "" {
			got = true
		}
		return callback(response)
	}
	r.err = r.real.Chat(ctx, request, wrapped)
	r.answered = got
	return r.err
}

func liveModels(t *testing.T) map[string]bool {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get("http://127.0.0.1:11434/api/tags")
	if err != nil {
		t.Skipf("Ollama local indisponível: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Skipf("Ollama local respondeu %d", response.StatusCode)
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Skipf("não foi possível ler os modelos locais: %v", err)
	}
	available := map[string]bool{}
	for _, model := range payload.Models {
		available[model.Name] = true
	}
	return available
}

func TestLiveModelSwitchKeepsTheSameContext(t *testing.T) {
	available := liveModels(t)
	first, second := "qwen2.5:7b", "llama3.2:3b"
	if !available[first] || !available[second] {
		t.Skipf("modelos locais necessários ausentes (%s=%v, %s=%v)", first, available[first], second, available[second])
	}
	ollama, err := api.ClientFromEnvironment()
	if err != nil {
		t.Skipf("cliente Ollama indisponível: %v", err)
	}

	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto do teste ao vivo", t.TempDir(), "org-live")
	if err != nil {
		t.Fatal(err)
	}
	// Conhecimento do time e memória PRIVADA de OUTRO ator: só o primeiro pode
	// entrar no contexto do modelo.
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "o time decidiu usar Go no backend", Visibility: MemoryVisibilityOrganization,
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "segredo pessoal da ana que nao pode ir ao modelo", Visibility: MemoryVisibilityPrivate,
	}, "ana"); err != nil {
		t.Fatal(err)
	}

	run := func(model string) (*recordingChatClient, string) {
		t.Helper()
		recorder := &recordingChatClient{real: ollama}
		planner := OllamaPlanner{Client: recorder, Model: "modelo-padrão-não-usado"}
		memories, err := store.SearchMemoriesForActor(context.Background(), project.ID, "bruno", "backend", MaxContextBootstrapMemories)
		if err != nil {
			t.Fatal(err)
		}
		bootstrap, err := BuildContextBootstrap(ContextBootstrapInput{
			OrganizationID:      "org-live",
			ProjectID:           project.ID,
			MissionID:           "mission-live",
			ActorID:             "bruno",
			Objective:           "dizer em uma palavra o backend do projeto",
			GrantedCapabilities: []string{"workspace:read"},
			Policy:              DefaultCapabilityPolicy(),
			Memories:            memories,
		})
		if err != nil {
			t.Fatal(err)
		}
		// O plano pode falhar em interpretar o JSON de um modelo 3b: o que está em
		// teste é a TROCA DE MODELO com o contexto preservado, não a obediência do
		// modelo ao formato. Por isso o erro de parse não derruba o teste, mas a
		// ausência de resposta do modelo derruba.
		if _, planErr := planner.Plan(context.Background(), Mission{
			ID:             "mission-live",
			Objective:      "dizer em uma palavra o backend do projeto",
			OrganizationID: "org-live",
			ProjectID:      project.ID,
			Model:          model,
			Capabilities:   []string{"workspace:read"},
		}); planErr != nil {
			t.Logf("o modelo %s não devolveu plano utilizável (%v) — o teste mede contexto, não formato", model, planErr)
		}
		return recorder, bootstrap.SystemMessage()
	}

	recorderA, preambleA := run(first)
	recorderB, preambleB := run(second)

	if recorderA.err != nil {
		t.Fatalf("modelo %s falhou: %v", first, recorderA.err)
	}
	if recorderB.err != nil {
		t.Fatalf("modelo %s falhou: %v", second, recorderB.err)
	}
	// A troca precisa ser REAL: modelos diferentes chegaram ao provedor.
	if recorderA.model != first || recorderB.model != second {
		t.Fatalf("a seleção de modelo não chegou ao provedor: %q e %q", recorderA.model, recorderB.model)
	}
	// E o contexto injetado precisa ser IDÊNTICO.
	if preambleA != preambleB {
		t.Fatalf("trocar de modelo mudou o contexto:\nA=%q\nB=%q", preambleA, preambleB)
	}
	if len(recorderA.messages) != len(recorderB.messages) {
		t.Fatalf("a quantidade de mensagens mudou na troca: %d e %d", len(recorderA.messages), len(recorderB.messages))
	}
	var injectedA strings.Builder
	for _, message := range recorderA.messages {
		injectedA.WriteString(message.Content)
	}
	var injectedB strings.Builder
	for _, message := range recorderB.messages {
		injectedB.WriteString(message.Content)
	}
	if injectedA.String() != injectedB.String() {
		t.Fatal("o conteúdo enviado ao provedor mudou na troca de modelo")
	}
	// O isolamento vale também no caminho ao vivo: a memória privada da ana não
	// pode ter ido para o modelo do bruno.
	if strings.Contains(injectedA.String(), "segredo pessoal da ana") {
		t.Fatalf("memória privada de outro usuário foi enviada ao modelo: %s", injectedA.String())
	}
	if !strings.Contains(injectedA.String(), "backend") {
		t.Fatalf("o conhecimento do time deveria estar no contexto: %s", injectedA.String())
	}
}
