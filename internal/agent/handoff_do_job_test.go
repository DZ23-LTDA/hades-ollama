package agent

import (
	"strings"
	"testing"
	"time"
)

// Testes da integração handoff ← orquestração (§4 + §6, E-593).

func momento() *time.Time {
	t := time.Date(2026, 10, 14, 20, 0, 0, 0, time.UTC)
	return &t
}

func jobDeTeste() OrchestrationJob {
	return OrchestrationJob{
		ID:             "job-1",
		OrganizationID: "org-a",
		ProjectID:      "proj-1",
		Objective:      "entregar a compactação com recuperação provada",
		Budget:         AgentBudget{MaxAgents: 3, MaxSeconds: 900, MaxRetries: 2, MaxOutputBytes: 4096},
		Tasks: []AgentTask{
			{ID: "t1", Role: "backend", Objective: "ligar o funil", State: AgentTaskSucceeded, Attempts: 1, FinishedAt: momento()},
			{ID: "t2", Role: "qa", Objective: "medir o race", State: AgentTaskFailed, Attempts: 3, Error: "sem CGO", FinishedAt: momento()},
			{ID: "t3", Role: "reviewer", Objective: "rever o PR", State: AgentTaskRunning, Attempts: 1},
		},
	}
}

func TestHandoffDoJobSeparaFeitoFalhadoEPendente(t *testing.T) {
	h, err := HandoffDoJob(jobDeTeste())
	if err != nil {
		t.Fatalf("recusou uma missão válida: %v", err)
	}
	if h.Objetivo == "" || h.ProjetoID != "proj-1" {
		t.Fatalf("não transportou objetivo/projeto: %+v", h)
	}
	juntas := strings.Join(h.EstadoDoTrabalho, " | ")
	if !strings.Contains(juntas, "backend") || !strings.Contains(juntas, "ligar o funil") {
		t.Fatalf("a tarefa concluída não está no estado do trabalho: %q", juntas)
	}
	// A FALHADA VAI PARA PENDENTES COM O MOTIVO — não é «feito» nem «por fazer às cegas».
	pend := strings.Join(h.Pendentes, " | ")
	if !strings.Contains(pend, "FALHOU: sem CGO") {
		t.Fatalf("a falha não ficou declarada com o motivo: %q", pend)
	}
	if !strings.Contains(pend, "rever o PR") {
		t.Fatalf("a tarefa não terminada não está pendente: %q", pend)
	}
	// E A CONCLUÍDA NÃO APARECE COMO PENDENTE.
	if strings.Contains(pend, "ligar o funil") {
		t.Fatalf("trabalho já feito reapareceu como pendente: %q", pend)
	}
	if h.Pendentes == nil {
		t.Fatal("Pendentes veio nil: nil não é o mesmo que vazio (E-588)")
	}
	if err := h.Validar(); err != nil {
		t.Fatalf("o handoff produzido não é válido: %v", err)
	}
}

// O CASO QUE EU QUASE DEIXEI PASSAR: missão SEM NADA CONCLUÍDO. É o caso mais urgente —
// uma missão mal começada a ser entregue — e o `Validar` do E-588 recusá-la-ia por estado
// do trabalho vazio. "Nada concluído" é informação, não ausência.
func TestHandoffDoJobAceitaMissaoSemNadaConcluido(t *testing.T) {
	job := OrchestrationJob{
		ID:        "job-2",
		Objective: "começar",
		Budget:    AgentBudget{MaxAgents: 1, MaxSeconds: 60, MaxRetries: 0, MaxOutputBytes: 1024},
		Tasks:     []AgentTask{{ID: "t1", Role: "backend", Objective: "primeiro passo"}},
	}
	h, err := HandoffDoJob(job)
	if err != nil {
		t.Fatalf("recusou: %v", err)
	}
	if len(h.EstadoDoTrabalho) == 0 || !strings.Contains(strings.Join(h.EstadoDoTrabalho, " "), "nada concluído ainda") {
		t.Fatalf("não declarou que nada foi concluído: %+v", h.EstadoDoTrabalho)
	}
	if err := h.Validar(); err != nil {
		t.Fatalf("uma missão retomável foi dada como inválida: %v", err)
	}
}

func TestHandoffDoJobNaoHerdaAutorizacao(t *testing.T) {
	h, err := HandoffDoJob(jobDeTeste())
	if err != nil {
		t.Fatal(err)
	}
	// NENHUMA capacidade herdada: uma troca de agente não pode elevar privilégio por arrastamento.
	if len(h.Autorizacao.Capacidades) != 0 {
		t.Fatalf("herdou capacidades: %v", h.Autorizacao.Capacidades)
	}
	if len(h.Autorizacao.AprovacoesPendentes) == 0 {
		t.Fatal("não declarou que a autorização não está registada")
	}
	if !strings.Contains(strings.Join(h.Autorizacao.AprovacoesPendentes, " "), "não regista capacidades") {
		t.Fatalf("motivo inesperado: %v", h.Autorizacao.AprovacoesPendentes)
	}
}

func TestHandoffDoJobLevaOBudgetComoRestricao(t *testing.T) {
	h, err := HandoffDoJob(jobDeTeste())
	if err != nil {
		t.Fatal(err)
	}
	todas := strings.Join(h.Restricoes, " | ")
	for _, esperado := range []string{"3 agentes", "900 segundos", "2 novas tentativas", "4096 bytes"} {
		if !strings.Contains(todas, esperado) {
			t.Fatalf("o orçamento não virou restrição (%q ausente): %q", esperado, todas)
		}
	}
	// E o texto sai como DADO marcado, não como instrução (§7-E).
	texto, _ := h.ParaContexto(TetoDeDigestPadrao)
	if !strings.HasPrefix(texto, MarcadorDeHandoff) {
		t.Fatal("o bloco não está marcado como dado")
	}
}

func TestHandoffDoJobRecusaMissaoSemOrigemOuObjetivo(t *testing.T) {
	// Sem id não há rastreabilidade da origem.
	job := jobDeTeste()
	job.ID = "  "
	if _, err := HandoffDoJob(job); err == nil {
		t.Fatal("aceitou missão sem id")
	}
	// Sem objetivo não há handoff possível.
	job = jobDeTeste()
	job.Objective = ""
	if _, err := HandoffDoJob(job); err == nil {
		t.Fatal("aceitou missão sem objetivo")
	}
}
