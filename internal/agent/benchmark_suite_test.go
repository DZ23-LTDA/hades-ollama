package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Estágio 21 — a suíte roda e MEDE. Este teste é o executor do benchmark: ele imprime o
// relatório e exige que todas as tarefas herméticas concluam com sucesso. Se uma capacidade
// regredir, uma tarefa deixa de concluir e o teste falha — é o que dá sentido a acompanhar
// números em vez de narrar desempenho.
//
// Quando `HADES_BENCHMARK_OUT` está definida, o relatório é gravado nesse caminho, para o
// índice de evidências apontar para um ARTEFATO medido em vez de uma descrição.
func TestBenchmarkSuiteMeasuresHermeticTasks(t *testing.T) {
	runtime := factoryRuntime(t)
	report := runtime.RunBenchmarkSuite(time.Now().UTC())

	if report.TotalTasks < 5 {
		t.Fatalf("a suíte precisa cobrir pelo menos 5 tarefas, tem %d", report.TotalTasks)
	}
	if report.ExternalComparison != "NAO_EXECUTADA" {
		t.Fatalf("a comparação externa NÃO pode ser declarada feita sem executar os concorrentes: %q", report.ExternalComparison)
	}
	if report.ExternalReason == "" {
		t.Fatal("a ausência de comparação externa precisa vir com o motivo declarado")
	}
	for _, task := range report.Tasks {
		t.Logf("%-28s concluida=%-5t passos=%2d duracao=%4dms aprovacoes=%d isolamento=%t recuperacao=%t | %s",
			task.Name, task.Completed, task.Steps, task.DurationMillis, task.HumanApprovals,
			task.IsolationProven, task.RecoveryProven, task.Detail)
	}
	t.Logf("VEREDITO=%s concluidas=%d/%d duracao_total=%dms", report.Verdict, report.CompletedTasks, report.TotalTasks, report.TotalDurationMillis)

	// TODAS as tarefas herméticas precisam concluir: se alguma não concluir, existe regressão
	// de capacidade, e o relatório precisa dizer qual.
	if report.CompletedTasks != report.TotalTasks {
		for _, task := range report.Tasks {
			if !task.Completed {
				t.Errorf("tarefa %q NÃO concluiu: %s", task.Name, task.Detail)
			}
		}
		t.Fatalf("tarefas concluídas=%d de %d", report.CompletedTasks, report.TotalTasks)
	}
	// Isolamento entre organizações é medido em pelo menos duas tarefas independentes.
	isolated := 0
	recovered := 0
	for _, task := range report.Tasks {
		if task.IsolationProven {
			isolated++
		}
		if task.RecoveryProven {
			recovered++
		}
	}
	if isolated == 0 {
		t.Fatal("a suíte precisa provar isolamento entre organizações em pelo menos uma tarefa")
	}
	if recovered == 0 {
		t.Fatal("a suíte precisa provar retomada/recuperação em pelo menos uma tarefa")
	}

	if path := os.Getenv("HADES_BENCHMARK_OUT"); path != "" {
		payload, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(payload, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Logf("relatório do benchmark gravado em %s", path)
	}
}
