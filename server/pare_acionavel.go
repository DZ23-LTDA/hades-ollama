package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// variavelDaAutoridadeDoPare nomeia a variavel que diz QUEM pode acionar o pare.
//
// # PORQUE UMA VARIAVEL, E PORQUE O DEFAULT E' NINGUEM
//
// O MECANISMO de parar tudo e' codigo; a AUTORIDADE de o fazer e' uma decisao de
// quem opera a plataforma. Sem esta variavel, o servidor tem o travao mas nao tem
// quem o puxe -- e isso e' um estado explicito (403 com motivo), nunca um silencio
// nem um "qualquer um pode". Uma lista vazia NAO significa "todos": significa
// "ninguem", e o erro diz isso.
//
// Formato: identificadores de organizacao separados por virgula.
//
//	HADES_PARE_AUTORIDADE=org-a,org-b
const variavelDaAutoridadeDoPare = "HADES_PARE_AUTORIDADE"

// autoridadeDoPare devolve o conjunto de identidades autorizadas a acionar/rearmar.
func autoridadeDoPare() map[string]bool {
	bruto := os.Getenv(variavelDaAutoridadeDoPare)
	autorizadas := make(map[string]bool)
	for _, parte := range strings.Split(bruto, ",") {
		if identidade := strings.TrimSpace(parte); identidade != "" {
			autorizadas[identidade] = true
		}
	}
	return autorizadas
}

// gravarPareSeConfigurado persiste o estado do pare quando HADES_PARE_ESTADO existe.
//
// # O QUE ISTO FECHA
//
// `GuardarEm` existia e ninguem a chamava: o pare podia ser acionado e um reinicio
// desarmava-o em silencio. Agora toda mudanca de estado tenta gravar -- e uma falha
// de escrita e' DEVOLVIDA, nunca engolida: um acionamento que nao persiste e' um
// acionamento que nao sobrevive, e o chamador tem de saber isso.
func gravarPareSeConfigurado() error {
	caminho := os.Getenv(variavelDoCaminhoDoPare)
	if caminho == "" {
		return nil
	}
	return pareDoServidor.GuardarEm(caminho)
}

// pedidoDeAcionamento e' o corpo aceito por `POST /build/pare`.
//
// NAO TEM CAMPO DE IDENTIDADE, por decisao: quem aciona vem da SESSAO. Um campo
// `por_quem` no corpo seria um convite a forjar autoria, e nao existe.
type pedidoDeAcionamento struct {
	Motivo string `json:"motivo"`
}

// pedidoDeRearme e' o corpo aceito por `POST /build/pare/rearmar`.
type pedidoDeRearme struct {
	Justificacao string `json:"justificacao"`
}

// exigirAutoridadeDoPare aplica a guarda comum as duas rotas de comando.
//
// Devolve a identidade (organizacao) da sessao e `true` quando pode seguir; caso
// contrario ja' escreveu a recusa e devolve `false`.
func exigirAutoridadeDoPare(c *gin.Context) (string, bool) {
	organizacao, ok := requireOrganizacaoDaSessao(c)
	if !ok {
		return "", false
	}
	autorizadas := autoridadeDoPare()
	if len(autorizadas) == 0 {
		// 403 e nao 501: o mecanismo EXISTE; o que falta e' a decisao de quem o pode usar.
		c.JSON(http.StatusForbidden, gin.H{
			"erro": "o gatilho do pare existe, mas a autoridade nao esta' configurada " +
				"(" + variavelDaAutoridadeDoPare + "); acionar ou rearmar exige decisao humana",
		})
		return "", false
	}
	if !autorizadas[organizacao] {
		c.JSON(http.StatusForbidden, gin.H{"erro": "identidade sem autoridade para comandar o pare"})
		return "", false
	}
	return organizacao, true
}

// acionarOPare atende `POST /build/pare` e ACIONA o travao.
//
// # O QUE FALTAVA, DITO AQUI
//
// Ate' esta fatia o pare era **observavel mas nao acionavel**: o executor consultava,
// o estado mostrava, o arranque adoptava -- e ninguem o puxava. Um travao que ninguem
// consegue acionar nao protege ninguem.
//
// # A FRONTEIRA ENTRE MECANISMO E AUTORIDADE
//
// O mecanismo (acionar, bloquear, persistir, rearmar) e' codigo e esta' provado por
// teste. A AUTORIDADE de producao -- que organizacao, com que credencial real, com que
// registo de auditoria -- e' decisao humana e vive em `HADES_PARE_AUTORIDADE`.
func (a *agentAPI) acionarOPare(c *gin.Context) {
	organizacao, ok := exigirAutoridadeDoPare(c)
	if !ok {
		return
	}
	var pedido pedidoDeAcionamento
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "corpo invalido: esperado {\"motivo\": \"...\"}"})
		return
	}
	motivo := strings.TrimSpace(pedido.Motivo)
	if motivo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "acionar o pare exige um motivo: sem motivo nao ha' registo"})
		return
	}
	if err := pareDoServidor.Acionar(motivo, organizacao); err != nil {
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
		return
	}
	if err := gravarPareSeConfigurado(); err != nil {
		// O pare ESTA' acionado em memoria; o que falhou foi a durabilidade. Dizer 200
		// aqui seria prometer que ele sobrevive ao reinicio sem que isso seja verdade.
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro": "pare acionado em memoria, mas NAO foi possivel persistir: " + err.Error(),
			"pare": pareDoServidor.Estado(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pare": pareDoServidor.Estado()})
}

// rearmarOPare atende `POST /build/pare/rearmar` e DESARMA o travao, com justificacao.
//
// Rearmar sem justificacao seria apagar a evidencia do que parou o sistema. O motivo
// de rearme fica no estado, ao lado do motivo do acionamento.
func (a *agentAPI) rearmarOPare(c *gin.Context) {
	organizacao, ok := exigirAutoridadeDoPare(c)
	if !ok {
		return
	}
	var pedido pedidoDeRearme
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "corpo invalido: esperado {\"justificacao\": \"...\"}"})
		return
	}
	justificacao := strings.TrimSpace(pedido.Justificacao)
	if justificacao == "" {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "rearmar exige justificacao registada"})
		return
	}
	if err := pareDoServidor.Rearmar(organizacao, justificacao); err != nil {
		c.JSON(http.StatusConflict, gin.H{"erro": err.Error()})
		return
	}
	if err := gravarPareSeConfigurado(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro": "pare rearmado em memoria, mas NAO foi possivel persistir: " + err.Error(),
			"pare": pareDoServidor.Estado(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pare": pareDoServidor.Estado()})
}
