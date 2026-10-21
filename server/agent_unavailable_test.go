// Teste do fallback de indisponibilidade do Agent Center (E-1378, P2).
//
// # O QUE ESTE TESTE DISCRIMINA
//
// Nao basta provar que o fallback responde 503. Tem de provar que a DIFERENCA existe: sem ele, a rota da'
// 404; com ele, da' 503 e diz por que.
//
//	*Um teste que mede o estado anterior aprova o defeito que devia travar.*
//
// Por isso os dois lados estao aqui: `semFallback` e' o router como era antes, `comFallback` e' o de agora, e
// o teste falha se ambos responderem o mesmo.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFallback503DizOPorque(t *testing.T) {
	gin.SetMode(gin.TestMode)
	motivo := "create agent store: mkdir C:\\Users\\x\\AppData\\Roaming\\ollama-agent\\data\\missions: Acesso negado."

	r := gin.New()
	r.Any("/api/agent/v1/*rest", agentAPINaoMontada(motivo))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agent/v1/agents/catalog", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503, veio %d", w.Code)
	}
	var corpo map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo ilegivel: %v", err)
	}
	if corpo["error"] != "agent_api_unavailable" {
		t.Fatalf("codigo de erro inesperado: %v", corpo["error"])
	}
	razao, _ := corpo["reason"].(string)
	if !strings.Contains(razao, "Acesso negado") {
		t.Fatalf("o motivo real nao chegou ao cliente: %q", razao)
	}
	if corpo["path"] != "/api/agent/v1/agents/catalog" {
		t.Fatalf("caminho errado no corpo: %v", corpo["path"])
	}
}

// O DISCRIMINANTE. Sem o fallback, 404; com ele, 503. Se os dois derem o mesmo, o teste falha.
func TestSemFallbackSeria404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	semFallback := gin.New()
	wSem := httptest.NewRecorder()
	semFallback.ServeHTTP(wSem, httptest.NewRequest(http.MethodGet, "/api/agent/v1/agents", nil))

	comFallback := gin.New()
	comFallback.Any("/api/agent/v1/*rest", agentAPINaoMontada("motivo qualquer"))
	wCom := httptest.NewRecorder()
	comFallback.ServeHTTP(wCom, httptest.NewRequest(http.MethodGet, "/api/agent/v1/agents", nil))

	if wSem.Code != http.StatusNotFound {
		t.Fatalf("sem fallback devia ser 404, veio %d", wSem.Code)
	}
	if wCom.Code != http.StatusServiceUnavailable {
		t.Fatalf("com fallback devia ser 503, veio %d", wCom.Code)
	}
	if wSem.Code == wCom.Code {
		t.Fatal("os dois respondem o mesmo: o fallback nao mudou nada e o 404 continua a mentir")
	}
}

// Motivo em falta nao vira motivo inventado.
func TestMotivoVazioNaoInventa(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/api/agent/v1/*rest", agentAPINaoMontada(""))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/agent/v1/agents", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503, veio %d", w.Code)
	}
	var corpo map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo ilegivel: %v", err)
	}
	razao, _ := corpo["reason"].(string)
	if razao == "" {
		t.Fatal("motivo vazio: o cliente nao sabe se o runtime caiu ou se a rota nao existe")
	}
	if !strings.Contains(razao, "nao registou um motivo") {
		t.Fatalf("devia declarar a ausencia de motivo, nao inventar: %q", razao)
	}
}

// A PROPRIEDADE DE QUE O CI DEPENDE (E-1423/E-1425).
//
// O job `agent-center-real` recusa correr se `GET /api/agent/v1/projects/detect` devolver 503, porque 503
// significa que o fallback foi registado -- ou seja, que o Agent Center NAO montou.
//
// A primeira versao deste teste assumia que o router daria preferencia ao caminho estatico e deixaria o
// wildcard apanhar so' o resto. MEDIDO: o gin NAO faz isso -- entra em panico:
//
//	catch-all wildcard '*rest' in new path '/api/agent/v1/*rest' conflicts with existing path
//	segment 'projects' in existing prefix '/api/agent/v1/projects'
//
// Ou seja, a garantia e' mais forte do que eu tinha escrito: o fallback e as rotas reais NAO conseguem
// coexistir; o servidor recusa arrancar em vez de responder 503 em silencio. E' isso que torna o 503 um
// sinal utilizavel: nao existe estado em que o Agent Center esteja montado e o 503 apareca por cima.
//
//	*Uma premissa que se assume nao se mede. Esta mediu-se e estava errada -- e a medicao e' que fica.*
func TestFallbackNaoCoexisteComRotaReal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	panico := func() (p any) {
		defer func() { p = recover() }()
		r := gin.New()
		r.GET("/api/agent/v1/projects/detect", func(c *gin.Context) { c.Status(http.StatusOK) })
		r.Any("/api/agent/v1/*rest", agentAPINaoMontada("nao devia ser alcancado"))
		return nil
	}

	p := panico()
	if p == nil {
		t.Fatal("o router aceitou o wildcard ao lado de uma rota real: o 503 deixaria de significar " +
			"«nao montou» e passaria a poder significar «montou e o fallback roubou a rota»")
	}
	// O valor recuperado pode ser `error` ou `string` conforme a versao do gin: o teste exige o TEXTO da
	// colisao, nao o tipo. *Uma assercao que depende do tipo do panico mede o gin, nao o contrato.*
	if texto := fmt.Sprint(p); !strings.Contains(texto, "conflicts with existing path") {
		t.Fatalf("o panico nao e' a colisao de rotas esperada: %v", p)
	}
}

// Um POST tambem tem de ser apanhado -- o painel usa POST em backup/restore/install.
func TestFallbackApanhaPOST(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/api/agent/v1/*rest", agentAPINaoMontada("motor em baixo"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/agent/v1/agents/install", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST devia dar 503, veio %d", w.Code)
	}
}
