package server

// FASE 28.7 do adendo V3: `build.smoke` -- a rota que EXECUTA o artefacto.
//
// PORQUE ESTA ROTA E' DIFERENTE DAS OUTRAS
//
// `build/plan`, `artifact/checksum`, `artifact/verify` e `sign/plan` sao **GET** e so' leem. Esta
// rota **executa um binario**. Sem uma fronteira, seria um servico de execucao remota:
//
//	*Uma rota que executa o que lhe mandarem nao e' um TestProvider: e' um shell com JSON.*
//
// A FRONTEIRA E' UMA RAIZ, NAO UMA LISTA DE EXTENSOES
//
// O caminho tem de estar DENTRO de `HADES_ARTEFACTOS` -- uma variavel de ambiente que o operador
// define. Nao ha' caminho no pedido que escolha a raiz. Tres verificacoes, nesta ordem:
//
//  1. **normalizar** o caminho pedido (`filepath.Abs` + `filepath.Clean`);
//  2. **resolver symlinks** nos dois lados (`filepath.EvalSymlinks`) -- sem isto, um symlink dentro da
//     raiz apontaria para fora dela, e a verificacao de prefixo passaria;
//  3. **conter**: `filepath.Rel(raiz, alvo)` nao pode comecar por `..`.
//
//	*Uma verificacao de prefixo sem resolver symlinks e' uma verificacao que aprova o link, nao o alvo.*
//
// O QUE A ROTA NAO FAZ
//
// Nao instala, nao desinstala, nao muda o sistema. Executa o artefacto com os argumentos declarados,
// mede o exit code e a saida, e devolve o que observou. Nao ha' shell: o binario e' executado
// directamente, com `exec.Command`, e os argumentos sao passados como lista -- nao ha' string de shell
// para interpretar.
//
//	*Um argumento que passa por um shell e' um argumento que o shell pode reescrever.*

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// variavelDosArtefactos e' a raiz autorizada para executar binarios.
const variavelDosArtefactos = "HADES_ARTEFACTOS"

// timeoutMaximoDoSmoke limita o que o pedido pode pedir.
//
//	*Um timeout escolhido pelo cliente e' uma pausa que o servidor aceita por escrito.*
const timeoutMaximoDoSmoke = 5 * time.Minute

// maxArgumentosDeSmoke limita a lista de argumentos.
const maxArgumentosDeSmoke = 32

// ErrForaDaRaiz e' devolvido quando o caminho pedido esta' fora da raiz autorizada.
var ErrForaDaRaiz = errors.New("caminho fora da raiz autorizada de artefactos")

// pedidoDeSmoke e' o corpo de `build.smoke`.
type pedidoDeSmoke struct {
	Caminho        string   `json:"caminho"`
	Argumentos     []string `json:"argumentos"`
	SaidaEsperada  string   `json:"saida_esperada,omitempty"`
	CodigoEsperado *int     `json:"codigo_esperado,omitempty"`
	TimeoutMS      int      `json:"timeout_ms,omitempty"`
}

// respostaDeSmoke e' a resposta de `build.smoke`.
type respostaDeSmoke struct {
	Executou       bool   `json:"executou"`
	Aprovado       bool   `json:"aprovado"`
	CodigoDeSaida  int    `json:"codigo_de_saida"`
	DuracaoMS      int64  `json:"duracao_ms"`
	Saida          string `json:"saida"`
	SaidaTruncada  bool   `json:"saida_truncada"`
	Timeout        bool   `json:"timeout"`
	Explicacao     string `json:"explicacao"`
	SemSandbox     bool   `json:"sem_sandbox"`
	AvisoDeSandbox string `json:"aviso_de_sandbox"`
	RaizAutorizada bool   `json:"raiz_autorizada"`
}

// avisoDeSandboxDaRota diz, no corpo, que nao houve isolamento.
func avisoDeSandboxDaRota() string {
	return "este smoke executou o artefacto com os privilegios deste processo. NAO ha' sandbox, " +
		"contentor, VM, emulador ou simulador nesta maquina. A fronteira e' apenas a raiz " +
		"`" + variavelDosArtefactos + "`: o caminho tem de estar dentro dela, com symlinks resolvidos. " +
		"Executar codigo nao confiavel exige isolamento que esta rota nao oferece."
}

