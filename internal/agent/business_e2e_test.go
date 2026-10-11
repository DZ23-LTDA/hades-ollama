package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// Estágio 38 — homologação de uma empresa em sandbox.
//
// As propriedades travadas aqui:
//  1. a jornada completa roda ponta a ponta em sandbox e cada etapa traz EVIDÊNCIA;
//  2. o veredito só é aprovado quando NENHUM recurso vaza entre organizações;
//  3. refazer os passos idempotentes NÃO duplica pedido nem receita;
//  4. o que depende de conta/provedor externo é DECLARADO, nunca omitido nem
//     apresentado como pronto;
//  5. a jornada sobrevive a um REINÍCIO REAL do runtime no mesmo diretório de dados.
func businessRuntime(t *testing.T) (*Runtime, string) {
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

func TestBusinessE2ECertifiesTheSandboxCompany(t *testing.T) {
	runtime, _ := businessRuntime(t)
	now := time.Now().UTC()
	input := DefaultBusinessJourneyInput(LocalOrganizationID, "org-concorrente")
	report, err := runtime.CertifyBusinessJourney(input, now)
	if err != nil {
		t.Fatalf("a jornada precisa concluir: %v", err)
	}
	if report.Verdict != BusinessVerdictOK {
		t.Fatalf("veredito esperado %s, veio %s: %+v", BusinessVerdictOK, report.Verdict, report.Steps)
	}
	// CADA etapa precisa trazer evidência (identificador do que foi criado). Um "OK"
	// sem evidência seria afirmação sem lastro.
	for _, step := range report.Steps {
		if step.Status != BusinessStepOK && step.Status != BusinessStepExternal {
			t.Fatalf("etapa %s com status inesperado: %+v", step.Step, step)
		}
		if step.Step == "" {
			t.Fatalf("etapa sem nome: %+v", step)
		}
		if step.Status == BusinessStepOK && step.Evidence == "" {
			t.Fatalf("etapa %s aprovada precisa de evidência: %+v", step.Step, step)
		}
	}
	// As etapas essenciais precisam estar no relatorio e aprovadas.
	expected := []string{
		"empresa_criada", "produto_e_estoque", "contato_no_crm", "consentimento_registrado",
		"mensagem_recebida", "agente_respondeu", "oportunidade_criada", "pedido_criado",
		"estoque_reservado", "pagamento_sandbox", "receita_lancada", "conciliacao",
		"pedido_atendido", "sla_medido", "financeiro_consistente", "recuperacao_idempotente",
		"isolamento_provado",
	}
	seen := map[string]BusinessJourneyStep{}
	for _, step := range report.Steps {
		seen[step.Step] = step
	}
	for _, name := range expected {
		step, ok := seen[name]
		if !ok {
			t.Fatalf("etapa ausente no relatório: %s", name)
		}
		if step.Status != BusinessStepOK {
			t.Fatalf("etapa %s precisa estar OK: %+v", name, step)
		}
	}
	// A dependência externa é DECLARADA, com motivo — não omitida.
	external := seen["externo_pendente_declarado"]
	if external.Status != BusinessStepExternal {
		t.Fatalf("a etapa de dependência externa precisa ser declarada: %+v", external)
	}
	if len(report.NotDone) < 5 {
		t.Fatalf("o relatório precisa declarar o que não foi feito: %+v", report.NotDone)
	}
	// Isolamento provado: cada verificação precisa ter dado negativo (não visível).
	if !report.IsolationProven {
		t.Fatalf("o isolamento precisa estar provado: %+v", report.Isolation)
	}
	if len(report.Isolation) < 3 {
		t.Fatalf("poucas verificações de isolamento: %+v", report.Isolation)
	}
	for _, check := range report.Isolation {
		if check.VisibleToOtherTenant {
			t.Fatalf("vazamento entre organizações em %s: %+v", check.Resource, check)
		}
	}
	if !report.RecoveryProven {
		t.Fatal("a idempotência da jornada precisa estar provada")
	}
	// O estado final da empresa é coerente: um pedido atendido, receita conciliada.
	if orders := runtime.Commerce().Orders(input.OrganizationID); len(orders) != 1 {
		t.Fatalf("a jornada precisa deixar um pedido: %d", len(orders))
	}
	balance := runtime.Commerce().Stock(input.OrganizationID, input.SKU)
	if balance.OnHand != input.InitialStock-input.Quantity || balance.Reserved != 0 {
		t.Fatalf("estoque final incorreto: %+v", balance)
	}
	finance, err := runtime.Finance().Report(input.OrganizationID, time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if finance.Ledger.IncomeCents != input.PriceCents*input.Quantity || len(finance.Divergences) != 0 {
		t.Fatalf("financeiro final incorreto: %+v", finance)
	}
}

// A jornada precisa sobreviver a um REINÍCIO REAL: novo runtime, mesmo diretório de
// dados. É o teste que o estágio 11.7 pede ("missão longa retoma após reiniciar") na
// versão de negócio: nenhum efeito externo acontece duas vezes.
func TestBusinessE2ESurvivesRealRestartWithoutDuplicating(t *testing.T) {
	runtime, dataRoot := businessRuntime(t)
	now := time.Now().UTC()
	input := DefaultBusinessJourneyInput(LocalOrganizationID, "org-concorrente")
	report, err := runtime.CertifyBusinessJourney(input, now)
	if err != nil {
		t.Fatalf("primeira jornada: %v", err)
	}
	if report.Verdict != BusinessVerdictOK {
		t.Fatalf("primeira jornada não aprovada: %+v", report.Steps)
	}
	orders := runtime.Commerce().Orders(input.OrganizationID)
	if len(orders) != 1 {
		t.Fatalf("um pedido após a jornada: %d", len(orders))
	}
	originalOrder := orders[0]
	originalEntry := report.Steps[10].Evidence

	// REINÍCIO: novo runtime sobre o MESMO diretório de dados.
	restarted, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(filepath.Dir(dataRoot), "ws"),
		DataRoot:      dataRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	// O pedido, o pagamento e o estoque continuam lá.
	reloaded, err := restarted.Commerce().Order(input.OrganizationID, originalOrder.ID)
	if err != nil {
		t.Fatalf("o pedido precisa sobreviver ao reinício: %v", err)
	}
	if reloaded.Status != CommerceOrderFulfilled || reloaded.TotalCents != originalOrder.TotalCents {
		t.Fatalf("estado do pedido após reinício: %+v", reloaded)
	}
	balance := restarted.Commerce().Stock(input.OrganizationID, input.SKU)
	if balance.OnHand != input.InitialStock-input.Quantity {
		t.Fatalf("estoque após reinício: %+v", balance)
	}
	// A receita continua lançada UMA vez, e a conciliação continua registrada.
	finance, err := restarted.Finance().Report(input.OrganizationID, time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if finance.Ledger.IncomeCents != input.PriceCents*input.Quantity {
		t.Fatalf("receita após reinício: %+v", finance.Ledger)
	}
	if finance.Ledger.ReconciledCount != 1 {
		t.Fatalf("a conciliação precisa sobreviver: %+v", finance.Ledger)
	}
	// E refazer o checkout idempotente depois do reinício NÃO cria um segundo pedido.
	repeated, err := restarted.Commerce().CreateOrder(CommerceOrderRequest{
		OrganizationID: input.OrganizationID, ContactID: originalOrder.ContactID,
		IdempotencyKey: input.CheckoutKey,
		Items:          []CommerceOrderItem{{SKU: input.SKU, Quantity: input.Quantity}},
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if repeated.ID != originalOrder.ID || !repeated.Duplicated {
		t.Fatalf("refazer após reinício precisa devolver o MESMO pedido: %+v", repeated)
	}
	if after := restarted.Commerce().Orders(input.OrganizationID); len(after) != 1 {
		t.Fatalf("um pedido apenas após reinício: %d", len(after))
	}
	// Repostar a receita depois do reinício também não duplica.
	posting, err := restarted.Finance().PostOrderRevenue(input.OrganizationID, originalOrder.ID, "certificacao", now)
	if err != nil {
		t.Fatal(err)
	}
	if posting.Created {
		t.Fatalf("a receita não pode ser lançada de novo após o reinício (evidência original %s)", originalEntry)
	}
	finance, err = restarted.Finance().Report(input.OrganizationID, time.Time{}, time.Time{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if finance.Ledger.IncomeCents != input.PriceCents*input.Quantity {
		t.Fatalf("receita duplicada após reinício: %+v", finance.Ledger)
	}
	// O consentimento também sobrevive: responder continua autorizado.
	if err := restarted.Consent().AuthorizeContact(input.OrganizationID, ConsentChannelWhatsApp, input.ContactPhone, now); err != nil {
		t.Fatalf("o consentimento precisa sobreviver ao reinício: %v", err)
	}
}
