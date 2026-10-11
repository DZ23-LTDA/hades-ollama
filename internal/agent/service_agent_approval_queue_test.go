package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// PRAZO DO PENDENTE e FILA com idade (estágio 35) — o limite declarado era "aprovação parcial não expira por
// si", e a pendência operacional era "o aprovador não é avisado".
//
// O que estes testes travam:
//
//  1. pedido pendente VENCE: sem decisão a tempo, ele expira e exige pedido novo;
//  2. a assinatura PARCIAL de um pedido vencido NÃO conta para o pedido seguinte — quórum com assinatura
//     velha não é quórum, é carimbo;
//  3. decidir um pedido VENCIDO é recusado (reabriria um contexto que ninguém revisou a tempo);
//  4. a FILA diz o que falta, há quanto tempo, e o que passou do limiar de atenção;
//  5. quem opera vê isso pela rota, sem depender de push que não existe.
func TestServiceAgentPendingApprovalExpiresAndRequiresNewRequest(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	antes := len(crm.ListOpportunities("org-a", ""))
	pedido := salesWriteTurn(conversationID, "Plano avançado", 79900)

	turn, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	primeiroPedido := call.ApprovalID

	// MUITO depois do prazo do pendente (padrão 24h): o pedido vence e é MARCADO como tal.
	depois := now.Add(48 * time.Hour)
	turn, err = runtime.RunTurn(pedido, depois)
	if err != nil {
		t.Fatal(err)
	}
	call = callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("pedido vencido precisa exigir decisão nova: %+v", turn.ToolCalls)
	}
	if call.ApprovalID == primeiroPedido {
		t.Fatal("depois de vencer, o pedido precisa ser NOVO — não o vencido")
	}
	// O vencido fica registrado como expirado: evidência, não sumiço.
	expirados, err := store.ListApprovals("org-a", ServiceAgentApprovalExpired, depois)
	if err != nil || len(expirados) != 1 || expirados[0].ID != primeiroPedido {
		t.Fatalf("o pedido vencido precisa ficar marcado: %+v (%v)", expirados, err)
	}
	// E decidir o vencido é RECUSADO.
	if _, err := store.Decide("org-a", primeiroPedido, true, "ana", "tarde demais", depois); !errors.Is(err, ErrServiceAgentApprovalExpired) {
		t.Fatalf("decidir pedido vencido precisa ser recusado: %v", err)
	}
	if depois2 := len(crm.ListOpportunities("org-a", "")); depois2 != antes {
		t.Fatalf("nada pode ser escrito: antes=%d depois=%d", antes, depois2)
	}
}

func TestServiceAgentStalePartialApprovalDoesNotCount(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	// Valor alto: exige duas assinaturas distintas.
	pedido := salesWriteTurn(conversationID, "Plano corporativo", 250000)

	turn, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	if _, err := store.Decide("org-a", call.ApprovalID, true, "gerente-ana", "primeiro olhar", now); err != nil {
		t.Fatal(err)
	}

	// Dois dias depois: o pedido com UMA assinatura vence. A assinatura antiga morre com ele — é o ponto.
	depois := now.Add(48 * time.Hour)
	turn, err = runtime.RunTurn(pedido, depois)
	if err != nil {
		t.Fatal(err)
	}
	call = callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("o pedido novo precisa estar pendente: %+v", turn.ToolCalls)
	}
	pendente, err := store.ListApprovals("org-a", ServiceAgentApprovalPending, depois)
	if err != nil || len(pendente) != 1 {
		t.Fatalf("um único pedido pendente: %+v (%v)", pendente, err)
	}
	if len(pendente[0].Approvers()) != 0 {
		t.Fatalf("a assinatura VELHA não pode contar para o pedido novo: %+v", pendente[0].Approvers())
	}
	if pendente[0].RequiredApprovals != 2 {
		t.Fatalf("o pedido novo mantém o quórum de duas pessoas: %+v", pendente[0])
	}
}

func TestServiceAgentApprovalQueueShowsAgeAndOverdue(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()

	// Um pedido recente e um pedido antigo (mas ainda dentro do prazo de 24h).
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Recente", 79900), now); err != nil {
		t.Fatal(err)
	}
	antigo := now.Add(-6 * time.Hour)
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Antigo", 45000), antigo); err != nil {
		t.Fatal(err)
	}

	fila, err := store.ApprovalQueue("org-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if fila.Pending != 2 {
		t.Fatalf("a fila precisa listar os dois pedidos: %+v", fila)
	}
	if fila.Overdue != 1 {
		t.Fatalf("só o pedido de 6h está atrasado (limiar de 4h): %+v", fila)
	}
	if fila.OldestSeconds < int64(5*time.Hour.Seconds()) {
		t.Fatalf("a idade do mais antigo precisa aparecer: %+v", fila)
	}
	// Mais antigos primeiro: a fila existe para atacar o que espera há mais tempo.
	if fila.Items[0].AgeSeconds < fila.Items[1].AgeSeconds {
		t.Fatalf("a fila precisa ordenar do mais antigo: %+v", fila.Items)
	}
	if !fila.Items[0].Overdue || fila.Items[1].Overdue {
		t.Fatalf("a marcação de atraso precisa ser por item: %+v", fila.Items)
	}
	if fila.Items[0].ApprovalsNeeded != 1 {
		t.Fatalf("o que falta precisa ser dito: %+v", fila.Items[0])
	}
	// O prazo vigente vai na resposta: quem opera sabe quando o pedido vence.
	if fila.PendingTTL != int64(DefaultServiceAgentPendingTTL.Seconds()) {
		t.Fatalf("o prazo do pendente precisa ser declarado: %+v", fila)
	}
}

func TestServiceAgentApprovalQueueCountsExpired(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Vai vencer", 79900), now); err != nil {
		t.Fatal(err)
	}

	fila, err := store.ApprovalQueue("org-a", now.Add(48*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if fila.Pending != 0 {
		t.Fatalf("pedido vencido não é pendente: %+v", fila)
	}
	if fila.ExpiresCount != 1 {
		t.Fatalf("a fila precisa contar o que venceu: %+v", fila)
	}
	// E a listagem por estado confirma o registro (evidência preservada).
	expirados, err := store.ListApprovals("org-a", ServiceAgentApprovalExpired, now.Add(48*time.Hour))
	if err != nil || len(expirados) != 1 {
		t.Fatalf("o vencido precisa constar como expirado: %+v (%v)", expirados, err)
	}
}

func TestServiceAgentPendingTTLZeroDisablesExpiry(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	store.PendingTTL = 0 // desligar o prazo é escolha explícita
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Sem prazo", 79900), now); err != nil {
		t.Fatal(err)
	}
	// Muito depois: sem prazo, o pedido continua pendente e decidível.
	depois := now.Add(30 * 24 * time.Hour)
	fila, err := store.ApprovalQueue("org-a", depois)
	if err != nil {
		t.Fatal(err)
	}
	if fila.Pending != 1 || fila.ExpiresCount != 0 {
		t.Fatalf("sem prazo o pedido não pode vencer: %+v", fila)
	}
	if strings.TrimSpace(fila.Items[0].ID) == "" {
		t.Fatalf("o item precisa ser identificável: %+v", fila.Items[0])
	}
}
