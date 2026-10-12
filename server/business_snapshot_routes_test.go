package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Estágio 32 pela ROTA: registrar amostra, ler a série e tratar erros de cliente.
//
// Nota deliberada: o handler usa o relógio REAL, então a segunda chamada pode cair na hora seguinte
// se o teste cruzar a virada de hora — nesse caso `created` é legitimamente `true`. Em vez de
// escrever uma asserção que falha de vez em quando, este teste afirma o que é SEMPRE verdade
// (códigos coerentes com `created`, e a série refletindo o que foi respondido); a semântica exata de
// idempotência por janela é verificada no store, com o instante injetado.
func TestBusinessSnapshotRoutesRecordListAndValidate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		WorkspaceRoot: shortServerRoot(t),
		DataRoot:      shortServerRoot(t),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	engine := gin.New()
	engine.GET("/api/agent/v1/companies/:id/business-snapshots", api.businessSnapshots)
	engine.POST("/api/agent/v1/companies/:id/business-snapshots", api.recordBusinessSnapshot)

	company, err := runtime.CompanyStore().CreateRequest(agent.CompanyCreateRequest{
		Name: "Loja de série", Mission: "vender", Currency: "BRL",
	}, agent.LocalOrganizationID)
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/agent/v1/companies/" + company.ID + "/business-snapshots"

	record := func() (int, map[string]any) {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, base, nil))
		var payload map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("resposta inválida (%d): %s", recorder.Code, recorder.Body.String())
		}
		return recorder.Code, payload
	}

	// Primeira amostra: criada (201) porque a série estava vazia.
	status, payload := record()
	if status != http.StatusCreated {
		t.Fatalf("primeira amostra precisa ser 201, veio %d", status)
	}
	if created, _ := payload["created"].(bool); !created {
		t.Fatalf("primeira amostra precisa vir como criação: %+v", payload)
	}
	snapshot, _ := payload["snapshot"].(map[string]any)
	if snapshot == nil || snapshot["organization_id"] != company.OrganizationID {
		t.Fatalf("amostra precisa ser da organização da sessão: %+v", payload)
	}
	// Com uma amostra, a série existe mas a tendência DECLARA ausência de base.
	trend, _ := payload["trend"].(map[string]any)
	if trend == nil || trend["sem_base"] != true || trend["motivo"] == "" {
		t.Fatalf("com uma amostra a tendência precisa declarar ausência de base: %+v", trend)
	}
	if deltas, _ := trend["deltas"].([]any); len(deltas) != 0 {
		t.Fatalf("sem base não pode haver delta: %+v", deltas)
	}

	// Segunda amostra: o código reflete a verdade de `created` (200 idempotente ou 201 nova janela).
	status2, payload2 := record()
	created2, _ := payload2["created"].(bool)
	if created2 && status2 != http.StatusCreated {
		t.Fatalf("criação precisa ser 201: %d", status2)
	}
	if !created2 && status2 != http.StatusOK {
		t.Fatalf("reaproveitamento de janela precisa ser 200: %d", status2)
	}
	trend2, _ := payload2["trend"].(map[string]any)
	count, _ := trend2["snapshots"].(float64)
	if count != 1 && count != 2 {
		t.Fatalf("a série precisa ter 1 ou 2 amostras: %v", count)
	}

	// GET devolve a mesma série que o POST informou.
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, base, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("leitura da série precisa ser 200, veio %d", recorder.Code)
	}
	var listed agent.BusinessTrend
	if err := json.Unmarshal(recorder.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if float64(listed.Snapshots) != count || len(listed.History) != int(count) {
		t.Fatalf("a leitura precisa refletir o que foi registrado: %d vs %v", listed.Snapshots, count)
	}
	if listed.Retention == 0 {
		t.Fatal("a resposta precisa declarar a retenção máxima")
	}

	// `from` inválido: 400 (erro de cliente, não série vazia).
	badFrom := httptest.NewRecorder()
	engine.ServeHTTP(badFrom, httptest.NewRequest(http.MethodGet, base+"?from=ontem", nil))
	if badFrom.Code != http.StatusBadRequest {
		t.Fatalf("from inválido precisa ser 400, veio %d", badFrom.Code)
	}

	// Empresa inexistente: 404, sem vazar série de outra organização.
	missing := httptest.NewRecorder()
	engine.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/agent/v1/companies/co_inexistente/business-snapshots", nil))
	// NÃO INFERÊNCIA (E-190): a empresa inexistente recebia 404 e a empresa de OUTRO tenant recebia 403 — a
	// diferença revelava a existência do ID. A recusa agora é a MESMA (403) para os dois casos; a propriedade é
	// provada de forma genérica em `company_scope_non_inference_test.go`.
	if missing.Code != http.StatusForbidden {
		t.Fatalf("empresa inexistente precisa ser a recusa unificada 403, veio %d", missing.Code)
	}
}
