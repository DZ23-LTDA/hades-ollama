package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func leituraDeTeste(raiz string, identidade string, permissoes PermissaoDeMemoria) LeituraGovernada {
	return LeituraGovernada{
		Raiz:     raiz,
		Politica: politicaDeTeste(identidade, EscopoTenant, permissoes),
	}
}

func gravarNotaDeTeste(t *testing.T, raiz, tenant, relativo string) {
	t.Helper()
	servico := ServicoDeAppendGovernado{
		Raiz:     raiz,
		Politica: politicaDeTeste("user-1", EscopoTenant, PermissaoLer|PermissaoEscrever),
	}
	pedido := pedidoDeTeste("user-1", tenant, relativo)
	if _, err := servico.Anexar(pedido); err != nil {
		t.Fatalf("nao consegui preparar a nota %q: %v", relativo, err)
	}
}

// POSITIVO: o que foi escrito volta, com a proveniencia (Fonte) preservada.
func TestLeituraGovernadaDevolveAMesmaNotaComFonte(t *testing.T) {
	raiz := t.TempDir()
	gravarNotaDeTeste(t, raiz, "org-1", "notas/uma.md")

	leitura := leituraDeTeste(raiz, "user-1", PermissaoLer)
	memoria, err := leitura.Ler(PedidoDeLeitura{Identidade: "user-1", Tenant: "org-1", Relativo: "notas/uma.md"})
	if err != nil {
		t.Fatalf("devia ler a propria nota: %v", err)
	}
	if memoria.Conteudo != "medicao de teste" {
		t.Fatalf("conteudo divergente: %q", memoria.Conteudo)
	}
	if memoria.Fonte != "go test" {
		t.Fatalf("a proveniencia perdeu-se: Fonte=%q", memoria.Fonte)
	}
}

// NEGATIVO: sem permissao de leitura, nega. Escrever nao implica ler e vice-versa.
func TestLeituraGovernadaSemPermissaoNega(t *testing.T) {
	raiz := t.TempDir()
	gravarNotaDeTeste(t, raiz, "org-1", "notas/uma.md")

	leitura := leituraDeTeste(raiz, "user-1", PermissaoEscrever) // so' escrita
	if _, err := leitura.Ler(PedidoDeLeitura{Identidade: "user-1", Tenant: "org-1", Relativo: "notas/uma.md"}); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("esperava ErrAcessoNegado, veio %v", err)
	}
}

// NEGATIVO E DECISIVO: isolamento entre tenants sem revelar existencia.
//
// Nota de outro tenant, nota inexistente e caminho invalido tem de devolver o MESMO erro. Se
// devolvessem erros diferentes, a propria diferenca entregaria a existencia.
func TestLeituraGovernadaNaoRevelaExistenciaEntreTenants(t *testing.T) {
	raiz := t.TempDir()
	gravarNotaDeTeste(t, raiz, "org-DONA", "notas/da-outra.md")

	leitura := leituraDeTeste(raiz, "user-1", PermissaoLer)

	_, errOutroTenant := leitura.Ler(PedidoDeLeitura{Identidade: "user-1", Tenant: "org-1", Relativo: "notas/da-outra.md"})
	_, errInexistente := leitura.Ler(PedidoDeLeitura{Identidade: "user-1", Tenant: "org-1", Relativo: "notas/nao-existe.md"})

	if !errors.Is(errOutroTenant, ErrAcessoNegado) {
		t.Fatalf("nota de outro tenant devia negar, veio %v", errOutroTenant)
	}
	if !errors.Is(errInexistente, ErrAcessoNegado) {
		t.Fatalf("nota inexistente devia negar, veio %v", errInexistente)
	}
	if errOutroTenant.Error() != errInexistente.Error() {
		t.Fatalf("os dois casos tem de ser indistinguiveis: %q vs %q", errOutroTenant.Error(), errInexistente.Error())
	}
}

// NEGATIVO: conteudo adulterado no disco e' detectado, nao servido.
func TestLeituraGovernadaDetectaAdulteracao(t *testing.T) {
	raiz := t.TempDir()
	gravarNotaDeTeste(t, raiz, "org-1", "notas/uma.md")

	alvo := filepath.Join(raiz, "notas", "uma.md")
	dados, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatalf("leitura de preparacao: %v", err)
	}
	adulterado := append(dados, []byte("\nconteudo inserido a mao\n")...)
	if err := os.WriteFile(alvo, adulterado, 0o600); err != nil {
		t.Fatalf("nao consegui adulterar: %v", err)
	}

	leitura := leituraDeTeste(raiz, "user-1", PermissaoLer)
	if _, err := leitura.Ler(PedidoDeLeitura{Identidade: "user-1", Tenant: "org-1", Relativo: "notas/uma.md"}); err == nil {
		t.Fatal("conteudo adulterado devia ser detectado")
	}
}

// NEGATIVO: travessia continua recusada na leitura.
func TestLeituraGovernadaRecusaTravessia(t *testing.T) {
	raiz := t.TempDir()
	leitura := leituraDeTeste(raiz, "user-1", PermissaoLer)
	if _, err := leitura.Ler(PedidoDeLeitura{Identidade: "user-1", Tenant: "org-1", Relativo: "../fora.md"}); !errors.Is(err, ErrForaDaRaiz) {
		t.Fatalf("esperava ErrForaDaRaiz, veio %v", err)
	}
}

// NEGATIVO: sem raiz, sem identidade, sem tenant.
func TestLeituraGovernadaRecusaSemContexto(t *testing.T) {
	if _, err := (LeituraGovernada{}).Ler(PedidoDeLeitura{Identidade: "u", Tenant: "t", Relativo: "n.md"}); !errors.Is(err, ErrRaizDoVaultAusente) {
		t.Fatalf("esperava ErrRaizDoVaultAusente, veio %v", err)
	}
	raiz := t.TempDir()
	leitura := leituraDeTeste(raiz, "user-1", PermissaoLer)
	if _, err := leitura.Ler(PedidoDeLeitura{Tenant: "org-1", Relativo: "n.md"}); !errors.Is(err, ErrIdentidadeDeSessaoAusente) {
		t.Fatalf("esperava ErrIdentidadeDeSessaoAusente, veio %v", err)
	}
	if _, err := leitura.Ler(PedidoDeLeitura{Identidade: "user-1", Relativo: "n.md"}); !errors.Is(err, ErrTenantDaSessaoAusente) {
		t.Fatalf("esperava ErrTenantDaSessaoAusente, veio %v", err)
	}
}
