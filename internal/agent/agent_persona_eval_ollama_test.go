// O harness de avaliacao contra um MODELO REAL.
//
// # PORQUE ESTE FICHEIRO EXISTE
//
// O E-1369 provou o harness com 16 testes e um `InferFunc` injectado. Isso prova que a LOGICA de avaliacao
// funciona; nao prova que ela corre contra o motor do produto. A FASE 30.7 exige:
//
//	compatibilidade com pelo menos um modelo local principal do HADES.
//
// Ate aqui o `/agents/evaluation` respondia `inference_available: false`. Honesto, e incompleto.
//
//	*Uma comparacao que so o programador consegue invocar nao e uma funcionalidade: e uma funcao.*
//
// # O QUE ESTE TESTE DISCRIMINA
//
// Nao discrimina se o `qwen2.5:7b` e' bom. Discrimina que:
//  1. o `InferFunc` e' CHAMADO uma vez por caso -- nao ha' caminho que "avalie" sem perguntar;
//  2. o relatorio tem um resultado por caso, com o texto que o modelo devolveu;
//  3. o estado final NUNCA e' `PRODUCTION_APPROVED` -- um modelo a responder bem nao aprova ninguem;
//  4. erro do modelo vira `EVALUATED_FAILED`, nao um passe silencioso.
//
// AUSENCIA DO MODELO NAO E' UM SKIP SILENCIOSO
//
// Se o endpoint nao responder ou o modelo nao existir, o teste PARA com `t.Skipf` e o motivo ESCRITO. O gate
// continua verde porque a dependencia e' externa e declarada -- mas o motivo vai no output, nao num buraco.
package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	ollamaBase  = "http://127.0.0.1:11434"
	ollamaModel = "qwen2.5:7b"
)

// ollamaResponde confirma que o endpoint serve e que o modelo existe, sem inventar disponibilidade.
func ollamaResponde(t *testing.T) {
	t.Helper()
	cli := &http.Client{Timeout: 5 * time.Second}
	resp, err := cli.Get(ollamaBase + "/api/tags")
	if err != nil {
		t.Skipf("DEPENDENCIA EXTERNA: o Ollama nao responde em %s (%v). "+
			"Este teste e' o unico que prova o harness contra o motor real; sem o motor ele nao pode correr. "+
			"Desbloqueio: `ollama serve` com o modelo %s disponivel.", ollamaBase, err, ollamaModel)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("DEPENDENCIA EXTERNA: /api/tags devolveu %d em %s", resp.StatusCode, ollamaBase)
	}
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		t.Skipf("DEPENDENCIA EXTERNA: /api/tags ilegivel: %v", err)
	}
	for _, m := range tags.Models {
		if m.Name == ollamaModel {
			return
		}
	}
	nomes := make([]string, 0, len(tags.Models))
	for _, m := range tags.Models {
		nomes = append(nomes, m.Name)
	}
	t.Skipf("DEPENDENCIA EXTERNA: o modelo %q nao esta' instalado. Disponiveis: %s. "+
		"Desbloqueio: `ollama pull %s`.", ollamaModel, strings.Join(nomes, ", "), ollamaModel)
}

