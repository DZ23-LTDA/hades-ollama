package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// ATO HUMANO na trilha de auditoria (estágio 35, §11.1 da missão).
//
// A trilha de ferramentas cobre o que o AGENTE executou. Aprovar, negar e mudar quem aprova são atos HUMANOS
// de risco, e a diretriz pede trilha de auditoria exatamente para esses atos. O que estes testes travam:
//
//  1. a DECISÃO de aprovar entra na trilha, com o ator;
//  2. a NEGATIVA entra como BLOQUEADA — negar é decisão tomada, não ausência de decisão;
//  3. a mudança de QUEM aprova entra na trilha;
//  4. sem trilha configurada, o erro é DECLARADO (não silencioso);
//  5. a trilha de política é isolada por organização.
func TestServiceAgentApprovalDecisionIsAudited(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	runtime.SetAuditStore(NewServiceAgentAuditStore(serviceAgentAuditRoot(t.TempDir())))
	store := runtime.ApprovalStore()

	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano avançado", 79900), now); err != nil {
		t.Fatal(err)
	}
	pending, err := store.ListApprovals("org-a", ServiceAgentApprovalPending, now)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pedido pendente ausente: %+v (%v)", pending, err)
	}
	decided, err := store.Decide("org-a", pending[0].ID, true, "gerente-ana", "cliente pediu por escrito", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.RecordApprovalDecision(decided, "gerente-ana", "cliente pediu por escrito", now); err != nil {
		t.Fatalf("o ato humano precisa entrar na trilha: %v", err)
	}

	trail, err := runtime.AuditTrail("org-a", 0)
	if err != nil {
		t.Fatal(err)
	}
	var encontrado *ServiceAgentAuditEntry
	for index := range trail {
		if trail[index].ToolName == "approval.decide" {
			encontrado = &trail[index]
		}
	}
	if encontrado == nil {
		t.Fatalf("a decisão precisa constar na trilha: %+v", trail)
	}
	if encontrado.Status != ServiceAgentToolExecuted {
		t.Fatalf("aprovação precisa constar como executada: %+v", encontrado)
	}
	if encontrado.AgentID != "policy" {
		t.Fatalf("o ato é de POLÍTICA, não de um agente de atendimento: %+v", encontrado)
	}
	if !strings.Contains(encontrado.Reason, "gerente-ana") {
		t.Fatalf("quem decidiu precisa constar: %q", encontrado.Reason)
	}
	if !strings.Contains(encontrado.Summary, pending[0].ID) {
		t.Fatalf("o resumo precisa identificar a aprovação: %q", encontrado.Summary)
	}
	// A trilha de política é verificável como o resto: a cadeia de hashes cobre estas entradas também.
	// São DUAS entradas: o turno registrou a tentativa de escrita (`approval_required`) e o ato humano entrou
	// em seguida — as duas convivem na mesma cadeia, que é o que se quer de uma auditoria.
	verification, err := runtime.audit.VerifyTrail("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Valid || verification.Entries != 2 {
		t.Fatalf("a entrada de política precisa entrar na cadeia: %+v", verification)
	}
}

func TestServiceAgentDeniedDecisionIsAuditedAsBlocked(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	runtime.SetAuditStore(NewServiceAgentAuditStore(serviceAgentAuditRoot(t.TempDir())))
	store := runtime.ApprovalStore()

	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano avançado", 79900), now); err != nil {
		t.Fatal(err)
	}
	pending, err := store.ListApprovals("org-a", ServiceAgentApprovalPending, now)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pedido pendente ausente: %+v (%v)", pending, err)
	}
	denied, err := store.Decide("org-a", pending[0].ID, false, "gerente-ana", "valor fora da política", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.RecordApprovalDecision(denied, "gerente-ana", "valor fora da política", now); err != nil {
		t.Fatal(err)
	}
	trail, err := runtime.AuditTrail("org-a", 0)
	if err != nil {
		t.Fatal(err)
	}
	// O turno também deixou a tentativa na trilha; o que importa aqui é a entrada do ATO humano.
	var negativa *ServiceAgentAuditEntry
	for index := range trail {
		if trail[index].ToolName == "approval.decide" {
			negativa = &trail[index]
		}
	}
	if negativa == nil {
		t.Fatalf("a negativa precisa estar na trilha: %+v", trail)
	}
	// Negar é decisão TOMADA: `blocked` na trilha diz isso; `executed` diria que a escrita passou.
	if negativa.Status != ServiceAgentToolBlocked {
		t.Fatalf("negativa precisa constar como BLOQUEADA: %+v", negativa)
	}
	if !strings.Contains(negativa.Reason, "fora da política") {
		t.Fatalf("o motivo da recusa precisa constar: %q", negativa.Reason)
	}
}

