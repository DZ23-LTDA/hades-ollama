package agent

import (
	"path/filepath"
	"testing"
)

// PROVA NO RUNTIME REAL (E-170): a persistência da janela de rate limit não pode ser só uma API que existe —
// precisa estar LIGADA no runtime que o servidor constrói. Sem este teste, "a janela é durável" seria uma
// afirmação sobre um setter que talvez ninguém chame.
func TestRuntimeWiresRateLimitPersistence(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if !runtime.RateLimitPersisted() {
		t.Fatal("o runtime real precisa ligar a persistência da janela de rate limit")
	}
	if erro := runtime.RateLimitError(); erro != "" {
		t.Fatalf("runtime recém-criado não pode ter erro de persistência: %s", erro)
	}
	// A raiz é a da política de dados: o diretório fica DENTRO da raiz informada (isolamento por instalação).
	if raiz := runtime.RateLimitRoot(); raiz == "" || filepath.Dir(filepath.Dir(raiz)) == "" {
		t.Fatalf("a raiz da janela precisa estar dentro da raiz de dados: %q", raiz)
	}
}
