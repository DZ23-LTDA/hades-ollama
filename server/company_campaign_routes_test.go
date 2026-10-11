package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Estágio 26 pelas ROTAS: agendar exige aprovação, métrica exige execução, orçamento limita e o
// painel DECLARA que nada foi publicado externamente.
func TestCampaignRoutesScheduleMetricsAndReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		WorkspaceRoot: shortServerRoot(t),
		DataRoot:      shortServerRoot(t),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	engine := gin.New()
	group := engine.Group("/api/agent/v1")
	group.POST("/companies/:id/campaigns", api.addCompanyCampaign)
	group.POST("/companies/:id/campaigns/:campaign_id/schedule", api.scheduleCompanyCampaign)
	group.POST("/companies/:id/campaigns/:campaign_id/metrics", api.recordCompanyCampaignMetrics)
	group.GET("/companies/:id/campaigns/report", api.companyCampaignReport)

	company, err := runtime.CompanyStore().CreateRequest(agent.CompanyCreateRequest{
		Name: "Loja de Rota", Mission: "vender", Currency: "BRL",
	}, agent.LocalOrganizationID)
	if err != nil {
		t.Fatal(err)
	}

	// A campanha nasce em rascunho e pede aprovação (201).
	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/companies/"+company.ID+"/campaigns",
		strings.NewReader(`{"name":"Campanha de rota","channel":"content","objective":"gerar demanda","daily_budget_cents":1000}`))
	createRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("criação de campanha precisa ser 201, veio %d (%s)", createRecorder.Code, createRecorder.Body.String())
	}
	var withCampaign agent.Company
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &withCampaign); err != nil {
		t.Fatal(err)
	}
	if len(withCampaign.Campaigns) != 1 {
		t.Fatalf("campanha não criada: %+v", withCampaign.Campaigns)
	}
	campaignID := withCampaign.Campaigns[0].ID

	// AGENDAR SEM APROVAÇÃO: 409.
	unapproved := httptest.NewRecorder()
	unapprovedRequest := httptest.NewRequest(http.MethodPost,
		"/api/agent/v1/companies/"+company.ID+"/campaigns/"+campaignID+"/schedule",
		strings.NewReader(`{"start_in_hours":2}`))
	unapprovedRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(unapproved, unapprovedRequest)
	if unapproved.Code != http.StatusConflict {
		t.Fatalf("agendar sem aprovação precisa ser 409, veio %d (%s)", unapproved.Code, unapproved.Body.String())
	}
	// `start_at` inválido: 400.
	badTime := httptest.NewRecorder()
	badTimeRequest := httptest.NewRequest(http.MethodPost,
		"/api/agent/v1/companies/"+company.ID+"/campaigns/"+campaignID+"/schedule",
		strings.NewReader(`{"start_at":"amanhã"}`))
	badTimeRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(badTime, badTimeRequest)
	if badTime.Code != http.StatusBadRequest {
		t.Fatalf("start_at inválido precisa ser 400, veio %d", badTime.Code)
	}

	// Aprovação pela mesma superfície de aprovações da empresa.
	if _, err := runtime.CompanyStore().ApproveCampaign(company.ID, campaignID); err != nil {
		t.Fatal(err)
	}

	// MÉTRICA ANTES DE EXECUTAR: 409 (apenas aprovada, nada rodou).
	earlyMetrics := httptest.NewRecorder()
	earlyRequest := httptest.NewRequest(http.MethodPost,
		"/api/agent/v1/companies/"+company.ID+"/campaigns/"+campaignID+"/metrics",
		strings.NewReader(`{"impressions":10}`))
	earlyRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(earlyMetrics, earlyRequest)
	if earlyMetrics.Code != http.StatusConflict {
		t.Fatalf("métrica sem execução precisa ser 409, veio %d (%s)", earlyMetrics.Code, earlyMetrics.Body.String())
	}

	// AGENDAR COM APROVAÇÃO: 200 e estado agendado.
	scheduleRecorder := httptest.NewRecorder()
	scheduleRequest := httptest.NewRequest(http.MethodPost,
		"/api/agent/v1/companies/"+company.ID+"/campaigns/"+campaignID+"/schedule",
		strings.NewReader(`{"start_at":"`+time.Now().UTC().Add(3*time.Hour).Format(time.RFC3339)+`"}`))
	scheduleRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(scheduleRecorder, scheduleRequest)
	if scheduleRecorder.Code != http.StatusOK {
		t.Fatalf("agendar com aprovação precisa ser 200, veio %d (%s)", scheduleRecorder.Code, scheduleRecorder.Body.String())
	}
	var scheduled agent.Company
	if err := json.Unmarshal(scheduleRecorder.Body.Bytes(), &scheduled); err != nil {
		t.Fatal(err)
	}
	if len(scheduled.Campaigns) != 1 || scheduled.Campaigns[0].Status != agent.CampaignStatusScheduled || scheduled.Campaigns[0].ScheduledAt == nil {
		t.Fatalf("campanha precisa ficar agendada com data: %+v", scheduled.Campaigns)
	}

	// ORÇAMENTO: incremento acima do diário é 409.
	overBudget := httptest.NewRecorder()
	overRequest := httptest.NewRequest(http.MethodPost,
		"/api/agent/v1/companies/"+company.ID+"/campaigns/"+campaignID+"/metrics",
		strings.NewReader(`{"spend_cents":1001}`))
	overRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(overBudget, overRequest)
	if overBudget.Code != http.StatusConflict {
		t.Fatalf("estourar o orçamento precisa ser 409, veio %d (%s)", overBudget.Code, overBudget.Body.String())
	}
	// Métrica NEGATIVA: 400 (erro de entrada).
	negative := httptest.NewRecorder()
	negativeRequest := httptest.NewRequest(http.MethodPost,
		"/api/agent/v1/companies/"+company.ID+"/campaigns/"+campaignID+"/metrics",
		strings.NewReader(`{"clicks":-5}`))
	negativeRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(negative, negativeRequest)
	if negative.Code != http.StatusBadRequest {
		t.Fatalf("contador negativo precisa ser 400, veio %d", negative.Code)
	}

	// MÉTRICA VÁLIDA: 200 com taxas e SEM publicação externa.
	metricsRecorder := httptest.NewRecorder()
	metricsRequest := httptest.NewRequest(http.MethodPost,
		"/api/agent/v1/companies/"+company.ID+"/campaigns/"+campaignID+"/metrics",
		strings.NewReader(`{"impressions":400,"clicks":40,"conversions":8,"spend_cents":700}`))
	metricsRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(metricsRecorder, metricsRequest)
	if metricsRecorder.Code != http.StatusOK {
		t.Fatalf("métrica válida precisa ser 200, veio %d (%s)", metricsRecorder.Code, metricsRecorder.Body.String())
	}
	var metrics agent.CampaignMetrics
	if err := json.Unmarshal(metricsRecorder.Body.Bytes(), &metrics); err != nil {
		t.Fatal(err)
	}
	if metrics.Impressions != 400 || metrics.Clicks != 40 || metrics.ClickRate != "10.00%" || metrics.ConversionRate != "20.00%" {
		t.Fatalf("métrica incorreta: %+v", metrics)
	}
	if metrics.PublicadoExternamente || metrics.PublicacaoExterna != agent.CampaignExternalPublication {
		t.Fatalf("a rota não pode declarar publicação externa: %+v", metrics)
	}

	// PAINEL: 200 declarando sandbox e publicação externa bloqueada.
	reportRecorder := httptest.NewRecorder()
	engine.ServeHTTP(reportRecorder, httptest.NewRequest(http.MethodGet, "/api/agent/v1/companies/"+company.ID+"/campaigns/report", nil))
	if reportRecorder.Code != http.StatusOK {
		t.Fatalf("painel precisa ser 200, veio %d (%s)", reportRecorder.Code, reportRecorder.Body.String())
	}
	var report agent.CampaignReport
	if err := json.Unmarshal(reportRecorder.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.Sandbox || report.PublicacaoExterna != agent.CampaignExternalPublication || report.Motivo == "" {
		t.Fatalf("painel precisa declarar sandbox e bloqueio com motivo: %+v", report)
	}
	if report.Total.SpendCents != 700 {
		t.Fatalf("total de gasto incorreto: %+v", report.Total)
	}

	// Empresa inexistente: 404.
	missing := httptest.NewRecorder()
	engine.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/agent/v1/companies/co_inexistente/campaigns/report", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("empresa inexistente precisa ser 404, veio %d", missing.Code)
	}
}
