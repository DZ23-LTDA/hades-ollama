package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// PRAZO e PAPEL da aprovação (estágio 35) — os dois limites que a rodada anterior declarou.
//
// O que estes testes travam:
//
//  1. aprovação VENCIDA não autoriza: o turno pede decisão nova e o CRM não muda;
//  2. a vencida é MARCADA como expirada (evidência), não apagada;
//  3. dentro do prazo, executa normalmente;
//  4. com LISTA DE APROVADORES, quem não está nela é RECUSADO;
//  5. sem lista, a decisão é aceita e gravada com `RoleChecked = false` — a ausência de verificação é DADO;
//  6. `ApprovalTTL = 0` desliga o prazo de forma EXPLÍCITA (escolha, não esquecimento).
func decidePending(t *testing.T, store *ServiceAgentApprovalStore, organizationID, toolName, actor string, when time.Time) ServiceAgentApproval {
	t.Helper()
	pending, err := store.ListApprovals(organizationID, ServiceAgentApprovalPending, when)
	if err != nil {
		t.Fatal(err)
	}
	for _, approval := range pending {
		if approval.ToolName == toolName {
			decided, err := store.Decide(organizationID, approval.ID, true, actor, "decidido no teste", when)
			if err != nil {
				t.Fatal(err)
			}
			return decided
		}
	}
	t.Fatalf("não há pedido pendente para %s: %+v", toolName, pending)
	return ServiceAgentApproval{}
}

func TestServiceAgentApprovalExpiresAndIsMarked(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	antes := len(crm.ListOpportunities("org-a", ""))

	pedido := salesWriteTurn(conversationID, "Plano avançado", 79900)
	if _, err := runtime.RunTurn(pedido, now); err != nil {
		t.Fatal(err)
	}
	// Decisão dada MUITO antes do uso: com TTL de 30 minutos, a aprovação já nasce vencida.
	decidida := decidePending(t, store, "org-a", "crm.opportunity.write", "ana", now.Add(-2*time.Hour))
	if decidida.ExpiresAt == nil {
		t.Fatal("a aprovação com prazo precisa declarar quando vence")
	}

	turn, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	// Vencida não autoriza: o caminho volta a exigir decisão, e NADA foi gravado.
	if call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("aprovação vencida não pode autorizar: %+v", turn.ToolCalls)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes {
		t.Fatalf("aprovação vencida não pode escrever: antes=%d depois=%d", antes, depois)
	}
	// E o registro conta a verdade: a aprovação aparece como EXPIRADA (evidência), não apagada.
	expiradas, err := store.ListApprovals("org-a", ServiceAgentApprovalExpired, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(expiradas) != 1 || expiradas[0].ID != decidida.ID {
		t.Fatalf("a aprovação vencida precisa ficar marcada como expirada: %+v", expiradas)
	}
	// O pedido novo é OUTRO, para o humano decidir de novo.
	if call.ApprovalID == decidida.ID {
		t.Fatal("depois de vencer, é preciso um pedido NOVO — não reusar o vencido")
	}
}

func TestServiceAgentApprovalWithinTTLStillExecutes(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	antes := len(crm.ListOpportunities("org-a", ""))
	pedido := salesWriteTurn(conversationID, "Plano avançado", 79900)

	if _, err := runtime.RunTurn(pedido, now); err != nil {
		t.Fatal(err)
	}
	// Decisão recente (1 minuto atrás) e uso dentro do prazo de 30 minutos.
	decidePending(t, runtime.ApprovalStore(), "org-a", "crm.opportunity.write", "ana", now.Add(-time.Minute))

	turn, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("dentro do prazo a aprovação precisa valer: %+v", turn.ToolCalls)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes+1 {
		t.Fatalf("a escrita aprovada precisa existir: antes=%d depois=%d", antes, depois)
	}
}

func TestServiceAgentApprovalRoleCheckRefusesUnlistedActor(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	if err := store.SetApprovers("org-a", []string{"ana", "gerente-bruno"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano avançado", 79900), now); err != nil {
		t.Fatal(err)
	}
	pending, err := store.ListApprovals("org-a", ServiceAgentApprovalPending, now)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pedido pendente ausente: %+v (%v)", pending, err)
	}

	// Ator FORA da lista: recusa declarada, e a aprovação continua pendente.
	_, err = store.Decide("org-a", pending[0].ID, true, "estagiario", "", now)
	if !errors.Is(err, ErrServiceAgentApprovalNotAllowed) {
		t.Fatalf("ator fora da lista precisa ser RECUSADO: %v", err)
	}
	aindaPendente, err := store.ListApprovals("org-a", ServiceAgentApprovalPending, now)
	if err != nil || len(aindaPendente) != 1 {
		t.Fatalf("a recusa não pode consumir nem decidir o pedido: %+v", aindaPendente)
	}

	// Ator NA lista: aprova, e o registro DIZ que o papel foi conferido.
	decided, err := store.Decide("org-a", pending[0].ID, true, "gerente-bruno", "ok", now)
	if err != nil {
		t.Fatal(err)
	}
	if !decided.RoleChecked {
		t.Fatal("com lista configurada, RoleChecked precisa ser true")
	}
}

func TestServiceAgentApprovalWithoutApproversDeclaresNoRoleCheck(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano avançado", 79900), now); err != nil {
		t.Fatal(err)
	}
	// Sem lista, qualquer ator identificado pode decidir — e o registro DECLARA que não houve checagem.
	decided := decidePending(t, store, "org-a", "crm.opportunity.write", "qualquer-um", now)
	if decided.RoleChecked {
		t.Fatal("sem lista de aprovadores, RoleChecked precisa ser false — a ausência de verificação é dado")
	}
	if decided.DecidedBy != "qualquer-um" {
		t.Fatalf("quem decidiu precisa constar: %+v", decided)
	}
}

