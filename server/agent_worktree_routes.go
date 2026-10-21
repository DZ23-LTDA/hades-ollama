package server

// FASE 2 do adendo V3: superficie canonica de worktrees.
//
// O QUE ESTAS ROTAS EXPONHEM
//
//	GET /api/agent/v1/worktrees            -- inventor do repositorio indicado
//	GET /api/agent/v1/worktrees/diagnose   -- estado de ciclo de vida por missao
//
// O motor esta' em `internal/agent/worktree_state.go` e `worktree_ops.go`; aqui so' se traduz o
// resultado para JSON. Nao ha' segunda fonte de verdade.
//
// PORQUE ESTA ROTA EXISTE
//
// As E-1226 a E-1231 entregaram a protecao de trabalho nao integrado, a maquina de estados e a
// descoberta de worktrees externos -- **e nada disso era visivel**. Um subsistema que so' existe
// em teste e' uma promessa; a superficie e' o que o torna utilizavel.
//
//	*Um mecanismo que ninguem consegue ver e' um mecanismo que ninguem consegue usar.*
//
// AS TRES DECISOES QUE ESTAS ROTAS TORNAM VISIVEIS
//
//  1. **Ver antes de apagar.** O inventor mostra TAMBEM os worktrees que o HADES nao criou:
//     *um worktree externo que nao se ve' e' um worktree que se apaga por engano.*
//
//  2. **O estado e' fechado.** Cada sessao sai com um dos 9 estados declarados e com a lista do
//     que pode acontecer a seguir -- *um estado que ninguem pode nomear invalido nao e' um estado:
//     e' um campo de texto.*
//
//  3. **Diagnostico antes da accao.** `diagnose` compara o estado logico com o disco e diz quando
//     discordam. *Um estado que mente e' pior do que nenhum.*
//
// O QUE ESTAS ROTAS NAO FAZEM
//
// Nao removem, nao suspendem, nao criam. Sao **so' leitura**. Remover exige uma decisao sobre
// trabalho nao integrado, e essa decisao tem de ser explicita -- nao um efeito lateral de um GET.

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// worktreeDescobertoResposta e' uma linha do inventor.
type worktreeDescobertoResposta struct {
	CaminhoResolvido string `json:"caminho_resolvido"`
	Branch           string `json:"branch,omitempty"`
	Commit           string `json:"commit,omitempty"`
	Detached         bool   `json:"detached"`
	CriadoPeloHades  bool   `json:"criado_pelo_hades"`
	Principal        bool   `json:"principal"`
	Bloqueado        bool   `json:"bloqueado"`
}

// worktreeInventarioResposta e' a resposta de GET /worktrees.
type worktreeInventarioResposta struct {
	Repo    string                       `json:"repo"`
	Resumo  string                       `json:"resumo"`
	Total   int                          `json:"total"`
	DoHades int                          `json:"do_hades"`
	Externo int                          `json:"externo"`
	Lista   []worktreeDescobertoResposta `json:"lista"`
	// Avisos explica o que pode surpreender quem le'.
	Avisos []string `json:"avisos"`
	// SomenteLeitura impede que um cliente trate isto como accao.
	SomenteLeitura bool      `json:"somente_leitura"`
	MedidoEm       time.Time `json:"medido_em"`
}

