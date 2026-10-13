package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// P0-B / T2 — REPETIÇÃO ESTATÍSTICA (E-345).
//
// **LIMITE DECLARADO EM E-344**: a matriz media **uma execução por combinação**. Uma execução não distingue capacidade
// de sorte. Aqui a mesma combinação roda **N vezes** e o resultado vira **taxa com contagem** — e nada além disso:
//
// ***Com N pequeno NÃO se calcula intervalo de confiança nem se afirma significância. O que este instrumento entrega é
// CONTAGEM: "resolveu 2 de 3", com os veredictos ao lado.*** Alegar "melhor modelo" a partir de N=3 seria exatamente o
// tipo de afirmação sem métrica que este projeto proíbe.

// T2RepetitionSummary agrega N execuções de uma mesma combinação (modelo × fixture).
type T2RepetitionSummary struct {
	Modelo           string   `json:"modelo"`
	Fixture          string   `json:"fixture"`
	Execucoes        int      `json:"execucoes"`
	Verde            int      `json:"verde"`
	Verde1aTentativa int      `json:"verde_primeira_tentativa"`
	Trapacou         int      `json:"trapacou"`
	Vermelho         int      `json:"vermelho"`
	Timeout          int      `json:"timeout"`
	NaoMedido        int      `json:"nao_medido"`
	Veredictos       []string `json:"veredictos"`
	DuracaoMedia     float64  `json:"duracao_media_s"`
	// Taxa é contagem, não probabilidade: "verde/execuções".
	Taxa string `json:"taxa"`
	// Aviso repete, em toda saída, o que N pequeno NÃO permite concluir.
	Aviso string `json:"aviso"`
}

// RepetirT2Matrix roda cada combinação (harness × fixture) N vezes e agrega por contagem.
func RepetirT2Matrix(ctx context.Context, harnesses []HarnessProposer, tasks []HarnessBenchmarkTask, tentativas, repeticoes int) ([]HarnessBenchmarkResult, []T2RepetitionSummary) {
	if repeticoes <= 0 {
		repeticoes = 1
	}
	resultados := make([]HarnessBenchmarkResult, 0, len(harnesses)*len(tasks)*repeticoes)
	agregados := make([]T2RepetitionSummary, 0, len(harnesses)*len(tasks))
	for _, harness := range harnesses {
		for _, task := range tasks {
			daCombinacao := make([]HarnessBenchmarkResult, 0, repeticoes)
			for repeticao := 1; repeticao <= repeticoes; repeticao++ {
				if ctx.Err() != nil {
					daCombinacao = append(daCombinacao, HarnessBenchmarkResult{Task: task.Name, Harness: harness.Name(),
						Model: harness.Model(), GreenExit: -1, Verdict: "NAO_MEDIDO", Reason: "contexto cancelado"})
					continue
				}
				resultado, err := RunHarnessBenchmark(ctx, harness, task, tentativas)
				if err != nil {
					resultado = HarnessBenchmarkResult{Task: task.Name, Harness: harness.Name(),
						Model: harness.Model(), GreenExit: -1, Verdict: "NAO_MEDIDO", Reason: err.Error()}
				}
				resultados = append(resultados, resultado)
				daCombinacao = append(daCombinacao, resultado)
			}
			// Agregação PURA: a contagem não depende de modelo nem de rede — só dos resultados observados.
			agregados = append(agregados, AgregarRepeticaoT2(daCombinacao, harness.Model(), task.Name))
		}
	}
	return resultados, agregados
}

