package agent

// FASE 23 do adendo V3: presets de missao como configuracao versionada, revisavel e testavel.
//
// O teste que DISTINGUE esta fase e' TestPresetSemCriterioDeConclusaoERecusado: ele falha se a
// validacao passar a aceitar um preset que nunca acaba.

import (
	"errors"
	"strings"
	"testing"
)

// TestCatalogoDePresetsEValido prova que os 11 presets embutidos passam a validacao.
func TestCatalogoDePresetsEValido(t *testing.T) {
	presets := BuiltInMissionPresets()
	if len(presets) < 11 {
		t.Fatalf("catalogo tem %d presets, esperado pelo menos 11", len(presets))
	}
	for _, p := range presets {
		if err := p.Validate(); err != nil {
			t.Errorf("preset %q invalido: %v", p.ID, err)
		}
	}
	if err := ValidateAllPresets(); err != nil {
		t.Fatalf("ValidateAllPresets: %v", err)
	}
}

// TestPresetSemCriterioDeConclusaoERecusado e' o teste que DISTINGUE a fase.
//
// Um preset sem criterio de conclusao e' um preset que nunca acaba. Sem esta validacao, o catalogo
// aceitaria "fazer o melhor que conseguir" como se fosse um criterio.
func TestPresetSemCriterioDeConclusaoERecusado(t *testing.T) {
	p := MissionPreset{
		ID:   "sem-fim",
		Name: "Sem fim",
		Goal: "Fazer algo",
		// CompletionCriteria deliberadamente ausente.
	}
	err := p.Validate()
	if !errors.Is(err, ErrPresetInvalido) {
		t.Fatalf("esperado ErrPresetInvalido, obtido %v", err)
	}
	if !strings.Contains(err.Error(), "completion criteria") {
		t.Fatalf("o erro devia nomear o criterio em falta: %v", err)
	}
	// E um criterio VAZIO tambem nao serve: um campo preenchido com espacos e' um campo vazio
	// com boa apresentacao.
	vazio := MissionPreset{ID: "x", Name: "X", Goal: "Y", CompletionCriteria: []string{"   "}}
	if err := vazio.Validate(); !errors.Is(err, ErrPresetInvalido) {
		t.Fatalf("criterio vazio devia ser recusado, obtido %v", err)
	}
}

// TestPresetComScheduleInvalidoERecusado prova que nao se promete um agendamento que nao se cumpre.
func TestPresetComScheduleInvalidoERecusado(t *testing.T) {
	p := MissionPreset{
		ID: "cron-inventado", Name: "Cron", Goal: "G",
		Schedule:           "*/5 * * * *",
		CompletionCriteria: []string{"termina"},
	}
	err := p.Validate()
	if !errors.Is(err, ErrPresetInvalido) {
		t.Fatalf("schedule arbitrario devia ser recusado, obtido %v", err)
	}
	if !strings.Contains(err.Error(), "schedule") {
		t.Fatalf("o erro devia nomear o schedule: %v", err)
	}
}

// TestPresetComOrcamentoNegativoERecusado prova que um tecto negativo nao passa.
func TestPresetComOrcamentoNegativoERecusado(t *testing.T) {
	p := MissionPreset{
		ID: "negativo", Name: "N", Goal: "G", BudgetTokens: -1,
		CompletionCriteria: []string{"termina"},
	}
	if err := p.Validate(); !errors.Is(err, ErrPresetInvalido) {
		t.Fatalf("orcamento negativo devia ser recusado, obtido %v", err)
	}
}

// TestMissionPresetByIDEncontraEOSeNaoExiste prova a busca por id.
func TestMissionPresetByIDEncontraEOSeNaoExiste(t *testing.T) {
	p, ok := MissionPresetByID("briefing-diario")
	if !ok {
		t.Fatal("preset briefing-diario nao encontrado")
	}
	if p.ID != "briefing-diario" || p.Schedule != "daily" {
		t.Fatalf("preset errado: %+v", p)
	}
	if _, ok := MissionPresetByID("nao-existe"); ok {
		t.Fatal("preset inexistente devia dar ok=false")
	}
	if _, ok := MissionPresetByID("  "); ok {
		t.Fatal("id vazio devia dar ok=false")
	}
}

// TestIDsDePresetSaoUnicos prova que dois presets nao colidem.
func TestIDsDePresetSaoUnicos(t *testing.T) {
	vistos := map[string]int{}
	for _, p := range BuiltInMissionPresets() {
		vistos[p.ID]++
	}
	for id, n := range vistos {
		if n > 1 {
			t.Fatalf("id duplicado: %s aparece %d vezes", id, n)
		}
	}
	// O catalogo e' ordenado por ID: duas execucoes tem de dar a MESMA ordem, e a ordem tem de
	// ser a alfabetica -- *um catalogo que muda de ordem entre execucoes nao e' um catalogo.*
	presets := BuiltInMissionPresets()
	for i := 1; i < len(presets); i++ {
		if presets[i-1].ID > presets[i].ID {
			t.Fatalf("catalogo fora de ordem: %q antes de %q", presets[i-1].ID, presets[i].ID)
		}
	}
}

// TestPresetNaoSeAutoAutoriza prova que Permissions e' PEDIDO, nao CONCESSAO.
//
// O preset declara o que precisa; nada nele outorga. Este teste protege contra a tentacao de
// transformar o campo num bypass: se alguem acrescentar um campo de concessao automatica, este
// teste falha na compilacao ou na assercao.
func TestPresetNaoSeAutoAutoriza(t *testing.T) {
	for _, p := range BuiltInMissionPresets() {
		if err := p.Validate(); err != nil {
			t.Fatalf("preset %q: %v", p.ID, err)
		}
		// Nenhum preset pode declarar ausencia de aprovacoes E permissao de alto impacto ao mesmo
		// tempo. Um preset que pede escrita no filesystem sem nenhum ponto de aprovacao declarado
		// esta a esconder o custo de quem o escolhe.
		temPermissaoDeEscrita := false
		for _, perm := range p.Permissions {
			if strings.Contains(perm, "write") || strings.Contains(perm, "spawn") {
				temPermissaoDeEscrita = true
			}
		}
		if temPermissaoDeEscrita && len(p.Approvals) == 0 {
			t.Errorf("preset %q pede escrita sem declarar nenhum ponto de aprovacao", p.ID)
		}
	}
}
