package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

func TestPlanningToolContractListaCamposExigidosEDeterministico(t *testing.T) {
	contrato := PlanningToolContract([]ToolDescriptor{
		{Name: "workspace.read", Risk: RiskRead, RequiredInputs: []string{"path"}},
		{Name: "workspace.list", Risk: RiskRead},
		{Name: "workspace.write", Risk: RiskWrite, RequiresApproval: true, RequiredInputs: []string{"path"}},
	})
	for _, exigido := range []string{"workspace.read", "path", "workspace.write", "aprovacao sim", "nenhum"} {
		if !strings.Contains(contrato, exigido) {
			t.Fatalf("contrato nao menciona %q:\n%s", exigido, contrato)
		}
	}
	// Ordem determinística: o registro não garante ordem estável, e um prompt que muda de
	// ordem sem motivo atrapalha comparação e cache.
	if strings.Index(contrato, "workspace.list") > strings.Index(contrato, "workspace.read") {
		t.Fatalf("contrato fora de ordem:\n%s", contrato)
	}
	if PlanningToolContract(nil) != "" {
		t.Fatal("sem descritores o contrato deve ser vazio")
	}
}

// clienteQueCapturaPrompt guarda a última requisição para inspeção do prompt.
type clienteQueCapturaPrompt struct {
	ultima api.ChatRequest
}

func (c *clienteQueCapturaPrompt) Chat(_ context.Context, request *api.ChatRequest, callback api.ChatResponseFunc) error {
	c.ultima = *request
	return callback(api.ChatResponse{Message: api.Message{Content: `{"steps":[{"kind":"workspace.list","title":"listar","risk":"read","requires_approval":false,"input":{}}]}`}})
}

func TestPlannerEnviaContratoRealDasFerramentasNoPrompt(t *testing.T) {
	cliente := &clienteQueCapturaPrompt{}
	planejador := &OllamaPlanner{Client: cliente, Model: "modelo"}
	planejador.SetPlanToolContract([]ToolDescriptor{
		{Name: "workspace.read", Risk: RiskRead, RequiredInputs: []string{"path"}},
	})
	if _, err := planejador.Plan(context.Background(), Mission{Objective: "ler algo"}); err != nil {
		t.Fatal(err)
	}
	if len(cliente.ultima.Messages) == 0 {
		t.Fatal("nenhuma mensagem enviada")
	}
	sistema := cliente.ultima.Messages[0].Content
	if !strings.Contains(sistema, "workspace.read") || !strings.Contains(sistema, "path") {
		t.Fatalf("prompt de sistema nao declara o contrato real das ferramentas:\n%s", sistema)
	}
}

func TestPlannerSemContratoMantemPromptAnterior(t *testing.T) {
	cliente := &clienteQueCapturaPrompt{}
	planejador := &OllamaPlanner{Client: cliente, Model: "modelo"}
	if _, err := planejador.Plan(context.Background(), Mission{Objective: "ler algo"}); err != nil {
		t.Fatal(err)
	}
	sistema := cliente.ultima.Messages[0].Content
	if strings.Contains(sistema, "Contrato das ferramentas registradas") {
		t.Fatalf("sem descritores injetados o prompt nao pode afirmar contrato:\n%s", sistema)
	}
	if !strings.Contains(sistema, "You are a mission planner") {
		t.Fatal("instrucao base do planejador sumiu")
	}
}
