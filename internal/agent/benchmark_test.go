package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

// Benchmarks do estágio 21: medem o que o Hades CONTROL (motor próprio), com
// metodologia registrada em docs/mission/HADES_BENCHMARK_METHOD.md.
//
// Limite declarado: isto NÃO compara o Hades com produtos concorrentes. Uma
// comparação honesta exige a mesma tarefa, a mesma máquina e a licença/conta de
// cada concorrente — sem isso, qualquer alegação de superioridade seria
// inventada. Aqui só se mede custo local determinístico, que qualquer um pode
// reproduzir com o comando do documento.

func benchmarkMemories(count int) []Memory {
	memories := make([]Memory, 0, count)
	for index := range count {
		memories = append(memories, Memory{
			ID:         "mem_" + time.Unix(int64(index), 0).UTC().Format("150405"),
			ProjectID:  "proj-bench",
			Kind:       "fact",
			Content:    "memória de benchmark com conteúdo textual suficiente para medir redação e truncamento",
			Source:     "bench",
			Confidence: 0.7,
			Embedding:  []float32{0.1, 0.2, 0.3, 0.4},
			CreatedAt:  time.Now().UTC().Add(-time.Duration(index) * time.Minute),
		})
	}
	return memories
}

func BenchmarkMemorySearchContext(b *testing.B) {
	store, err := NewContextStore(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	for _, memory := range benchmarkMemories(500) {
		if _, err := store.AddMemory(memory); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := store.SearchMemoriesContext(context.Background(), "proj-bench", "benchmark", 20); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginInstallUpdateRollback(b *testing.B) {
	registry, err := NewPluginRegistry(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	policy := DefaultCapabilityPolicy()
	manifest := PluginManifest{
		ID:      "bench-plugin",
		Version: "1.0.0",
		Name:    "Plugin de benchmark",
		Kind:    PluginKindConnector,
		Scopes:  []string{"connector:external"},
	}
	b.ReportAllocs()
	for iteration := range b.N {
		version := "1.0." + time.Unix(int64(iteration), 0).UTC().Format("05")
		current := manifest
		current.Version = version
		if _, err := registry.InstallForOrganization("org-bench", current, policy); err != nil {
			b.Fatal(err)
		}
		next := current
		next.Version = version + ".1"
		if _, err := registry.UpdateForOrganization("org-bench", "bench-plugin", version, next, policy); err != nil {
			b.Fatal(err)
		}
		if _, err := registry.RollbackForOrganization("org-bench", "bench-plugin"); err != nil {
			b.Fatal(err)
		}
		if err := registry.RemoveForOrganization("org-bench", "bench-plugin"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPluginPromotionWithSignature(b *testing.B) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}
	policy := DefaultCapabilityPolicy().WithTrustedSkillKey("bench-key", publicKey)
	signed, err := SignPluginManifest(PluginManifest{
		ID:      "bench-plugin",
		Version: "1.0.0",
		Name:    "Plugin assinado",
		Kind:    PluginKindSkill,
		Scopes:  []string{"workspace:read"},
	}, privateKey, "bench-key")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for iteration := range b.N {
		registry, err := NewPluginRegistry(b.TempDir())
		if err != nil {
			b.Fatal(err)
		}
		manifest := signed
		manifest.Version = "1.0." + time.Unix(int64(iteration), 0).UTC().Format("05")
		resigned, err := SignPluginManifest(manifest, privateKey, "bench-key")
		if err != nil {
			b.Fatal(err)
		}
		if _, err := registry.InstallForOrganization("org-bench", resigned, policy); err != nil {
			b.Fatal(err)
		}
		if _, err := registry.PromoteTrustedForOrganization("org-bench", "bench-plugin", policy); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGroundedContextBuild(b *testing.B) {
	sources := make([]ScoredMemory, 0, 32)
	for index := range 32 {
		sources = append(sources, ScoredMemory{
			Memory: Memory{
				ID:      "mem",
				Kind:    "fact",
				Content: "trecho de documento indexado para montagem de contexto fundamentado",
			},
			Score: 0.9 - float64(index)/100,
		})
	}
	b.ReportAllocs()
	for b.Loop() {
		block, citations := BuildGroundedContext("pergunta de benchmark", sources)
		if block == "" || len(citations) == 0 {
			b.Fatal("grounded context must carry citations")
		}
	}
}
