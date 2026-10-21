package agent

// FASE 23 do adendo V3: o preset escolhido na UI tem de CHEGAR ao registo da missao.
//
// O defeito que estes testes impedem e' o que o proprio `types.go` documenta para o ActorID:
// *«a atribuicao existia no pedido e perdia-se no registo».* Antes desta fatia o
// `CreateMissionRequest` nao tinha `PresetID`, a UI guardava a escolha em estado React e o
// servidor nunca a via -- a superficie mostrava um preset e a missao nascia sem ele.

import (
	"context"
	"errors"
	"testing"
)

// TestCreateMissionRecusaPresetInexistente e' o teste que DISTINGUE a validacao.
//
// *Um preset que nao existe nao pode produzir uma missao com a configuracao errada em silencio.*
func TestCreateMissionRecusaPresetInexistente(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	_, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:      "tarefa com preset inventado",
		ProjectID:      project.ID,
		OrganizationID: "org-a",
		PresetID:       "preset-que-nao-existe",
	})
	if !errors.Is(err, ErrPresetInvalido) {
		t.Fatalf("esperado ErrPresetInvalido, obtido %v", err)
	}
}

// TestCreateMissionGuardaOPresetNoRegisto prova que o preset nao se perde a caminho do store.
func TestCreateMissionGuardaOPresetNoRegisto(t *testing.T) {
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
	if missao.PresetID != "briefing-diario" {
		t.Fatalf("a missao devolvida nao guardou o preset: %q", missao.PresetID)
	}
	// E tem de estar no STORE, nao so' na struct devolvida: *um campo que so' existe no
	// retorno e' um campo que se perde no proximo carregamento.*
	recarregada, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatalf("GetMission: %v", err)
	}
	if recarregada.PresetID != "briefing-diario" {
		t.Fatalf("o preset nao sobreviveu ao store: %q", recarregada.PresetID)
	}
}

// TestCreateMissionSemPresetContinuaAFuncionar prova a compatibilidade: vazio = sem preset.
func TestCreateMissionSemPresetContinuaAFuncionar(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:      "tarefa sem preset",
		ProjectID:      project.ID,
		OrganizationID: "org-a",
	})
	if err != nil {
		t.Fatalf("CreateMission sem preset: %v", err)
	}
	if missao.PresetID != "" {
		t.Fatalf("missao sem preset devia ter PresetID vazio, veio %q", missao.PresetID)
	}
}

// TestTodosOsPresetsEmbutidosSaoAceites prova que o catalogo e o validador concordam.
//
// *Um validador que recusa um preset do proprio catalogo e' um catalogo pela metade.*
func TestTodosOsPresetsEmbutidosSaoAceites(t *testing.T) {
	runtime, _, project := idempotencyRuntime(t)
	for _, preset := range BuiltInMissionPresets() {
		_, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
			Objective:      "validar preset " + preset.ID,
			ProjectID:      project.ID,
			OrganizationID: "org-a",
			PresetID:       preset.ID,
		})
		if err != nil {
			t.Errorf("preset %q do catalogo foi recusado: %v", preset.ID, err)
		}
	}
}
