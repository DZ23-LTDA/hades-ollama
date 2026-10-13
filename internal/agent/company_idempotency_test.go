package agent

import (
	"errors"
	"testing"
)

func TestCompanySpendRequestIdempotencyPreventsDuplicateApprovalAndDebit(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{OrganizationID: "org-a", Name: "Idempotent spend", Budget: CompanyBudget{MonthlyLimitCents: 10000, ApprovalThresholdCents: 1000}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.RecordSpendRequestWithIdempotency(company.ID, "ads", 2500, "spend-1")
	if !errors.Is(err, ErrCompanySpendApprovalPending) {
		t.Fatalf("first spend error = %v", err)
	}
	first, err := store.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Approvals) != 1 || first.Budget.SpentCents != 0 {
		t.Fatalf("first spend state = %+v", first)
	}
	version := first.Version
	_, err = store.RecordSpendRequestWithIdempotency(company.ID, "ads", 2500, "spend-1")
	if !errors.Is(err, ErrCompanyIdempotentReplay) {
		t.Fatalf("replay error = %v", err)
	}
	second, err := store.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Approvals) != 1 || second.Budget.SpentCents != 0 || second.Version != version {
		t.Fatalf("replay changed state = %+v", second)
	}
	_, err = store.RecordSpendRequestWithIdempotency(company.ID, "ads", 2600, "spend-1")
	if !errors.Is(err, ErrCompanyIdempotencyConflict) {
		t.Fatalf("key reuse conflict = %v", err)
	}

	company, err = store.Create(Company{OrganizationID: "org-a", Name: "Immediate spend", Budget: CompanyBudget{MonthlyLimitCents: 10000}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordSpendRequestWithIdempotency(company.ID, "tools", 100, "spend-2"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordSpendRequestWithIdempotency(company.ID, "tools", 100, "spend-2"); !errors.Is(err, ErrCompanyIdempotentReplay) {
		t.Fatalf("immediate replay error = %v", err)
	}
	final, err := store.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.Budget.SpentCents != 100 {
		t.Fatalf("immediate replay debited %d cents", final.Budget.SpentCents)
	}
}

func TestCompanyAffiliateAndSocialIdempotencyPreventsDuplicateMetrics(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{OrganizationID: "org-a", Name: "Idempotent growth"})
	if err != nil {
		t.Fatal(err)
	}
	company, err = store.AddAffiliateProgram(company.ID, CompanyAffiliateProgram{Name: "Network", Network: "network", CommissionBps: 100})
	if err != nil {
		t.Fatal(err)
	}
	programID := company.AffiliatePrograms[0].ID
	company, err = store.ApproveAffiliateProgram(company.ID, programID)
	if err != nil {
		t.Fatal(err)
	}
	company, err = store.AddAffiliateLink(company.ID, CompanyAffiliateLink{ProgramID: programID, Destination: "https://example.com/product"})
	if err != nil {
		t.Fatal(err)
	}
	linkID := company.AffiliateLinks[0].ID
	if _, err := store.RecordAffiliateConversionWithIdempotency(company.ID, linkID, 500, "conversion-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordAffiliateConversionWithIdempotency(company.ID, linkID, 500, "conversion-1"); !errors.Is(err, ErrCompanyIdempotentReplay) {
		t.Fatalf("conversion replay error = %v", err)
	}

	company, err = store.CreateSocialDraft(company.ID, CompanySocialDraft{Provider: "x", Title: "Draft", Body: "Body", Mode: "sandbox"})
	if err != nil {
		t.Fatal(err)
	}
	draftID := company.SocialDrafts[0].ID
	// E-307: MÉTRICA SÓ EXISTE PARA CONTEÚDO PUBLICADO. Este teste é sobre IDEMPOTÊNCIA, não sobre métrica de
	// rascunho — então ele publica primeiro, como manda a regra medida. (Antes, ele gravava métrica de rascunho
	// nunca publicado: o teste cristalizava o comportamento errado.)
	if _, err := store.ApproveSocialDraft(company.ID, draftID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublishSocialDraft(company.ID, draftID); err != nil {
		t.Fatal(err)
	}
	metric := CompanySocialMetric{DraftID: draftID, Provider: "x", Impressions: 10, Clicks: 2, Conversions: 1}
	if _, err := store.RecordSocialMetricWithIdempotency(company.ID, metric, "metric-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordSocialMetricWithIdempotency(company.ID, metric, "metric-1"); !errors.Is(err, ErrCompanyIdempotentReplay) {
		t.Fatalf("metric replay error = %v", err)
	}
	final, err := store.Get(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(final.AffiliateLinks) != 1 || final.AffiliateLinks[0].Conversions != 1 || len(final.SocialMetrics) != 1 {
		t.Fatalf("duplicate growth effects = %+v", final)
	}
}
