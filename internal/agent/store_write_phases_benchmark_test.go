package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// DIAGNÓSTICO DA ESCRITA (E-166): depois de E-165 eliminar duas hipóteses SEM encontrar o ganho, a pergunta
// ficou registrada como "o custo interno (`Sync`, `Rename`, `MkdirAll`, serialização) NÃO foi isolado".
//
// Estes benchmarks isolam cada FASE da escrita atômica. Eles são uma RÉPLICA da sequência real de
// `writeJSONAtomicBounded` (mesmo tamanho de carga, mesmas chamadas, na mesma ordem) — não a própria função,
// porque desligar o `Sync` exigiria mudar produção só para medir. A soma das fases é comparada com o benchmark
// da função REAL (`BenchmarkMemoryFileWrite`), que serve de controle: se a soma não fechar, isso é declarado.
//
// O objetivo é dizer ONDE o tempo está, para que a próxima decisão seja sobre a fase certa.

const fasesMemorias = 200

// cargaDeMemorias monta a mesma carga do benchmark real (200 memórias).
func cargaDeMemorias(projectID string) []Memory {
	memorias := make([]Memory, 0, fasesMemorias)
	for i := range fasesMemorias {
		memorias = append(memorias, Memory{
			ID: fmt.Sprintf("mem_%d", i), ProjectID: projectID, Kind: "fact",
			Content: fmt.Sprintf("memória %d", i), Source: "manual",
		})
	}
	return memorias
}

// serializarFase faz exatamente o que a função real faz para produzir os bytes.
func serializarFase(b *testing.B, memorias []Memory) []byte {
	b.Helper()
	dados, err := json.MarshalIndent(memorias, "", "  ")
	if err != nil {
		b.Fatal(err)
	}
	return append(dados, '\n')
}

func BenchmarkWritePhase_Marshal(b *testing.B) {
	memorias := cargaDeMemorias("prj_fase")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = serializarFase(b, memorias)
	}
}

func BenchmarkWritePhase_MkdirAll(b *testing.B) {
	dir := b.TempDir()
	alvo := filepath.Join(dir, "sub", "arquivo.json")
	if err := os.MkdirAll(filepath.Dir(alvo), 0o700); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := os.MkdirAll(filepath.Dir(alvo), 0o700); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWritePhase_CreateTemp mede só a criação do temporário (com o Chmod e a limpeza), sem conteúdo.
func BenchmarkWritePhase_CreateTemp(b *testing.B) {
	dir := b.TempDir()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		tmp, err := os.CreateTemp(dir, ".agent-*.tmp")
		if err != nil {
			b.Fatal(err)
		}
		nome := tmp.Name()
		if err := tmp.Chmod(0o600); err != nil {
			b.Fatal(err)
		}
		if err := tmp.Close(); err != nil {
			b.Fatal(err)
		}
		if err := os.Remove(nome); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWritePhase_Sync mede a fase de DURABILIDADE: criar, escrever e `Sync`.
func BenchmarkWritePhase_Sync(b *testing.B) {
	dir := b.TempDir()
	memorias := cargaDeMemorias("prj_fase")
	dados := serializarFase(b, memorias)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		tmp, err := os.CreateTemp(dir, ".agent-*.tmp")
		if err != nil {
			b.Fatal(err)
		}
		nome := tmp.Name()
		if err := tmp.Chmod(0o600); err != nil {
			b.Fatal(err)
		}
		if _, err := tmp.Write(dados); err != nil {
			b.Fatal(err)
		}
		if err := tmp.Sync(); err != nil {
			b.Fatal(err)
		}
		if err := tmp.Close(); err != nil {
			b.Fatal(err)
		}
		if err := os.Remove(nome); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWritePhase_Rename mede a fase de TROCA ATÔMICA sem o `Sync`.
func BenchmarkWritePhase_Rename(b *testing.B) {
	dir := b.TempDir()
	alvo := filepath.Join(dir, "destino.json")
	memorias := cargaDeMemorias("prj_fase")
	dados := serializarFase(b, memorias)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		tmp, err := os.CreateTemp(dir, ".agent-*.tmp")
		if err != nil {
			b.Fatal(err)
		}
		nome := tmp.Name()
		if err := tmp.Chmod(0o600); err != nil {
			b.Fatal(err)
		}
		if _, err := tmp.Write(dados); err != nil {
			b.Fatal(err)
		}
		if err := tmp.Close(); err != nil {
			b.Fatal(err)
		}
		if err := os.Rename(nome, alvo); err != nil {
			b.Fatal(err)
		}
	}
}
