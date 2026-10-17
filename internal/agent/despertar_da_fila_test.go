package agent

import (
	"context"
	"testing"
	"time"
)

// PONTA-A-PONTA DO DESPERTAR (E-640): o que faltava para fechar o item do heartbeat.
//
// Sobe um runtime REAL (store + fila + agente), cria uma missão COM agente atribuído e verifica que
// `DespertarAgentes` a enfileira — e que uma missão que espera aprovação NÃO entra na fila, mesmo
// com o mesmo agente. *É este o par que separa «informar» de «executar».*
func TestDespertarAgentesEnfileiraSoOQuePrecisaExecutar(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	pronta, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "missao pronta com agente atribuido",
		AgentID:   "agente-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pronta.AgentID != "agente-a" {
		t.Fatalf("*** a atribuicao nao sobreviveu ao registo ***: %q", pronta.AgentID)
	}

	// A mesma leitura, com uma missão que espera aprovação do MESMO agente.
	aguardando := missaoDeDespertar("mis_aguardando", "agente-a", MissionAwaitingApproval)

	resultado := runtime.DespertarAgentes([]Mission{pronta, aguardando}, time.Now().UTC())

	if len(resultado.Despertares) != 1 {
		t.Fatalf("devia haver UM despertar para agente-a: %+v", resultado.Despertares)
	}
	if len(resultado.Despertares[0].Motivos) != 2 {
		t.Fatalf("o despertar devia levar os dois motivos (trabalho e aprovacao): %+v", resultado.Despertares[0].Motivos)
	}
	if len(resultado.Enfileiradas) != 1 || resultado.Enfileiradas[0] != pronta.ID {
		t.Fatalf("*** so' a missao pronta pode ir a' fila ***: %v", resultado.Enfileiradas)
	}
	if len(resultado.Falhas) != 0 {
		t.Fatalf("nao devia haver falhas: %+v", resultado.Falhas)
	}

	// IDEMPOTENCIA: repetir o despertar nao pode FALHAR nem inventar trabalho. A fila ja' e'
	// idempotente por missao - e e' isso que torna seguro chamar isto a cada tique.
	segundo := runtime.DespertarAgentes([]Mission{pronta, aguardando}, time.Now().UTC())
	if len(segundo.Falhas) != 0 {
		t.Fatalf("repetir o despertar nao pode falhar: %+v", segundo.Falhas)
	}
	if len(segundo.Enfileiradas) != 1 || segundo.Enfileiradas[0] != pronta.ID {
		t.Fatalf("repetir devia apontar a mesma missao: %v", segundo.Enfileiradas)
	}
}

// UMA MISSÃO SEM AGENTE NÃO ENTRA NA FILA POR ESTA VIA — não há a quem atribuir o trabalho, e
// enfileirar «em nome de ninguém» seria inventar um dono.
func TestDespertarAgentesNaoEnfileiraMissaoSemAgente(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	semAgente := missaoDeDespertar("mis_sem_agente", "", MissionReady)
	resultado := runtime.DespertarAgentes([]Mission{semAgente}, time.Now().UTC())
	if len(resultado.Despertares) != 0 || len(resultado.Enfileiradas) != 0 || len(resultado.Falhas) != 0 {
		t.Fatalf("missao sem agente nao devia produzir nada: %+v", resultado)
	}
}
