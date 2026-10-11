package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Agendamento, métricas e painel de campanha (estágio 26).
//
// A capacidade existia apenas no runtime: sem superfície, agendar e medir campanha não era
// operável por quem usa o produto. Tudo aqui é escopado pela organização da sessão via
// `companyForRequest`, e NADA publica em plataforma externa.

// scheduleCompanyCampaign agenda uma campanha APROVADA. Aceita `start_at` em RFC 3339 ou
// `start_in_hours` como conveniência; as duas formas passam pelo mesmo domínio, que exige
// aprovação e início no futuro.
func (a *agentAPI) scheduleCompanyCampaign(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input struct {
		StartAt      string `json:"start_at"`
		StartInHours int    `json:"start_in_hours"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	now := time.Now().UTC()
	startAt := now.Add(time.Duration(input.StartInHours) * time.Hour)
	if trimmed := input.StartAt; trimmed != "" {
		parsed, err := time.Parse(time.RFC3339, trimmed)
		if err != nil {
			writeAgentError(c, http.StatusBadRequest, err)
			return
		}
		startAt = parsed
	}
	company, err := a.runtime.CompanyStore().ScheduleCampaign(c.Param("id"), c.Param("campaign_id"), startAt, now)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, company)
}

// recordCompanyCampaignMetrics registra desempenho MEDIDO de campanha em sandbox.
func (a *agentAPI) recordCompanyCampaignMetrics(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input agent.CampaignMetricsInput
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	metrics, err := a.runtime.CompanyStore().RecordCampaignMetrics(c.Param("id"), c.Param("campaign_id"), input, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, metrics)
}

// companyCampaignReport devolve o painel de campanhas e DECLARA que nada foi publicado.
func (a *agentAPI) companyCampaignReport(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	report, err := a.runtime.CompanyStore().CampaignReport(c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, report)
}
