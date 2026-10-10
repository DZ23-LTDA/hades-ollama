package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Execução de comandos slash (§6 / §11.1).
//
// O registro canônico (internal/agent/slash_registry.go) diz o que existe e para
// qual rota cada comando aponta. Esta camada liga o comando à AÇÃO, com
// autorização server-side, e diz explicitamente quando a execução direta ainda
// não cobre o comando — em vez de responder sucesso sem ter feito nada.

// commandDiscovery lista o registro para a interface (help + autocomplete).
func (a *agentAPI) commandDiscovery(c *gin.Context) {
	prefix := strings.TrimSpace(c.Query("prefix"))
	commands := agent.CompleteSlashCommand(prefix)
	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
		"matched":  len(commands),
		"total":    len(agent.DefaultSlashRegistry()),
		"help":     agent.SlashHelp(),
	})
}

// runCommand interpreta o texto, valida e executa o que já está ligado.
func (a *agentAPI) runCommand(c *gin.Context) {
	var request struct {
		Input string `json:"input"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	invocation, err := agent.ParseSlashInvocation(request.Input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, agent.ErrSlashCommandUnavailable) {
			// 501: o comando é reconhecido, mas a capacidade não existe.
			status = http.StatusNotImplemented
		}
		writeAgentError(c, status, err)
		return
	}
	result, executed, status, err := a.executeCommand(c, invocation)
	if err != nil {
		// Alguns caminhos já escreveram a resposta (guarda de versão/organização):
		// não escrever duas vezes.
		if !errors.Is(err, errCommandAlreadyAnswered) {
			writeAgentError(c, status, err)
		}
		return
	}
	payload := gin.H{
		"command":  invocation.Spec.Name,
		"backend":  invocation.Spec.Backend,
		"executed": executed,
	}
	if executed {
		payload["result"] = result
	} else {
		// Honestidade: o comando está ligado a uma rota real, mas esta camada
		// ainda não executa o comando diretamente.
		payload["hint"] = "comando ligado à rota canônica; a execução direta por /commands ainda não cobre este comando — chame a rota indicada em backend"
	}
	c.JSON(http.StatusOK, payload)
}

// executeCommand despacha o subconjunto já ligado. Devolve (resultado, executado,
// statusHTTP, erro). A organização vem sempre da sessão.
func (a *agentAPI) executeCommand(c *gin.Context, invocation agent.SlashInvocation) (any, bool, int, error) {
	switch invocation.Spec.Name {
	case "/status":
		mission, err := a.missionByID(c, invocation.Args["missao"])
		if err != nil {
			return nil, false, statusForAgentError(err), err
		}
		return mission, true, http.StatusOK, nil

	case "/cancel":
		mission, err := a.missionByID(c, invocation.Args["missao"])
		if err != nil {
			return nil, false, statusForAgentError(err), err
		}
		// Mesma guarda de concorrência otimista da rota (no-op sem If-Match).
		if !missionVersionMatches(c, mission) {
			return nil, false, http.StatusConflict, errCommandAlreadyAnswered
		}
		cancelled, err := a.scopedRuntime(c).Cancel(mission.ID)
		if err != nil {
			return nil, false, statusForAgentError(err), err
		}
		return cancelled, true, http.StatusOK, nil

	case "/memory":
		projectID := invocation.Args["projeto"]
		// Traduz o comando para a forma da rota de projeto e reusa a MESMA
		// checagem de organização, em vez de reimplementar autorização.
		c.Params = append(c.Params, gin.Param{Key: "id", Value: projectID})
		if _, err := a.projectForRequest(c); err != nil {
			return nil, false, statusForAgentError(err), err
		}
		memories, err := a.context.SearchMemoriesContext(c.Request.Context(), projectID, invocation.Args["consulta"], 20)
		if err != nil {
			return nil, false, http.StatusBadRequest, err
		}
		return gin.H{"project_id": projectID, "memories": memories}, true, http.StatusOK, nil

	case "/mcp":
		organizationID := agentOrganizationID(c)
		if !a.authRequired && organizationID == "" {
			organizationID = agent.LocalOrganizationID
		}
		return gin.H{
			"servers":        a.runtime.MCPServersForOrganization(organizationID),
			"remote_servers": a.runtime.RemoteMCPServersForOrganization(organizationID),
		}, true, http.StatusOK, nil

	case "/skills":
		if _, ok := a.requireContextOrganization(c); !ok {
			return nil, false, http.StatusForbidden, errCommandAlreadyAnswered
		}
		return gin.H{"skills": a.context.SkillsForOrganization(a.organizationID(c))}, true, http.StatusOK, nil
	}
	return nil, false, http.StatusOK, nil
}

// errCommandAlreadyAnswered marca que o handler interno já escreveu a resposta
// (por exemplo, o guarda de versão ou de organização), então a camada de comando
// não deve escrever de novo.
var errCommandAlreadyAnswered = errors.New("command response already written")
