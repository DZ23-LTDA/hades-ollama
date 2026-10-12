package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// Estagio 30 — APROVACAO DE GASTO ACIMA DO ORCAMENTO (E-264), limite declarado em E-263.
//
// O CONTROLE CENTRAL: quem pede nao aprova. Sem isso a "aprovacao" seria o proprio gasto com um carimbo.
func TestAreaCostApprovalExigeAprovadorDiferenteESemLancamentoDuplo(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{
		Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot: filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	empresa, err := runtime.CompanyStore().Create(Company{OrganizationID: LocalOrganizationID, Name: "Loja", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	financas := runtime.Finance()
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if _, err := financas.SetAreaBudget(empresa.ID, "marketing", 10000, now); err != nil {
		t.Fatal(err)
	}
	if _, err := financas.PostAreaCost(empresa.ID, "marketing", "ADS-1", "anuncio", 8000, "ana", now); err != nil {
		t.Fatal(err)
	}
	// 1. Acima do orcamento SEM aprovacao continua RECUSADO (E-262 nao regride).
	if _, err := financas.PostAreaCost(empresa.ID, "marketing", "ADS-2", "anuncio grande", 5000, "ana", now); !errors.Is(err, ErrAreaBudgetExceeded) {
		t.Fatalf("sem aprovacao o estouro precisa continuar recusado: %v", err)
	}
	// Custo que CABE no orcamento nao precisa de aprovacao: pedir seria burocracia sem sentido.
	if _, err := financas.RequestAreaCostApproval(empresa.ID, "marketing", "ADS-3", "anuncio pequeno", 1000, "ana", now); !errors.Is(err, ErrAreaCostApprovalInvalid) {
		t.Fatalf("valor dentro do orcamento nao precisa de aprovacao: %v", err)
	}

	// 2. O PEDIDO nao lanca nada: pedir nao e gastar.
	pedido, err := financas.RequestAreaCostApproval(empresa.ID, "marketing", "ADS-2", "anuncio grande", 5000, "ana", now)
	if err != nil {
		t.Fatalf("pedido de aprovacao: %v", err)
	}
	if pedido.Status != AreaCostPending {
		t.Fatalf("pedido precisa nascer pendente: %+v", pedido)
	}
	status, err := financas.AreaBudgetStatus(empresa.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if status[0].SpentCents != 8000 {
		t.Fatalf("o PEDIDO nao pode lancar nada: gasto=%d", status[0].SpentCents)
	}

	// 3. QUEM PEDE NAO APROVA.
	if _, err := financas.DecideAreaCostApproval(empresa.ID, pedido.ID, "ana", true, now); !errors.Is(err, ErrAreaCostSelfApproval) {
		t.Fatalf("quem pediu nao pode aprovar: %v", err)
	}
	status, err = financas.AreaBudgetStatus(empresa.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if status[0].SpentCents != 8000 {
		t.Fatalf("a tentativa de autoaprovacao nao pode lancar: gasto=%d", status[0].SpentCents)
	}

	// 4. Aprovado por OUTRA pessoa: o custo E lancado, acima do limite, porque houve autorizacao humana.
	decidido, err := financas.DecideAreaCostApproval(empresa.ID, pedido.ID, "bruno", true, now)
	if err != nil {
		t.Fatalf("aprovacao por outra pessoa: %v", err)
	}
	if decidido.Status != AreaCostApproved || decidido.CostEntryID == "" {
		t.Fatalf("aprovacao precisa registrar o lancamento: %+v", decidido)
	}
	status, err = financas.AreaBudgetStatus(empresa.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if status[0].SpentCents != 13000 {
		t.Fatalf("o custo autorizado precisa ser lancado acima do limite: %+v", status[0])
	}
	if !status[0].Exceeded {
		t.Fatalf("o status precisa mostrar que a area ESTOUROU: %+v", status[0])
	}

	// 5. Decidir DE NOVO e recusado (sem lancamento duplo).
	if _, err := financas.DecideAreaCostApproval(empresa.ID, pedido.ID, "bruno", true, now); !errors.Is(err, ErrAreaCostApprovalDecided) {
		t.Fatalf("decidir duas vezes precisa ser recusado: %v", err)
	}
	status, err = financas.AreaBudgetStatus(empresa.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if status[0].SpentCents != 13000 {
		t.Fatalf("lancamento duplo pela releitura da decisao: %d", status[0].SpentCents)
	}

	// 6. RECUSA nao lanca nada.
	outro, err := financas.RequestAreaCostApproval(empresa.ID, "marketing", "ADS-4", "outro", 20000, "ana", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := financas.DecideAreaCostApproval(empresa.ID, outro.ID, "bruno", false, now); err != nil {
		t.Fatal(err)
	}
	status, err = financas.AreaBudgetStatus(empresa.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if status[0].SpentCents != 13000 {
		t.Fatalf("pedido recusado nao pode lancar: %d", status[0].SpentCents)
	}

	// PERSISTENCIA e ISOLAMENTO.
	reaberto, err := NewRuntime(RuntimeConfig{
		Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot: filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	aprovacoes, err := reaberto.Finance().AreaCostApprovals(empresa.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(aprovacoes) != 2 {
		t.Fatalf("os pedidos precisam sobreviver ao reinicio: %d", len(aprovacoes))
	}
	outraEmpresa, err := runtime.CompanyStore().Create(Company{OrganizationID: "org-b", Name: "Outra", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	if outras, err := reaberto.Finance().AreaCostApprovals(outraEmpresa.ID); err != nil || len(outras) != 0 {
		t.Fatalf("pedidos nao podem vazar entre empresas: %v %+v", err, outras)
	}
}
