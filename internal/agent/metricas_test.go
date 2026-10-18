package agent

// FASE 21 do adendo V3 — os discriminadores da telemetria.
//
// O que estes testes recusam aceitar:
//   - uma omissao lida como zero (*uma omissao silenciosa le-se como zero*);
//   - um percentil interpolado, que inventa um valor que nenhuma medicao produziu;
//   - uma media de medias a passar por tokens/segundo;
//   - um cancelamento do utilizador contado como falha do modelo;
//   - um previsto que mentiu e nao aparece.

import "testing"

// TestSemAmostrasTodosOsEixosSaoNaoMedidos e' o discriminador central. *Devolver zeros sem os
// declarar seria pior do que devolver nada: pareceria que se mediu e deu zero.*
func TestSemAmostrasTodosOsEixosSaoNaoMedidos(t *testing.T) {
	resumo := ResumirMetricas(nil)
	if resumo.Amostras != 0 {
		t.Fatalf("amostras = %d", resumo.Amostras)
	}
	declarados := map[string]bool{}
	for _, eixo := range resumo.EixosNaoMedidos {
		declarados[eixo] = true
	}
	for _, esperado := range []string{"latencia", "ttft", "tokens", "custo", "energia", "vram"} {
		if !declarados[esperado] {
			t.Fatalf("o eixo %q tinha de ser declarado NAO MEDIDO; veio %v", esperado, resumo.EixosNaoMedidos)
		}
	}
	// E os ponteiros tem de ser nil, nao zero.
	if resumo.EnergiaMilijoules != nil {
		t.Fatal("energia tem de ser nil (nao medido), nao zero")
	}
	if resumo.VRAMMiB != nil {
		t.Fatal("vram tem de ser nil (nao medido), nao zero")
	}
	if resumo.TTFT != nil {
		t.Fatal("ttft tem de ser nil (nao medido), nao uma faixa zerada")
	}
}

// TestEnergiaAusenteNaoViraZero: nenhuma amostra mediu energia -> `nil` e o eixo declarado.
// *Um `int64` zero seria uma medida legitima ("consumiu 0 mJ"); `nil` e' a ausencia de medida.*
func TestEnergiaAusenteNaoViraZero(t *testing.T) {
	resumo := ResumirMetricas([]AmostraDeInferencia{
		{
			ModelID: "a", Provider: "p", LatenciaMS: 100, TokensSaida: 10, Sucesso: true,
			CustoPrevistoCentavos: 1, CustoRealizadoCentavos: 1,
		},
	})
	if resumo.EnergiaMilijoules != nil {
		t.Fatalf("sem medicao de energia o campo tem de ser nil, veio %d", *resumo.EnergiaMilijoules)
	}
	if resumo.VRAMMiB != nil {
		t.Fatalf("sem medicao de vram o campo tem de ser nil, veio %d", *resumo.VRAMMiB)
	}
	achouEnergia := false
	for _, eixo := range resumo.EixosNaoMedidos {
		if eixo == "energia" {
			achouEnergia = true
		}
	}
	if !achouEnergia {
		t.Fatalf("`energia` tem de constar em EixosNaoMedidos; veio %v", resumo.EixosNaoMedidos)
	}
}

