package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// Estágio 31 — supervisor autônomo de negócio.
//
// As propriedades travadas aqui:
//  1. o supervisor OBSERVA os três domínios (atendimento, comércio, financeiro) e
//     registra os achados nos MESMOS ocorrências que o operador consulta;
//  2. ação REVERSÍVEL/CONTÁBIL é autônoma e produz efeito REAL;
//  3. ação IRREVERSÍVEL (dinheiro, desfazer venda) fica AGUARDANDO APROVAÇÃO e NÃO
//     executa sozinha;
//  4. ORÇAMENTO é limite duro: estourar NÃO executa e vira bloqueio com motivo;
//  5. aprovar executa o efeito real; recusar não executa; decidir de novo é idempotente;
//  6. recuperação observada resolve o incidente;
//  7. ação desconhecida é recusada em vez de "concluída" sem efeito.
func businessSupervisorFixture(t *testing.T) (*BusinessSupervisor, *Runtime, *CommerceStore, *FinanceOps) {
	t.Helper()
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot:      filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultSupervisorPolicy()
	return NewBusinessSupervisor(runtime, policy), runtime, runtime.Commerce(), runtime.Finance()
}

// preparaEmpresa cria a empresa do sandbox com catálogo e estoque.
func preparaEmpresa(t *testing.T, runtime *Runtime, organizationID, sku string, price int64, stock int64) {
	t.Helper()
	now := time.Now().UTC()
	if _, err := runtime.CompanyStore().Create(Company{OrganizationID: organizationID, Name: "Loja", Currency: "BRL"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: organizationID, SKU: sku, Name: "Camiseta", PriceCents: price, Active: true,
	}, now); err != nil {
		t.Fatal(err)
	}
	if stock > 0 {
		if _, err := runtime.Commerce().RecordStock(organizationID, sku, stock, CommerceMovementEntry, "carga", now); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBusinessSupervisorObservesDomainsAndRecordsIncidents(t *testing.T) {
	supervisor, runtime, commerce, finance := businessSupervisorFixture(t)
	now := time.Now().UTC()
	preparaEmpresa(t, runtime, "org-a", "CAM-001", 7900, 10)
	// Gera dois achados ALTOS: receita não lançada e estoque baixo.
	order, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "sup-1",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 9}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.ConfirmPayment("org-a", order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "r1"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.Fulfill("org-a", order.ID, now); err != nil {
		t.Fatal(err)
	}
	paymentOrder, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "sup-2",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.ConfirmPayment("org-a", paymentOrder.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "r2"}, "ana", now); err != nil {
		t.Fatal(err)
	}

	result, err := supervisor.BusinessTick("org-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Signals) == 0 {
		t.Fatalf("o supervisor precisa observar sinais: %+v", result)
	}
	domains := map[string]int{}
	for _, signal := range result.Signals {
		domains[signal.Domain]++
	}
	if domains["financeiro"] == 0 {
		t.Fatalf("o achado de receita não lançada precisa aparecer: %+v", result.Signals)
	}
	// Os achados ALTOS viram ocorrências nos MESMOS registros que o operador consulta.
	if result.IncidentsRecorded == 0 {
		t.Fatalf("achados altos precisam virar incidente: %+v", result)
	}
	incidents := runtime.Supervisor().Incidents()
	foundBusiness := false
	for _, incident := range incidents {
		if len(incident.Kind) > 8 && incident.Kind[:8] == "negocio_" {
			foundBusiness = true
		}
	}
	if !foundBusiness {
		t.Fatalf("os ocorrências de negócio precisam ficar no journal do supervisor existente: %+v", incidents)
	}
	// A receita não lançada é resolvida por ação AUTÔNOMA (registro contábil idempotente).
	report, err := finance.Report("org-a", time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if report.UnpostedRevenue != 0 {
		t.Fatalf("a ação autônoma precisa ter lançado a receita: %+v", report)
	}
}

func TestBusinessSupervisorReplenishesStockWithinBudget(t *testing.T) {
	supervisor, runtime, commerce, _ := businessSupervisorFixture(t)
	now := time.Now().UTC()
	preparaEmpresa(t, runtime, "org-a", "CAM-001", 7900, 1)
	// Reposição tem custo: 10 unidades × 3500 centavos = 35000.
	result, err := supervisor.BusinessTick("org-a", now)
	if err != nil {
		t.Fatal(err)
	}
	var replenish *BusinessAction
	for index := range result.Actions {
		if result.Actions[index].Kind == BusinessActionReplenish {
			replenish = &result.Actions[index]
		}
	}
	if replenish == nil {
		t.Fatalf("estoque baixo precisa propor reposição: %+v", result.Actions)
	}
	if replenish.Status != BusinessActionExecuted || replenish.Risk != BusinessActionAutonomous {
		t.Fatalf("reposição é ajuste reversível e precisa ser autônoma: %+v", replenish)
	}
	if result.BudgetSpentCents != 35000 || result.BudgetLeftCents != 65000 {
		t.Fatalf("o custo precisa entrar no orçamento: %+v", result)
	}
	// O efeito é REAL: o estoque subiu e o movimento ficou no livro.
	balance := commerce.Stock("org-a", "CAM-001")
	if balance.OnHand != 11 {
		t.Fatalf("a reposição precisa acontecer de verdade: %+v", balance)
	}
	found := false
	for _, movement := range commerce.Movements("org-a", "CAM-001") {
		if movement.Reference == "reposicao-automatica" {
			found = true
		}
	}
	if !found {
		t.Fatal("a reposição precisa ser auditável no livro de estoque")
	}
}

func TestBusinessSupervisorBlocksActionWhenBudgetIsExceeded(t *testing.T) {
	supervisor, runtime, commerce, _ := businessSupervisorFixture(t)
	now := time.Now().UTC()
	policy := DefaultSupervisorPolicy()
	policy.BudgetCents = 1000 // menor que o custo da reposição (35000)
	supervisor.SetPolicy(policy)
	preparaEmpresa(t, runtime, "org-a", "CAM-001", 7900, 0)

	result, err := supervisor.BusinessTick("org-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if result.BlockedByBudget != 1 {
		t.Fatalf("estourar orçamento precisa BLOQUEAR a ação: %+v", result)
	}
	var blocked BusinessAction
	for _, action := range result.Actions {
		if action.Status == BusinessActionBudgetStop {
			blocked = action
		}
	}
	if blocked.Effect == "" {
		t.Fatalf("o bloqueio precisa vir com motivo: %+v", blocked)
	}
	// E o efeito NÃO aconteceu.
	if balance := commerce.Stock("org-a", "CAM-001"); balance.OnHand != 0 {
		t.Fatalf("ação bloqueada não pode executar: %+v", balance)
	}
	if result.BudgetSpentCents != 0 {
		t.Fatalf("ação bloqueada não pode consumir orçamento: %+v", result)
	}
}

func TestBusinessSupervisorRequiresApprovalForIrreversibleActions(t *testing.T) {
	supervisor, runtime, commerce, finance := businessSupervisorFixture(t)
	now := time.Now().UTC()
	preparaEmpresa(t, runtime, "org-a", "CAM-001", 7900, 10)
	// Pedido pago, receita lançada e depois CANCELADO: gera estorno DEVIDO.
	order, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "sup-refund",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.ConfirmPayment("org-a", order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "r3"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := finance.PostOrderRevenue("org-a", order.ID, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.Cancel("org-a", order.ID, "cliente desistiu", now); err != nil {
		t.Fatal(err)
	}

	result, err := supervisor.BusinessTick("org-a", now)
	if err != nil {
		t.Fatal(err)
	}
	// A receita do pedido cancelado NÃO é "não lançada": ela gera ESTORNO, que exige humano.
	refund := BusinessAction{}
	for _, action := range result.Actions {
		if action.Kind == BusinessActionRefund {
			refund = action
		}
	}
	if refund.ID == "" {
		t.Fatalf("o estorno devido precisa virar ação: %+v", result.Actions)
	}
	if refund.Risk != BusinessActionApproval || refund.Status != BusinessActionPending {
		t.Fatalf("mexer em dinheiro exige aprovação humana: %+v", refund)
	}
	// NADA foi executado sem decisão.
	entries, err := runtime.CompanyStore().ListLedgerEntries(runtime.CompanyStore().List("org-a")[0].ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Category == "estorno" {
			t.Fatalf("o estorno não pode ser lançado sem aprovação: %+v", entry)
		}
	}
	// Aprovar executa o efeito REAL e registra quem decidiu.
	approved, err := supervisor.Decide("org-a", refund.ID, "APROVAR", "gerente-ana", "cliente pediu devolução", now)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != BusinessActionExecuted || approved.DecidedBy != "gerente-ana" {
		t.Fatalf("a aprovação precisa executar e registrar o ator: %+v", approved)
	}
	entries, err = runtime.CompanyStore().ListLedgerEntries(runtime.CompanyStore().List("org-a")[0].ID, time.Time{}, time.Time{})
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
	// Decidir DE NOVO é idempotente: não repete o efeito.
	again, err := supervisor.Decide("org-a", refund.ID, "APROVAR", "gerente-ana", "de novo", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != BusinessActionExecuted {
		t.Fatalf("a segunda decisão precisa devolver a mesma ação: %+v", again)
	}
	entries, err = runtime.CompanyStore().ListLedgerEntries(runtime.CompanyStore().List("org-a")[0].ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	refunds = 0
	for _, entry := range entries {
		if entry.Category == "estorno" {
			refunds++
		}
	}
	if refunds != 1 {
		t.Fatalf("aprovar duas vezes não pode estornar duas vezes: %d", refunds)
	}
}

func TestBusinessSupervisorDenyDoesNotExecuteAndNeedsActor(t *testing.T) {
	supervisor, runtime, commerce, finance := businessSupervisorFixture(t)
	now := time.Now().UTC()
	preparaEmpresa(t, runtime, "org-a", "CAM-001", 7900, 10)
	order, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "sup-deny",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.ConfirmPayment("org-a", order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "r4"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := finance.PostOrderRevenue("org-a", order.ID, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.Cancel("org-a", order.ID, "desistiu", now); err != nil {
		t.Fatal(err)
	}
	result, err := supervisor.BusinessTick("org-a", now)
	if err != nil {
		t.Fatal(err)
	}
	var refund BusinessAction
	for _, action := range result.Actions {
		if action.Kind == BusinessActionRefund {
			refund = action
		}
	}
	if refund.ID == "" {
		t.Fatalf("esperava ação de estorno: %+v", result.Actions)
	}
	// Sem ator identificado não há decisão: aprovação anônima não é trilha de auditoria.
	if _, err := supervisor.Decide("org-a", refund.ID, "APROVAR", "  ", "sem ator", now); err == nil {
		t.Fatal("decisão sem ator precisa falhar")
	}
	// Decisão inválida também é recusada.
	if _, err := supervisor.Decide("org-a", refund.ID, "TALVEZ", "ana", "", now); err == nil {
		t.Fatal("decisão inválida precisa falhar")
	}
	denied, err := supervisor.Decide("org-a", refund.ID, "RECUSAR", "gerente-bruno", "não autorizado", now)
	if err != nil {
		t.Fatal(err)
	}
	if denied.Status != BusinessActionDenied || denied.DecidedBy != "gerente-bruno" {
		t.Fatalf("a recusa precisa ser registrada com o ator: %+v", denied)
	}
	// Recusar NÃO executa: nenhuma despesa de estorno no livro.
	entries, err := runtime.CompanyStore().ListLedgerEntries(runtime.CompanyStore().List("org-a")[0].ID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Category == "estorno" {
			t.Fatalf("recusar não pode executar: %+v", entry)
		}
	}
	// Ação inexistente falha explícito.
	if _, err := supervisor.Decide("org-a", "bact_inexistente", "APROVAR", "ana", "", now); !errors.Is(err, ErrBusinessActionNotFound) {
		t.Fatalf("ação inexistente precisa falhar, veio %v", err)
	}
}

func TestBusinessSupervisorResolvesIncidentOnRecoveryAndIsScoped(t *testing.T) {
	supervisor, runtime, commerce, _ := businessSupervisorFixture(t)
	now := time.Now().UTC()
	preparaEmpresa(t, runtime, "org-a", "CAM-001", 7900, 10)
	order, err := commerce.CreateOrder(CommerceOrderRequest{
		OrganizationID: "org-a", IdempotencyKey: "sup-recover",
		Items: []CommerceOrderItem{{SKU: "CAM-001", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commerce.ConfirmPayment("org-a", order.ID, CommercePayment{Method: CommercePaymentSandbox, Reference: "r5"}, "ana", now); err != nil {
		t.Fatal(err)
	}
	// Primeiro ciclo: receita não lançada gera incidente.
	first, err := supervisor.BusinessTick("org-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if first.IncidentsRecorded == 0 {
		t.Fatalf("esperava incidente no primeiro ciclo: %+v", first)
	}
	// Segundo ciclo: a ação autônoma lançou a receita, então o incidente é RESOLVIDO.
	second, err := supervisor.BusinessTick("org-a", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if second.IncidentsRecorded != 0 {
		t.Fatalf("sem achado novo não pode abrir incidente: %+v", second)
	}
	openBusiness := 0
	for _, incident := range runtime.Supervisor().Incidents() {
		if len(incident.Kind) > 8 && incident.Kind[:8] == "negocio_" && incident.State == SupervisorIncidentOpen {
			openBusiness++
		}
	}
	if openBusiness != 0 {
		t.Fatalf("a recuperação observada precisa resolver o incidente: %d abertos", openBusiness)
	}
	// ESCOPO: a outra organização não vê pendências nem histórico da primeira.
	if pending := supervisor.PendingActions("org-b"); len(pending) != 0 {
		t.Fatalf("pendências precisam ser por organização: %+v", pending)
	}
	// Organização SEM empresa criada: o domínio financeiro é ILEGÍVEL, e isso precisa
	// ser DECLARADO no resultado — sem derrubar a supervisão dos outros domínios.
	other, err := supervisor.BusinessTick("org-b", now)
	if err != nil {
		t.Fatalf("falha de um domínio não pode derrubar o ciclo: %v", err)
	}
	declared := false
	for _, message := range other.Errors {
		if len(message) >= 10 && message[:10] == "financeiro" {
			declared = true
		}
	}
	if !declared {
		t.Fatalf("a falha do domínio financeiro precisa ser declarada: %+v", other.Errors)
	}
	for _, action := range other.Actions {
		if action.OrganizationID != "org-b" {
			t.Fatalf("ação de outra organização no ciclo: %+v", action)
		}
	}
	// Ação desconhecida é RECUSADA em vez de "concluída" sem efeito.
	if _, err := supervisor.execute(BusinessAction{Kind: "voar", OrganizationID: "org-a"}); !errors.Is(err, ErrBusinessActionUnknown) {
		t.Fatalf("tipo desconhecido precisa ser recusado, veio %v", err)
	}
}
