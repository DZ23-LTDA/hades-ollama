package server

// FASE 28.7: a rota `build.start`.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// O `agent.ProviderDeBuildGo` existe desde o E-1261, tem `Construir` e `ConstruirAlvos`, e esta'
// provado com testes. **Nenhuma rota o expunha.** O adendo da FASE 28.7 pede `build.start` e o
// proprio `agent_build_verify_route.go` nomeia o defeito:
//
//	*Uma comparacao que so' o programador consegue invocar nao e' uma funcionalidade: e' uma funcao.*
//
// O QUE ESTA ROTA **FAZ**
//
// Constroi um alvo Go, uma vez, e devolve o artefacto medido. E' a primeira rota do Release Hub que
// EXECUTA alguma coisa -- por isso as defesas nao estao aqui: estao no provider, que ja' as tinha.
//
// O QUE ESTA ROTA **NAO** FAZ
//
// Nao guarda registo do build. Nao ha' `build.status`, `build.logs`, `build.cancel` nem
// `build.artifacts` -- e nao se inventam por simetria com o adendo:
//
//	*Um ID que resolve para um ficheiro que ninguem registou e' um apelido, nao uma identidade.*
//
// O resultado e' devolvido UMA vez, na resposta. Quem o quer guardar, guarda. Um `build.status/:id`
// sem store seria uma rota que responde 404 a tudo o que ela propria prometeu.
//
// AS DEFESAS QUE JA' EXISTEM NO PROVIDER (medidas, nao prometidas)
//
//  1. `plano.Pronto()` -- um plano com blockers nao constroi. Um artefacto feito com ferramenta
//     ausente e' um artefacto que ninguem reproduz.
//  2. `cobreAlvo` -- so' windows/linux/macos. O catalogo tem mais alvos; este provider cobre tres.
//  3. raiz resolvida com `filepath.Abs`, verificada como directoria e com `go.mod`.
//  4. `TimeoutPorOmissaoDeBuild` = 10 min, e o chamador nao o pode esticar acima de um tecto.
//  5. `maxSaidaDeComando` = 4096 B -- a saida de uma build e' metadado, nao uma copia do log.
//  6. `DryRun` devolve o comando sem o executar.
//
// O QUE ESTA ROTA ACRESCENTA A ISSO
//
//  1. validacao do corpo antes de tocar no filesystem;
//  2. `dry_run` explicitamente aceite pelo corpo, para o painel poder mostrar o comando;
//  3. um aviso de fronteira no CORPO, porque a resposta contem um caminho de artefacto e "construido"
//     nao e' "assinado" nem "publicado".
//
//	*Um campo que precisa de documentacao externa para nao ser mal interpretado e' um campo mal
//	desenhado.*

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// maxDuracaoDeBuildPorPedido e' o tecto que o chamador pode pedir.
//
// O provider tem o seu proprio limite por omissao; este e' o maximo que o corpo pode pedir. Um
// `timeout_segundos` de 86400 seria uma build que fica depois de quem a pediu sair.
//
//	*Um timeout escolhido pelo cliente e' uma pausa que o servidor aceita por escrito.*
const maxDuracaoDeBuildPorPedido = 30 * time.Minute

// pedidoDeInicioDeBuild e' o corpo de `build.start`.
//
// A raiz vem no corpo a pedido de quem chama, mas o servidor NAO a inventa: o provider resolve-a
// para absoluto e verifica que e' uma directoria com `go.mod`.
type pedidoDeInicioDeBuild struct {
	// Raiz e' o directorio do projection. Obrigatoria.
	Raiz string `json:"raiz"`
	// Alvo e' o artefato pretendido (ex.: "windows"). Obrigatorio.
	Alvo string `json:"alvo"`
	// Destino e' onde o artefacto e' escrito. Vazio usa `Raiz/dist`.
	Destino string `json:"destino,omitempty"`
	// DryRun calcula e devolve o comando sem o executar.
	DryRun bool `json:"dry_run,omitempty"`
	// TimeoutSegundos limita a build. Zero usa o limite por omissao do provider.
	TimeoutSegundos int `json:"timeout_segundos,omitempty"`
}

// respostaDeInicioDeBuild e' a resposta de `build.start`.
type respostaDeInicioDeBuild struct {
	// Construiu diz, sem ambiguidade, se um comando de build chegou a correr.
	Construiu bool `json:"construiu"`
	// NadaAssinado repete a garantia num campo proprio.
	NadaAssinado bool `json:"nada_assinado"`
	// NadaPublicado repete a garantia num campo proprio.
	NadaPublicado bool `json:"nada_publicado"`
	// Alvo e' o alvo que correu, ecoado para o cliente nao ter de o inferir.
	Alvo string `json:"alvo"`
	// Plano e' o `CapabilityPlan` que autorizou (ou bloqueou) a build.
	Plano agent.CapabilityPlan `json:"plano"`
	// Resultado e' o artefacto medido. Ausente quando o plano bloqueou.
	Resultado agent.ResultadoDeBuild `json:"resultado"`
	// AvisoDeFronteira explica a fronteira em linguagem simples.
	AvisoDeFronteira string `json:"aviso_de_fronteira"`
}

