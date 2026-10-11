package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Estágios 27 e 35 / comando `/crm`: as rotas precisam existir DE VERDADE, com
// deduplicação, funil, consentimento pelo CRM e isolamento por organização.
func setupCRMAPI(t *testing.T) (*gin.Engine, *agent.Runtime) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
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
	group.GET("/crm/contacts", api.listCRMContacts)
	group.POST("/crm/contacts", api.upsertCRMContact)
	group.GET("/crm/contacts/:id", api.getCRMContact)
	group.POST("/crm/contacts/:id/opportunities", api.createCRMOpportunity)
	group.POST("/crm/contacts/:id/consent", api.grantCRMContactConsent)
	group.POST("/crm/contacts/:id/consent/revoke", api.revokeCRMContactConsent)
	group.POST("/crm/opportunities/:id/stage", api.setCRMOpportunityStage)
	group.GET("/crm/pipeline", api.crmPipeline)
	return engine, runtime
}

func crmRequest(engine *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestCRMRoutesDeduplicateAndReportPipeline(t *testing.T) {
	engine, _ := setupCRMAPI(t)
	contactsPath := "/api/agent/v1/crm/contacts"

	first := crmRequest(engine, http.MethodPost, contactsPath, `{"name":"Ana Souza","phone":"+55 (11) 98888-7777","tags":["loja"]}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("primeiro contato precisa ser 201, veio %d (%s)", first.Code, first.Body.String())
	}
	var created struct {
		Contact agent.CRMContact `json:"contact"`
		Created bool             `json:"created"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if !created.Created {
		t.Fatalf("precisa ser criação: %s", first.Body.String())
	}

	// Mesmo cliente por outro formato: 200 com o MESMO contato, nunca 201 duplicado.
	second := crmRequest(engine, http.MethodPost, contactsPath, `{"name":"Ana S.","phone":"5511988887777"}`)
	if second.Code != http.StatusOK {
		t.Fatalf("deduplicação precisa responder 200, veio %d (%s)", second.Code, second.Body.String())
	}
	var repeated struct {
		Contact agent.CRMContact `json:"contact"`
		Created bool             `json:"created"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &repeated); err != nil {
		t.Fatal(err)
	}
	if repeated.Created || repeated.Contact.ID != created.Contact.ID {
		t.Fatalf("o reenvio precisa devolver o mesmo contato: %+v", repeated)
	}

	// Contato sem identidade é 400.
	if bad := crmRequest(engine, http.MethodPost, contactsPath, `{"name":"Sem identidade"}`); bad.Code != http.StatusBadRequest {
		t.Fatalf("contato sem identidade precisa ser 400, veio %d (%s)", bad.Code, bad.Body.String())
	}

	// Oportunidades e funil.
	deal := crmRequest(engine, http.MethodPost, contactsPath+"/"+created.Contact.ID+"/opportunities", `{"title":"Plano mensal","value_cents":29900}`)
	if deal.Code != http.StatusCreated {
		t.Fatalf("oportunidade precisa ser 201, veio %d (%s)", deal.Code, deal.Body.String())
	}
	var opportunity agent.CRMOpportunity
	if err := json.Unmarshal(deal.Body.Bytes(), &opportunity); err != nil {
		t.Fatal(err)
	}
	pipeline := crmRequest(engine, http.MethodGet, "/api/agent/v1/crm/pipeline", "")
	if pipeline.Code != http.StatusOK {
		t.Fatalf("funil precisa ser 200, veio %d", pipeline.Code)
	}
	var summary agent.CRMPipeline
	if err := json.Unmarshal(pipeline.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.OpenCount != 1 || summary.OpenValueCents != 29900 {
		t.Fatalf("funil incorreto: %+v", summary)
	}

	// Fase: ganho e depois tentativa de reabrir (recusada).
	won := crmRequest(engine, http.MethodPost, "/api/agent/v1/crm/opportunities/"+opportunity.ID+"/stage", `{"stage":"won"}`)
	if won.Code != http.StatusOK {
		t.Fatalf("mudança de fase precisa ser 200, veio %d (%s)", won.Code, won.Body.String())
	}
	reopen := crmRequest(engine, http.MethodPost, "/api/agent/v1/crm/opportunities/"+opportunity.ID+"/stage", `{"stage":"proposal"}`)
	if reopen.Code == http.StatusOK {
		t.Fatalf("reabrir oportunidade ganha precisa ser recusado, veio 200 (%s)", reopen.Body.String())
	}
	badStage := crmRequest(engine, http.MethodPost, "/api/agent/v1/crm/opportunities/"+opportunity.ID+"/stage", `{"stage":"inventado"}`)
	if badStage.Code != http.StatusBadRequest {
		t.Fatalf("fase inválida precisa ser 400, veio %d", badStage.Code)
	}
	// Oportunidade de contato inexistente é recusada.
	missing := crmRequest(engine, http.MethodPost, contactsPath+"/ct_inexistente/opportunities", `{"title":"x","value_cents":1}`)
	if missing.Code == http.StatusCreated || missing.Code == http.StatusOK {
		t.Fatalf("contato inexistente precisa ser recusado, veio %d", missing.Code)
	}
}

func TestCRMRoutesGrantAndRevokeConsent(t *testing.T) {
	engine, runtime := setupCRMAPI(t)
	contactsPath := "/api/agent/v1/crm/contacts"
	phone := "+5511977776666"
	created := crmRequest(engine, http.MethodPost, contactsPath, `{"name":"Titular","phone":"`+phone+`"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("contato precisa ser criado: %d (%s)", created.Code, created.Body.String())
	}
	var contact struct {
		Contact agent.CRMContact `json:"contact"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &contact); err != nil {
		t.Fatal(err)
	}

	// Conceder consentimento PELO CRM fecha o limite declarado em E-101.
	grant := crmRequest(engine, http.MethodPost, contactsPath+"/"+contact.Contact.ID+"/consent", `{"channel":"whatsapp","source":"site","evidence":"checkbox marcado"}`)
	if grant.Code != http.StatusCreated {
		t.Fatalf("concessão precisa ser 201, veio %d (%s)", grant.Code, grant.Body.String())
	}
	// O contato passa a mostrar o canal LIBERADO, lido do registro.
	view := crmRequest(engine, http.MethodGet, contactsPath+"/"+contact.Contact.ID, "")
	if view.Code != http.StatusOK {
		t.Fatalf("visão do contato precisa ser 200, veio %d", view.Code)
	}
	var detail struct {
		Consent []agent.ConsentRecord `json:"consent"`
		Blocked map[string]bool       `json:"blocked_channels"`
	}
	if err := json.Unmarshal(view.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.Consent) != 1 || detail.Blocked["whatsapp"] {
		t.Fatalf("consentimento precisa aparecer e não estar bloqueado: %+v", detail)
	}

	// Revogar sem motivo é 400.
	if noReason := crmRequest(engine, http.MethodPost, contactsPath+"/"+contact.Contact.ID+"/consent/revoke", `{"channel":"whatsapp"}`); noReason.Code != http.StatusBadRequest {
		t.Fatalf("revogação sem motivo precisa ser 400, veio %d (%s)", noReason.Code, noReason.Body.String())
	}
	// Com motivo: opt-out registrado e canal BLOQUEADO.
	revoke := crmRequest(engine, http.MethodPost, contactsPath+"/"+contact.Contact.ID+"/consent/revoke", `{"channel":"whatsapp","reason":"pedido do titular por e-mail"}`)
	if revoke.Code != http.StatusOK {
		t.Fatalf("revogação precisa ser 200, veio %d (%s)", revoke.Code, revoke.Body.String())
	}
	view = crmRequest(engine, http.MethodGet, contactsPath+"/"+contact.Contact.ID, "")
	if err := json.Unmarshal(view.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if !detail.Blocked["whatsapp"] {
		t.Fatalf("após opt-out o canal precisa aparecer bloqueado: %+v", detail.Blocked)
	}

	// E o envio pelo gateway respeita o opt-out registrado pelo CRM: a fonte do
	// consentimento é ÚNICA, então o CRM não pode destravar o canal.
	if registry := runtime.Consent(); registry == nil {
		t.Fatal("runtime precisa ter registro de consentimento")
	} else if err := registry.AuthorizeContact(contact.Contact.OrganizationID, agent.ConsentChannelWhatsApp, phone, time.Now().UTC()); err == nil {
		t.Fatal("o opt-out registrado pelo CRM precisa bloquear o envio")
	}
}

func TestCRMRoutesAreScopedByOrganization(t *testing.T) {
	engine, runtime := setupCRMAPI(t)
	// Contato criado direto no store com OUTRA organização.
	other, _, err := runtime.CRM().UpsertContact(agent.CRMContact{OrganizationID: "org-terceira", Name: "De outra org", Phone: "+5511933332222"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if got := crmRequest(engine, http.MethodGet, "/api/agent/v1/crm/contacts/"+other.ID, ""); got.Code == http.StatusOK {
		t.Fatalf("contato de outra organização não pode ser lido, veio 200 (%s)", got.Body.String())
	}
	list := crmRequest(engine, http.MethodGet, "/api/agent/v1/crm/contacts", "")
	var payload struct {
		Contacts []agent.CRMContact `json:"contacts"`
		Count    int                `json:"count"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Count != 0 {
		t.Fatalf("a listagem não pode trazer contato de outra organização: %+v", payload.Contacts)
	}
	// Oportunidade em contato de outra organização é recusada.
	if got := crmRequest(engine, http.MethodPost, "/api/agent/v1/crm/contacts/"+other.ID+"/opportunities", `{"title":"x","value_cents":1}`); got.Code == http.StatusCreated {
		t.Fatalf("oportunidade cruzando organizações precisa ser recusada, veio %d", got.Code)
	}
}
