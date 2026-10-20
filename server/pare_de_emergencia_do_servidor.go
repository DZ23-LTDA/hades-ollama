package server

import (
	"net/http"
	"os"

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
// Nao ha' rota para O ACIONAR. O executor HTTP **consulta** o pare, `estadoDoPare` mostra o ESTADO e
// `adoptarParePersistido` recupera-o no arranque; mas ninguem o consegue accionar por HTTP. Expor o
// accionamento e' uma decisao de superficie (quem pode parar tudo? com que autorizacao?) por tomar.
var pareDoServidor = agent.NovoPareDeEmergencia()

// variavelDoCaminhoDoPare nomeia a variavel de ambiente que diz ONDE o estado do pare vive.
//
// # PORQUE UMA VARIAVEL, E NAO UM CAMINHO FIXO
//
// Um caminho fixo faria o servidor ler (e, mais tarde, escrever) num sitio que o operador nao escolheu --
// e um teste que corresse na mesma maquina adoptaria o estado real de quem a usa. Sem a variavel, a
// persistencia fica **desligada**, e desligada e' um estado explicito.
const variavelDoCaminhoDoPare = "HADES_PARE_ESTADO"

// adoptarParePersistido carrega o estado guardado, se a variavel de ambiente o indicar.
//
// # O QUE ISTO FECHA
//
// Sem esta chamada, `GuardarEm`/`CarregarDe` existiam e ninguem as usava: um reinicio desarmava o travao
// em silencio. Com ela, o estado guardado e' adoptado **no arranque**.
//
// # O QUE ISTO NAO FAZ
//
// Nao GRAVA. Gravar so' faz sentido quando o estado muda, e o estado so' muda por codigo (nao ha' rota
// para acionar). Quem acionar por codigo tem de chamar `GuardarEm` -- e essa ligacao continua em falta.
func adoptarParePersistido() {
	caminho := os.Getenv(variavelDoCaminhoDoPare)
	if caminho == "" {
		return
	}
	// Um estado ilegivel devolve erro e o pare fica como estava: nao ha' nada a fazer aqui senao
	// continuar -- mas nao se apaga o ficheiro nem se assume "desarmado".
	_ = pareDoServidor.CarregarDe(caminho)
}

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
