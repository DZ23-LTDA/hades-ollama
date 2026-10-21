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

const (
	rotaDoAcionamentoDoPare = "/api/agent/v1/build/pare"
	rotaDoRearmeDoPare      = "/api/agent/v1/build/pare/rearmar"
)

// pedirComandoDoPare monta o router com (ou sem) sessao e dispara um comando.
//
// A sessao e' o UNICO caminho de identidade: `sessaoDeBuild` poe `agent.organization`
// no contexto, como o middleware real faz.
func pedirComandoDoPare(t *testing.T, rota, metodo, corpo string, sessao string) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if sessao != "" {
		router.Use(sessaoDeBuild(sessao))
	}
	api := &agentAPI{}
	router.POST(rotaDoAcionamentoDoPare, api.acionarOPare)
	router.POST(rotaDoRearmeDoPare, api.rearmarOPare)

	pedido := httptest.NewRequest(metodo, rota, strings.NewReader(corpo))
	pedido.Header.Set("Content-Type", "application/json")
	resposta := httptest.NewRecorder()
	router.ServeHTTP(resposta, pedido)

	var saida map[string]any
	if err := json.Unmarshal(resposta.Body.Bytes(), &saida); err != nil {
		t.Fatalf("resposta nao e' JSON: %v (corpo %q)", err, resposta.Body.String())
	}
	return resposta.Code, saida
}

// limparPare garante que um teste que aciona o travao global o desarma no fim.
func limparPare(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		if pareDoServidor.Bloqueia() {
			_ = pareDoServidor.Rearmar("teste", "limpeza do teste")
		}
	})
}

// NEGATIVO 1: sem autoridade configurada, NINGUEM aciona. Nao e' 501: o mecanismo
// existe, o que falta e' a decisao de quem o pode usar.
func TestAcionarPareSemAutoridadeConfiguradaRecusa(t *testing.T) {
	limparPare(t)
	t.Setenv(variavelDaAutoridadeDoPare, "")
	t.Setenv(variavelDoCaminhoDoPare, "")

	codigo, corpo := pedirComandoDoPare(t, rotaDoAcionamentoDoPare, http.MethodPost,
		`{"motivo":"incidente em producao"}`, "org-teste")
	if codigo != http.StatusForbidden {
		t.Fatalf("sem autoridade devia dar 403, veio %d (%v)", codigo, corpo)
	}
	if !strings.Contains(strings.ToLower(corpo["erro"].(string)), "autoridade") {
		t.Fatalf("o erro tem de nomear a autoridade em falta: %v", corpo)
	}
	if pareDoServidor.Bloqueia() {
		t.Fatalf("recusa NAO pode acionar o pare")
	}
}

// NEGATIVO 2: com autoridade configurada, uma identidade DIFERENTE nao aciona.
func TestAcionarPareIdentidadeForaDaAutoridadeRecusa(t *testing.T) {
	limparPare(t)
	t.Setenv(variavelDaAutoridadeDoPare, "org-operador")
	t.Setenv(variavelDoCaminhoDoPare, "")

	codigo, corpo := pedirComandoDoPare(t, rotaDoAcionamentoDoPare, http.MethodPost,
		`{"motivo":"tentativa de terceiro"}`, "org-intrusa")
	if codigo != http.StatusForbidden {
		t.Fatalf("identidade fora da autoridade devia dar 403, veio %d (%v)", codigo, corpo)
	}
	if pareDoServidor.Bloqueia() {
		t.Fatalf("recusa NAO pode acionar o pare")
	}
}

// NEGATIVO 3: a identidade do CORPO e' ignorada. Um corpo a dizer "por_quem" nao
// muda quem o servidor considera ser o autor -- a sessao manda.
func TestAcionarPareIgnoraIdentidadeDoCorpo(t *testing.T) {
	limparPare(t)
	t.Setenv(variavelDaAutoridadeDoPare, "org-operador")
	t.Setenv(variavelDoCaminhoDoPare, "")

	codigo, corpo := pedirComandoDoPare(t, rotaDoAcionamentoDoPare, http.MethodPost,
		`{"motivo":"forjar autoria","por_quem":"org-operador","organizacao":"org-operador"}`, "org-intrusa")
	if codigo != http.StatusForbidden {
		t.Fatalf("identidade do corpo NAO pode autorizar: esperado 403, veio %d (%v)", codigo, corpo)
	}
	if pareDoServidor.Bloqueia() {
		t.Fatalf("tentativa de forja NAO pode acionar o pare")
	}
}