// worktreeDiagnosticoResposta e' uma linha de GET /worktrees/diagnose.
type worktreeDiagnosticoResposta struct {
	MissionID string `json:"mission_id"`
	Estado    string `json:"estado"`
	// EstadoValido diz se o estado pertence a' lista fechada (apos normalizacao).
	EstadoValido bool     `json:"estado_valido"`
	Traduzido    bool     `json:"estado_traduzido"`
	Proximos     []string `json:"proximos"`
	// AlvoSeguro diz se o caminho pode ser removido sem risco.
	AlvoSeguro bool `json:"alvo_seguro"`
	// TrabalhoNaoIntegrado conta commits e ficheiros que ninguem integrou.
	CommitsNaoIntegrados    int    `json:"commits_nao_integrados"`
	FicheirosModificados    int    `json:"ficheiros_modificados"`
	TemTrabalhoNaoIntegrado bool   `json:"tem_trabalho_nao_integrado"`
	Detalhe                 string `json:"detalhe,omitempty"`
	// Inconsistente explica quando o estado logico e o disco discordam.
	Inconsistente string `json:"inconsistente,omitempty"`
	// EstadoObservado distingue o que o DISCO prova do que a sessao em memoria sabia.
	EstadoObservado string `json:"estado_observado"`
	PodeRemover     bool   `json:"pode_remover"`
	MotivoRemover   string `json:"motivo_remover,omitempty"`
	// EstadoDoRegisto diz se o estado veio do registo PERSISTIDO em disco.
	//
	// *Um estado que morre com o processo nao e' um estado: e' uma variavel.*
	EstadoDoRegisto bool `json:"estado_do_registo"`
	// Registo diz o que se fez com o registo: `consultado`, `ausente` ou `ilegivel: ...`.
	//
	// Distinguir estes tres casos e' o que impede um registo ilegivel de passar por ausente --
	// *um registo corrompido lido como vazio apagaria o estado de todos os worktrees com um
	// encolher de ombros.*
	Registo string `json:"registo"`
}

func (a *agentAPI) worktreeInventario(c *gin.Context) {
	repo := c.Query("repo")
	if repo == "" {
		// Sem `repo` explicito nao se adivinha: *inventariar o checkout errado e' pior do que
		// recusar.*
		c.JSON(http.StatusBadRequest, gin.H{
			"erro":    "parametro 'repo' e' obrigatorio (caminho absoluto da raiz do repositorio)",
			"exemplo": "/api/agent/v1/worktrees?repo=/caminho/para/repositorio",
		})
		return
	}

	lista, err := agent.InventariarWorktrees(c.Request.Context(), repo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": err.Error(),
		})
		return
	}

	resp := worktreeInventarioResposta{
		Repo:           repo,
		Resumo:         agent.ResumoDoInventario(lista),
		Total:          len(lista),
		Lista:          make([]worktreeDescobertoResposta, 0, len(lista)),
		SomenteLeitura: true,
		MedidoEm:       time.Now().UTC(),
	}
	for _, w := range lista {
		if w.CriadoPeloHades {
			resp.DoHades++
		} else {
			resp.Externo++
		}
		resp.Lista = append(resp.Lista, worktreeDescobertoResposta{
			CaminhoResolvido: w.CaminhoResolvido,
			Branch:           w.Branch,
			Commit:           w.Commit,
			Detached:         w.Detached,
			CriadoPeloHades:  w.CriadoPeloHades,
			Principal:        w.Principal,
			Bloqueado:        w.Bloqueado,
		})
	}
	if resp.Externo > 0 {
		resp.Avisos = append(resp.Avisos,
			"existem worktrees que o HADES nao criou; remover um deles apaga trabalho que nao e' dele")
	}
	if resp.DoHades > 0 {
		resp.Avisos = append(resp.Avisos,
			"worktrees do HADES sao preservados por DiagnosticarWorktree antes de qualquer remocao")
	}
	if resp.Avisos == nil {
		resp.Avisos = []string{}
	}
	c.JSON(http.StatusOK, resp)
}

