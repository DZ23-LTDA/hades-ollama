package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// Achado do estágio 18 (integridade de produto, §11.3): o endpoint
// /api/agent/v1/personalization respondia {"status":"updated"} sem persistir
// NADA e devolvia no GET instruções, tom e memória fixos no código. Era um
// controle que fingia funcionar — e, pior, quando a API agêntica não subia
// (`agentAPI == nil`) o grupo era registrado SEM middleware de autenticação e
// origem, deixando o caminho aberto.
//
// Estes testes travam as duas verdades: a rota não mente mais e não existe sem
// o middleware que a protege.
func TestDesktopPersonalizationEndpointDoesNotFakeSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{}
	engine := gin.New()
	server.registerDesktopLocalRoutes(engine, &agentAPI{authRequired: false})

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, "/api/agent/v1/personalization", strings.NewReader(`{"tone":"x"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://localhost:3000")
		engine.ServeHTTP(recorder, request)

		if recorder.Code == http.StatusOK {
			t.Fatalf("%s respondeu 200 para um endpoint sem persistência: %s", method, recorder.Body.String())
		}
		body := recorder.Body.String()
		if strings.Contains(body, `"status":"updated"`) {
			t.Fatalf("%s ainda afirma sucesso sem gravar nada: %s", method, body)
		}
		if strings.Contains(body, "arquitetura limpa") {
			t.Fatalf("%s ainda devolve conteúdo fabricado: %s", method, body)
		}
		if recorder.Code != http.StatusNotImplemented {
			t.Fatalf("%s deveria responder 501 (não implementado), veio %d", method, recorder.Code)
		}
		var payload map[string]string
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(payload["detail"], "custom_instructions") {
			t.Fatalf("a resposta deveria indicar o caminho real das instruções: %s", body)
		}
	}
}

func TestDesktopPersonalizationIsAbsentWithoutAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{}
	engine := gin.New()
	// agentAPI nil é o modo degradado (auth/store indisponível): nada sob
	// /api/agent/v1 pode ser registrado sem middleware.
	server.registerDesktopLocalRoutes(engine, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/personalization", strings.NewReader(`{"tone":"x"}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("sem middleware a rota deveria não existir (404), veio %d (%s)", recorder.Code, recorder.Body.String())
	}
}