// avisoDaFronteiraDoInicioDeBuild e' o texto que acompanha TODA resposta com sucesso.
func avisoDaFronteiraDoInicioDeBuild() string {
	return "esta rota e' `build.start`: constroi UM alvo Go com o `go build` e devolve o artefacto " +
		"medido. NAO assina, NAO verifica assinatura, NAO publica e NAO guarda registo do build -- " +
		"nao existe `build.status`, `build.logs`, `build.cancel` nem `build.artifacts`, e nao se " +
		"inventam sem um store. O resultado e' devolvido uma vez; o caminho do artefacto e' do " +
		"SERVIDOR. Construir nao e' assinar, e assinar nao e' publicar: cada etapa exige autorizacao " +
		"propria. A autorizacao vigente desta missao proibe publicacao externa."
}

// iniciarBuild atende `POST /build/start`.
func (a *agentAPI) iniciarBuild(c *gin.Context) {
	var pedido pedidoDeInicioDeBuild
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":    "corpo invalido",
			"detalhe": err.Error(),
		})
		return
	}

	raiz := strings.TrimSpace(pedido.Raiz)
	if raiz == "" {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "`raiz` obrigatoria: informe o caminho absoluto do projection a construir",
		})
		return
	}
	// Recusar antes de tocar no filesystem: um caminho relativo seria resolvido contra o cwd do
	// servidor, que nao e' o do cliente.
	if !filepath.IsAbs(raiz) {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "`raiz` nao e' um caminho absoluto: " + raiz,
		})
		return
	}

	alvo := strings.TrimSpace(pedido.Alvo)
	if alvo == "" {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro":  "`alvo` obrigatorio: informe um alvo do catalogo",
			"alvos": agent.ProviderDeBuildGo{}.Alvos(),
		})
		return
	}

	// O tecto do pedido nunca pode exceder o maximo da rota. Um timeout negativo e' recusado em vez
	// de convertido em zero -- zero significa "usa o limite por omissao", que e' outra coisa.
	if pedido.TimeoutSegundos < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":    "`timeout_segundos` negativo",
			"detalhe": "use 0 para o limite por omissao, ou um valor positivo",
		})
		return
	}
	timeout := time.Duration(pedido.TimeoutSegundos) * time.Second
	if timeout > maxDuracaoDeBuildPorPedido {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":    "`timeout_segundos` acima do tecto",
			"detalhe": "maximo " + maxDuracaoDeBuildPorPedido.String(),
		})
		return
	}

	// O PLANO VEM DO RESOLVER, NAO DO PEDIDO.
	//
	// O cliente escolhe o ALVO (o que quer); nao escolhe se as ferramentas existem (o que o servidor
	// mede). Deixar o cliente declarar o plano seria deixar o cliente declarar que a toolchain esta'
	// pronta.
	resolver := agent.NovoResolverDeToolchain()
	plano, err := resolver.Plano(c.Request.Context(), alvo)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"erro":    "alvo nao reconhecido pelo catalogo",
			"detalhe": err.Error(),
		})
		return
	}

	resultado, err := agent.ProviderDeBuildGo{}.Construir(
		c.Request.Context(),
		plano,
		agent.OpcoesDeBuild{
			Raiz:    raiz,
			Destino: strings.TrimSpace(pedido.Destino),
			DryRun:  pedido.DryRun,
			Timeout: timeout,
		},
	)
	if err != nil {
		// O PLANO INCOMPLETO E' 412, NAO 500: o servidor funcionou, o pre-requisito e' que falta.
		//
		//	*"Depende de conta externa" nao e' falha do servidor, e o codigo HTTP tem de o dizer.*
		if errors.Is(err, agent.ErrPlanoIncompleto) {
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"erro":               "plano incompleto: ha' blockers obrigatorios por resolver",
				"detalhe":            err.Error(),
				"plano":              plano,
				"blockers":           plano.Blockers,
				"aviso_de_fronteira": avisoDaFronteiraDoInicioDeBuild(),
			})
			return
		}
		if errors.Is(err, agent.ErrSemProviderParaAlvo) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"erro":    "provider nao cobre este alvo",
				"detalhe": err.Error(),
				"alvos":   agent.ProviderDeBuildGo{}.Alvos(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro":    "a build falhou",
			"detalhe": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, respostaDeInicioDeBuild{
		Construiu:        !pedido.DryRun,
		NadaAssinado:     true,
		NadaPublicado:    true,
		Alvo:             alvo,
		Plano:            plano,
		Resultado:        resultado,
		AvisoDeFronteira: avisoDaFronteiraDoInicioDeBuild(),
	})
}
