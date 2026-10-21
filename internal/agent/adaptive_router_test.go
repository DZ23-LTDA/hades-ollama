package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/ollama/ollama/internal/multillm"
)

// E-658 — O ROTEADOR ADAPTATIVO LIGADO AO CAMINHO CANONICO.
//
// **O QUE ESTAVA MEDIDO**: `adaptive_router.go` existia, compilava e **ninguem o chamava** — e, pior, decidia por um
// **segundo** caminho (`multillmReg.Route`), paralelo ao `Runtime.RouteModelForStep` (o caminho auditado, com a ponte
// E-435 e o contrato do passo). Dois roteadores decidindo em paralelo e' a forma mais barata de divergir em
// silencio: os testes abaixo **exigem a fonte unica de decisao** e a **mesma recusa** do caminho canonico.
//
// A prova central nao depende de "qual modelo ganha" (*isso e' politica do motor de scoring*): ela compara a decisao
// do router com a do caminho canonico para a MESMA exigencia.

func runtimeDeRoteamento(t *testing.T) *Runtime {
	t.Helper()
	rt, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		Models:        registryDeTeste(t),
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatalf("construir runtime de roteamento: %v", err)
	}
	return rt
}

// (1) FONTE UNICA DE DECISAO: o router nao pode escolher diferente do caminho canonico.
func TestAdaptiveRouterDecidePeloMesmoCaminhoDoRuntime(t *testing.T) {
	rt := runtimeDeRoteamento(t)
	router := NewAdaptiveRouterParaRuntime(rt, DefaultModelCapabilityRegistry())

	caps := []string{"coding"}
	sub := Subtask{ID: "a", Objective: "escolher executor", RequiredCapabilities: caps}

	decidido, err := router.RouteSubtask(context.Background(), sub)
	if err != nil {
		t.Fatalf("RouteSubtask: %v", err)
	}
	canonica, err := rt.RouteModelForStep(Mission{}, Step{ID: "a", RequiredCapabilities: caps})
	if err != nil {
		t.Fatalf("RouteModelForStep: %v", err)
	}
	if decidido.ChosenModel == "" {
		t.Fatal("o router devolveu decisao SEM modelo escolhido")
	}
	if decidido.ChosenModel != canonica.Model.ID {
		t.Fatalf("DIVERGENCIA ENTRE ROTEADORES: adaptativo escolheu %q e o caminho canonico %q — *dois roteadores "+
			"decidindo em paralelo*", decidido.ChosenModel, canonica.Model.ID)
	}
	if decidido.SubtaskID != "a" {
		t.Fatalf("a decisao tem de carregar a subtarefa que perguntou, veio %q", decidido.SubtaskID)
	}
	if decidido.Reason == "" {
		t.Fatal("a decisao precisa trazer a JUSTIFICATIVA (o Estagio 43 exige registra-la)")
	}
	// A confianca NAO pode ser inventada: o score do motor e' um inteiro BRUTO, nao uma confianca 0..1.
	if decidido.Confidence != 0 {
		t.Fatalf("a confianca nao pode ser inventada a partir do score bruto; veio %v", decidido.Confidence)
	}
}

// (2) FALHA FECHADO: sem runtime (ou sem registry) NAO se escolhe — a mesma recusa do caminho canonico.
func TestAdaptiveRouterFalhaFechadoSemRuntimeESemRegistry(t *testing.T) {
	semRuntime := NewAdaptiveRouterParaRuntime(nil, nil)
	if _, err := semRuntime.RouteSubtask(context.Background(),
		Subtask{ID: "a", RequiredCapabilities: []string{"coding"}}); !errors.Is(err, ErrModelRegistryUnavailable) {
		t.Fatalf("ARMADILHA: router sem runtime devolveu %v — *tem de FALHAR, nao de escolher fingindo que honrou a "+
			"exigencia da subtarefa*", err)
	}

	rtSemRegistry, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatalf("runtime sem registry: %v", err)
	}
	router := NewAdaptiveRouterParaRuntime(rtSemRegistry, nil)
	if _, err := router.RouteSubtask(context.Background(),
		Subtask{ID: "a", RequiredCapabilities: []string{"coding"}}); !errors.Is(err, ErrModelRegistryUnavailable) {
		t.Fatalf("router sobre runtime sem registry devolveu %v; tinha de recusar igual ao caminho canonico", err)
	}
}

