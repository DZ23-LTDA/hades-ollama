package agent

import (
	"errors"
	"fmt"
	"time"
)

// ErrBusinessJourneyAlreadyCertified recusa certificar uma empresa que JÁ operou: a jornada é uma cerimônia de
// empresa nova e reexecutá-la não é suportado (E-305).
var ErrBusinessJourneyAlreadyCertified = errors.New("business journey requires a fresh company")

// Business E2E Certification (estágio 38): percorre a jornada completa de uma empresa
// de sandbox e devolve **evidência por etapa**, com prova de isolamento e a lista
// explícita do que NÃO foi feito.
//
// DECISÕES DELIBERADAS:
//   - **CADA ETAPA CARREGA A EVIDÊNCIA**: o relatório não diz "passou"; ele diz qual
//     identificador foi produzido e verificado por etapa. Um "OK" sem identificador
//     seria afirmação sem lastro;
//   - **A PROVA DE ISOLAMENTO É PARTE DO RELATÓRIO**: cada recurso criado para a
//     organização é LIDO como a outra organização, e a tentativa precisa falhar. O
//     veredito só é aprovado se **nenhum** vazamento aparecer;
//   - **O QUE NÃO FOI FEITO É DECLARADO**: marketplace, redes sociais, telefonia, PSP
//     real e fiscal aparecem como `SKIPPED_EXTERNAL` com o motivo — declarar é
//     diferente de omitir. Simulação **nunca** é chamada de produção;
//   - **ETAPA FALHA NÃO É ESCONDIDA**: erro vira etapa `FALHOU` com o motivo, e o
//     veredito é `FALHOU`. Não existe aprovação com etapa pulada em silêncio;
//   - **A JORNADA É IDEMPOTENTE**: refazer checkout e lançamento devolve os **mesmos**
//     identificadores, e a etapa prova isso. Sem isso, "certificar duas vezes" cobraria
//     duas vezes.

const (
	BusinessStepOK       = "OK"
	BusinessStepFailed   = "FALHOU"
	BusinessStepExternal = "SKIPPED_EXTERNAL"

	BusinessVerdictOK     = "SANDBOX_APROVADO"
	BusinessVerdictFailed = "FALHOU"
)

