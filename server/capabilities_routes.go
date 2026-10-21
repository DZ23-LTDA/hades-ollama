package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// capabilities expõe a VERDADE de capacidade por modelo (§9 da meta).
//
// A LACUNA QUE ISTO FECHA, medida: `ModelCapabilityRegistry.CapabilitySummary()` existe desde ha'
// varias rodadas e o proprio comentario diz *«para exibicao honesta na interface»* — mas a funcao
// NUNCA era chamada fora do teste do proprio pacote: **nao havia rota nem superficie**. O registo
// era honesto e **invisivel para o produto**. *Capacidade que o usuario nao consegue ver nao
// corrige a expectativa dele.*
//
// DUAS RESPOSTAS, e a diferenca entre elas e' o ponto:
//   - sem `model`: os FATOS que o produto tem, cada um com a sua ORIGEM (`medido`/`declarado`) e a
//     POLITICA para o que nao esta' na lista;
//   - com `model`: a resposta sobre AQUELE modelo — e para um modelo desconhecido o estado e'
//     `unknown` com o motivo dito por extenso, ***nunca um "suporta" inventado***.
//
// ferramentasRegistradas devolve os kinds de ferramenta que ESTE runtime registrou, e — quando nao ha' o que
// declarar — o MOTIVO, por extenso (E-668).
//
// A REGRA QUE GOVERNA O VAZIO: *lista vazia e' "nao medido", nunca "nenhuma ferramenta".* Sem runtime ligado nao
// ha' medicao; com runtime ligado e registry vazio o estado tambem e' declarado, e nao apresentado como
// disponibilidade zero. *O sentido do vazio escolhe-se uma vez, e escolhe-se pelo lado que falha fechado.*
func ferramentasRegistradas(runtime *agent.Runtime) ([]string, string) {
	if runtime == nil {
		return nil, "runtime nao ligado nesta resposta: a disponibilidade de ferramentas NAO foi medida"
	}
	kinds := runtime.RegisteredToolKinds()
	if len(kinds) == 0 {
		return nil, "runtime ligado mas sem ferramentas registradas: NAO MEDIDO (registry vazio nao e' 'nenhuma ferramenta')"
	}
	return kinds, ""
}

func (a *agentAPI) capabilities(c *gin.Context) {
	registro := agent.DefaultModelCapabilityRegistry()
	modelo := strings.TrimSpace(c.Query("model"))
	if modelo == "" {
		kinds, motivoAusente := ferramentasRegistradas(a.runtime)
		c.JSON(http.StatusOK, gin.H{
			"models": registro.CapabilitySummary(),
			"policy": "Modelo ausente desta lista esta' NAO_MEDIDO: o runtime cai em modo COMPATIVEL " +
				"(instrucao por texto) e NAO afirma tool call nativo. *Capacidade nao medida nao e' capacidade.*",
			"scope": "fatos medidos/declarados neste ambiente — nao ha' medicao nesta resposta",
			// E-668 (§9, eixo "tool"): as capacidades de MODELO acima sao um registro estatico; o que ESTE
			// processo consegue executar de fato sao as ferramentas REGISTRADAS no runtime - e elas dependem da
			// injecao de dependencia (por exemplo, `model.infer` so' existe quando ha' cliente de inferencia).
			// *Publicar so' os modelos daria a impressao de que a ferramenta esta' disponivel quando o kind nem
			// existe neste processo.*
			"tools": gin.H{
				"registered":    kinds,
				"unavailable":   motivoAusente,
				"dimension":     "kinds de ferramenta registrados no runtime — eixo distinto do `models`",
				"how_to_enable": "`model.infer` exige cliente de inferencia: configure OLLAMA_AGENT_MODEL no servidor",
			},
		})
		return
	}
	if capacidades, conhecido := registro.Lookup(modelo); conhecido {
		modo, motivo := registro.ResolveToolMode(modelo)
		c.JSON(http.StatusOK, gin.H{
			"model":        modelo,
			"known":        true,
			"tool_mode":    modo,
			"reason":       motivo,
			"capabilities": capacidades,
		})
		return
	}
	// DESCONHECIDO: a resposta honesta e' «nao medido», e nao «nao suporta» — sao coisas diferentes,
	// e confundi-las faria o produto recusar um modelo que talvez funcione, ou prometer um que nao.
	c.JSON(http.StatusOK, gin.H{
		"model":     modelo,
		"known":     false,
		"tool_mode": string(agent.ToolModeUnknown),
		"reason": "capacidade NAO MEDIDA para este modelo: o runtime usa modo compativel e DECLARA " +
			"isso em vez de fingir tool call nativo",
	})
}
