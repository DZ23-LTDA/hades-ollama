package server

// FASE 2 do adendo V3: a rota que ESCREVE o registo de estado de worktree.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1234 provou o registo e a E-1235 ligou-o a' leitura. Mas nenhuma rota o ESCREVIA: na
// pratica so' codigo interno suspendia ou arquivava um worktree. O utilizador via um estado que
// nao podia mudar.
//
//	*Um estado que so' o programador muda nao e' um estado do produto: e' um valor de fabrica.*
//
// PORQUE E' QUE ESTA ROTA NAO REMOVE
//
// `POST /worktrees/state` aceita apenas transicoes publicadas em `motivosDeMudancaDeEstado`.
// `removido` NAO esta' nessa lista, de proposito:
//
//	*Remover exige uma decisao sobre trabalho nao integrado, e essa decisao nao cabe num clique
//	que o painel nao sabe explicar.*
//
// A remocao tem o seu proprio caminho guardado (`RemoveGitWorktreeSeguro`,
// `RemoveGitWorktreeComPerda`) e nao e' exposta por HTTP, porque nao existe hoje uma forma de
// apresentar ao utilizador, num unico pedido, o diagnostico completo do que se perde.
//
// A VALIDACAO NAO E' LOCAL
//
// A rota nao decide que transicoes sao legais: pergunta ao `agent.Transitar`, que aplica a tabela
// fechada. *Uma segunda lista de transicoes na rota divergiria da primeira no primeiro dia em que
// alguem mudasse uma delas.*

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// motivoDaMudancaDeEstado e' uma transicao que a API aceita, com a accao que a produz.
type motivoDaMudancaDeEstado struct {
	// Para e' o estado de destino.
	Para agent.EstadoDoWorktree
	// Acao e' o verbo que o utilizador reconhece.
	Acao string
	// Descricao explica o efeito, porque o cliente nao deve ter de o inferir do nome do estado.
	Descricao string
}

// worktreeEstado e' o tipo de estado, nao uma copia da lista fechada.
type worktreeEstado = agent.EstadoDoWorktree

// motivosDeMudancaDeEstado e' a ALLOWLIST das transicoes expostas por HTTP.
//
//	*Uma transicao que nao existe obriga a quem a quer a perguntar por que a quer.*
//
// `removido` esta' ausente de proposito. Ver o cabecalho deste ficheiro.
var motivosDeMudancaDeEstado = []motivoDaMudancaDeEstado{
	{Para: agent.WorktreeSuspenso, Acao: "suspender", Descricao: "pausa o worktree sem apagar nada"},
	{Para: agent.WorktreeEmUso, Acao: "retomar", Descricao: "devolve o worktree ao uso"},
	{Para: agent.WorktreeArquivado, Acao: "arquivar", Descricao: "marca como terminado, preservado"},
}

// estadoDePedido e' o corpo de POST /worktrees/state.
type estadoDePedido struct {
	Repo   string `json:"repo"`
	Missao string `json:"missao"`
	// Para e' o estado de destino. Aceita o NOME ou a accao ("suspender").
	Para string `json:"para"`
}

// estadoDeResposta e' a resposta de POST /worktrees/state.
type estadoDeResposta struct {
	MissionID string `json:"mission_id"`
	// De e Para sao o estado antes e depois, ja' validados pela tabela fechada.
	De   string `json:"de"`
	Para string `json:"para"`
	// Acao e' o verbo que o cliente pediu.
	Acao string `json:"acao"`
	// EstadoDoRegisto diz se o estado anterior veio do registo persistido.
	EstadoDoRegisto bool `json:"estado_do_registo"`
	// RegistadoEm diz onde o registo foi gravado, para o cliente poder verificar.
	RegistadoEm string `json:"registado_em"`
	// Avisos explica o que pode surpreender.
	Avisos []string `json:"avisos"`
	// SomenteLeitura e' sempre falso aqui, mas o campo existe para o cliente distinguir esta
	// resposta das respostas de leitura sem mudar de contrato.
	SomenteLeitura bool `json:"somente_leitura"`
}

// transicaoAceite devolve o motivo correspondente a um pedido, se existir na allowlist.
func transicaoAceite(pedido string) (motivoDaMudancaDeEstado, bool) {
	limpo := strings.ToLower(strings.TrimSpace(pedido))
	if limpo == "" {
		return motivoDaMudancaDeEstado{}, false
	}
	for _, m := range motivosDeMudancaDeEstado {
		if string(m.Para) == limpo || m.Acao == limpo {
			return m, true
		}
	}
	return motivoDaMudancaDeEstado{}, false
}

// acoesPermitidas lista as accoes aceites, para a mensagem de erro ser accionavel.
func acoesPermitidas() []string {
	nomes := make([]string, 0, len(motivosDeMudancaDeEstado))
	for _, m := range motivosDeMudancaDeEstado {
		nomes = append(nomes, m.Acao)
	}
	return nomes
}

