package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// Estagio 27 — PRECIFICACAO DE PROPOSTA (E-269). O limite declarado dizia "sem desconto negociado, sem imposto/frete".
func TestPropostaComDescontoImpostoEFreteUsaPrecoDoCatalogo(t *testing.T) {
	root := t.TempDir()
	commerce, err := NewCommerceStoreAt(filepath.Join(root, "commerce"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	org := LocalOrganizationID
	if _, err := commerce.UpsertProduct(CommerceProduct{OrganizationID: org, SKU: "KIT-1", Name: "Kit", PriceCents: 10000, Active: true}, now); err != nil {
		t.Fatal(err)
	}
	vendas := NewSalesWorkforceStore(filepath.Join(root, "sales"), commerce, nil)
	pedido := func(bp, imposto, frete int64) SalesProposalRequest {
		return SalesProposalRequest{
			OrganizationID: org, ContactID: "ct_1",
			Items:               []SalesProposalItemRequest{{SKU: "KIT-1", Quantity: 1}},
			DiscountBasisPoints: bp, TaxBasisPoints: imposto, ShippingCents: frete,
		}
	}

	primeira, err := vendas.CreateProposal(pedido(0, 0, 0), now)
	if err != nil {
		t.Fatalf("proposta sem ajustes: %v", err)
	}
	if primeira.Items[0].UnitPriceCents != 10000 || primeira.TotalCents != 10000 {
		t.Fatalf("preco precisa vir do catalogo: %+v", primeira)
	}
	if primeira.ShippingDeclared {
		t.Fatal("frete zero NAO informado nao pode se declarar informado")
	}

	// SEM POLITICA, DESCONTO NENHUM: o teto padrao e zero.
	if _, err := vendas.CreateProposal(pedido(500, 0, 0), now); !errors.Is(err, ErrProposalDiscountExceeded) {
		t.Fatalf("sem politica qualquer desconto precisa ser recusado: %v", err)
	}
	if propostas := vendas.Proposals(org, ""); len(propostas) != 1 {
		t.Fatalf("a proposta recusada nao pode ter sido gravada: %d", len(propostas))
	}

	if err := vendas.SetDiscountPolicy(org, 2000); err != nil {
		t.Fatal(err)
	}
	if _, err := vendas.CreateProposal(pedido(2500, 0, 0), now); !errors.Is(err, ErrProposalDiscountExceeded) {
		t.Fatalf("desconto acima do teto precisa ser recusado: %v", err)
	}
	comDesconto, err := vendas.CreateProposal(pedido(1500, 1000, 990), now)
	if err != nil {
		t.Fatalf("desconto dentro do teto: %v", err)
	}
	// subtotal 10000 - 15% (1500) = 8500; imposto 10% SOBRE A BASE (850); frete 990 => 10340.
	if comDesconto.DiscountCents != 1500 || comDesconto.TaxCents != 850 || comDesconto.TotalCents != 10340 {
		t.Fatalf("conta errada: desconto=%d imposto=%d total=%d", comDesconto.DiscountCents, comDesconto.TaxCents, comDesconto.TotalCents)
	}
	if !comDesconto.ShippingDeclared {
		t.Fatal("frete informado precisa se declarar informado")
	}
	if comDesconto.TaxCents == 1000 {
		t.Fatal("o imposto incidiu sobre o SUBTOTAL: teria de incidir sobre a base com desconto")
	}

	if _, err := vendas.CreateProposal(pedido(0, 10001, 0), now); !errors.Is(err, ErrProposalPricingInvalid) {
		t.Fatalf("imposto acima de 100%% precisa ser recusado: %v", err)
	}
	if _, err := vendas.CreateProposal(pedido(0, 0, -1), now); !errors.Is(err, ErrProposalPricingInvalid) {
		t.Fatalf("frete negativo precisa ser recusado: %v", err)
	}

	// PERSISTENCIA da politica: reinicio REAL sobre o mesmo diretorio.
	reaberta := NewSalesWorkforceStore(filepath.Join(root, "sales"), commerce, nil)
	depoisDoRestart, err := reaberta.CreateProposal(pedido(1500, 1000, 990), now)
	if err != nil {
		t.Fatalf("a politica precisa sobreviver ao reinicio: %v", err)
	}
	if depoisDoRestart.TotalCents != 10340 {
		t.Fatalf("conta diferente apos o reinicio: %d", depoisDoRestart.TotalCents)
	}

	// OUTRA organizacao nao herda a politica: para ela o teto padrao e ZERO.
	// (o catalogo dela precisa existir ANTES: sem produto o erro seria de SKU, nao de desconto)
	if _, err := commerce.UpsertProduct(CommerceProduct{OrganizationID: "org-b", SKU: "KIT-1", Name: "Kit", PriceCents: 10000, Active: true}, now); err != nil {
		t.Fatal(err)
	}
	outro := pedido(1500, 0, 0)
	outro.OrganizationID = "org-b"
	if _, err := reaberta.CreateProposal(outro, now); !errors.Is(err, ErrProposalDiscountExceeded) {
		t.Fatalf("politica de desconto nao pode vazar entre organizacoes: %v", err)
	}
}
