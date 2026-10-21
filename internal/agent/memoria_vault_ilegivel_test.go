package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// O DEFEITO MEDIDO (nilerr): `os.ReadFile` devolvia `nil` em QUALQUER erro, e o comentario dizia
// "nao existe: criacao". Um erro de PERMISSAO passava como criacao, e o append escreveria por
// cima de uma nota que nao conseguiu ler -- exactamente o que a verificacao de integridade existe
// para impedir.
//
// O caso portavel de "existe mas nao se le" e' um DIRECTORIO no lugar da nota.
func TestAnexarGovernadoRecusaCaminhoIlegivel(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "notas", "ilegivel.md"), 0o700); err != nil {
		t.Fatalf("preparar directorio: %v", err)
	}
	s := ServicoDeAppendGovernado{Raiz: raiz, Politica: politicaDeTeste("user-1", EscopoTenant, PermissaoLer|PermissaoEscrever)}
	_, err := s.Anexar(PedidoDeAppendGovernado{
		Identidade: "user-1", Tenant: "org-1", Relativo: "notas/ilegivel.md",
		Memoria: Memoria{ID: "m1", Rotulo: MemoriaFatoVerificado, Conteudo: "x", Fonte: "chat"},
	})
	if err == nil {
		t.Fatal("um caminho ilegivel tem de ser recusado, nao tratado como criacao")
	}
	if !errors.Is(err, ErrConteudoAlterado) {
		t.Fatalf("devia ser ErrConteudoAlterado (falha fechada), veio %v", err)
	}
}
