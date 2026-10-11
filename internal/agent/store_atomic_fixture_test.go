package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// unwritableStoreRoot devolve uma raiz de store onde a PERSISTÊNCIA FALHA de forma portátil.
//
// Antes, os testes de rollback apontavam a raiz para um diretório inexistente. Quando a escrita
// atômica passou a CRIAR o diretório (correção de raiz da família de defeitos do primeiro gravação),
// essa injeção deixou de falhar — e os testes de rollback paravam de exercitar o rollback sem
// avisar. Aqui um ARQUIVO ocupa o lugar onde precisaria existir um diretório: `MkdirAll` falha com
// ENOTDIR em Linux, macOS e Windows, mantendo a verificação de rollback significativa.
func unwritableStoreRoot(t *testing.T, root, leaf string) string {
	t.Helper()
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("arquivo, não diretório"), 0o600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(blocker, leaf)
}
