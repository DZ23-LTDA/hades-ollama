package agent

import (
	"testing"

	"github.com/ollama/ollama/api"
)

func TestComputeModelCostCents(t *testing.T) {
	casos := []struct {
		nome          string
		prompt        int
		completion    int
		price         ModelPrice
		querCentavos  int64
		querConhecido bool
	}{
		{
			nome:   "sem preco declarado nao vira custo zero",
			prompt: 1000, completion: 1000,
			price:        ModelPrice{},
			querCentavos: 0, querConhecido: false,
		},
		{
			nome:   "modelo local gratuito e zero CONHECIDO",
			prompt: 5000, completion: 5000,
			price:        ModelPrice{Known: true},
			querCentavos: 0, querConhecido: true,
		},
		{
			nome:         "arredonda o valor em centavos para cima (1 token a 3c/1K = 1 centavo)",
			prompt:       1,
			price:        ModelPrice{InputCentsPer1K: 3, Known: true},
			querCentavos: 1, querConhecido: true,
		},
		{
			nome:         "nao arredonda por bloco: 1001 tokens a 1c/1K = 2 centavos",
			prompt:       1001,
			price:        ModelPrice{InputCentsPer1K: 1, Known: true},
			querCentavos: 2, querConhecido: true,
		},
		{
			nome:   "entrada e saida somam no custo da chamada",
			prompt: 2000, completion: 3000,
			price:        ModelPrice{InputCentsPer1K: 10, OutputCentsPer1K: 20, Known: true},
			querCentavos: 20 + 60, querConhecido: true,
		},
		{
			nome:   "tokens negativos nao creditam",
			prompt: -100, completion: -100,
			price:        ModelPrice{InputCentsPer1K: 10, OutputCentsPer1K: 10, Known: true},
			querCentavos: 0, querConhecido: true,
		},
	}
	for _, caso := range casos {
		centavos, conhecido := ComputeModelCostCents(caso.prompt, caso.completion, caso.price)
		if centavos != caso.querCentavos || conhecido != caso.querConhecido {
			t.Fatalf("%s: custo = (%d, %v), quer (%d, %v)", caso.nome, centavos, conhecido, caso.querCentavos, caso.querConhecido)
		}
	}
}

func TestAddModelUsageNaoDeclaraCustoDesconhecidoComoConhecido(t *testing.T) {
	conhecido := ModelUsage{Calls: 1, PromptTokens: 100, CostCents: 5, CostKnown: true}
	desconhecido := ModelUsage{Calls: 1, PromptTokens: 200, CostKnown: false}

	soma := AddModelUsage(conhecido, desconhecido)
	if soma.CostKnown {
		t.Fatal("somar conhecido com desconhecido NAO pode declarar custo conhecido")
	}
	if soma.Calls != 2 || soma.PromptTokens != 300 {
		t.Fatalf("acumulo incorreto: %+v", soma)
	}

	ambos := AddModelUsage(conhecido, conhecido)
	if !ambos.CostKnown || ambos.CostCents != 10 || ambos.Calls != 2 {
		t.Fatalf("acumulo de dois conhecidos incorreto: %+v", ambos)
	}
}

func TestAddModelUsagePreservaIdentidadeEEngoleEntradasInvalidas(t *testing.T) {
	comIdentidade := ModelUsage{Provider: "ollama", Model: "qwen2.5:7b", Calls: 1, CostKnown: true}
	semIdentidade := ModelUsage{Calls: 1, PromptTokens: -50, CostCents: -10, CostKnown: true}

	soma := AddModelUsage(comIdentidade, semIdentidade)
	if soma.Provider != "ollama" || soma.Model != "qwen2.5:7b" {
		t.Fatalf("identidade perdida no acumulo: %+v", soma)
	}
	if soma.PromptTokens != 0 || soma.CostCents != 0 {
		t.Fatalf("entrada negativa nao pode reduzir o total: %+v", soma)
	}
}

func TestUsageFromMetricsContaChamadaMesmoSemContadores(t *testing.T) {
	semContadores := UsageFromMetrics(api.Metrics{}, "ollama", "qwen2.5:7b", ModelPrice{Known: true})
	if semContadores.Calls != 1 {
		t.Fatalf("uma chamada sem contadores ainda e uma chamada: %+v", semContadores)
	}
	if !semContadores.CostKnown || semContadores.CostCents != 0 {
		t.Fatalf("modelo local gratuito deve ter custo zero CONHECIDO: %+v", semContadores)
	}

	comContadores := UsageFromMetrics(api.Metrics{PromptEvalCount: 1500, EvalCount: 500}, "groq", "llama-3.1-8b",
		ModelPrice{InputCentsPer1K: 6, OutputCentsPer1K: 12, Known: true})
	if comContadores.PromptTokens != 1500 || comContadores.CompletionTokens != 500 {
		t.Fatalf("contadores nao mapeados: %+v", comContadores)
	}
	// 1500 tokens x 6c/1K = 9 centavos e 500 tokens x 12c/1K = 6 centavos (proporcional,
	// arredondado para cima em centavos) = 15 centavos.
	if !comContadores.CostKnown || comContadores.CostCents != 15 {
		t.Fatalf("custo incorreto: %+v", comContadores)
	}
	if comContadores.Provider != "groq" || comContadores.Model != "llama-3.1-8b" {
		t.Fatalf("identidade do provedor/modelo nao registrada: %+v", comContadores)
	}

	semPreco := UsageFromMetrics(api.Metrics{PromptEvalCount: 10, EvalCount: 10}, "desconhecido", "x", ModelPrice{})
	if semPreco.CostKnown {
		t.Fatalf("sem preco, o custo NAO pode ser declarado conhecido: %+v", semPreco)
	}
	if semPreco.PromptTokens != 10 || semPreco.CompletionTokens != 10 {
		t.Fatalf("tokens devem ser registrados mesmo sem preco: %+v", semPreco)
	}
}
