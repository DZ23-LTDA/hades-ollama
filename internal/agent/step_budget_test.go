package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ollama/ollama/internal/multillm"
)

// E-669 — O TETO DE CUSTO POR PASSO: APLICADO, DECLARADO, e INERTE quando nao declarado.
//
// **O QUE ESTAVA MEDIDO:** `multillm.RouteRequest.MaxCostCents` existia, o motor ja' o aplicava (`router.go:92`) e o
// **unico caller que o definia era um TESTE do proprio `multillm`** ⇒ em producao o teto **nunca existia**. *Um
// controle que ninguem declara nao protege orcamento nenhum* — mesma classe do `RequireNativeToolCalling` que o
// E-663 ligou.
//
// AS TRES DIRECOES, e a terceira e' a que costuma faltar:
//  1. com teto APERTADO, o modelo caro e' excluido e o barato e' escolhido;
//  2. com teto IMPOSSIVEL, a recusa **nomeia o teto** (e nao um "no route" generico, que manda procurar no lugar
//     errado);
//  3. **sem teto declarado, NADA MUDA** — *um controle que altera o comportamento quando nao foi pedido e' tao
//     perigoso quanto um que nunca se aplica.*
func TestTetoDeCustoDoPasso(t *testing.T) {
	// `registryDeTeste`: barato = 1+1 = **2** centavos/1k · visual = 9+9 · forte = 50+50 = **100** centavos/1k.
	registro := registryDeTeste(t)
	missao := Mission{ID: "missao-do-teto", OrganizationID: "org-a"}
	passo := Step{ID: "passo", Kind: "workspace.read", RequiredCapabilities: []string{"coding"}}

	casos := []struct {
		nome       string
		teto       int64
		querModelo string
		querErro   bool
	}{
		{nome: "teto apertado exclui o caro", teto: 10, querModelo: "local/barato"},
		{nome: "sem teto declarado o forte continua vencendo", teto: 0, querModelo: "local/forte"},
		{nome: "teto impossivel recusa", teto: 1, querErro: true},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			runtime, err := NewRuntime(RuntimeConfig{
				Store: NewMemoryStore(), Planner: RulePlanner{}, Models: registro,
				MaxStepCostCents: caso.teto,
				WorkspaceRoot:    t.TempDir() + "-ws", DataRoot: t.TempDir() + "-data",
			})
			if err != nil {
				t.Fatalf("construir o runtime: %v", err)
			}
			// A CONFIGURACAO tem de CHEGAR ao runtime: *configuracao que nao chega e' controle que nao existe.*
			if runtime.maxStepCostCents != caso.teto {
				t.Fatalf("teto nao chegou ao runtime (config=%d, campo=%d)", caso.teto, runtime.maxStepCostCents)
			}

			decisao, err := runtime.RouteModelForStep(missao, passo)
			if caso.querErro {
				if err == nil {
					t.Fatalf("teto %d devia recusar tudo e escolheu %q", caso.teto, decisao.Model.ID)
				}
				if !errors.Is(err, multillm.ErrNoRoute) {
					t.Fatalf("a recusa tem de continuar sendo `ErrNoRoute` para quem ja' a trata, veio: %v", err)
				}
				if !strings.Contains(err.Error(), "teto de custo") || !strings.Contains(err.Error(), "1") {
					t.Fatalf("a recusa tem de NOMEAR o teto aplicado (senao o operador procura no lugar errado): %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("rotejar: %v", err)
			}
			if decisao.Model.ID != caso.querModelo {
				t.Fatalf("com teto %d o modelo tinha de ser %q e foi %q", caso.teto, caso.querModelo, decisao.Model.ID)
			}
		})
	}
}

// O EVENTO DECLARA o teto aplicado e se ele foi declarado — *publicar so' o modelo nao diz sob que restricao de
// orcamento a escolha foi feita, e `0` sem rotulo se confundiria com "sem limite".*
func TestEventoRegistraOTetoDeCustoAplicado(t *testing.T) {
	emissor := &ferramentaQueAnotaModelo{}
	passos := []Step{{
		ID: "anotar", Kind: "workspace.list", Title: "anotar",
		State: StepPending, RequiredCapabilities: []string{"coding"},
	}}
	runtime, id, eventos := missaoRoteada(t, emissor, passos, registryDeTeste(t), true)
	runtime.maxStepCostCents = 10

	if err := runtime.Run(context.Background(), id); err != nil {
		t.Fatalf("executar a missao sob teto: %v", err)
	}
	if emissor.modeloVisto != "local/barato" {
		t.Fatalf("sob teto de 10 centavos o passo tinha de executar no barato e executou em %q", emissor.modeloVisto)
	}
	pacote := payloadDoEvento(t, eventos(), "step.model.route")
	custo, ok := numeroDoPayload(pacote["budget_cents"])
	if !ok || int64(custo) != 10 {
		t.Fatalf("`budget_cents` devia ser 10 no evento, veio %v", pacote["budget_cents"])
	}
	if declarado, _ := pacote["budget_declared"].(bool); !declarado {
		t.Fatalf("`budget_declared` devia ser true com teto declarado, veio %v", pacote["budget_declared"])
	}
}
