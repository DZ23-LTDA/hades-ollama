package agent

import (
	"errors"
	"strings"

	"github.com/ollama/ollama/internal/multillm"
)

// ErrModelRegistryUnavailable marca a tentativa de rotear modelo por passo num runtime **sem** registry.
//
// *Sentinel proprio de proposito:* "nao ha como escolher por passo" e' uma condicao de CONFIGURACAO, e a resposta
// honesta e' **recusar** - nunca cair em silencio para o modelo da missao, porque isso esconderia exactamente o
// defeito que E-433/E-435 vieram corrigir (*a escolha ser do turno e nao do passo*).
var ErrModelRegistryUnavailable = errors.New("model registry is not configured for this runtime")

// Models devolve o registry de modelos injetado, ou nil. *Nulo e' um estado DECLARADO, nao um erro de leitura.*
func (r *Runtime) Models() *multillm.Registry {
	if r == nil {
		return nil
	}
	return r.models
}

// RouteModelForStep escolhe o modelo de UM passo, usando a ponte (E-435) e o registry injetado (E-436).
//
// **ESTA E' A ESTRADA DEPOIS DA PONTE** - a funcao que existe para o executor chamar. *E ela vem com o limite
// declarado no proprio corpo: quem ainda NAO a chama e' o laco de execucao, e isso esta' registrado como limite de
// E-436. Expor a entrada sem ligar o consumidor e' melhor que ligar sem testar - mas nao e' o mesmo que integrado.*
func (r *Runtime) RouteModelForStep(missao Mission, passo Step) (multillm.RouteDecision, error) {
	if r == nil || r.models == nil {
		// FALHA FECHADO: sem registry nao se escolhe por passo, e **nao** se devolve o modelo da missao fingindo
		// que a exigencia do passo foi honrada.
		return multillm.RouteDecision{}, ErrModelRegistryUnavailable
	}
	return r.models.Route(RouteRequestForStep(missao, passo))
}

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
	// 🚨 E-437 — **DOIS VOCABULÁRIOS DIFERENTES, E EU OS HAVIA MISTURADO.** `mission.Capabilities` são **escopos de
	// FERRAMENTA** (`workspace:write`, `crm:write`) — é o que o **motor de ferramentas** confere
	// (`tool must declare known capabilities`). As `RequiredCapabilities` do `multillm.RouteRequest` são
	// **capacidades de MODELO** (`coding`, `vision`, `tools`). *São eixos distintos: um modelo não "tem
	// `workspace:write`".*
	//
	// A minha primeira versão fazia a **união** dos dois ⇒ *todo passo de missão exigiria um modelo com a
	// capacidade-ferramenta, `ErrNoRoute` garantido, e o passo **falharia fechado** por um motivo que não existe* ✗.
	// ***Misturar vocabulários não é rigor: é um erro que passa por rigor - e o modo de falha é fechado, então
	// ninguém desconfia.***
	//
	// ⇒ A ponte usa **SOMENTE** `passo.RequiredCapabilities`. O `Mission` não tem campo de capacidade de **modelo**;
	// *enquanto não tiver, não há o que unir* — e inventar um seria pior que declarar a ausência.
	_ = missao
	capacidades := make([]string, 0, len(passo.RequiredCapabilities))
	vistas := map[string]bool{}
	for _, capacidade := range passo.RequiredCapabilities {
		limpa := strings.TrimSpace(capacidade)
		if limpa == "" || vistas[limpa] {
			continue
		}
		vistas[limpa] = true
		capacidades = append(capacidades, limpa)
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