func TestServiceAgentApprovalTTLZeroDisablesExpiryExplicitly(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	store.ApprovalTTL = 0 // desligar o prazo é escolha explícita
	antes := len(crm.ListOpportunities("org-a", ""))
	pedido := salesWriteTurn(conversationID, "Plano avançado", 79900)

	if _, err := runtime.RunTurn(pedido, now); err != nil {
		t.Fatal(err)
	}
	decidida := decidePending(t, store, "org-a", "crm.opportunity.write", "ana", now.Add(-6*time.Hour))
	if decidida.ExpiresAt != nil {
		t.Fatal("sem prazo, a aprovação não pode declarar vencimento")
	}
	turn, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "crm.opportunity.write"); call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("com o prazo desligado, a aprovação continua valendo: %+v", turn.ToolCalls)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes+1 {
		t.Fatalf("a escrita precisa acontecer: antes=%d depois=%d", antes, depois)
	}
}

func TestServiceAgentApprovalConsumeRefusesExpiredEvenIfApproved(t *testing.T) {
	store := NewServiceAgentApprovalStore(serviceAgentApprovalRoot(t.TempDir()))
	now := time.Now().UTC()
	// Caminho direto no store: aprova no passado e tenta consumir agora. O prazo precisa valer no CONSUMO,
	// e não só na busca — um caminho que consome sem conferir anularia o prazo.
	requested, err := store.Request("org-a", ServiceAgentApproval{
		OrganizationID: "org-a", ToolName: "crm.opportunity.write", Fingerprint: "fp-x", RequestedAt: now.Add(-3 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Decide("org-a", requested.ID, true, "ana", "", now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Consume("org-a", requested.ID, now); !errors.Is(err, ErrServiceAgentApprovalExpired) {
		t.Fatalf("consumir aprovação vencida precisa ser RECUSADO: %v", err)
	}
	expiradas, err := store.ListApprovals("org-a", ServiceAgentApprovalExpired, now)
	if err != nil || len(expiradas) != 1 {
		t.Fatalf("a aprovação precisa ficar marcada como expirada: %+v", expiradas)
	}
}

func TestServiceAgentApprovalApproversAreScopedAndDurable(t *testing.T) {
	root := serviceAgentApprovalRoot(t.TempDir())
	store := NewServiceAgentApprovalStore(root)
	if err := store.SetApprovers("org-a", []string{" ana ", "", "gerente-bruno"}); err != nil {
		t.Fatal(err)
	}
	// Reinício: a lista de quem aprova precisa sobreviver — senão a checagem de papel desaparece em silêncio.
	depois := NewServiceAgentApprovalStore(root)
	approvers, err := depois.Approvers("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(approvers) != 2 || approvers[0] != "ana" || approvers[1] != "gerente-bruno" {
		t.Fatalf("a lista precisa sobreviver e vir limpa: %+v", approvers)
	}
	// Isolamento: outra organização não herda a lista.
	outra, err := depois.Approvers("org-b")
	if err != nil {
		t.Fatal(err)
	}
	if len(outra) != 0 {
		t.Fatalf("lista de aprovadores não pode vazar entre organizações: %+v", outra)
	}
	// A gravação de aprovações PRESERVA a lista (o arquivo guarda as duas coisas).
	if _, err := depois.Request("org-a", ServiceAgentApproval{
		OrganizationID: "org-a", ToolName: "crm.opportunity.write", Fingerprint: "fp-y",
	}); err != nil {
		t.Fatal(err)
	}
	preservada, err := depois.Approvers("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(preservada, ",") != "ana,gerente-bruno" {
		t.Fatalf("gravar aprovação não pode apagar a lista de aprovadores: %+v", preservada)
	}
}
