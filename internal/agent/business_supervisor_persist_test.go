package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// Estágio 31 (recuperação) — o supervisor de negócio precisa sobreviver a um REINÍCIO
// REAL: uma decisão que espera humano não pode desaparecer, e uma decisão já tomada não
// pode voltar a "pendente".
func persistentSupervisorRuntime(t *testing.T) (*Runtime, string) {
	t.Helper()
	root := t.TempDir()
	dataRoot := filepath.Join(root, "data")
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot:      dataRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime, dataRoot
}

// geraEstornoDevido cria o cenário que exige decisão humana: pedido pago, receita
// lançada e depois cancelado.
func geraEstornoDevido(t *testing.T, runtime *Runtime, organizationID, key string, now time.Time) {
	t.Helper()
	if _, err := runtime.CompanyStore().Create(Company{OrganizationID: organizationID, Name: "Loja", Currency: "BRL"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: organizationID, SKU: "CAM-001", Name: "Camiseta", PriceCents: 7900, Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().RecordStock(organizationID, "CAM-001", 10, CommerceMovementEntry, "carga", now); err != nil {
		t.Fatal(err)
	}
	order, err := runtime.Commerce().CreateOrder(CommerceOrderRequest{
		OrganizationID: organizationID, IdempotencyKey: key,
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().ConfirmPayment(organizationID, order.ID, CommercePayment{
		Method: CommercePaymentSandbox, Reference: "r-" + key,
	}, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Finance().PostOrderRevenue(organizationID, order.ID, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().Cancel(organizationID, order.ID, "desistiu", now); err != nil {
		t.Fatal(err)
	}
}

func TestBusinessSupervisorPendingDecisionSurvivesRealRestart(t *testing.T) {
	runtime, dataRoot := persistentSupervisorRuntime(t)
	now := time.Now().UTC()
	organizationID := LocalOrganizationID
	geraEstornoDevido(t, runtime, organizationID, "sup-persist", now)

	// Ciclo: o estorno devido fica AGUARDANDO decisão humana.
	result, err := runtime.BusinessSupervisor().BusinessTick(organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.AwaitingApproval == 0 {
		t.Fatalf("esperava pendência de aprovação: %+v", result.Actions)
	}
	pending := runtime.BusinessSupervisor().PendingActions(organizationID)
	if len(pending) != 1 {
		t.Fatalf("uma pendência esperada: %+v", pending)
	}
	actionID := pending[0].ID

	// REINÍCIO REAL: novo runtime sobre o MESMO diretório de dados.
	restarted, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(filepath.Dir(dataRoot), "ws"),
		DataRoot:      dataRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	reloaded := restarted.BusinessSupervisor().PendingActions(organizationID)
	if len(reloaded) != 1 {
		t.Fatalf("a pendência precisa sobreviver ao reinício: %+v", reloaded)
	}
	if reloaded[0].ID != actionID || reloaded[0].Kind != BusinessActionRefund {
		t.Fatalf("a pendência precisa ser a MESMA: %+v", reloaded[0])
	}
	// A política vigente também sobrevive.
	if restarted.BusinessSupervisor().Policy().BudgetCents == 0 {
		t.Fatal("a política precisa sobreviver ao reinício")
	}
	// E a decisão pode ser tomada DEPOIS do reinício, com efeito real.
	decided, err := restarted.BusinessSupervisor().Decide(organizationID, actionID, "APROVAR", "gerente-ana", "cliente pediu", now)
	if err != nil {
		t.Fatal(err)
	}
	if decided.Status != BusinessActionExecuted || decided.DecidedBy != "gerente-ana" {
		t.Fatalf("a decisão pós-reinício precisa executar e registrar o ator: %+v", decided)
	}
	entries, err := restarted.CompanyStore().ListLedgerEntries(restarted.CompanyStore().List(organizationID)[0].ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	refunds := 0
	for _, entry := range entries {
		if entry.Category == "estorno" {
			refunds++
		}
	}
	if refunds != 1 {
		t.Fatalf("o estorno precisa ser registrado uma vez: %d", refunds)
	}
}

func TestBusinessSupervisorDecidedActionDoesNotReturnToPending(t *testing.T) {
	runtime, dataRoot := persistentSupervisorRuntime(t)
	now := time.Now().UTC()
	organizationID := LocalOrganizationID
	geraEstornoDevido(t, runtime, organizationID, "sup-decided", now)
	if _, err := runtime.BusinessSupervisor().BusinessTick(organizationID, now); err != nil {
		t.Fatal(err)
	}
	pending := runtime.BusinessSupervisor().PendingActions(organizationID)
	if len(pending) == 0 {
		t.Fatal("esperava pendência")
	}
	// RECUSA registrada antes do reinício.
	if _, err := runtime.BusinessSupervisor().Decide(organizationID, pending[0].ID, "RECUSAR", "gerente-bruno", "não autorizado", now); err != nil {
		t.Fatal(err)
	}

	restarted, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(filepath.Dir(dataRoot), "ws"),
		DataRoot:      dataRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	// A decisão NÃO volta a pendente: recusado continua recusado.
	if stillPending := restarted.BusinessSupervisor().PendingActions(organizationID); len(stillPending) != 0 {
		t.Fatalf("ação recusada não pode voltar a pendente: %+v", stillPending)
	}
	// E decidir de novo é idempotente: devolve a MESMA recusa sem executar efeito.
	again, err := restarted.BusinessSupervisor().Decide(organizationID, pending[0].ID, "APROVAR", "gerente-ana", "mudei de ideia", now)
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != BusinessActionDenied || again.DecidedBy != "gerente-bruno" {
		t.Fatalf("a decisão original precisa prevalecer: %+v", again)
	}
	entries, err := restarted.CompanyStore().ListLedgerEntries(restarted.CompanyStore().List(organizationID)[0].ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Category == "estorno" {
			t.Fatalf("ação recusada não pode ser executada depois: %+v", entry)
		}
	}
}

// O orçamento consumido e a contagem de ciclos também sobrevivem: sem isso, um reinício
// devolveria orçamento já gasto e o supervisor gastaria duas vezes.
func TestBusinessSupervisorBudgetAndTickCountSurviveRestart(t *testing.T) {
	runtime, dataRoot := persistentSupervisorRuntime(t)
	now := time.Now().UTC()
	organizationID := LocalOrganizationID
	if _, err := runtime.CompanyStore().Create(Company{OrganizationID: organizationID, Name: "Loja", Currency: "BRL"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: organizationID, SKU: "CAM-001", Name: "Camiseta", PriceCents: 7900, Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	// Estoque abaixo do mínimo: reposição autônoma com custo.
	if _, err := runtime.Commerce().RecordStock(organizationID, "CAM-001", 1, CommerceMovementEntry, "carga", now); err != nil {
		t.Fatal(err)
	}
	first, err := runtime.BusinessSupervisor().BusinessTick(organizationID, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.BudgetSpentCents != 35000 || first.TickNumber != 1 {
		t.Fatalf("primeiro ciclo incorreto: %+v", first)
	}

	restarted, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(filepath.Dir(dataRoot), "ws"),
		DataRoot:      dataRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := restarted.BusinessSupervisor().BusinessTick(organizationID, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if second.TickNumber != 2 {
		t.Fatalf("a contagem de ciclos precisa continuar: %+v", second)
	}
	if second.BudgetSpentCents != 35000 {
		t.Fatalf("o orçamento gasto precisa sobreviver: %+v", second)
	}
	if second.BudgetLeftCents != 65000 {
		t.Fatalf("o orçamento restante precisa sobreviver: %+v", second)
	}
}