// BusinessJourneyStep é uma etapa da jornada com a evidência observada.
type BusinessJourneyStep struct {
	Step     string `json:"step"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
	Detail   string `json:"detail,omitempty"`
}

// BusinessIsolationCheck é a tentativa de leitura cruzada entre organizações.
type BusinessIsolationCheck struct {
	Resource string `json:"resource"`
	// VisibleToOtherTenant precisa ser false: se for true, houve vazamento.
	VisibleToOtherTenant bool   `json:"visible_to_other_tenant"`
	Detail               string `json:"detail,omitempty"`
}

// BusinessJourneyReport é o relatório de homologação em sandbox.
type BusinessJourneyReport struct {
	OrganizationID string                   `json:"organization_id"`
	CompanyID      string                   `json:"company_id"`
	StartedAt      time.Time                `json:"started_at"`
	FinishedAt     time.Time                `json:"finished_at"`
	Verdict        string                   `json:"verdict"`
	Steps          []BusinessJourneyStep    `json:"steps"`
	Isolation      []BusinessIsolationCheck `json:"isolation"`
	// IsolationProven é true somente se NENHUM recurso vazou.
	IsolationProven bool `json:"isolation_proven"`
	// RecoveryProven é true se refazer os passos idempotentes não duplicou nada.
	RecoveryProven bool `json:"recovery_proven"`
	// NotDone declara o que depende de conta, credencial ou provedor externo.
	NotDone []string `json:"not_done"`
	// Supervisor registra o ciclo de supervisão executado dentro da jornada: quantos
	// sinais foram observados, quantas ações aguardaram humano e qual foi decidido.
	Supervisor BusinessSupervisionEvidence `json:"supervisor"`
}

// BusinessSupervisionEvidence é a evidência do ciclo de supervisão dentro da jornada.
type BusinessSupervisionEvidence struct {
	Signals          int      `json:"signals"`
	AwaitingApproval int      `json:"awaiting_approval"`
	BlockedByBudget  int      `json:"blocked_by_budget"`
	HumanDecided     string   `json:"human_decided,omitempty"`
	DecidedBy        string   `json:"decided_by,omitempty"`
	LedgerEffect     string   `json:"ledger_effect,omitempty"`
	AutonomousEffect string   `json:"autonomous_effect,omitempty"`
	Pending          []string `json:"pending_kinds,omitempty"`
}

// BusinessJourneyInput reúne o que o passo a passo precisa criar.
type BusinessJourneyInput struct {
	OrganizationID string
	OtherOrgID     string
	CompanyName    string
	ContactName    string
	ContactPhone   string
	SKU            string
	ProductName    string
	PriceCents     int64
	InitialStock   int64
	Quantity       int64
	CheckoutKey    string
	StatementRef   string
}

// DefaultBusinessJourneyInput devolve uma jornada de sandbox determinística.
func DefaultBusinessJourneyInput(organizationID, otherOrgID string) BusinessJourneyInput {
	return BusinessJourneyInput{
		OrganizationID: organizationID,
		OtherOrgID:     otherOrgID,
		CompanyName:    "Empresa de Teste (sandbox)",
		ContactName:    "Ana Souza",
		ContactPhone:   "+5511999998888",
		SKU:            "CAM-001",
		ProductName:    "Camiseta",
		PriceCents:     7900,
		InitialStock:   10,
		Quantity:       2,
		CheckoutKey:    "certificacao-e2e-1",
		StatementRef:   "extrato-sandbox-1",
	}
}

// CertifyBusinessJourney executa a jornada completa e devolve o relatório.
func (r *Runtime) CertifyBusinessJourney(input BusinessJourneyInput, now time.Time) (BusinessJourneyReport, error) {
	report := BusinessJourneyReport{
		OrganizationID: consentScope(input.OrganizationID),
		StartedAt:      now,
		Verdict:        BusinessVerdictFailed,
		Steps:          []BusinessJourneyStep{},
		Isolation:      []BusinessIsolationCheck{},
	}
	if r == nil {
		return report, fmt.Errorf("runtime indisponível")
	}
	// Toda jornada declara de saída o que NÃO cobre: dependência externa não vira
	// "aprovado" por omissão.
	report.NotDone = []string{
		"marketplace e redes sociais: exigem conta autorizada e scopes válidos",
		"telefonia e voz: exigem número e provedor real",
		"pagamento real: não há PSP nem transferência de dinheiro",
		"fiscal (NF-e, impostos) e integração contábil",
		"canais externos de mensagem: exigem credencial do provedor",
	}
	step := func(name, status, evidence, detail string) {
		report.Steps = append(report.Steps, BusinessJourneyStep{Step: name, Status: status, Evidence: evidence, Detail: detail})
	}
	fail := func(name string, err error) (BusinessJourneyReport, error) {
		step(name, BusinessStepFailed, "", err.Error())
		report.FinishedAt = now
		report.Verdict = BusinessVerdictFailed
		return report, err
	}

	// 1. Empresa criada (Company Factory).
	company, err := r.CompanyStore().Create(Company{
		OrganizationID: input.OrganizationID, Name: input.CompanyName, Currency: CommerceDefaultCurrency,
	})
	if err != nil {
		return fail("empresa_criada", err)
	}
	report.CompanyID = company.ID
	step("empresa_criada", BusinessStepOK, "company="+company.ID, company.Name)

	// A JORNADA NÃO SUPORTA REEXECUÇÃO NA MESMA EMPRESA (E-305). Ela é uma cerimônia de empresa NOVA: cria
	// produto, carrega estoque e VENDE. Reexecutar numa empresa que já operou inflava estoque e depois falhava
	// com o erro interno "reserva inesperada" — mensagem confusa para quem opera. Recusar EXPLICITAMENTE é o
	// desfecho honesto: o defeito não se disfarça de falha obscura.
	if len(r.Commerce().Orders(input.OrganizationID)) > 0 {
		return fail("empresa_ja_operou", ErrBusinessJourneyAlreadyCertified)
	}

	// 2. Produto e estoque (Commerce Engine).
	product, err := r.Commerce().UpsertProduct(CommerceProduct{
		OrganizationID: input.OrganizationID, SKU: input.SKU, Name: input.ProductName,
		PriceCents: input.PriceCents, Active: true,
	}, now)
	if err != nil {
		return fail("produto_e_estoque", err)
	}
	// A CARGA INICIAL precisa ser IDEMPOTENTE (E-305). MEDIDO: reexecutar a jornada somava `InitialStock` de novo, e o
	// estoque ficava inflado (10 -> 18 no diagnóstico). Estoque inflado é inventário errado: o saldo deixa de
	// corresponder ao que existe. A carga só acontece quando o SKU ainda NÃO tem movimento nenhum — assim a primeira
	// execução é idêntica e qualquer reexecução (no mesmo processo OU após reinício) não soma estoque.
	if len(r.Commerce().Movements(input.OrganizationID, input.SKU)) == 0 {
		if _, err := r.Commerce().RecordStock(input.OrganizationID, input.SKU, input.InitialStock, CommerceMovementEntry, "carga inicial", now); err != nil {
			return fail("produto_e_estoque", err)
		}
	}
	step("produto_e_estoque", BusinessStepOK, "product="+product.ID,
		fmt.Sprintf("%s com %d unidades disponíveis", product.SKU, input.InitialStock))

	// 3. Contato no CRM.
	contact, created, err := r.CRM().UpsertContact(CRMContact{
		OrganizationID: input.OrganizationID, Name: input.ContactName, Phone: input.ContactPhone,
	}, now)
	if err != nil {
		return fail("contato_no_crm", err)
	}
	step("contato_no_crm", BusinessStepOK, "contact="+contact.ID, fmt.Sprintf("criado=%v", created))

	// 4. Consentimento do titular (obrigatório para responder).
	if _, err := r.Consent().GrantConsent(input.OrganizationID, ConsentChannelWhatsApp, input.ContactPhone, "checkout", "checkbox", "certificacao", nil, now); err != nil {
		return fail("consentimento_registrado", err)
	}
	step("consentimento_registrado", BusinessStepOK, "canal="+ConsentChannelWhatsApp, "opt-in registrado pelo titular")

	// 5. Mensagem recebida entra na caixa de entrada.
	conversation, _, err := r.Inbox().AppendInbound(input.OrganizationID, InboxChannelWhatsApp, input.ContactPhone, input.ContactName, "quero comprar a camiseta", "wamid.e2e.1", now)
	if err != nil {
		return fail("mensagem_recebida", err)
	}
	step("mensagem_recebida", BusinessStepOK, "conversation="+conversation.ID, "mensagem de entrada registrada")

	// 6. Agente comercial responde usando contexto real.
	turn, err := r.ServiceAgents().RunTurn(ServiceAgentTurnRequest{
		OrganizationID: input.OrganizationID, AgentID: ServiceAgentSales,
		ConversationID: conversation.ID, Intent: "quero comprar a camiseta", ModelID: "sandbox-deterministico",
	}, now)
	if err != nil {
		return fail("agente_respondeu", err)
	}
	if turn.Escalated {
		return fail("agente_respondeu", fmt.Errorf("o agente escalou em vez de responder: %s", turn.Escalation))
	}
	step("agente_respondeu", BusinessStepOK, "message="+turn.Message.ID,
		"procedência do texto: "+turn.ReplySource)

	// 7. Oportunidade registrada no CRM.
	opportunity, err := r.CRM().CreateOpportunity(input.OrganizationID, contact.ID, "Camiseta", input.PriceCents*input.Quantity, "agente-vendas", now)
	if err != nil {
		return fail("oportunidade_criada", err)
	}
	step("oportunidade_criada", BusinessStepOK, "opportunity="+opportunity.ID, opportunity.Title)

	// 8. Pedido com checkout IDEMPOTENTE.
	order, err := r.Commerce().CreateOrder(CommerceOrderRequest{
		OrganizationID: input.OrganizationID, ContactID: contact.ID, IdempotencyKey: input.CheckoutKey,
		Items: []CommerceOrderItem{{SKU: input.SKU, Quantity: input.Quantity}},
	}, now)
	if err != nil {
		return fail("pedido_criado", err)
	}
	step("pedido_criado", BusinessStepOK, "order="+order.ID, fmt.Sprintf("%s total %d centavos", order.Number, order.TotalCents))

	// 9. Reserva de estoque (não se vende o que não existe).
	balance := r.Commerce().Stock(input.OrganizationID, input.SKU)
	if balance.Available != input.InitialStock-input.Quantity || balance.OnHand != input.InitialStock {
		return fail("estoque_reservado", fmt.Errorf("reserva inesperada: %+v", balance))
	}
	step("estoque_reservado", BusinessStepOK, fmt.Sprintf("disponivel=%d", balance.Available),
		fmt.Sprintf("físico %d, reservado %d", balance.OnHand, balance.Reserved))

	// 10. Pagamento em SANDBOX, por referência (nenhum dado de cartão).
	if _, err := r.Commerce().ConfirmPayment(input.OrganizationID, order.ID, CommercePayment{
		Method: CommercePaymentSandbox, Reference: "sandbox-ref-1",
	}, "certificacao", now); err != nil {
		return fail("pagamento_sandbox", err)
	}
	step("pagamento_sandbox", BusinessStepOK, "method="+CommercePaymentSandbox, "quitação por referência, sem dado de cartão")

	// 11. Receita lançada no livro financeiro.
	posting, err := r.Finance().PostOrderRevenue(input.OrganizationID, order.ID, "certificacao", now)
	if err != nil {
		return fail("receita_lancada", err)
	}
	if posting.Entry.AmountCents != order.TotalCents {
		return fail("receita_lancada", fmt.Errorf("receita %d difere do pedido %d", posting.Entry.AmountCents, order.TotalCents))
	}
	step("receita_lancada", BusinessStepOK, "ledger="+posting.Entry.ID, fmt.Sprintf("%d centavos em %s", posting.Entry.AmountCents, posting.Entry.Currency))

	// 12. Conciliação com o extrato.
	reconciled, err := r.Finance().ReconcileOrderWithStatement(input.OrganizationID, order.ID, input.StatementRef, "certificacao", now)
	if err != nil {
		return fail("conciliacao", err)
	}
	step("conciliacao", BusinessStepOK, "ledger="+reconciled.ID, "referência do extrato: "+reconciled.ReconciledReference)

	// 13. Atendimento do pedido: o físico diminui.
	if _, err := r.Commerce().Fulfill(input.OrganizationID, order.ID, now); err != nil {
		return fail("pedido_atendido", err)
	}
	balance = r.Commerce().Stock(input.OrganizationID, input.SKU)
	if balance.OnHand != input.InitialStock-input.Quantity || balance.Reserved != 0 {
		return fail("pedido_atendido", fmt.Errorf("saldo após atendimento: %+v", balance))
	}
	step("pedido_atendido", BusinessStepOK, fmt.Sprintf("fisico=%d", balance.OnHand), "reserva convertida em saída")

	// 14. SLA medido na caixa de entrada.
	metrics := r.Inbox().Metrics(input.OrganizationID, now)
	if metrics.MessagesInbound == 0 || metrics.MessagesOutbound == 0 {
		return fail("sla_medido", fmt.Errorf("volume inesperado: %+v", metrics))
	}
	step("sla_medido", BusinessStepOK, fmt.Sprintf("aguardando_primeira_resposta=%d", metrics.AwaitingFirstReply),
		fmt.Sprintf("entrada %d, saída %d, violações %d", metrics.MessagesInbound, metrics.MessagesOutbound, metrics.SLABreached))

	// 15. Relatório financeiro limpo (sem divergência).
	financeReport, err := r.Finance().Report(input.OrganizationID, time.Time{}, time.Time{}, now)
	if err != nil {
		return fail("financeiro_consistente", err)
	}
	if len(financeReport.Divergences) != 0 || financeReport.UnpostedRevenue != 0 {
		return fail("financeiro_consistente", fmt.Errorf("divergências inesperadas: %+v", financeReport.Divergences))
	}
	step("financeiro_consistente", BusinessStepOK, fmt.Sprintf("receita=%d", financeReport.Ledger.IncomeCents),
		fmt.Sprintf("líquido %d, conciliados %d", financeReport.Ledger.NetCents, financeReport.Ledger.ReconciledCount))

	// 16. IDEMPOTÊNCIA: refazer checkout e receita devolve os MESMOS identificadores.
	repeated, err := r.Commerce().CreateOrder(CommerceOrderRequest{
		OrganizationID: input.OrganizationID, ContactID: contact.ID, IdempotencyKey: input.CheckoutKey,
		Items: []CommerceOrderItem{{SKU: input.SKU, Quantity: input.Quantity}},
	}, now)
	if err != nil {
		return fail("recuperacao_idempotente", err)
	}
	repeatPosting, err := r.Finance().PostOrderRevenue(input.OrganizationID, order.ID, "certificacao", now)
	if err != nil {
		return fail("recuperacao_idempotente", err)
	}
	if repeated.ID != order.ID || !repeated.Duplicated || repeatPosting.Created {
		return fail("recuperacao_idempotente", fmt.Errorf("refazer duplicou: order=%s duplicated=%v nova_receita=%v",
			repeated.ID, repeated.Duplicated, repeatPosting.Created))
	}
	if orders := r.Commerce().Orders(input.OrganizationID); len(orders) != 1 {
		return fail("recuperacao_idempotente", fmt.Errorf("pedidos após refazer: %d", len(orders)))
	}
	report.RecoveryProven = true
	step("recuperacao_idempotente", BusinessStepOK, "order="+repeated.ID, "refazer não duplicou pedido nem receita")

	// 17. PROVA DE ISOLAMENTO: a outra organização NÃO enxerga nada.
	other := consentScope(input.OtherOrgID)
	checkIsolation := func(resource string, visible bool, detail string) {
		report.Isolation = append(report.Isolation, BusinessIsolationCheck{Resource: resource, VisibleToOtherTenant: visible, Detail: detail})
	}
	if _, _, err := r.Inbox().Conversation(other, conversation.ID); err == nil {
		checkIsolation("conversa", true, "a outra organização leu a conversa")
	} else {
		checkIsolation("conversa", false, "recusado")
	}
	if _, err := r.Commerce().Order(other, order.ID); err == nil {
		checkIsolation("pedido", true, "a outra organização leu o pedido")
	} else {
		checkIsolation("pedido", false, "recusado")
	}
	if _, err := r.Finance().Report(other, time.Time{}, time.Time{}, now); err == nil {
		checkIsolation("livro_financeiro", true, "a outra organização leu o livro")
	} else {
		checkIsolation("livro_financeiro", false, "recusado")
	}
	for _, otherContact := range r.CRM().ListContacts(other, "") {
		if otherContact.ID == contact.ID {
			checkIsolation("contato", true, "a outra organização viu o contato")
		}
	}
	if len(report.Isolation) < 3 {
		report.Isolation = append(report.Isolation, BusinessIsolationCheck{Resource: "contato", VisibleToOtherTenant: false, Detail: "recusado"})
	}
	report.IsolationProven = true
	for _, check := range report.Isolation {
		if check.VisibleToOtherTenant {
			report.IsolationProven = false
		}
	}
	if !report.IsolationProven {
		step("isolamento_provado", BusinessStepFailed, "", "houve leitura cruzada entre organizações")
		report.FinishedAt = now
		report.Verdict = BusinessVerdictFailed
		return report, fmt.Errorf("isolamento entre organizações violado")
	}
	step("isolamento_provado", BusinessStepOK, fmt.Sprintf("verificacoes=%d", len(report.Isolation)),
		"nenhum recurso visível para a outra organização")

	// 18. CICLO DO SUPERVISOR: um segundo pedido é pago, tem receita lançada e é
	// cancelado — o que gera ESTORNO DEVIDO, que por política EXIGE humano.
	refundOrder, err := r.Commerce().CreateOrder(CommerceOrderRequest{
		OrganizationID: input.OrganizationID, ContactID: contact.ID,
		IdempotencyKey: input.CheckoutKey + "-estorno",
		Items:          []CommerceOrderItem{{SKU: input.SKU, Quantity: 1}},
	}, now)
	if err != nil {
		return fail("supervisor_ciclo", err)
	}
	if _, err := r.Commerce().ConfirmPayment(input.OrganizationID, refundOrder.ID, CommercePayment{
		Method: CommercePaymentSandbox, Reference: "sandbox-ref-estorno",
	}, "certificacao", now); err != nil {
		return fail("supervisor_ciclo", err)
	}
	if _, err := r.Finance().PostOrderRevenue(input.OrganizationID, refundOrder.ID, "certificacao", now); err != nil {
		return fail("supervisor_ciclo", err)
	}
	if _, err := r.Commerce().Cancel(input.OrganizationID, refundOrder.ID, "cliente desistiu", now); err != nil {
		return fail("supervisor_ciclo", err)
	}
	tick, err := r.BusinessSupervisor().BusinessTick(input.OrganizationID, now)
	if err != nil {
		return fail("supervisor_ciclo", err)
	}
	if tick.AwaitingApproval == 0 {
		return fail("supervisor_ciclo", fmt.Errorf("o estorno devido precisava aguardar decisão humana: %+v", tick.Actions))
	}
	report.Supervisor = BusinessSupervisionEvidence{
		Signals:          len(tick.Signals),
		AwaitingApproval: tick.AwaitingApproval,
		BlockedByBudget:  tick.BlockedByBudget,
	}
	pending := r.BusinessSupervisor().PendingActions(input.OrganizationID)
	if len(pending) == 0 {
		return fail("supervisor_ciclo", fmt.Errorf("a pendência precisa ser consultável"))
	}
	for _, action := range pending {
		report.Supervisor.Pending = append(report.Supervisor.Pending, action.Kind)
	}
	step("supervisor_ciclo", BusinessStepOK, fmt.Sprintf("tick=%d", tick.TickNumber),
		fmt.Sprintf("%d sinais, %d aguardando humano, %d bloqueadas por orçamento",
			len(tick.Signals), tick.AwaitingApproval, tick.BlockedByBudget))

	// 19. APROVAÇÃO HUMANA: a decisão de uma pessoa produz o efeito real no livro.
	var refundAction BusinessAction
	for _, action := range pending {
		if action.Kind == BusinessActionRefund {
			refundAction = action
		}
	}
	if refundAction.ID == "" {
		return fail("aprovacao_humana", fmt.Errorf("esperava ação de estorno: %+v", pending))
	}
	decided, err := r.BusinessSupervisor().Decide(input.OrganizationID, refundAction.ID, "APROVAR", "gerente-certificacao", "cliente pediu a devolução", now)
	if err != nil {
		return fail("aprovacao_humana", err)
	}
	if decided.Status != BusinessActionExecuted || decided.DecidedBy != "gerente-certificacao" {
		return fail("aprovacao_humana", fmt.Errorf("a aprovação precisa executar e registrar o ator: %+v", decided))
	}
	report.Supervisor.HumanDecided = refundAction.Kind
	report.Supervisor.DecidedBy = decided.DecidedBy
	report.Supervisor.LedgerEffect = decided.Effect
	step("aprovacao_humana", BusinessStepOK, "action="+refundAction.ID,
		"decisão de gerente-certificacao: "+decided.Effect)

	// 20. ISOLAMENTO DO SUPERVISOR: a outra organização não vê a pendência.
	if pendingOther := r.BusinessSupervisor().PendingActions(other); len(pendingOther) != 0 {
		checkIsolation("supervisor_pendencia", true, "a outra organização viu a pendência de decisão")
	} else {
		checkIsolation("supervisor_pendencia", false, "recusado")
	}
	for _, check := range report.Isolation {
		if check.VisibleToOtherTenant {
			report.IsolationProven = false
		}
	}
	if !report.IsolationProven {
		return fail("supervisor_isolamento", fmt.Errorf("isolamento violado: %+v", report.Isolation))
	}
	step("supervisor_isolamento", BusinessStepOK, fmt.Sprintf("verificacoes=%d", len(report.Isolation)),
		"a pendência de decisão não vaza entre organizações")

	// 21. Declaração do que NÃO foi feito (dependência externa).
	step("externo_pendente_declarado", BusinessStepExternal, fmt.Sprintf("itens=%d", len(report.NotDone)),
		"marketplace, redes sociais, telefonia, PSP real, fiscal e canais externos")

	report.FinishedAt = now
	report.Verdict = BusinessVerdictOK
	return report, nil
}
