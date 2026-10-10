package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func commandRequest(t *testing.T, method, body, organization string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, "/api/agent/v1/commands", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if organization != "" {
		ctx.Set("agent.organization", agent.Organization{ID: organization})
	}
	return ctx, recorder
}

func newCommandAPI(t *testing.T) (*agentAPI, agent.Mission, agent.Mission) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	workspace := t.TempDir()
	contextStore, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Context:       contextStore,
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	own, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{
		Objective:      "missão da organização A",
		OrganizationID: "org-a",
		Workspace:      workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{
		Objective:      "missão da organização B",
		OrganizationID: "org-b",
		Workspace:      workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &agentAPI{runtime: runtime, context: contextStore, authRequired: true}, own, foreign
}

func TestCommandDiscoveryListsRegistryAndHelp(t *testing.T) {
	api, _, _ := newCommandAPI(t)
	ctx, recorder := commandRequest(t, http.MethodGet, "", "org-a")
	api.commandDiscovery(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Commands []agent.SlashCommandSpec `json:"commands"`
		Matched  int                      `json:"matched"`
		Total    int                      `json:"total"`
		Help     string                   `json:"help"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v body=%s", err, recorder.Body.String())
	}
	if payload.Total != len(agent.DefaultSlashRegistry()) || payload.Matched != payload.Total {
		t.Fatalf("discovery payload = %+v", payload)
	}
	if !strings.Contains(payload.Help, "Comandos indisponíveis nesta versão (não executam nada)") {
		t.Fatalf("help must state what does nothing: %s", payload.Help)
	}
	// Prefixo filtra e a resposta continua honesta sobre disponibilidade.
	prefixedCtx, prefixedRecorder := commandRequest(t, http.MethodGet, "", "org-a")
	prefixedCtx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/commands?prefix=/st", nil)
	api.commandDiscovery(prefixedCtx)
	var filtered struct {
		Commands []agent.SlashCommandSpec `json:"commands"`
	}
	if err := json.Unmarshal(prefixedRecorder.Body.Bytes(), &filtered); err != nil {
		t.Fatal(err)
	}
	// O prefixo casa por começo de nome: /st traz /status e /store.
	if len(filtered.Commands) != 2 {
		t.Fatalf("prefix filter = %+v", filtered.Commands)
	}
	if filtered.Commands[0].Name != "/status" || filtered.Commands[1].Name != "/store" {
		t.Fatalf("prefix /st must match /status and /store, got %+v", filtered.Commands)
	}
}

func TestRunCommandReportsParseErrorsAndUnavailableCommands(t *testing.T) {
	api, _, _ := newCommandAPI(t)

	// Comando desconhecido: 400.
	ctx, recorder := commandRequest(t, http.MethodPost, `{"input":"/nao-existe x"}`, "org-a")
	api.runCommand(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown command status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	// Parâmetro obrigatório ausente: 400 com o nome do parâmetro.
	ctx2, recorder2 := commandRequest(t, http.MethodPost, `{"input":"/status"}`, "org-a")
	api.runCommand(ctx2)
	if recorder2.Code != http.StatusBadRequest || !strings.Contains(recorder2.Body.String(), "missao") {
		t.Fatalf("missing argument status=%d body=%s", recorder2.Code, recorder2.Body.String())
	}

	// Comando reconhecido sem capacidade: 501 com o motivo, nunca 200 fingindo.
	ctx3, recorder3 := commandRequest(t, http.MethodPost, `{"input":"/inbox"}`, "org-a")
	api.runCommand(ctx3)
	if recorder3.Code != http.StatusNotImplemented {
		t.Fatalf("unavailable command status=%d body=%s", recorder3.Code, recorder3.Body.String())
	}
	if !strings.Contains(recorder3.Body.String(), "caixa de entrada") {
		t.Fatalf("unavailable response must carry the reason: %s", recorder3.Body.String())
	}

	// Corpo inválido: 400.
	ctx4, recorder4 := commandRequest(t, http.MethodPost, `{"input":`, "org-a")
	api.runCommand(ctx4)
	if recorder4.Code != http.StatusBadRequest {
		t.Fatalf("invalid body status=%d body=%s", recorder4.Code, recorder4.Body.String())
	}
}

func TestRunCommandExecutesStatusAndHidesForeignMission(t *testing.T) {
	api, own, foreign := newCommandAPI(t)

	ctx, recorder := commandRequest(t, http.MethodPost, `{"input":"/status `+own.ID+`"}`, "org-a")
	api.runCommand(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Command  string        `json:"command"`
		Backend  string        `json:"backend"`
		Executed bool          `json:"executed"`
		Result   agent.Mission `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v body=%s", err, recorder.Body.String())
	}
	if !payload.Executed || payload.Command != "/status" || payload.Result.ID != own.ID {
		t.Fatalf("status payload = %+v", payload)
	}
	if payload.Backend != "GET /api/agent/v1/missions/:id" {
		t.Fatalf("backend = %q", payload.Backend)
	}

	// Missão de outra organização: 404, sem vazar existência nem objetivo.
	foreignCtx, foreignRecorder := commandRequest(t, http.MethodPost, `{"input":"/status `+foreign.ID+`"}`, "org-a")
	api.runCommand(foreignCtx)
	if foreignRecorder.Code != http.StatusNotFound {
		t.Fatalf("foreign mission status=%d body=%s", foreignRecorder.Code, foreignRecorder.Body.String())
	}
	if strings.Contains(foreignRecorder.Body.String(), foreign.Objective) {
		t.Fatalf("foreign objective leaked: %s", foreignRecorder.Body.String())
	}
}

func TestRunCommandCancelsOnlyTheTargetMission(t *testing.T) {
	api, own, foreign := newCommandAPI(t)

	ctx, recorder := commandRequest(t, http.MethodPost, `{"input":"/cancel `+own.ID+`"}`, "org-a")
	api.runCommand(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("cancel status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Executed bool          `json:"executed"`
		Result   agent.Mission `json:"result"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Executed || payload.Result.State != agent.MissionCancelled {
		t.Fatalf("cancel payload = %+v", payload)
	}

	// A outra organização continua intacta: /cancel não atravessa tenant.
	cancelForeignCtx, cancelForeignRecorder := commandRequest(t, http.MethodPost, `{"input":"/cancel `+foreign.ID+`"}`, "org-a")
	api.runCommand(cancelForeignCtx)
	if cancelForeignRecorder.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant cancel status=%d body=%s", cancelForeignRecorder.Code, cancelForeignRecorder.Body.String())
	}
	foreignCtx, foreignRecorder := commandRequest(t, http.MethodPost, `{"input":"/status `+foreign.ID+`"}`, "org-b")
	api.runCommand(foreignCtx)
	var foreignPayload struct {
		Result agent.Mission `json:"result"`
	}
	if err := json.Unmarshal(foreignRecorder.Body.Bytes(), &foreignPayload); err != nil {
		t.Fatal(err)
	}
	if foreignPayload.Result.State == agent.MissionCancelled {
		t.Fatalf("command from another organization cancelled the mission: %+v", foreignPayload.Result)
	}
}

func TestRunCommandExecutesLocalReadsAndResolvesTheRest(t *testing.T) {
	api, own, _ := newCommandAPI(t)

	// /mcp e /skills executam de verdade (listas vazias são resposta legítima).
	for _, command := range []string{"/mcp", "/skills"} {
		ctx, recorder := commandRequest(t, http.MethodPost, `{"input":"`+command+`"}`, "org-a")
		api.runCommand(ctx)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", command, recorder.Code, recorder.Body.String())
		}
		var payload struct {
			Executed bool `json:"executed"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if !payload.Executed {
			t.Fatalf("%s must execute: %s", command, recorder.Body.String())
		}
	}

	// /memory usa a mesma checagem de organização da rota de projeto. O projeto
	// precisa viver dentro do workspace do runtime, como na rota real.
	project, err := api.context.CreateProject("projeto de comando", filepath.Join(api.runtime.WorkspaceRoot(), "proj-cmd"), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.context.AddMemoryContext(context.Background(), agent.Memory{
		ID: "mem_cmd", ProjectID: project.ID, Kind: "fact", Content: "memória acessada por comando",
	}); err != nil {
		t.Fatal(err)
	}
	memoryCtx, memoryRecorder := commandRequest(t, http.MethodPost, `{"input":"/memory `+project.ID+`"}`, "org-a")
	api.runCommand(memoryCtx)
	if memoryRecorder.Code != http.StatusOK {
		t.Fatalf("memory status=%d body=%s", memoryRecorder.Code, memoryRecorder.Body.String())
	}
	var memoryPayload struct {
		Executed bool `json:"executed"`
	}
	if err := json.Unmarshal(memoryRecorder.Body.Bytes(), &memoryPayload); err != nil {
		t.Fatal(err)
	}
	if !memoryPayload.Executed {
		t.Fatalf("/memory must execute: %s", memoryRecorder.Body.String())
	}
	// Projeto de outra organização: recusado pela mesma checagem da rota.
	foreignProject, err := api.context.CreateProject("projeto alheio", filepath.Join(api.runtime.WorkspaceRoot(), "proj-alheio"), "org-b")
	if err != nil {
		t.Fatal(err)
	}
	foreignCtx, foreignRecorder := commandRequest(t, http.MethodPost, `{"input":"/memory `+foreignProject.ID+`"}`, "org-a")
	api.runCommand(foreignCtx)
	if foreignRecorder.Code == http.StatusOK || foreignRecorder.Code == http.StatusCreated {
		t.Fatalf("cross-tenant memory command must be refused: %d %s", foreignRecorder.Code, foreignRecorder.Body.String())
	}

	// Comando ligado a rota mas ainda não executado aqui: 200 com executed=false,
	// backend real e dica honesta — nunca "sucesso" sem ação.
	goalCtx, goalRecorder := commandRequest(t, http.MethodPost, `{"input":"/goal auditar"}`, "org-a")
	api.runCommand(goalCtx)
	if goalRecorder.Code != http.StatusOK {
		t.Fatalf("goal status=%d body=%s", goalRecorder.Code, goalRecorder.Body.String())
	}
	var goalPayload struct {
		Command  string `json:"command"`
		Backend  string `json:"backend"`
		Executed bool   `json:"executed"`
		Hint     string `json:"hint"`
	}
	if err := json.Unmarshal(goalRecorder.Body.Bytes(), &goalPayload); err != nil {
		t.Fatal(err)
	}
	if goalPayload.Executed {
		t.Fatalf("/goal must not claim execution yet: %s", goalRecorder.Body.String())
	}
	if goalPayload.Backend != "POST /api/agent/v1/missions" || !strings.Contains(goalPayload.Hint, "rota canônica") {
		t.Fatalf("goal payload = %+v", goalPayload)
	}
	_ = own
}

func TestCommandRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	found := map[string]bool{}
	for _, route := range engine.Routes() {
		found[route.Method+" "+route.Path] = true
	}
	for _, want := range []string{"GET /api/agent/v1/commands", "POST /api/agent/v1/commands"} {
		if !found[want] {
			t.Fatalf("route %s is not registered", want)
		}
	}
}
