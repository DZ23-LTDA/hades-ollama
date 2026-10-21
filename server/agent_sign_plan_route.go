package server

// FASE 28.7 do adendo V3: `sign.plan`.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1263 implementou o `SigningProvider` e provou-o com 12 testes -- incluindo o certificado Apple
// REAL do kit. Mas **so' os testes o invocavam**:
//
//	*Uma comparacao que so' o programador consegue invocar nao e' uma funcionalidade: e' uma funcao.*
//
// `sign.plan` e' o nome que o adendo 28.7 da' a esta operacao. Ela **planeia**, nao assina.
//
// PORQUE NAO HA' `sign.start`
//
// O adendo lista `sign.start` e `sign.status`. Nao os implemento nesta fatia, e a razao e' a
// autorizacao vigente:
//
//	"ao terminar a implementacao, pare antes de merge, release ou publicacao"
//	"O DeepSeek NAO esta autorizado nesta missao a usar a chave privada (...) ou assinar manifesto"
//
// Uma rota `sign.start` que existisse so' para devolver `403 nao autorizado` seria:
//
//	*Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
//
// `sign.plan` e' util AGORA: diz se a identidade serve antes de alguem tentar assinar, e evita o erro
// opaco da politica do formato.
//
// O QUE A ROTA NAO FAZ, DITO NO CORPO
//
// Nao le' a chave privada. Nao assina. Nao escreve. Nao valida cadeia criptografica nem revogacao --
// e o `aviso_de_fronteira` diz isso, para que `aprovada` nao seja lido como "assinatura feita".

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// timeoutDoPlanoDeAssinatura limita a leitura do certificado.
const timeoutDoPlanoDeAssinatura = 30 * time.Second

// pedidoDePlanoDeAssinatura e' o corpo de `sign.plan`.
type pedidoDePlanoDeAssinatura struct {
	Tipo       string                       `json:"tipo"`
	Identidade agent.IdentidadeDeAssinatura `json:"identidade"`
}

// respostaDePlanoDeAssinatura e' a resposta de `sign.plan`.
type respostaDePlanoDeAssinatura struct {
	SomenteLeitura   bool                        `json:"somente_leitura"`
	NadaAssinado     bool                        `json:"nada_assinado"`
	Avaliacao        agent.AvaliacaoDaIdentidade `json:"avaliacao"`
	AlvosSuportados  []agent.TipoDeAssinatura    `json:"alvos_suportados"`
	AvisoDeFronteira string                      `json:"aviso_de_fronteira"`
}

// avisoDeFronteiraDoPlanoDeAssinatura diz, no corpo, o que "apta" significa e o que nao aconteceu.
func avisoDeFronteiraDoPlanoDeAssinatura() string {
	return "esta rota apenas AVALIA se a identidade serve para o tipo pedido; nao le' a chave " +
		"privada, nao assina e nao escreve. `apta: true` significa que os gates que este provider " +
		"sabe aplicar passaram -- nao significa que o artefacto foi assinado. A cadeia " +
		"criptografica e a revogacao NAO sao validadas aqui: so' o nome da CA emissora e' " +
		"conferido. Assinar continua a exigir aprovacao humana especifica no momento da accao."
}

// planearAssinatura responde a `sign.plan`.
func (a *agentAPI) planearAssinatura(c *gin.Context) {
	var pedido pedidoDePlanoDeAssinatura
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "corpo invalido: esperado {\"tipo\": \"AUTHENTICODE|<...>\", \"identidade\": " +
				"{\"caminho_do_certificado\": ..., \"rotulo\": ..., \"ambiente\": ...}}",
		})
		return
	}

	tipo := agent.TipoDeAssinatura(strings.ToUpper(strings.TrimSpace(pedido.Tipo)))
	if tipo == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "o campo `tipo` e' obrigatorio: sem ele nao se sabe qual EKU exigir",
		})
		return
	}
	if strings.TrimSpace(pedido.Identidade.CaminhoDoCertificado) == "" {
		// A recusa acontece AQUI e nao no provider, para o cliente receber a razao certa: uma
		// identidade sem certificado nao e' uma identidade, e' um rotulo.
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "a identidade nao aponta para nenhum `caminho_do_certificado`: sem certificado " +
				"nao ha' o que avaliar",
		})
		return
	}

	ctx, cancelar := context.WithTimeout(c.Request.Context(), timeoutDoPlanoDeAssinatura)
	defer cancelar()

	provider := agent.ProviderDeAssinaturaX509{}
	avaliacao, err := provider.Avaliar(ctx, tipo, pedido.Identidade)
	if err != nil {
		if errorsIsSemReferencia(err) {
			c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(codigoDeErroDeArtefacto(err), gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, respostaDePlanoDeAssinatura{
		SomenteLeitura:   true,
		NadaAssinado:     true,
		Avaliacao:        avaliacao,
		AlvosSuportados:  provider.Alvos(),
		AvisoDeFronteira: avisoDeFronteiraDoPlanoDeAssinatura(),
	})
}

// errorsIsSemReferencia diz se o erro e' o de referencia ausente.
func errorsIsSemReferencia(err error) bool {
	return err != nil && strings.Contains(err.Error(), agent.ErrSemReferencia.Error())
}
