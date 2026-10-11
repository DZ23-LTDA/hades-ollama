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

// §11.1: TODO comando do catálogo precisa mapear para rota real. Este teste
// institucionaliza a regra: em vez de auditar à mão, ele compara o `Backend`
// declarado com as rotas REALMENTE registradas no roteador.
//
// Dois defeitos reais motivaram este teste: `/crm` apontava para nada (era marcado
// indisponível, honestamente) e `/research` estava marcado indisponível **enquanto a
// rota e o motor de pesquisa existiam** — capacidade real escondida do usuário.
func TestEveryAvailableCommandMapsToARegisteredRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)

	registered := map[string]bool{}
	for _, route := range engine.Routes() {
		registered[route.Method+" "+routeShape(route.Path)] = true
	}

	commands := agent.DefaultSlashRegistry()
	if len(commands) == 0 {
		t.Fatal("o registro de comandos não pode estar vazio")
	}
	for _, spec := range commands {
		if !spec.Available {
			// Indisponível só é aceitável com MOTIVO declarado: um comando mudo
			// escondido é pior do que um comando ausente.
			if strings.TrimSpace(spec.UnavailableReason) == "" {
				t.Fatalf("comando %s está indisponível SEM motivo declarado", spec.Name)
			}
			continue
		}
		method, path, ok := splitBackendRoute(spec.Backend)
		if !ok {
			t.Fatalf("comando %s está marcado disponível com Backend inválido: %q", spec.Name, spec.Backend)
		}
		if !registered[method+" "+routeShape(path)] {
			t.Fatalf("comando %s aponta para rota INEXISTENTE: %s %s", spec.Name, method, path)
		}
	}
}

// routeShape normaliza nomes de parâmetro para que `:id` e `:approval_id` sejam
// comparados pelo FORMATO da rota, e não pelo nome escolhido em cada lugar.
func routeShape(path string) string {
	segments := strings.Split(path, "/")
	for index, segment := range segments {
		if strings.HasPrefix(segment, ":") || strings.HasPrefix(segment, "*") {
			segments[index] = ":"
		}
	}
	return strings.Join(segments, "/")
}

func splitBackendRoute(backend string) (string, string, bool) {
	fields := strings.Fields(strings.TrimSpace(backend))
	if len(fields) != 2 {
		return "", "", false
	}
	method := strings.ToUpper(fields[0])
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return "", "", false
	}
	return method, fields[1], true
}

// A rota da pesquisa existe e responde de verdade: prova que o comando deixou de
// esconder capacidade real.
func TestResearchRouteAnswersThroughTheRealEngine(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
			return
		}
		_, _ = w.Write([]byte("<html><head><title>Fonte de teste</title></head><body>conteudo relevante sobre o tema</body></html>"))
	}))
	t.Cleanup(source.Close)

	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatal(err)
	}
	runtime.Research().AllowHTTPForTests = true
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	group := engine.Group("/api/agent/v1")
	group.POST("/research", api.research)

	recorder := httptest.NewRecorder()
	body := `{"query":"tema","urls":["` + source.URL + `/fonte"],"respect_robots":true}`
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/research", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("pesquisa precisa responder 200, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	var report agent.ResearchReport
	if err := json.Unmarshal(recorder.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Sources) != 1 {
		t.Fatalf("a pesquisa precisa devolver a fonte: %+v", report.Sources)
	}
	if !strings.Contains(report.Sources[0].Text, "conteudo relevante") {
		t.Fatalf("o conteúdo da fonte precisa ser o real: %q", report.Sources[0].Text)
	}
}

// O status do Telegram precisa ser HONESTO: sem credencial, NOT_CONFIGURED com o
// motivo real — nunca "pronto para usar".
func TestTelegramStatusIsHonestAndSendRequiresConsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatal(err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	group := engine.Group("/api/agent/v1")
	group.GET("/telegram/status", api.telegramStatus)
	group.POST("/telegram/send", api.telegramSend)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/telegram/status", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status precisa responder 200, veio %d", recorder.Code)
	}
	var status struct {
		Configured        bool   `json:"configured"`
		GateStatus        string `json:"gate_status"`
		ConsentRequired   bool   `json:"consent_required"`
		UnavailableReason string `json:"unavailable_reason"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Configured || status.GateStatus == "PASS" {
		t.Fatalf("sem credencial o canal NÃO pode aparecer configurado: %+v", status)
	}
	if !status.ConsentRequired {
		t.Fatal("o status precisa declarar que consentimento é exigido")
	}
	if strings.TrimSpace(status.UnavailableReason) == "" {
		t.Fatalf("a indisponibilidade precisa vir com motivo real: %+v", status)
	}

	// Envio sem credencial falha de forma explícita e NÃO devolve sucesso.
	sendRecorder := httptest.NewRecorder()
	sendRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/telegram/send",
		strings.NewReader(`{"chat_id":"123456","text":"mensagem de teste"}`))
	sendRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(sendRecorder, sendRequest)
	if sendRecorder.Code == http.StatusOK {
		t.Fatalf("não pode haver sucesso sem credencial e sem consentimento: %s", sendRecorder.Body.String())
	}
	if !strings.Contains(sendRecorder.Body.String(), "error") {
		t.Fatalf("a falha precisa ser explícita: %s", sendRecorder.Body.String())
	}
	// E a resposta de erro não pode vazar token nem aparecer como enviada.
	if strings.Contains(sendRecorder.Body.String(), "TELEGRAM_BOT_TOKEN=") {
		t.Fatal("a resposta não pode conter a credencial")
	}
	if strings.Contains(sendRecorder.Body.String(), "message_id") {
		t.Fatalf("resposta de erro não pode parecer envio: %s", sendRecorder.Body.String())
	}
}

func TestTelegramAndResearchRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	found := map[string]bool{}
	for _, route := range engine.Routes() {
		found[route.Method+" "+route.Path] = true
	}
	for _, want := range []string{
		"GET /api/agent/v1/telegram/status",
		"POST /api/agent/v1/telegram/send",
		"POST /api/agent/v1/research",
	} {
		if !found[want] {
			t.Fatalf("rota %s não está registrada", want)
		}
	}
}