// TestEnergiaMedidaAparece: quando UMA amostra mede, o eixo deixa de ser nao-medido e o total
// e' a soma -- nao a media.
func TestEnergiaMedidaAparece(t *testing.T) {
	resumo := ResumirMetricas([]AmostraDeInferencia{
		{
			ModelID: "a", Provider: "p", LatenciaMS: 100, TokensSaida: 10, Sucesso: true,
			EnergiaMilijoules: 500, VRAMMiB: 1024,
		},
		{
			ModelID: "a", Provider: "p", LatenciaMS: 100, TokensSaida: 10, Sucesso: true,
			EnergiaMilijoules: 300, VRAMMiB: 2048,
		},
	})
	if resumo.EnergiaMilijoules == nil {
		t.Fatal("com duas amostras medidas a energia nao pode ser nil")
	}
	if *resumo.EnergiaMilijoules != 800 {
		t.Fatalf("a energia tem de ser a SOMA (800), veio %d", *resumo.EnergiaMilijoules)
	}
	if resumo.VRAMMiB == nil || *resumo.VRAMMiB != 2048 {
		t.Fatalf("a VRAM tem de ser o PICO (2048), veio %v", resumo.VRAMMiB)
	}
	for _, eixo := range resumo.EixosNaoMedidos {
		if eixo == "energia" || eixo == "vram" {
			t.Fatalf("o eixo %q nao pode constar como nao medido depois de medido", eixo)
		}
	}
}

// TestPercentilNearestRankNaoInterpola: com 4 amostras (10,20,30,40) o p50 nearest-rank e' 20
// (indice ceil(0.5*4)-1 = 1), nao a media 25 de uma interpolacao.
func TestPercentilNearestRankNaoInterpola(t *testing.T) {
	valores := []int64{10, 20, 30, 40}
	casos := []struct {
		p        int
		esperado int64
	}{
		{1, 10},
		{25, 10},
		{50, 20},
		{75, 30},
		{95, 40},
		{99, 40},
		{100, 40},
	}
	for _, caso := range casos {
		if obtido := Percentil(valores, caso.p); obtido != caso.esperado {
			t.Fatalf("p%d = %d, esperado %d (nearest-rank, sem interpolacao)", caso.p, obtido, caso.esperado)
		}
	}
}

// TestTokensPorSegundoUsaSomaSobreSoma: *a media de medias da' peso igual a uma chamada de
// 100 ms e a uma de 10 s -- e o numero que sai nao descreve nenhuma das duas.*
//
// Amostra A: 100 tokens em  100 ms -> 1000 tok/s
// Amostra B: 1000 tokens em 10000 ms -> 100 tok/s
//
//	media de medias  = 550 tok/s     (ERRADO)
//	soma sobre soma  = 1100 / 10,1 s = 108,91 tok/s (CORRECTO)
func TestTokensPorSegundoUsaSomaSobreSoma(t *testing.T) {
	resumo := ResumirMetricas([]AmostraDeInferencia{
		{ModelID: "a", Provider: "p", LatenciaMS: 100, TokensSaida: 100, Sucesso: true},
		{ModelID: "a", Provider: "p", LatenciaMS: 10000, TokensSaida: 1000, Sucesso: true},
	})
	esperado := 1100.0 / 10.1
	diferenca := resumo.TokensPorSegundo - esperado
	if diferenca < -0.01 || diferenca > 0.01 {
		t.Fatalf("tokens/s = %.4f, esperado %.4f (soma sobre soma); se desse 550 seria media de medias",
			resumo.TokensPorSegundo, esperado)
	}
	if resumo.TokensPorSegundo > 500 {
		t.Fatalf("tokens/s = %.2f -- demasiado alto, parece media de medias", resumo.TokensPorSegundo)
	}
}

// TestCancelamentoNaoContaComoFalha: *contar um cancelamento pedido pelo utilizador como falha
// do modelo seria culpar o modelo por uma decisao humana.*
func TestCancelamentoNaoContaComoFalha(t *testing.T) {
	resumo := ResumirMetricas([]AmostraDeInferencia{
		{ModelID: "a", Provider: "p", LatenciaMS: 100, Sucesso: true},
		{ModelID: "a", Provider: "p", LatenciaMS: 200, Sucesso: false, Cancelada: true},
	})
	if resumo.Cancelamentos != 1 {
		t.Fatalf("cancelamentos = %d", resumo.Cancelamentos)
	}
	if resumo.Falhas != 0 {
		t.Fatalf("falhas = %d -- um cancelamento nao e' falha do modelo", resumo.Falhas)
	}
	if resumo.TaxaDeSucesso != 1.0 {
		t.Fatalf("taxa de sucesso = %.3f, esperado 1.0 (1 sucesso, 0 falhas)", resumo.TaxaDeSucesso)
	}
}