// ollamaInfer implementa InferFunc contra o motor real.
//
// `POST /api/generate` com `stream:false`: uma resposta completa por prompt. Nada de streaming aqui -- a
// avaliacao compara o TEXTO final, nao o caminho ate' ele.
func ollamaInfer(t *testing.T, modelo string, chamadas *int64) InferFunc {
	cli := &http.Client{Timeout: 120 * time.Second}
	return func(prompt string) (string, error) {
		atomic.AddInt64(chamadas, 1)
		corpo, err := json.Marshal(map[string]any{
			"model":  modelo,
			"prompt": prompt,
			"stream": false,
			// Temperatura baixa: a avaliacao compara respostas, e uma resposta que muda a cada corrida nao
			// compara nada. *Um teste que as vezes passa nao e' um teste: e' uma moeda ao ar que se pode citar.*
			"options": map[string]any{"temperature": 0.0},
		})
		if err != nil {
			return "", err
		}
		resp, err := cli.Post(ollamaBase+"/api/generate", "application/json", bytes.NewReader(corpo))
		if err != nil {
			return "", fmt.Errorf("ollama generate: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("ollama generate: HTTP %d", resp.StatusCode)
		}
		var saida struct {
			Response string `json:"response"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&saida); err != nil {
			return "", fmt.Errorf("ollama generate: resposta ilegivel: %w", err)
		}
		return saida.Response, nil
	}
}

// O TESTE. Corre a suite REAL de engenharia contra o modelo REAL.
func TestEvaluateContraOllamaReal(t *testing.T) {
	if os.Getenv("HADES_SKIP_OLLAMA_EVAL") == "1" {
		t.Skip("HADES_SKIP_OLLAMA_EVAL=1 declarado pelo operador")
	}
	ollamaResponde(t)

	suite, ok := EvalSuiteFor("engenharia")
	if !ok {
		t.Fatal("a divisao engenharia devia ter suite")
	}
	if err := ValidateSuite(suite); err != nil {
		t.Fatalf("suite invalida: %v", err)
	}
	persona, ok := CatalogLookup("hades-executor")
	if !ok {
		t.Fatal("hades-executor devia existir no catalogo")
	}

	var chamadas int64
	infer := ollamaInfer(t, ollamaModel, &chamadas)

	inicio := time.Now()
	rel := Evaluate(persona, suite, infer)
	decorrido := time.Since(inicio)

	// 1. O infer foi chamado uma vez por caso.
	if int(chamadas) != len(suite.Cases) {
		t.Fatalf("o harness chamou o modelo %d vezes para %d casos", chamadas, len(suite.Cases))
	}

	// 2. Um resultado por caso, com texto.
	if len(rel.Cases) != len(suite.Cases) {
		t.Fatalf("relatorio com %d resultados para %d casos", len(rel.Cases), len(suite.Cases))
	}
	for _, c := range rel.Cases {
		if strings.TrimSpace(c.Output) == "" {
			t.Fatalf("caso %q voltou vazio do modelo", c.ID)
		}
	}

	// 3. NUNCA aprovacao de producao.
	// *A ausencia da capacidade e a garantia mais forte que existe.* -- e aqui a capacidade de aprovar.
	if rel.State == EvalProductionApproved {
		t.Fatal("o harness aprovou para producao por o modelo ter respondido bem")
	}
	// 4. O estado tem de ser um dos que o harness pode emitir.
	validos := map[EvalState]bool{
		EvalPass: true, EvalLimited: true, EvalFailed: true,
	}
	if !validos[rel.State] {
		t.Fatalf("estado inesperado: %q", rel.State)
	}

	t.Logf("RESULTADO REAL: estado=%s casos=%d modelo=%s versao_suite=%s duracao=%s",
		rel.State, len(rel.Cases), ollamaModel, rel.SuiteVersion, decorrido.Round(time.Millisecond))
	for _, c := range rel.Cases {
		saida := strings.ReplaceAll(strings.TrimSpace(c.Output), "\n", " ")
		if len(saida) > 100 {
			saida = saida[:100] + "..."
		}
		t.Logf("  [%s] id=%s passou=%v saida=%q", c.Kind, c.ID, c.Passed, saida)
	}
}

// Erro do modelo tem de virar FALHA, nao passe silencioso.
//
// Discriminante: se `Evaluate` tratasse um erro como "sem evidencia, logo passa", este teste falharia. E' o
// mesmo defeito que o `res.Passed` da ronda 172 tinha -- *o zero value de um bool e uma afirmacao silenciosa.*
func TestEvaluateErroDoModeloViraFalha(t *testing.T) {
	suite, ok := EvalSuiteFor("engenharia")
	if !ok {
		t.Fatal("sem suite")
	}
	persona, ok := CatalogLookup("hades-executor")
	if !ok {
		t.Fatal("sem persona")
	}
	var chamadas int64
	infer := func(string) (string, error) {
		atomic.AddInt64(&chamadas, 1)
		return "", fmt.Errorf("modelo indisponivel de proposito")
	}
	rel := Evaluate(persona, suite, infer)
	if rel.State != EvalFailed {
		t.Fatalf("erro do modelo devia dar EVALUATED_FAILED, deu %q", rel.State)
	}
	if int(chamadas) != len(suite.Cases) {
		t.Fatalf("parou no primeiro erro: %d de %d casos", chamadas, len(suite.Cases))
	}
	for _, c := range rel.Cases {
		if c.Passed {
			t.Fatalf("caso %q passou com o modelo a falhar", c.ID)
		}
	}
}
