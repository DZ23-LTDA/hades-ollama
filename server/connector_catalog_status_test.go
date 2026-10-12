package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// CATALOGAR NÃO É HABILITAR — GUARDA DE SUPERFÍCIE (E-200).
//
// O §11.3 do master exige que todo conector exponha estado (`INSTALLED`, `CONNECTED`, `CONFIG_REQUIRED`,
// `UNAUTHORIZED`, `UNHEALTHY`, `UNSUPPORTED`, `TESTED_LIVE`, `CAPABILITIES`) e que a UI **não** indique "pronto para
// usar" sem verificação real no ambiente. O catálogo estático do Hades declara, por entrada, um `status` de
// configuração (`available`, `operator_setup_required`, `provider_selection_required`), e o B-10 registra que **105 de
// 123** exigem configuração do operador.
//
// O `internal/agent` já garante o vocabulário fechado e que `available` exige caminho de auth publicado. O que
// **faltava** era um guarda na **superfície**: se a rota do catálogo deixasse de devolver `status` (ou devolvesse um
// rótulo fora do vocabulário), o cliente perderia o sinal de "exige configuração" e poderia apresentar tudo como
// pronto — exatamente o que o §11.3 proíbe. Este teste fecha essa lacuna.
func TestConnectorCatalogRouteKeepsSetupStatusVisible(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// A rota do catálogo é estática (não depende do runtime montado), então o teste é hermético de propósito.
	ctx, rec := orgContext(t, http.MethodGet, "/api/agent/v1/connector-catalog", "org-a")
	connectorCatalog(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("a rota do catálogo precisa responder 200, veio %d", rec.Code)
	}

	var corpo struct {
		Connectors []agent.ConnectorCatalogEntry `json:"connectors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &corpo); err != nil {
		t.Fatal(err)
	}
	if len(corpo.Connectors) == 0 {
		t.Fatal("o catálogo não pode vir vazio: sem entrada nenhuma não há estado para distinguir")
	}

	vocabulario := map[string]bool{
		"available":                   true,
		"operator_setup_required":     true,
		"provider_selection_required": true,
	}

	var exigemConfiguracao int
	for _, entrada := range corpo.Connectors {
		if entrada.Status == "" {
			t.Fatalf("a entrada %q chegou SEM status: o cliente perderia o sinal de que exige configuração", entrada.ID)
		}
		if !vocabulario[entrada.Status] {
			t.Fatalf("a entrada %q trouxe status %q fora do vocabulário declarado", entrada.ID, entrada.Status)
		}
		if entrada.Status != "available" {
			exigemConfiguracao++
		}
	}

	// A invariante que interessa NÃO é um número fixo (isso envelheceria mal): é que o catálogo continue declarando
	// que parte dele exige configuração. Um catálogo em que tudo é "available" seria a mentira que o §11.3 proíbe.
	if exigemConfiguracao == 0 {
		t.Fatal("o catálogo declara tudo como available: catalogar virou habilitar")
	}
	t.Logf("catálogo: %d entradas, %d exigem configuração do operador", len(corpo.Connectors), exigemConfiguracao)
}
