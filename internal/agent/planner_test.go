package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

type plannerChatStub struct {
	model    string
	response string
	err      error
	messages []api.Message
}

func (s *plannerChatStub) Chat(_ context.Context, request *api.ChatRequest, callback api.ChatResponseFunc) error {
	s.model = request.Model
	s.messages = append([]api.Message(nil), request.Messages...)
	if s.err != nil {
		return s.err
	}
	return callback(api.ChatResponse{Message: api.Message{Content: s.response}})
}

func TestOllamaPlannerUsesMissionSelectedModel(t *testing.T) {
	stub := &plannerChatStub{response: `{"steps":[{"kind":"workspace.read","title":"inspect","risk":"read","input":{"path":"."}}]}`}
	planner := OllamaPlanner{Client: stub, Model: "default-model"}
	steps, err := planner.Plan(context.Background(), Mission{Objective: "inspect", Model: "selected-model"})
	if err != nil {
		t.Fatal(err)
	}
	if stub.model != "selected-model" {
		t.Fatalf("request model = %q, want selected-model", stub.model)
	}
	if len(steps) != 1 || steps[0].Kind != "workspace.read" {
		t.Fatalf("steps = %+v", steps)
	}
}

func TestOllamaPlannerSurfacesProviderFailure(t *testing.T) {
	planner := OllamaPlanner{
		Client: &plannerChatStub{err: errors.New("provider unavailable")},
		Model:  "selected-model",
	}
	steps, err := planner.Plan(context.Background(), Mission{Objective: "inspect"})
	if err == nil || !strings.Contains(err.Error(), "planner provider request failed") {
		t.Fatalf("err = %v, want explicit provider failure", err)
	}
	if steps != nil {
		t.Fatalf("steps = %+v, want nil on provider failure", steps)
	}
}

func TestOllamaPlannerSurfacesInvalidPlan(t *testing.T) {
	planner := OllamaPlanner{
		Client: &plannerChatStub{response: `{}`},
		Model:  "selected-model",
	}
	steps, err := planner.Plan(context.Background(), Mission{Objective: "inspect"})
	if err == nil || !strings.Contains(err.Error(), "planner returned invalid plan") {
		t.Fatalf("err = %v, want explicit invalid plan", err)
	}
	if steps != nil {
		t.Fatalf("steps = %+v, want nil on invalid plan", steps)
	}
}

func TestOllamaPlannerDoesNotSendHostWorkspaceOrProjectID(t *testing.T) {
	stub := &plannerChatStub{response: `{"steps":[{"kind":"workspace.read","title":"inspect","risk":"read","input":{"path":"."}}]}`}
	planner := OllamaPlanner{Client: stub, Model: "default-model"}
	_, err := planner.Plan(context.Background(), Mission{Objective: "inspect repository", Workspace: "/srv/private/customer-x/repo", ProjectID: "project-secret-id", Model: "selected-model"})
	if err != nil {
		t.Fatal(err)
	}
	var prompt strings.Builder
	for _, message := range stub.messages {
		prompt.WriteString(message.Content)
	}
	if strings.Contains(prompt.String(), "/srv/private/customer-x/repo") || strings.Contains(prompt.String(), "project-secret-id") {
		t.Fatalf("planner prompt contains local workspace or project identifier: %q", prompt.String())
	}
}

