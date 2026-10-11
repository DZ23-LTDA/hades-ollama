package agent

import (
	"strings"
	"testing"
	"time"
)

// QUÓRUM de aprovação para valores altos (estágio 35) — o limite declarado era "uma decisão de ator
// autorizado basta".
//
// O que estes testes travam:
//
//  1. acima do limite, UMA decisão NÃO basta: o pedido segue pendente e nada é escrito;
//  2. a MESMA pessoa decidindo duas vezes NÃO cumpre quórum — é o ponto inteiro da exigência;
//  3. duas pessoas distintas cumprem, e a escrita acontece UMA vez;
//  4. uma RECUSA encerra o pedido: quórum soma cuidados, não supera objeção;
//  5. abaixo do limite, uma decisão continua bastando (compatível com o que já existia);
//  6. a exigência fica GRAVADA no pedido: mudar a configuração depois não altera o que já foi exigido.
func TestServiceAgentQuorumRequiresDistinctApprovers(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	alto := salesWriteTurn(conversationID, "Plano corporativo", 250000) // R$ 2.500: acima do limite padrão
	antes := len(crm.ListOpportunities("org-a", ""))

	// Primeiro turno: pedido criado com exigência de DUAS decisões distintas.
	turn, err := runtime.RunTurn(alto, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("escrita de valor alto precisa ficar pendente: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Reason, "2 aprovações") || !strings.Contains(call.Reason, "0 de 2") {
		t.Fatalf("o motivo precisa declarar o quórum e o progresso: %q", call.Reason)
	}

	// PRIMEIRA aprovação: ainda não basta.
	if _, err := store.Decide("org-a", call.ApprovalID, true, "gerente-ana", "primeiro olhar", now); err != nil {
		t.Fatal(err)
	}
	turn, err = runtime.RunTurn(alto, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call = callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("com uma de duas aprovações a escrita NÃO pode executar: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Reason, "1 de 2") {
		t.Fatalf("o progresso do quórum precisa aparecer: %q", call.Reason)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes {
		t.Fatalf("nada pode ser escrito com quórum incompleto: antes=%d depois=%d", antes, depois)
	}

	// A MESMA pessoa de novo: continua não bastando — quórum de atores DISTINTOS.
	if _, err := store.Decide("org-a", call.ApprovalID, true, "gerente-ana", "de novo", now.Add(90*time.Second)); err != nil {
		t.Fatal(err)
	}
	pendente, err := store.ListApprovals("org-a", ServiceAgentApprovalPending, now.Add(2*time.Minute))
	if err != nil || len(pendente) != 1 {
		t.Fatalf("a mesma pessoa não pode cumprir o quórum: %+v (%v)", pendente, err)
	}
	if len(pendente[0].Approvers()) != 1 {
		t.Fatalf("aprovações DISTINTAS precisam ser contadas assim: %+v", pendente[0].Approvers())
	}

	// SEGUNDA pessoa: agora cumpre, e a escrita acontece UMA vez.
	if _, err := store.Decide("org-a", call.ApprovalID, true, "diretor-bruno", "segundo olhar", now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	turn, err = runtime.RunTurn(alto, now.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call = callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("com o quórum cumprido a escrita precisa executar: %+v", turn.ToolCalls)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes+1 {
		t.Fatalf("a oportunidade precisa existir UMA vez: antes=%d depois=%d", antes, depois)
	}
	// Uso único continua valendo: repetir exige decisão nova (e o quórum de novo).
	turn, err = runtime.RunTurn(alto, now.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "crm.opportunity.write"); call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("aprovação consumida não autoriza de novo: %+v", turn.ToolCalls)
	}
}

func TestServiceAgentQuorumDenialEndsTheRequest(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	alto := salesWriteTurn(conversationID, "Plano corporativo", 250000)
	antes := len(crm.ListOpportunities("org-a", ""))

	turn, err := runtime.RunTurn(alto, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	if _, err := store.Decide("org-a", call.ApprovalID, true, "gerente-ana", "primeiro olhar", now); err != nil {
		t.Fatal(err)
	}
	// A segunda pessoa RECUSA: o pedido encerra. Quórum existe para somar cuidados, não para superar objeção.
	if _, err := store.Decide("org-a", call.ApprovalID, false, "diretor-bruno", "valor fora da política", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	turn, err = runtime.RunTurn(alto, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call = callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolBlocked {
		t.Fatalf("recusa no quórum precisa BLOQUEAR: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Reason, "diretor-bruno") || !strings.Contains(call.Reason, "fora da política") {
		t.Fatalf("a recusa precisa nomear quem negou e por quê: %q", call.Reason)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes {
		t.Fatalf("nada pode ser escrito com recusa: antes=%d depois=%d", antes, depois)
	}
}

func TestServiceAgentQuorumBelowThresholdStillNeedsOne(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	// R$ 799 está ABAIXO do limite padrão: uma decisão continua bastando.
	baixo := salesWriteTurn(conversationID, "Plano avançado", 79900)
	antes := len(crm.ListOpportunities("org-a", ""))

	turn, err := runtime.RunTurn(baixo, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	pendente, err := store.ListApprovals("org-a", ServiceAgentApprovalPending, now)
	if err != nil || len(pendente) != 1 {
		t.Fatalf("pedido pendente ausente: %+v (%v)", pendente, err)
	}
	if pendente[0].RequiredApprovals != 1 {
		t.Fatalf("abaixo do limite a exigência é UMA decisão: %+v", pendente[0].RequiredApprovals)
	}
	if strings.Contains(call.Reason, "distintas") {
		t.Fatalf("abaixo do limite o motivo não pode falar em quórum: %q", call.Reason)
	}
	if _, err := store.Decide("org-a", call.ApprovalID, true, "gerente-ana", "ok", now); err != nil {
		t.Fatal(err)
	}
	turn, err = runtime.RunTurn(baixo, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "crm.opportunity.write"); call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("uma decisão precisa bastar abaixo do limite: %+v", turn.ToolCalls)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes+1 {
		t.Fatalf("a escrita precisa acontecer: antes=%d depois=%d", antes, depois)
	}
}

func TestServiceAgentQuorumIsFrozenInTheRequest(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	alto := salesWriteTurn(conversationID, "Plano corporativo", 250000)

	turn, err := runtime.RunTurn(alto, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")

	// A configuração MUDA depois do pedido: o que já foi exigido de quem pediu NÃO muda.
	store.QuorumAboveCents = 0
	pendente, err := store.ListApprovals("org-a", ServiceAgentApprovalPending, now)
	if err != nil || len(pendente) != 1 {
		t.Fatalf("pedido pendente ausente: %+v (%v)", pendente, err)
	}
	if pendente[0].RequiredApprovals != 2 {
		t.Fatalf("a exigência precisa ficar GRAVADA no pedido: %+v", pendente[0].RequiredApprovals)
	}
	// E uma única decisão continua insuficiente para ESTE pedido.
	if _, err := store.Decide("org-a", call.ApprovalID, true, "gerente-ana", "ok", now); err != nil {
		t.Fatal(err)
	}
	turn, err = runtime.RunTurn(alto, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "crm.opportunity.write"); call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("o quórum gravado precisa valer mesmo com a configuração desligada: %+v", turn.ToolCalls)
	}
}

func TestServiceAgentQuorumWithApproverListRequiresTwoAuthorizedPeople(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	store := runtime.ApprovalStore()
	// Duas pessoas autorizadas: o quórum precisa das DUAS, e o papel é conferido em cada decisão.
	if err := store.SetApprovers("org-a", []string{"gerente-ana", "diretor-bruno"}); err != nil {
		t.Fatal(err)
	}
	alto := salesWriteTurn(conversationID, "Plano corporativo", 250000)
	turn, err := runtime.RunTurn(alto, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")

	// Forasteiro não conta para o quórum nem decide nada.
	if _, err := store.Decide("org-a", call.ApprovalID, true, "estagiario", "", now); err == nil {
		t.Fatal("ator fora da lista precisa ser recusado mesmo sob quórum")
	}
	ana, err := store.Decide("org-a", call.ApprovalID, true, "gerente-ana", "ok", now)
	if err != nil {
		t.Fatal(err)
	}
	if !ana.RoleChecked {
		t.Fatal("com lista configurada, RoleChecked precisa ser true")
	}
	bruno, err := store.Decide("org-a", call.ApprovalID, true, "diretor-bruno", "ok", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if bruno.Status != ServiceAgentApprovalApproved {
		t.Fatalf("duas pessoas autorizadas cumprem o quórum: %+v", bruno)
	}
	turn, err = runtime.RunTurn(alto, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "crm.opportunity.write"); call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("cumprido o quórum, precisa executar: %+v", turn.ToolCalls)
	}
	if len(crm.ListOpportunities("org-a", "")) != 2 { // o fixture já cria uma
		t.Fatalf("a oportunidade precisa ser criada uma vez: %+v", crm.ListOpportunities("org-a", ""))
	}
}
