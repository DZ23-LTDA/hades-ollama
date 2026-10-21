package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// I1b: o caminho do artefacto vem do MANIFESTO, mas mesmo assim e' verificado. Estes casos
// existem porque um manifesto forjado/corrompido nao pode virar escrita arbitraria.
func TestCaminhoDoArtefactoNoWorkspace(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "ws")
	if err := os.MkdirAll(raiz, 0o755); err != nil {
		t.Fatalf("criar raiz: %v", err)
	}

	// POSITIVO: caminho relativo normal fica dentro da raiz.
	bom, err := caminhoDoArtefactoNoWorkspace(raiz, filepath.Join("site", "index.html"))
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if !filepath.IsAbs(bom) || filepath.Dir(filepath.Dir(bom)) != filepath.Clean(raiz) {
		t.Fatalf("resolucao inesperada: %q (raiz %q)", bom, raiz)
	}

	// NEGATIVO: travessia com `..` nao escapa.
	if _, err := caminhoDoArtefactoNoWorkspace(raiz, filepath.Join("..", "fora.txt")); !errors.Is(err, ErrCaminhoForaDoWorkspace) {
		t.Fatalf("travessia devia ser recusada, veio %v", err)
	}
	if _, err := caminhoDoArtefactoNoWorkspace(raiz, "site/../../fora.txt"); !errors.Is(err, ErrCaminhoForaDoWorkspace) {
		t.Fatalf("travessia aninhada devia ser recusada, veio %v", err)
	}

	// NEGATIVO: caminho absoluto e' recusado (nao se cola ao join).
	// Absoluto de verdade: derivado do directorio actual, para valer nos dois sistemas.
	absoluto, errAbs := filepath.Abs(filepath.Join("..", "fora.txt"))
	if errAbs != nil {
		t.Fatalf("preparar caminho absoluto: %v", errAbs)
	}
	if _, err := caminhoDoArtefactoNoWorkspace(raiz, absoluto); !errors.Is(err, ErrCaminhoForaDoWorkspace) {
		t.Fatalf("caminho absoluto devia ser recusado, veio %v", err)
	}
	// Enraizado sem drive (o caso que falhou no Windows: \\etc\\passwd nao e IsAbs).
	enraizado := string(filepath.Separator) + "etc" + string(filepath.Separator) + "passwd"
	if _, err := caminhoDoArtefactoNoWorkspace(raiz, enraizado); !errors.Is(err, ErrCaminhoForaDoWorkspace) {
		t.Fatalf("caminho absoluto devia ser recusado, veio %v", err)
	}

	// NEGATIVO: irmao que apenas PARECE dentro ("ws-x" com raiz "ws").
	irmao := filepath.Clean(raiz) + "-x/ficheiro.txt"
	if _, err := caminhoDoArtefactoNoWorkspace(raiz, "../../"+filepath.Base(raiz)+"-x/ficheiro.txt"); !errors.Is(err, ErrCaminhoForaDoWorkspace) {
		t.Fatalf("irmao com prefixo parecido devia ser recusado (%s), veio %v", irmao, err)
	}

	// NEGATIVO: vazio, raiz invalida, e o proprio directorio.
	if _, err := caminhoDoArtefactoNoWorkspace(raiz, ""); !errors.Is(err, ErrCaminhoForaDoWorkspace) {
		t.Fatalf("caminho vazio devia ser recusado, veio %v", err)
	}
	if _, err := caminhoDoArtefactoNoWorkspace("", "index.html"); !errors.Is(err, ErrCaminhoForaDoWorkspace) {
		t.Fatalf("raiz vazia devia ser recusada, veio %v", err)
	}
	if _, err := caminhoDoArtefactoNoWorkspace(raiz, "."); !errors.Is(err, ErrCaminhoForaDoWorkspace) {
		t.Fatalf("o proprio directorio devia ser recusado, veio %v", err)
	}
}
