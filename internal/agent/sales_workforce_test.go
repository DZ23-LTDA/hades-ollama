package agent

import (
	"errors"
	"testing"
	"time"
)

// Estágio 27 — propostas comerciais e pós-venda.
//
// Propriedades travadas:
//  1. o PREÇO vem do catálogo e o total é recalculado no servidor (não se aceita preço do
//     chamador nem SKU desconhecido);
//  2. ENVIAR exige consentimento: quem pediu opt-out NÃO recebe proposta;
//  3. proposta VENCIDA não pode ser aceita;
//  4. decisão é TERMINAL (não volta a rascunho);
//  5. pós-venda tem prazo, é IDEMPOTENTE por pedido e o atraso é CONTÁVEL;
//  6. tudo é escopado por organização.
func salesRuntime(t *testing.T) *Runtime {
	t.Helper()
	return factoryRuntime(t)
}

func seedSalesCatalog(t *testing.T, runtime *Runtime, organization string, now time.Time) {
	t.Helper()
	for _, product := range []CommerceProduct{
		{OrganizationID: organization, SKU: "PROP-1", Name: "Plano básico", PriceCents: 9900, Currency: "BRL", Active: true},
		{OrganizationID: organization, SKU: "PROP-2", Name: "Plano avançado", PriceCents: 19900, Currency: "BRL", Active: true},
	} {
		if _, err := runtime.Commerce().UpsertProduct(product, now); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProposalPricesComeFromCatalogAndSendingRequiresConsent(t *testing.T) {
	runtime := salesRuntime(t)
	now := time.Now().UTC()
	organization := LocalOrganizationID
	seedSalesCatalog(t, runtime, organization, now)

	proposal, err := runtime.SalesWorkforce().CreateProposal(SalesProposalRequest{
		OrganizationID: organization, ContactID: "contato-proposta", Notes: "condições iniciais",
		Items: []SalesProposalItemRequest{{SKU: "PROP-1", Quantity: 2}, {SKU: "PROP-2", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	// TOTAL recalculado no servidor: 2×9900 + 1×19900 = 39700.
	if proposal.TotalCents != 39700 {
		t.Fatalf("total precisa vir do catálogo: %d", proposal.TotalCents)
	}
	for _, item := range proposal.Items {
		if item.UnitPriceCents == 0 || item.TotalCents != item.UnitPriceCents*item.Quantity {
			t.Fatalf("linha com preço inconsistente: %+v", item)
		}
	}
	if proposal.Status != SalesProposalDraft || proposal.ValidUntil.Before(now) {
		t.Fatalf("proposta precisa nascer em rascunho com vigência: %+v", proposal)
	}

	// SKU fora do catálogo é ERRO, não linha com preço zero.
	if _, err := runtime.SalesWorkforce().CreateProposal(SalesProposalRequest{
		OrganizationID: organization, ContactID: "contato-proposta",
		Items: []SalesProposalItemRequest{{SKU: "NAO-EXISTE", Quantity: 1}},
	}, now); !errors.Is(err, ErrProposalUnknownSKU) {
		t.Fatalf("SKU desconhecido precisa falhar, veio %v", err)
	}
	// Quantidade não positiva é erro.
	if _, err := runtime.SalesWorkforce().CreateProposal(SalesProposalRequest{
		OrganizationID: organization, ContactID: "contato-proposta",
		Items: []SalesProposalItemRequest{{SKU: "PROP-1", Quantity: 0}},
	}, now); !errors.Is(err, ErrProposalInvalid) {
		t.Fatalf("quantidade zero precisa falhar, veio %v", err)
	}

	// SEM consentimento o envio é RECUSADO.
	if _, err := runtime.SalesWorkforce().SendProposal(organization, proposal.ID, "whatsapp", "vendedor", now); !errors.Is(err, ErrProposalConsentRequired) {
		t.Fatalf("sem consentimento o envio precisa ser recusado, veio %v", err)
	}
	// Consentimento concedido: o envio passa.
	if _, err := runtime.Consent().GrantConsent(organization, "whatsapp", "contato-proposta", "formulario", "aceite", "operador", nil, now); err != nil {
		t.Fatal(err)
	}
	sent, err := runtime.SalesWorkforce().SendProposal(organization, proposal.ID, "whatsapp", "vendedor", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if sent.Status != SalesProposalSent || sent.SentAt == nil {
		t.Fatalf("a proposta precisa ficar enviada com data: %+v", sent)
	}

	// OPT-OUT depois do envio: uma NOVA proposta não pode ser enviada.
	if _, err := runtime.Consent().RevokeConsent(organization, "whatsapp", "contato-proposta", "pediu para parar", "contato", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	second, err := runtime.SalesWorkforce().CreateProposal(SalesProposalRequest{
		OrganizationID: organization, ContactID: "contato-proposta",
		Items: []SalesProposalItemRequest{{SKU: "PROP-1", Quantity: 1}},
	}, now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.SalesWorkforce().SendProposal(organization, second.ID, "whatsapp", "vendedor", now.Add(3*time.Minute)); !errors.Is(err, ErrProposalConsentRequired) {
		t.Fatalf("depois do opt-out o envio precisa ser recusado, veio %v", err)
	}
}

func TestProposalExpiryAndTerminalDecision(t *testing.T) {
	runtime := salesRuntime(t)
	now := time.Now().UTC()
	organization := LocalOrganizationID
	seedSalesCatalog(t, runtime, organization, now)
	if _, err := runtime.Consent().GrantConsent(organization, "email", "contato-vigencia", "formulario", "aceite", "operador", nil, now); err != nil {
		t.Fatal(err)
	}

	proposal, err := runtime.SalesWorkforce().CreateProposal(SalesProposalRequest{
		OrganizationID: organization, ContactID: "contato-vigencia", Validity: time.Hour,
		Items: []SalesProposalItemRequest{{SKU: "PROP-1", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.SalesWorkforce().SendProposal(organization, proposal.ID, "email", "vendedor", now); err != nil {
		t.Fatal(err)
	}
	// Decidir ANTES de enviar é recusado (a decisão precisa de proposta enviada).
	draft, err := runtime.SalesWorkforce().CreateProposal(SalesProposalRequest{
		OrganizationID: organization, ContactID: "contato-vigencia",
		Items: []SalesProposalItemRequest{{SKU: "PROP-1", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.SalesWorkforce().DecideProposal(organization, draft.ID, "aceitar", "cliente", now); !errors.Is(err, ErrProposalStateInvalid) {
		t.Fatalf("decidir rascunho precisa falhar, veio %v", err)
	}

	// VENCIDA: aceitar depois da vigência é RECUSADO.
	if _, err := runtime.SalesWorkforce().DecideProposal(organization, proposal.ID, "aceitar", "cliente", now.Add(2*time.Hour)); !errors.Is(err, ErrProposalExpired) {
		t.Fatalf("proposta fora da vigência precisa ser recusada, veio %v", err)
	}
	// Dentro da vigência, aceitar funciona e a decisão é TERMINAL.
	accepted, err := runtime.SalesWorkforce().DecideProposal(organization, proposal.ID, "aceitar", "cliente", now.Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != SalesProposalAccepted || accepted.DecidedAt == nil || accepted.DecidedBy != "cliente" {
		t.Fatalf("decisão precisa ser registrada: %+v", accepted)
	}
	if _, err := runtime.SalesWorkforce().DecideProposal(organization, proposal.ID, "recusar", "cliente", now.Add(40*time.Minute)); !errors.Is(err, ErrProposalStateInvalid) {
		t.Fatalf("decisão terminal não pode ser refeita, veio %v", err)
	}
	// Decisão desconhecida é erro declarado.
	another, err := runtime.SalesWorkforce().CreateProposal(SalesProposalRequest{
		OrganizationID: organization, ContactID: "contato-vigencia",
		Items: []SalesProposalItemRequest{{SKU: "PROP-2", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.SalesWorkforce().SendProposal(organization, another.ID, "email", "vendedor", now); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.SalesWorkforce().DecideProposal(organization, another.ID, "talvez", "cliente", now.Add(time.Minute)); !errors.Is(err, ErrProposalInvalid) {
		t.Fatalf("decisão desconhecida precisa falhar, veio %v", err)
	}
}

func TestFollowUpIsIdempotentPerOrderAndOverdueIsMeasurable(t *testing.T) {
	runtime := salesRuntime(t)
	now := time.Now().UTC()
	organization := LocalOrganizationID

	followUp, err := runtime.SalesWorkforce().ScheduleFollowUp(organization, "contato-pos", "ord-1", "confirmar satisfação", 0, now)
	if err != nil {
		t.Fatal(err)
	}
	if followUp.Status != SalesFollowUpScheduled {
		t.Fatalf("acompanhamento precisa nascer agendado: %+v", followUp)
	}
	if got := followUp.DueAt.Sub(now); got != DefaultPostSaleFollowUp {
		t.Fatalf("prazo padrão precisa ser usado: %s", got)
	}
	// IDEMPOTENTE por pedido: agendar de novo o MESMO pedido devolve o mesmo acompanhamento.
	again, err := runtime.SalesWorkforce().ScheduleFollowUp(organization, "contato-pos", "ord-1", "confirmar satisfação", 0, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != followUp.ID {
		t.Fatalf("não pode duplicar acompanhamento do mesmo pedido: %s vs %s", again.ID, followUp.ID)
	}
	// Motivo é obrigatório.
	if _, err := runtime.SalesWorkforce().ScheduleFollowUp(organization, "contato-pos", "ord-2", "   ", 0, now); !errors.Is(err, ErrFollowUpInvalid) {
		t.Fatalf("motivo vazio precisa falhar, veio %v", err)
	}
	// Escopo obrigatório.
	//
	// E-341: a guarda `requireOrganization` faz a recusa ANTES do módulo, com erro próprio
	// (`ErrTenantScopeRequired`). O contrato aqui e' "organizacao ausente PRECISA FALHAR" - e ela falha, mais cedo e
	// com motivo mais especifico. Aceitar as DUAS formas e' registrar a evolucao, nao afrouxar a exigencia.
	if _, err := runtime.SalesWorkforce().ScheduleFollowUp("", "contato-pos", "ord-3", "motivo", 0, now); !errors.Is(err, ErrFollowUpInvalid) && !errors.Is(err, ErrTenantScopeRequired) {
		t.Fatalf("organização ausente precisa falhar (ErrFollowUpInvalid ou ErrTenantScopeRequired), veio %v", err)
	}

	// ATRASO é MEDIDO: antes do prazo não há atraso; depois, sim.
	if overdue := runtime.SalesWorkforce().FollowUps(organization, true, now.Add(time.Hour)); len(overdue) != 0 {
		t.Fatalf("não pode haver atraso antes do prazo: %+v", overdue)
	}
	overdue := runtime.SalesWorkforce().FollowUps(organization, true, now.Add(DefaultPostSaleFollowUp+time.Hour))
	if len(overdue) != 1 || overdue[0].ID != followUp.ID {
		t.Fatalf("o atraso precisa ser contável: %+v", overdue)
	}

	// Concluir fecha; concluir de novo é recusado.
	done, err := runtime.SalesWorkforce().CompleteFollowUp(organization, followUp.ID, "pos-venda", "", now.Add(80*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != SalesFollowUpDone || done.CompletedAt == nil {
		t.Fatalf("acompanhamento precisa ficar concluído: %+v", done)
	}
	if _, err := runtime.SalesWorkforce().CompleteFollowUp(organization, followUp.ID, "pos-venda", "", now.Add(90*time.Hour)); !errors.Is(err, ErrFollowUpStateInvalid) {
		t.Fatalf("concluir duas vezes precisa falhar, veio %v", err)
	}
	// Concluído não conta mais como atraso.
	if overdue := runtime.SalesWorkforce().FollowUps(organization, true, now.Add(100*time.Hour)); len(overdue) != 0 {
		t.Fatalf("acompanhamento concluído não pode contar como atraso: %+v", overdue)
	}
}

func TestSalesWorkforceIsScopedByOrganization(t *testing.T) {
	runtime := salesRuntime(t)
	now := time.Now().UTC()
	seedSalesCatalog(t, runtime, LocalOrganizationID, now)

	proposal, err := runtime.SalesWorkforce().CreateProposal(SalesProposalRequest{
		OrganizationID: LocalOrganizationID, ContactID: "contato-escopo",
		Items: []SalesProposalItemRequest{{SKU: "PROP-1", Quantity: 1}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.SalesWorkforce().ScheduleFollowUp(LocalOrganizationID, "contato-escopo", "ord-escopo", "motivo", 0, now); err != nil {
		t.Fatal(err)
	}
	// Outra organização NÃO vê nada.
	if proposals := runtime.SalesWorkforce().Proposals("org-vizinho", ""); len(proposals) != 0 {
		t.Fatalf("propostas precisam ser por organização: %+v", proposals)
	}
	if followUps := runtime.SalesWorkforce().FollowUps("org-vizinho", false, now); len(followUps) != 0 {
		t.Fatalf("pós-venda precisa ser por organização: %+v", followUps)
	}
	if _, err := runtime.SalesWorkforce().Proposal("org-vizinho", proposal.ID); !errors.Is(err, ErrProposalNotFound) {
		t.Fatalf("proposta de outra organização precisa ser inacessível, veio %v", err)
	}
	// E a própria organização vê.
	if proposals := runtime.SalesWorkforce().Proposals(LocalOrganizationID, SalesProposalDraft); len(proposals) != 1 {
		t.Fatalf("a organização dona precisa ver a proposta: %+v", proposals)
	}
}
