package server

// FASE 28.6/28.7 do adendo V3: a rota que expoe o PLANO DE RELEASE do projection.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1245 criou o `ProjectDetector` e a E-1246 o `PlanoParaProjeto` que o consulta. Mas **nenhuma
// rota nem nenhum painel os chamava**: o sexto caso, neste trabalho, do mesmo defeito.
//
//	*Um plano que so' o programador consegue pedir e' um plano que o produto nao tem.*
//
// O QUE ESTA ROTA E'
//
// `GET /api/agent/v1/build/plan` responde a pergunta que o Mission Control precisa de fazer antes
// de prometer um build:
//
//	"este projection pode produzir alguma coisa nesta maquina, e se nao, o que falta?"
//
// E' de LEITURA: detecta o projection e sonda ferramentas. **Nao instala, nao constroi, nao altera.**
// A resposta declara isso explicitamente em `somente_leitura`.
//
// PORQUE E' GET E NAO POST
//
// Nao tem corpo: o sujeito e' um caminho no servidor. Um GET deixa o browser, o `curl` e um agente
// poderem faze-lo sem inventar um corpo vazio.
//
//	*Uma rota que se le' mas exige POST obriga quem a usa a saber mais do que precisa.*
//
// O QUE A ROTA NAO ACEITA, E PORQUE ISSO E' DELIBERADO
//
// Nao aceita um campo `alvo`. Se aceitasse, o plano voltaria a nascer de um alvo escolhido por quem
// chamou -- que e' exactamente o defeito que a E-1245/E-1246 fecharam.
//
//	*Um parametro que reintroduz o defeito que a fatia anterior corrigiu nao e' flexibilidade: e'
//	uma porta dos fundos.*

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// maxProfundidadeDaDeteccao limita a busca de manifests na rota.
//
//	*Uma varredura de arvore disparada por HTTP sem limite e' uma pausa que o cliente pode pedir ao
//	servidor e que o servidor nao pode recusar depois de comecar.*
const maxProfundidadeDaDeteccao = 4

// planoDeRelease responde com o plano de build do projection indicado.
//
// Query: `repo` (obrigatorio) -- caminho do projection no servidor.
// Query: `profundidade` (opcional, 1..maxProfundidadeDaDeteccao).
func (a *agentAPI) planoDeRelease(c *gin.Context) {
	repo := strings.TrimSpace(c.Query("repo"))
	if repo == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":    "parametro `repo` e' obrigatorio",
			"detalhe": "indique o caminho do projection a analisar (ex.: ?repo=/caminho/do/projection)",
		})
		return
	}
	// Caminho absoluto: um caminho relativo dependeria do directorio de trabalho do SERVIDOR, que
	// nao e' o do cliente. *Dois relativos diferentes com o mesmo texto sao dois caminhos
	// diferentes.*
	caminho, err := filepath.Abs(repo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":    "caminho `repo` invalido",
			"detalhe": err.Error(),
		})
		return
	}
	info, err := os.Stat(caminho)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"erro":    "projection nao encontrado",
			"detalhe": "`repo` nao existe ou nao e' acessivel pelo servidor",
		})
		return
	}
	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":    "`repo` nao e' um directorio",
			"detalhe": "o plano de build analisa um projection, nao um ficheiro",
		})
		return
	}

	profundidade := maxProfundidadeDaDeteccao
	if bruto := strings.TrimSpace(c.Query("profundidade")); bruto != "" {
		n, ok := parseLimite(bruto)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"erro":    "`profundidade` invalida",
				"detalhe": "esperado um inteiro entre 1 e 4",
			})
			return
		}
		profundidade = n
	}

	resolver := agent.NovoResolverDeToolchain()
	resolver.ProfundidadeDeDeteccao = profundidade
	plano, err := resolver.PlanoParaProjeto(c.Request.Context(), caminho)
	if err != nil {
		// `ErrProjectoNaoDetectado` e' 422: o caminho existe, mas nao e' um projection. Nao e' 404
		// (o recurso existe) nem 500 (nada falhou no servidor).
		//
		//	*Um erro que diz "nao sei o que isto e'" nao e' o mesmo que "isto nao existe", e o
		//	codigo HTTP tem de os distinguir para o cliente poder agir.*
		if errorsIsProjectoNaoDetectado(err) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"erro":    "projection nao reconhecido",
				"detalhe": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"erro":    "nao foi possivel construir o plano",
			"detalhe": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"somente_leitura":    true,
		"nada_instalado":     true,
		"nada_construido":    true,
		"repo":               caminho,
		"profundidade":       profundidade,
		"plano":              plano,
		"blockers":           plano.ResumoDeBlockers(),
		"motivos":            plano.MotivosLegiveis(),
		"aviso_de_fronteira": avisoDeFronteiraDoPlano(plano),
	})
}

// avisoDeFronteiraDoPlano explica o que a rota NAO fez.
//
// Existe porque a resposta contem "pronto" e "bloqueado", e um cliente apressado pode ler "pronto"
// como "pode construir agora". O aviso tira a ambiguidade no corpo, nao na documentacao.
//
//	*Um campo que precisa de documentacao externa para nao ser mal interpretado e' um campo mal
//	desenhado.*
func avisoDeFronteiraDoPlano(plano agent.PlanoDoProjeto) string {
	if plano.SemAlvos {
		return "o projection nao declara alvos que o catalogo do HADES conheca; nada foi construido"
	}
	return "esta rota apenas DETECTA o projection e SONDA ferramentas; nao instala nada, nao " +
		"constroi nada e nao assina nada"
}

// errorsIsProjectoNaoDetectado diz se o erro e' "isto nao e' um projection reconhecivel".
func errorsIsProjectoNaoDetectado(err error) bool {
	return err != nil && strings.Contains(err.Error(), agent.ErrProjectoNaoDetectado.Error())
}

// parseLimite converte e valida um limite entre 1 e maxProfundidadeDaDeteccao.
func parseLimite(bruto string) (int, bool) {
	n := 0
	for _, r := range bruto {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
		if n > maxProfundidadeDaDeteccao {
			return 0, false
		}
	}
	if n < 1 {
		return 0, false
	}
	return n, true
}
