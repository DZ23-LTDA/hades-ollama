package agent

import (
	"errors"
	"testing"
)

func TestCompanyGrowthLifecycleAndApprovals(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{OrganizationID: "org_local", Name: "Growth Smoke", Budget: CompanyBudget{Currency: "BRL", MonthlyLimitCents: 100000}})
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.AddCampaign(company.ID, CompanyCampaign{Name: "Lançamento", Channel: "social", Objective: "Gerar leads", DailyBudgetCents: 500})
	if err != nil || len(created.Campaigns) != 1 {
		t.Fatalf("campaign create failed: %v", err)
	}
	if created.Campaigns[0].Mode != "sandbox" {
		t.Fatalf("campaign mode=%q, want sandbox", created.Campaigns[0].Mode)
	}
	campaignID := created.Campaigns[0].ID
	if _, err := store.LaunchCampaign(company.ID, campaignID); !errors.Is(err, ErrCompanyApprovalRequiredForExternal) {
		t.Fatalf("expected campaign approval, got %v", err)
	}
	if _, err := store.ApproveCampaign(company.ID, campaignID); err != nil {
		t.Fatal(err)
	}
	created, err = store.LaunchCampaign(company.ID, campaignID)
	if err != nil || created.Campaigns[0].Status != "active" {
		t.Fatalf("campaign launch failed: %v", err)
	}

	created, err = store.AddAffiliateProgram(company.ID, CompanyAffiliateProgram{Name: "Parceiros", Network: "sandbox-network", CommissionBps: 1000})
	if err != nil {
		t.Fatal(err)
	}
	programID := created.AffiliatePrograms[0].ID
	if _, err := store.AddAffiliateLink(company.ID, CompanyAffiliateLink{ProgramID: programID, Destination: "https://shop.example.test/product"}); !errors.Is(err, ErrCompanyApprovalRequiredForExternal) {
		t.Fatalf("expected affiliate approval, got %v", err)
	}
	if _, err := store.ApproveAffiliateProgram(company.ID, programID); err != nil {
		t.Fatal(err)
	}
	created, err = store.AddAffiliateLink(company.ID, CompanyAffiliateLink{ProgramID: programID, Destination: "https://shop.example.test/product"})
	if err != nil || len(created.AffiliateLinks) != 1 {
		t.Fatalf("affiliate link failed: %v", err)
	}
	if created.AffiliatePrograms[0].Mode != "sandbox" || created.AffiliateLinks[0].Mode != "sandbox" {
		t.Fatalf("affiliate modes=%q/%q, want sandbox", created.AffiliatePrograms[0].Mode, created.AffiliateLinks[0].Mode)
	}
	if _, err := store.RecordAffiliateConversion(company.ID, created.AffiliateLinks[0].ID, 1200); err != nil {
		t.Fatal(err)
	}

	created, err = store.AddProduct(company.ID, CompanyProduct{SKU: "SKU-001", Name: "Produto sandbox", Supplier: "Fornecedor sandbox", CostCents: 500, PriceCents: 1200, Inventory: 3})
	if err != nil {
		t.Fatal(err)
	}
	productID := created.Products[0].ID
	// E-467: `CreateOrder` passou a devolver o pedido (criado ou existente) além da empresa.
	var pedidoCriado CompanyOrder
	created, pedidoCriado, err = store.CreateOrder(company.ID, CompanyOrder{ProductID: productID, CustomerRef: "customer-test", Quantity: 2})
	if err != nil || len(created.Orders) != 1 || created.Orders[0].Status != "pending_approval" {
		t.Fatalf("order create failed: %v", err)
	}
	if pedidoCriado.ID == "" || pedidoCriado.Replayed {
		t.Fatalf("pedido criado tem de vir com ID e SEM `Replayed`: %+v", pedidoCriado)
	}
	if created.Orders[0].Mode != "sandbox" {
		t.Fatalf("order mode=%q, want sandbox", created.Orders[0].Mode)
	}
	orderID := created.Orders[0].ID
	if _, err := store.FulfillOrder(company.ID, orderID, "TRACK-001"); !errors.Is(err, ErrCompanyApprovalRequiredForExternal) {
		t.Fatalf("expected order approval, got %v", err)
	}
	if _, err := store.ApproveOrder(company.ID, orderID); err != nil {
		t.Fatal(err)
	}
	created, err = store.FulfillOrder(company.ID, orderID, "TRACK-001")
	if err != nil || created.Orders[0].Status != "fulfilled" || created.Products[0].Inventory != 1 {
		t.Fatalf("fulfillment failed: %v", err)
	}
	if _, err := store.FulfillOrder(company.ID, orderID, "TRACK-001"); err != nil {
		t.Fatalf("fulfillment should be idempotent: %v", err)
	}
	if _, err := store.AddProduct(company.ID, CompanyProduct{SKU: "sku-001", Name: "Duplicado", PriceCents: 1200, Inventory: 1}); err == nil {
		t.Fatal("expected duplicate SKU rejection")
	}

	report, err := store.GrowthReport(company.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !report.SandboxOnly || report.CampaignsActive != 1 || report.AffiliateConversions != 1 || report.FulfilledOrders != 1 || report.RevenueCents != 3600 {
		t.Fatalf("unexpected growth report: %+v", report)
	}
}