func TestNormalizeStepsAllowsReadOnlyGitRepositoryInspection(t *testing.T) {
	steps, err := normalizeSteps([]Step{{Kind: "git.repo.inspect"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 || steps[0].Kind != "git.repo.inspect" || steps[0].Risk != RiskRead || steps[0].RequiresApproval {
		t.Fatalf("normalized Git inspection step=%+v", steps)
	}
}

func TestOllamaPlannerCanSelectReadOnlyGitRepositoryInspection(t *testing.T) {
	stub := &plannerChatStub{response: `{"steps":[{"kind":"git.repo.inspect","title":"Inspect repository","input":{}}]}`}
	steps, err := (OllamaPlanner{Client: stub, Model: "selected-model"}).Plan(context.Background(), Mission{Objective: "inspect the repository"})
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 || steps[0].Kind != "git.repo.inspect" || steps[0].Risk != RiskRead {
		t.Fatalf("planner steps=%+v", steps)
	}
	if len(stub.messages) == 0 || !strings.Contains(stub.messages[0].Content, "git.repo.inspect") {
		t.Fatalf("planner prompt does not advertise git.repo.inspect: %+v", stub.messages)
	}
}

func TestParsePlanAcceptsStringInput(t *testing.T) {
	steps, err := parsePlan("```json\n" + `{"steps":[{"kind":"workspace.list","title":"Listar","risk":"read","input":"."},{"kind":"workspace.read","title":"Ler","risk":"read","input":["a","b"]},{"kind":"workspace.read","title":"Ok","risk":"read","input":{"path":"x"}}]}` + "\n```")
	if err != nil {
		t.Fatal(err)
	}
	if steps[0].Input["text"] != "." {
		t.Fatalf("string input = %#v", steps[0].Input)
	}
	if _, ok := steps[1].Input["value"]; !ok {
		t.Fatalf("array input = %#v", steps[1].Input)
	}
	if steps[2].Input["path"] != "x" {
		t.Fatalf("object input = %#v", steps[2].Input)
	}
	if _, err := parsePlan(`{"steps":"nope"}`); err == nil {
		t.Fatal("non-array steps must still be rejected")
	}
}

func TestOllamaPlannerInjectsContextBootstrapForScopedMissions(t *testing.T) {
	stub := &plannerChatStub{response: `{"steps":[{"kind":"workspace.read","title":"inspect","risk":"read","input":{"path":"."}}]}`}
	planner := OllamaPlanner{Client: stub, Model: "default-model"}
	_, err := planner.Plan(context.Background(), Mission{
		ID:             "mission-1",
		Objective:      "corrigir o bug",
		OrganizationID: "org-a",
		ProjectID:      "proj-1",
		Workspace:      "/srv/privado/repo",
		Capabilities:   []string{"workspace:read"},
		Plan:           []Step{{Title: "Inspecionar"}, {Title: "Corrigir"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(stub.messages) != 3 {
		t.Fatalf("expected system + bootstrap + user, got %d messages", len(stub.messages))
	}
	preamble := stub.messages[1]
	if preamble.Role != "system" {
		t.Fatalf("bootstrap must be a system message, got %q", preamble.Role)
	}
	for _, want := range []string{"Hades", "Permissões declaradas", "workspace:read", "Nunca invente resultados"} {
		if !strings.Contains(preamble.Content, want) {
			t.Errorf("bootstrap preamble missing %q", want)
		}
	}
	for _, secret := range []string{"/srv/privado/repo", "proj-1", "mission-1", "org-a"} {
		if strings.Contains(preamble.Content, secret) {
			t.Errorf("bootstrap preamble leaked %q", secret)
		}
	}
	if stub.messages[2].Role != "user" || !strings.Contains(stub.messages[2].Content, "corrigir o bug") {
		t.Fatalf("objective must remain the last user message: %+v", stub.messages[2])
	}
}

func TestOllamaPlannerSkipsContextBootstrapWithoutTenantScope(t *testing.T) {
	stub := &plannerChatStub{response: `{"steps":[{"kind":"workspace.read","title":"inspect","risk":"read","input":{"path":"."}}]}`}
	planner := OllamaPlanner{Client: stub, Model: "default-model"}
	if _, err := planner.Plan(context.Background(), Mission{Objective: "sem escopo"}); err != nil {
		t.Fatal(err)
	}
	if len(stub.messages) != 2 {
		t.Fatalf("unscoped mission must keep the original two messages, got %d", len(stub.messages))
	}
}

func TestOllamaPlannerTreatsInvalidCapabilitiesAsNoPermission(t *testing.T) {
	stub := &plannerChatStub{response: `{"steps":[{"kind":"workspace.read","title":"inspect","risk":"read","input":{"path":"."}}]}`}
	planner := OllamaPlanner{Client: stub, Model: "default-model"}
	_, err := planner.Plan(context.Background(), Mission{
		ID:             "mission-1",
		Objective:      "escopo com capacidade inválida",
		OrganizationID: "org-a",
		Capabilities:   []string{"root:tudo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(stub.messages) != 3 {
		t.Fatalf("expected the bootstrap message, got %d", len(stub.messages))
	}
	preamble := stub.messages[1].Content
	if !strings.Contains(preamble, "root:tudo: INDISPONIVEL") {
		t.Fatalf("invalid capability must be declared unavailable, got %q", preamble)
	}
	if strings.Contains(preamble, "root:tudo: PODE") {
		t.Fatal("invalid capability must never be granted")
	}
}
