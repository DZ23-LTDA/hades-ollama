package server

// FASE 19 + FASE 24 do adendo V3: a rota `/routing` devolve a decisao COM as alternativas.
//
// O que este teste prova, e que o teste do motor nao podia provar: que a fronteira de Pareto
// ATRAVESSA o handler, chega ao JSON e sobrevive a serializacao. *Uma capacidade que existe no
// motor e nao chega a' superficie nao e' uma capacidade do produto.*

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/internal/multillm"
)

// registryDeRotas carrega dois modelos `coding` de provedores DIFERENTES: um local e um remoto.
//
//	barato  local   custo 2,   qualidade 10
//	forte   remoto  custo 100, qualidade 90
//
// Nenhum domina o outro -- a fronteira TEM de conter ambos.
func registryDeRotas(t *testing.T) *multillm.Registry {
	t.Helper()
	reg, err := multillm.LoadBytes([]byte(`{"providers":[
		{"name":"casa","type":"openai-compatible","base_url":"https://casa.example","allow_private":true,"models":[
			{"id":"barato","capabilities":["coding"],"cost_per_1k_input_cents":2,"quality_score":10}
		]},
		{"name":"nuvem","type":"openai-compatible","base_url":"https://nuvem.example","models":[
			{"id":"forte","capabilities":["coding"],"cost_per_1k_input_cents":100,"quality_score":90}
		]}
	]}`))
	if err != nil {
		t.Fatalf("carregar registry de rotas: %v", err)
	}
	return reg
}

func runtimeComModelos(t *testing.T, reg *multillm.Registry) *agent.Runtime {
	t.Helper()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
		Models:        reg,
	})
	if err != nil {
		t.Fatalf("runtime com modelos: %v", err)
	}
	return runtime
}

func pedirRota(t *testing.T, runtime *agent.Runtime, query string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{runtime: runtime}).register(engine)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/routing"+query, nil))
	return recorder
}

