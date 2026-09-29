package agent

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Hardening de large-files/zip-slip do Builder (§8): alem do caso "../secret"
// ja coberto, garante rejeicao de caminho absoluto, traversal aninhado, arquivo
// acima do limite e contagem acima do limite — e que nada escapa do projeto.
func TestBuilderRejectsPathAndSizeAndCountAbuse(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "builder")
	service, err := NewBuilderService(root)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	cases := []struct {
		name  string
		files map[string]string
	}{
		{"absolute-path", map[string]string{"/etc/pwned": "x", "index.html": "ok"}},
		{"nested-traversal", map[string]string{"docs/../../pwned": "x", "index.html": "ok"}},
		{"oversized-file", map[string]string{"index.html": strings.Repeat("a", (2<<20)+1)}},
	}
	for _, c := range cases {
		if _, err := service.Create(ctx, BuilderSpec{Name: c.name, Kind: BuilderWebsite, Files: c.files}); err == nil {
			t.Fatalf("%s: abuse aceito", c.name)
		}
	}

	// Contagem: mais de 200 arquivos deve ser rejeitado.
	many := make(map[string]string, 250)
	for i := range 250 {
		many["file_"+strconv.Itoa(i)+".txt"] = "x"
	}
	if _, err := service.Create(ctx, BuilderSpec{Name: "too-many", Kind: BuilderWebsite, Files: many}); err == nil {
		t.Fatal("mais de 200 arquivos aceito")
	}

	// Nada pode ter escapado para fora do root do builder.
	if _, err := os.Stat(filepath.Join(base, "pwned")); err == nil {
		t.Fatal("arquivo escapou para o diretorio-pai (zip-slip)")
	}
	if _, err := os.Stat("/etc/pwned"); err == nil {
		t.Fatal("arquivo escapou para caminho absoluto")
	}
}
