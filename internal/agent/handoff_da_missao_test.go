package agent

import (
	"strings"
	"testing"
)

// Testes do handoff derivado da missão (§4 + §6, E-595).

func missaoDeTeste() Mission {
	return Mission{
		ID:             "m-1",
		Version:        7,
		Objective:      "entregar a compactação com recuperação provada",
		ProjectID:      "proj-1",
		OrganizationID: "org-a",
		State:          MissionState("running"),
		Capabilities:   []string{"workspace:read", "tests:run"},
		Workspace:      "/tmp/ws",
		GitBranch:      "fix/checker-relogio",
		Plan: []Step{
			{ID: "s1", Kind: "code", Title: "ligar o funil", Risk: RiskClass("baixo")},
			{ID: "s2", Kind: "deploy", Title: "publicar", Risk: RiskClass("alto"), RequiresApproval: true},
		},
		Approvals: []Approval{{}},
	}
}

func TestHandoffDaMissaoTranspoeOPlanoEOEstado(t *testing.T) {
	h, err := HandoffDaMissao(missaoDeTeste())
	if err != nil {
		t.Fatalf("recusou uma missão válida: %v", err)
	}
	estado := strings.Join(h.EstadoDoTrabalho, " | ")
	if !strings.Contains(estado, "running") || !strings.Contains(estado, "2 passo") {
		t.Fatalf("estado do trabalho incompleto: %q", estado)
	}
	pend := strings.Join(h.Pendentes, " | ")
	// OS DOIS PASSOS ESTÃO PENDENTES, com id, risco e título.
	for _, esperado := range []string{"s1 — ligar o funil", "s2 — publicar", "risco declarado: alto"} {
		if !strings.Contains(pend, esperado) {
			t.Fatalf("pendentes sem %q: %q", esperado, pend)
		}
	}
	// E O PASSO QUE ESPERA DECISÃO HUMANA VAI MARCADO — é o que impede a próxima IA de o
	// executar por não saber que ele esperava.
	if !strings.Contains(pend, "EXIGE APROVAÇÃO") {
		t.Fatalf("o passo que exige aprovação não está marcado: %q", pend)
	}
	// As aprovações registadas também aparecem: são trabalho parado, não trabalho feito.
	if !strings.Contains(pend, "aprovação(ões) registada(s)") {
		t.Fatalf("as aprovações registadas não foram declaradas: %q", pend)
	}
	if err := h.Validar(); err != nil {
		t.Fatalf("o handoff derivado não é válido: %v", err)
	}
}

// A PROPRIEDADE DE SEGURANÇA: só viaja a autorização CONCEDIDA, e nunca se inventa.
func TestHandoffDaMissaoTransportaSoACapacidadeConcedida(t *testing.T) {
	h, err := HandoffDaMissao(missaoDeTeste())
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Autorizacao.Capacidades) != 2 {
		t.Fatalf("capacidades = %v, esperado as 2 concedidas", h.Autorizacao.Capacidades)
	}
	if len(h.Autorizacao.AprovacoesPendentes) != 0 {
		t.Fatal("declarou aprovação pendente havendo capacidades concedidas")
	}

	// SEM CAPACIDADES CONCEDIDAS: sai vazio E com a falta declarada. Nada é herdado por
	// omissão — uma troca de agente não pode elevar privilégio por arrastamento.
	semCaps := missaoDeTeste()
	semCaps.Capabilities = nil
	h2, err := HandoffDaMissao(semCaps)
	if err != nil {
		t.Fatal(err)
	}
	if len(h2.Autorizacao.Capacidades) != 0 {
		t.Fatalf("inventou capacidades: %v", h2.Autorizacao.Capacidades)
	}
	if len(h2.Autorizacao.AprovacoesPendentes) == 0 {
		t.Fatal("não declarou que não há autorização concedida")
	}
	if !strings.Contains(strings.Join(h2.Autorizacao.AprovacoesPendentes, " "), "não tem capacidades concedidas") {
		t.Fatalf("motivo inesperado: %v", h2.Autorizacao.AprovacoesPendentes)
	}
}

func TestHandoffDaMissaoLevaRestricoesReais(t *testing.T) {
	m := missaoDeTeste()
	m.LastError = "sem CGO nesta máquina"
	m.WorkspaceIsolated = true
	h, err := HandoffDaMissao(m)
	if err != nil {
		t.Fatal(err)
	}
	todas := strings.Join(h.Restricoes, " | ")
	for _, esperado := range []string{"sem CGO nesta máquina", "workspace isolado", "fix/checker-relogio"} {
		if !strings.Contains(todas, esperado) {
			t.Fatalf("restrição %q ausente: %q", esperado, todas)
		}
	}
	// E a evidência aponta para o REGISTO (id + versão), não para texto arrastado.
	if len(h.Evidencias) == 0 || !strings.Contains(h.Evidencias[0], "m-1") || !strings.Contains(h.Evidencias[0], "versão 7") {
		t.Fatalf("evidência não referencia o registo: %v", h.Evidencias)
	}
}

// Missão recém-criada: plano vazio, sem nada feito. É o caso em que a próxima IA mais
// precisa de saber que não há nada feito — e o `Validar` (E-588) recusaria estado vazio.
func TestHandoffDaMissaoAceitaMissaoRecemCriada(t *testing.T) {
	m := missaoDeTeste()
	m.Plan = nil
	m.Approvals = nil
	h, err := HandoffDaMissao(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.EstadoDoTrabalho) == 0 {
		t.Fatal("estado do trabalho vazio: Validar recusaria")
	}
	if h.Pendentes == nil {
		t.Fatal("Pendentes nil: nil não é o mesmo que vazio (E-588)")
	}
	if err := h.Validar(); err != nil {
		t.Fatalf("missão recém-criada dada como inválida: %v", err)
	}
}

func TestHandoffDaMissaoRecusaSemOrigemOuObjetivo(t *testing.T) {
	m := missaoDeTeste()
	m.ID = "  "
	if _, err := HandoffDaMissao(m); err == nil {
		t.Fatal("aceitou missão sem id")
	}
	m = missaoDeTeste()
	m.Objective = ""
	if _, err := HandoffDaMissao(m); err == nil {
		t.Fatal("aceitou missão sem objetivo")
	}
}
