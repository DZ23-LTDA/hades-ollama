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

// Estágio 30: o razão precisa ser alcançável com idempotência REAL pelo HTTP e com
// isolamento por organização.
func setupLedgerAPI(t *testing.T) (*gin.Engine, *agent.Runtime, agent.Company) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store:         agent.NewMemoryStore(),
		Planner:       agent.RulePlanner{},
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	company, err := runtime.CompanyStore().Create(agent.Company{
		Name: "Loja HTTP", OrganizationID: agent.LocalOrganizationID,
		Budget: agent.CompanyBudget{MonthlyLimitCents: 900000, Currency: "BRL"},
	})
	if err != nil {
		t.Fatalf("create company: %v", err)
	}
	api, err := newAgentAPI(runtime)
	if err != nil {
		t.Fatalf("create agent API: %v", err)
	}
	engine := gin.New()
	group := engine.Group("/api/agent/v1")
	group.GET("/companies/:id/ledger", api.listLedgerEntries)
	group.POST("/companies/:id/ledger", api.recordLedgerEntry)
	group.GET("/companies/:id/ledger/summary", api.ledgerSummary)
	group.POST("/companies/:id/ledger/:entry_id/reconcile", api.reconcileLedgerEntry)
	return engine, runtime, company
}

func ledgerRequest(engine *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestLedgerRoutesAreIdempotentAndReconcile(t *testing.T) {
	engine, _, company := setupLedgerAPI(t)
	base := "/api/agent/v1/companies/" + company.ID + "/ledger"
	payload := `{"kind":"expense","amount_cents":4200,"category":"infra","description":"servidor","idempotency_key":"fatura-2026-10"}`

	first := ledgerRequest(engine, http.MethodPost, base, payload)
	if first.Code != http.StatusCreated {
		t.Fatalf("primeiro lançamento precisa ser 201, veio %d (%s)", first.Code, first.Body.String())
	}
	var created struct {
		Entry   agent.LedgerEntry `json:"entry"`
		Created bool              `json:"created"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if !created.Created {
		t.Fatalf("o primeiro precisa ser criação: %s", first.Body.String())
	}

	// Reenvio da MESMA operação: 200 com o lançamento ORIGINAL, nunca 201 duplicado.
	second := ledgerRequest(engine, http.MethodPost, base, payload)
	if second.Code != http.StatusOK {
		t.Fatalf("reenvio idempotente precisa ser 200, veio %d (%s)", second.Code, second.Body.String())
	}
	var repeated struct {
		Entry   agent.LedgerEntry `json:"entry"`
		Created bool              `json:"created"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &repeated); err != nil {
		t.Fatal(err)
	}
	if repeated.Created {
		t.Fatal("o reenvio não pode criar outro lançamento")
	}
	if repeated.Entry.ID != created.Entry.ID {
		t.Fatalf("o reenvio precisa devolver o MESMO lançamento: %s vs %s", repeated.Entry.ID, created.Entry.ID)
	}

	// Entrada inválida é recusada com 400.
	invalid := ledgerRequest(engine, http.MethodPost, base, `{"kind":"expense","amount_cents":0}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("valor zero precisa ser 400, veio %d (%s)", invalid.Code, invalid.Body.String())
	}

	// Listagem traz o resumo coerente.
	list := ledgerRequest(engine, http.MethodGet, base, "")
	if list.Code != http.StatusOK {
		t.Fatalf("listagem precisa ser 200, veio %d", list.Code)
	}
	var listing struct {
		Count   int                 `json:"count"`
		Summary agent.LedgerSummary `json:"summary"`
		Entries []agent.LedgerEntry `json:"entries"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listing); err != nil {
		t.Fatal(err)
	}
	if listing.Count != 1 || len(listing.Entries) != 1 {
		t.Fatalf("precisa haver exatamente um lançamento: %+v", listing)
	}
	if listing.Summary.ExpenseCents != 4200 || listing.Summary.UnreconciledCount != 1 {
		t.Fatalf("resumo incorreto: %+v", listing.Summary)
	}

	// Conciliação: primeira muda, repetir com a mesma referência não muda, e
	// referência diferente é conflito.
	reconcileTarget := base + "/" + created.Entry.ID + "/reconcile"
	firstReconcile := ledgerRequest(engine, http.MethodPost, reconcileTarget, `{"reference":"extrato-linha-9"}`)
	if firstReconcile.Code != http.StatusOK {
		t.Fatalf("conciliação precisa ser 200, veio %d (%s)", firstReconcile.Code, firstReconcile.Body.String())
	}
	var reconciled struct {
		Changed bool `json:"changed"`
	}
	if err := json.Unmarshal(firstReconcile.Body.Bytes(), &reconciled); err != nil {
		t.Fatal(err)
	}
	if !reconciled.Changed {
		t.Fatal("a primeira conciliação precisa mudar o estado")
	}
	repeat := ledgerRequest(engine, http.MethodPost, reconcileTarget, `{"reference":"extrato-linha-9"}`)
	if repeat.Code != http.StatusOK {
		t.Fatalf("repetir a mesma conciliação precisa ser 200, veio %d", repeat.Code)
	}
	if err := json.Unmarshal(repeat.Body.Bytes(), &reconciled); err != nil {
		t.Fatal(err)
	}
	if reconciled.Changed {
		t.Fatal("repetir a mesma referência não pode mudar nada")
	}
	conflict := ledgerRequest(engine, http.MethodPost, reconcileTarget, `{"reference":"outro-extrato"}`)
	if conflict.Code == http.StatusOK {
		t.Fatalf("referência diferente precisa ser recusada, veio 200 (%s)", conflict.Body.String())
	}
	noReference := ledgerRequest(engine, http.MethodPost, base+"/"+created.Entry.ID+"/reconcile", `{"reference":""}`)
	if noReference.Code != http.StatusBadRequest {
		t.Fatalf("conciliação sem referência precisa ser 400, veio %d", noReference.Code)
	}

	// Resumo com fila de conciliação: o que foi conciliado sai da fila.
	summary := ledgerRequest(engine, http.MethodGet, base+"/summary?older_than_days=0", "")
	if summary.Code != http.StatusOK {
		t.Fatalf("resumo precisa ser 200, veio %d (%s)", summary.Code, summary.Body.String())
	}
	var summaryPayload struct {
		Summary     agent.LedgerSummary `json:"summary"`
		Pending     []agent.LedgerEntry `json:"pending_reconciliation"`
		PendingSize int                 `json:"pending_count"`
	}
	if err := json.Unmarshal(summary.Body.Bytes(), &summaryPayload); err != nil {
		t.Fatal(err)
	}
	if summaryPayload.Summary.ReconciledCount != 1 {
		t.Fatalf("o resumo precisa contar o conciliado: %+v", summaryPayload.Summary)
	}
	if summaryPayload.PendingSize != 0 {
		t.Fatalf("nada pode ficar pendente depois de conciliar: %+v", summaryPayload.Pending)
	}

	// Período inválido é recusado.
	if bad := ledgerRequest(engine, http.MethodGet, base+"?from=31-12-2026", ""); bad.Code != http.StatusBadRequest {
		t.Fatalf("data inválida precisa ser 400, veio %d", bad.Code)
	}
}

func TestLedgerRoutesAreScopedByOrganization(t *testing.T) {
	engine, runtime, company := setupLedgerAPI(t)
	// Empresa de OUTRA organização: a rota não pode entregar nem aceitar lançamento.
	other, err := runtime.CompanyStore().Create(agent.Company{Name: "Outra org", OrganizationID: "org-terceira"})
	if err != nil {
		t.Fatal(err)
	}
	target := "/api/agent/v1/companies/" + other.ID + "/ledger"
	if recorder := ledgerRequest(engine, http.MethodPost, target,
		`{"kind":"income","amount_cents":1000}`); recorder.Code == http.StatusCreated || recorder.Code == http.StatusOK {
		t.Fatalf("lançamento em empresa de outra organização precisa ser recusado, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	if recorder := ledgerRequest(engine, http.MethodGet, target, ""); recorder.Code == http.StatusOK {
		t.Fatalf("leitura de outra organização precisa ser recusada, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	// E a empresa da própria organização continua funcionando.
	own := "/api/agent/v1/companies/" + company.ID + "/ledger"
	if recorder := ledgerRequest(engine, http.MethodGet, own, ""); recorder.Code != http.StatusOK {
		t.Fatalf("a própria empresa precisa ser acessível, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	// Sanidade temporal: o período vazio não pode esconder lançamento recém-criado.
	created := ledgerRequest(engine, http.MethodPost, own,
		`{"kind":"income","amount_cents":777,"idempotency_key":"venda-hoje"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("lançamento de hoje precisa ser criado: %d (%s)", created.Code, created.Body.String())
	}
	today := time.Now().UTC().Format("2006-01-02")
	period := ledgerRequest(engine, http.MethodGet, own+"?from="+today+"&to="+today, "")
	if period.Code != http.StatusOK {
		t.Fatalf("recorte de hoje precisa ser 200, veio %d", period.Code)
	}
	var payload struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(period.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Count == 0 {
		t.Fatal("o recorte de hoje precisa incluir o lançamento de hoje (limite superior inclusivo)")
	}
}
