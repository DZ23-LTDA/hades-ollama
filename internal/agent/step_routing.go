package agent

import (
	"errors"
	"fmt"
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
// ⚠️ **CORRECAO POR MEDICAO (E-910, 2026-10-16)**: o texto abaixo dizia *"quem ainda NAO a chama e' o laco de
// execucao"* - *e isso ficou **DESACTUALIZADO**:* **`runtime.go:2275` CHAMA `r.RouteModelForStep(mission, *step)`**,
// dentro do laco de execucao. Texto antigo, preservado como registo historico:
//
//	*"E ela vem com o limite declarado no proprio corpo: quem ainda NAO a chama e' o laco de execucao, e isso
//	 esta' registrado como limite de E-436."*
//
// O que **passou a existir** (medido): a chamada por passo, a falha fechado quando o roteamento falha, a conferencia
// de chamada nativa (`E-663`), o evento `step.model.route` com candidatos e custo (`E-664`), e a entrega do
// `modeloRoteado` ao `ToolContext` (`E-438`). *O que **nao** mudou: o executor do passo.*
func (r *Runtime) RouteModelForStep(missao Mission, passo Step) (multillm.RouteDecision, error) {
	if r == nil || r.models == nil {
		// FALHA FECHADO: sem registry nao se escolhe por passo, e **nao** se devolve o modelo da missao fingindo
		// que a exigencia do passo foi honrada.
		return multillm.RouteDecision{}, ErrModelRegistryUnavailable
	}
	requisicao := RouteRequestForStep(missao, passo)
	// E-669 (§4/§7) — O TETO DE CUSTO QUE EXISTIA E NINGUEM DECLARAVA.
	//
	// `multillm.RouteRequest.MaxCostCents` ja' existia e o motor ja' o APLICAVA (`router.go:92`) — mas o **unico
	// caller que o definia era um TESTE do proprio `multillm`**. *Um teto que ninguem declara nao protege orcamento
	// nenhum.* Aqui a politica do runtime entra na requisicao, **num unico lugar** — e nao em cada chamador, que e'
	// como um controle se perde.
	requisicao.MaxCostCents = r.maxStepCostCents
	decisao, err := r.models.Route(requisicao)
	if err != nil && r.maxStepCostCents > 0 && errors.Is(err, multillm.ErrNoRoute) {
		// A recusa passa a NOMEAR a causa. `ErrNoRoute` sozinho ("no provider route satisfies the requested
		// constraints") nao diz QUAL restricao excluiu tudo — e mandar o operador procurar no lugar errado e' pior
		// do que nao dizer nada. *O erro tem de dizer qual restricao excluiu.*
		return multillm.RouteDecision{}, fmt.Errorf("%w: nenhum modelo cabe no teto de custo por passo declarado (%d centavos/1k)",
			err, r.maxStepCostCents)
	}
	return decisao, err
}

// E-435 (Estagio 43) — A PONTE ENTRE O NO DO PLANO E O MOTOR DE SELECAO.
//
// ⚠️ **CORRECAO POR MEDICAO (E-910, 2026-10-16)**: as duas afirmacoes abaixo ficaram **DESACTUALIZADAS**.
// Texto antigo, preservado como registo historico:
//
//	*"`grep 'multillm.'` em `internal/agent` so' encontra plugins e `ContractRegistry` ⇒ o Registry de modelos
//	 nunca entra no runtime do agente."*  e  *"Consumir exige injetar o Registry no Runtime … mas ainda nao esta'
//	 feito."*
//
// **Medido agora**: o Registry **entra** - `runtime.go:2274` testa `r.models != nil` antes de rotear por passo - e
// `runtime.go:2275` **chama** `r.RouteModelForStep`. *O `grep` era um instrumento que **nao via** a injecao: ela
// passa por `RuntimeConfig`, nao por uma mencao literal a `multillm.` no sitio onde eu procurei.*
//
// ⇒ E-433 declarou o contrato (`Step.RequiredCapabilities`/`PinnedModelID`) e o consumo **foi fechado** no
// `E-437`/`E-438`: a decisao e' tomada, registada no evento `step.model.route` e entregue ao `ToolContext`.
//
// **O QUE ESTA FUNCAO RESOLVE, e o que ela NAO resolve** - *a distincao e' o ponto*:
//   - RESOLVE: a traducao `Step` -> `multillm.RouteRequest`. *Sem ela, quem for ligar o Registry teria de repetir o
//     mapeamento (e o mapeamento errado nao aparece: um campo trocado produz uma escolha plausivel e errada).*
//   - **RESOLVIDO desde entao** (E-910): a injecao no Runtime e a chamada por passo. *O texto antigo dizia
//     «a injecao no Runtime e a chamada por passo. **Nada consome esta funcao ainda** ✗» - e isso deixou de ser
//     verdade: `runtime.go:2275` e' o consumidor.* O que **segue aberto** e' outro incremento: fazer o **executor**
//     do passo seguir o modelo roteado (hoje so' a **ferramenta de inferencia** o segue).
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

// registroDeCapacidadesDeModelo devolve o registro injetado ou, quando ausente, o registro MEDIDO do ambiente.
//
// *Nunca devolve nil:* o gate do passo consulta este registro, e um nil transformaria "nao medido" em panic.
func registroDeCapacidadesDeModelo(injetado *ModelCapabilityRegistry) *ModelCapabilityRegistry {
	if injetado != nil {
		return injetado
	}
	return DefaultModelCapabilityRegistry()
}

// E-663 — A EXIGENCIA DE CHAMADA NATIVA DE FERRAMENTA PASSA A SER APLICADA NO PASSO.
//
// **O QUE ESTAVA MEDIDO (06:45-03):** `ModelCapabilityRegistry.RequireNativeToolCalling` existia
// (`model_capabilities.go:139`) e **so' era chamado em testes**. Um passo que declara a capacidade de modelo
// `tools` podia ser roteado para um modelo **MEDIDO** como `compatible_text` e **executar assim mesmo** - o
// "tool call fingido" que o V2.2 proibe. *O registro sabia dizer; ninguem perguntava.*
//
// **A DISTINCAO QUE O PROPRIO REGISTRO FAZ - e que aqui e' respeitada (ela e' o ponto):**
//   - **MEDIDO como sem chamada nativa** (`ErrModelWithoutNativeToolCalling`) ⇒ **RECUSA** o passo: a exigencia
//     declarada nao pode ser honrada, e executar esconderia exatamente isso;
//   - **NAO MEDIDO** (`ErrModelCapabilityUnknown`) ⇒ **nao recusa**: *ausencia de medicao nao e' prova de
//     incapacidade* - o proprio `ResolveToolMode` diz "nao medido", nunca "nao suporta". O fato, porem, vai para
//     a trilha, porque *resultado ausente e' `NAO_MEDIDO`, nao aprovacao*.
//
// Devolve o rotulo honesto para o evento; **erro nao-nulo significa RECUSAR o passo**.
func (r *Runtime) conferirFerramentaNativa(passo Step, modeloRoteado string) (string, error) {
	if !exigeFerramentaNativa(passo) {
		return "nao_exigido", nil
	}
	if strings.TrimSpace(modeloRoteado) == "" {
		// O ramo so' entra com registry de modelos, e o roteamento falha fechado antes daqui. Declarar a ausencia
		// e' melhor que fingir conformidade.
		return "sem_modelo_roteado", nil
	}
	erro := r.modelCapabilities.RequireNativeToolCalling(modeloRoteado)
	switch {
	case erro == nil:
		return "nativo", nil
	case errors.Is(erro, ErrModelCapabilityUnknown):
		return "nao_medido", nil
	default:
		return "recusado", fmt.Errorf("passo %q exige chamada nativa de ferramenta: %w", passo.ID, erro)
	}
}

// exigeFerramentaNativa diz se o passo declara a capacidade de MODELO `tools`.
//
// A comparacao e' por CAPACIDADE e nao por texto livre: `RequiredCapabilities` e' o vocabulario de capacidade de
// **modelo** (`coding`, `vision`, `tools`) - eixo distinto do escopo de **ferramenta** (`workspace:write`), como o
// E-437 deixou registrado. *Misturar os dois vocabularios foi um defeito real deste repositorio: um passo de missao
// passou a exigir uma capacidade-ferramenta do MODELO e falhava fechado por um motivo que nao existia.*
func exigeFerramentaNativa(passo Step) bool {
	for _, capacidade := range passo.RequiredCapabilities {
		if strings.EqualFold(strings.TrimSpace(capacidade), "tools") {
			return true
		}
	}
	return false
}

// limiteDeCandidatosNaTrilha e' quantos candidatos a trilha guarda. *O numero e' DECLARADO de proposito:*
// guardar todos faria o evento crescer sem limite, e guardar zero apagaria a resposta a "por que ESTE e nao aquele".
const limiteDeCandidatosNaTrilha = 5

// trilhaDeCandidatos resume QUEM foi considerado na escolha do modelo para o passo (E-664).
//
// *POR QUE ISTO EXISTE:* o §4 exige persistir **decisao, motivo, CANDIDATO e custo estimado**. O evento
// `step.model.route` registrava o **vencedor** e o motivo, mas **nao os preteridos** - e uma escolha sem os
// preteridos nao e' auditavel: nao da' para responder *"por que ESTE e nao aquele"*, que e' a pergunta que o
// estagio 43 faz.
//
// A fonte e' o **proprio motor**: `multillm.Registry.RouteCandidates`, que ja' devolve a lista elegivel ordenada.
// *Nao se recalcula o scoring aqui* - recalcular seria criar uma **segunda verdade** sobre a mesma decisao, e as
// duas divergiriam em silencio (a classe de defeito do E-619).
//
// Devolve `nil` quando nao ha como medir, e o chamador **declara** isso: `nil` significa "nao medido", nunca
// "nenhum candidato".
func (r *Runtime) trilhaDeCandidatos(missao Mission, passo Step, limite int) []map[string]any {
	if r == nil || r.models == nil {
		return nil
	}
	candidatos, err := r.models.RouteCandidates(RouteRequestForStep(missao, passo))
	if err != nil || len(candidatos) == 0 {
		return nil
	}
	if limite <= 0 || limite > len(candidatos) {
		limite = len(candidatos)
	}
	trilha := make([]map[string]any, 0, limite)
	for _, candidato := range candidatos[:limite] {
		trilha = append(trilha, map[string]any{
			"model":             candidato.Model.ID,
			"provider":          candidato.Model.Provider,
			"score":             candidato.Score,
			"cost_cents_per_1k": candidato.Model.CostPer1KInputCents + candidato.Model.CostPer1KOutputCents,
			"reason":            candidato.Reason,
		})
	}
	return trilha
}
