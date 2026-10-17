package agent

import (
	"path/filepath"
	"testing"
)

// Testes do acessor `Root` (E-577): é o que permite ao servidor colocar o histórico bruto
// DENTRO do escopo já governado, em vez de inventar um segundo regime de permissões.

func TestRootDevolveARaizUsadaNaConstrucao(t *testing.T) {
	raiz := t.TempDir()
	store, err := NewContextStore(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if obtido := store.Root(); obtido != raiz {
		t.Fatalf("Root() = %q, esperado %q", obtido, raiz)
	}
	// E o histórico pode ser derivado dela sem adivinhar constantes.
	esperado := filepath.Join(raiz, "historico", "org-a", "proj-1.jsonl")
	if filepath.Dir(filepath.Dir(esperado)) != filepath.Join(raiz, "historico") {
		t.Fatal("a derivação do caminho do histórico não parte da raiz")
	}
}

func TestRootVazioQuandoOStoreEhSoEmMemoria(t *testing.T) {
	// `NewContextStore("")` é um store em memória: NÃO há raiz, e devolver um caminho
	// inventado seria pior do que devolver "". Quem for gravar tem de decidir o que faz
	// com a ausência — e a compactação (E-573) já recusa gravar sem histórico.
	store, err := NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	if obtido := store.Root(); obtido != "" {
		t.Fatalf("Root() de um store em memória = %q, esperado vazio", obtido)
	}
}

func TestRootNaoExplodeComStoreNulo(t *testing.T) {
	// Um acessor que entra em pânico com recetor nulo é uma armadilha para quem o chama
	// em caminhos de limpeza ou de erro — onde o store já pode estar desmontado.
	var nulo *ContextStore
	if obtido := nulo.Root(); obtido != "" {
		t.Fatalf("Root() nulo = %q, esperado vazio", obtido)
	}
}
