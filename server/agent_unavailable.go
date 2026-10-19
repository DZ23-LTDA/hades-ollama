// Fallback do Agent Center quando o runtime nao arranca.
//
// O DEFEITO QUE ISTO CORRIGE (E-1378, P2)
//
// `routes.go` degrada em vez de recusar arrancar quando o runtime do agente falha -- o que esta' certo, porque
// o Ollama core tem de continuar a servir. Mas o efeito observavel era:
//
//	404  /api/agent/v1/agents
//	404  /api/agent/v1/agents/catalog
//	404  ... (todas)
//
// Um 404 diz «esta rota nao existe». A verdade era «a rota existe e o runtime nao arrancou». O motivo ficava
// num log que o utilizador nao le'.
//
//	*Um estado que mente e' pior do que nenhum.*
//	*Uma lista vazia e um erro nao sao a mesma coisa.*
//
// # O QUE ESTE FICHEIRO FAZ
//
// Monta um handler em `/api/agent/v1/*rest` que responde **503** com o motivo REAL, e um 503 e' a resposta
// honesta: o servidor esta' vivo, esta capacidade e' que nao esta'. Nao ha' aqui nenhuma rota funcional a
// esconder -- o fallback so' e' registado quando nenhuma rota do agent foi montada.
package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// agentAPINaoMontada devolve o handler de indisponibilidade, com o motivo que o arranque deu.
//
// Nunca inventa um motivo: se a causa for desconhecida, diz que e' desconhecida.
//
//	*Um numero publicado sem a qualidade ao lado e' narrativa.* -- e um motivo inventado tambem.
func agentAPINaoMontada(motivo string) gin.HandlerFunc {
	if motivo == "" {
		motivo = "o runtime do agente nao foi inicializado e nao registou um motivo"
	}
	return func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			// Nomes diferentes dos de sucesso, de proposito: um cliente que so' olhe para o corpo nao deve
			// poder confundir uma indisponibilidade com um resultado vazio.
			"error":  "agent_api_unavailable",
			"reason": motivo,
			"path":   c.Request.URL.Path,
			"note": "o servidor está a servir o Ollama core; o Agent Center não foi montado. " +
				"O estado é INDISPONÍVEL, não vazio nem ausente.",
		})
	}
}
