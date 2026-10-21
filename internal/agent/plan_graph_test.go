package agent

import (
	"errors"
	"strings"
	"testing"
)

// E-434 (Estagio 43) — O DAG DO PLANO E' VALIDADO: DEPENDENCIA INEXISTENTE, EM SI MESMA E **CICLO**.
//
// `Step.DependsOn` entrou em E-433 como a aresta do DAG, **e nada a validava** ✗ ⇒ *um plano podia declarar
// dependencia de um passo que **nao existe**, de **si mesmo**, ou formar **ciclo*** ⇒ *"esperar o passo X" vira
// **espera infinita**, ou uma ordem que **parece** ter causa e nao tem.*
//
// ***Sem esta validacao, `DependsOn` e' decoracao: um campo que ninguem confere nao expressa dependencia, expressa
// intencao.***
//
// E O QUE ELA NAO FAZ, declarado no proprio codigo: **nao ordena** o plano (*a execucao segue em ordem de lista*) e
// **nao promete** que o runtime respeita a espera. *Validar o grafo e' a condicao para poder respeita-lo depois -
// nao e' o mesmo que respeitar.*
func TestValidatePlanGraphRecusaGrafoInvalido(t *testing.T) {
	casos := []struct {
		nome   string
		plano  []Step
		contem string
		valido bool
		porque string
	}{
		{
			nome:   "plano sem dependencias e' valido",
			plano:  []Step{{ID: "a"}, {ID: "b"}},
			valido: true,
			porque: "controle: sem arestas nao ha grafo a recusar",
		},
		{
			nome:   "cadeia simples e' valida",
			plano:  []Step{{ID: "a"}, {ID: "b", DependsOn: []string{"a"}}, {ID: "c", DependsOn: []string{"b"}}},
			valido: true,
			porque: "controle: DAG sem ciclo tem de passar, senao o teste so' mede recusa",
		},
		{
			nome:   "dependencia INEXISTENTE",
			plano:  []Step{{ID: "a"}, {ID: "b", DependsOn: []string{"fantasma"}}},
			contem: "NAO EXISTE",
			porque: "*esperar o que nao esta no plano e' espera que nunca termina*",
		},
		{
			nome:   "dependencia de SI MESMO",
			plano:  []Step{{ID: "a", DependsOn: []string{"a"}}},
			contem: "SI MESMO",
			porque: "auto-dependencia e' um ciclo de tamanho 1",
		},
		{
			nome:   "CICLO de dois",
			plano:  []Step{{ID: "a", DependsOn: []string{"b"}}, {ID: "b", DependsOn: []string{"a"}}},
			contem: "CICLO",
			porque: "*um plano ciclico nunca termina, e a ordem da lista esconderia isso*",
		},
		{
			nome:   "CICLO de tres com uma aresta valida no meio",
			plano:  []Step{{ID: "ok"}, {ID: "a", DependsOn: []string{"b", "ok"}}, {ID: "b", DependsOn: []string{"c"}}, {ID: "c", DependsOn: []string{"a"}}},
			contem: "CICLO",
			porque: "o ciclo tem de ser achado MESMO com passo valido no mesmo plano",
		},
		{
			nome:   "ID repetido",
			plano:  []Step{{ID: "a"}, {ID: "a", DependsOn: []string{"a"}}},
			contem: "mais de uma vez",
			porque: "*a aresta apontaria para 'algum deles' - ambiguidade disfarcada de determinismo* " +
				"(*o caso precisa de ARESTA: sem ela o plano e' lista, e ID nao e' exigido por compatibilidade*)",
		},
		{
			nome:   "ID vazio",
			plano:  []Step{{ID: "  ", DependsOn: []string{"alvo"}}, {ID: "alvo"}},
			contem: "sem ID",
			porque: "passo sem ID nao pode ser alvo de dependencia (*e o alvo existe: o que falta e' o ID do passo*)",
		},
		{
			nome:   "dependencia VAZIA",
			plano:  []Step{{ID: "a"}, {ID: "b", DependsOn: []string{"   "}}},
			contem: "VAZIA",
			porque: "aresta em branco e' aresta que nao se pode conferir",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			err := ValidatePlanGraph(caso.plano)
			if caso.valido {
				if err != nil {
					t.Fatalf("plano valido recusado (%s): %v", caso.porque, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ARMADILHA: plano INVALIDO aceito (%s) - *o grafo nao esta sendo conferido*", caso.porque)
			}
			if !errors.Is(err, ErrPlanGraphInvalid) {
				t.Fatalf("o erro precisa ser o sentinela ErrPlanGraphInvalid: %v", err)
			}
			if !strings.Contains(err.Error(), caso.contem) {
				t.Fatalf("a mensagem precisa dizer o motivo (%q), veio: %v", caso.contem, err)
			}
		})
	}
}

// E-434: e a validacao esta' LIGADA no caminho que todo save atravessa.
//
// *Um validador que ninguem chama e' a mesma classe do instrumento que nunca dispara (E-432): parece controle e nao
// e'.* Este teste falha se alguem remover a chamada de `validateMissionPersistenceBounds`.
func TestPlanoCiclicoNaoPodeSerPersistido(t *testing.T) {
	missao := Mission{
		ID:        "mis_ciclo",
		Objective: "provar que o ciclo nao entra no disco",
		State:     MissionCreated,
		Plan: []Step{
			{ID: "a", Kind: "agent", Title: "a", State: StepPending, DependsOn: []string{"b"}},
			{ID: "b", Kind: "agent", Title: "b", State: StepPending, DependsOn: []string{"a"}},
		},
	}
	if err := validateMissionPersistenceBounds(missao); err == nil {
		t.Fatal("ARMADILHA: um plano CICLICO passou pela validacao de persistencia - *a chamada de " +
			"ValidatePlanGraph saiu de validateMissionPersistenceBounds*")
	}
	// CONTROLE: sem o ciclo, a mesma missao tem de passar - *senao o teste acima passaria com a validacao recusando
	// tudo, que e' um defeito diferente.*
	missao.Plan[0].DependsOn = nil
	missao.Plan[1].DependsOn = nil
	if err := validateMissionPersistenceBounds(missao); err != nil {
		t.Fatalf("a missao sem ciclo tem de passar: %v", err)
	}
}
