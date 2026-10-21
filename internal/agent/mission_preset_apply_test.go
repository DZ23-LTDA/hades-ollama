package agent

// FASE 23 do adendo V3: o preset passa a GOVERNAR parte da missao -- e a NAO governar outra.
//
// Aplicam-se so' as coisas que e' seguro aplicar sozinho (orcamento declarado, criterios de
// conclusao). NAO se aplicam `Permissions`: *escolher um preset nao pode conceder permissao --
// seria um formulario que assina por baixo.*

import (
	"context"
	"testing"
)

// TestCreateMissionAplicaCriteriosEOrcamentoDoPreset prova o que o preset GOVERNA.
func TestCreateMissionAplicaCriteriosEOrcamentoDoPreset(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:      "produzir o briefing do dia",
		ProjectID:      project.ID,
		OrganizationID: "org-a",
		PresetID:       "briefing-diario",
	})
	if err != nil {
		t.Fatalf("CreateMission: %v", err)
	}

	catalogo, ok := MissionPresetByID("briefing-diario")
	if !ok {
		t.Fatal("o preset briefing-diario devia existir")
	}
	if len(missao.PresetCompletionCriteria) != len(catalogo.CompletionCriteria) {
		t.Fatalf("criterios: missao=%d catalogo=%d",
			len(missao.PresetCompletionCriteria), len(catalogo.CompletionCriteria))
	}
	for i, criterio := range catalogo.CompletionCriteria {
		if missao.PresetCompletionCriteria[i] != criterio {
			t.Errorf("criterio %d: missao=%q catalogo=%q", i, missao.PresetCompletionCriteria[i], criterio)
		}
	}
	if missao.PresetBudgetSeconds != catalogo.BudgetSeconds {
		t.Fatalf("orcamento: missao=%d catalogo=%d",
			missao.PresetBudgetSeconds, catalogo.BudgetSeconds)
	}
	if missao.PresetBudgetSeconds <= 0 {
		t.Fatalf("briefing-diario devia declarar orcamento positivo, veio %d", missao.PresetBudgetSeconds)
	}

	// E tem de sobreviver ao store, nao apenas existir na struct devolvida.
	recarregada, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatalf("GetMission: %v", err)
	}
	if len(recarregada.PresetCompletionCriteria) == 0 {
		t.Fatal("os criterios nao sobreviveram ao store")
	}
	if recarregada.PresetBudgetSeconds != missao.PresetBudgetSeconds {
		t.Fatal("o orcamento nao sobreviveu ao store")
	}
}

// TestPresetNAOConcedePermissoes e' o teste de SEGURANCA da fatia.
//
// `monitor-agendado` declara `permissions: ["network:read"]`. Se o preset elevasse privilegio,
// a missao nascia com essa capability so' porque o utilizador escolheu um preset no ecra.
// *Um preset que se auto-autorizasse seria um formulario que assina por baixo.*
func TestPresetNAOConcedePermissoes(t *testing.T) {
	catalogo, ok := MissionPresetByID("monitor-agendado")
	if !ok {
		t.Fatal("o preset monitor-agendado devia existir")
	}
	if len(catalogo.Permissions) == 0 {
		t.Fatal("este teste perde o sentido se o preset nao declarar permissoes")
	}

	runtime, _, project := idempotencyRuntime(t)
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:      "vigiar o servico",
		ProjectID:      project.ID,
		OrganizationID: "org-a",
		PresetID:       "monitor-agendado",
		// O pedido NAO declara capabilities: so' o preset as pede.
	})
	if err != nil {
		t.Fatalf("CreateMission: %v", err)
	}

	concedidas := map[string]bool{}
	for _, capability := range missao.Capabilities {
		concedidas[capability] = true
	}
	for _, permissao := range catalogo.Permissions {
		if concedidas[permissao] {
			t.Errorf("o preset concedeu %q sozinho -- isso e' elevacao de privilegio por formulario",
				permissao)
		}
	}
}

// TestCreateMissionSemPresetNaoTemCriteriosNemOrcamento prova a compatibilidade.
func TestCreateMissionSemPresetNaoTemCriteriosNemOrcamento(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:      "tarefa sem preset",
		ProjectID:      project.ID,
		OrganizationID: "org-a",
	})
	if err != nil {
		t.Fatalf("CreateMission: %v", err)
	}
	if len(missao.PresetCompletionCriteria) != 0 {
		t.Fatalf("sem preset nao devia haver criterios, vieram %d", len(missao.PresetCompletionCriteria))
	}
	if missao.PresetBudgetSeconds != 0 {
		t.Fatalf("sem preset nao devia haver orcamento, veio %d", missao.PresetBudgetSeconds)
	}
}

// TestTodosOsPresetsComCriteriosAplicamNoRegisto prova que nenhum preset perde os criterios.
func TestTodosOsPresetsComCriteriosAplicamNoRegisto(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	for _, preset := range BuiltInMissionPresets() {
		if len(preset.CompletionCriteria) == 0 {
			continue
		}
		missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
			Objective:      "aplicar " + preset.ID,
			ProjectID:      project.ID,
			OrganizationID: "org-a",
			PresetID:       preset.ID,
		})
		if err != nil {
			t.Errorf("%s: %v", preset.ID, err)
			continue
		}
		if len(missao.PresetCompletionCriteria) != len(preset.CompletionCriteria) {
			t.Errorf("%s: criterios perdidos (%d de %d)",
				preset.ID, len(missao.PresetCompletionCriteria), len(preset.CompletionCriteria))
		}
	}
}
