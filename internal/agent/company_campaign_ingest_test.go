package agent

import (
	"errors"
	"testing"
	"time"
)

// Estagio 26 — INGESTAO DE METRICA DE CAMPANHA (E-267). O limite declarado dizia "sem ingestao automatica de metrica
// (registro explicito)".
//
// A propriedade central: REENTREGA NAO PODE CONTAR DUAS VEZES. E a marca de "ja visto" so entra DEPOIS do sucesso,
// senao a entrega recusada seria engolida na retentativa.
func TestIngestaoDeMetricaDeCampanhaNaoContaDuasVezes(t *testing.T) {
	root := t.TempDir()
	store, err := NewCompanyStore(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	empresa, err := store.Create(Company{OrganizationID: LocalOrganizationID, Name: "Loja", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddCampaign(empresa.ID, CompanyCampaign{Name: "Lancamento", Objective: "vendas", Mode: "sandbox"}); err != nil {
		t.Fatalf("criar campanha: %v", err)
	}
	relatorio, err := store.CampaignReport(empresa.ID)
	if err != nil || len(relatorio.Campaigns) == 0 {
		t.Fatalf("a campanha precisa aparecer no relatorio: %v %+v", err, relatorio)
	}
	campanha := relatorio.Campaigns[0]
	// Metrica so existe para campanha que EXECUTOU: aprovar e lancar e' pre-requisito (o proprio motor recusa medir
	// campanha em rascunho).
	if _, err := store.ApproveCampaign(empresa.ID, campanha.ID); err != nil {
		t.Fatalf("aprovar campanha: %v", err)
	}
	if _, err := store.LaunchCampaign(empresa.ID, campanha.ID); err != nil {
		t.Fatalf("lancar campanha: %v", err)
	}

	// Entrega sem SourceID nao pode ser ingerida: sem chave nao ha reentrega segura.
	if _, err := store.IngestCampaignMetrics(empresa.ID, campanha.ID, CampaignMetricDelivery{Channel: "instagram"}, now); !errors.Is(err, ErrCampaignMetricDeliveryInvalid) {
		t.Fatalf("entrega sem SourceID precisa ser recusada: %v", err)
	}
	primeira, err := store.IngestCampaignMetrics(empresa.ID, campanha.ID, CampaignMetricDelivery{
		SourceID: "evt-1", Channel: "instagram",
		Metrics: CampaignMetricsInput{Impressions: 1000, Clicks: 40, Conversions: 4},
	}, now)
	if err != nil {
		t.Fatalf("primeira entrega: %v", err)
	}
	if primeira.Duplicate {
		t.Fatal("a primeira entrega nao pode ser marcada como duplicada")
	}
	if primeira.Metrics.Impressions != 1000 {
		t.Fatalf("a primeira entrega precisa somar: %+v", primeira.Metrics)
	}

	// REENTREGA da MESMA entrega: nada soma.
	repetida, err := store.IngestCampaignMetrics(empresa.ID, campanha.ID, CampaignMetricDelivery{
		SourceID: "evt-1", Channel: "instagram",
		Metrics: CampaignMetricsInput{Impressions: 1000, Clicks: 40, Conversions: 4},
	}, now)
	if err != nil {
		t.Fatalf("reentrega: %v", err)
	}
	if !repetida.Duplicate {
		t.Fatal("a reentrega precisa ser declarada duplicada")
	}
	if repetida.Metrics.Impressions != 1000 {
		t.Fatalf("REENTREGA NAO PODE CONTAR DUAS VEZES: %+v", repetida.Metrics)
	}

	// MESMO SourceID em CANAL DIFERENTE e' outra entrega (dois provedores podem colidir identificadores).
	outroCanal, err := store.IngestCampaignMetrics(empresa.ID, campanha.ID, CampaignMetricDelivery{
		SourceID: "evt-1", Channel: "facebook",
		Metrics: CampaignMetricsInput{Impressions: 500},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if outroCanal.Duplicate || outroCanal.Metrics.Impressions != 1500 {
		t.Fatalf("canal diferente precisa contar como entrega propria: %+v", outroCanal)
	}

	// PAYLOAD NAO CONFIAVEL: contador absurdo e' recusado e NAO entra.
	if _, err := store.IngestCampaignMetrics(empresa.ID, campanha.ID, CampaignMetricDelivery{
		SourceID: "evt-forjado", Channel: "instagram",
		Metrics: CampaignMetricsInput{Impressions: CampaignMetricIngestCeiling + 1},
	}, now); !errors.Is(err, ErrCampaignMetricDeliveryInvalid) {
		t.Fatalf("contador acima do teto precisa ser recusado: %v", err)
	}
	// E a entrega recusada NAO foi marcada como vista: corrigida, ela entra normalmente.
	corrigida, err := store.IngestCampaignMetrics(empresa.ID, campanha.ID, CampaignMetricDelivery{
		SourceID: "evt-forjado", Channel: "instagram",
		Metrics: CampaignMetricsInput{Impressions: 10},
	}, now)
	if err != nil {
		t.Fatalf("entrega corrigida precisa ser aceita (a recusa nao pode marcar como vista): %v", err)
	}
	if corrigida.Duplicate || corrigida.Metrics.Impressions != 1510 {
		t.Fatalf("entrega corrigida: %+v", corrigida)
	}

	// ISOLAMENTO: campanha de OUTRA empresa responde como inexistente.
	outra, err := store.Create(Company{OrganizationID: "org-b", Name: "Outra", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	_, errAlheia := store.IngestCampaignMetrics(outra.ID, campanha.ID, CampaignMetricDelivery{
		SourceID: "evt-x", Channel: "instagram", Metrics: CampaignMetricsInput{Impressions: 1},
	}, now)
	if !errors.Is(errAlheia, ErrCompanyCampaignNotFound) {
		t.Fatalf("campanha de outra empresa precisa responder como inexistente: %v", errAlheia)
	}

	// PERSISTENCIA: a marca de idempotencia sobrevive ao reinicio.
	reaberto, err := NewCompanyStore(root)
	if err != nil {
		t.Fatal(err)
	}
	depoisDoRestart, err := reaberto.IngestCampaignMetrics(empresa.ID, campanha.ID, CampaignMetricDelivery{
		SourceID: "evt-1", Channel: "instagram",
		Metrics: CampaignMetricsInput{Impressions: 1000},
	}, now)
	if err != nil {
		t.Fatalf("reentrega apos reinicio: %v", err)
	}
	if !depoisDoRestart.Duplicate || depoisDoRestart.Metrics.Impressions != 1510 {
		t.Fatalf("a idempotencia precisa sobreviver ao reinicio: %+v", depoisDoRestart)
	}
}
