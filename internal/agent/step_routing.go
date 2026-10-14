package agent

import (
	"strings"

	"github.com/ollama/ollama/internal/multillm"
)

// E-435 (Estagio 43) — A PONTE ENTRE O NO DO PLANO E O MOTOR DE SELECAO.
//
// **O QUE ESTAVA MEDIDO** (e e' uma descoberta, nao um detalhe): o runtime **nao** usa o `multillm.Registry` para
// escolher modelo de PASSO ✗. O roteamento que existe e' o do **planejador** -
// `runtime.go:1141-1151` chama `RoutedPlannerResolver.ResolvePlannerForMission` e **escreve de volta** em
// `mission.Provider`/`mission.Model` ⇒ ***um modelo para a MISSao inteira, escolhido uma vez*** ✓. `grep 'multillm.'`
// em `internal/agent` so' encontra **plugins** e `ContractRegistry` ⇒ *o Registry de modelos nunca entra no runtime
// do agente.*
//
// ⇒ E-433 declarou o contrato (`Step.RequiredCapabilities`/`PinnedModelID`) e **o consumo ficou aberto**. Consumir
// exige **injetar** o Registry no Runtime, o que `RuntimeConfig` permite (`Tools *Registry` ja' mora la') mas
// **ainda nao esta' feito**.
//
// **O QUE ESTA FUNCAO RESOLVE, e o que ela NAO resolve** - *a distincao e' o ponto*:
//   - RESOLVE: a traducao `Step` -> `multillm.RouteRequest`. *Sem ela, quem for ligar o Registry teria de repetir o
//     mapeamento (e o mapeamento errado nao aparece: um campo trocado produz uma escolha plausivel e errada).*
//   - NAO RESOLVE: a injecao no Runtime e a chamada por passo. **Nada consome esta funcao ainda** ✗.
//
// ***Uma ponte nao e' a estrada - mas ponte sem teste e' pior que ponte nenhuma, porque parece caminho.***
func RouteRequestForStep(missao Mission, passo Step) multillm.RouteRequest {
	// As capacidades do PASSO sao pre-requisito do que a MISSAO ja' exige: *um passo que precisa de visao nao
	// dispensa a capacidade que a missao declarou.* Uniao sem duplicata, para o pedido nao crescer a' toa.
	capacidades := make([]string, 0, len(missao.Capabilities)+len(passo.RequiredCapabilities))
	vistas := map[string]bool{}
	for _, origem := range [][]string{missao.Capabilities, passo.RequiredCapabilities} {
		for _, capacidade := range origem {
			limpa := strings.TrimSpace(capacidade)
			if limpa == "" || vistas[limpa] {
				continue
			}
			vistas[limpa] = true
			capacidades = append(capacidades, limpa)
		}
	}
	pedido := multillm.RouteRequest{
		RequiredCapabilities: capacidades,
		PinnedModelID:        strings.TrimSpace(passo.PinnedModelID),
	}
	// O PIN do passo vence o da missao: e' o mais ESPECIFICO. *Um pin de missao nao pode sequestrar a decisao de um
	// passo que declarou o seu proprio.*
	if pedido.PinnedModelID == "" {
		pedido.PinnedModelID = strings.TrimSpace(missao.Model)
	}
	return pedido
}
