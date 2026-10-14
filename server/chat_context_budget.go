package server

import (
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/internal/agent"
)

// ORÇAMENTO DE CONTEXTO NO FUNIL DO CHAT (complemento universal de memória, §2).
//
// ONDE ISTO SE LIGA: `withAgentChatContext` (chat_context.go:113), entre a construção
// do preâmbulo (:139) e a injeção no corpo (:146) — as mensagens já estão em mão e o
// pedido ainda não foi enviado ao modelo. É o funil ÚNICO de produção do chat
// (`agent_routes.go:599`), e não o protocolo Responses/Codex, que era o único sítio
// onde existia compactação.
//
// O QUE ESTE FICHEIRO NÃO FAZ: não inventa um teto de contexto. Sem
// `OLLAMA_AGENT_CHAT_CONTEXT_TOKENS` declarado, o orçamento é 0, o gatilho responde
// "não declarado" e o comportamento do chat fica EXACTAMENTE como estava. Um limite
// inventado em código seria uma decisão de produto tomada por engano.

// ChaveDeOrcamentoDeContexto é a chave do gin.Context onde a decisão fica acessível a
// quem vier depois do middleware (handler, log ou uma tela de Cowork). Exposta para
// que a decisão seja VERIFICÁVEL, e não só registada num log que ninguém lê.
const ChaveDeOrcamentoDeContexto = "hades.chat.contexto.orcamento"

// DecisaoDeOrcamento é a decisão tomada no funil, com a medição que a sustenta.
type DecisaoDeOrcamento struct {
	Uso       agent.OrcamentoDeTokens    `json:"uso"`
	Gatilho   agent.GatilhoDeCompactacao `json:"gatilho"`
	Compactar bool                       `json:"compactar"`
	Motivo    string                     `json:"motivo"`
}

// OrcamentoDeContextoDoChat lê o teto declarado pelo operador. Zero significa NÃO
// DECLARADO — e é o padrão, de propósito.
func OrcamentoDeContextoDoChat() agent.GatilhoDeCompactacao {
	g := agent.GatilhoDeCompactacao{}
	if bruto := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_CHAT_CONTEXT_TOKENS")); bruto != "" {
		if v, err := strconv.Atoi(bruto); err == nil && v > 0 {
			g.OrcamentoTokens = v
		}
	}
	if bruto := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_CHAT_CONTEXT_RESERVE_TOKENS")); bruto != "" {
		if v, err := strconv.Atoi(bruto); err == nil && v >= 0 {
			g.ReservaTokens = v
		}
	}
	return g
}

// MedirUsoDoContexto soma o preâmbulo e as mensagens com a estimativa ROTULADA do
// pacote do agente. Devolve o orçamento inteiro para que o rótulo (`Exata=false`)
// viaje junto — um número sem rótulo seria precisão falsa.
func MedirUsoDoContexto(preamble string, mensagens []api.Message) agent.OrcamentoDeTokens {
	uso := agent.EstimarTokens(preamble)
	for _, m := range mensagens {
		uso = agent.SomarTokens(uso, agent.EstimarTokens(m.Content))
	}
	return uso
}

// DecidirOrcamentoDoChat é a função pura que o middleware chama: medir e decidir,
// sempre com motivo. Não compacta nada — a compactação em si é o passo seguinte, e
// declará-la feita sem o histórico bruto gravado primeiro seria a mentira que o §2
// proíbe.
func DecidirOrcamentoDoChat(preamble string, mensagens []api.Message, gatilho agent.GatilhoDeCompactacao) DecisaoDeOrcamento {
	uso := MedirUsoDoContexto(preamble, mensagens)
	compactar, motivo := gatilho.PrecisaCompactar(uso)
	return DecisaoDeOrcamento{Uso: uso, Gatilho: gatilho, Compactar: compactar, Motivo: motivo}
}

// RegistarDecisaoDeOrcamento deixa a decisão acessível a quem vier depois.
func RegistarDecisaoDeOrcamento(c *gin.Context, d DecisaoDeOrcamento) {
	if c == nil {
		return
	}
	c.Set(ChaveDeOrcamentoDeContexto, d)
}

// DecisaoDeOrcamentoRegistada lê a decisão, e DIZ se não a encontrou: ausência de
// medição não pode ser lida como "dentro do orçamento".
func DecisaoDeOrcamentoRegistada(c *gin.Context) (DecisaoDeOrcamento, bool) {
	if c == nil {
		return DecisaoDeOrcamento{}, false
	}
	v, ok := c.Get(ChaveDeOrcamentoDeContexto)
	if !ok {
		return DecisaoDeOrcamento{}, false
	}
	d, ok := v.(DecisaoDeOrcamento)
	return d, ok
}
