package agent

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// planejadorContabilizado registra quantas vezes foi chamado e reporta consumo a cada chamada.
// Serve para provar que o registro da missão CONSISTE com o número real de chamadas ao modelo:
// se o runtime trocasse o consumo em vez de somar, o custo da retomada seria SUBDECLARADO em
// silêncio — um erro que não aparece em CI verde nem em teste de soma isolada.
type planejadorContabilizado struct {
	chamadas int
	destino  *ModelUsage
	passos   []Step
}

func (p *planejadorContabilizado) SetPlanUsageSink(destino *ModelUsage) { p.destino = destino }

func (p *planejadorContabilizado) Plan(_ context.Context, _ Mission) ([]Step, error) {
	p.chamadas++
	if p.destino != nil {
		p.destino.Calls++
		p.destino.Provider = "provedor-de-teste"
		p.destino.Model = "modelo-de-teste"
		p.destino.PromptTokens += 10
		p.destino.CompletionTokens += 5
		p.destino.CostCents += 1
		p.destino.CostKnown = true
	}
	return p.passos, nil
}

// TestConsumoDoPlanejamentoConsisteComAsChamadasReais prova a invariante de contabilidade: o
// consumo registrado na missão tem de ser a SOMA de todas as chamadas de planejamento, não apenas
// a última. A missão pede aprovação (workspace.write), então ela é executada em DUAS rodadas —
// exatamente o caminho de retomada em que a contabilidade pode errar.
func TestConsumoDoPlanejamentoConsisteComAsChamadasReais(t *testing.T) {
	raiz := t.TempDir()
	store, err := NewJSONStore(filepath.Join(raiz, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	planejador := &planejadorContabilizado{passos: []Step{{
		ID:    "step_1",
		Kind:  "workspace.write",
		Title: "escrever arquivo de contabilidade",
		Input: map[string]any{"path": "contabilidade.txt", "content": "ok"},
	}}}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         store,
		Planner:       planejador,
		WorkspaceRoot: raiz, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	missao, err := runtime.CreateMission(ctx, CreateMissionRequest{Objective: "escrever arquivo de contabilidade", Capabilities: []string{"workspace:write"}})
	if err != nil {
		t.Fatal(err)
	}

	// Duas rodadas: a primeira planeja e para na aprovação; a segunda executa o passo aprovado.
	for rodada := range 2 {
		if err := runtime.Run(ctx, missao.ID); err != nil {
			t.Fatalf("rodada %d: %v", rodada+1, err)
		}
		atual, err := runtime.GetMission(missao.ID)
		if err != nil {
			t.Fatal(err)
		}
		missao = atual
		for _, a := range missao.Approvals {
			if strings.EqualFold(string(a.Status), "pending") {
				if _, err := runtime.DecideApproval(missao.ID, a.ID, true, "aprovado no teste de contabilidade"); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	if missao.Usage == nil {
		t.Fatal("missão sem registro de consumo")
	}
	t.Logf("CONTABILIDADE: chamadas_ao_planejador=%d chamadas_registradas=%d tokens=%d/%d custo_centavos=%d",
		planejador.chamadas, missao.Usage.Calls, missao.Usage.PromptTokens, missao.Usage.CompletionTokens, missao.Usage.CostCents)

	if planejador.chamadas == 0 {
		t.Fatal("o planejador nunca foi chamado: o teste perdeu o propósito")
	}
	// Invariante central: o registro tem de refletir TODAS as chamadas (soma), com 10/5 e 1 centavo cada.
	if missao.Usage.Calls != planejador.chamadas {
		t.Fatalf("chamadas registradas=%d, chamadas reais ao planejador=%d: contabilidade divergente", missao.Usage.Calls, planejador.chamadas)
	}
	if esperado := 10 * planejador.chamadas; missao.Usage.PromptTokens != esperado {
		t.Fatalf("tokens de prompt=%d, esperado %d (10 por chamada x %d): consumo trocado em vez de somado",
			missao.Usage.PromptTokens, esperado, planejador.chamadas)
	}
	if esperado := 5 * planejador.chamadas; missao.Usage.CompletionTokens != esperado {
		t.Fatalf("tokens de conclusão=%d, esperado %d", missao.Usage.CompletionTokens, esperado)
	}
	if esperado := int64(planejador.chamadas); missao.Usage.CostCents != esperado {
		t.Fatalf("custo=%d centavos, esperado %d", missao.Usage.CostCents, esperado)
	}
	if !missao.Usage.CostKnown {
		t.Fatal("custo marcado como desconhecido apesar de todas as chamadas terem tarifa")
	}
}

// TestConsumoDePlanejamentoSobreviveAReexecucaoGaranteSoma cobre a segunda execução DEPOIS de a
// missão já ter consumo gravado no store: o consumo tem de continuar somando, nunca zerar nem
// substituir — é o caso de retomada após reinício do processo.
func TestConsumoDePlanejamentoPersistidoNaoEhPerdidoNemSubstituido(t *testing.T) {
	raiz := t.TempDir()
	store, err := NewJSONStore(filepath.Join(raiz, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	planejador := &planejadorContabilizado{passos: []Step{{
		ID:    "step_1",
		Kind:  "workspace.list",
		Title: "listar o workspace",
		Input: map[string]any{"path": "."},
	}}}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Planner: planejador, WorkspaceRoot: raiz, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	missao, err := runtime.CreateMission(ctx, CreateMissionRequest{Objective: "listar o workspace"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Run(ctx, missao.ID); err != nil {
		t.Fatal(err)
	}
	primeira, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatal(err)
	}
	if primeira.Usage == nil || primeira.Usage.PromptTokens == 0 {
		t.Fatal("primeira execução não registrou consumo")
	}
	// Recarrega do store (mesmo caminho de um processo reiniciado) e executa de novo.
	store2, err := NewJSONStore(filepath.Join(raiz, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	runtime2, err := NewRuntime(RuntimeConfig{Store: store2, Planner: planejador, WorkspaceRoot: raiz, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime2.Run(ctx, missao.ID); err != nil {
		t.Fatal(err)
	}
	segunda, err := runtime2.GetMission(missao.ID)
	if err != nil {
		t.Fatal(err)
	}
	if segunda.Usage == nil {
		t.Fatal("consumo perdido após reexecução")
	}
	if segunda.Usage.PromptTokens < primeira.Usage.PromptTokens {
		t.Fatalf("consumo DIMINUIU após reexecução: %d -> %d", primeira.Usage.PromptTokens, segunda.Usage.PromptTokens)
	}
	t.Logf("RETOMADA: chamadas=%d tokens=%d/%d custo_centavos=%d",
		segunda.Usage.Calls, segunda.Usage.PromptTokens, segunda.Usage.CompletionTokens, segunda.Usage.CostCents)
}
