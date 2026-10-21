package agent

import (
	"errors"
	"testing"
)

// FATIA 1: o caminho publicado e' construido a partir de identificadores, NUNCA de caminho vindo
// do cliente. Estes casos fixam isso.
func TestNomePublicado(t *testing.T) {
	// POSITIVO
	for _, bom := range []string{"index.html", "app.css", "logo.svg", "dados_v2.json"} {
		obtido, err := nomePublicado(bom)
		if err != nil || obtido != bom {
			t.Fatalf("nomePublicado(%q) = %q, %v", bom, obtido, err)
		}
	}
	// NEGATIVO: caminho com directorio NAO e' remapeado para a base -- e' recusado, para nao
	// esconder uma tentativa de travessia.
	for _, mau := range []string{"", "..", ".", "a/b", "a\\b", "site/index.html", "CON", "nul.txt", "nome com espaco.html"} {
		if _, err := nomePublicado(mau); !errors.Is(err, ErrArtefactoPublicacaoInvalida) {
			t.Fatalf("nomePublicado(%q) devia recusar, veio %v", mau, err)
		}
	}
}

func TestChavePublicacao(t *testing.T) {
	// POSITIVO: a chave inclui a VERSAO, por isso publicar de novo nao apaga a anterior.
	a, err := chavePublicacao("art1", 3)
	if err != nil || a != "art1-v3" {
		t.Fatalf("chave = %q, %v", a, err)
	}
	b, _ := chavePublicacao("art1", 4)
	if a == b {
		t.Fatal("versoes diferentes tem de dar chaves diferentes")
	}
	// NEGATIVO: id com separador ou vazio, e versao negativa.
	for _, mau := range []string{"", "../x", "a/b"} {
		if _, err := chavePublicacao(mau, 1); !errors.Is(err, ErrArtefactoPublicacaoInvalida) {
			t.Fatalf("chavePublicacao(%q) devia recusar, veio %v", mau, err)
		}
	}
	if _, err := chavePublicacao("art1", -1); !errors.Is(err, ErrArtefactoPublicacaoInvalida) {
		t.Fatalf("versao negativa devia ser recusada, veio %v", err)
	}
}