func TestServiceAgentApproversChangeIsAudited(t *testing.T) {
	runtime, _, _, now := approvalFixture(t)
	runtime.SetAuditStore(NewServiceAgentAuditStore(serviceAgentAuditRoot(t.TempDir())))
	store := runtime.ApprovalStore()

	if err := store.SetApprovers("org-a", []string{"ana", "bruno"}); err != nil {
		t.Fatal(err)
	}
	if err := runtime.RecordApproversChange("org-a", "ana", []string{"ana", "bruno"}, now); err != nil {
		t.Fatalf("mudar quem aprova precisa entrar na trilha: %v", err)
	}
	trail, err := runtime.AuditTrail("org-a", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(trail) != 1 || trail[0].ToolName != "policy.approvers.set" {
		t.Fatalf("a mudança de política precisa constar: %+v", trail)
	}
	if !strings.Contains(trail[0].Summary, "2 aprovador") {
		t.Fatalf("o resumo precisa dizer o tamanho da lista: %q", trail[0].Summary)
	}
	if !strings.Contains(trail[0].Reason, "ana") {
		t.Fatalf("quem mudou precisa constar: %q", trail[0].Reason)
	}
	// Isolamento: outra organização não vê o ato de política.
	outra, err := runtime.AuditTrail("org-b", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(outra) != 0 {
		t.Fatalf("ato de política não pode vazar entre organizações: %+v", outra)
	}
}

func TestServiceAgentPolicyAuditDeclaresMissingTrail(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	// Sem `SetAuditStore`, o ato de política NÃO pode ser registrado — e isso é ERRO declarado.
	store := runtime.ApprovalStore()
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano avançado", 79900), now); err != nil {
		t.Fatal(err)
	}
	pending, err := store.ListApprovals("org-a", ServiceAgentApprovalPending, now)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pedido pendente ausente: %+v (%v)", pending, err)
	}
	decided, err := store.Decide("org-a", pending[0].ID, true, "ana", "", now)
	if err != nil {
		t.Fatal(err)
	}
	err = runtime.RecordApprovalDecision(decided, "ana", "", now)
	if !errors.Is(err, ErrServiceAgentAuditUnavailable) {
		t.Fatalf("sem trilha, o erro precisa ser DECLARADO: %v", err)
	}
	// A decisão em si continua registrada no store de aprovações: a falta de trilha não apaga o ato.
	decididas, err := store.ListApprovals("org-a", ServiceAgentApprovalApproved, now)
	if err != nil || len(decididas) != 1 || decididas[0].DecidedBy != "ana" {
		t.Fatalf("a decisão precisa continuar registrada: %+v (%v)", decididas, err)
	}
}

func TestServiceAgentPolicyAuditKeepsTheHashChain(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	auditRoot := serviceAgentAuditRoot(t.TempDir())
	runtime.SetAuditStore(NewServiceAgentAuditStore(auditRoot))
	store := runtime.ApprovalStore()

	// Um turno (entrada de ferramenta) + um ato de política: os dois convivem na MESMA cadeia.
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano avançado", 79900), now); err != nil {
		t.Fatal(err)
	}
	if err := runtime.RecordApproversChange("org-a", "ana", []string{"ana"}, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	verification, err := runtime.audit.VerifyTrail("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Valid {
		t.Fatalf("misturar eventos de ferramenta e de política não pode quebrar a cadeia: %+v", verification)
	}
	if verification.Entries != 2 {
		t.Fatalf("esperava 2 entradas (ferramenta + política): %+v", verification)
	}
	_ = store
}