func TestRoutingDevolveVencedorCandidatosEFronteira(t *testing.T) {
	recorder := pedirRota(t, runtimeComModelos(t, registryDeRotas(t)), "?capabilities=coding")

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/agent/v1/routing -> %d (corpo: %s)", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Winner struct {
			Model string `json:"model"`
			Score int64  `json:"score"`
		} `json:"winner"`
		Candidates []struct {
			Rank    int                `json:"rank"`
			Model   string             `json:"model"`
			Score   int64              `json:"score"`
			Factors map[string]float64 `json:"factors"`
		} `json:"candidates"`
		ParetoFrontier []struct {
			Model   string `json:"model"`
			Quality int64  `json:"quality"`
			Cost    int64  `json:"cost"`
		} `json:"pareto_frontier"`
		AxesMeasured    []string `json:"axes_measured"`
		AxesNotMeasured []string `json:"axes_not_measured"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("o corpo nao e' JSON valido: %v (corpo: %s)", err, recorder.Body.String())
	}

	// (1) Ha' um vencedor e ha' preteridos.
	if payload.Winner.Model == "" {
		t.Fatal("a rota tem de declarar um vencedor")
	}
	if len(payload.Candidates) < 2 {
		t.Fatalf("esperados 2 candidatos `coding`, vieram %d -- *uma escolha sem preteridos nao e' auditavel*",
			len(payload.Candidates))
	}
	if payload.Candidates[0].Model != payload.Winner.Model {
		t.Fatalf("o primeiro candidato (%s) tem de ser o vencedor (%s) -- *duas contas sobre a mesma decisao*",
			payload.Candidates[0].Model, payload.Winner.Model)
	}

	// (2) O DISCRIMINADOR: a soma dos factors tem de ser igual ao score, no JSON.
	for _, candidato := range payload.Candidates {
		soma := float64(0)
		for _, valor := range candidato.Factors {
			soma += valor
		}
		if soma != float64(candidato.Score) {
			t.Fatalf("candidato %s: soma dos factors no JSON = %v, score = %d (factors: %v)",
				candidato.Model, soma, candidato.Score, candidato.Factors)
		}
	}

	// (3) A FRONTEIRA chegou a' superficie, e tem AMBOS: nenhum modelo domina o outro.
	modelosDaFronteira := map[string]bool{}
	for _, alternativa := range payload.ParetoFrontier {
		modelosDaFronteira[alternativa.Model] = true
	}
	for _, esperado := range []string{"casa/barato", "nuvem/forte"} {
		if !modelosDaFronteira[esperado] {
			t.Fatalf("o modelo %q tem de estar na fronteira exposta (nenhum domina o outro); fronteira=%v",
				esperado, modelosDaFronteira)
		}
	}
	// E o vencedor tem de estar nela: *um vencedor fora da fronteira e' um vencedor DOMINADO*.
	if !modelosDaFronteira[payload.Winner.Model] {
		t.Fatalf("o vencedor %q NAO esta' na fronteira exposta -- escolha dominada", payload.Winner.Model)
	}

	// (4) Os eixos NAO medidos sao DECLARADOS. *Inventar energia ou VRAM quando a plataforma nao
	// os fornece seria um numero falso num relatorio.*
	naoMedidos := map[string]bool{}
	for _, eixo := range payload.AxesNotMeasured {
		naoMedidos[eixo] = true
	}
	for _, esperado := range []string{"energy", "vram"} {
		if !naoMedidos[esperado] {
			t.Fatalf("o eixo %q tem de ser declarado como NAO medido; veio %v", esperado, payload.AxesNotMeasured)
		}
	}
}

// TestRoutingSemRegistryDevolve503: *uma lista vazia pareceria "nao ha' modelos" em vez de "nao
// ha' motor de modelos ligado".*
func TestRoutingSemRegistryDevolve503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/routing?capabilities=coding", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("sem registry a rota tem de devolver 503, veio %d (corpo: %s)",
			recorder.Code, recorder.Body.String())
	}
}

// TestRoutingRecusaPreferenciaInvalida: adivinhar a preferencia seria escolher por conta propria.
func TestRoutingRecusaPreferenciaInvalida(t *testing.T) {
	recorder := pedirRota(t, runtimeComModelos(t, registryDeRotas(t)), "?preference=turbo")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("preferencia invalida tem de ser 400, veio %d (corpo: %s)", recorder.Code, recorder.Body.String())
	}
}

// TestRoutingLocalOnlyRecusaRemoto: a RESTRICAO rigida continua a funcionar pela superficie.
func TestRoutingLocalOnlyRecusaRemoto(t *testing.T) {
	reg, err := multillm.LoadBytes([]byte(`{"providers":[
		{"name":"nuvem","type":"openai-compatible","base_url":"https://nuvem.example","models":[
			{"id":"forte","capabilities":["coding"],"cost_per_1k_input_cents":100,"quality_score":90}
		]}
	]}`))
	if err != nil {
		t.Fatalf("registry so' remoto: %v", err)
	}
	recorder := pedirRota(t, runtimeComModelos(t, reg), "?capabilities=coding&local_only=true")
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("local_only com so' remotos tem de devolver 422, veio %d (corpo: %s)",
			recorder.Code, recorder.Body.String())
	}
}

// TestRoutingPrivatePreferenceNaoFiltra: `preference=private` ORDENA, nao filtra. A diferenca para
// `local_only` e' a diferenca entre preferir e exigir.
func TestRoutingPrivatePreferenceNaoFiltra(t *testing.T) {
	recorder := pedirRota(t, runtimeComModelos(t, registryDeRotas(t)), "?capabilities=coding&preference=private")
	if recorder.Code != http.StatusOK {
		t.Fatalf("preference=private nao pode filtrar; veio %d (corpo: %s)", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Candidates []struct {
			Model string `json:"model"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if len(payload.Candidates) != 2 {
		t.Fatalf("preference=private tem de manter os 2 candidatos, veio %d", len(payload.Candidates))
	}
}
