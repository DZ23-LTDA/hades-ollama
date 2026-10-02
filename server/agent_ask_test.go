package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

type fakeAskEmbedder map[string][]float32

func (e fakeAskEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return e[text], nil
}

func askRequest(t *testing.T, projectID, query, org string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/projects/"+projectID+"/ask?q="+query, nil)
	ctx.Params = gin.Params{{Key: "id", Value: projectID}}
	ctx.Set("agent.organization", agent.Organization{ID: org})
	return ctx, recorder
}

func TestAskProjectDocumentsGroundsAndGates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	contextStore, err := agent.NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	contextStore.SetEmbedder(fakeAskEmbedder{
		"a politica de ferias e de 30 dias": {1, 0},
		"ferias":                            {1, 0},
		"salario":                           {0, 1},
	})
	root := t.TempDir()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Context: contextStore, Planner: agent.RulePlanner{}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("RH", filepath.Join(root, "rh"), "org_rh")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contextStore.AddMemoryContext(context.Background(), agent.Memory{ProjectID: project.ID, Kind: "doc", Content: "a politica de ferias e de 30 dias", Source: "rh.pdf#1", Confidence: 1}); err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, context: contextStore}

	// Relevant query: grounded with a citation to the indexed source.
	ctx, rec := askRequest(t, project.ID, "ferias", "org_rh")
	api.askProjectDocuments(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("relevant ask status=%d body=%s", rec.Code, rec.Body.String())
	}
	var ok struct {
		Grounded  bool `json:"grounded"`
		Citations []struct {
			Source string `json:"source"`
		} `json:"citations"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &ok); err != nil {
		t.Fatal(err)
	}
	if !ok.Grounded || len(ok.Citations) != 1 || ok.Citations[0].Source != "rh.pdf#1" {
		t.Fatalf("relevant ask not grounded/cited: %s", rec.Body.String())
	}

	// Irrelevant query: no sources -> grounded=false (so the UI will not fabricate).
	ctx2, rec2 := askRequest(t, project.ID, "salario", "org_rh")
	api.askProjectDocuments(ctx2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("irrelevant ask status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	var none struct {
		Grounded  bool          `json:"grounded"`
		Citations []interface{} `json:"citations"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &none); err != nil {
		t.Fatal(err)
	}
	if none.Grounded || len(none.Citations) != 0 {
		t.Fatalf("irrelevant ask should not be grounded: %s", rec2.Body.String())
	}
}