func TestCompanyGrowthRejectsNonSandboxExecutionMode(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{OrganizationID: "org_local", Name: "External Guard"})
	if err != nil {
		t.Fatal(err)
	}
	company, err = store.AddCampaign(company.ID, CompanyCampaign{Name: "Campaign", Objective: "Objective"})
	if err != nil {
		t.Fatal(err)
	}
	company, err = store.mutate(company.ID, func(value *Company) error {
		value.Campaigns[0].Mode = "external"
		value.Campaigns[0].Approved = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.LaunchCampaign(company.ID, company.Campaigns[0].ID); !errors.Is(err, ErrCompanyGrowthExternalUnavailable) {
		t.Fatalf("expected external growth rejection, got %v", err)
	}
}

func TestGrowthExternalURLPolicy(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{OrganizationID: "org_local", Name: "URL Policy"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.AddAffiliateProgram(company.ID, CompanyAffiliateProgram{Name: "Program", Network: "sandbox", CommissionBps: 100})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ApproveAffiliateProgram(company.ID, created.AffiliatePrograms[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddAffiliateLink(company.ID, CompanyAffiliateLink{ProgramID: created.AffiliatePrograms[0].ID, Destination: "http://external.example.test/product"}); !errors.Is(err, ErrCompanyInvalidExternalURL) {
		t.Fatalf("expected HTTPS policy, got %v", err)
	}
}

func TestCompanyApprovalBindsActorNonceAndVersion(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{OrganizationID: "org_a", Name: "Approval Company"})
	if err != nil {
		t.Fatal(err)
	}
	company, err = store.AddCampaign(company.ID, CompanyCampaign{Name: "Campaign", Objective: "Objective"})
	if err != nil {
		t.Fatal(err)
	}
	approval, err := store.PendingApproval(company.ID, "campaign", company.Campaigns[0].ID)
	if err != nil || approval.Nonce == "" {
		t.Fatalf("approval=%+v err=%v", approval, err)
	}
	if _, err := store.DecideApproval(company.ID, approval.ID, true, "approved", "user_a", "org_a", company.Version, "wrong"); !errors.Is(err, ErrCompanyApprovalNonce) {
		t.Fatalf("wrong nonce err=%v", err)
	}
	decided, err := store.DecideApproval(company.ID, approval.ID, true, "approved", "user_a", "org_a", company.Version, approval.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	if !decided.Campaigns[0].Approved || decided.Approvals[0].ActorID != "user_a" || decided.Approvals[0].Status != CompanyApprovalApproved {
		t.Fatalf("decided company=%+v", decided)
	}
	if _, err := store.DecideApproval(company.ID, approval.ID, true, "replay", "user_a", "org_a", decided.Version, approval.Nonce); !errors.Is(err, ErrCompanyApprovalNotFound) {
		t.Fatalf("expected single-use approval, got %v", err)
	}
}

func TestCompanySpendRequiresApprovalBeforeBudgetMutation(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	company, err := store.Create(Company{OrganizationID: "org_a", Name: "Spend Company", Budget: CompanyBudget{MonthlyLimitCents: 10000, ApprovalThresholdCents: 1000}})
	if err != nil {
		t.Fatal(err)
	}
	pending, err := store.RecordSpendRequest(company.ID, "ads", 2500)
	if !errors.Is(err, ErrCompanySpendApprovalPending) {
		t.Fatalf("spend request err=%v", err)
	}
	if pending.Budget.SpentCents != 0 || len(pending.Approvals) != 1 {
		t.Fatalf("spend mutated before approval: %+v", pending)
	}
	approval := pending.Approvals[0]
	approved, err := store.DecideApproval(company.ID, approval.ID, true, "campaign budget approved", "admin_a", "org_a", pending.Version, approval.Nonce)
	if err != nil || approved.Budget.SpentCents != 2500 {
		t.Fatalf("approved spend=%+v err=%v", approved, err)
	}
	if approved.Approvals[0].Status != CompanyApprovalApproved {
		t.Fatalf("approval=%+v", approved.Approvals[0])
	}
}
