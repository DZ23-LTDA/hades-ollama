package agent

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BENCHMARK DO ESTÁGIO 21 — metodologia registrada (docs/mission/HADES_BENCHMARK_21.md).
//
// O que estes benchmarks medem: o comportamento do PRÓPRIO Hades em caminhos determinísticos (turno de
// atendimento com plano por regras, busca literal, busca semântica e re-embed), com dados fixos e sem rede.
//
// O que eles NÃO medem, e é declarado no documento: comparação com sistemas externos (Manus, n8n, Cursor…).
// Rodar o concorrente exige instalar, autenticar e pagar por ele — sem isso, qualquer número comparativo
// seria invenção. O `ExternalComparison` fica NAO_EXECUTADA com o procedimento exato de desbloqueio.
//
// Regra de honestidade: número sem comando reproduzível e sem ambiente declarado não entra no relatório.

// benchmarkAgentRuntime monta o runtime de atendimento COMPLETO — contato, consentimento e conversa reais —
// reutilizando o mesmo fixture dos testes (`serviceAgentRuntime`, que aceita `testing.TB`).
//
// O turno exige conversa existente no inbox: um runtime sem conversa falha com "conversation not found", e o
// benchmark mediria erro em vez de trabalho.
func benchmarkAgentRuntime(b *testing.B) (*ServiceAgentRuntime, string) {
	b.Helper()
	agents, _, _, _, conversationID := serviceAgentRuntime(b, true)
	return agents, conversationID
}

