package agent

import "github.com/ollama/ollama/api"

// ModelUsage registra o consumo REAL de chamadas de modelo atribuídas a uma missão.
//
// Por que existe: o master exige que a linha do tempo da missão carregue custo. Antes
// deste tipo, nem tokens nem custo chegavam ao registro da missão — o preço existia no
// catálogo de provedores e os contadores existiam na resposta da API, mas nada ligava as
// duas pontas. Aqui o consumo passa a ser dado de primeira classe e persistido.
type ModelUsage struct {
	Provider         string `json:"provider,omitempty"`
	Model            string `json:"model,omitempty"`
	Calls            int    `json:"calls,omitempty"`
	PromptTokens     int    `json:"prompt_tokens,omitempty"`
	CompletionTokens int    `json:"completion_tokens,omitempty"`
	CostCents        int64  `json:"cost_cents,omitempty"`
	// CostKnown distingue "custo zero porque o modelo é local/gratuito" de "custo
	// desconhecido porque não há preço declarado". Nunca tratar desconhecido como zero.
	CostKnown bool `json:"cost_known"`
}

// ModelPrice é o preço declarado do modelo, em centavos por 1K tokens.
type ModelPrice struct {
	InputCentsPer1K  int64
	OutputCentsPer1K int64
	// Known declara se há preço confiável. Um modelo local gratuito é Known=true com
	// tarifas zero; ausência de informação é Known=false.
	Known bool
}

// ComputeModelCostCents devolve o custo em centavos de UMA chamada.
//
// Regras, todas testadas em model_usage_test.go:
//   - sem preço declarado (Known=false) devolve (0, false): desconhecido não é zero;
//   - calcula o valor PROPORCIONAL (tokens/1000 × tarifa) e arredonda o valor em centavos
//     para CIMA, porque cobrança fracionada nunca favorece o operador. Não é arredondamento
//     por bloco: 1 token a 6¢/1K custa 1 centavo, não 6;
//   - contagem negativa (defensivo) é tratada como zero: nunca creditar por entrada inválida.
func ComputeModelCostCents(promptTokens, completionTokens int, price ModelPrice) (int64, bool) {
	if !price.Known {
		return 0, false
	}
	entrada := centavosArredondadosParaCima(promptTokens, price.InputCentsPer1K)
	saida := centavosArredondadosParaCima(completionTokens, price.OutputCentsPer1K)
	return entrada + saida, true
}

// centavosArredondadosParaCima devolve ceil(tokens × tarifa / 1000) em centavos.
func centavosArredondadosParaCima(tokens int, centsPer1K int64) int64 {
	if tokens <= 0 || centsPer1K <= 0 {
		return 0
	}
	return (int64(tokens)*centsPer1K + 999) / 1000
}

// AddModelUsage acumula dois consumos. O custo só é declarado conhecido quando os DOIS
// lados são conhecidos: somar um custo conhecido com um desconhecido produziria um total
// que mente sobre a própria exatidão.
func AddModelUsage(a, b ModelUsage) ModelUsage {
	soma := ModelUsage{
		Provider:         a.Provider,
		Model:            a.Model,
		Calls:            a.Calls + b.Calls,
		PromptTokens:     max(0, a.PromptTokens) + max(0, b.PromptTokens),
		CompletionTokens: max(0, a.CompletionTokens) + max(0, b.CompletionTokens),
		CostCents:        max64(0, a.CostCents) + max64(0, b.CostCents),
		CostKnown:        a.CostKnown && b.CostKnown,
	}
	if soma.Provider == "" {
		soma.Provider = b.Provider
	}
	if soma.Model == "" {
		soma.Model = b.Model
	}
	return soma
}

// UsageFromMetrics traduz os contadores da resposta da API em consumo da missão.
// Sempre conta a chamada, mesmo quando os contadores vêm zerados: uma chamada que não
// reporta tokens ainda é uma chamada, e esconder isso infla a confiança no orçamento.
func UsageFromMetrics(metrics api.Metrics, provider, model string, price ModelPrice) ModelUsage {
	usage := ModelUsage{
		Provider:         provider,
		Model:            model,
		Calls:            1,
		PromptTokens:     max(0, metrics.PromptEvalCount),
		CompletionTokens: max(0, metrics.EvalCount),
	}
	if custo, conhecido := ComputeModelCostCents(usage.PromptTokens, usage.CompletionTokens, price); conhecido {
		usage.CostCents = custo
		usage.CostKnown = true
	}
	return usage
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
