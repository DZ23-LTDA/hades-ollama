package agent

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Testes da extração de bundle (correção do defeito B-13, E-184). Extrair conteúdo de terceiro é operação de risco:
// cada proteção tem um caso que tenta furá-la.

func zipDeArquivos(t *testing.T, entradas map[string]string) string {
	t.Helper()
	return zipComEntradas(t, entradas, nil)
}

func zipComEntradas(t *testing.T, arquivos map[string]string, extra func(*zip.Writer) error) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "bundle.zip")
	destino, err := os.Create(caminho)
	if err != nil {
		t.Fatal(err)
	}
	escritor := zip.NewWriter(destino)
	for nome, conteudo := range arquivos {
		entrada, err := escritor.Create(nome)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entrada.Write([]byte(conteudo)); err != nil {
			t.Fatal(err)
		}
	}
	if extra != nil {
		if err := extra(escritor); err != nil {
			t.Fatal(err)
		}
	}
	if err := escritor.Close(); err != nil {
		t.Fatal(err)
	}
	if err := destino.Close(); err != nil {
		t.Fatal(err)
	}
	return caminho
}

func TestPreparePluginWorkspaceExtractsBundleAndCleansUp(t *testing.T) {
	bundle := zipDeArquivos(t, map[string]string{
		"main.js":         "console.log(require('../lib/ajudante.js'));",
		"lib/ajudante.js": "module.exports = 'ajudante presente';",
		"LICENSE.txt":     "licença do terceiro",
	})
	workspace, cleanup, err := preparePluginWorkspace(bundle, t.TempDir())
	if err != nil {
		t.Fatalf("extração precisa funcionar: %v", err)
	}
	// O layout do bundle é preservado - é isso que deixa o irmão alcançável a partir do código em execução.
	for _, relativo := range []string{"main.js", "lib/ajudante.js", "LICENSE.txt"} {
		if _, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(relativo))); err != nil {
			t.Fatalf("o arquivo extraído %q precisa existir: %v", relativo, err)
		}
	}
	conteudo, err := os.ReadFile(filepath.Join(workspace, "lib", "ajudante.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(conteudo), "ajudante presente") {
		t.Fatalf("o conteúdo extraído precisa ser o do bundle: %q", conteudo)
	}
	// E a limpeza remove o diretório inteiro.
	cleanup()
	if _, err := os.Stat(workspace); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a limpeza precisa remover o diretório de execução: %v", err)
	}
}

func TestPreparePluginWorkspaceRefusesPathTraversal(t *testing.T) {
	pai := t.TempDir()
	bundle := zipDeArquivos(t, map[string]string{
		"main.js":      "console.log('ok');",
		"../escapa.js": "conteúdo hostil",
	})
	_, _, err := preparePluginWorkspace(bundle, pai)
	if !errors.Is(err, ErrPluginExecutionBundleUnsafe) {
		t.Fatalf("travessia de caminho precisa ser recusada, got %v", err)
	}
	// Nada pode ter sido escrito FORA: nem no pai, nem um nível acima dele.
	for _, alvo := range []string{filepath.Join(pai, "escapa.js"), filepath.Join(filepath.Dir(pai), "escapa.js")} {
		if _, statErr := os.Stat(alvo); statErr == nil {
			t.Fatalf("a extração não pode criar %q", alvo)
		}
	}
	if _, statErr := os.Stat(filepath.Join(pai, "plugins", "exec-run")); statErr == nil {
		entradas, _ := os.ReadDir(filepath.Join(pai, "plugins", "exec-run"))
		if len(entradas) != 0 {
			t.Fatalf("bundle recusado não pode deixar diretório de execução para trás: %v", entradas)
		}
	}
}

func TestPreparePluginWorkspaceRefusesAbsolutePathAndSymlink(t *testing.T) {
	casos := map[string]struct {
		arquivos map[string]string
		extra    func(*zip.Writer) error
	}{
		"caminho absoluto": {arquivos: map[string]string{"/etc/hostil.js": "x"}},
		"barra invertida":  {arquivos: map[string]string{"..\\escapa.js": "x"}},
		"link simbólico": {
			arquivos: map[string]string{"main.js": "console.log('ok');"},
			extra: func(escritor *zip.Writer) error {
				cabecalho := &zip.FileHeader{Name: "atalho.js", Method: zip.Deflate}
				cabecalho.SetMode(os.ModeSymlink | 0o777)
				entrada, err := escritor.CreateHeader(cabecalho)
				if err != nil {
					return err
				}
				_, err = entrada.Write([]byte("/etc/passwd"))
				return err
			},
		},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			bundle := zipComEntradas(t, caso.arquivos, caso.extra)
			if _, _, err := preparePluginWorkspace(bundle, t.TempDir()); !errors.Is(err, ErrPluginExecutionBundleUnsafe) {
				t.Fatalf("entrada hostil (%s) precisa ser recusada, got %v", nome, err)
			}
		})
	}
}

func TestPreparePluginWorkspaceRefusesTooManyFiles(t *testing.T) {
	arquivos := make(map[string]string, pluginExecutionMaxBundleFiles+1)
	for i := 0; i <= pluginExecutionMaxBundleFiles; i++ {
		arquivos[filepath.ToSlash(filepath.Join("muitos", "arquivo-"+strconv.Itoa(i)+".js"))] = "x"
	}
	bundle := zipDeArquivos(t, arquivos)
	if _, _, err := preparePluginWorkspace(bundle, t.TempDir()); !errors.Is(err, ErrPluginExecutionBundleUnsafe) {
		t.Fatalf("bundle com arquivos demais precisa ser recusado, got %v", err)
	}
}

// O teto por arquivo é conferido pelo TAMANHO DECLARADO, sem descomprimir: é o que evita que um zip que MENTE no
// cabeçalho encha o disco.
func TestPreparePluginWorkspaceRefusesOversizedEntry(t *testing.T) {
	bundle := zipDeArquivos(t, map[string]string{
		"grande.bin": strings.Repeat("a", pluginExecutionMaxBundleFileBytes+1),
	})
	if _, _, err := preparePluginWorkspace(bundle, t.TempDir()); !errors.Is(err, ErrPluginExecutionBundleUnsafe) {
		t.Fatalf("arquivo acima do teto precisa ser recusado, got %v", err)
	}
}
