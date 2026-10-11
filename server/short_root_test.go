package server

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// shortServerRoot cria uma raiz de teste CURTA, sem o nome do teste.
//
// MOTIVO (o mesmo do pacote agent): no Windows o limite clássico de caminho (MAX_PATH = 260)
// é atingido pelo caminho INTERNO do git
// (`<raiz>/.agent-imports/<id>/origin/.git/objects/xx/<38 caracteres>`). Como o `t.TempDir()`
// acrescenta o nome do teste, o teste reprovava por comprimento de caminho em vez do que
// verifica. O produto continua reportando esse limite com erro TIPADO e acionável.
func shortServerRoot(t *testing.T) string {
	t.Helper()
	token := make([]byte, 3)
	if _, err := rand.Read(token); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(os.TempDir(), "s"+hex.EncodeToString(token))
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Skipf("sem raiz de teste curta (TMP com %d caracteres): %v", len(os.TempDir()), err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}