// BenchmarkServiceAgentTurn mede o turno COMPLETO do agente de vendas com ferramenta de leitura: é o caminho
// que um cliente real aciona a cada mensagem.
func BenchmarkServiceAgentTurn(b *testing.B) {
	agents, conversationID := benchmarkAgentRuntime(b)
	now := time.Now().UTC()
	pedido := ServiceAgentTurnRequest{
		OrganizationID: "org-a",
		AgentID:        ServiceAgentSales,
		ConversationID: conversationID,
		Intent:         "quero saber os preços do catálogo",
		RequestedTools: []string{"catalog.read"},
	}
	// Aquece uma vez: o primeiro turno paga custos de primeira escrita (arquivos, índices).
	if _, err := agents.RunTurn(pedido, now); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		if _, err := agents.RunTurn(pedido, now.Add(time.Duration(i)*time.Millisecond)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemorySearchLiteral mede a busca LITERAL no motor de memória com 1.000 memórias.
func BenchmarkMemorySearchLiteral(b *testing.B) {
	store, project := benchmarkMemoryStore(b, 1000, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if matches := store.SearchMemoriesLexical(project.ID, "política de troca", 20); len(matches) == 0 {
			b.Fatal("a busca literal precisa encontrar a memória plantada")
		}
	}
}

// BenchmarkMemorySearchSemantic mede a busca SEMÂNTICA (com embedder) no mesmo volume — o par com o literal
// mostra o custo do caminho vetorial.
func BenchmarkMemorySearchSemantic(b *testing.B) {
	embedder := stubEmbedder{modelo: "bench-embed", vetores: map[string][]float32{"política": {1, 0, 0}}}
	store, project := benchmarkMemoryStore(b, 1000, embedder)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := store.SearchMemoriesContext(ctx, project.ID, "política de troca", 20); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryReembed mede o custo do re-embed de 200 memórias (o trabalho de manutenção do E-158/E-159).
//
// Cada iteração monta uma store NOVA com o embedder desligado e só então o liga: com o embedder já ligado, as
// memórias nasceriam carimbadas e o re-embed não teria o que recalcular — foi exatamente o que a primeira
// versão deste benchmark mediu por engano (`Skipped: 200`). O que se mede é o RECÁLCULO: criação do cenário e
// do projeto fica fora do cronômetro.
func BenchmarkMemoryReembed(b *testing.B) {
	embedder := stubEmbedder{modelo: "bench-embed"}
	ctx := context.Background()
	const total = 200
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		b.StopTimer()
		store, err := NewContextStore(b.TempDir())
		if err != nil {
			b.Fatal(err)
		}
		project, err := store.CreateProject(fmt.Sprintf("projeto-%d", i), b.TempDir(), "org-bench")
		if err != nil {
			b.Fatal(err)
		}
		for j := range total {
			if _, err := store.AddMemory(Memory{ProjectID: project.ID, Kind: "fact", Content: fmt.Sprintf("memória %d", j), Source: "manual"}); err != nil {
				b.Fatal(err)
			}
		}
		store.SetEmbedder(embedder)
		b.StartTimer()
		resultado, err := store.ReembedMemories(ctx, project.ID, 0)
		if err != nil {
			b.Fatal(err)
		}
		if resultado.Reembedded != total {
			b.Fatalf("o re-embed precisa recalcular as %d memórias: %+v", total, resultado)
		}
	}
}

// BenchmarkServiceAgentPlan mede SÓ o planejador: sem ferramenta, sem persistência e sem I/O.
//
// É o CONTROLE do experimento de E-164: se o planejador custa o mesmo nas duas plataformas e o turno completo
// custa ~40% mais no Linux, a diferença está no que o turno faz ALÉM de planejar (ferramenta, trilha, arquivos).
func BenchmarkServiceAgentPlan(b *testing.B) {
	planner := RulePlanner{}
	missao := Mission{ID: "mis_bench", Objective: "quero saber os preços do catálogo"}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := planner.Plan(ctx, missao); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryFileWrite mede a ESCRITA ATÔMICA do arquivo de memórias — o custo de sistema de arquivos que o
// turno e o re-embed pagam (no modo PADRÃO, com `fsync`).
func BenchmarkMemoryFileWrite(b *testing.B) {
	benchmarkMemoryFileWrite(b)
}

// BenchmarkMemoryFileWriteRelaxed mede a MESMA escrita no modo `relaxed` (sem `fsync`): é a evidência de quanto
// custa a garantia de durabilidade — o número que sustenta a política de E-168 sem inventar ganho.
func BenchmarkMemoryFileWriteRelaxed(b *testing.B) {
	anterior := WriteDurability()
	defer func() { _, _ = SetWriteDurability(anterior) }()
	if _, err := SetWriteDurability(WriteDurabilityRelaxed); err != nil {
		b.Fatal(err)
	}
	b.Logf("modo efetivo deste benchmark: %s", WriteDurability())
	benchmarkMemoryFileWrite(b)
}

func benchmarkMemoryFileWrite(b *testing.B) {
	b.Helper()
	store, err := NewContextStore(b.TempDir() + "-write")
	if err != nil {
		b.Fatal(err)
	}
	project, err := store.CreateProject("Base de escrita", b.TempDir(), "org-write")
	if err != nil {
		b.Fatal(err)
	}
	memorias := make([]Memory, 0, 200)
	for i := range 200 {
		memorias = append(memorias, Memory{
			ID: fmt.Sprintf("mem_%d", i), ProjectID: project.ID, Kind: "fact",
			Content: fmt.Sprintf("memória %d", i), Source: "manual",
		})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := store.writeMemoriesFile(project.ID, memorias); err != nil {
			b.Fatal(err)
		}
	}
}

// benchmarkMemoryStore cria um projeto com `total` memórias (uma delas é o alvo da busca).
func benchmarkMemoryStore(b *testing.B, total int, embedder Embedder) (*ContextStore, Project) {
	b.Helper()
	store, err := NewContextStore(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	if embedder != nil {
		store.SetEmbedder(embedder)
	}
	project, err := store.CreateProject("Base de benchmark", b.TempDir(), "org-bench")
	if err != nil {
		b.Fatal(err)
	}
	// A memória ALVO fica no meio: em busca literal, a posição altera o custo de varredura.
	alvo := total / 2
	for i := range total {
		conteudo := fmt.Sprintf("memória irrelevante %d sobre assuntos diversos", i)
		if i == alvo {
			conteudo = "política de troca em até 30 dias para produtos sem uso"
		}
		if _, err := store.AddMemory(Memory{ProjectID: project.ID, Kind: "fact", Content: conteudo, Source: "manual"}); err != nil {
			b.Fatal(err)
		}
	}
	return store, project
}
