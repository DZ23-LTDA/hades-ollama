package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// A rota de desenvolvimento criava SEMPRE uma organizacao nova, o que tornava impossivel
// pôr dois utilizadores no mesmo tenant -- e sem isso nao se prova "duas identidades leem a
// mesma memoria permitida" nem "sem scope nega e nao infere".
//
// As tres provas abaixo cobrem a mudanca: entra, mantem o comportamento antigo, e falha
// fechado com id inexistente.
func TestHTTPDevTokenEntraEmTenantExistente(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_AGENT_AUTH_DEV", "true")
	t.Setenv("OLLAMA_AGENT_AUTH_DEV_SECRET", "segredo-de-teste")
	t.Setenv("OLLAMA_HOST", "127.0.0.1:0")

	store, err := agent.NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewAuthStore: %v", err)
	}
	dono, err := store.CreateUser("dono@local", "Dono")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	organizacao, _, err := store.CreateOrganization("Org Partilhada", dono)
	if err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}

	api := &agentAPI{auth: store}
	router := gin.New()
	router.POST("/api/agent/v1/auth/dev/token", api.devToken)

	pedir := func(corpo string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/agent/v1/auth/dev/token", strings.NewReader(corpo))
		req.RemoteAddr = "127.0.0.1:54321"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Ollama-Agent-Dev-Secret", "segredo-de-teste")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// POSITIVO: entra no tenant existente e o token sai para ESSE tenant.
	rec := pedir(`{"email":"membro@local","name":"Membro","organization_id":"` + organizacao.ID + `"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("devia entrar no tenant existente, veio %d (%s)", rec.Code, rec.Body.String())
	}
	var resposta struct {
		AccessToken  string `json:"access_token"`
		Organization struct {
			ID string `json:"id"`
		} `json:"organization"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("resposta nao e' JSON: %v", err)
	}
	if resposta.Organization.ID != organizacao.ID {
		t.Fatalf("o token devia ser do tenant %q, veio %q", organizacao.ID, resposta.Organization.ID)
	}
	if resposta.AccessToken == "" {
		t.Fatal("devia devolver access_token")
	}

	// O membro tem de estar MESMO no tenant: Authenticate devolve a organizacao do token.
	if _, orgDoToken, _, err := store.Authenticate(resposta.AccessToken); err != nil {
		t.Fatalf("Authenticate: %v", err)
	} else if orgDoToken.ID != organizacao.ID {
		t.Fatalf("Authenticate devolveu tenant %q, esperava %q", orgDoToken.ID, organizacao.ID)
	}

	// NEGATIVO: id inexistente falha fechado (400), e NAO cria tenant com esse id.
	recMau := pedir(`{"email":"outro@local","name":"Outro","organization_id":"org-que-nao-existe"}`)
	if recMau.Code != http.StatusBadRequest {
		t.Fatalf("id inexistente devia dar 400, veio %d (%s)", recMau.Code, recMau.Body.String())
	}

	// POSITIVO (comportamento antigo intacto): sem organization_id cria organizacao nova.
	recNovo := pedir(`{"email":"nova@local","name":"Nova","organization":"Org Nova"}`)
	if recNovo.Code != http.StatusCreated {
		t.Fatalf("sem organization_id devia criar tenant novo, veio %d (%s)", recNovo.Code, recNovo.Body.String())
	}
	var respostaNova struct {
		Organization struct {
			ID string `json:"id"`
		} `json:"organization"`
	}
	if err := json.Unmarshal(recNovo.Body.Bytes(), &respostaNova); err != nil {
		t.Fatalf("resposta nova nao e' JSON: %v", err)
	}
	if respostaNova.Organization.ID == "" || respostaNova.Organization.ID == organizacao.ID {
		t.Fatalf("devia ser uma organizacao NOVA e distinta, veio %q", respostaNova.Organization.ID)
	}
}
