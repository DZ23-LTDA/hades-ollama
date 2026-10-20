package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compactacaoDeTeste(raiz, identidade string, permissoes PermissaoDeMemoria) CompactacaoGovernada {
	return CompactacaoGovernada{Raiz: raiz, Politica: politicaDeTeste(identidade, EscopoTenant, permissoes)}
}

// NEGATIVO: sem permissao de escrita, nada se move.
func TestCompactarSemPermissaoNaoMoveNada(t *testing.T) {
	raiz := t.TempDir()
	gravarNotaDeTeste(t, raiz, "org-1", "notas/a.md")

	c := compactacaoDeTeste(raiz, "user-1", PermissaoLer)
	_, err := c.Compactar(PedidoDeCompactacao{Identidade: "user-1", Tenant: "org-1", Pasta: "notas", ResumoRelativo: "notas/RESUMO.md"})
	if !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("esperava ErrAcessoNegado, veio %v", err)
	}
	if _, err := os.Stat(filepath.Join(raiz, "notas", "a.md")); err != nil {
		t.Fatalf("a nota devia continuar no lugar: %v", err)
	}
}

// NEGATIVO: pasta sem notas do tenant nao inventa compactacao.
func TestCompactarSemNotasDaErro(t *testing.T) {
	raiz := t.TempDir()
	os.MkdirAll(filepath.Join(raiz, "notas"), 0o700)
	c := compactacaoDeTeste(raiz, "user-1", PermissaoLer|PermissaoEscrever)
	if _, err := c.Compactar(PedidoDeCompactacao{Identidade: "user-1", Tenant: "org-1", Pasta: "notas", ResumoRelativo: "notas/RESUMO.md"}); !errors.Is(err, ErrNadaACompactar) {
		t.Fatalf("esperava ErrNadaACompactar, veio %v", err)
	}
}

// POSITIVO: compacta movendo os ORIGINAIS (nada se perde) e o resumo CITA cada fonte com hash.
func TestCompactarMoveOriginaisEscreveResumoComFontes(t *testing.T) {
	raiz := t.TempDir()
	gravarNotaDeTeste(t, raiz, "org-1", "notas/a.md")
	gravarNotaDeTeste(t, raiz, "org-1", "notas/b.md")

	c := compactacaoDeTeste(raiz, "user-1", PermissaoLer|PermissaoEscrever)
	rel, err := c.Compactar(PedidoDeCompactacao{
		Identidade: "user-1", Tenant: "org-1", Pasta: "notas", ResumoRelativo: "notas/RESUMO.md",
		TextoDoResumo: "duas notas de decisao",
	})
	if err != nil {
		t.Fatalf("compactacao devia passar: %v", err)
	}
	if rel.TotalCompactado != 2 {
		t.Fatalf("esperava 2 compactadas, veio %d", rel.TotalCompactado)
	}

	// Os originais deixaram a pasta mas EXISTEM no arquivo, com os mesmos bytes.
	for _, nome := range []string{"a.md", "b.md"} {
		if _, err := os.Stat(filepath.Join(raiz, "notas", nome)); err == nil {
			t.Fatalf("%s devia ter saido da pasta", nome)
		}
		arquivada := filepath.Join(raiz, filepath.FromSlash(filepath.Join(rel.PastaDeArquivo, nome)))
		if _, err := os.Stat(arquivada); err != nil {
			t.Fatalf("%s devia estar no arquivo: %v", nome, err)
		}
	}

	resumo, err := os.ReadFile(filepath.Join(raiz, "notas", "RESUMO.md"))
	if err != nil {
		t.Fatalf("o resumo devia existir: %v", err)
	}
	texto := string(resumo)
	for _, f := range rel.Fontes {
		if !strings.Contains(texto, f.Relativo) {
			t.Fatalf("o resumo devia citar %q", f.Relativo)
		}
		if !strings.Contains(texto, f.ContentHash) {
			t.Fatalf("o resumo devia guardar o hash de %q", f.Relativo)
		}
	}
	if !strings.Contains(texto, "duas notas de decisao") {
		t.Fatal("o texto do resumo perdeu-se")
	}

	// E a nota arquivada continua a ser legivel pelo adaptador (recuperacao possivel).
	leitura := leituraDeTeste(raiz, "user-1", PermissaoLer)
	if _, err := leitura.Ler(PedidoDeLeitura{Identidade: "user-1", Tenant: "org-1", Relativo: filepath.ToSlash(filepath.Join(rel.PastaDeArquivo, "a.md"))}); err != nil {
		t.Fatalf("a nota arquivada devia continuar legivel: %v", err)
	}
}

// NEGATIVO: nota de OUTRO tenant nao e' movida pela compactacao.
func TestCompactarNaoMoveNotaDeOutroTenant(t *testing.T) {
	raiz := t.TempDir()
	gravarNotaDeTeste(t, raiz, "org-DONA", "notas/alheia.md")
	gravarNotaDeTeste(t, raiz, "org-1", "notas/minha.md")

	c := compactacaoDeTeste(raiz, "user-1", PermissaoLer|PermissaoEscrever)
	rel, err := c.Compactar(PedidoDeCompactacao{Identidade: "user-1", Tenant: "org-1", Pasta: "notas", ResumoRelativo: "notas/RESUMO.md"})
	if err != nil {
		t.Fatalf("devia compactar a do tenant: %v", err)
	}
	if rel.TotalCompactado != 1 {
		t.Fatalf("esperava 1 compactada, veio %d", rel.TotalCompactado)
	}
	if _, err := os.Stat(filepath.Join(raiz, "notas", "alheia.md")); err != nil {
		t.Fatalf("a nota de outro tenant devia ficar intacta: %v", err)
	}
}