// NEGATIVO 4: sem motivo nao ha' acionamento -- e nao ha' registo do porque.
func TestAcionarPareSemMotivoRecusa(t *testing.T) {
	limparPare(t)
	t.Setenv(variavelDaAutoridadeDoPare, "org-operador")
	t.Setenv(variavelDoCaminhoDoPare, "")

	codigo, corpo := pedirComandoDoPare(t, rotaDoAcionamentoDoPare, http.MethodPost,
		`{"motivo":"   "}`, "org-operador")
	if codigo != http.StatusBadRequest {
		t.Fatalf("motivo vazio devia dar 400, veio %d (%v)", codigo, corpo)
	}
	if pareDoServidor.Bloqueia() {
		t.Fatalf("sem motivo o pare NAO pode ser acionado")
	}
}

// POSITIVO: identidade autorizada + motivo => aciona, bloqueia e registra quem e por que.
func TestAcionarPareAutorizadoAcionaEBloqueia(t *testing.T) {
	limparPare(t)
	t.Setenv(variavelDaAutoridadeDoPare, "org-operador")
	t.Setenv(variavelDoCaminhoDoPare, "")

	codigo, corpo := pedirComandoDoPare(t, rotaDoAcionamentoDoPare, http.MethodPost,
		`{"motivo":"incidente em producao"}`, "org-operador")
	if codigo != http.StatusOK {
		t.Fatalf("autorizado devia dar 200, veio %d (%v)", codigo, corpo)
	}
	if !pareDoServidor.Bloqueia() {
		t.Fatalf("depois de acionar, o pare tem de bloquear execucao")
	}
	pare, _ := corpo["pare"].(map[string]any)
	if acionado, _ := pare["acionado"].(bool); !acionado {
		t.Fatalf("o corpo devia reflectir o estado acionado: %v", pare)
	}
	if motivo, _ := pare["motivo"].(string); !strings.Contains(motivo, "incidente") {
		t.Fatalf("o estado tem de guardar o motivo: %v", pare)
	}
}

// POSITIVO/NEGATIVO do rearme: sem justificacao recusa; com justificacao desarma e registra.
func TestRearmarPareExigeJustificacaoEDesarma(t *testing.T) {
	limparPare(t)
	t.Setenv(variavelDaAutoridadeDoPare, "org-operador")
	t.Setenv(variavelDoCaminhoDoPare, "")

	if err := pareDoServidor.Acionar("incidente", "org-operador"); err != nil {
		t.Fatalf("preparar: %v", err)
	}
	codigo, corpo := pedirComandoDoPare(t, rotaDoRearmeDoPare, http.MethodPost,
		`{"justificacao":""}`, "org-operador")
	if codigo != http.StatusBadRequest {
		t.Fatalf("rearme sem justificacao devia dar 400, veio %d (%v)", codigo, corpo)
	}
	if !pareDoServidor.Bloqueia() {
		t.Fatalf("rearme recusado NAO pode desarmar o pare")
	}

	codigo, corpo = pedirComandoDoPare(t, rotaDoRearmeDoPare, http.MethodPost,
		`{"justificacao":"causa raiz corrigida e verificada"}`, "org-operador")
	if codigo != http.StatusOK {
		t.Fatalf("rearme autorizado devia dar 200, veio %d (%v)", codigo, corpo)
	}
	if pareDoServidor.Bloqueia() {
		t.Fatalf("depois do rearme o pare tem de deixar de bloquear")
	}
}

// DURABILIDADE: com HADES_PARE_ESTADO definido, o acionamento PERSISTE e sobrevive a
// uma instancia nova -- a ligacao a `GuardarEm`, que antes nao existia.
func TestAcionarParePersisteESobreviveAReinicio(t *testing.T) {
	limparPare(t)
	caminho := t.TempDir() + "/pare.json"
	t.Setenv(variavelDaAutoridadeDoPare, "org-operador")
	t.Setenv(variavelDoCaminhoDoPare, caminho)

	codigo, corpo := pedirComandoDoPare(t, rotaDoAcionamentoDoPare, http.MethodPost,
		`{"motivo":"queda do datastore"}`, "org-operador")
	if codigo != http.StatusOK {
		t.Fatalf("acionamento com persistencia devia dar 200, veio %d (%v)", codigo, corpo)
	}

	// Instancia NOVA: simula o reinicio do servidor.
	outro := agent.NovoPareDeEmergencia()
	if err := outro.CarregarDe(caminho); err != nil {
		t.Fatalf("carregar estado gravado: %v", err)
	}
	if !outro.Bloqueia() {
		t.Fatalf("o pare tem de sobreviver ao reinicio; o ficheiro %s nao o desarmou", caminho)
	}
	if motivo := outro.Estado().Motivo; !strings.Contains(motivo, "datastore") {
		t.Fatalf("o motivo tem de sobreviver ao reinicio, veio %q", motivo)
	}
}
