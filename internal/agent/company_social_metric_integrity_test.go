package agent

import (
	"testing"
	"time"
)

// ESTÁGIO 36 — MÉTRICA SÓ EXISTE PARA CONTEÚDO PUBLICADO (E-307).
//
// O `RecordSocialMetricWithIdempotency` confere que o RASCUNHO EXISTE, mas não que ele tenha sido PUBLICADO. Isso
// permite lançar impressões/cliques/conversões de um post que nunca saiu — e o relatório passa a contar números sem
// causa. O documento da missão é explícito: não inventar métricas.
//
// Propriedades travadas:
//  1. métrica para rascunho NÃO publicado ⇒ recusada com erro próprio (`ErrCompanySocialMetricNotPublished`);
//  2. métrica para rascunho PUBLICADO ⇒ aceita;
//  3. rascunho inexistente ⇒ continua recusado (regressão travada);
//  4. a mesma métrica com a MESMA chave de idempotência não duplica;
//  5. o relatório só conta métrica de conteúdo publicado.
func TestMetricaSocialExigeConteudoPublicado(t *testing.T) {
	store, empresa, agora := socialMetricStore(t)
	naoPublicado := socialMetricDraft(t, store, empresa, agora)

	metricas := CompanySocialMetric{DraftID: naoPublicado.ID, Provider: "instagram", Impressions: 100, Clicks: 5, Conversions: 1}

	// 1. NÃO PUBLICADO: recusa. Medido antes da correção: era ACEITO — número sem causa.
	if _, err := store.RecordSocialMetricWithIdempotency(empresa, metricas, "chave-nao-publicado"); err == nil {
		t.Fatal("métrica de conteúdo NÃO publicado precisa ser recusada: número sem causa é métrica inventada")
	}

	// 3. Rascunho inexistente continua recusado.
	if _, err := store.RecordSocialMetricWithIdempotency(empresa, CompanySocialMetric{
		DraftID: "sdraft_inexistente", Provider: "instagram", Impressions: 1,
	}, "chave-inexistente"); err == nil {
		t.Fatal("métrica de rascunho inexistente precisa continuar recusada")
	}

	// 2. PUBLICADO: aceita.
	if _, err := store.ApproveSocialDraft(empresa, naoPublicado.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublishSocialDraft(empresa, naoPublicado.ID); err != nil {
		t.Fatalf("publicar: %v", err)
	}
	if _, err := store.RecordSocialMetricWithIdempotency(empresa, metricas, "chave-publicado"); err != nil {
		t.Fatalf("métrica de conteúdo publicado precisa ser aceita: %v", err)
	}

	// 4. Idempotência: a MESMA métrica com a MESMA chave não duplica.
	antes, err := store.SocialReport(empresa)
	if err != nil {
		t.Fatal(err)
	}
	// MEDIDO: o replay idempotente deste store é RECUSA EXPLÍCITA (`ErrCompanyIdempotentReplay`), não silêncio.
	// O que importa para a propriedade é que os TOTAIS não cresçam.
	if _, err := store.RecordSocialMetricWithIdempotency(empresa, metricas, "chave-publicado"); err == nil {
		t.Fatal("o replay idempotente precisa ser recusado explicitamente")
	}
	depois, err := store.SocialReport(empresa)
	if err != nil {
		t.Fatal(err)
	}
	if depois.Impressions != antes.Impressions {
		t.Fatalf("a mesma chave não pode duplicar a métrica (recusa explícita, totais iguais): %d -> %d", antes.Impressions, depois.Impressions)
	}

	// 5. O relatório conta a métrica do publicado — e ela é do post, não de outro.
	if depois.Impressions == 0 {
		t.Fatal("o relatório precisa contar a métrica do conteúdo publicado")
	}
}

// socialMetricStore cria o store de empresa isolado para o teste de métricas.
func socialMetricStore(t *testing.T) (*CompanyStore, string, time.Time) {
	t.Helper()
	store, err := NewCompanyStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	criada, err := store.Create(Company{ID: "emp-metrica", OrganizationID: "org-a", Name: "Loja Métrica", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	return store, criada.ID, agora
}

// socialMetricDraft cria UM rascunho (não publicado) e devolve o estado dele.
func socialMetricDraft(t *testing.T, store *CompanyStore, empresa string, agora time.Time) CompanySocialDraft {
	t.Helper()
	atualizada, err := store.CreateSocialDraft(empresa, CompanySocialDraft{
		Provider: "instagram",
		Title:    "conteúdo para medir",
		Body:     "corpo do conteúdo",
	})
	if err != nil {
		t.Fatalf("criar rascunho: %v", err)
	}
	for _, d := range atualizada.SocialDrafts {
		if d.Title == "conteúdo para medir" {
			return d
		}
	}
	t.Fatal("rascunho não encontrado")
	return CompanySocialDraft{}
}
