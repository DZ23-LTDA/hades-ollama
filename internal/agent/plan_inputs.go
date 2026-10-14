package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrPlanGraphInvalid marca plano cujas DEPENDENCIAS nao formam grafo valido (E-434).
//
// *Sentinel separado de proposito:* "o grafo do plano esta quebrado" e' uma condicao de PLANEJAMENTO que o
// operador corrige mexendo no prompt, na politica ou no modelo - e nao um erro de execucao a ser retentado.
var ErrPlanGraphInvalid = errors.New("plan dependency graph is invalid")

// ValidatePlanGraph recusa, NO PLANEJAMENTO, plano cujas DEPENDENCIAS nao formam um grafo valido.
//
// E-434 (Estagio 43): `Step.DependsOn` entrou como a aresta do DAG, mas **nada a validava** - um plano podia
// declarar dependencia de um passo que **nao existe**, depender de **si mesmo**, ou formar **ciclo**. Num plano
// assim, "esperar o passo X" vira **espera infinita** ou, pior, uma ordem que parece ter causa e nao tem.
//
// ***Sem esta validacao, `DependsOn` e' decoracao: um campo que ninguem confere nao expressa dependencia, expressa
// intencao.***
//
// O que ela NAO faz: nao ORDENA o plano (*a execucao continua em ordem de lista*) e nao promete que o runtime
// respeita a espera. *Validar o grafo e' a condicao para poder respeita-lo depois - nao e' o mesmo que respeitar.*
func ValidatePlanGraph(plan []Step) error {
	// COMPATIBILIDADE DECLARADA: um plano **sem nenhuma** `DependsOn` e' uma LISTA, e lista nao precisa que os
	// passos tenham ID para ser executada em ordem. *Exigir ID de plano antigo seria quebrar o que funciona para
	// impor uma regra que so' importa quando existe aresta.*
	temAresta := false
	for _, passo := range plan {
		if len(passo.DependsOn) > 0 {
			temAresta = true
			break
		}
	}
	if !temAresta {
		return nil
	}
	conhecidos := make(map[string]bool, len(plan))
	for _, passo := range plan {
		id := strings.TrimSpace(passo.ID)
		if id == "" {
			return fmt.Errorf("%w: passo sem ID nao pode ser alvo de dependencia", ErrPlanGraphInvalid)
		}
		if conhecidos[id] {
			// ID repetido e' pior que ausente: a aresta aponta para "algum deles", e a escolha fica sendo a ordem
			// de iteracao - *ambiguidade que se disfarca de determinismo.*
			return fmt.Errorf("%w: passo %q aparece mais de uma vez; dependencia apontaria para um alvo ambiguo",
				ErrPlanGraphInvalid, id)
		}
		conhecidos[id] = true
	}
	for _, passo := range plan {
		id := strings.TrimSpace(passo.ID)
		for _, dependencia := range passo.DependsOn {
			alvo := strings.TrimSpace(dependencia)
			if alvo == "" {
				return fmt.Errorf("%w: passo %q tem dependencia VAZIA", ErrPlanGraphInvalid, id)
			}
			if alvo == id {
				return fmt.Errorf("%w: passo %q depende de SI MESMO", ErrPlanGraphInvalid, id)
			}
			if !conhecidos[alvo] {
				return fmt.Errorf("%w: passo %q depende de %q, que NAO EXISTE no plano", ErrPlanGraphInvalid, id, alvo)
			}
		}
	}
	return validatePlanAcyclic(plan, conhecidos)
}

// validatePlanAcyclic detecta CICLO por remocao sucessiva de passos cujas dependencias ja' foram satisfeitas
// (ordenacao topologica de Kahn). *Um ciclo aqui nao e' erro de digitação: e' um plano que nunca termina.*
func validatePlanAcyclic(plan []Step, conhecidos map[string]bool) error {
	pendentes := make(map[string][]string, len(plan))
	restantes := 0
	for _, passo := range plan {
		id := strings.TrimSpace(passo.ID)
		// 🚨 E-434: **TODO** passo entra em `pendentes`, inclusive o que NAO tem dependencia. *A minha primeira
		// versao so' adicionava dentro do laco de `DependsOn`, entao um passo sem aresta nunca era removido,
		// `removidos` ficava 0 e o plano VALIDO era acusado de CICLO* ✗ — e quem pegou isso foi o **controle** do
		// teste ("senao o teste passaria com a validacao recusando tudo"). ***Validacao que recusa tudo nao e'
		// validacao: e' um alarme que sempre toca.***
		arestas := make([]string, 0, len(passo.DependsOn))
		for _, dependencia := range passo.DependsOn {
			arestas = append(arestas, strings.TrimSpace(dependencia))
		}
		pendentes[id] = arestas
		restantes++
	}
	// Em cada rodada, removemos os passos sem dependencia pendente. Se uma rodada nao remove NADA e ainda sobram
	// passos, o que sobrou esta num ciclo.
	for restantes > 0 {
		removidos := 0
		for _, passo := range plan {
			id := strings.TrimSpace(passo.ID)
			if _, ainda := pendentes[id]; !ainda {
				continue
			}
			livre := true
			for _, dependencia := range pendentes[id] {
				if _, segue := pendentes[dependencia]; segue {
					livre = false
					break
				}
			}
			if livre {
				delete(pendentes, id)
				removidos++
				restantes--
			}
		}
		if removidos == 0 {
			presos := make([]string, 0, len(pendentes))
			for id := range pendentes {
				presos = append(presos, id)
			}
			sort.Strings(presos)
			return fmt.Errorf("%w: CICLO de dependencia entre %s - *um plano ciclico nunca termina, e a ordem da "+
				"lista esconderia isso*", ErrPlanGraphInvalid, strings.Join(presos, ", "))
		}
	}
	return nil
}

// ValidatePlanStepInputs recusa, NO PLANEJAMENTO, etapa cujo input não traz os campos que a
// própria ferramenta declara como obrigatórios.
//
// Por que existe: um modelo real pode devolver um plano sintaticamente válido e
// semanticamente inútil — observado em teste opt-in, `qwen2.5:7b` planejou `workspace.read`
// com `input` vazio. O runtime recusava corretamente (fail-closed), mas com a mensagem
// "anchored workspace path is required", que NÃO diz qual etapa nem qual campo faltou.
// Um plano inválido nunca deve virar erro obscuro na execução: deve ser recusado aqui,
// com a informação que o operador precisa para corrigir o prompt, a política ou o modelo.
func ValidatePlanStepInputs(step Step, descriptor ToolDescriptor) error {
	faltando := make([]string, 0, len(descriptor.RequiredInputs))
	for _, campo := range descriptor.RequiredInputs {
		if !planInputPresente(step.Input, campo) {
			faltando = append(faltando, campo)
		}
	}
	if len(faltando) == 0 {
		return nil
	}
	return fmt.Errorf("plano invalido: etapa %q (%s) exige input %s", step.ID, step.Kind, strings.Join(faltando, ", "))
}

// planInputPresente considera ausente o campo inexistente, nulo, ou string em branco: um
// caminho vazio não ancora nada, então tratá-lo como presente só empurraria o erro adiante.
func planInputPresente(input map[string]any, campo string) bool {
	valor, ok := input[campo]
	if !ok || valor == nil {
		return false
	}
	if texto, ok := valor.(string); ok {
		return strings.TrimSpace(texto) != ""
	}
	return true
}
