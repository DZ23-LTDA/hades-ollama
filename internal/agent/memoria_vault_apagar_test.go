package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func remocaoDeTeste(raiz, identidade string, permissoes PermissaoDeMemoria) RemocaoGovernada {
	return RemocaoGovernada{
		Raiz:     raiz,
		Politica: politicaDeTeste(identidade, EscopoTenant, permissoes),
	}
}

// NEGATIVO: sem permissao de escrita, recusa e a nota CONTINUA la'.
func TestApagarSemPermissaoNaoRemove(t *testing.T) {
	raiz := t.TempDir()
	gravarNotaDeTeste(t, raiz, "org-1", "notas/manter.md")

	remocao := remocaoDeTeste(raiz, "user-1", PermissaoLer) // so' leitura
	apagou, err := remocao.Apagar(PedidoDeRemocao{Identidade: "user-1", Tenant: "org-1", Relativo: "notas/manter.md"})
	if !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("esperava ErrAcessoNegado, veio %v", err)
	}
	if apagou {
		t.Fatal("nao podia ter apagado")
	}
	if _, err := os.Stat(filepath.Join(raiz, "notas", "manter.md")); err != nil {
		t.Fatalf("a nota devia continuar: %v", err)
	}
}

// NEGATIVO: nota de outro tenant nao e' apagavel e nao revela existencia.
func TestApagarNotaDeOutroTenantNaoRemove(t *testing.T) {
	raiz := t.TempDir()
	gravarNotaDeTeste(t, raiz, "org-DONA", "notas/alheia.md")

	remocao := remocaoDeTeste(raiz, "user-1", PermissaoLer|PermissaoEscrever)
	_, errOutro := remocao.Apagar(PedidoDeRemocao{Identidade: "user-1", Tenant: "org-1", Relativo: "notas/alheia.md"})
	_, errInexistente := remocao.Apagar(PedidoDeRemocao{Identidade: "user-1", Tenant: "org-1", Relativo: "notas/nao-existe.md"})
	if !errors.Is(errOutro, ErrAcessoNegado) || !errors.Is(errInexistente, ErrAcessoNegado) {
		t.Fatalf("ambos deviam negar: %v / %v", errOutro, errInexistente)
	}
	if errOutro.Error() != errInexistente.Error() {
		t.Fatalf("os dois casos tem de ser indistinguiveis: %q vs %q", errOutro.Error(), errInexistente.Error())
	}
	if _, err := os.Stat(filepath.Join(raiz, "notas", "alheia.md")); err != nil {
		t.Fatalf("a nota alheia devia continuar intacta: %v", err)
	}
}

// POSITIVO: com permissao, apaga, o ficheiro desaparece e fica UMA linha de auditoria SEM o conteudo.
func TestApagarComPermissaoRemoveERegistaSemConteudo(t *testing.T) {
	raiz := t.TempDir()
	gravarNotaDeTeste(t, raiz, "org-1", "notas/apagar.md")

	remocao := remocaoDeTeste(raiz, "user-1", PermissaoLer|PermissaoEscrever)
	apagou, err := remocao.Apagar(PedidoDeRemocao{Identidade: "user-1", Tenant: "org-1", Relativo: "notas/apagar.md"})
	if err != nil {
		t.Fatalf("esperava apagar, veio %v", err)
	}
	if !apagou {
		t.Fatal("esperava apagou=true")
	}
	if _, err := os.Stat(filepath.Join(raiz, "notas", "apagar.md")); err == nil {
		t.Fatal("o ficheiro devia ter desaparecido")
	}

	livro, err := os.ReadFile(filepath.Join(raiz, NomeDoLivroDeAuditoria))
	if err != nil {
		t.Fatalf("o livro de auditoria devia existir: %v", err)
	}
	if !strings.Contains(string(livro), "remocao") || !strings.Contains(string(livro), "notas/apagar.md") {
		t.Fatalf("a auditoria tem de registar o facto e o alvo: %q", string(livro))
	}
	if strings.Contains(string(livro), "medicao de teste") {
		t.Fatal("a auditoria NAO pode guardar o conteudo apagado")
	}

	// E a nota deixa de ser recuperavel pelos caminhos do adaptador.
	leitura := leituraDeTeste(raiz, "user-1", PermissaoLer)
	if _, err := leitura.Ler(PedidoDeLeitura{Identidade: "user-1", Tenant: "org-1", Relativo: "notas/apagar.md"}); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("a nota apagada nao pode ser lida, veio %v", err)
	}
}

// NEGATIVO: travessia continua recusada na remocao.
func TestApagarRecusaTravessia(t *testing.T) {
	raiz := t.TempDir()
	remocao := remocaoDeTeste(raiz, "user-1", PermissaoLer|PermissaoEscrever)
	if _, err := remocao.Apagar(PedidoDeRemocao{Identidade: "user-1", Tenant: "org-1", Relativo: "../fora.md"}); !errors.Is(err, ErrForaDaRaiz) {
		t.Fatalf("esperava ErrForaDaRaiz, veio %v", err)
	}
}
