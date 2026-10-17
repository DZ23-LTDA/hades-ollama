package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func TestBuilderHandlersRejectCrossTenantMutationAndListing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	builder, err := agent.NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := builder.Create(context.Background(), agent.BuilderSpec{
		OrganizationID: "org-a",
		Name:           "Org A project",
		Kind:           agent.BuilderWebsite,
	})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: t.TempDir(), Builder: builder, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime}

	listRecorder := httptest.NewRecorder()
	listContext, _ := gin.CreateTestContext(listRecorder)
	listContext.Set("agent.organization", agent.Organization{ID: "org-b"})
	api.builders(listContext)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("cross-tenant list status=%d", listRecorder.Code)
	}
	var payload struct {
		Projects []agent.BuilderProject `json:"projects"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Projects) != 0 {
		t.Fatalf("cross-tenant list returned %d projects", len(payload.Projects))
	}

	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/builders/"+project.ID+"/visual", bytes.NewBufferString(`{"components":[{"id":"attacker","type":"text"}]}`))
	request.Header.Set("Content-Type", "application/json")
	mutationRecorder := httptest.NewRecorder()
	mutationContext, _ := gin.CreateTestContext(mutationRecorder)
	mutationContext.Request = request
	mutationContext.Params = gin.Params{{Key: "id", Value: project.ID}}
	mutationContext.Set("agent.organization", agent.Organization{ID: "org-b"})
	api.updateBuilderVisual(mutationContext)
	if mutationRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant mutation status=%d body=%s", mutationRecorder.Code, mutationRecorder.Body.String())
	}
	unchanged, err := builder.GetForOrganization(project.ID, "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(unchanged.Components) != 0 {
		t.Fatalf("cross-tenant mutation changed project: %+v", unchanged.Components)
	}
}
