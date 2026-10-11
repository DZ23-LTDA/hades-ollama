package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Razão financeiro e conciliação (estágio 30). Toda rota passa por
// `companyForRequest`, que já verifica a ORGANIZAÇÃO: um lançamento de uma empresa
// não pode ser lido nem conciliado por outra.
const ledgerPeriodLayout = "2006-01-02"

func ledgerPeriod(c *gin.Context) (time.Time, time.Time, error) {
	parse := func(key string) (time.Time, error) {
		raw := strings.TrimSpace(c.Query(key))
		if raw == "" {
			return time.Time{}, nil
		}
		return time.Parse(ledgerPeriodLayout, raw)
	}
	from, err := parse("from")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := parse("to")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	// O limite superior é o fim do dia informado: sem isso, pedir "até hoje"
	// esconderia os lançamentos de hoje depois da meia-noite.
	if !to.IsZero() {
		to = to.Add(24*time.Hour - time.Nanosecond)
	}
	return from, to, nil
}

// recordLedgerEntry registra um lançamento. Com a mesma chave de idempotência,
// devolve o lançamento ORIGINAL (200) em vez de criar outro (201).
func (a *agentAPI) recordLedgerEntry(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input struct {
		Kind           string `json:"kind"`
		Category       string `json:"category"`
		AmountCents    int64  `json:"amount_cents"`
		Currency       string `json:"currency"`
		Description    string `json:"description"`
		Reference      string `json:"reference"`
		IdempotencyKey string `json:"idempotency_key"`
		OccurredAt     string `json:"occurred_at"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	entry := agent.LedgerEntry{
		CompanyID:   company.ID,
		Kind:        input.Kind,
		Category:    input.Category,
		AmountCents: input.AmountCents,
		Currency:    input.Currency,
		Description: input.Description,
		Reference:   input.Reference,
		Idempotency: input.IdempotencyKey,
		CreatedBy:   agentActorID(c),
	}
	if raw := strings.TrimSpace(input.OccurredAt); raw != "" {
		occurredAt, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeAgentError(c, http.StatusBadRequest, err)
			return
		}
		entry.OccurredAt = occurredAt
	}
	recorded, created, err := a.runtime.CompanyStore().RecordLedgerEntry(entry, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"entry": recorded, "created": created})
}

// listLedgerEntries devolve os lançamentos e o resumo de fluxo de caixa.
func (a *agentAPI) listLedgerEntries(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	from, to, err := ledgerPeriod(c)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	store := a.runtime.CompanyStore()
	entries, err := store.ListLedgerEntries(company.ID, from, to)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	summary, err := store.LedgerSummary(company.ID, from, to)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries, "count": len(entries), "summary": summary})
}

// ledgerSummary devolve indicadores e a fila do que ainda não foi conciliado.
func (a *agentAPI) ledgerSummary(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	from, to, err := ledgerPeriod(c)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	olderThanDays := 30
	if raw := strings.TrimSpace(c.Query("older_than_days")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeAgentError(c, http.StatusBadRequest, err)
			return
		}
		olderThanDays = parsed
	}
	store := a.runtime.CompanyStore()
	summary, err := store.LedgerSummary(company.ID, from, to)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	pending, err := store.UnreconciledLedgerEntries(company.ID, olderThanDays, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": summary, "pending_reconciliation": pending, "pending_count": len(pending)})
}

// reconcileLedgerEntry concilia com uma referência externa. Repetir a MESMA
// referência é no-op (`changed=false`); referência diferente é conflito.
func (a *agentAPI) reconcileLedgerEntry(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input struct {
		Reference string `json:"reference"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	entry, changed, err := a.runtime.CompanyStore().ReconcileLedgerEntry(company.ID, c.Param("entry_id"), input.Reference, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entry": entry, "changed": changed})
}
