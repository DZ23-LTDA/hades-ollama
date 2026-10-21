package server

// FASE 3 (E-1240): testes da rota POST /solutions/compare.
//
// O que estes testes travam:
//
//  1. a rota responde e devolve o ranking que a funcao Go calculou;
//  2. `integra_automaticamente` e' SEMPRE falso tambem no JSON;
//  3. `somente_leitura` e' sempre verdadeiro -- o POST nao muta nada;
//  4. teste vermelho e' rejeitado PELA ROTA, nao so' pela funcao;
//  5. pedido vazio, candidatos a mais e JSON invalido sao recusados com o codigo certo;
//  6. os avisos derivam do RESULTADO, nao de constantes -- um empate avisa, uma vitoria limpa nao;
//  7. a comparacao e' DETERMINISTICA: o mesmo pedido da' a mesma resposta.
//
//	Uma rota que devolve o mesmo que a funcao mas sem provas de que e' a MESMA resposta nao prova o
//	elo: prova que o handler compila.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// compararPelaRota monta o router de PRODUCAO e faz o POST, devolvendo codigo e corpo ja' decodificado.
func compararPelaRota(t *testing.T, corpo any) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// `register` recebe o ENGINE, nao um RouterGroup: e' o mesmo ponto de entrada do produto, para
	// o teste nao provar um router que a producao nao usa.
	(&agentAPI{}).register(router)

	var bruto []byte
	switch v := corpo.(type) {
	case string:
		bruto = []byte(v)
	default:
		var err error
		bruto, err = json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal do pedido: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agent/v1/solutions/compare", bytes.NewReader(bruto))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resposta map[string]any
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &resposta); err != nil {
			t.Fatalf("resposta nao e' JSON: %v\ncorpo: %s", err, w.Body.String())
		}
	}
	return w.Code, resposta
}

// candidatoDeTeste monta um candidato com uma medicao de tempo.
func candidatoDeTeste(id string, falhas int, tempo float64) agent.Candidato {
	return agent.Candidato{
		ID:             id,
		TestesPassaram: 5,
		TestesFalharam: falhas,
		Medicoes: []agent.MedicaoDeCandidato{
			{Criterio: agent.CriterioTempo, Valor: tempo, Medido: true, Fonte: "teste"},
		},
	}
}

// A rota devolve o ranking e NUNCA integra.
func TestComparacaoDeSolucoesDevolveRankingSemIntegrar(t *testing.T) {
	codigo, corpo := compararPelaRota(t, map[string]any{
		"missao": "missao-compare",
		"candidatos": []agent.Candidato{
			candidatoDeTeste("lenta", 0, 100),
			candidatoDeTeste("rapida", 0, 1),
		},
	})
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200. corpo=%v", codigo, corpo)
	}
	if corpo["vencedor"] != "rapida" {
		t.Fatalf("vencedor = %v; esperado rapida", corpo["vencedor"])
	}
	// AS DUAS GARANTIAS DO CONTRATO, verificadas no JSON e nao no struct.
	if corpo["integra_automaticamente"] != false {
		t.Fatalf("integra_automaticamente = %v; tem de ser false", corpo["integra_automaticamente"])
	}
	if corpo["somente_leitura"] != true {
		t.Fatalf("somente_leitura = %v; tem de ser true", corpo["somente_leitura"])
	}
	if corpo["exige_aprovacao_humana"] != true {
		t.Fatalf("exige_aprovacao_humana = %v; tem de ser true", corpo["exige_aprovacao_humana"])
	}
	if corpo["missao"] != "missao-compare" {
		t.Fatalf("missao = %v; esperado eco", corpo["missao"])
	}
}

// Teste vermelho e' rejeitado pela ROTA, nao so' pela funcao.
func TestComparacaoDeSolucoesRejeitaTesteVermelho(t *testing.T) {
	codigo, corpo := compararPelaRota(t, map[string]any{
		"candidatos": []agent.Candidato{
			candidatoDeTeste("quebrada", 2, 1),
			candidatoDeTeste("correcta", 0, 50),
		},
	})
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200", codigo)
	}
	if corpo["vencedor"] != "correcta" {
		t.Fatalf("vencedor = %v; esperado correcta (a quebrada falhou 2 testes)", corpo["vencedor"])
	}
	rejeitados, ok := corpo["rejeitados"].([]any)
	if !ok || len(rejeitados) != 1 {
		t.Fatalf("rejeitados = %v; esperado 1", corpo["rejeitados"])
	}
}

// Empate: a rota NAO escolhe e AVISA.
func TestComparacaoDeSolucoesEmpateAvisaENaoEscolhe(t *testing.T) {
	iguais := func(id string) agent.Candidato { return candidatoDeTeste(id, 0, 30) }
	codigo, corpo := compararPelaRota(t, map[string]any{
		"candidatos": []agent.Candidato{iguais("a"), iguais("b")},
	})
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200", codigo)
	}
	if corpo["empate"] != true {
		t.Fatalf("empate = %v; esperado true", corpo["empate"])
	}
	if corpo["vencedor"] != nil && corpo["vencedor"] != "" {
		t.Fatalf("vencedor = %v; esperado vazio num empate", corpo["vencedor"])
	}
	// O aviso tem de existir e falar de empate: deriva do RESULTADO.
	avisos, _ := corpo["avisos"].([]any)
	achou := false
	for _, a := range avisos {
		if s, ok := a.(string); ok && bytes.Contains([]byte(s), []byte("empate")) {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("nenhum aviso fala de empate; avisos=%v", avisos)
	}
}

