package server

// FASE 28.9 passo 7: a rota `build.verify`.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1261 implementou `agent.VerificarConstrucao` e provou-a com 5 testes -- incluindo o artefacto
// real da E-1251. Mas **so' os testes a invocavam**. E' o mesmo defeito que a propria E-1261 corrigiu
// para o `ConstruirAlvos`, e o adendo nomeia-o:
//
//	*Uma comparacao que so' o programador consegue invocar nao e' uma funcionalidade: e' uma funcao.*
//
// PORQUE E' `POST` E NAO `GET`
//
// A lista de artefactos de um build e os seus SHA-256 nao cabem num URL sem limites arbitrarios: um
// build de 7 alvos seriam 7 caminhos absolutos e 7 checksums. O corpo leva a lista.
//
//	*Uma rota que se le' mas exige POST obriga quem a usa a saber mais do que precisa.*
//
// O QUE A ROTA **NAO** FAZ
//
// Nao constroi. Nao assina. Nao guarda o resultado. Nao procura o build por ID -- **nao existe
// registo de builds**, e nao se inventa por simetria com o adendo:
//
//	*Um ID que resolve para um ficheiro que ninguem registou e' um apelido, nao uma identidade.*
//
// Quem chama JA' TEM o resultado do build (foi ele que o pediu) e reenvia-o. A rota verifica-o.

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// maxArtefactosPorVerificacao limita o lote.
//
// *Uma varredura de arvore disparada por HTTP sem limite e' uma pausa que o cliente pode pedir ao
// servidor e que o servidor nao pode recusar depois de comecar.*
const maxArtefactosPorVerificacao = 64

// pedidoDeVerificacaoDeBuild e' o corpo de `build.verify`.
type pedidoDeVerificacaoDeBuild struct {
	// Artefatos sao os artefactos que o build produziu, com o SHA-256 medido nesse momento.
	Artefatos []agent.Artefato `json:"artefatos"`
}

// respostaDeVerificacaoDeBuild e' a resposta de `build.verify`.
type respostaDeVerificacaoDeBuild struct {
	SomenteLeitura   bool                          `json:"somente_leitura"`
	NadaAssinado     bool                          `json:"nada_assinado"`
	Verificacao      agent.VerificacaoDaConstrucao `json:"verificacao"`
	AvisoDeFronteira string                        `json:"aviso_de_fronteira"`
}

// avisoDeFronteiraDaVerificacaoDeBuild diz, no corpo, o que a rota nao faz e o que "aprovada"
// significa.
func avisoDeFronteiraDaVerificacaoDeBuild() string {
	return "esta rota apenas COMPARA cada artefacto com o SHA-256 que o cliente enviou; nao " +
		"constroi, nao assina, nao instala e nao procura o build por ID. `aprovada` significa que " +
		"todos os ficheiros continuam iguais ao que o build mediu -- nao significa que exista " +
		"assinatura: leia `nao_verificadas` para saber quantos ficaram por inspeccionar."
}

// verificarBuild responde a `build.verify`.
func (a *agentAPI) verificarBuild(c *gin.Context) {
	var pedido pedidoDeVerificacaoDeBuild
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "corpo invalido: esperado {\"artefatos\": [{\"caminho\": ..., \"sha256\": ...}]}",
		})
		return
	}

	if len(pedido.Artefatos) == 0 {
		// Um build que nao produziu nada NAO e' um build aprovado. O servico devolve o mesmo erro
		// que a funcao; a rota nao o transforma num 200 com lista vazia.
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "o corpo nao declara artefactos: nao ha' o que verificar",
		})
		return
	}
	if len(pedido.Artefatos) > maxArtefactosPorVerificacao {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"erro": "demasiados artefactos num so' pedido; o limite e' " +
				"64 e um build com mais alvos deve ser verificado em lotes",
		})
		return
	}

	// Recusar entradas incompletas ANTES de tocar no disco: um caminho vazio ou um checksum vazio
	// nao e' um artefacto, e' um pedido mal formado.
	for i, artefato := range pedido.Artefatos {
		if artefato.Caminho == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"erro": "artefato sem `caminho`: um artefacto sem caminho nao e' verificavel",
			})
			return
		}
		if artefato.SHA256 == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"erro": "artefato sem `sha256`: sem referencia isto seria uma medicao, nao uma " +
					"verificacao",
			})
			return
		}
		_ = i
	}

	ctx, cancelar := context.WithTimeout(c.Request.Context(), timeoutDaVerificacao)
	defer cancelar()

	// Sem adaptador de assinatura nesta rota: `NAO_VERIFICADA` e' o estado honesto, e o corpo da
	// resposta diz quantos ficaram por inspeccionar.
	verificacao, err := agent.VerificarConstrucao(ctx, agent.ResultadoDeBuild{
		Artefatos: pedido.Artefatos,
	}, agent.VerificadorDeArtefacto{})
	if err != nil {
		if errors.Is(err, agent.ErrSemArtefactosParaVerificar) {
			// Defesa em profundidade: o `len == 0` acima ja' cobre isto, mas a funcao tambem o
			// recusa e a rota nao deve divergir dela.
			c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
			return
		}
		c.JSON(codigoDeErroDeArtefacto(err), gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, respostaDeVerificacaoDeBuild{
		SomenteLeitura:   true,
		NadaAssinado:     true,
		Verificacao:      verificacao,
		AvisoDeFronteira: avisoDeFronteiraDaVerificacaoDeBuild(),
	})
}
