package agent

// FASE 23 do adendo V3: o preset governa o PLANO.
//
// Ate' aqui o preset governava o orcamento e os criterios de conclusao -- o que muda a
// FACTURA. Estes testes provam o que muda o TRABALHO: quais passos existem.
//
// Os discriminadores que falham sem a mudanca sao os dois primeiros: sem o filtro,
// `conversa-simples` (que nao declara testes) produziria um passo `project.test.run`.

import (
	"context"
	"strings"
	"testing"
)

func planoPara(t *testing.T, mission Mission) []Step {
	t.Helper()
	passos, err := RulePlanner{}.Plan(context.Background(), mission)
	if err != nil {
		t.Fatalf("Plan devolveu erro: %v", err)
	}
	return passos
}

func kindsDoPlano(passos []Step) []string {
	kinds := make([]string, 0, len(passos))
	for _, p := range passos {
		kinds = append(kinds, p.Kind)
	}
	return kinds
}

func temKind(passos []Step, kind string) bool {
	for _, p := range passos {
		if p.Kind == kind {
			return true
		}
	}
	return false
}

// TestPresetSemTestesNaoGeraPassoDeTestes e' o discriminador principal.
//
// `conversa-simples` nao declara `Tests`. Um objectivo que menciona "testar" nao pode
// obrigar o plano a testar quando o preset escolhido nao pede testes.
//
// *Um preset que nao declara testes nao gera um passo de testes: seria trabalho que o
// preset nao pediu.*
func TestPresetSemTestesNaoGeraPassoDeTestes(t *testing.T) {
	passos := planoPara(t, Mission{
		Objective: "testar e corrigir o pacote",
		PresetID:  "conversa-simples",
	})
	if temKind(passos, "project.test.run") {
		t.Fatalf("conversa-simples nao declara testes, mas o plano tem project.test.run: %v",
			kindsDoPlano(passos))
	}
	if len(passos) == 0 {
		t.Fatal("o plano ficou vazio: um preset restritivo nao pode anular o trabalho")
	}
}

func TestPresetComTestesGeraPassoDeTestes(t *testing.T) {
	passos := planoPara(t, Mission{
		Objective: "testar e corrigir o pacote",
		PresetID:  "construcao-aplicativo",
	})
	if !temKind(passos, "project.test.run") {
		t.Fatalf("construcao-aplicativo declara testes, mas o plano nao tem project.test.run: %v",
			kindsDoPlano(passos))
	}
}

// TestPresetSemMergeNaoGeraMerge e' o segundo discriminador.
//
// `construcao-aplicativo` nao declara a accao `merge`. Mesmo com objectivo e worktree a
// pedir integracao, o plano nao integra sozinho.
//
// *Integrar de volta e' uma accao que o preset tem de pedir explicitamente, nao um extra
// que o planeador oferece.*
func TestPresetSemMergeNaoGeraMerge(t *testing.T) {
	passos := planoPara(t, Mission{
		Objective:         "testar e fazer merge no repositorio",
		PresetID:          "construcao-aplicativo",
		GitWorktreeActive: true,
		GitRepoRoot:       "/tmp/repo",
		GitBranch:         "agent/x",
		GitWorktreePath:   "/tmp/repo/.agent-worktrees/x",
	})
	if temKind(passos, "git.merge.origin") {
		t.Fatalf("construcao-aplicativo nao declara merge, mas o plano integra: %v",
			kindsDoPlano(passos))
	}
}

func TestPresetComMergeGeraMerge(t *testing.T) {
	passos := planoPara(t, Mission{
		Objective:         "testar e corrigir o pacote",
		PresetID:          "assistente-codigo",
		GitWorktreeActive: true,
		GitRepoRoot:       "/tmp/repo",
		GitBranch:         "agent/x",
		GitWorktreePath:   "/tmp/repo/.agent-worktrees/x",
	})
	if !temKind(passos, "git.merge.origin") {
		t.Fatalf("assistente-codigo declara merge, mas o plano nao integra: %v",
			kindsDoPlano(passos))
	}
	for _, p := range passos {
		if p.Kind == "git.merge.origin" && !p.RequiresApproval {
			t.Fatal("o passo de merge TEM de exigir aprovacao")
		}
	}
}

