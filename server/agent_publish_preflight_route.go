package server

// FASE 28.7 do adendo V3: a ROTA do `publish.preflight`.
//
// O PROVIDER EXISTIA E NAO TINHA PORTA
//
// `internal/agent/publishing_provider.go` implementa o item 6 do adendo 28.1 desde a E-1269: seis
// canais, `PreflightDePublicacao`, `CondicaoDoPreflight`, `CanaisDePublicacao()` e nove testes. Mas
// nenhuma rota o chamava.
//
//	*Um provider que existe no catalogo e nunca correu e' uma promessa com nome de modulo.*
//
// O QUE ESTA ROTA E' -- E O QUE ELA NAO E'
//
// Ela e' `publish.preflight`: verifica se o canal existe, se o artefacto candidato existe e e'
// integro, e se a credencial necessaria esta' presente.
//
// Ela NAO e' `publish.submit`. O provider nao tem `Publicar()` -- nem para devolver "nao autorizado".
//
//	*Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
//
// A AUTORIZACAO VIGENTE PROIBE PUBLICAR
//
// "Esta fase nao autoriza publicar nada agora. A autorizacao vigente continua proibindo release, tag,
// loja, marketplace e publicacao externa." Por isso `publicado` e' sempre `false` no corpo, e para
// canal publico o provider marca `requer_aprovacao` e declara o canal indisponivel.
//
// OS CAMINHOS VEM DO AMBIENTE, NUNCA DO PEDIDO
//
//	*Uma rota que aceita qualquer caminho e' uma rota que le' qualquer caminho.*
//
// O cliente nao escolhe que ficheiro o servidor abre: `HADES_ARTEFACTO_A_PUBLICAR` e
// `HADES_CREDENCIAL_DE_PUBLICACAO` sao do servidor. O pedido so' escolhe o CANAL, e o canal e'
// validado contra `CanaisDePublicacao()`.

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

const (
	// variavelDoArtefactoAPublicar aponta o artefacto candidato. Separada de `HADES_ARTEFACTOS` (a
	// fronteira dos binarios), de `HADES_RAIZ_SBOM` (a fronteira do codigo), de
	// `HADES_RAIZ_PROJECTO` (a fronteira do projecto) e de `HADES_KIT_CREDENCIAIS` (o cofre).
	variavelDoArtefactoAPublicar = "HADES_ARTEFACTO_A_PUBLICAR"
	// variavelDaCredencialDePublicacao aponta a credencial do canal publico. O conteudo NUNCA e'
	// lido: o provider mede a existencia e o tamanho.
	variavelDaCredencialDePublicacao = "HADES_CREDENCIAL_DE_PUBLICACAO"
)

// respostaDoPreflightDePublicacao e' o corpo de sucesso.
type respostaDoPreflightDePublicacao struct {
	// SomenteLeitura diz, no corpo, que a rota nao publica.
	SomenteLeitura bool `json:"somente_leitura"`
	// NadaPublicado repete a garantia num campo proprio, para quem so' olha para este.
	NadaPublicado bool `json:"nada_publicado"`
	// Preflight e' o veredicto do provider.
	Preflight agent.PreflightDePublicacao `json:"preflight"`
	// AvisoDeFronteira explica a fronteira em linguagem simples.
	AvisoDeFronteira string `json:"aviso_de_fronteira"`
}

// avisoDaFronteiraDoPreflight e' o texto que acompanha TODA resposta.
func avisoDaFronteiraDoPreflight() string {
	return "esta rota e' `publish.preflight`: verifica se o canal, o artefacto e a credencial " +
		"existiriam para publicar. NAO publica, NAO submete, NAO promove e NAO inicia rollout: " +
		"`publicado` e' sempre false e este provider nao tem `submit`. Publicar exige autorizacao " +
		"humana especifica no momento da accao, e a autorizacao vigente desta missao proibe " +
		"publicacao externa. Os caminhos do artefacto e da credencial vem do ambiente do servidor " +
		"(`HADES_ARTEFACTO_A_PUBLICAR` e `HADES_CREDENCIAL_DE_PUBLICACAO`), NUNCA do pedido. " +
		"`LOCAL_ONLY` significa que o artefacto esta' no disco, NAO que foi publicado."
}

// canaisConhecidos devolve o catalogo para validacao.
func canaisConhecidos() map[agent.CanalDePublicacao]bool {
	return agent.CanaisDePublicacao()
}

// preflightDePublicacao atende `POST /publish/preflight/:canal`.
//
// O canal vem no caminho porque e' o que o cliente ESCOLHE. Os caminhos do artefacto e da credencial
// vem do ambiente porque e' o que o cliente NAO escolhe.
func (a *agentAPI) preflightDePublicacao(c *gin.Context) {
	bruto := strings.TrimSpace(c.Param("canal"))
	if bruto == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "canal obrigatorio: informe um canal do catalogo",
		})
		return
	}

	// O canal e' normalizado para maiusculas porque o catalogo usa `LOCAL_ONLY`, `GOOGLE_PLAY`,
	// etc. Recusamos antes de tocar no filesystem: um canal invalido nao deve provocar leitura.
	canal := agent.CanalDePublicacao(strings.ToUpper(bruto))
	if _, ok := canaisConhecidos()[canal]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":   "canal desconhecido: " + bruto,
			"canais": nomesDosCanais(),
		})
		return
	}

	artefacto := strings.TrimSpace(os.Getenv(variavelDoArtefactoAPublicar))
	if artefacto == "" {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "artefacto nao configurado: defina `" + variavelDoArtefactoAPublicar + "` com o " +
				"caminho absoluto do artefacto candidato; sem artefacto nao ha' o que publicar",
		})
		return
	}
	if !caminhoAbsoluto(artefacto) {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "`" + variavelDoArtefactoAPublicar + "` nao e' um caminho absoluto: " + artefacto,
		})
		return
	}

	credential := strings.TrimSpace(os.Getenv(variavelDaCredencialDePublicacao))
	if credential != "" && !caminhoAbsoluto(credential) {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "`" + variavelDaCredencialDePublicacao + "` nao e' um caminho absoluto",
		})
		return
	}

	resultado, err := agent.ProviderDePublicacao{}.Preflight(
		c.Request.Context(), canal, artefacto, credential,
	)
	if err != nil {
		// Canal desconhecido ja' foi recusado acima; um erro aqui significa outra coisa.
		c.JSON(http.StatusConflict, gin.H{
			"erro":  "falha no preflight: " + err.Error(),
			"canal": string(canal),
		})
		return
	}

	c.JSON(http.StatusOK, respostaDoPreflightDePublicacao{
		SomenteLeitura:   true,
		NadaPublicado:    true,
		Preflight:        resultado,
		AvisoDeFronteira: avisoDaFronteiraDoPreflight(),
	})
}

// nomesDosCanais devolve os canais conhecidos, ordenados, para a mensagem de erro.
func nomesDosCanais() []string {
	catalogo := canaisConhecidos()
	nomes := make([]string, 0, len(catalogo))
	for canal := range catalogo {
		nomes = append(nomes, string(canal))
	}
	// Ordenacao estavel: a mensagem de erro nao pode mudar de ordem entre pedidos.
	for i := 1; i < len(nomes); i++ {
		for j := i; j > 0 && nomes[j] < nomes[j-1]; j-- {
			nomes[j], nomes[j-1] = nomes[j-1], nomes[j]
		}
	}
	return nomes
}
