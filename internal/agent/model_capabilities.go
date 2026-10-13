package agent

import (
	"errors"
	"fmt"
	"strings"
)

// CAPACIDADES DE MODELO E MODO DE FERRAMENTA (V2.2 §11.2 / adendo P0-D) — E-325.
//
// O V2.2 é explícito: *"Se modelo não usa ferramentas, explicar limitação: **NÃO fingir tool calls**; oferecer modo
// compatível"*. E o adendo manda **descobrir capacidades reais** em vez de supor.
//
// **AUDITADO ANTES DE ESCREVER**: existe `CapabilityPolicy` no repositório, mas ela governa **ferramentas e skills**
// (escopos, descritores, atestação) — **não** capacidades de **modelo**. Não havia nada que soubesse que
// `qwen2.5-coder:7b` e `deepseek-r1:8b` **não emitem tool calls nativos** — fato **medido** e registrado nas instruções
// do próprio repositório. Ou seja: ***o produto podia prometer tool calling nativo para um modelo que não o tem, e
// ninguém no código saberia.***
//
// TRÊS REGRAS DESTE MÓDULO:
//
//  1. **Capacidade não medida NÃO é capacidade.** Modelo desconhecido entra como `NAO_MEDIDO` e cai em modo
//     **compatível** (texto), nunca em nativo. *Falhar fechado é a única postura honesta quando não se sabe.*
//  2. **Pedido explícito de tool call nativo num modelo sem suporte é RECUSADO** com erro próprio — não rebaixado em
//     silêncio, porque rebaixar calado faria o agente "funcionar" parecendo o que não é.
//  3. **Falar a verdade sobre a origem do dado**: cada capacidade carrega `Source` (`medido` / `declarado`). Capacidade
//     sem origem é exatamente o que o V2.2 proíbe.

var (
	// ErrModelWithoutNativeToolCalling: o modelo não emite tool calls nativos e o pedido exigia nativo.
	ErrModelWithoutNativeToolCalling = errors.New("model does not support native tool calling")
	// ErrModelCapabilityUnknown: não há medição para este modelo (a resposta honesta é "não medido", não "não suporta").
	ErrModelCapabilityUnknown = errors.New("model capability not measured")
)

// ToolMode diz COMO as ferramentas podem ser usadas com um modelo.
type ToolMode string

const (
	// ToolModeNative: o modelo emite tool calls nativos (capacidade MEDIDA).
	ToolModeNative ToolMode = "native"
	// ToolModeCompatibleText: sem tool call nativo — o modo compatível instrui por texto e o runtime faz o parsing.
	ToolModeCompatibleText ToolMode = "compatible_text"
	// ToolModeUnknown: não há medição; tratar como compatível e DIZER isso.
	ToolModeUnknown ToolMode = "unknown"
)

// Origem do dado de capacidade. Capacidade sem origem não entra no registro.
const (
	CapabilitySourceMeasured = "medido"
	CapabilitySourceDeclared = "declarado"
)

// ModelCapabilities são as capacidades conhecidas de um modelo.
type ModelCapabilities struct {
	Model            string `json:"model"`
	ToolCalling      bool   `json:"tool_calling"`
	Vision           bool   `json:"vision"`
	StructuredOutput bool   `json:"structured_output"`
	Embeddings       bool   `json:"embeddings"`
	ContextTokens    int    `json:"context_tokens,omitempty"`
	// Source registra DE ONDE veio o dado: `medido` (sondado no ambiente) ou `declarado` (documentação do provedor).
	Source string `json:"source"`
	// Note explica a limitação em pt-BR — o V2.2 exige EXPLICAR, não apenas recusar.
	Note string `json:"note,omitempty"`
}

// ModelCapabilityRegistry é o conhecimento do produto sobre modelos. Somente leitura: nada aqui instala, copia ou
// altera modelos do Ollama local — *o motor local do usuário não é mexido para satisfazer um teste*.
type ModelCapabilityRegistry struct {
	conhecidos map[string]ModelCapabilities
}