// Vitoria limpa NAO ganha aviso de empate: os avisos derivam do resultado.
func TestComparacaoDeSolucoesVitoriaLimpaNaoAvisaEmpate(t *testing.T) {
	_, corpo := compararPelaRota(t, map[string]any{
		"candidatos": []agent.Candidato{
			candidatoDeTeste("a", 0, 1),
			candidatoDeTeste("b", 0, 2),
		},
	})
	avisos, _ := corpo["avisos"].([]any)
	for _, a := range avisos {
		if s, ok := a.(string); ok && bytes.Contains([]byte(s), []byte("empate")) {
			t.Fatalf("aviso de empate numa vitoria limpa: %q", s)
		}
	}
}

// Pedido sem candidatos e' 400, nao 200 com resultado vazio.
func TestComparacaoDeSolucoesSemCandidatosDa400(t *testing.T) {
	codigo, corpo := compararPelaRota(t, map[string]any{"candidatos": []agent.Candidato{}})
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400. corpo=%v", codigo, corpo)
	}
	if corpo["erro"] == nil || corpo["erro"] == "" {
		t.Fatalf("400 sem mensagem de erro: %v", corpo)
	}
}

// Campo ausente e' 400.
func TestComparacaoDeSolucoesSemCampoCandidatosDa400(t *testing.T) {
	codigo, _ := compararPelaRota(t, map[string]any{"missao": "x"})
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400", codigo)
	}
}

// JSON invalido e' 400 com detalhe, nao panico.
func TestComparacaoDeSolucoesJSONInvalidoDa400(t *testing.T) {
	codigo, corpo := compararPelaRota(t, "{isso nao e' json")
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400", codigo)
	}
	if corpo["detalhe"] == nil || corpo["detalhe"] == "" {
		t.Fatalf("400 sem detalhe: %v", corpo)
	}
}

// Demasiados candidatos e' 413, nao 200 nem estouro de memoria.
//
//	*Uma rota que aceita candidatos sem limite e' um vector de esgotamento de memoria com nome de
//	funcionalidade.*
func TestComparacaoDeSolucoesDemasiadosCandidatosDa413(t *testing.T) {
	muitos := make([]agent.Candidato, 0, maxCandidatosNaComparacao+1)
	for i := 0; i <= maxCandidatosNaComparacao; i++ {
		muitos = append(muitos, candidatoDeTeste(
			"c"+string(rune('a'+i%26))+string(rune('a'+i/26)), 0, float64(i)))
	}
	codigo, corpo := compararPelaRota(t, map[string]any{"candidatos": muitos})
	if codigo != http.StatusRequestEntityTooLarge {
		t.Fatalf("codigo = %d; esperado 413. corpo=%v", codigo, corpo)
	}
	if corpo["maximo"] == nil {
		t.Fatalf("413 sem informar o maximo: %v", corpo)
	}
}

// A comparacao e' DETERMINISTICA: o mesmo pedido da' a mesma resposta.
func TestComparacaoDeSolucoesEDeterministica(t *testing.T) {
	pedido := map[string]any{
		"candidatos": []agent.Candidato{
			candidatoDeTeste("a", 0, 10),
			candidatoDeTeste("b", 0, 20),
			candidatoDeTeste("c", 3, 1),
		},
	}
	_, primeiro := compararPelaRota(t, pedido)
	for i := range 3 {
		_, seguinte := compararPelaRota(t, pedido)
		a, _ := json.Marshal(primeiro)
		b, _ := json.Marshal(seguinte)
		if !bytes.Equal(a, b) {
			t.Fatalf("execucao %d divergiu:\n%s\n%s", i, a, b)
		}
	}
}

// Nenhum candidato aceite: 200, sem vencedor, com aviso.
func TestComparacaoDeSolucoesTodosRejeitados(t *testing.T) {
	codigo, corpo := compararPelaRota(t, map[string]any{
		"candidatos": []agent.Candidato{
			candidatoDeTeste("a", 1, 1),
			candidatoDeTeste("b", 2, 2),
		},
	})
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200", codigo)
	}
	if corpo["vencedor"] != nil && corpo["vencedor"] != "" {
		t.Fatalf("vencedor = %v; esperado vazio", corpo["vencedor"])
	}
	avisos, _ := corpo["avisos"].([]any)
	if len(avisos) == 0 {
		t.Fatal("todos rejeitados devia avisar")
	}
	if corpo["integra_automaticamente"] != false {
		t.Fatalf("integra_automaticamente = %v; tem de ser false", corpo["integra_automaticamente"])
	}
}
