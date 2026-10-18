package server

// FASE 28.7 do adendo V3: `publish.rollback`.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1268 implementou o `RollbackProvider` e provou-o com 7 testes -- **mas so' os testes o
// invocavam**:
//
//	*Uma comparacao que so' o programador consegue invocar nao e' uma funcionalidade: e' uma funcao.*
//
// `publish.rollback` e' o nome que o adendo 28.7 da'.
//
// O DIRECTORIO VEM DO AMBIENTE, NAO DO PEDIDO
//
// A rota le' `HADES_ARTEFACTOS` -- a MESMA variavel que `build.smoke` usa. Nao aceita um
// `directorio` no corpo.
//
//	*Duas rotas que leem o mesmo cofre nao podem ter duas fronteiras diferentes.*
//
// Sem a variavel, a rota recusa com **412** e diz qual definir: *nao ha' plano de rollback sem uma
// raiz declarada*.
//
// E NAO EXECUTA O ROLLBACK
//
// `Planear` monta o plano e mede o SHA-256 de cada candidato. **Nao restaura.** Restaurar um binario
// de producao e' a accao que o adendo 28.5 poe atras de *"confirmacao humana no momento da accao"*:
//
//	*Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
//
// `executado` e' sempre `false`, e o `aviso` garante por extenso que nenhum ficheiro foi movido.

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// timeoutDoPlanoDeRollback limita a medicao dos candidatos.
const timeoutDoPlanoDeRollback = 60 * time.Second

// pedidoDeRollback e' o corpo de `publish.rollback`.
//
// **Nao tem `directorio`.** O directorio vem do ambiente.
type pedidoDeRollback struct {
	Tipo agent.TipoDeRollback `json:"tipo"`
}

// respostaDeRollback e' a resposta de `publish.rollback`.
type respostaDeRollback struct {
	// Executado e' sempre `false`: o provider planeia, nao restaura.
	Executado        bool                  `json:"executado"`
	Suportado        bool                  `json:"suportado"`
	Tipo             agent.TipoDeRollback  `json:"tipo"`
	Alvo             *agent.VersaoAnterior `json:"alvo,omitempty"`
	TotalCandidatos  int                   `json:"total_de_candidatos"`
	Integro          bool                  `json:"integro"`
	RequerAprovacao  string                `json:"requer_aprovacao,omitempty"`
	Explicacao       string                `json:"explicacao"`
	Aviso            string                `json:"aviso"`
	AvisoDeFronteira string                `json:"aviso_de_fronteira"`
}

// avisoDeFronteiraDoRollback diz, no corpo, que nada foi restaurado.
func avisoDeFronteiraDoRollback() string {
	return "esta rota apenas PLANEIA o rollback: mede o SHA-256 dos candidatos e propoe o mais " +
		"recente. `executado` e' sempre false -- nenhum ficheiro foi movido, copiado ou apagado. " +
		"Restaurar um artefacto de producao exige aprovacao humana especifica no momento da accao. " +
		"O directorio vem da variavel de ambiente `" + variavelDosArtefactos + "`, nunca do pedido."
}

// planearRollback responde a `publish.rollback`.
func (a *agentAPI) planearRollback(c *gin.Context) {
	raiz := strings.TrimSpace(os.Getenv(variavelDosArtefactos))
	if raiz == "" {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "a raiz de artefactos nao esta' configurada neste servidor; defina " +
				variavelDosArtefactos + " para o directorio dos artefactos",
		})
		return
	}

	var pedido pedidoDeRollback
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "corpo invalido: esperado {\"tipo\": \"WINDOWS_INSTALLER\"}",
		})
		return
	}
	if pedido.Tipo == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "o campo `tipo` e' obrigatorio: o canal de rollback nao tem valor por omissao",
		})
		return
	}

	contexto, cancelar := context.WithTimeout(c.Request.Context(), timeoutDoPlanoDeRollback)
	defer cancelar()

	provider := agent.ProviderDeRollback{}
	plano, err := provider.Planear(contexto, pedido.Tipo, raiz)
	if err != nil {
		// Canal sem credencial/host: o plano diz por que, e o erro e' 409 (nao 500): o servidor
		// esta' bem, o canal e' que nao da'.
		//
		//	*Um 500 aqui confundiria "o servidor falhou" com "o canal nao existe".*
		c.JSON(http.StatusConflict, gin.H{
			"erro":               err.Error(),
			"suportado":          plano.Suportado,
			"executado":          false,
			"explicacao":         plano.Explicacao,
			"aviso_de_fronteira": avisoDeFronteiraDoRollback(),
		})
		return
	}

	c.JSON(http.StatusOK, respostaDeRollback{
		Executado:        false,
		Suportado:        plano.Suportado,
		Tipo:             plano.Tipo,
		Alvo:             plano.Alvo,
		TotalCandidatos:  plano.TotalDeCandidatos,
		Integro:          plano.Integro,
		RequerAprovacao:  plano.RequerAprovacao,
		Explicacao:       plano.Explicacao,
		Aviso:            plano.Aviso,
		AvisoDeFronteira: avisoDeFronteiraDoRollback(),
	})
}