// (3) PIN: o pin da subtarefa vence e e' honrado (sem passar pelo scoring).
func TestAdaptiveRouterHonraPinDaSubtarefa(t *testing.T) {
	router := NewAdaptiveRouterParaRuntime(runtimeDeRoteamento(t), DefaultModelCapabilityRegistry())
	decidido, err := router.RouteSubtask(context.Background(), Subtask{
		ID:                   "p1",
		RequiredCapabilities: []string{"coding"},
		PinnedModelID:        "local/barato",
	})
	if err != nil {
		t.Fatalf("rota com pin: %v", err)
	}
	if decidido.ChosenModel != "local/barato" {
		t.Fatalf("o pin da subtarefa nao foi honrado: veio %q", decidido.ChosenModel)
	}
}

// (4) CAPACIDADE IMPOSSIVEL: erro — nunca uma decisao silenciosa.
func TestAdaptiveRouterNaoInventaModeloParaCapacidadeImpossivel(t *testing.T) {
	router := NewAdaptiveRouterParaRuntime(runtimeDeRoteamento(t), DefaultModelCapabilityRegistry())
	decidido, err := router.RouteSubtask(context.Background(), Subtask{
		ID:                   "imp",
		RequiredCapabilities: []string{"capability-que-ninguem-tem"},
	})
	if err == nil {
		t.Fatalf("ARMADILHA: exigencia impossivel devolveu decisao %+v em vez de erro", decidido)
	}
	if !errors.Is(err, multillm.ErrNoRoute) {
		t.Fatalf("o erro tem de ser o do motor de rotas (ErrNoRoute), veio %v", err)
	}
}

// (5) CACHE: acerta, NAO troca a identidade de quem pergunta, e conta a estatistica.
func TestAdaptiveRouterCacheNaoTrocaIdentidade(t *testing.T) {
	router := NewAdaptiveRouterParaRuntime(runtimeDeRoteamento(t), DefaultModelCapabilityRegistry())
	ctx := context.Background()

	primeira, err := router.RouteSubtask(ctx, Subtask{ID: "a", RequiredCapabilities: []string{"coding"}})
	if err != nil {
		t.Fatalf("primeira rota: %v", err)
	}
	if primeira.CacheHit {
		t.Fatal("a PRIMEIRA rota nao pode ser cache hit")
	}
	segunda, err := router.RouteSubtask(ctx, Subtask{ID: "b", RequiredCapabilities: []string{"coding"}})
	if err != nil {
		t.Fatalf("segunda rota: %v", err)
	}
	if !segunda.CacheHit {
		t.Fatal("a segunda chamada com a MESMA exigencia tinha de ser cache hit")
	}
	if segunda.ChosenModel != primeira.ChosenModel {
		t.Fatalf("o cache devolveu modelo diferente: %q vs %q", segunda.ChosenModel, primeira.ChosenModel)
	}
	if segunda.SubtaskID != "b" {
		t.Fatalf("DEFEITO: o cache devolveu a identidade %q em vez da subtarefa que perguntou (\"b\")", segunda.SubtaskID)
	}
	estatisticas := router.Stats()
	if estatisticas.CacheHits != 1 || estatisticas.CacheMisses != 1 || estatisticas.TotalRequests != 2 {
		t.Fatalf("estatistica incoerente: %+v", estatisticas)
	}

	// InvalidateCache tem de zerar o cache (nao a estatistica).
	router.InvalidateCache()
	terceira, err := router.RouteSubtask(ctx, Subtask{ID: "c", RequiredCapabilities: []string{"coding"}})
	if err != nil {
		t.Fatalf("terceira rota: %v", err)
	}
	if terceira.CacheHit {
		t.Fatal("apos InvalidateCache a rota nao pode vir do cache")
	}
}

