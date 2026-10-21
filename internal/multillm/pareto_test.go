package multillm

// FASE 19 do adendo V3: roteamento multiobjectivo e fronteira de Pareto.
//
// O discriminador central desta fatia e' `TestSomaDosFactorsIgualAoScore`: se os factors
// fossem decoracao (preenchidos a mao, ou calculados por um caminho diferente do score),
// a soma nao bateria. *Uma explicacao que nao soma ao resultado nao explica o resultado.*

import (
	"testing"
)

func registryDeTeste() *Registry {
	return &Registry{
		providers: map[string]Provider{
			"local":  {Name: "local", AllowPrivate: true},
			"remoto": {Name: "remoto", AllowPrivate: false},
		},
		models: map[string]Model{},
	}
}

func modelo(id, provider string, qualidade int, custo int64) Model {
	return Model{
		ID:                   id,
		Provider:             provider,
		QualityScore:         qualidade,
		CostPer1KInputCents:  custo,
		CostPer1KOutputCents: 0,
		Capabilities:         []string{"coding"},
	}
}

func candidatosDe(t *testing.T, modelos []Model, pref RoutePreference) []RouteDecision {
	t.Helper()
	registry := registryDeTeste()
	candidatos, err := registry.RouteCandidates(RouteRequest{
		RequiredCapabilities: []string{"coding"},
		SelectableModels:     modelos,
		Preference:           pref,
	})
	if err != nil {
		t.Fatalf("RouteCandidates devolveu erro: %v", err)
	}
	return candidatos
}

// TestSomaDosFactorsIgualAoScore e' o discriminador que impede factors decorativos.
func TestSomaDosFactorsIgualAoScore(t *testing.T) {
	modelos := []Model{
		modelo("local/bom", "local", 9, 0),
		modelo("remoto/medio", "remoto", 6, 3),
		modelo("remoto/caro", "remoto", 10, 40),
	}
	for _, pref := range []RoutePreference{
		RoutePreferenceAuto,
		RoutePreferenceBestQuality,
		RoutePreferenceCheapest,
		RoutePreferenceFastest,
		RoutePreferenceMostPrivate,
	} {
		for _, c := range candidatosDe(t, modelos, pref) {
			soma := int64(0)
			for _, valor := range c.Factors {
				soma += valor
			}
			if soma != c.Score {
				t.Fatalf("preferencia %q, modelo %s: soma dos factors = %d, score = %d (factors: %v)",
					pref, c.Model.ID, soma, c.Score, c.Factors)
			}
		}
	}
}

// TestFactorsExplicamOQueFoiConsiderado: um modelo gratuito tem de dizer que e' gratuito;
// um modelo com latencia medida tem de dizer que a latencia contou.
func TestFactorsExplicamOQueFoiConsiderado(t *testing.T) {
	modelos := []Model{modelo("local/free", "local", 7, 0)}
	c := candidatosDe(t, modelos, RoutePreferenceAuto)[0]
	if _, tem := c.Factors["gratuito"]; !tem {
		t.Fatalf("modelo de custo zero devia registar o factor `gratuito`: %v", c.Factors)
	}
	if _, tem := c.Factors["qualidade"]; !tem {
		t.Fatalf("a qualidade devia aparecer nos factors: %v", c.Factors)
	}
}

// TestParetoRemoveODominado: um modelo pior em TODOS os eixos sai da fronteira.
func TestParetoRemoveODominado(t *testing.T) {
	candidatos := []RouteDecision{
		{Model: modelo("bom", "local", 9, 1), Score: 900, Factors: map[string]int64{}},
		{Model: modelo("pior", "remoto", 3, 20), Score: 300, Factors: map[string]int64{}},
	}
	fronteira := ParetoFrontier(candidatos, nil)
	if len(fronteira) != 1 || fronteira[0].Model.ID != "bom" {
		t.Fatalf("o modelo dominado devia sair; fronteira = %v", idsDaFronteira(fronteira))
	}
}

// TestParetoMantemOsNaoDominados: um melhor em qualidade e outro em custo ficam AMBOS.
//
// *Escolher bem quando ha' um vencedor claro e' facil; escolher quando nao ha' e' que
// precisa de ser explicado.*
func TestParetoMantemOsNaoDominados(t *testing.T) {
	candidatos := []RouteDecision{
		{Model: modelo("qualidade", "remoto", 10, 50), Score: 1000},
		{Model: modelo("barato", "remoto", 5, 0), Score: 500},
	}
	fronteira := ParetoFrontier(candidatos, nil)
	if len(fronteira) != 2 {
		t.Fatalf("dois nao-dominados deviam ficar ambos; fronteira = %v", idsDaFronteira(fronteira))
	}
}

