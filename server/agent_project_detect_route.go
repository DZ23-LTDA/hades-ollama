package server

// FASE 28.7: a rota `project.detect` -- a porta do `ProjectDetector`.
//
// PORQUE ESTA ROTA EXISTE
//
// O `ProjectDetector` existe desde a E-1245, com 12 KB de codigo e o seu proprio teste. Mas nao tinha
// NENHUMA rota: so' o programador o podia invocar.
//
//	*Uma comparacao que so' o programador consegue invocar nao e' uma funcionalidade: e' uma funcao.*
//
// O QUE A ROTA NAO FAZ
//
// **Nao escolhe a raiz.** Vem de `HADES_RAIZ_PROJECTO` no servidor. O cliente nao manda caminho.
//
//	*Um cliente que escolhe a raiz dos binarios e' um cliente que decide o que corre.* (E-1273)
//
// **Nao escreve nada.** O detector so' le' nomes de ficheiros e o conteudo dos manifests para
// classificar a stack. Nao modifica, nao instala, nao constroi.
//
// **Nao devolve o conteudo dos manifestos.** Devolve os NOMES dos ficheiros encontrados e a stack
// classificada -- o suficiente para o `CapabilityResolver` decidir o que fazer, sem despejar o
// repositorio na resposta.

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// variavelDaRaizDoProjecto e' a fronteira de leitura do detector.
//
// Separada de `HADES_ARTEFACTOS` (binarios), `HADES_RAIZ_SBOM` (codigo para SBOM) e
// `HADES_KIT_CREDENCIAIS` (cofre) de proposito: cada rota le' uma raiz diferente, e nenhuma delas e'
// escolhida pelo cliente.
const variavelDaRaizDoProjecto = "HADES_RAIZ_PROJECTO"

// profundidadeMaximaDaDeteccao limita a varredura.
//
//	*Uma varredura sem limite e' uma pausa que o servidor nao pode recusar depois de comecar.* (E-1273)
const profundidadeMaximaDaDeteccao = 6

// respostaDaDeteccaoDeProjecto e' o corpo de `project.detect`.
type respostaDaDeteccaoDeProjecto struct {
	SomenteLeitura   bool                   `json:"somente_leitura"`
	Detectado        agent.ProjetoDetectado `json:"detectado"`
	AvisoDeFronteira string                 `json:"aviso_de_fronteira"`
}

// avisoDeFronteiraDaDeteccao diz, no corpo, o que a rota nao faz.
func avisoDeFronteiraDaDeteccao() string {
	return "esta rota apenas LE' nomes de ficheiros e o conteudo dos manifests para classificar a " +
		"stack e listar alvos; nao escreve, nao instala, nao constroi e nao assina. A raiz vem da " +
		"variavel de ambiente `" + variavelDaRaizDoProjecto + "`, nunca do pedido. Os alvos listados " +
		"sao uma SUGESTAO derivada dos ficheiros presentes -- nao significam que a toolchain esteja " +
		"instalada: consulte `toolchain.doctor` antes de planear um build."
}

// detectarProjecto responde a `project.detect`.
func (a *agentAPI) detectarProjecto(c *gin.Context) {
	raiz := strings.TrimSpace(os.Getenv(variavelDaRaizDoProjecto))
	if raiz == "" {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "a raiz do projecto nao esta' configurada: defina `" + variavelDaRaizDoProjecto +
				"` com o caminho absoluto do projecto a detectar",
		})
		return
	}

	// A raiz tem de ser absoluta. Um caminho relativo dependeria do CWD do servidor, e o cliente nao
	// pode influenciar isso.
	if !filepath.IsAbs(raiz) {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "`" + variavelDaRaizDoProjecto + "` tem de ser um caminho ABSOLUTO, nao relativo",
		})
		return
	}

	if _, err := os.Stat(raiz); err != nil {
		// Nao revelamos o erro do sistema (pode conter detalhes do host); dizemos o que fazer.
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "a raiz configurada em `" + variavelDaRaizDoProjecto + "` nao existe ou nao esta' " +
				"acessivel ao processo do servidor",
		})
		return
	}

	info, err := os.Stat(raiz)
	if err == nil && !info.IsDir() {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "a raiz configurada em `" + variavelDaRaizDoProjecto + "` e' um FICHEIRO, nao um " +
				"directorio",
		})
		return
	}

	detector := agent.ProjectDetector{
		Raiz:            raiz,
		MaxProfundidade: profundidadeMaximaDaDeteccao,
	}
	detectado, err := detector.Detectar()
	if err != nil {
		// A causa REAL e' dita, nao substituida por "erro interno".
		c.JSON(http.StatusConflict, gin.H{"erro": "falha a detectar o projecto: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, respostaDaDeteccaoDeProjecto{
		SomenteLeitura:   true,
		Detectado:        detectado,
		AvisoDeFronteira: avisoDeFronteiraDaDeteccao(),
	})
}
