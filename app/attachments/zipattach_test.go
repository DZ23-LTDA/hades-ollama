package attachments

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var extsTeste = []string{"ts", "tsx", "go", "md", "json"}

// escreverZip cria um arquivo temporario com as entradas dadas (nome -> conteudo).
func escreverZip(t *testing.T, entradas map[string]string) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "proj.zip")
	f, err := os.Create(caminho)
	if err != nil {
		t.Fatalf("criar zip: %v", err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for nome, conteudo := range entradas {
		w, err := zw.Create(nome)
		if err != nil {
			t.Fatalf("criar entrada %s: %v", nome, err)
		}
		if _, err := w.Write([]byte(conteudo)); err != nil {
			t.Fatalf("escrever entrada %s: %v", nome, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("fechar zip: %v", err)
	}
	return caminho
}

// POSITIVO: arquivo valido com ficheiros suportados continua a produzir anexos (sem regressao).
func TestBuildZipAttachmentsPositivo(t *testing.T) {
	caminho := escreverZip(t, map[string]string{
		"src/app.ts":        "export const x = 1;",
		"README.md":         "# projeto",
		"node_modules/a.js": "ruido",
	})

	resultado, err := BuildZipAttachments(caminho, extsTeste, DefaultZipLimits)
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if len(resultado.Attachments) != 2 {
		t.Fatalf("esperava 2 anexos, veio %d", len(resultado.Attachments))
	}
	nomes := []string{resultado.Attachments[0].Name, resultado.Attachments[1].Name}
	prefixo := filepath.Base(caminho) + "/"
	for _, nome := range nomes {
		if !strings.HasPrefix(nome, prefixo) {
			t.Fatalf("nome %q sem prefixo %q", nome, prefixo)
		}
	}
	if len(resultado.Skipped) == 0 {
		t.Fatal("esperava pelo menos uma entrada ignorada (node_modules)")
	}
	if resultado.Reason == "" {
		t.Fatal("esperava Reason a resumir o que foi ignorado")
	}
}

// NEGATIVO: arquivo sem ficheiros suportados tem de devolver ERRO e motivo, nao vazio em silencio.
func TestBuildZipAttachmentsSemTextoDevolveErro(t *testing.T) {
	caminho := escreverZip(t, map[string]string{
		"logo.png":          "\x89PNG",
		"bin/app.exe":       "MZ",
		"node_modules/x.js": "ruido",
	})

	resultado, err := BuildZipAttachments(caminho, extsTeste, DefaultZipLimits)
	if err == nil {
		t.Fatal("DEFEITO: zip sem texto devolveu sucesso silencioso")
	}
	if !errors.Is(err, ErrZipNoTextFiles) {
		t.Fatalf("esperava ErrZipNoTextFiles, veio %v", err)
	}
	if resultado.Reason == "" {
		t.Fatal("esperava Reason com a contagem de ignorados")
	}
	if len(resultado.Attachments) != 0 {
		t.Fatalf("nao devia produzir anexos, veio %d", len(resultado.Attachments))
	}
	if !strings.Contains(resultado.Reason, "ignoradas") {
		t.Fatalf("Reason sem contagem de ignorados: %q", resultado.Reason)
	}
}

// NEGATIVO: ficheiro que nao e' um zip valido tem de devolver erro (nao vazio).
func TestBuildZipAttachmentsCorrompidoDevolveErro(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "quebrado.zip")
	if err := os.WriteFile(caminho, []byte("isto nao e um zip"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	if _, err := BuildZipAttachments(caminho, extsTeste, DefaultZipLimits); err == nil {
		t.Fatal("DEFEITO: zip corrompido devolveu sucesso")
	}
}

// NEGATIVO: limites apertados devolvem ErrZipLimit, que o chamador tem de mostrar ao utilizador.
func TestBuildZipAttachmentsLimiteDeEntradas(t *testing.T) {
	caminho := escreverZip(t, map[string]string{
		"a.ts": "1", "b.ts": "2", "c.ts": "3",
	})
	limites := DefaultZipLimits
	limites.MaxEntries = 1

	_, err := BuildZipAttachments(caminho, extsTeste, limites)
	if err == nil {
		t.Fatal("DEFEITO: limite de entradas nao gerou erro")
	}
	if !errors.Is(err, ErrZipLimit) {
		t.Fatalf("esperava ErrZipLimit, veio %v", err)
	}
}

func TestResumirIgnorados(t *testing.T) {
	if got := resumirIgnorados([]string{"a", "b"}, 5); got != "a, b" {
		t.Fatalf("resumo curto errado: %q", got)
	}
	got := resumirIgnorados([]string{"a", "b", "c"}, 2)
	if !strings.HasPrefix(got, "a, b") || !strings.Contains(got, "(+1)") {
		t.Fatalf("resumo truncado errado: %q", got)
	}
}
