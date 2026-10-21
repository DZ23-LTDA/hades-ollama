package agent

// FASE 3 do adendo V3: testes da comparacao objectiva de solucoes.
//
// O que estes testes travam:
//
//  1. teste vermelho desqualifica, por muito rapido que seja;
//  2. sem teste nenhum nao ha' ranking (ausencia de evidencia nao e' evidencia de nada);
//  3. medicao ausente NAO vira zero -- nao se inventa uma derrota;
//  4. empate NAO se desempata por sorte;
//  5. `IntegraAutomaticamente` e' SEMPRE falso;
//  6. o ranking obedece aos criterios correctos (menor-melhor vs maior-melhor);
//  7. criterios que nao foram medidos por todos sao REPORTADOS, nao escondidos.

import (
	"errors"
	"testing"
)

// medido e' um atalho para uma medicao valida.
func medido(c CriterioDeComparacao, valor float64) MedicaoDeCandidato {
	return MedicaoDeCandidato{Criterio: c, Valor: valor, Medido: true, Fonte: "teste"}
}

// naoMedido e' um atalho para uma medicao ausente.
func naoMedido(c CriterioDeComparacao) MedicaoDeCandidato {
	return MedicaoDeCandidato{Criterio: c, Medido: false}
}

// Teste vermelho desqualifica, mesmo sendo o mais rapido.
//
//	*Uma solucao rapida que nao funciona nao e' uma solucao rapida: e' uma solucao que falhou
//	depressa.*
func TestCompararDesqualificaTesteVermelho(t *testing.T) {
	res, err := CompararSolucoes([]Candidato{
		{
			ID: "rapida-e-quebrada", TestesPassaram: 10, TestesFalharam: 1,
			Medicoes: []MedicaoDeCandidato{medido(CriterioTempo, 1)},
		},
		{
			ID: "lenta-e-correcta", TestesPassaram: 10, TestesFalharam: 0,
			Medicoes: []MedicaoDeCandidato{medido(CriterioTempo, 100)},
		},
	})
	if err != nil {
		t.Fatalf("comparar: %v", err)
	}
	if res.Vencedor != "lenta-e-correcta" {
		t.Fatalf("vencedor = %q; esperado a correcta (a rapida falhou testes)", res.Vencedor)
	}
	if len(res.Rejeitados) != 1 || res.Rejeitados[0].CandidatoID != "rapida-e-quebrada" {
		t.Fatalf("rejeitados = %+v; esperado a rapida-e-quebrada", res.Rejeitados)
	}
	if res.Rejeitados[0].Motivo == "" {
		t.Fatal("rejeicao sem motivo")
	}
}

// Sem teste nenhum nao ha' ranking.
//
//	*A ausencia de evidencia nao e' evidencia de nada.*
func TestCompararSemTestesNaoRanqueia(t *testing.T) {
	res, err := CompararSolucoes([]Candidato{
		{ID: "sem-teste", TestesPassaram: 0, TestesFalharam: 0},
	})
	if err != nil {
		t.Fatalf("comparar: %v", err)
	}
	if len(res.Ranking) != 0 {
		t.Fatalf("ranking = %v; esperado vazio", res.Ranking)
	}
	if res.Vencedor != "" {
		t.Fatalf("vencedor = %q; esperado vazio", res.Vencedor)
	}
	if len(res.Rejeitados) != 1 {
		t.Fatalf("rejeitados = %d; esperado 1", len(res.Rejeitados))
	}
}