// conterNaRaiz verifica que `alvo` esta' dentro de `raiz`, com symlinks resolvidos.
//
//	*Uma verificacao de prefixo sem resolver symlinks e' uma verificacao que aprova o link, nao o alvo.*
func conterNaRaiz(raiz, alvo string) error {
	raizAbsoluta, err := filepath.Abs(raiz)
	if err != nil {
		return fmt.Errorf("%w: resolver a raiz %q: %v", ErrForaDaRaiz, raiz, err)
	}
	alvoAbsoluto, err := filepath.Abs(alvo)
	if err != nil {
		return fmt.Errorf("%w: resolver %q: %v", ErrForaDaRaiz, alvo, err)
	}
	// Resolver symlinks nos dois lados. Se a raiz nao existir, nao ha' como conter nada.
	raizReal, err := filepath.EvalSymlinks(raizAbsoluta)
	if err != nil {
		return fmt.Errorf("%w: a raiz autorizada %q nao existe: %v", ErrForaDaRaiz, raizAbsoluta, err)
	}
	alvoReal, err := filepath.EvalSymlinks(alvoAbsoluto)
	if err != nil {
		// O alvo pode nao existir; nesse caso contem-se o caminho NAO resolvido, que e' o melhor que
		// se pode fazer sem inventar. A execucao vai falhar depois, no `os.Stat` do provider.
		alvoReal = alvoAbsoluto
	}
	relativo, err := filepath.Rel(raizReal, alvoReal)
	if err != nil {
		return fmt.Errorf("%w: %q nao e' relativo a %q: %v", ErrForaDaRaiz, alvoReal, raizReal, err)
	}
	if relativo == ".." || strings.HasPrefix(relativo, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: %q esta' fora de %q", ErrForaDaRaiz, alvoReal, raizReal)
	}
	return nil
}

// executarSmoke responde a `build.smoke`.
func (a *agentAPI) executarSmoke(c *gin.Context) {
	raiz := strings.TrimSpace(os.Getenv(variavelDosArtefactos))
	if raiz == "" {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"erro": "a raiz de artefactos nao esta' configurada neste servidor; defina " +
				variavelDosArtefactos + " para o directorio dos artefactos. Esta rota EXECUTA " +
				"binarios e nao corre sem uma fronteira declarada",
		})
		return
	}

	var pedido pedidoDeSmoke
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "corpo invalido: esperado {\"caminho\": ..., \"argumentos\": [...], " +
				"\"saida_esperada\": ..., \"codigo_esperado\": ..., \"timeout_ms\": ...}",
		})
		return
	}

	// A fronteira. Um 403 aqui e' a diferenca entre "executei o que me mandaram" e "executei o que
	// estava autorizado".
	if err := conterNaRaiz(raiz, pedido.Caminho); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"erro": err.Error()})
		return
	}

	if len(pedido.Argumentos) > maxArgumentosDeSmoke {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"erro": fmt.Sprintf("%d argumentos; o maximo e' %d", len(pedido.Argumentos),
				maxArgumentosDeSmoke),
		})
		return
	}

	limite := time.Duration(pedido.TimeoutMS) * time.Millisecond
	if limite <= 0 || limite > timeoutMaximoDoSmoke {
		limite = agent.TimeoutPorOmissaoDeSmoke
		if pedido.TimeoutMS > 0 {
			limite = timeoutMaximoDoSmoke
		}
	}

	provider := agent.ProviderDeSmoke{}
	resultado, err := provider.ExecutarSmoke(c.Request.Context(), agent.PedidoDeSmoke{
		Caminho:        pedido.Caminho,
		Argumentos:     pedido.Argumentos,
		Timeout:        limite,
		SaidaEsperada:  pedido.SaidaEsperada,
		CodigoEsperado: pedido.CodigoEsperado,
	})
	if err != nil {
		// Recusas do provider sao 400: o pedido e' que estava mal, nao o servidor.
		codigo := http.StatusBadRequest
		if errors.Is(err, agent.ErrArtefactoInexistente) ||
			errors.Is(err, agent.ErrArtefactoNaoExecutavel) ||
			errors.Is(err, agent.ErrSmokeSemArgumentos) {
			codigo = http.StatusBadRequest
		}
		c.JSON(codigo, gin.H{"erro": err.Error()})
		return
	}

	c.JSON(http.StatusOK, respostaDeSmoke{
		Executou:       resultado.Executado,
		Aprovado:       resultado.Aprovado,
		CodigoDeSaida:  resultado.CodigoDeSaida,
		DuracaoMS:      resultado.DuracaoMS,
		Saida:          resultado.Saida,
		SaidaTruncada:  resultado.SaidaTruncada,
		Timeout:        resultado.Timeout,
		Explicacao:     resultado.Explicacao,
		SemSandbox:     true,
		AvisoDeSandbox: avisoDeSandboxDaRota(),
		RaizAutorizada: true,
	})
}

// garantirContexto mantem o import de `context` explicito para quem ler o ficheiro.
var _ = context.Background
