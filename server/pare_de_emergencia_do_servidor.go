package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// pareDoServidor e' o travao consultado pelos executores HTTP.
//
// # PORQUE UMA INSTANCIA DO SERVIDOR, E NAO UMA POR PEDIDO
//
// Um pare que nasce em cada pedido nunca esta' acionado -- nao e' um travao, e' uma variavel local com
// nome de travao. Para parar execucoes, o estado tem de sobreviver ao pedido.
//
// # O QUE AINDA FALTA, DITO AQUI
//
// Nao ha' rota para O ACIONAR. Ou seja: o executor HTTP ja' **consulta** o pare e `estadoDoPare`
// mostra o ESTADO, mas ninguem o consegue accionar em producao. Expor o accionamento e' uma decisao
// de superficie (quem pode parar tudo? com que autorizacao?) que esta fatia NAO toma.
var pareDoServidor = agent.NovoPareDeEmergencia()

// estadoDoPare atende `GET /build/pare` e devolve o ESTADO do travao.
//
// # LEITURA APENAS, POR DECISAO
//
// Sem isto, ninguem sabe se o sistema esta' parado -- e um travao cujo estado ninguem consegue ver e'
// indistinguivel de um travao que nao existe. O que esta fatia NAO faz e' deixar alguem acionar: essa
// rota exigiria responder primeiro a "quem pode parar tudo, com que credencial e com que registo".
//
// A identidade vem da SESSAO (`requireOrganizacaoDaSessao`), nunca do corpo nem da query.
func (a *agentAPI) estadoDoPare(c *gin.Context) {
	if _, ok := requireOrganizacaoDaSessao(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"pare": pareDoServidor.Estado()})
}
