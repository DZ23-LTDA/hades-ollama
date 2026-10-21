//go:build windows || darwin

package ui

import (
	"path/filepath"
	"testing"
)

// DEFEITO MEDIDO pelo dono: "Nao foi possivel cadastrar o provedor. Detalhe: OLLAMA_DZ23_CONFIG
// nao esta definido; instale novamente o aplicativo ou defina essa variavel".
//
// O caminho e' DERIVAVEL (o instalador define "{userappdata}\Ollama DZ23\dz23-providers.json"),
// por isso a falta da variavel nao pode impedir o cadastro.
func TestProviderConfigPathUsaCaminhoPorOmissaoSemEnv(t *testing.T) {
	t.Setenv("OLLAMA_DZ23_CONFIG", "")

	obtido := providerConfigPath()
	if obtido == "" {
		t.Fatal("sem OLLAMA_DZ23_CONFIG devia usar o caminho por omissao, veio vazio")
	}
	if filepath.Base(obtido) != "dz23-providers.json" {
		t.Fatalf("nome inesperado: %q", filepath.Base(obtido))
	}
	if filepath.Base(filepath.Dir(obtido)) != "Ollama DZ23" {
		t.Fatalf("directorio inesperado: %q", filepath.Dir(obtido))
	}
	if !filepath.IsAbs(obtido) {
		t.Fatalf("caminho devia ser absoluto: %q", obtido)
	}
}

// A variavel continua a mandar (override), para quem aponta o ficheiro para outro sitio.
func TestProviderConfigPathRespeitaEnvQuandoDefinido(t *testing.T) {
	alvo := filepath.Join(t.TempDir(), "custom.json")
	t.Setenv("OLLAMA_DZ23_CONFIG", alvo)

	if obtido := providerConfigPath(); obtido != alvo {
		t.Fatalf("devia respeitar o override: veio %q, queria %q", obtido, alvo)
	}
}
