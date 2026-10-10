package agent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// chatOutputTool simula uma ferramenta que produz saída textual, no formato da
// CLI governada (stdout/stderr), para provar que a saída chega à trilha de
// execução do chat.
type chatOutputTool struct{}

func (chatOutputTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "workspace.read", Version: "test", Description: "saída de teste da trilha", Risk: RiskRead, Scopes: []string{"workspace:read"}}
}

func (chatOutputTool) Execute(context.Context, ToolContext, map[string]any) (ToolResult, error) {
	return ToolResult{
		Value: map[string]any{
			"stdout":    "missão concluída\narquivo gerado em relatorio.md",
			"stderr":    "aviso: api_key=super-secret-token-value",
			"exit_code": 0,
		},
		Artifacts: []ArtifactManifest{{
			ID:     "art_output",
			Name:   "relatorio.md",
			Path:   "relatorio.md",
			Size:   42,
			SHA256: strings.Repeat("a", 64),
		}},
	}, nil
}

func TestStepOutputExcerptExtractsBoundedToolText(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{name: "valor nulo não gera trecho", value: nil, want: ""},
		{name: "texto simples é aparado", value: "  pronto  ", want: "pronto"},
		{name: "stdout vem antes de stderr", value: map[string]any{"stderr": "erro", "stdout": "saida"}, want: "saida\nerro"},
		{name: "ordem fixa entre output e result", value: map[string]any{"result": "final", "output": "principal", "exit_code": 1}, want: "principal\nfinal"},
		{name: "mapa sem chave textual não gera trecho", value: map[string]any{"exit_code": 1, "duration_ms": 12}, want: ""},
		{name: "screenshot base64 não vaza para a trilha", value: map[string]any{"screenshot": strings.Repeat("iVBORw0KGgo", 500)}, want: ""},
		{name: "valores em branco não geram trecho", value: map[string]any{"stdout": "   ", "stderr": ""}, want: ""},
		{name: "tipo não textual não gera trecho", value: 42, want: ""},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := stepOutputExcerpt(testCase.value); got != testCase.want {
				t.Fatalf("stepOutputExcerpt() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestStepOutputExcerptTruncatesOnRuneBoundary(t *testing.T) {
	long := strings.Repeat("çãéí ", maxStepOutputExcerptRunes)
	excerpt := stepOutputExcerpt(long)
	if !strings.HasSuffix(excerpt, stepOutputTruncationMarker) {
		t.Fatalf("trecho cortado sem marcador explícito: %q", excerpt)
	}
	body := strings.TrimSuffix(excerpt, stepOutputTruncationMarker)
	if got := utf8.RuneCountInString(body); got != maxStepOutputExcerptRunes {
		t.Fatalf("runas no trecho = %d, want %d", got, maxStepOutputExcerptRunes)
	}
	if !strings.HasPrefix(long, body) {
		t.Fatal("trecho cortado não corresponde ao início da saída original")
	}
	if !utf8.ValidString(excerpt) {
		t.Fatal("o corte por runas quebrou um caractere multibyte")
	}
}

func TestStepOutputExcerptKeepsTextInsideLimit(t *testing.T) {
	exact := strings.Repeat("a", maxStepOutputExcerptRunes)
	if got := stepOutputExcerpt(exact); got != exact {
		t.Fatalf("trecho no limite foi alterado: %d runas", utf8.RuneCountInString(got))
	}
}

func TestRuntimePublishesRedactedToolOutputInStepSucceededEvent(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(filepath.Join(root, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	registry.Register(chatOutputTool{})
	traces, err := NewTraceStore("")
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Tools: registry, Traces: traces, Planner: fixedPlanner{steps: []Step{{ID: "step_output", Kind: "workspace.read", Title: "saida visivel", Risk: RiskRead, State: StepPending}}}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "saída no chat", OrganizationID: "org_a"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Run(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
	completed, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(completed.Plan) != 1 {
		t.Fatalf("plano da missão = %d passos, want 1", len(completed.Plan))
	}
	stepID := completed.Plan[0].ID
	events, err := runtime.Events(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	var succeeded *Event
	for index := range events {
		if events[index].Type == "step.succeeded" && events[index].StepID == stepID {
			succeeded = &events[index]
		}
	}
	if succeeded == nil {
		state := "desconhecido"
		if current, getErr := runtime.GetMission(mission.ID); getErr == nil {
			state = string(current.State)
		}
		types := make([]string, 0, len(events))
		for _, event := range events {
			types = append(types, event.Type)
		}
		t.Fatalf("evento step.succeeded não foi emitido; estado=%s eventos=%v", state, types)
	}
	payload, ok := succeeded.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload do evento = %#v, want map[string]any", succeeded.Payload)
	}
	output, ok := payload["output"].(string)
	if !ok {
		t.Fatalf("evento step.succeeded sem a saída da ferramenta: %#v", payload)
	}
	if !strings.Contains(output, "missão concluída") || !strings.Contains(output, "relatorio.md") {
		t.Fatalf("saída incompleta no evento: %q", output)
	}
	if strings.Contains(output, "super-secret-token-value") {
		t.Fatalf("evento vazou credencial: %q", output)
	}
	if _, ok := payload["artifacts"]; !ok {
		t.Fatalf("campo artifacts perdido no evento: %#v", payload)
	}
	encoded, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "super-secret-token-value") {
		t.Fatalf("eventos persistidos vazaram credencial: %s", encoded)
	}
	if len(encoded) > maxJSONStoreEventPayloadBytes {
		t.Fatalf("eventos acima do orçamento de payload: %d bytes", len(encoded))
	}
}
