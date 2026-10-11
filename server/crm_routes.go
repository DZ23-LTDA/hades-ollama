package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// errCRMUnavailable é o erro explícito quando o runtime não tem CRM: melhor
// dizer que não está disponível do que devolver lista vazia como se não houvesse
// contato.
var errCRMUnavailable = errors.New("CRM não está disponível no runtime")

// CRM de contatos, leads e oportunidades (estágios 27 e 35) e o comando `/crm`.
// Toda rota é escopada pela organização da sessão: contato de uma organização não é
// lido nem alterado por outra.
func (a *agentAPI) crmStore() *agent.CRMStore {
	if a.runtime == nil {
		return nil
	}
	return a.runtime.CRM()
}

func (a *agentAPI) listCRMContacts(c *gin.Context) {
	store := a.crmStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCRMUnavailable)
		return
	}
	contacts := store.ListContacts(agentOrganizationID(c), c.Query("q"))
	c.JSON(http.StatusOK, gin.H{"contacts": contacts, "count": len(contacts)})
}

// upsertCRMContact cria ou atualiza. A identidade normalizada (telefone/e-mail)
// evita duplicar o mesmo cliente: 201 na criação e 200 na atualização.
func (a *agentAPI) upsertCRMContact(c *gin.Context) {
	store := a.crmStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCRMUnavailable)
		return
	}
	var input struct {
		Name    string   `json:"name"`
		Phone   string   `json:"phone"`
		Email   string   `json:"email"`
		Tags    []string `json:"tags"`
		Notes   string   `json:"notes"`
		OwnerID string   `json:"owner_id"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	contact, created, err := store.UpsertContact(agent.CRMContact{
		OrganizationID: agentOrganizationID(c),
		Name:           input.Name,
		Phone:          input.Phone,
		Email:          input.Email,
		Tags:           input.Tags,
		Notes:          input.Notes,
		OwnerID:        input.OwnerID,
	}, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"contact": contact, "created": created})
}

// getCRMContact devolve o contato com o consentimento REAL por canal, lido do
// registro próprio — o CRM não guarda cópia do opt-out.
func (a *agentAPI) getCRMContact(c *gin.Context) {
	store := a.crmStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCRMUnavailable)
		return
	}
	var registry *agent.ConsentRegistry
	if a.runtime != nil {
		registry = a.runtime.Consent()
	}
	view, err := store.ContactView(agentOrganizationID(c), c.Param("id"), registry)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	opportunities := store.ListOpportunities(agentOrganizationID(c), "")
	own := make([]agent.CRMOpportunity, 0)
	for _, opportunity := range opportunities {
		if opportunity.ContactID == view.Contact.ID {
			own = append(own, opportunity)
		}
	}
	c.JSON(http.StatusOK, gin.H{"contact": view.Contact, "consent": view.Consent, "blocked_channels": view.Blocked, "opportunities": own})
}

func (a *agentAPI) createCRMOpportunity(c *gin.Context) {
	store := a.crmStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCRMUnavailable)
		return
	}
	var input struct {
		Title      string `json:"title"`
		ValueCents int64  `json:"value_cents"`
		OwnerID    string `json:"owner_id"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	opportunity, err := store.CreateOpportunity(agentOrganizationID(c), c.Param("id"), input.Title, input.ValueCents, input.OwnerID, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, opportunity)
}

func (a *agentAPI) setCRMOpportunityStage(c *gin.Context) {
	store := a.crmStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCRMUnavailable)
		return
	}
	var input struct {
		Stage string `json:"stage"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	opportunity, err := store.SetOpportunityStage(agentOrganizationID(c), c.Param("id"), input.Stage, agentActorID(c), time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, opportunity)
}

func (a *agentAPI) crmPipeline(c *gin.Context) {
	store := a.crmStore()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCRMUnavailable)
		return
	}
	c.JSON(http.StatusOK, store.PipelineSummary(agentOrganizationID(c)))
}

// grantCRMContactConsent registra consentimento pelo próprio CRM. Fecha o limite
// declarado em E-101: antes o registro existia mas não tinha rota.
func (a *agentAPI) grantCRMContactConsent(c *gin.Context) {
	store := a.crmStore()
	if store == nil || a.runtime == nil || a.runtime.Consent() == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCRMUnavailable)
		return
	}
	contact, err := store.GetContact(agentOrganizationID(c), c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input struct {
		Channel  string `json:"channel"`
		Source   string `json:"source"`
		Evidence string `json:"evidence"`
		// Contact é o identificador usado pelo canal (telefone ou chat id).
		Contact string `json:"contact"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	identifier := strings.TrimSpace(input.Contact)
	if identifier == "" {
		identifier = firstNonEmpty(contact.Phone, contact.Email)
	}
	record, err := a.runtime.Consent().GrantConsent(contact.OrganizationID, input.Channel, identifier, input.Source, input.Evidence, agentActorID(c), nil, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, record)
}

// revokeCRMContactConsent registra o opt-out pelo CRM. Exige motivo.
func (a *agentAPI) revokeCRMContactConsent(c *gin.Context) {
	store := a.crmStore()
	if store == nil || a.runtime == nil || a.runtime.Consent() == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCRMUnavailable)
		return
	}
	contact, err := store.GetContact(agentOrganizationID(c), c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input struct {
		Channel string `json:"channel"`
		Reason  string `json:"reason"`
		Contact string `json:"contact"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	identifier := strings.TrimSpace(input.Contact)
	if identifier == "" {
		identifier = firstNonEmpty(contact.Phone, contact.Email)
	}
	record, err := a.runtime.Consent().RevokeConsent(contact.OrganizationID, input.Channel, identifier, input.Reason, agentActorID(c), time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, record)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
