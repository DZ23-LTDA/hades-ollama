package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Propostas comerciais e pós-venda (estágio 27) e a superfície do comando `/sell`.
//
// Toda rota é escopada pela organização da sessão: proposta ou acompanhamento de uma
// organização não é lido nem alterado por outra. O preço NUNCA vem do corpo da requisição —
// a rota recebe apenas SKU e quantidade, e o preço é o do catálogo, recalculado no servidor.

var errSalesUnavailable = errors.New("força de vendas não está disponível no runtime")

func (a *agentAPI) salesStore() *agent.SalesWorkforceStore {
	if a.runtime == nil {
		return nil
	}
	return a.runtime.SalesWorkforce()
}

// createSalesProposal aceita SKU e quantidade. Preço NÃO é aceito de propósito: aceitá-lo
// permitiria proposta com valor inventado.
func (a *agentAPI) createSalesProposal(c *gin.Context) {
	store := a.salesStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errSalesUnavailable)
		return
	}
	var input struct {
		ContactID     string `json:"contact_id"`
		OpportunityID string `json:"opportunity_id"`
		Currency      string `json:"currency"`
		Notes         string `json:"notes"`
		ValidityHours int    `json:"validity_hours"`
		Items         []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
			// UnitPriceCents existe apenas para ser IGNORADO: quem tentar mandar preço não
			// muda o resultado, e o teste garante isso.
			UnitPriceCents int64 `json:"unit_price_cents"`
		} `json:"items"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	items := make([]agent.SalesProposalItemRequest, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, agent.SalesProposalItemRequest{SKU: item.SKU, Quantity: item.Quantity})
	}
	validity := time.Duration(0)
	if input.ValidityHours > 0 {
		validity = time.Duration(input.ValidityHours) * time.Hour
	}
	proposal, err := store.CreateProposal(agent.SalesProposalRequest{
		OrganizationID: agentOrganizationID(c),
		ContactID:      input.ContactID,
		OpportunityID:  input.OpportunityID,
		Currency:       input.Currency,
		Notes:          input.Notes,
		Validity:       validity,
		Items:          items,
	}, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, proposal)
}

func (a *agentAPI) listSalesProposals(c *gin.Context) {
	store := a.salesStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errSalesUnavailable)
		return
	}
	proposals := store.Proposals(agentOrganizationID(c), c.Query("status"))
	c.JSON(http.StatusOK, gin.H{"proposals": proposals, "count": len(proposals)})
}

func (a *agentAPI) getSalesProposal(c *gin.Context) {
	store := a.salesStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errSalesUnavailable)
		return
	}
	proposal, err := store.Proposal(agentOrganizationID(c), c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, proposal)
}

// sendSalesProposal registra o envio. Sem consentimento do contato a proposta NÃO sai.
func (a *agentAPI) sendSalesProposal(c *gin.Context) {
	store := a.salesStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errSalesUnavailable)
		return
	}
	var input struct {
		Channel string `json:"channel"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	proposal, err := store.SendProposal(agentOrganizationID(c), c.Param("id"), input.Channel, "api", time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, proposal)
}

func (a *agentAPI) decideSalesProposal(c *gin.Context) {
	store := a.salesStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errSalesUnavailable)
		return
	}
	var input struct {
		Decision string `json:"decision"`
		Actor    string `json:"actor"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	proposal, err := store.DecideProposal(agentOrganizationID(c), c.Param("id"), input.Decision, input.Actor, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, proposal)
}

func (a *agentAPI) createSalesFollowUp(c *gin.Context) {
	store := a.salesStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errSalesUnavailable)
		return
	}
	var input struct {
		ContactID string `json:"contact_id"`
		OrderID   string `json:"order_id"`
		Reason    string `json:"reason"`
		DueHours  int    `json:"due_hours"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	due := time.Duration(0)
	if input.DueHours > 0 {
		due = time.Duration(input.DueHours) * time.Hour
	}
	followUp, err := store.ScheduleFollowUp(agentOrganizationID(c), input.ContactID, input.OrderID, input.Reason, due, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, followUp)
}

// listSalesFollowUps aceita `?overdue=true` para devolver SÓ o que passou do prazo: o atraso
// precisa ser consultável, não apenas calculado.
func (a *agentAPI) listSalesFollowUps(c *gin.Context) {
	store := a.salesStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errSalesUnavailable)
		return
	}
	overdueOnly, _ := strconv.ParseBool(strings.TrimSpace(c.Query("overdue")))
	followUps := store.FollowUps(agentOrganizationID(c), overdueOnly, time.Now().UTC())
	c.JSON(http.StatusOK, gin.H{"follow_ups": followUps, "count": len(followUps)})
}

func (a *agentAPI) completeSalesFollowUp(c *gin.Context) {
	store := a.salesStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errSalesUnavailable)
		return
	}
	var input struct {
		Outcome string `json:"outcome"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	followUp, err := store.CompleteFollowUp(agentOrganizationID(c), c.Param("id"), "api", input.Outcome, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, followUp)
}
