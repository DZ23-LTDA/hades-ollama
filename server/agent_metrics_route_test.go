package server

// FASE 21 + FASE 24 do adendo V3: a rota `/metrics/telemetry` devolve o agregado dos eventos
// REAIS, com os eixos que nao foram medidos DECLARADOS.
//
// O que este teste recusa aceitar:
//   - que a rota se chame `/metrics` (ja' existe, e devolve CONTADORES, nao agregados);
//   - que `axes_not_measured` venha vazio quando os eventos nao trazem custo nem energia;
//   - que o escopo venha implicito (*um agregado sem escopo le-se como global*).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMetricsTelemetryRouteDevolveOAgregado(t *testing.T) {
	runtime := runtimeComModelos(t, registryDeRotas(t))
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{runtime: runtime}).register(engine)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/metrics/telemetry", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/agent/v1/metrics/telemetry -> %d (corpo: %s)",
			recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Metrics struct {
			Samples         int      `json:"samples"`
			AxesNotMeasured []string `json:"axes_not_measured"`
			Percentiles     string   `json:"percentiles_method"`
		} `json:"metrics"`
		ByModel            []json.RawMessage `json:"by_model"`
		Scope              string            `json:"scope"`
		MissionsUnreadable int               `json:"missions_unreadable"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("o corpo nao e' JSON valido: %v (corpo: %s)", err, recorder.Body.String())
	}

	// (1) O ESCOPO e' declarado. *Um agregado sem escopo le-se como global.*
	if payload.Scope == "" {
		t.Fatal("a rota tem de DECLARAR o escopo")
	}

	// (2) O metodo dos percentis e' declarado. *Um numero sem metodo nao se compara com outro.*
	if payload.Metrics.Percentiles == "" {
		t.Fatal("a rota tem de declarar o metodo dos percentis")
	}

	// (3) O DISCRIMINADOR: os eixos que os eventos NAO trazem sao declarados NAO MEDIDOS.
	// Sem missoes executadas, TODOS o sao -- e a rota nao pode devolver uma lista vazia, porque
	// *uma omissao silenciosa le-se como zero*.
	if len(payload.Metrics.AxesNotMeasured) == 0 {
		t.Fatalf("sem eventos medidos os eixos tem de ser declarados NAO MEDIDOS; veio %v",
			payload.Metrics.AxesNotMeasured)
	}
	achouEnergia := false
	for _, eixo := range payload.Metrics.AxesNotMeasured {
		if eixo == "energia" {
			achouEnergia = true
		}
	}
	if !achouEnergia {
		t.Fatalf("`energia` tem de constar em axes_not_measured; veio %v", payload.Metrics.AxesNotMeasured)
	}

	// (4) Missões ilegiveis e' um numero, nao um campo ausente.
	if payload.MissionsUnreadable < 0 {
		t.Fatalf("missions_unreadable = %d", payload.MissionsUnreadable)
	}
}

// TestMetricsTelemetryNaoSubstituiMetrics: as DUAS rotas existem e sao diferentes.
func TestMetricsTelemetryNaoSubstituiMetrics(t *testing.T) {
	runtime := runtimeComModelos(t, registryDeRotas(t))
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{runtime: runtime}).register(engine)

	contadores := httptest.NewRecorder()
	engine.ServeHTTP(contadores, httptest.NewRequest(http.MethodGet, "/api/agent/v1/metrics", nil))
	agregado := httptest.NewRecorder()
	engine.ServeHTTP(agregado, httptest.NewRequest(http.MethodGet, "/api/agent/v1/metrics/telemetry", nil))

	if contadores.Code != http.StatusOK || agregado.Code != http.StatusOK {
		t.Fatalf("/metrics -> %d, /metrics/telemetry -> %d", contadores.Code, agregado.Code)
	}
	if contadores.Body.String() == agregado.Body.String() {
		t.Fatal("/metrics e /metrics/telemetry devolvem o MESMO corpo -- *dois conteudos " +
			"diferentes com o mesmo nome seriam duas verdades sobre a mesma pergunta*, mas " +
			"dois conteudos IGUAIS com nomes diferentes sao uma rota a mais sem funcao")
	}
}