func (a *agentAPI) worktreeDiagnostico(c *gin.Context) {
	repo := c.Query("repo")
	mission := c.Query("missao")
	if repo == "" || mission == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"erro": "parameters 'repo' e 'missao' sao obrigatorios",
		})
		return
	}

	// Descobrir o worktree PELO INVENTOR, nunca por um caminho construido.
	//
	// A tentacao era `filepath.Join(dataRoot, ".agent-worktrees", mission)`. Mas o `dataRoot` nao
	// e' conhecido nesta rota, e construir o caminho repetiria exactamente o erro da E-1231:
	//
	//	*um teste que compara caminhos CONSTRUIDOS com caminhos REPORTADOS pelo sistema testa a
	//	string, nao o alvo.*
	//
	// O inventor e' a fonte que o proprio git reporta. Se a missao nao estiver la', o worktree
	// nao existe -- e e' isso que se responde, em vez de inventar um estado.
	lista, err := agent.InventariarWorktrees(c.Request.Context(), repo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": err.Error()})
		return
	}
	var (
		achado    *agent.WorktreeDescoberto
		candidato string
	)
	sufixo := string(filepath.Separator) + mission
	for i := range lista {
		if strings.HasSuffix(lista[i].CaminhoResolvido, sufixo) {
			achado = &lista[i]
			break
		}
		if lista[i].CriadoPeloHades {
			candidato = lista[i].CaminhoResolvido
		}
	}
	if achado == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"erro":     "missao nao encontrada no inventor deste repositorio",
			"missao":   mission,
			"repo":     repo,
			"dica":     "use GET /api/agent/v1/worktrees?repo=... para ver o que existe",
			"inventor": agent.ResumoDoInventario(lista),
		})
		_ = candidato
		return
	}

	sessao := &agent.GitWorktreeSession{
		MissionID:   mission,
		RepoRoot:    repo,
		WorktreeDir: achado.CaminhoResolvido,
		// O estado LOGICO (em_uso, suspension, arquivado) so' o processo que criou a sessao o
		// conhece. O estado que o DISCO prova e' um so': o worktree existe e esta' pronto.
		//
		//	*Um diagnostico que nao sabe o estado e inventa `criado` mente com um valor
		//	plausivel -- e `criado` nem sequer permite remover.*
		//
		// Derivar `preparado` do que o git reporta e' a leitura mais forte que se pode fazer
		// sem inventar. O campo `estado_observado` diz isso mesmo ao cliente.
		Status: string(agent.WorktreePreparado),
	}

	// Consultar o REGISTO PERSISTIDO, derivando o data root do caminho que o git REPORTou.
	//
	// A derivacao usa o segmento `.agent-worktrees` como ancora. Nao se constroi o caminho do
	// worktree a partir de um data root suposto -- faz-se o inverso.
	//
	//	*Inventar um data root a partir de um caminho desconhecido apontaria o diagnostico para o
	//	registo de outro projection.*
	var registo *agent.RegistoDeWorktrees
	registoEstado := "nao consultado: o caminho nao segue a convencao .agent-worktrees"
	if dataRoot, ok := agent.DataRootDoWorktree(achado.CaminhoResolvido); ok {
		carregado, errRegisto := agent.CarregarRegistoDeWorktrees(agent.CaminhoDoRegisto(dataRoot))
		switch {
		case errRegisto != nil:
			// O registo existe mas nao se le'. NAO se inventa estado nem se trata como ausente:
			// *um registo corrompido lido como vazio apagaria o estado de todos os worktrees.*
			registoEstado = "ilegivel: " + errRegisto.Error()
		case carregado.Contagem() == 0:
			registo = carregado
			registoEstado = "ausente: nenhuma entrada persistida neste data root"
		default:
			registo = carregado
			registoEstado = "consultado"
		}
	}

	diagCiclo := agent.DiagnosticarCicloDeVidaComRegisto(c.Request.Context(), sessao, registo)

	resp := worktreeDiagnosticoResposta{
		MissionID:               mission,
		Estado:                  string(diagCiclo.Estado),
		EstadoValido:            diagCiclo.Estado.Valido(),
		Traduzido:               diagCiclo.Traduzido,
		CommitsNaoIntegrados:    diagCiclo.Disco.CommitsNaoIntegrados,
		FicheirosModificados:    diagCiclo.Disco.FicheirosModificados,
		TemTrabalhoNaoIntegrado: diagCiclo.Disco.TemTrabalhoNaoIntegrado,
		Detalhe:                 diagCiclo.Disco.Detalhe,
		Inconsistente:           diagCiclo.Inconsistente,
		PodeRemover:             diagCiclo.PodeRemover,
		MotivoRemover:           diagCiclo.MotivoRemover,
		AlvoSeguro:              diagCiclo.Disco.AlvoSeguro,
		EstadoDoRegisto:         diagCiclo.EstadoDoRegisto,
		Registo:                 registoEstado,
		EstadoObservado:         diagCiclo.EstadoObservado,
		Proximos:                make([]string, 0, len(diagCiclo.Proximos)),
	}
	for _, e := range diagCiclo.Proximos {
		resp.Proximos = append(resp.Proximos, string(e))
	}
	c.JSON(http.StatusOK, resp)
}