// TestDesvioDeCustoMostraQuandoOPrevistoMentiu: separar previsto de realizado e' o que permite
// detectar que o previsto estava errado.
func TestDesvioDeCustoMostraQuandoOPrevistoMentiu(t *testing.T) {
	resumo := ResumirMetricas([]AmostraDeInferencia{
		{
			ModelID: "a", Provider: "p", LatenciaMS: 100, Sucesso: true,
			CustoPrevistoCentavos: 10, CustoRealizadoCentavos: 25,
		},
	})
	if resumo.DesvioDeCustoCentavos != 15 {
		t.Fatalf("desvio = %d, esperado +15 (gastou 15 mais do que previu)",
			resumo.DesvioDeCustoCentavos)
	}
	if resumo.CustoPrevistoCentavos != 10 || resumo.CustoRealizadoCentavos != 25 {
		t.Fatalf("previsto=%d realizado=%d", resumo.CustoPrevistoCentavos, resumo.CustoRealizadoCentavos)
	}
}

// TestLatenciaExcluiAmostrasQueFalharam: misturar a latencia de um timeout com a de uma chamada
// boa descreveria uma chamada que nao existe.
func TestLatenciaExcluiAmostrasQueFalharam(t *testing.T) {
	resumo := ResumirMetricas([]AmostraDeInferencia{
		{ModelID: "a", Provider: "p", LatenciaMS: 100, Sucesso: true},
		{ModelID: "a", Provider: "p", LatenciaMS: 30000, Sucesso: false},
	})
	if resumo.Latencia.Max != 100 {
		t.Fatalf("a latencia so' pode contar as bem-sucedidas (max=100), veio %d", resumo.Latencia.Max)
	}
}

// TestTTFTNilQuandoNinguemMediu: TTFT so' existe em chamadas de streaming.
func TestTTFTNilQuandoNinguemMediu(t *testing.T) {
	semTTFT := ResumirMetricas([]AmostraDeInferencia{
		{ModelID: "a", Provider: "p", LatenciaMS: 100, Sucesso: true},
	})
	if semTTFT.TTFT != nil {
		t.Fatalf("sem medicao de ttft o campo tem de ser nil, veio %+v", semTTFT.TTFT)
	}
	comTTFT := ResumirMetricas([]AmostraDeInferencia{
		{ModelID: "a", Provider: "p", LatenciaMS: 100, TTFTMS: 30, Sucesso: true},
	})
	if comTTFT.TTFT == nil || comTTFT.TTFT.P50 != 30 {
		t.Fatalf("com medicao de ttft o campo tem de trazer a faixa (p50=30), veio %+v", comTTFT.TTFT)
	}
}