// worktreeMudarEstado trata POST /worktrees/state.
func (a *agentAPI) worktreeMudarEstado(c *gin.Context) {
	var pedido estadoDePedido
	if err := c.ShouldBindJSON(&pedido); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":    "corpo JSON invalido",
			"detalhe": err.Error(),
			"exemplo": map[string]string{"repo": "/caminho/do/repositorio", "missao": "minha-missao", "para": "suspender"},
		})
		return
	}
	if pedido.Repo == "" || pedido.Missao == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "campos 'repo' e 'missao' sao obrigatorios",
		})
		return
	}

	motivo, ok := transicaoAceite(pedido.Para)
	if !ok {
		// `removido` cai aqui. A mensagem NAO pode sugerir que existe outra forma de o pedir
		// por esta rota: nao existe, e inventar uma dica seria pior do que recusar.
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":       "transicao nao permitida por esta rota",
			"pedido":     pedido.Para,
			"permitidas": acoesPermitidas(),
			"nota": "remover um worktree nao e' exposto por HTTP: exige decisao sobre trabalho " +
				"nao integrado e um diagnostico completo que esta rota nao apresenta",
		})
		return
	}

	// Descobrir o worktree PELO INVENTOR -- nunca construir o caminho.
	lista, err := agent.InventariarWorktrees(c.Request.Context(), pedido.Repo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	var achado *agent.WorktreeDescoberto
	sufixo := string(filepath.Separator) + pedido.Missao
	for i := range lista {
		if strings.HasSuffix(lista[i].CaminhoResolvido, sufixo) {
			achado = &lista[i]
			break
		}
	}
	if achado == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"erro":   "missao nao encontrada no inventor deste repositorio",
			"missao": pedido.Missao,
			"dica":   "use GET /api/agent/v1/worktrees?repo=... para ver o que existe",
		})
		return
	}

	// Ler o registo persistido, se houver data root derivavel.
	var registo *agent.RegistoDeWorktrees
	registadoEm := ""
	if dataRoot, tem := agent.DataRootDoWorktree(achado.CaminhoResolvido); tem {
		carregado, errRegisto := agent.CarregarRegistoDeWorktrees(agent.CaminhoDoRegisto(dataRoot))
		if errRegisto != nil {
			// Um registo ilegivel NAO impede a mudanca, mas tambem nao e' ignorado: o pedido
			// prossegue e o aviso viaja na resposta. *Recusar a mudanca por causa de um ficheiro
			// estragado deixaria o utilizador sem forma de recuperar o estado.*
			c.JSON(http.StatusConflict, gin.H{
				"erro":    "o registo de estado existe mas nao pode ser lido; reparar antes de mudar",
				"detalhe": errRegisto.Error(),
			})
			return
		}
		registo = carregado
		registadoEm = agent.CaminhoDoRegisto(dataRoot)
	}

	// Construir a sessao com o estado que o registo conhece, e NAO com um valor inventado.
	estadoAtual := agent.WorktreePreparado
	if registo != nil {
		if estado, existe := registo.EstadoDe(pedido.Missao); existe {
			estadoAtual = estado
		}
	}
	sessao := &agent.GitWorktreeSession{
		MissionID:   pedido.Missao,
		RepoRoot:    pedido.Repo,
		WorktreeDir: achado.CaminhoResolvido,
		Status:      string(estadoAtual),
	}

	de, paraFinal, errTransicao := agent.Transitar(sessao, motivo.Para)
	if errTransicao != nil {
		// A recusa vem da tabela fechada, nao de uma copia local dela.
		c.JSON(http.StatusConflict, gin.H{
			"erro":       errTransicao.Error(),
			"de":         string(de),
			"pedido":     string(motivo.Para),
			"permitidos": proximosComoTexto(de),
		})
		return
	}

	resp := estadoDeResposta{
		MissionID:       pedido.Missao,
		De:              string(de),
		Para:            string(paraFinal),
		Acao:            motivo.Acao,
		EstadoDoRegisto: registo != nil,
		RegistadoEm:     registadoEm,
		Avisos:          []string{},
		SomenteLeitura:  false,
	}

	if registo != nil {
		if err := registo.Registar(pedido.Missao, paraFinal, achado.CaminhoResolvido, achado.Branch); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"erro":    "estado mudou em memoria mas o registo nao foi gravado",
				"detalhe": err.Error(),
				"de":      string(de),
				"para":    string(paraFinal),
			})
			return
		}
		if err := registo.Guardar(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"erro":    "estado mudou em memoria mas o registo nao foi gravado em disco",
				"detalhe": err.Error(),
				"de":      string(de),
				"para":    string(paraFinal),
			})
			return
		}
		resp.Avisos = append(resp.Avisos,
			"o estado foi persistido e sobrevive ao reinicio do processo")
	} else {
		// Sem data root nao ha' onde gravar. Dizer isto e' obrigatorio: um cliente que leia
		// `para` sem ler `avisos` pensaria que o estado ficou guardado.
		resp.Avisos = append(resp.Avisos,
			"o caminho do worktree nao segue a convencao .agent-worktrees: "+
				"a mudanca vale apenas para este pedido e NAO foi persistida")
	}

	c.JSON(http.StatusOK, resp)
}

// proximosComoTexto converte os destinos possiveis em texto, para a mensagem de recusa.
func proximosComoTexto(de worktreeEstado) []string {
	destinos := agent.TransicoesDe(de)
	nomes := make([]string, 0, len(destinos))
	for _, d := range destinos {
		nomes = append(nomes, string(d))
	}
	return nomes
}