// Medicao ausente NAO vira zero: o criterio nao separa, e e' reportado.
//
//	*Marcar como zero o que nao foi medido e' inventar uma derrota.*
func TestCompararMedicaoAusenteNaoViraZero(t *testing.T) {
	res, err := CompararSolucoes([]Candidato{
		{
			ID: "a", TestesPassaram: 5,
			// O custo NAO foi medido em a.
			Medicoes: []MedicaoDeCandidato{naoMedido(CriterioCusto), medido(CriterioTempo, 10)},
		},
		{
			ID: "b", TestesPassaram: 5,
			// Em b foi medido e deu um valor alto.
			Medicoes: []MedicaoDeCandidato{medido(CriterioCusto, 500), medido(CriterioTempo, 20)},
		},
	})
	if err != nil {
		t.Fatalf("comparar: %v", err)
	}
	// O custo nao pode ter decidido: nao foi medido nos dois.
	// O custo foi medido em b e NAO em a: e' "medido parcialmente", nao "nao medido".
	//
	//	*Um criterio medido num candidato e nao noutro nao compara: informa.*
	achou := false
	for _, c := range res.MedidosParcialmente {
		if c == CriterioCusto {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("custo devia constar em medidos_parcialmente; veio %v (nao_medidos=%v)",
			res.MedidosParcialmente, res.NaoMedidos)
	}
	for _, c := range res.NaoMedidos {
		if c == CriterioCusto {
			t.Fatal("custo nao pode constar em nao_medidos: foi medido em um dos candidatos")
		}
	}
	// Quem decide e' o tempo (medido nos dois): 10 < 20, portanto "a".
	if res.Vencedor != "a" {
		t.Fatalf("vencedor = %q; esperado a (tempo menor, custo nao medido em ambos)", res.Vencedor)
	}
}

// Empate NAO se desempata por sorte.
//
//	*Escolher um deles seria decidir por um criterio que nao esta' escrito.*
func TestCompararEmpateNaoEscolhe(t *testing.T) {
	iguais := func(id string) Candidato {
		return Candidato{
			ID: id, TestesPassaram: 7,
			Medicoes: []MedicaoDeCandidato{
				medido(CriterioTempo, 30),
				medido(CriterioTamanhoDoDiff, 40),
			},
		}
	}
	res, err := CompararSolucoes([]Candidato{iguais("a"), iguais("b"), iguais("c")})
	if err != nil {
		t.Fatalf("comparar: %v", err)
	}
	if !res.Empate {
		t.Fatal("tres candidatos iguais deviam dar empate")
	}
	if res.Vencedor != "" {
		t.Fatalf("vencedor = %q; esperado vazio num empate", res.Vencedor)
	}
	if len(res.Ranking) != 3 {
		t.Fatalf("ranking = %v; esperado os 3", res.Ranking)
	}
}

// `IntegraAutomaticamente` e' SEMPRE falso, em todos os desfechos.
//
//	*O sistema pode recomendar um vencedor; nao pode escolher por quem assina o merge.*
func TestCompararNuncaIntegraAutomaticamente(t *testing.T) {
	casos := [][]Candidato{
		{},
		{{ID: "x", TestesPassaram: 1}},
		{{ID: "x", TestesPassaram: 1}, {ID: "y", TestesPassaram: 2}},
		{{ID: "x", TestesFalharam: 1}},
		{
			{ID: "a", TestesPassaram: 1, Medicoes: []MedicaoDeCandidato{medido(CriterioTempo, 1)}},
			{ID: "b", TestesPassaram: 1, Medicoes: []MedicaoDeCandidato{medido(CriterioTempo, 2)}},
		},
	}
	for i, candidatos := range casos {
		res, err := CompararSolucoes(candidatos)
		if errors.Is(err, ErrComparacaoSemCandidatos) {
			continue
		}
		if err != nil {
			t.Fatalf("caso %d: %v", i, err)
		}
		if res.IntegraAutomaticamente {
			t.Fatalf("caso %d: integra_automaticamente = true; tem de ser SEMPRE false", i)
		}
		if !res.ExigeAprovacaoHumana {
			t.Fatalf("caso %d: exige_aprovacao_humana = false; ha' decisao a tomar", i)
		}
	}
}

// O ranking respeita a direccao de cada criterio.
//
//	*Um criterio sem direccao e' um numero; nao e' um criterio.*
func TestCompararRespeitaDireccaoDosCriterios(t *testing.T) {
	// `regressoes` e' menor-melhor: menos regressoes ganha.
	res, err := CompararSolucoes([]Candidato{
		{
			ID: "muitas-regressoes", TestesPassaram: 5,
			Medicoes: []MedicaoDeCandidato{medido(CriterioRegressao, 9)},
		},
		{
			ID: "poucas-regressoes", TestesPassaram: 5,
			Medicoes: []MedicaoDeCandidato{medido(CriterioRegressao, 1)},
		},
	})
	if err != nil {
		t.Fatalf("comparar: %v", err)
	}
	if res.Vencedor != "poucas-regressoes" {
		t.Fatalf("vencedor = %q; esperado poucas-regressoes (menos e' melhor)", res.Vencedor)
	}

	// `evidencias` e' maior-melhor: mais evidencias ganha.
	res2, err := CompararSolucoes([]Candidato{
		{
			ID: "menos-evidencias", TestesPassaram: 5,
			Medicoes: []MedicaoDeCandidato{medido(CriterioEvidencias, 1)},
		},
		{
			ID: "mais-evidencias", TestesPassaram: 5,
			Medicoes: []MedicaoDeCandidato{medido(CriterioEvidencias, 8)},
		},
	})
	if err != nil {
		t.Fatalf("comparar: %v", err)
	}
	if res2.Vencedor != "mais-evidencias" {
		t.Fatalf("vencedor = %q; esperado mais-evidencias (mais e' melhor)", res2.Vencedor)
	}
}

// Candidato sem identificador e' rejeitado, e nao rebenta a comparacao.
func TestCompararRejeitaCandidatoSemID(t *testing.T) {
	res, err := CompararSolucoes([]Candidato{
		{ID: "", TestesPassaram: 1},
		{ID: "valido", TestesPassaram: 1},
	})
	if err != nil {
		t.Fatalf("comparar: %v", err)
	}
	if res.Vencedor != "valido" {
		t.Fatalf("vencedor = %q; esperado valido", res.Vencedor)
	}
	if len(res.Rejeitados) != 1 {
		t.Fatalf("rejeitados = %d; esperado 1", len(res.Rejeitados))
	}
}

// Sem candidatos e' erro explicito, nao resultado vazio disfarcado.
func TestCompararSemCandidatosDaErro(t *testing.T) {
	if _, err := CompararSolucoes(nil); !errors.Is(err, ErrComparacaoSemCandidatos) {
		t.Fatalf("erro = %v; esperado ErrComparacaoSemCandidatos", err)
	}
}

// A explicacao acompanha o desfecho, para o utilizador nao ter de inferir.
func TestCompararSempreExplica(t *testing.T) {
	casos := [][]Candidato{
		{{ID: "so-um", TestesPassaram: 1}},
		{
			{ID: "a", TestesPassaram: 1, Medicoes: []MedicaoDeCandidato{medido(CriterioTempo, 1)}},
			{ID: "b", TestesPassaram: 1, Medicoes: []MedicaoDeCandidato{medido(CriterioTempo, 2)}},
		},
		{{ID: "a", TestesFalharam: 3}},
	}
	for i, candidatos := range casos {
		res, err := CompararSolucoes(candidatos)
		if err != nil {
			t.Fatalf("caso %d: %v", i, err)
		}
		if res.Explicacao == "" {
			t.Fatalf("caso %d: resultado sem explicacao", i)
		}
	}
}

// Nenhum candidato aceite: nao ha' recomendacao, mas ha' explicacao e aprovacao exigida.
func TestCompararTodosRejeitadosNaoRecomenda(t *testing.T) {
	res, err := CompararSolucoes([]Candidato{
		{ID: "a", TestesFalharam: 1},
		{ID: "b", TestesFalharam: 2},
	})
	if err != nil {
		t.Fatalf("comparar: %v", err)
	}
	if res.Vencedor != "" || len(res.Ranking) != 0 {
		t.Fatalf("vencedor=%q ranking=%v; esperado nenhum", res.Vencedor, res.Ranking)
	}
	if len(res.Rejeitados) != 2 {
		t.Fatalf("rejeitados = %d; esperado 2", len(res.Rejeitados))
	}
	if res.Explicacao == "" {
		t.Fatal("sem explicacao quando todos sao rejeitados")
	}
}