// (6) CONCORRENCIA: prova sob `-race` que a estatistica e o cache nao tem corrida.
func TestAdaptiveRouterConcorrenteSemCorrida(t *testing.T) {
	router := NewAdaptiveRouterParaRuntime(runtimeDeRoteamento(t), DefaultModelCapabilityRegistry())
	ctx := context.Background()

	const goroutines, porGoroutine = 16, 8
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := range porGoroutine {
				subtarefa := Subtask{
					ID:                   fmt.Sprintf("g%d-i%d", g, i),
					RequiredCapabilities: []string{"coding"},
				}
				decidido, err := router.RouteSubtask(ctx, subtarefa)
				if err != nil {
					t.Errorf("rota concorrente %s: %v", subtarefa.ID, err)
					return
				}
				if decidido.SubtaskID != subtarefa.ID {
					t.Errorf("identidade trocada em concorrencia: pedido %q, devolvido %q", subtarefa.ID, decidido.SubtaskID)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	estatisticas := router.Stats()
	if estatisticas.TotalRequests != goroutines*porGoroutine {
		t.Fatalf("requisicoes contadas %d, esperadas %d", estatisticas.TotalRequests, goroutines*porGoroutine)
	}
	if estatisticas.CacheHits+estatisticas.CacheMisses != estatisticas.TotalRequests {
		t.Fatalf("hits+misses (%d) tem de fechar com o total (%d)",
			estatisticas.CacheHits+estatisticas.CacheMisses, estatisticas.TotalRequests)
	}
}

// (7) BATCH: ordem por prioridade (maior primeiro) e cancelamento honrado.
func TestAdaptiveRouterBatchOrdenaPorPrioridadeECancela(t *testing.T) {
	router := NewAdaptiveRouterParaRuntime(runtimeDeRoteamento(t), DefaultModelCapabilityRegistry())
	lote := []Subtask{
		{ID: "baixa", RequiredCapabilities: []string{"coding"}, Priority: 1},
		{ID: "alta", RequiredCapabilities: []string{"vision"}, Priority: 9},
		{ID: "media", RequiredCapabilities: []string{"coding", "tools"}, Priority: 5},
	}
	ordem, err := router.RouteBatch(context.Background(), lote)
	if err != nil {
		t.Fatalf("RouteBatch: %v", err)
	}
	if len(ordem) != len(lote) {
		t.Fatalf("RouteBatch devolveu %d decisoes para %d subtarefas", len(ordem), len(lote))
	}
	for indice, esperado := range []string{"alta", "media", "baixa"} {
		if ordem[indice].SubtaskID != esperado {
			t.Fatalf("ordem por prioridade errada na posicao %d: veio %q, esperado %q", indice, ordem[indice].SubtaskID, esperado)
		}
	}

	// Contexto cancelado: devolve o que ja' decidiu + o erro do contexto (nao finge lote completo).
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	parcial, err := router.RouteBatch(cancelado, lote)
	if err == nil {
		t.Fatalf("lote com contexto cancelado devolveu %d decisoes e NENHUM erro", len(parcial))
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("o erro do lote cancelado tem de ser context.Canceled, veio %v", err)
	}
}

// (8) ROUBO DE PIN QUE NAO PODE SER HONRADO: falha (a capacidade tem de ser respeitada mesmo com pin).
func TestAdaptiveRouterPinNaoAtropelaCapacidade(t *testing.T) {
	router := NewAdaptiveRouterParaRuntime(runtimeDeRoteamento(t), DefaultModelCapabilityRegistry())
	if _, err := router.RouteSubtask(context.Background(), Subtask{
		ID:                   "x",
		RequiredCapabilities: []string{"vision"},
		PinnedModelID:        "local/barato",
	}); err == nil {
		t.Fatal("ARMADILHA: pin sem a capacidade exigida passou — *um pin que nao pode ser honrado tem de falhar*")
	}
}

// (9) DAG: a DEPENDENCIA vence a PRIORIDADE (o comentario da interface prometia isso e a implementacao nao fazia).
func TestRouteBatchRespeitaDependenciaAcimaDaPrioridade(t *testing.T) {
	router := NewAdaptiveRouterParaRuntime(runtimeDeRoteamento(t), DefaultModelCapabilityRegistry())
	lote := []Subtask{
		{ID: "depois", RequiredCapabilities: []string{"coding"}, Priority: 9, Dependencies: []string{"antes"}},
		{ID: "antes", RequiredCapabilities: []string{"coding"}, Priority: 1},
	}
	ordem, err := router.RouteBatch(context.Background(), lote)
	if err != nil {
		t.Fatalf("RouteBatch: %v", err)
	}
	if len(ordem) != 2 {
		t.Fatalf("esperadas 2 decisoes, vieram %d", len(ordem))
	}
	if ordem[0].SubtaskID != "antes" {
		t.Fatalf("DEFEITO: a subtarefa %q foi decidida primeiro — *a dependencia tem de vencer a prioridade*",
			ordem[0].SubtaskID)
	}
}

// (10) DAG invalido: ciclo, dependencia fantasma, ID repetido e ID vazio sao RECUSADOS (nao roteados "na ordem").
func TestRouteBatchRecusaGrafoInvalido(t *testing.T) {
	router := NewAdaptiveRouterParaRuntime(runtimeDeRoteamento(t), DefaultModelCapabilityRegistry())
	ctx := context.Background()
	casos := map[string][]Subtask{
		"ciclo": {
			{ID: "a", RequiredCapabilities: []string{"coding"}, Dependencies: []string{"b"}},
			{ID: "b", RequiredCapabilities: []string{"coding"}, Dependencies: []string{"a"}},
		},
		"auto dependencia": {
			{ID: "a", RequiredCapabilities: []string{"coding"}, Dependencies: []string{"a"}},
		},
		"dependencia fantasma": {
			{ID: "a", RequiredCapabilities: []string{"coding"}, Dependencies: []string{"nao-existe"}},
		},
		"ID repetido": {
			{ID: "a", RequiredCapabilities: []string{"coding"}},
			{ID: "a", RequiredCapabilities: []string{"coding"}},
		},
		"ID vazio": {
			{ID: "   ", RequiredCapabilities: []string{"coding"}},
		},
	}
	for nome, lote := range casos {
		if _, err := router.RouteBatch(ctx, lote); !errors.Is(err, ErrPlanGraphInvalid) {
			t.Fatalf("caso %q: esperado ErrPlanGraphInvalid, veio %v", nome, err)
		}
	}
}

// (11) DECOMPOSITOR: projeta o plano (nao inventa um segundo) e COPIA as fatias.
func TestDecomporPlanoEmSubtarefasProjetaOPlanoValidado(t *testing.T) {
	plano := []Step{
		{ID: "pesquisa", Title: "levantar fontes", RequiredCapabilities: []string{"research"}},
		{ID: "codigo", Title: "implementar", RequiredCapabilities: []string{"coding"}, PinnedModelID: "local/forte", DependsOn: []string{"pesquisa"}},
	}
	if err := ValidatePlanGraph(plano); err != nil {
		t.Fatalf("o plano do teste tinha de ser valido para o validador EXISTENTE: %v", err)
	}
	subtarefas := DecomporPlanoEmSubtarefas(plano)
	if len(subtarefas) != 2 {
		t.Fatalf("esperadas 2 subtarefas, vieram %d", len(subtarefas))
	}
	if subtarefas[0].ID != "pesquisa" || subtarefas[1].ID != "codigo" {
		t.Fatalf("a projecao trocou a ordem/identidade: %q, %q", subtarefas[0].ID, subtarefas[1].ID)
	}
	if len(subtarefas[1].Dependencies) != 1 || subtarefas[1].Dependencies[0] != "pesquisa" {
		t.Fatalf("a ARESTA nao foi projetada: %+v", subtarefas[1].Dependencies)
	}
	if subtarefas[1].PinnedModelID != "local/forte" {
		t.Fatalf("o pin nao foi projetado: %q", subtarefas[1].PinnedModelID)
	}
	if len(subtarefas[1].RequiredCapabilities) != 1 || subtarefas[1].RequiredCapabilities[0] != "coding" {
		t.Fatalf("as capacidades nao foram projetadas: %+v", subtarefas[1].RequiredCapabilities)
	}
	if subtarefas[0].Priority <= subtarefas[1].Priority {
		t.Fatalf("o primeiro passo do plano tem de ter prioridade MAIOR (desempate): %d vs %d",
			subtarefas[0].Priority, subtarefas[1].Priority)
	}
	// PROVA DA COPIA: mutar a subtarefa nao pode tocar o PLANO.
	subtarefas[1].RequiredCapabilities[0] = "mutado"
	if plano[1].RequiredCapabilities[0] != "coding" {
		t.Fatal("a decomposicao compartilhou a fatia: mutar a subtarefa alterou o PLANO")
	}
}

// (12) PONTA A PONTA: plano validado -> subtarefas -> decisoes NA ORDEM DO DAG.
func TestPlanoValidadoViraRoteamentoNaOrdemDoDAG(t *testing.T) {
	router := NewAdaptiveRouterParaRuntime(runtimeDeRoteamento(t), DefaultModelCapabilityRegistry())
	plano := []Step{
		{ID: "a", Title: "primeiro", RequiredCapabilities: []string{"coding"}},
		{ID: "b", Title: "segundo", RequiredCapabilities: []string{"coding"}, DependsOn: []string{"a"}},
		{ID: "c", Title: "terceiro", RequiredCapabilities: []string{"coding"}, DependsOn: []string{"b"}},
	}
	if err := ValidatePlanGraph(plano); err != nil {
		t.Fatalf("plano do teste invalido: %v", err)
	}
	decisoes, err := router.RouteBatch(context.Background(), DecomporPlanoEmSubtarefas(plano))
	if err != nil {
		t.Fatalf("RouteBatch sobre o plano projetado: %v", err)
	}
	if len(decisoes) != 3 {
		t.Fatalf("esperadas 3 decisoes, vieram %d", len(decisoes))
	}
	for indice, esperado := range []string{"a", "b", "c"} {
		if decisoes[indice].SubtaskID != esperado {
			t.Fatalf("posicao %d: esperado %q, veio %q", indice, esperado, decisoes[indice].SubtaskID)
		}
		if decisoes[indice].ChosenModel == "" {
			t.Fatalf("a decisao da subtarefa %q veio SEM modelo escolhido", esperado)
		}
	}
}
