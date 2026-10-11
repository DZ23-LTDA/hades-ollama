package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Guarda de regressão da FAMÍLIA de defeito: a escrita atômica JSON precisa CRIAR o diretório.
//
// Dois stores (força de vendas e marketplace) falharam na PRIMEIRA gravação com "caminho não
// encontrado" porque `writeJSONAtomic` exigia um diretório que ninguém criava — o defeito se
// repetiu duas vezes antes de ser corrigido na raiz. Esta guarda impede a terceira repetição:
// uma gravação em caminho aninhado inexistente precisa funcionar e produzir o arquivo.
func TestWriteJSONAtomicCreatesParentDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "a", "b", "c", "dados.json")
	if err := writeJSONAtomic(target, map[string]string{"campo": "valor"}); err != nil {
		t.Fatalf("a escrita atômica precisa criar o diretório: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("o arquivo precisa existir depois da escrita: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("o arquivo gravado não pode estar vazio")
	}
	// O conteúdo precisa ser JSON legível de volta.
	var conteudo map[string]string
	if err := readJSON(target, &conteudo); err != nil {
		t.Fatalf("o conteúdo precisa ser JSON válido: %v", err)
	}
	if conteudo["campo"] != "valor" {
		t.Fatalf("conteúdo incorreto: %+v", conteudo)
	}
	// A permissão restrita só é verificável onde o sistema tem semântica POSIX: no Windows o
	// `Chmod` de arquivo reporta 0666 independentemente do que foi pedido, e afirmar isso ali
	// mediria o sistema operacional, não a escrita.
	if runtime.GOOS != "windows" {
		if modo := info.Mode().Perm(); modo&0o077 != 0 {
			t.Fatalf("o arquivo não pode ser legível por outros: %o", modo)
		}
	}
}
