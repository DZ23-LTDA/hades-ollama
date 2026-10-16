package agent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// P0-B / T2 — REPETIÇÃO ESTATÍSTICA: TESTES (E-345).

// E-367: `harnessAlternado` e `isTestFile` foram REMOVIDOS daqui. O lint os apontava como `unused` e a leitura
// confirmou: a agregação testada abaixo é PURA e monta `HarnessBenchmarkResult` direto, sem harness nenhum.
// *Código morto não é documentação — é peso que parece intenção.*
//
// E foram removidos DUAS vezes: a primeira tentativa por REGEX quebrou o arquivo (`expected declaration, found
// 'for'`); a segunda foi cirúrgica; e um `git merge --no-edit` posterior **desfez** a remoção sem avisar ✗.
// ***Merge que "dá certo" não é merge que preserva — o teste é o que estava lá antes.***

func TestAgregacaoDeRepeticaoContaCadaVeredictoNoLugarCerto(t *testing.T) {
	// Agregação é PURA: a partir de resultados conhecidos, as contagens têm de bater — sem depender de modelo.
	resultados := []HarnessBenchmarkResult{
		{Verdict: "GREEN", Attempts: 1, Model: "m"},
		{Verdict: "GREEN", Attempts: 2, Model: "m"},
		{Verdict: "HARNESS_TRAFICOU", Attempts: 2, Model: "m"},
		{Verdict: "TIMEOUT", Attempts: 2, Model: "m"},
		{Verdict: "RED", Attempts: 2, Model: "m"},
		{Verdict: "NAO_MEDIDO", GreenExit: -1, Model: "m"},
	}
	// O agregador recebe resultados prontos: usa a mesma função de agregação por contagem.
	resumo := AgregarRepeticaoT2(resultados, "m", "fixture-x")
	if resumo.Execucoes != 6 {
		t.Fatalf("execuções contadas: %d", resumo.Execucoes)
	}
	if resumo.Verde != 2 || resumo.Verde1aTentativa != 1 {
		t.Fatalf("verde/1ª tentativa: %+v", resumo)
	}
	if resumo.Trapacou != 1 || resumo.Timeout != 1 || resumo.Vermelho != 1 || resumo.NaoMedido != 1 {
		t.Fatalf("cada veredicto precisa cair no seu contador: %+v", resumo)
	}
	if resumo.Taxa != "2/6" {
		t.Fatalf("taxa é CONTAGEM, não probabilidade: %q", resumo.Taxa)
	}
	// ARMADILHA: o aviso precisa estar SEMPRE presente — sem ele, N pequeno vira "superioridade".
	if resumo.Aviso == "" {
		t.Fatal("o resumo precisa carregar o aviso de que N pequeno não permite conclusão de superioridade")
	}
	if len(resumo.Veredictos) != 6 {
		t.Fatalf("os veredictos precisam ficar registrados um a um: %v", resumo.Veredictos)
	}
}

// REPETIÇÃO REAL: roda N vezes por combinação com modelo local; PULA com motivo declarado se o Ollama não responder.
func TestT2RepeticaoEstatistica(t *testing.T) {
	// E-968: MEDICAO COM MODELO REAL e' OPT-IN.
	// *Este teste pede 30 min de contexto contra os **10 min** do pacote `go test`, e o
	// Ollama local responde — logo ele NAO pulava e o pacote estourava (medido: 905 s no CI).*
	// O `t.Skip` abaixo protege contra «sem Ollama»; este guarda protege contra «com Ollama»,
	// que e' o caso normal. *O orcamento nao se esconde: declara-se.*
	if strings.TrimSpace(os.Getenv("HADES_T2_MEDIR_MODELO")) == "" {
		t.Skip("medicao com modelo real = NAO_MEDIDO (opt-in): pede 30 min contra os 10 min do pacote. " +
			"Defina HADES_T2_MEDIR_MODELO=1 para a correr.")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancelar()
	if !OllamaDisponivel(ctx, "") {
		t.Skip("Ollama local não respondeu: repetição estatística = NAO_MEDIDO (nunca 'ok' por ausência)")
	}
	modelo := os.Getenv("HADES_T2_MODELS")
	if modelo == "" {
		modelo = "qwen2.5:7b"
	}
	repeticoes := 3
	if bruto := os.Getenv("HADES_T2_REPEATS_N"); bruto != "" {
		if valor, err := parsePositivo(bruto); err == nil {
			repeticoes = valor
		}
	}
	resultados, agregados := RepetirT2Matrix(ctx, []HarnessProposer{OllamaHarness{Modelo: modelo}}, AllT2Fixtures(), 2, repeticoes)
	if len(resultados) == 0 {
		t.Fatal("a repetição precisa produzir resultados")
	}
	for _, item := range agregados {
		t.Logf("T2 repetição: modelo=%-14s fixture=%-24s execuções=%d verde=%d 1a=%d trapaceou=%d vermelho=%d timeout=%d nao_medido=%d dur_media=%.1fs taxa=%s",
			item.Modelo, item.Fixture, item.Execucoes, item.Verde, item.Verde1aTentativa, item.Trapacou,
			item.Vermelho, item.Timeout, item.NaoMedido, item.DuracaoMedia, item.Taxa)
	}
	// O teste NÃO julga o modelo: a repetição é MEDIÇÃO. Ele falha apenas se a medição não cobrir as combinações.
	if len(agregados) != len(AllT2Fixtures()) {
		t.Fatalf("a repetição precisa cobrir os %d fixtures: %d", len(AllT2Fixtures()), len(agregados))
	}
	EscreverArtefatoDaRepeticaoT2(t, agregados, 2)
}

func parsePositivo(bruto string) (int, error) {
	valor := 0
	for _, ch := range bruto {
		if ch < '0' || ch > '9' {
			return 0, os.ErrInvalid
		}
		valor = valor*10 + int(ch-'0')
	}
	if valor <= 0 {
		return 0, os.ErrInvalid
	}
	return valor, nil
}