// EscreverArtefatoDaRepeticaoT2 grava a repetição medida (contagem por combinação) em `docs/parity/HADES_T2_REPETICAO.md`.
func EscreverArtefatoDaRepeticaoT2(t interface {
	Logf(string, ...any)
}, agregados []T2RepetitionSummary, tentativas int) {
	raiz, achou := raizDoRepositorioParaArtefato()
	if !achou {
		t.Logf("artefato de repetição NÃO gravado: docs/parity não encontrado a partir do arquivo-fonte")
		return
	}
	modelos := map[string]bool{}
	for _, item := range agregados {
		if item.Modelo != "" {
			modelos[item.Modelo] = true
		}
	}
	nomes := make([]string, 0, len(modelos))
	for nome := range modelos {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)

	var b strings.Builder
	b.WriteString("# HADES — T2: REPETIÇÃO ESTATÍSTICA (MEDIDA)\n\n")
	b.WriteString("**Gerado por:** `TestT2RepeticaoEstatistica` · **Data:** " + time.Now().UTC().Format(time.RFC3339) + "\n\n")
	b.WriteString("Tentativas por execução: **" + fmt.Sprint(tentativas) + "**. Modelos: `" + strings.Join(nomes, "`, `") + "`.\n\n")
	b.WriteString("| modelo | fixture | execuções | verde | verde 1ª tent. | trapaceou | vermelho | timeout | não medido | duração média (s) | taxa |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, item := range agregados {
		b.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %d | %d | %d | %d | %d | %.1f | **%s** |\n",
			item.Modelo, item.Fixture, item.Execucoes, item.Verde, item.Verde1aTentativa, item.Trapacou,
			item.Vermelho, item.Timeout, item.NaoMedido, item.DuracaoMedia, item.Taxa))
	}
	b.WriteString("\n## O que este artefato **NÃO** permite afirmar\n\n")
	b.WriteString("- **N pequeno** ⇒ os números são **contagem**, não probabilidade; **não** há intervalo de confiança.\n")
	b.WriteString("- **Sem ranking de modelos**: com poucas execuções, diferença de taxa **não** é superioridade — *alegar ")
	b.WriteString("superioridade sem métrica é o que o projeto proíbe*.\n")
	b.WriteString("- `HARNESS_TRAFICOU` significa que o modelo **alterou o arquivo de teste** (o anti-trapaça registrou).\n")
	b.WriteString("- `não medido` = execução que **não aconteceu** (contexto cancelado/limite): ausência **não** é aprovação.\n")
	b.WriteString("- Harness **mínimo** sobre Ollama local; Claude Code/Codex/Manus ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` ⇒ `NAO_MEDIDO`.\n")
	if err := os.WriteFile(filepath.Join(raiz, "HADES_T2_REPETICAO.md"), []byte(b.String()), 0o600); err != nil {
		t.Logf("artefato de repetição não gravado: %v", err)
	}
}

// AgregarRepeticaoT2 conta os veredictos de N execuções da MESMA combinação (modelo × fixture).
//
// Função PURA, de propósito: *contagem não precisa de modelo nem de rede para ser testada* — e um agregador que só roda
// dentro do laço caro nunca seria verificado.
func AgregarRepeticaoT2(resultados []HarnessBenchmarkResult, modelo, fixture string) T2RepetitionSummary {
	resumo := T2RepetitionSummary{Modelo: modelo, Fixture: fixture}
	somaDuracao := 0.0
	for _, resultado := range resultados {
		resumo.Execucoes++
		somaDuracao += resultado.DurationSeconds
		resumo.Veredictos = append(resumo.Veredictos, resultado.Verdict)
		switch resultado.Verdict {
		case "GREEN":
			resumo.Verde++
			if resultado.Attempts == 1 {
				resumo.Verde1aTentativa++
			}
		case "HARNESS_TRAFICOU":
			resumo.Trapacou++
		case "TIMEOUT":
			resumo.Timeout++
		case "NAO_MEDIDO":
			resumo.NaoMedido++
		default:
			resumo.Vermelho++
		}
	}
	if resumo.Execucoes > 0 {
		resumo.DuracaoMedia = somaDuracao / float64(resumo.Execucoes)
		resumo.Taxa = fmt.Sprintf("%d/%d", resumo.Verde, resumo.Execucoes)
	} else {
		resumo.Taxa = "NAO_MEDIDO"
	}
	resumo.Aviso = "N pequeno: contagem, não probabilidade. Sem intervalo de confiança e sem afirmação de " +
		"superioridade entre modelos a partir deste N."
	return resumo
}
