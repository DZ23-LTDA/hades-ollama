package server

// FASE 28.5 (continuacao): a rota do Signing Identity Center.
//
// Le o ficheiro de METADADOS de fora do repositorio e devolve o que a UI precisa. Nunca devolve
// valor privado: o tipo `IdentidadeDeAssinatura` nao tem campo para ele, e a invariante e'
// verificada por reflexao no teste do pacote.
//
// Todas as rotas sao LEITURA. Criar/importar/revogar identidade exige confirmacao humana no
// momento da accao e pertence a outra superficie -- *uma rota que assina por HTTP transformaria um
// pedido em acto, e um acto precisa de um rosto.*

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/multillm"
)

// signingIdentities responde `credentials.status`: o registo completo projetado para a UI.
//
// Devolve 200 mesmo quando o ficheiro nao existe: "nao configurado" e' um estado do produto, nao
// uma falha do servidor. *Um 404 aqui faria a UI mostrar um erro de rede onde o que falta e' uma
// accao do proprietario.*
func (a *agentAPI) signingIdentities(c *gin.Context) {
	caminho := multillm.CaminhoDoFicheiroDeIdentidades()
	registo, estado := multillm.DescreverEstadoDoRegisto(caminho)
	agora := time.Now().UTC()

	problemas := registo.Auditar(agora)
	projetadas := multillm.ProjetarIdentidades(registo, agora)
	resumo := registo.Resumir(agora)

	// Listas garantidamente nao nulas: um contrato que promete lista devolve lista.
	if problemas == nil {
		problemas = []multillm.ProblemaDeIdentidade{}
	}
	c.JSON(http.StatusOK, gin.H{
		"estado":        estado,
		"identidades":   projetadas,
		"problemas":     problemas,
		"resumo":        resumo,
		"verificado_em": agora.Format(time.RFC3339),
		// Lembrete explicito no payload: o consumidor nao deve ter de adivinhar.
		"aviso": "apenas metadados publicos; o valor privado vive no broker de credenciais e " +
			"nunca atravessa esta rota",
	})
}

// signingIdentitySummary responde so' o resumo, para o cartao do Mission Control.
//
// Rota separada de proposito: o cartao faz polling barato e nao precisa da lista completa.
// *Pedir a lista inteira para mostrar tres numeros e' pagar largura de banda por dados que se
// descartam.*
func (a *agentAPI) signingIdentitySummary(c *gin.Context) {
	caminho := multillm.CaminhoDoFicheiroDeIdentidades()
	registo, estado := multillm.DescreverEstadoDoRegisto(caminho)
	agora := time.Now().UTC()
	c.JSON(http.StatusOK, gin.H{
		"configurado": estado.Configurado,
		"carregado":   estado.Carregado,
		"total":       estado.Total,
		"mensagem":    estado.Mensagem,
		"resumo":      registo.Resumir(agora),
		"problemas":   len(registo.Auditar(agora)),
	})
}
