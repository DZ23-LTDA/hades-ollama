package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Autonomous Business Supervisor (estágio 31): coordena os domínios de negócio sob
// LIMITES, ORÇAMENTO e APROVAÇÃO HUMANA.
//
// O supervisor de missões já existia (tick, journal, ocorrências, aprovação de risco).
// Este é o supervisor de **negócio**: ele lê o que os outros módulos já produzem —
// caixa de entrada (SLA), comércio (estoque e pedidos) e financeiro (receita e
// divergências) — e age sob política declarada. Ele **não** duplica o journal de
// ocorrências: registra nos mesmos registros de ocorrência que o operador já consulta.
//
// DECISÕES DELIBERADAS:
//   - **AÇÃO IRREVERSÍVEL NÃO É EXECUTADA SOZINHO**: mexer em dinheiro (estorno) ou
//     desfazer uma venda (cancelar pedido) fica **aguardando aprovação humana**, com
//     motivo e custo à vista. O supervisor não decide isso;
//   - **AÇÃO REVERSÍVEL E CONTÁBIL É AUTÔNOMA**: lançar receita de pedido já pago é
//     registro idempotente (não move dinheiro) e repor estoque é ajuste reversível —
//     esses podem ser autônomos, e o efeito real é registrado;
//   - **ORÇAMENTO É LIMITE, NÃO SUGESTÃO**: quando o custo da ação estoura o orçamento,
//     ela **não executa**, vira `BLOQUEADA_ORCAMENTO` e gera incidente. Estourar
//     orçamento em silêncio é o defeito que o estágio aponta;
//   - **APROVAR EXECUTA, E SÓ UMA VEZ**: a aprovação é idempotente; aprovar de novo
//     devolve a mesma decisão sem repetir o efeito externo;
//   - **O QUE NÃO SEI EXECUTAR É RECUSADO**: tipo de ação desconhecido falha explícito,
//     em vez de "concluir" sem efeito;
//   - **RECUPERAÇÃO É OBSERVADA**: quando o sinal volta ao normal, o incidente é
//     resolvido automaticamente — como no supervisor de missões.

const (
	// Classe de risco da ação.
	BusinessActionAutonomous = "AUTONOMA"
	BusinessActionApproval   = "REQUER_APROVACAO"

	// Estados da ação.
	BusinessActionExecuted   = "EXECUTADA"
	BusinessActionPending    = "AGUARDANDO_APROVACAO"
	BusinessActionDenied     = "RECUSADA"
	BusinessActionBudgetStop = "BLOQUEADA_ORCAMENTO"

	// Tipos de ação.
	BusinessActionPostRevenue = "lancar_receita"
	BusinessActionReplenish   = "repor_estoque"
	BusinessActionAssignQueue = "atribuir_fila_humana"
	BusinessActionRefund      = "estornar_pagamento"
	BusinessActionCancelOrder = "cancelar_pedido"

	// Severidade dos achados.
	BusinessSeverityHigh   = "ALTO"
	BusinessSeverityMedium = "MEDIO"
)

var (
	ErrBusinessActionNotFound = errors.New("business action not found")
	ErrBusinessActionDecided  = errors.New("business action already decided")
	ErrBusinessActionUnknown  = errors.New("business action kind is not executable")
	ErrBusinessBudgetExceeded = errors.New("action cost exceeds the remaining budget")
)

// SupervisorPolicy são os LIMITES declarados da operação.
type SupervisorPolicy struct {
	// MaxSLABreaches é quantas conversas podem estar com SLA violado.
	MaxSLABreaches int `json:"max_sla_breaches"`
	// MinAvailableUnits é o mínimo disponível por SKU antes da reposição.
	MinAvailableUnits int64 `json:"min_available_units"`
	// ReplenishUnits é quanto repor quando o mínimo é rompido.
	ReplenishUnits int64 `json:"replenish_units"`
	// ReplenishUnitCostCents é o custo unitário da reposição (entra no orçamento).
	ReplenishUnitCostCents int64 `json:"replenish_unit_cost_cents"`
	// MaxUnpostedRevenueCents é quanto de receita pode ficar sem lançamento.
	MaxUnpostedRevenueCents int64 `json:"max_unposted_revenue_cents"`
	// MaxRefundDueCents é quanto de estorno pode ficar pendente sem decisão humana.
	MaxRefundDueCents int64 `json:"max_refund_due_cents"`
	// MaxDLQDepth é a profundidade aceitável da fila de mortos de canais.
	MaxDLQDepth int `json:"max_dlq_depth"`
	// BudgetCents é o orçamento disponível para ações autônomas.
	BudgetCents int64 `json:"budget_cents"`
}

