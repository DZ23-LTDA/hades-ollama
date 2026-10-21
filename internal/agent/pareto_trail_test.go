package agent

// FASE 19 do adendo V3, na trilha REAL do runtime.
//
// O teste irmao (`TestTrilhaDaDecisaoCarregaCandidatosECusto`) ja' provava que os preteridos e
// o custo chegam ao evento `step.model.route`. Estes dois provam o que faltava: os FACTORS de
// cada candidato e a FRONTEIRA DE PARETO das alternativas.
//
// O discriminador que importa e' o segundo: *um vencedor que nao esta' na fronteira de Pareto
// e' um vencedor DOMINADO -- foi escolhido apesar de haver outro melhor ou igual em todos os
// eixos.* Se o motor escolher um dominado, o evento tem de o mostrar; e se o escolhido estiver
// na fronteira, a fronteira tem de o conter.

import (
	"context"
	"testing"
)

func executarMissaoRoteada(t *testing.T) map[string]any {
	t.Helper()
	passos := []Step{{
		ID: "escolha", Kind: "workspace.list", Title: "escolher modelo",
		State: StepPending, RequiredCapabilities: []string{"coding"},
	}}
	runtime, id, eventos := missaoRoteada(t, &ferramentaQueAnotaModelo{}, passos, registryDeTeste(t), true)
	if err := runtime.Run(context.Background(), id); err != nil {
		t.Fatalf("executar a missao roteada: %v", err)
	}
	return payloadDoEvento(t, eventos(), "step.model.route")
}

// TestCadaCandidatoTrazOsFactoresDoScore: *um score sem os componentes que o somam obriga quem
// o le' a confiar em quem o escreveu -- e confianca nao e' um formato de evidencia.*
func TestCadaCandidatoTrazOsFactoresDoScore(t *testing.T) {
	pacote := executarMissaoRoteada(t)
	candidatos := candidatosDoPayload(t, pacote["candidates"])
	if len(candidatos) == 0 {
		t.Fatal("sem candidatos")
	}
	for indice, candidato := range candidatos {
		modelo, _ := candidato["model"].(string)
		factors, ok := candidato["factors"]
		if !ok {
			t.Fatalf("o candidato %d (%s) nao traz `factors`", indice, modelo)
		}
		mapa, ok := factors.(map[string]any)
		if !ok || len(mapa) == 0 {
			t.Fatalf("os `factors` do candidato %s tem de ser um mapa nao vazio (veio %T)", modelo, factors)
		}
		// A INVARIANTE: a soma dos factors tem de ser igual ao score do MESMO candidato.
		// *Uma explicacao que nao soma ao resultado nao explica o resultado.*
		score, ok := numeroDoPayload(candidato["score"])
		if !ok {
			t.Fatalf("o candidato %s nao traz `score`", modelo)
		}
		soma := float64(0)
		for nome, valor := range mapa {
			numero, ok := numeroDoPayload(valor)
			if !ok {
				t.Fatalf("o factor %q do candidato %s veio como %T", nome, modelo, valor)
			}
			soma += numero
		}
		if soma != score {
			t.Fatalf("candidato %s: soma dos factors = %v, score = %v (factors: %v)",
				modelo, soma, score, mapa)
		}
	}
}

// TestFronteiraDeParetoAcompanhaADecisao e' o discriminador da FASE 19.
func TestFronteiraDeParetoAcompanhaADecisao(t *testing.T) {
	pacote := executarMissaoRoteada(t)

	medido, ok := pacote["pareto_measured"].(bool)
	if !ok || !medido {
		t.Fatalf("o evento tem de DECLARAR `pareto_measured` verdadeiro (veio %v)", pacote["pareto_measured"])
	}
	fronteira := candidatosDoPayload(t, pacote["pareto_frontier"])
	if len(fronteira) == 0 {
		t.Fatal("a fronteira de Pareto nao pode ser vazia quando ha' candidatos medidos")
	}

	// O registry de teste oferece DOIS modelos para `coding`: `barato` (custo 2, qualidade 10) e
	// `forte` (custo 100, qualidade 90). Nenhum domina o outro -- um tem mais qualidade, o outro
	// tem menos custo. *Esconder o empate faz o utilizador achar que a escolha foi melhor do que
	// foi.*
	modelosDaFronteira := map[string]bool{}
	for _, alternativa := range fronteira {
		if modelo, ok := alternativa["model"].(string); ok {
			modelosDaFronteira[modelo] = true
		}
	}
	for _, esperado := range []string{"local/barato", "local/forte"} {
		if !modelosDaFronteira[esperado] {
			t.Fatalf("o candidato %q tem de estar na fronteira (nenhum domina o outro); fronteira=%v",
				esperado, modelosDaFronteira)
		}
	}

	// O DISCRIMINADOR: o vencedor tem de estar na fronteira. Se nao estiver, o motor escolheu um
	// candidato DOMINADO -- e o evento tem de o mostrar em vez de o esconder.
	vencedor, _ := pacote["model"].(string)
	if vencedor == "" {
		t.Fatal("o evento nao traz o modelo vencedor")
	}
	if !modelosDaFronteira[vencedor] {
		t.Fatalf("o vencedor %q NAO esta' na fronteira de Pareto -- *um vencedor dominado e' uma "+
			"escolha pior do que outra disponivel*; fronteira=%v", vencedor, modelosDaFronteira)
	}

	// E a fronteira nunca pode ser MAIOR que a lista de candidatos.
	candidatos := candidatosDoPayload(t, pacote["candidates"])
	if len(fronteira) > len(candidatos) {
		t.Fatalf("a fronteira (%d) nao pode exceder os candidatos (%d)", len(fronteira), len(candidatos))
	}
}

// TestFronteiraSemRegistryENaoMedida: sem registry, `nil` -- e o chamador declara. *`nil` significa
// NAO MEDIDO, nunca "nenhum candidato".*
func TestFronteiraSemRegistryENaoMedida(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatalf("runtime sem registry: %v", err)
	}
	if fronteira := runtime.fronteiraDeParetoDoPasso(nil); fronteira != nil {
		t.Fatalf("sem candidatos a fronteira tem de ser NIL, veio %v", fronteira)
	}
}
