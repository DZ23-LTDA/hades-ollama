package agent

import (
	"errors"
	"testing"
)

// E-436 — O RUNTIME RECEBE O REGISTRY DE MODELOS, E **FALHA FECHADO** QUANDO NAO HA.
//
// **O QUE ESTAVA MEDIDO** (E-435): o runtime do agente **nao** usava o `multillm.Registry` para escolher modelo de
// passo ✗ — `runtime.go` roteia o **planejador** e escreve **um** modelo na missao. *Sem injecao, `Step.
// RequiredCapabilities`/`PinnedModelID` (E-433) nao tinham **de onde** ser lidos.*
//
// **O QUE ESTE TESTE MEDE, em duas direcoes** - *e a segunda e' a que quase sempre falta*:
//  1. **com** registry: `RouteModelForStep` escolhe, honrando a ponte (*capacidade do passo e pin*);
//  2. **sem** registry: devolve **`ErrModelRegistryUnavailable`** — *nao cai em silencio para o modelo da missao.*
//
// ***Um runtime que nao sabe rotear por passo e devolve o modelo da missao "como se" roteasse esconde exactamente o
// defeito que E-433/E-435 vieram corrigir: a escolha ser do turno e nao do passo.***
func TestRuntimeRoteiaModeloPorPassoEFalhaFechadoSemRegistry(t *testing.T) {
	// (1) SEM registry: falha fechado, e a mensagem identifica a condicao de CONFIGURACAO.
	semRegistry, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatalf("runtime sem registry: %v", err)
	}
	if semRegistry.Models() != nil {
		t.Fatal("um runtime construido sem `Models` nao pode inventar um registry")
	}
	if _, err := semRegistry.RouteModelForStep(Mission{}, Step{ID: "s1"}); !errors.Is(err, ErrModelRegistryUnavailable) {
		t.Fatalf("ARMADILHA: sem registry o runtime devolveu %v — *tem de FALHAR, nao de escolher o modelo da missao "+
			"fingindo que honrou a exigencia do passo*", err)
	}

	// (2) COM registry: escolhe por passo, honrando capacidade e pin.
	comRegistry, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		Models:        registryDeTeste(t),
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatalf("runtime com registry: %v", err)
	}
	if comRegistry.Models() == nil {
		t.Fatal("o registry injetado tem de ficar acessivel")
	}

	missao := Mission{Objective: "provar o roteamento por passo", Capabilities: []string{"coding"}}
	decisao, err := comRegistry.RouteModelForStep(missao, Step{ID: "s2", PinnedModelID: "local/barato"})
	if err != nil {
		t.Fatalf("rota com pin: %v", err)
	}
	if decisao.Model.ID != "local/barato" {
		t.Fatalf("o pin do passo tinha de vencer no runtime, veio %q", decisao.Model.ID)
	}
	// E a justificativa continua vindo do motor (*routeReason*), que e' o que o Estagio 43 exige registrar.
	if decisao.Reason == "" {
		t.Fatal("a decisao precisa trazer a JUSTIFICATIVA da escolha (*o Estagio 43 exige registra-la*)")
	}

	// E uma exigencia que o pin nao satisfaz tem de FALHAR no runtime tambem - *a ponte nao atropela a capacidade.*
	if _, err := comRegistry.RouteModelForStep(
		Mission{Capabilities: []string{"vision"}},
		Step{ID: "s3", PinnedModelID: "local/barato"}); err == nil {
		t.Fatal("ARMADILHA: pin sem a capacidade exigida passou no runtime — *um pin que nao pode ser honrado tem de " +
			"falhar, em vez de trocar de modelo em silencio (E-428)*")
	}
}
