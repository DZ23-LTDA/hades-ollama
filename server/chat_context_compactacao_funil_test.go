package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/internal/agent"
)

// TESTE AO NÍVEL DO FUNIL (E-585) — o que a secção 10 exige para a compactação.
//
// O E-583 provou a unidade e o E-584 ligou-a. Faltava medir o CAMINHO REAL: pedido HTTP
// -> middleware -> compactação -> corpo reescrito, com o bruto em disco. Uma unidade
// testada e uma ligação a compilar não provam que o pedido que CHEGA ao handler já vai
// compactado, nem que o material coberto ficou recuperável.
//
// A INJEÇÃO DO TETO É O GATILHO: sem `OLLAMA_AGENT_CHAT_CONTEXT_TOKENS` a decisão é "não
// declarado" e NADA muda - é essa a garantia de que ligar isto não altera o produto para
// quem não o configurar.

func TestChatContextCompactaQuandoOTetoEstaDeclarado(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Teto minúsculo: qualquer conversa real fica acima dele.
	t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_TOKENS", "1")
	t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_RESERVE_TOKENS", "0")

	raiz := t.TempDir()
	var capturado []byte
	engine := gin.New()
	engine.POST("/api/agent/v1/chat", func(c *gin.Context) {
		c.Set("agent.organization", agent.Organization{ID: "org-a"})
		withAgentChatContext(raiz)(c)
	}, chatContextRecorder(&capturado))

	// 10 mensagens, para haver faixa antiga a resumir (a janela padrão é 6).
	var corpo strings.Builder
	corpo.WriteString(`{"model":"qwen2.5:7b","messages":[`)
	for i := 0; i < 10; i++ {
		if i > 0 {
			corpo.WriteString(",")
		}
		papel := "user"
		if i%2 == 1 {
			papel = "assistant"
		}
		corpo.WriteString(`{"role":"` + papel + `","content":"mensagem ` + string(rune('a'+i)) + ` do historico"}`)
	}
	corpo.WriteString(`]}`)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/chat", strings.NewReader(corpo.String()))
	request.Header.Set("Content-Type", "application/json")
	// O PROJETO VEM DO CABEÇALHO: sem ele o caminho do histórico é recusado (falha fechado)
	// e a compactação não acontece - por isso o teste tem de o declarar.
	request.Header.Set("X-Hades-Project", "proj-1")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var recebido api.ChatRequest
	if err := json.Unmarshal(capturado, &recebido); err != nil {
		t.Fatalf("json: %v (body=%s)", err, string(capturado))
	}

	// O CORPO QUE CHEGA AO HANDLER TEM DE ESTAR COMPACTADO: preâmbulo + resumo + 6 intactas.
	if len(recebido.Messages) != 8 {
		t.Fatalf("mensagens no handler = %d, esperado 8 (preâmbulo + resumo + 6 intactas)", len(recebido.Messages))
	}
	juntas := ""
	for _, m := range recebido.Messages {
		juntas += m.Content + "\n"
	}
	if !strings.Contains(juntas, agent.MarcadorDeResumo) {
		t.Fatalf("o corpo não traz a mensagem de resumo:\n%s", juntas)
	}
	// As últimas mensagens seguem INTACTAS: o conteúdo original tem de estar lá.
	if !strings.Contains(juntas, "mensagem j do historico") {
		t.Fatal("a última mensagem original desapareceu do corpo enviado")
	}
	// A SUBSTITUIÇÃO É ESTRUTURAL, E NÃO APAGAMENTO DE TEXTO. O digest é FIEL e COPIA o
	// material coberto, logo o texto antigo aparece DENTRO do resumo - isso é o que
	// "resumo fiel" significa. O que tem de desaparecer é a MENSAGEM original como
	// unidade própria. A primeira versão deste teste procurou a substring e reprovou o
	// comportamento correcto: o instrumento é que estava errado.
	for _, m := range recebido.Messages {
		if m.Content == "mensagem a do historico" {
			t.Fatal("a mensagem original continua no corpo como unidade própria: não foi substituída")
		}
	}

	// O BRUTO TEM DE ESTAR EM DISCO, no caminho derivado de organização e projeto.
	caminho := filepath.Join(raiz, "historico", "org-a", "proj-1.jsonl")
	if _, err := os.Stat(caminho); err != nil {
		t.Fatalf("o bruto não foi gravado em %q: %v", caminho, err)
	}
	historico, err := agent.AbrirHistoricoBruto(caminho)
	if err != nil {
		t.Fatalf("abrir o bruto gravado: %v", err)
	}
	defer historico.Fechar()
	registos, err := historico.LerTudo()
	if err != nil {
		t.Fatal(err)
	}
	if len(registos) != 4 {
		t.Fatalf("o bruto tem %d registos; a faixa coberta era de 4", len(registos))
	}
	if err := agent.VerificarCadeia(registos); err != nil {
		t.Fatalf("cadeia do bruto quebrada: %v", err)
	}
	// O MATERIAL COBERTO É RECUPERÁVEL a partir do bruto — é isto que torna a perda
	// "controlada" em vez de perda.
	if registos[0].Conteudo != "mensagem a do historico" {
		t.Fatalf("o primeiro registo do bruto não é a primeira mensagem coberta: %q", registos[0].Conteudo)
	}
}

// O CONTRAPONTO, e é ele que protege o produto: SEM teto declarado, o corpo segue IGUAL
// ao que o cliente enviou. Ligar a compactação não pode mudar nada para quem não a
// configurou.
func TestChatContextNaoCompactaSemTetoDeclarado(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_TOKENS", "")
	t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_RESERVE_TOKENS", "")

	raiz := t.TempDir()
	var capturado []byte
	engine := gin.New()
	engine.POST("/api/agent/v1/chat", func(c *gin.Context) {
		c.Set("agent.organization", agent.Organization{ID: "org-a"})
		withAgentChatContext(raiz)(c)
	}, chatContextRecorder(&capturado))

	var corpo strings.Builder
	corpo.WriteString(`{"model":"qwen2.5:7b","messages":[`)
	for i := 0; i < 10; i++ {
		if i > 0 {
			corpo.WriteString(",")
		}
		corpo.WriteString(`{"role":"user","content":"mensagem ` + string(rune('a'+i)) + ` do historico"}`)
	}
	corpo.WriteString(`]}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/chat", strings.NewReader(corpo.String()))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Hades-Project", "proj-1")
	engine.ServeHTTP(recorder, request)

	var recebido api.ChatRequest
	if err := json.Unmarshal(capturado, &recebido); err != nil {
		t.Fatalf("json: %v", err)
	}
	// Preâmbulo + as 10 originais: NENHUMA substituição.
	if len(recebido.Messages) != 11 {
		t.Fatalf("mensagens = %d, esperado 11 (preâmbulo + 10 intactas)", len(recebido.Messages))
	}
	for _, m := range recebido.Messages {
		if strings.Contains(m.Content, agent.MarcadorDeResumo) {
			t.Fatal("compactou SEM teto declarado")
		}
	}
	// E nenhum histórico foi criado: sem compactação não há material a guardar.
	if _, err := os.Stat(filepath.Join(raiz, "historico")); err == nil {
		t.Fatal("criou histórico sem ter compactado")
	}
}