// DefaultModelCapabilityRegistry devolve o registro com os fatos MEDIDOS neste ambiente.
//
// As três primeiras entradas vêm das instruções do próprio repositório (`~/.dsh/AGENTS.md`), que registram medição
// feita pelo operador. As duas últimas são o caso crítico: modelos bons para texto/código que **não** emitem tool
// calls nativos — usá-los como agente sem avisar é o "tool call fingido" que o V2.2 proíbe.
func DefaultModelCapabilityRegistry() *ModelCapabilityRegistry {
	medidos := []ModelCapabilities{
		{
			Model: "qwen2.5:7b", ToolCalling: true, StructuredOutput: true, Source: CapabilitySourceMeasured,
			Note: "tool calls nativos funcionam com este modelo (medido no ambiente do operador).",
		},
		{
			Model: "llama3.1:8b", ToolCalling: true, StructuredOutput: true, Source: CapabilitySourceMeasured,
			Note: "tool calls nativos funcionam com este modelo (medido no ambiente do operador).",
		},
		{
			Model: "llama3.2:3b", ToolCalling: true, StructuredOutput: true, Source: CapabilitySourceMeasured,
			Note: "tool calls nativos funcionam com este modelo (medido no ambiente do operador).",
		},
		{
			Model: "qwen2.5-coder:7b", ToolCalling: false, StructuredOutput: true, Source: CapabilitySourceMeasured,
			Note: "não emite tool calls nativos: use em modo compatível (texto) ou escolha um modelo com ferramentas.",
		},
		{
			Model: "deepseek-r1:8b", ToolCalling: false, StructuredOutput: false, Source: CapabilitySourceMeasured,
			Note: "não emite tool calls nativos: use em modo compatível (texto) ou escolha um modelo com ferramentas.",
		},
	}
	registro := &ModelCapabilityRegistry{conhecidos: map[string]ModelCapabilities{}}
	for _, capacidades := range medidos {
		registro.conhecidos[normalizeModelName(capacidades.Model)] = capacidades
	}
	return registro
}

func normalizeModelName(modelo string) string {
	return strings.ToLower(strings.TrimSpace(modelo))
}

// Lookup devolve as capacidades conhecidas. O segundo valor é FALSO quando não há medição — e isso é uma resposta,
// não um erro: inventar "não suporta" para um modelo desconhecido seria tão mentiroso quanto inventar "suporta".
func (r *ModelCapabilityRegistry) Lookup(modelo string) (ModelCapabilities, bool) {
	if r == nil {
		return ModelCapabilities{}, false
	}
	capacidades, achou := r.conhecidos[normalizeModelName(modelo)]
	return capacidades, achou
}

// ResolveToolMode diz como usar ferramentas e EXPLICA a limitação quando não há tool call nativo.
func (r *ModelCapabilityRegistry) ResolveToolMode(modelo string) (ToolMode, string) {
	capacidades, achou := r.Lookup(modelo)
	if !achou {
		return ToolModeUnknown, fmt.Sprintf(
			"capacidade de %q NÃO MEDIDA: operando em modo compatível (texto). Não afirmo que o modelo usa ferramentas; "+
				"para afirmar, é preciso sondar o endpoint.", strings.TrimSpace(modelo))
	}
	if capacidades.ToolCalling {
		return ToolModeNative, ""
	}
	return ToolModeCompatibleText, capacidades.Note
}

// RequireNativeToolCalling RECUSA um pedido explícito de tool call nativo quando o modelo não o suporta (ou quando a
// capacidade não foi medida). Rebaixar em silêncio faria o agente "funcionar" parecendo o que não é.
func (r *ModelCapabilityRegistry) RequireNativeToolCalling(modelo string) error {
	capacidades, achou := r.Lookup(modelo)
	if !achou {
		return fmt.Errorf("%w: %q", ErrModelCapabilityUnknown, strings.TrimSpace(modelo))
	}
	if !capacidades.ToolCalling {
		return fmt.Errorf("%w: %q", ErrModelWithoutNativeToolCalling, capacidades.Model)
	}
	return nil
}

// WithCapabilities acrescenta/atualiza uma capacidade DESCOBERTA em tempo de execução (por sonda no endpoint).
//
// Exige `Source` não vazio: ***capacidade sem origem é exatamente o que o V2.2 proíbe.***
func (r *ModelCapabilityRegistry) WithCapabilities(capacidades ModelCapabilities) error {
	if r == nil {
		return errors.New("registro de capacidades ausente")
	}
	if strings.TrimSpace(capacidades.Model) == "" {
		return errors.New("capacidade sem modelo")
	}
	if strings.TrimSpace(capacidades.Source) == "" {
		return fmt.Errorf("capacidade de %q sem origem (medido/declarado): não se registra capacidade sem procedência", capacidades.Model)
	}
	r.conhecidos[normalizeModelName(capacidades.Model)] = capacidades
	return nil
}

// CapabilitySummary resume as capacidades conhecidas, para exibição honesta na interface.
func (r *ModelCapabilityRegistry) CapabilitySummary() []ModelCapabilities {
	resumo := make([]ModelCapabilities, 0, len(r.conhecidos))
	for _, capacidades := range r.conhecidos {
		resumo = append(resumo, capacidades)
	}
	// ordenação estável por nome, para a interface não mudar de ordem a cada chamada
	for i := 1; i < len(resumo); i++ {
		for j := i; j > 0 && resumo[j-1].Model > resumo[j].Model; j-- {
			resumo[j-1], resumo[j] = resumo[j], resumo[j-1]
		}
	}
	return resumo
}
