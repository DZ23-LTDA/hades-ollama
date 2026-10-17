package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ollama/ollama/api"
)

// E-666 — O EXECUTOR QUE FALTAVA (§4): um passo que FAZ INFERENCIA, no modelo ROTEADO PARA ELE.
//
// O DIAGNOSTICO MEDIDO (e ele e' constrangedor de dizer em voz alta): `ToolContext.ModelID` existe desde o E-438,
// o evento `step.model.route` registra a decisao, o roteador escolhe por capacidade medida, o `AdaptiveRouter`
// decompoe em subtarefas... **e nao havia NENHUMA ferramenta que chamasse modelo.** A rota por subtarefa era uma
// decisao *observada* e nunca *aplicada*: *um orquestrador multimodelo em que nada consome o modelo escolhido nao
// orquestra - ele narra o roteamento.*
//
// Este e' o primeiro consumidor. **Ele NAO escolhe modelo**: usa o que o roteador escolheu, e **falha** se nao
// houver nada escolhido. *Escolher por conta propria aqui seria criar uma segunda decisao - e a segunda decisao
// vence a primeira em silencio, que e' a classe de defeito do E-619.*
var (
	// ErrInferenciaSemModeloRoteado: o passo pediu inferencia mas nao houve roteamento por passo. *Zero rota nao e'
	// licenca para usar "o modelo da missao" - e' ausencia de decisao, e ausencia de decisao nao se inventa.*
	ErrInferenciaSemModeloRoteado = errors.New("inference step has no routed model for this step")
	// ErrInferenciaSemCliente: nao ha cliente de inferencia injetado. Falha fechado em vez de devolver texto vazio.
	ErrInferenciaSemCliente = errors.New("inference client is unavailable")
	// ErrInferenciaSemPrompt: o passo nao trouxe o input obrigatorio `prompt`.
	ErrInferenciaSemPrompt = errors.New("inference step requires the `prompt` input")
)

// InferenceChatClient e' o cliente de inferencia injetavel no runtime (E-666).
//
// Ele e' a **mesma** abstracao que o planejador ja' usa (`plannerChatClient`, em `planner.go`): um `Chat` com
// callback de resposta. *Reusar a abstracao existente em vez de inventar uma segunda e' o que impede o runtime de
// ter dois caminhos para falar com modelo - a mesma disciplina de fonte unica do E-619/E-658.*
type InferenceChatClient interface {
	Chat(ctx context.Context, request *api.ChatRequest, callback api.ChatResponseFunc) error
}

// KindsDeInferencia e' o kind por onde a ferramenta se registra, declarado uma vez so'.
const KindDeInferencia = "model.infer"

// inferenceTool executa UMA chamada de inferencia no modelo roteado para o passo.
type inferenceTool struct {
	cliente InferenceChatClient
}

func (t inferenceTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:    KindDeInferencia,
		Version: "1.0.0",
		Description: "Runs one inference call on the model the ROUTER selected for this step (ToolContext.ModelID). " +
			"Refuses when no model was routed instead of picking one. Returns the model actually used, so the " +
			"evidence chain can be confronted with the routing decision.",
		// Uma chamada de modelo sai da maquina, custa dinheiro e pode enviar dado sensivel: **e' efeito externo**, e
		// se declara como tal. *Chamar de `read` para o passo nao pedir aprovacao seria mentir no descritor.*
		Risk:             RiskExternalSideEffect,
		Scopes:           []string{"model:infer"},
		RequiresApproval: true,
		RequiredInputs:   []string{"prompt"},
	}
}

func (t inferenceTool) Execute(ctx context.Context, contexto ToolContext, input map[string]any) (ToolResult, error) {
	if t.cliente == nil {
		return ToolResult{}, ErrInferenciaSemCliente
	}
	modelo := strings.TrimSpace(contexto.ModelID)
	if modelo == "" {
		return ToolResult{}, ErrInferenciaSemModeloRoteado
	}
	prompt, _ := input["prompt"].(string)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ToolResult{}, ErrInferenciaSemPrompt
	}

	stream := false
	requisicao := &api.ChatRequest{
		Model:  modelo,
		Stream: &stream,
		Messages: []api.Message{
			{Role: "system", Content: "You are a single-step executor inside an audited agent runtime. Answer the " +
				"requested step directly and concisely. Never claim work you did not do, and never invent command " +
				"output, test results or file contents."},
			{Role: "user", Content: prompt},
		},
	}

	var resposta string
	err := t.cliente.Chat(ctx, requisicao, func(chatResponse api.ChatResponse) error {
		// Em streaming os pedacos chegam um a um; concatenar mantem o texto completo.
		resposta += chatResponse.Message.Content
		return nil
	})
	if err != nil {
		return ToolResult{}, fmt.Errorf("inference on routed model %q failed: %w", modelo, err)
	}

	// `empty` e' DECLARADO: uma resposta vazia nao pode ser confundida com "o passo entregou algo".
	return ToolResult{Value: map[string]any{
		"model": modelo,
		"text":  resposta,
		"empty": strings.TrimSpace(resposta) == "",
	}}, nil
}
