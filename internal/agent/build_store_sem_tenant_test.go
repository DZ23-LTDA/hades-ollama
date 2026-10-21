package agent

import (
	"errors"
	"testing"
)

// O `E-1470` fechou a leitura sem dono; este teste fecha a CRIACAO sem dono pelo caminho de produto.
// Um handler que passasse `""` (por esquecer a sessao) voltaria a criar um registo que qualquer
// organizacao le -- e o compilador nao o impede, porque a string vazia e' um valor legitimo de string.
func TestRegistarDoTenantRecusaOrganizacaoVazia(t *testing.T) {
	s := NovoBuildStore()
	for _, vazio := range []string{"", "   ", "\t"} {
		if _, err := s.RegistarDoTenant("alvo", "raiz", "destino", vazio); !errors.Is(err, ErrBuildSemTenant) {
			t.Fatalf("tenant %q devia dar ErrBuildSemTenant, veio %v", vazio, err)
		}
	}
	if n := len(s.ListarDoTenant("")); n != 0 {
		t.Fatalf("nenhum registo devia ter sido criado, existem %d", n)
	}
	// O caminho de compatibilidade continua a poder criar sem dono, e o nome diz o que faz.
	if _, err := s.registarSemDono("alvo", "raiz", "destino"); err != nil {
		t.Fatalf("registarSemDono devia continuar a funcionar: %v", err)
	}
	// E o caminho de produto continua a funcionar com dono.
	if _, err := s.RegistarDoTenant("alvo", "raiz", "destino", "org-1"); err != nil {
		t.Fatalf("com tenant devia registar: %v", err)
	}
}
