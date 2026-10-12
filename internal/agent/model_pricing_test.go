package agent

import (
	"testing"

	"github.com/ollama/ollama/api"
)

func TestLocalModelPriceEZeroConhecidoNaoDesconhecido(t *testing.T) {
	preco := LocalModelPrice()
	if !preco.Known {
		t.Fatal("modelo local e gratuito POR CONSTRUCAO: o zero tem de ser conhecido")
	}
	if preco.InputCentsPer1K != 0 || preco.OutputCentsPer1K != 0 {
		t.Fatalf("modelo local nao tem tarifa: %+v", preco)
	}
}

func TestModelPriceFromCatalogNaoInventaCustoZero(t *testing.T) {
	casos := []struct {
		nome      string
		entrada   int64
		saida     int64
		etiqueta  string
		querKnown bool
	}{
		{"catalogo sem nenhuma informacao de custo e DESCONHECIDO", 0, 0, "", false},
		{"tarifa de entrada declarada", 6, 0, "", true},
		{"tarifa de saida declarada", 0, 12, "", true},
		{"apenas etiqueta de custo declarada", 0, 0, "free-tier", true},
		{"tarifas declaradas", 10, 20, "paid", true},
	}
	for _, caso := range casos {
		preco := ModelPriceFromCatalog(caso.entrada, caso.saida, caso.etiqueta)
		if preco.Known != caso.querKnown {
			t.Fatalf("%s: Known = %v, quer %v (%+v)", caso.nome, preco.Known, caso.querKnown, preco)
		}
		if preco.InputCentsPer1K != caso.entrada || preco.OutputCentsPer1K != caso.saida {
			t.Fatalf("%s: tarifas alteradas: %+v", caso.nome, preco)
		}
	}
}

func TestCatalogoSemPrecoProduzCustoDesconhecidoMasRegistraTokens(t *testing.T) {
	// Fim a fim da regra: catalogo sem preco nao pode produzir "custo zero conhecido",
	// mas os tokens da chamada continuam sendo registrados.
	semPreco := ModelPriceFromCatalog(0, 0, "")
	usage := UsageFromMetrics(api.Metrics{PromptEvalCount: 1200, EvalCount: 300}, "provedor-sem-preco", "modelo-x", semPreco)
	if usage.CostKnown {
		t.Fatalf("sem preco no catalogo, o custo NAO pode ser conhecido: %+v", usage)
	}
	if usage.PromptTokens != 1200 || usage.CompletionTokens != 300 || usage.Calls != 1 {
		t.Fatalf("tokens e chamada devem ser registrados de qualquer forma: %+v", usage)
	}
}