// DefaultSupervisorPolicy devolve limites conservadores de operação.
func DefaultSupervisorPolicy() SupervisorPolicy {
	return SupervisorPolicy{
		MaxSLABreaches:          0,
		MinAvailableUnits:       2,
		ReplenishUnits:          10,
		ReplenishUnitCostCents:  3500,
		MaxUnpostedRevenueCents: 0,
		MaxRefundDueCents:       0,
		MaxDLQDepth:             5,
		BudgetCents:             100000,
	}
}

// BusinessSignal é um indicador observado com o limite que ele rompeu.
type BusinessSignal struct {
	Domain   string `json:"domain"`
	Name     string `json:"name"`
	Value    int64  `json:"value"`
	Limit    int64  `json:"limit"`
	Severity string `json:"severity"`
	Detail   string `json:"detail,omitempty"`
}

// BusinessAction é uma ação proposta, executada, bloqueada ou aguardando decisão.
type BusinessAction struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	Kind           string     `json:"kind"`
	Subject        string     `json:"subject"`
	Reason         string     `json:"reason"`
	Risk           string     `json:"risk"`
	CostCents      int64      `json:"cost_cents"`
	Status         string     `json:"status"`
	Effect         string     `json:"effect,omitempty"`
	DecidedBy      string     `json:"decided_by,omitempty"`
	DecidedAt      *time.Time `json:"decided_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// BusinessTickResult é o resultado de um ciclo de supervisão de negócio.
type BusinessTickResult struct {
	TickNumber        int64            `json:"tick_number"`
	OrganizationID    string           `json:"organization_id"`
	ExecutedAt        time.Time        `json:"executed_at"`
	Signals           []BusinessSignal `json:"signals"`
	Actions           []BusinessAction `json:"actions"`
	BudgetSpentCents  int64            `json:"budget_spent_cents"`
	BudgetLeftCents   int64            `json:"budget_left_cents"`
	AutonomousCount   int              `json:"autonomous_executed"`
	AwaitingApproval  int              `json:"awaiting_approval"`
	BlockedByBudget   int              `json:"blocked_by_budget"`
	IncidentsRecorded int              `json:"incidents_recorded"`
	Errors            []string         `json:"errors,omitempty"`
}

// businessSupervisorFile é o que sobrevive ao reinício: decisões pendentes, política,
// orçamento consumido e a contagem de ciclos.
type businessSupervisorFile struct {
	Policy  SupervisorPolicy          `json:"policy"`
	Ticks   int64                     `json:"ticks"`
	Actions map[string]BusinessAction `json:"actions"`
	Spent   map[string]int64          `json:"spent"`
	// History e' o DIARIO dos ciclos (E-275). Entra no MESMO arquivo de estado: o diario nao e' um cidadao de
	// segunda classe que ganha arquivo proprio e pode ficar fora de sincronia com o resto.
	History []BusinessTickResult `json:"history,omitempty"`
}

// BusinessSupervisor coordena os domínios sob política.
type BusinessSupervisor struct {
	mu      sync.Mutex
	runtime *Runtime
	policy  SupervisorPolicy
	ticks   int64
	actions map[string]BusinessAction
	history []BusinessTickResult
	spent   map[string]int64
	// root vazio mantém o supervisor em memória.
	root string
}

// NewBusinessSupervisor cria o supervisor de negócio EM MEMÓRIA.
func NewBusinessSupervisor(runtime *Runtime, policy SupervisorPolicy) *BusinessSupervisor {
	return &BusinessSupervisor{
		runtime: runtime,
		policy:  policy,
		actions: map[string]BusinessAction{},
		spent:   map[string]int64{},
	}
}

// NewBusinessSupervisorAt cria o supervisor PERSISTENTE e carrega o que já existe.
//
// Sem persistência, um reinício perderia a **decisão pendente** — e uma ação
// irreversível que esperava aprovação humana voltaria a ser proposta como se nunca
// tivesse sido vista, ou pior, sumiria do radar do operador.
func NewBusinessSupervisorAt(root string, runtime *Runtime, policy SupervisorPolicy) (*BusinessSupervisor, error) {
	supervisor := NewBusinessSupervisor(runtime, policy)
	supervisor.root = strings.TrimSpace(root)
	if supervisor.root == "" {
		return supervisor, nil
	}
	if err := os.MkdirAll(supervisor.root, 0o700); err != nil {
		return nil, err
	}
	stored := businessSupervisorFile{}
	if err := readJSON(supervisor.path(), &stored); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("load business supervisor: %w", err)
		}
		return supervisor, nil
	}
	// A política persistida é a vigente: limites não se perdem no reinício.
	if stored.Policy.BudgetCents != 0 || stored.Policy.MinAvailableUnits != 0 {
		supervisor.policy = stored.Policy
	}
	supervisor.ticks = stored.Ticks
	// O diario volta TRUNCADO pelo cap, mantendo os MAIS RECENTES: guardar os mais ANTIGOS
	// esconderia justamente o que aconteceu por ultimo.
	supervisor.history = append([]BusinessTickResult(nil), stored.History...)
	if len(supervisor.history) > MaxSupervisorTickJournal {
		supervisor.history = supervisor.history[len(supervisor.history)-MaxSupervisorTickJournal:]
	}
	for id, action := range stored.Actions {
		supervisor.actions[id] = action
	}
	for organizationID, value := range stored.Spent {
		supervisor.spent[organizationID] = value
	}
	return supervisor, nil
}

// path é o arquivo único do supervisor: as decisões são poucas e a leitura é sempre
// completa, então um arquivo basta e mantém a auditoria num lugar só.
func (b *BusinessSupervisor) path() string {
	if b.root == "" {
		return ""
	}
	return filepath.Join(b.root, "business-supervisor.json")
}

// persistLocked grava política, decisões, orçamento e contagem de ciclos.
func (b *BusinessSupervisor) persistLocked() error {
	path := b.path()
	if path == "" {
		return nil
	}
	payload := businessSupervisorFile{
		Policy:  b.policy,
		Ticks:   b.ticks,
		Actions: map[string]BusinessAction{},
		Spent:   map[string]int64{},
	}
	for id, action := range b.actions {
		payload.Actions[id] = action
	}
	for organizationID, value := range b.spent {
		payload.Spent[organizationID] = value
	}
	payload.History = append([]BusinessTickResult(nil), b.history...)
	return writeJSONAtomic(path, payload)
}

// Policy devolve os limites vigentes.
func (b *BusinessSupervisor) Policy() SupervisorPolicy {
	if b == nil {
		return SupervisorPolicy{}
	}
	return b.policy
}

// SetPolicy troca os limites declarados.
func (b *BusinessSupervisor) SetPolicy(policy SupervisorPolicy) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.policy = policy
	_ = b.persistLocked()
}

// PendingActions lista as ações aguardando decisão humana.
func (b *BusinessSupervisor) PendingActions(organizationID string) []BusinessAction {
	organizationID = consentScope(organizationID)
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]BusinessAction, 0)
	for _, action := range b.actions {
		if action.OrganizationID == organizationID && action.Status == BusinessActionPending {
			out = append(out, action)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// History devolve os últimos ciclos.
func (b *BusinessSupervisor) History() []BusinessTickResult {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]BusinessTickResult(nil), b.history...)
}

// BusinessTick executa um ciclo de supervisão de negócio.
func (b *BusinessSupervisor) BusinessTick(organizationID string, now time.Time) (BusinessTickResult, error) {
	if b == nil || b.runtime == nil {
		return BusinessTickResult{}, fmt.Errorf("supervisor de negócio indisponível")
	}
	organizationID = consentScope(organizationID)
	result := BusinessTickResult{OrganizationID: organizationID, ExecutedAt: now}
	b.mu.Lock()
	b.ticks++
	result.TickNumber = b.ticks
	if err := b.persistLocked(); err != nil {
		result.Errors = append(result.Errors, "persistencia: "+err.Error())
	}
	b.mu.Unlock()

	// 1. SINAIS: o supervisor observa o que os módulos já produzem.
	inboxMetrics := b.runtime.Inbox().Metrics(organizationID, now)
	commerceSummary := b.runtime.Commerce().Summary(organizationID)
	financeReport, financeErr := b.runtime.Finance().Report(organizationID, time.Time{}, time.Time{}, now)
	if financeErr != nil {
		// O domínio financeiro pode estar ilegível (por exemplo, organização sem empresa
		// criada). Isso é DECLARADO no resultado, e a supervisão dos outros domínios
		// CONTINUA: derrubar o ciclo inteiro esconderia os achados que já existem.
		result.Errors = append(result.Errors, "financeiro: "+financeErr.Error())
	}
	dlqDepth := int64(len(b.runtime.Omnichannel().DeadLetters(organizationID)))

	signal := func(domain, name string, value, limit int64, severity, detail string) {
		result.Signals = append(result.Signals, BusinessSignal{Domain: domain, Name: name, Value: value, Limit: limit, Severity: severity, Detail: detail})
	}
	if int64(inboxMetrics.SLABreached) > int64(b.policy.MaxSLABreaches) {
		signal("atendimento", "sla_violado", int64(inboxMetrics.SLABreached), int64(b.policy.MaxSLABreaches),
			BusinessSeverityHigh, fmt.Sprintf("%d conversas sem primeira resposta no prazo", inboxMetrics.SLABreached))
	}
	if int64(inboxMetrics.PendingHuman) > 0 {
		signal("atendimento", "fila_humana", int64(inboxMetrics.PendingHuman), 0,
			BusinessSeverityMedium, fmt.Sprintf("%d conversas aguardando pessoa", inboxMetrics.PendingHuman))
	}
	// O limite é o da POLÍTICA do supervisor, não o do resumo do comércio (que só marca
	// saldo zerado). Usar o limiar do outro módulo faria a política nunca disparar.
	lowStock := []string{}
	for _, product := range b.runtime.Commerce().Products(organizationID) {
		balance := b.runtime.Commerce().Stock(organizationID, product.SKU)
		if balance.Available >= b.policy.MinAvailableUnits {
			continue
		}
		lowStock = append(lowStock, product.SKU)
		signal("comercio", "estoque_baixo_"+product.SKU, balance.Available, b.policy.MinAvailableUnits,
			BusinessSeverityMedium, fmt.Sprintf("%s com %d disponível", product.SKU, balance.Available))
	}
	if financeReport.UnpostedRevenue > b.policy.MaxUnpostedRevenueCents {
		signal("financeiro", "receita_nao_lancada", financeReport.UnpostedRevenue, b.policy.MaxUnpostedRevenueCents,
			BusinessSeverityHigh, "pedido pago sem receita lançada")
	}
	if financeReport.RefundDue > b.policy.MaxRefundDueCents {
		signal("financeiro", "estorno_devido", financeReport.RefundDue, b.policy.MaxRefundDueCents,
			BusinessSeverityHigh, "pedido cancelado com receita lançada")
	}
	if dlqDepth > int64(b.policy.MaxDLQDepth) {
		signal("canais", "fila_de_mortos", dlqDepth, int64(b.policy.MaxDLQDepth),
			BusinessSeverityHigh, "eventos de canal aguardando reprocessamento")
	}
	if commerceSummary.Pending > 0 {
		signal("comercio", "pedidos_aguardando_pagamento", int64(commerceSummary.Pending), 0,
			BusinessSeverityMedium, "pedidos aguardando pagamento")
	}

	// 2. ACHADOS PERSISTIDOS: os mesmos registros de ocorrência que o operador já consulta.
	if supervisor := b.runtime.Supervisor(); supervisor != nil {
		for _, finding := range result.Signals {
			if finding.Severity != BusinessSeverityHigh {
				continue
			}
			if err := supervisor.recordIncident("negocio_"+finding.Name, organizationID, finding.Detail, now); err != nil {
				result.Errors = append(result.Errors, "ocorrencia: "+err.Error())
				continue
			}
			result.IncidentsRecorded++
		}
		// RECUPERAÇÃO OBSERVADA: sinais altos que sumiram têm incidente resolvido.
		openKinds := map[string]bool{}
		for _, finding := range result.Signals {
			openKinds["negocio_"+finding.Name] = true
		}
		for _, incident := range supervisor.Incidents() {
			if !strings.HasPrefix(incident.Kind, "negocio_") || incident.State != SupervisorIncidentOpen {
				continue
			}
			if openKinds[incident.Kind] {
				continue
			}
			if _, err := supervisor.resolveIncident(incident.Kind, incident.Subject, "indicador voltou ao normal", now); err != nil {
				result.Errors = append(result.Errors, "recuperacao: "+err.Error())
			}
		}
	}

	// 3. AÇÕES: autônomas executam; irreversíveis aguardam aprovação.
	b.mu.Lock()
	spent := b.spent[organizationID]
	b.mu.Unlock()
	result.BudgetSpentCents = spent
	result.BudgetLeftCents = b.policy.BudgetCents - spent

	propose := func(kind, subject, reason, risk string, cost int64) {
		action := BusinessAction{
			ID:             "bact_" + uuid.NewString()[:12],
			OrganizationID: organizationID,
			Kind:           kind,
			Subject:        subject,
			Reason:         reason,
			Risk:           risk,
			CostCents:      cost,
			Status:         BusinessActionPending,
			CreatedAt:      now,
		}
		if risk == BusinessActionApproval {
			b.mu.Lock()
			b.actions[action.ID] = action
			// A PENDÊNCIA é gravada antes de sair: uma decisão que espera humano não pode
			// desaparecer num reinício.
			if err := b.persistLocked(); err != nil {
				result.Errors = append(result.Errors, "persistencia: "+err.Error())
			}
			b.mu.Unlock()
			result.Actions = append(result.Actions, action)
			result.AwaitingApproval++
			return
		}
		// ORÇAMENTO: limite duro para ação autônoma com custo.
		if cost > 0 && cost > result.BudgetLeftCents {
			action.Status = BusinessActionBudgetStop
			action.Effect = fmt.Sprintf("custo %d excede o orçamento restante %d", cost, result.BudgetLeftCents)
			b.mu.Lock()
			b.actions[action.ID] = action
			if err := b.persistLocked(); err != nil {
				result.Errors = append(result.Errors, "persistencia: "+err.Error())
			}
			b.mu.Unlock()
			result.Actions = append(result.Actions, action)
			result.BlockedByBudget++
			return
		}
		effect, err := b.execute(action)
		if err != nil {
			action.Status = BusinessActionBudgetStop
			action.Effect = "falha: " + err.Error()
			result.Errors = append(result.Errors, action.Effect)
			b.mu.Lock()
			b.actions[action.ID] = action
			b.mu.Unlock()
			result.Actions = append(result.Actions, action)
			return
		}
		decision := now
		action.Status = BusinessActionExecuted
		action.Effect = effect
		action.DecidedAt = &decision
		action.DecidedBy = "supervisor"
		if cost > 0 {
			b.mu.Lock()
			b.spent[organizationID] += cost
			b.mu.Unlock()
			result.BudgetSpentCents += cost
			result.BudgetLeftCents -= cost
		}
		b.mu.Lock()
		b.actions[action.ID] = action
		if err := b.persistLocked(); err != nil {
			result.Errors = append(result.Errors, "persistencia: "+err.Error())
		}
		b.mu.Unlock()
		result.Actions = append(result.Actions, action)
		result.AutonomousCount++
	}

	// Receita não lançada: registro contábil IDEMPOTENTE e sem mover dinheiro => autônomo.
	if financeReport.UnpostedRevenue > b.policy.MaxUnpostedRevenueCents {
		orders := b.runtime.Commerce().Orders(organizationID)
		posted := map[string]bool{}
		for _, order := range orders {
			if order.Status != CommerceOrderPaid && order.Status != CommerceOrderFulfilled {
				continue
			}
			if _, err := b.runtime.Finance().PostOrderRevenue(organizationID, order.ID, "supervisor", now); err == nil {
				posted[order.ID] = true
			}
		}
		if len(posted) > 0 {
			keys := make([]string, 0, len(posted))
			for id := range posted {
				keys = append(keys, id)
			}
			sort.Strings(keys)
			propose(BusinessActionPostRevenue, strings.Join(keys, ","),
				fmt.Sprintf("%d centavos de receita não lançada", financeReport.UnpostedRevenue),
				BusinessActionAutonomous, 0)
		}
	}
	// Estoque baixo: reposição é AJUSTE reversível => autônoma, com custo no orçamento.
	for _, sku := range lowStock {
		balance := b.runtime.Commerce().Stock(organizationID, sku)
		cost := b.policy.ReplenishUnits * b.policy.ReplenishUnitCostCents
		propose(BusinessActionReplenish, sku,
			fmt.Sprintf("%s com %d disponível (mínimo %d)", sku, balance.Available, b.policy.MinAvailableUnits),
			BusinessActionAutonomous, cost)
	}
	// Fila humana: atribuição é reversível => autônoma.
	for _, conversation := range b.runtime.Inbox().ListConversations(organizationID, InboxStatusPendingHuman) {
		if strings.TrimSpace(conversation.AssignedTo) != "" {
			continue
		}
		propose(BusinessActionAssignQueue, conversation.ID, "conversa aguardando pessoa sem responsável",
			BusinessActionAutonomous, 0)
	}
	// Estorno devido: MEXE EM DINHEIRO => exige aprovação humana.
	if financeReport.RefundDue > b.policy.MaxRefundDueCents {
		for _, divergence := range financeReport.Divergences {
			if divergence.Kind != FinanceDivergenceRefund {
				continue
			}
			propose(BusinessActionRefund, divergence.OrderID,
				fmt.Sprintf("pedido %s cancelado com receita lançada: %d centavos",
					divergence.OrderNumber, divergence.AmountCents),
				BusinessActionApproval, divergence.AmountCents)
		}
	}
	// Pedido aguardando pagamento: cancelar DESFAZ uma venda => exige aprovação humana.
	for _, order := range b.runtime.Commerce().Orders(organizationID) {
		if order.Status != CommerceOrderPending {
			continue
		}
		if now.Sub(order.CreatedAt) < 24*time.Hour {
			continue
		}
		propose(BusinessActionCancelOrder, order.ID, "pedido aguardando pagamento há mais de 24 horas",
			BusinessActionApproval, 0)
	}

	b.mu.Lock()
	b.history = append(b.history, result)
	if len(b.history) > MaxSupervisorTickJournal {
		b.history = b.history[len(b.history)-MaxSupervisorTickJournal:]
	}
	// DIARIO DURAVEL (E-275): sem esta escrita, um ciclo que nao propoe acao nao deixava rastro nenhum - o diario
	// so existia em memoria. Um trade-off e' declarado: ciclo COM acao pode escrever duas vezes (acao + diario).
	persistErr := b.persistLocked()
	b.mu.Unlock()
	if persistErr != nil {
		return result, persistErr
	}
	return result, nil
}

// execute aplica o EFEITO REAL de uma ação autônoma.
func (b *BusinessSupervisor) execute(action BusinessAction) (string, error) {
	switch action.Kind {
	case BusinessActionReplenish:
		// Reposição entra como movimento de estoque: fica no livro, auditável.
		movement, err := b.runtime.Commerce().RecordStock(action.OrganizationID, action.Subject,
			b.policy.ReplenishUnits, CommerceMovementEntry, "reposicao-automatica", action.CreatedAt)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("movimento %s de +%d em %s", movement.ID, movement.Delta, movement.SKU), nil
	case BusinessActionAssignQueue:
		conversation, err := b.runtime.Inbox().Assign(action.OrganizationID, action.Subject, "fila-humana", "human", action.CreatedAt)
		if err != nil {
			return "", err
		}
		return "conversa " + conversation.ID + " atribuída a " + conversation.AssignedTo, nil
	case BusinessActionPostRevenue:
		return "receita lançada para " + action.Subject, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrBusinessActionUnknown, action.Kind)
	}
}

// Decide aplica a decisão humana sobre uma ação que aguardava aprovação.
//
// Aprovar EXECUTA o efeito real; recusar não executa nada. Repetir a decisão devolve
// a mesma ação **sem repetir o efeito** (idempotente).
func (b *BusinessSupervisor) Decide(organizationID, actionID, decision, actorID, reason string, now time.Time) (BusinessAction, error) {
	organizationID = consentScope(organizationID)
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return BusinessAction{}, fmt.Errorf("aprovação exige ator identificado")
	}
	decision = strings.ToUpper(strings.TrimSpace(decision))
	if decision != "APROVAR" && decision != "RECUSAR" {
		return BusinessAction{}, fmt.Errorf("decisão precisa ser APROVAR ou RECUSAR")
	}
	b.mu.Lock()
	action, ok := b.actions[strings.TrimSpace(actionID)]
	if !ok || action.OrganizationID != organizationID {
		b.mu.Unlock()
		return BusinessAction{}, fmt.Errorf("%w: %s", ErrBusinessActionNotFound, actionID)
	}
	if action.Status != BusinessActionPending {
		// Já decidida: devolve a MESMA decisão, sem repetir o efeito.
		b.mu.Unlock()
		return action, nil
	}
	b.mu.Unlock()

	decided := now
	if decision == "RECUSAR" {
		action.Status = BusinessActionDenied
		action.Effect = "recusada por " + actorID + ": " + strings.TrimSpace(reason)
	} else {
		effect, err := b.executeApproved(action)
		if err != nil {
			return BusinessAction{}, err
		}
		action.Status = BusinessActionExecuted
		action.Effect = effect
	}
	action.DecidedBy = actorID
	action.DecidedAt = &decided

	b.mu.Lock()
	b.actions[action.ID] = action
	// A DECISÃO é gravada: aprovado/recusado não pode voltar a "pendente" num reinício.
	if err := b.persistLocked(); err != nil {
		b.mu.Unlock()
		return BusinessAction{}, err
	}
	b.mu.Unlock()
	return action, nil
}

// executeApproved aplica o efeito REAL de uma ação irreversível aprovada por humano.
//
// O estorno é registrado como DESPESA no livro, com idempotência por pedido: é o
// registro contábil da devolução. **Nenhum dinheiro é transferido** — o pagamento real
// continua exigindo o provedor, e isso é declarado, não simulado.
func (b *BusinessSupervisor) executeApproved(action BusinessAction) (string, error) {
	switch action.Kind {
	case BusinessActionRefund:
		order, err := b.runtime.Commerce().Order(action.OrganizationID, action.Subject)
		if err != nil {
			return "", err
		}
		company, err := b.runtime.Finance().companyForOrganization(action.OrganizationID)
		if err != nil {
			return "", err
		}
		entry, created, err := b.runtime.CompanyStore().RecordLedgerEntry(LedgerEntry{
			CompanyID:   company.ID,
			Kind:        LedgerKindExpense,
			Category:    "estorno",
			AmountCents: order.TotalCents,
			Currency:    order.Currency,
			Description: "estorno aprovado do pedido " + order.Number,
			Reference:   order.Number,
			Idempotency: "refund:" + order.ID,
			OccurredAt:  action.CreatedAt,
			CreatedBy:   action.DecidedBy,
		}, action.CreatedAt)
		if err != nil {
			return "", err
		}
		if !created {
			return "estorno já registrado (lançamento " + entry.ID + ")", nil
		}
		return "despesa de estorno registrada no lançamento " + entry.ID + " (sem transferência real de dinheiro)", nil
	case BusinessActionCancelOrder:
		order, err := b.runtime.Commerce().Cancel(action.OrganizationID, action.Subject, "aprovado pelo supervisor", action.CreatedAt)
		if err != nil {
			return "", err
		}
		return "pedido " + order.Number + " cancelado e estoque liberado", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrBusinessActionUnknown, action.Kind)
	}
}

// ErrSupervisorLeaseHeld diz que OUTRA instancia detem o lease: o ciclo NAO roda.
var ErrSupervisorLeaseHeld = errors.New("business supervisor lease is held by another instance")

// SupervisorLeaseTTLPadrao e' a validade do lease quando o chamador nao informa uma.
const SupervisorLeaseTTLPadrao = 30 * time.Second

// businessSupervisorLease e' o conteudo do arquivo de lease.
type businessSupervisorLease struct {
	Holder    string    `json:"holder"`
	ExpiresAt time.Time `json:"expires_at"`
}

// leasePath devolve o caminho do lease (vazio quando o supervisor esta em memoria).
func (b *BusinessSupervisor) leasePath() string {
	if strings.TrimSpace(b.root) == "" {
		return ""
	}
	return filepath.Join(b.root, "supervisor.lease")
}

// AcquireSupervisorLease tenta tomar o lease para `holder`. Devolve true quando o titular SOMOS nos.
//
// Tres situacoes concedem o lease: (1) nao existe lease; (2) o lease EXPIROU (assuncao - a instancia anterior pode
// ter morrido: negar para sempre travaria a operacao); (3) o lease ja e' NOSSO (renovacao). Enquanto outra
// instancia tiver lease vivo, a resposta e' `false` e o ciclo NAO roda.
func (b *BusinessSupervisor) AcquireSupervisorLease(holder string, now time.Time, ttl time.Duration) (bool, error) {
	holder = strings.TrimSpace(holder)
	if holder == "" {
		return false, errors.New("supervisor lease requires a holder")
	}
	caminho := b.leasePath()
	if caminho == "" {
		// Modo em memoria: um unico processo, um unico supervisor. Declarado, nao fingido.
		return true, nil
	}
	if ttl <= 0 {
		ttl = SupervisorLeaseTTLPadrao
	}
	atual := businessSupervisorLease{}
	err := readJSON(caminho, &atual)
	switch {
	case err == nil:
		vivo := atual.ExpiresAt.After(now.UTC())
		if vivo && strings.TrimSpace(atual.Holder) != holder {
			return false, nil
		}
	case errors.Is(err, os.ErrNotExist):
		// sem lease: pode tomar
	default:
		return false, fmt.Errorf("ler lease do supervisor: %w", err)
	}
	novo := businessSupervisorLease{Holder: holder, ExpiresAt: now.UTC().Add(ttl)}
	if err := writeJSONAtomic(caminho, novo); err != nil {
		return false, err
	}
	return true, nil
}

// ReleaseSupervisorLease remove o lease SOMENTE se ele for do titular informado. Quem nao e' titular nao libera.
func (b *BusinessSupervisor) ReleaseSupervisorLease(holder string) error {
	caminho := b.leasePath()
	if caminho == "" {
		return nil
	}
	atual := businessSupervisorLease{}
	if err := readJSON(caminho, &atual); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(atual.Holder) != strings.TrimSpace(holder) {
		return nil
	}
	if err := os.Remove(caminho); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// RunSupervisedTick roda o ciclo SOMENTE com o lease. Sem lease, o ciclo NAO acontece e o erro diz por que.
func (b *BusinessSupervisor) RunSupervisedTick(holder, organizationID string, now time.Time, ttl time.Duration) (BusinessTickResult, error) {
	obtido, err := b.AcquireSupervisorLease(holder, now, ttl)
	if err != nil {
		return BusinessTickResult{}, err
	}
	if !obtido {
		return BusinessTickResult{}, ErrSupervisorLeaseHeld
	}
	return b.BusinessTick(organizationID, now)
}