// TestParetoEmpateNaoEscondeNinguem: sem dominancia, ninguem sai.
func TestParetoEmpateNaoEscondeNinguem(t *testing.T) {
	candidatos := []RouteDecision{
		{Model: modelo("a", "remoto", 7, 5), Score: 700},
		{Model: modelo("b", "remoto", 7, 5), Score: 700},
	}
	fronteira := ParetoFrontier(candidatos, nil)
	if len(fronteira) != 2 {
		t.Fatalf("empate nao e' dominancia; fronteira = %v", idsDaFronteira(fronteira))
	}
}

// TestMostPrivatePrefereLocal sem torná-lo obrigatório.
func TestMostPrivatePrefereLocal(t *testing.T) {
	// Qualidade e custo IDENTICOS: a unica diferenca e' a localidade.
	modelos := []Model{
		modelo("remoto/igual", "remoto", 8, 0),
		modelo("local/igual", "local", 8, 0),
	}
	candidatos := candidatosDe(t, modelos, RoutePreferenceMostPrivate)
	if len(candidatos) == 0 {
		t.Fatal("sem candidatos")
	}
	if candidatos[0].Model.ID != "local/igual" {
		t.Fatalf("MostPrivate devia preferir o local; primeiro = %s (%v)",
			candidatos[0].Model.ID, candidatos[0].Factors)
	}
	if _, tem := candidatos[0].Factors["privacidade_local"]; !tem {
		t.Fatalf("a decisao devia registar o factor de privacidade: %v", candidatos[0].Factors)
	}
}

// TestMostPrivateNaoFiltraORemoto: a diferenca para LocalOnly e' ORDENAR vs FILTRAR.
//
// *Confundir as duas faz um utilizador que queria "preferir local" ficar sem servico
// quando o local cai -- sem saber porque.*
func TestMostPrivateNaoFiltraORemoto(t *testing.T) {
	modelos := []Model{modelo("remoto/unico", "remoto", 8, 0)}
	candidatos := candidatosDe(t, modelos, RoutePreferenceMostPrivate)
	if len(candidatos) != 1 {
		t.Fatalf("MostPrivate nao pode filtrar o remoto; ficaram %d candidatos", len(candidatos))
	}
}

// TestLocalOnlyContinuaAFiltrar: a restricao RIGIDA nao mudou de semantica.
func TestLocalOnlyContinuaAFiltrar(t *testing.T) {
	registry := registryDeTeste()
	_, err := registry.RouteCandidates(RouteRequest{
		RequiredCapabilities: []string{"coding"},
		SelectableModels:     []Model{modelo("remoto/x", "remoto", 9, 0)},
		LocalOnly:            true,
	})
	if err == nil {
		t.Fatal("LocalOnly TEM de recusar um provider remoto -- e' restricao, nao preferencia")
	}
}

// TestSemPreferenciaNaoMuda: retrocompatibilidade. A preferencia Auto nao pode alterar
// a ordenacao historica.
func TestSemPreferenciaNaoMuda(t *testing.T) {
	modelos := []Model{
		modelo("a/medio", "local", 5, 1),
		modelo("b/melhor", "remoto", 9, 1),
	}
	semPref := candidatosDe(t, modelos, RoutePreferenceAuto)
	if semPref[0].Model.ID != "b/melhor" {
		t.Fatalf("Auto devia escolher o de maior qualidade/custo igual; veio %s", semPref[0].Model.ID)
	}
	// E nao deve haver factor de privacidade quando a preferencia e' Auto.
	if _, tem := semPref[0].Factors["privacidade_local"]; tem {
		t.Fatalf("Auto nao devia registar preferencia de privacidade: %v", semPref[0].Factors)
	}
}

// TestParetoResumoDeterministico: duas leituras iguais tem de dar a mesma ordem.
//
// *Uma lista que muda de ordem entre duas leituras iguais faz duvidar do dado.*
func TestParetoResumoDeterministico(t *testing.T) {
	candidatos := []RouteDecision{
		{Model: modelo("zeta", "remoto", 5, 0), Score: 500},
		{Model: modelo("alfa", "remoto", 9, 9), Score: 900},
		{Model: modelo("meio", "remoto", 7, 4), Score: 700},
	}
	local := func(RouteDecision) bool { return false }
	a := ParetoResumo(candidatos, local)
	b := ParetoResumo(candidatos, local)
	if len(a) != len(b) {
		t.Fatalf("tamanhos diferentes: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ModelID != b[i].ModelID {
			t.Fatalf("ordem instavel na posicao %d: %s vs %s", i, a[i].ModelID, b[i].ModelID)
		}
	}
}

func idsDaFronteira(fronteira []RouteDecision) []string {
	ids := make([]string, 0, len(fronteira))
	for _, c := range fronteira {
		ids = append(ids, c.Model.ID)
	}
	return ids
}