// TestComparacaoPorModeloOrdenaPelaCauda: *ordenar pela media esconderia a cauda, e a cauda e'
// o que o utilizador sente.*
//
// O cenario tem de fazer a ORDEM DIVERGIR entre media e p95 -- senao o teste passaria com
// qualquer um dos dois criterios e nao discriminaria nada. Duas tentativas anteriores falharam
// por ERRO MEU no cenario, e cada falha ensinou o que o nearest-rank faz:
//
//	1a: 9 amostras de 10 ms + 1 de 10 s. Achei que p95 seria 10 ms; e' 10 s (o outlier e' a
//	    ultima amostra e o p95 nearest-rank apanha-a).
//	2a: 95 de 100 ms + 5 de 5 s. Achei que p95 seria 5 s; com 5% de outliers o p95 calha
//	    EXACTAMENTE no ultimo valor bom (indice 94 de 100) e continua 100 ms. Um outlier so'
//	    entra no p95 quando ocupa MAIS de 5% da amostra.
//
// Este cenario (10% de outliers):
//
//	modelo "instavel": 90 amostras de 100 ms + 10 de 5000 ms
//	    media = (90*100 + 10*5000) / 100 = 590 ms
//	    p95   = 5000 ms  (indice 94 de 100 -- dentro da cauda de 10%)
//	modelo "uniforme": 100 amostras de 600 ms
//	    media = 600 ms
//	    p95   = 600 ms
//
//	media: instavel (590) < uniforme (600)  -> instavel primeiro  (ERRADO para o utilizador)
//	p95:   uniforme (600) < instavel (5000) -> uniforme primeiro  (CORRECTO)
func TestComparacaoPorModeloOrdenaPelaCauda(t *testing.T) {
	instavel := make([]AmostraDeInferencia, 0, 100)
	uniforme := make([]AmostraDeInferencia, 0, 100)
	for range 90 {
		instavel = append(instavel, AmostraDeInferencia{
			ModelID: "instavel", Provider: "p",
			LatenciaMS: 100, Sucesso: true,
		})
	}
	for range 10 {
		instavel = append(instavel, AmostraDeInferencia{
			ModelID: "instavel", Provider: "p",
			LatenciaMS: 5000, Sucesso: true,
		})
	}
	for range 100 {
		uniforme = append(uniforme, AmostraDeInferencia{
			ModelID: "uniforme", Provider: "p",
			LatenciaMS: 600, Sucesso: true,
		})
	}

	comparacao := ComparacaoPorModelo(append(instavel, uniforme...))
	if len(comparacao) != 2 {
		t.Fatalf("esperados 2 modelos, vieram %d", len(comparacao))
	}
	porModelo := map[string]ResumoDeMetricas{}
	for _, linha := range comparacao {
		porModelo[linha.Modelo] = linha.Resumo
	}
	// Confirmar que o cenario diverge mesmo -- senao o teste nao prova nada.
	if porModelo["p/instavel"].Latencia.Media >= porModelo["p/uniforme"].Latencia.Media {
		t.Fatalf("o cenario tem de ter media(instavel)=%d < media(uniforme)=%d, senao nao discrimina",
			porModelo["p/instavel"].Latencia.Media, porModelo["p/uniforme"].Latencia.Media)
	}
	if porModelo["p/instavel"].Latencia.P95 <= porModelo["p/uniforme"].Latencia.P95 {
		t.Fatalf("o cenario tem de ter p95(instavel)=%d > p95(uniforme)=%d, senao nao discrimina",
			porModelo["p/instavel"].Latencia.P95, porModelo["p/uniforme"].Latencia.P95)
	}
	// A ordenacao TEM de seguir o p95, nao a media.
	if comparacao[0].Modelo != "p/uniforme" {
		t.Fatalf("a ordenacao tem de ser por p95 -- o uniforme (p95=600) vem antes do instavel "+
			"(p95=5000), apesar de a media do instavel ser MENOR (590 vs 600); veio primeiro %q",
			comparacao[0].Modelo)
	}
}

// TestResumoDeterministico: duas chamadas com as mesmas amostras dao' o mesmo resultado.
func TestResumoDeterministico(t *testing.T) {
	amostras := []AmostraDeInferencia{
		{ModelID: "a", Provider: "p", LatenciaMS: 10, TokensSaida: 5, Sucesso: true},
		{ModelID: "b", Provider: "p", LatenciaMS: 10, TokensSaida: 5, Sucesso: true},
	}
	primeira := ComparacaoPorModelo(amostras)
	segunda := ComparacaoPorModelo(amostras)
	for i := range primeira {
		if primeira[i].Modelo != segunda[i].Modelo {
			t.Fatalf("ordenacao nao determinista na posicao %d: %q vs %q",
				i, primeira[i].Modelo, segunda[i].Modelo)
		}
	}
}