// TestSemPresetMantemPlanoAnterior e' a prova de retrocompatibilidade.
//
// Sem a minha mudanca este teste passaria; com ela, tem de continuar a passar. Um preset
// e' opt-in: quem nao escolhe nenhum nao pode ver o plano mudar.
func TestSemPresetMantemPlanoAnterior(t *testing.T) {
	passos := planoPara(t, Mission{Objective: "testar o pacote"})
	kinds := kindsDoPlano(passos)
	if len(kinds) != 2 || kinds[0] != "workspace.write" || kinds[1] != "project.test.run" {
		t.Fatalf("sem preset o plano tem de ser o historico [workspace.write project.test.run], veio %v", kinds)
	}
	// E nenhum passo pode dizer que foi governado por um preset que nao existe.
	for _, p := range passos {
		if _, tem := p.Input["preset_id"]; tem {
			t.Fatalf("sem preset nenhum passo pode registar preset_id: %v", p.Input)
		}
	}
}

// TestPresetDesconhecidoFalhaFechado: um erro de escrita no id nao vira "plano sem preset".
func TestPresetDesconhecidoFalhaFechado(t *testing.T) {
	_, err := RulePlanner{}.Plan(context.Background(), Mission{
		Objective: "testar",
		PresetID:  "preset-que-nao-existe",
	})
	if err == nil {
		t.Fatal("preset desconhecido devia devolver erro em vez de plano silencioso")
	}
	if !strings.Contains(err.Error(), "preset-que-nao-existe") {
		t.Fatalf("o erro devia nomear o id: %v", err)
	}
}

// TestPassosSaoRenumerados: apos o filtro, um plano com step_2 sem step_1 mentiria sobre a
// sua propria ordem.
func TestPassosSaoRenumerados(t *testing.T) {
	passos := planoPara(t, Mission{
		Objective: "testar e corrigir",
		PresetID:  "conversa-simples",
	})
	if len(passos) == 0 {
		t.Fatal("plano vazio")
	}
	for i, p := range passos {
		querido := "step_" + string(rune('1'+i))
		if p.ID != querido {
			t.Fatalf("passo %d tem ID %q, queria %q (plano: %v)", i, p.ID, querido, kindsDoPlano(passos))
		}
	}
}

// TestCadaPassoRegistaOPreset: sem isto, um plano filtrado e' indistinguivel de um plano
// que nunca teve aqueles passos.
func TestCadaPassoRegistaOPreset(t *testing.T) {
	passos := planoPara(t, Mission{
		Objective: "testar e corrigir",
		PresetID:  "construcao-aplicativo",
	})
	if len(passos) == 0 {
		t.Fatal("plano vazio")
	}
	for _, p := range passos {
		if p.Input == nil {
			t.Fatalf("passo %s sem Input", p.ID)
		}
		if p.Input["preset_id"] != "construcao-aplicativo" {
			t.Fatalf("passo %s nao regista o preset: %v", p.ID, p.Input)
		}
	}
}

// TestPresetGovernaPlanoUsandoKindsReais: o preset nao pode introduzir um Kind que o
// executor nao conhece.
//
// *Um Kind que o executor nao sabe executar seria um passo que falha no fim, e um plano
// que so' falha depois de correr nao e' um plano.*
func TestPresetGovernaPlanoUsandoKindsReais(t *testing.T) {
	conhecidos := map[string]bool{
		"workspace.write":  true,
		"workspace.list":   true,
		"project.test.run": true,
		"git.merge.origin": true,
		"browser.operator": true,
	}
	ids := MissionPresetIDs()
	if len(ids) == 0 {
		t.Fatal("catalogo de presets vazio")
	}
	for _, id := range ids {
		passos := planoPara(t, Mission{
			Objective:         "testar, navegar no browser e escrever",
			PresetID:          id,
			GitWorktreeActive: true,
			GitRepoRoot:       "/tmp/repo",
			GitBranch:         "agent/x",
			GitWorktreePath:   "/tmp/repo/.agent-worktrees/x",
		})
		for _, p := range passos {
			if !conhecidos[p.Kind] {
				t.Fatalf("preset %q gerou Kind desconhecido %q: o executor nao o sabe correr", id, p.Kind)
			}
		}
	}
}

// TestCadaPassoRegistadoSobreviveAoPreset: um preset restritivo nao pode produzir um plano
// vazio -- um plano vazio e' um no-op disfarcado de sucesso.
func TestNenhumPresetProduzPlanoVazio(t *testing.T) {
	for _, id := range MissionPresetIDs() {
		passos := planoPara(t, Mission{Objective: "inspecionar o workspace", PresetID: id})
		if len(passos) == 0 {
			t.Fatalf("preset %q produziu um plano VAZIO para um objectivo que tem caminho", id)
		}
	}
}
