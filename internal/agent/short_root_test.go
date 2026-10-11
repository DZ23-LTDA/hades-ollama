package agent

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// shortTestRoot cria uma raiz de teste CURTA, sem o nome do teste.
//
// POR QUE: no Windows o limite clássico de caminho (MAX_PATH = 260) é atingido não só pelo
// nosso diretório, mas pelo caminho INTERNO do git
// (`<raiz>/.agent-imports/<id>/origin/.git/objects/xx/<38 caracteres>`). O `t.TempDir()` do
// Go acrescenta o NOME DO TESTE (dezenas de caracteres) ao temporário, e com o repositório
// fundo isso estoura o limite e o `git` falha com `Filename too long` — o teste então
// reprovava por COMPRIMENTO DE CAMINHO em vez de pelo que ele verifica. A raiz curta devolve
// esse orçamento ao teste.
//
// O produto continua reportando o caso de caminho longo com erro TIPADO e acionável
// (`ErrImportPathTooLong`), porque para o usuário com workspace longo esse é um limite real.
func shortTestRoot(t *testing.T) string {
	t.Helper()
	token := make([]byte, 3)
	if _, err := rand.Read(token); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(os.TempDir(), "t"+hex.EncodeToString(token))
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Skipf("sem raiz de teste curta (TMP com %d caracteres): %v", len(os.TempDir()), err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}
