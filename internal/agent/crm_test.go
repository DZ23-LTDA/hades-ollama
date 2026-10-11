package agent

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Estágios 27 e 35 / comando `/crm`.
//
// As propriedades travadas aqui:
//  1. DEDUPLICAÇÃO por identidade normalizada: o mesmo cliente por canais diferentes
//     é UM contato — duplicar quebra histórico, consentimento e cobrança;
//  2. contato SEM telefone e SEM e-mail é recusado: sem identidade não há como
//     reconhecer nem respeitar opt-out;
//  3. funil com indicadores por fase;
//  4. fase terminal é TERMINAL: `won`/`lost` não voltam atrás nem apagam o desfecho;
//  5. o consentimento vem do REGISTRO, nunca de uma cópia no contato;
//  6. escopo por organização e persistência.
func crmStore(t *testing.T) (*CRMStore, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "crm")
	store, err := NewCRMStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return store, root
}

func TestCRMContactDeduplicationByIdentity(t *testing.T) {
	store, _ := crmStore(t)
	now := time.Now().UTC()
	// Mesmo cliente, telefone escrito de três formas diferentes.
	first, created, err := store.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Ana Souza", Phone: "+55 (11) 98888-7777", Tags: []string{"loja"}}, now)
	if err != nil || !created {
		t.Fatalf("primeiro contato precisa ser criado (created=%v err=%v)", created, err)
	}
	second, createdAgain, err := store.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Ana S.", Phone: "5511988887777"}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if createdAgain {
		t.Fatal("o mesmo telefone normalizado NÃO pode criar segundo contato")
	}
	if second.ID != first.ID {
		t.Fatalf("a deduplicação precisa devolver o MESMO contato: %s vs %s", second.ID, first.ID)
	}
	// Terceiro formato: sem o NONO dígito — é o MESMO celular, e bases legadas
	// guardam as duas formas. Números fixos NÃO são mesclados (ver adiante).
	if _, createdThird, err := store.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Ana", Phone: "551188887777"}, now); err != nil {
		t.Fatal(err)
	} else if createdThird {
		t.Fatal("celular sem o nono dígito não pode duplicar o mesmo contato")
	}
	// Telefone FIXO com DDD não é celular: não pode ser mesclado por engano.
	if _, createdLandline, err := store.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Loja fixa", Phone: "551133334444"}, now); err != nil {
		t.Fatal(err)
	} else if !createdLandline {
		t.Fatal("telefone fixo é outro número e precisa criar contato próprio")
	}
	contacts := store.ListContacts("org-a", "")
	if len(contacts) != 2 {
		t.Fatalf("precisam existir dois contatos (celular + fixo): %+v", contacts)
	}

	// E-mail também deduplica, ignorando maiúsculas.
	if _, createdEmail, err := store.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Bruno", Email: "BRUNO@Exemplo.com"}, now); err != nil || !createdEmail {
		t.Fatalf("contato por e-mail precisa ser criado (created=%v err=%v)", createdEmail, err)
	}
	if _, createdEmailAgain, err := store.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Bruno Lima", Email: "bruno@exemplo.com"}, now); err != nil || createdEmailAgain {
		t.Fatalf("e-mail com caixa diferente não pode duplicar (created=%v err=%v)", createdEmailAgain, err)
	}
	// Agora são três: o celular, o fixo e o contato por e-mail.
	if got := len(store.ListContacts("org-a", "")); got != 3 {
		t.Fatalf("precisam existir três contatos (celular + fixo + e-mail): %d", got)
	}
}

func TestCRMRejectsContactWithoutIdentity(t *testing.T) {
	store, _ := crmStore(t)
	now := time.Now().UTC()
	casos := []CRMContact{
		{OrganizationID: "org-a", Name: "Sem identidade"},
		{OrganizationID: "org-a", Name: "", Phone: "+5511988887777"},
		{OrganizationID: "org-a", Name: "Telefone curto", Phone: "123"},
	}
	for index, contact := range casos {
		if _, _, err := store.UpsertContact(contact, now); !errors.Is(err, ErrCRMContactInvalid) {
			t.Fatalf("caso %d precisava ser recusado, veio %v", index, err)
		}
	}
	if got := len(store.ListContacts("org-a", "")); got != 0 {
		t.Fatalf("nada podia ser gravado: %d", got)
	}
}

func TestCRMPipelineAndTerminalStage(t *testing.T) {
	store, _ := crmStore(t)
	now := time.Now().UTC()
	contact, _, err := store.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Cliente", Phone: "+5511977776666"}, now)
	if err != nil {
		t.Fatal(err)
	}
	openDeal, err := store.CreateOpportunity("org-a", contact.ID, "Plano mensal", 29900, "ana", now)
	if err != nil {
		t.Fatal(err)
	}
	wonDeal, err := store.CreateOpportunity("org-a", contact.ID, "Implantação", 150000, "ana", now)
	if err != nil {
		t.Fatal(err)
	}
	lostDeal, err := store.CreateOpportunity("org-a", contact.ID, "Consultoria", 80000, "ana", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetOpportunityStage("org-a", wonDeal.ID, CRMStageWon, "ana", now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetOpportunityStage("org-a", lostDeal.ID, "perdido", "ana", now); err != nil {
		t.Fatal(err)
	}

	pipeline := store.PipelineSummary("org-a")
	if pipeline.OpenCount != 1 || pipeline.OpenValueCents != 29900 {
		t.Fatalf("abertas incorretas: %+v", pipeline)
	}
	if pipeline.WonValueCents != 150000 || pipeline.LostCount != 1 {
		t.Fatalf("ganhas/perdidas incorretas: %+v", pipeline)
	}
	totals := map[string]CRMPipelineStage{}
	for _, stage := range pipeline.Stages {
		totals[stage.Stage] = stage
	}
	if totals[CRMStageWon].Count != 1 || totals[CRMStageLost].Count != 1 || totals[CRMStageNew].Count != 1 {
		t.Fatalf("indicadores por fase incorretos: %+v", pipeline.Stages)
	}

	// Fase terminal NÃO volta atrás: o desfecho já registrado não pode ser apagado.
	if _, err := store.SetOpportunityStage("org-a", wonDeal.ID, CRMStageProposal, "ana", now); !errors.Is(err, ErrCRMStageTerminal) {
		t.Fatalf("reabrir oportunidade ganha precisa ser recusado, veio %v", err)
	}
	// Reafirmar a MESMA fase terminal é permitido (idempotente).
	if _, err := store.SetOpportunityStage("org-a", wonDeal.ID, CRMStageWon, "ana", now); err != nil {
		t.Fatalf("reafirmar a mesma fase não pode falhar: %v", err)
	}
	// Fase inválida é recusada.
	if _, err := store.SetOpportunityStage("org-a", openDeal.ID, "qualquer coisa", "ana", now); !errors.Is(err, ErrCRMStageInvalid) {
		t.Fatalf("fase inválida precisa ser recusada, veio %v", err)
	}
	// O `ClosedAt` fica gravado na oportunidade encerrada.
	for _, opportunity := range store.ListOpportunities("org-a", "") {
		if opportunity.ID == lostDeal.ID && opportunity.ClosedAt == nil {
			t.Fatalf("oportunidade perdida precisa ter data de fechamento: %+v", opportunity)
		}
	}
}

func TestCRMConsentIsReadNotCopied(t *testing.T) {
	store, _ := crmStore(t)
	registry, err := NewConsentRegistry(filepath.Join(t.TempDir(), "consent"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	phone := "+5511966665555"
	contact, _, err := store.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Titular", Phone: phone}, now)
	if err != nil {
		t.Fatal(err)
	}
	// Sem consentimento: o contato aparece com o canal bloqueado.
	view, err := store.ContactView("org-a", contact.ID, registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Consent) != 0 {
		t.Fatalf("não havia consentimento registrado: %+v", view.Consent)
	}

	// Consentimento vive no REGISTRO; a visão passa a refleti-lo sem cópia local.
	if _, err := registry.GrantConsent("org-a", ConsentChannelWhatsApp, phone, "site", "checkbox", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	view, err = store.ContactView("org-a", contact.ID, registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Consent) != 1 || view.Consent[0].Channel != ConsentChannelWhatsApp {
		t.Fatalf("a visão precisa mostrar o consentimento real: %+v", view.Consent)
	}
	if view.Blocked[ConsentChannelWhatsApp] {
		t.Fatalf("canal consentido não pode aparecer bloqueado: %+v", view.Blocked)
	}

	// Opt-out no registro reflete imediatamente: nada a sincronizar.
	if _, err := registry.RevokeConsent("org-a", ConsentChannelWhatsApp, phone, "pedido do titular", "bruno", now); err != nil {
		t.Fatal(err)
	}
	view, err = store.ContactView("org-a", contact.ID, registry)
	if err != nil {
		t.Fatal(err)
	}
	if !view.Blocked[ConsentChannelWhatsApp] {
		t.Fatalf("após opt-out o canal precisa aparecer bloqueado: %+v", view.Blocked)
	}
	// E o contato em si NÃO ganhou campo de consentimento.
	if strings.Contains(strings.ToLower(view.Contact.Notes), "consent") {
		t.Fatal("o CRM não pode guardar cópia do consentimento")
	}
}

func TestCRMSurvivesRestartAndIsScopedByOrganization(t *testing.T) {
	store, root := crmStore(t)
	now := time.Now().UTC()
	contactA, _, err := store.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Da A", Phone: "+5511955554444"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.UpsertContact(CRMContact{OrganizationID: "org-b", Name: "Da B", Phone: "+5511944443333"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOpportunity("org-a", contactA.ID, "Negócio A", 5000, "ana", now); err != nil {
		t.Fatal(err)
	}
	// Oportunidade de contato de OUTRA organização é recusada.
	contactB := store.ListContacts("org-b", "")
	if len(contactB) != 1 {
		t.Fatalf("contato da org-b: %+v", contactB)
	}
	if _, err := store.CreateOpportunity("org-a", contactB[0].ID, "Negócio cruzado", 100, "ana", now); !errors.Is(err, ErrCRMContactNotFound) {
		t.Fatalf("oportunidade cruzando organizações precisa ser recusada, veio %v", err)
	}
	// Leitura cruzada não enxerga o contato da outra organização.
	if _, err := store.GetContact("org-b", contactA.ID); !errors.Is(err, ErrCRMContactNotFound) {
		t.Fatalf("contato de outra organização não pode ser lido, veio %v", err)
	}
	if got := len(store.ListContacts("org-b", "")); got != 1 {
		t.Fatalf("a listagem precisa ser escopada: %d", got)
	}

	// REINÍCIO: contatos e oportunidades voltam do disco com o funil intacto.
	restarted, err := NewCRMStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(restarted.ListContacts("org-a", "")); got != 1 {
		t.Fatalf("o contato precisa sobreviver ao reinício: %d", got)
	}
	if got := len(restarted.ListOpportunities("org-a", "")); got != 1 {
		t.Fatalf("a oportunidade precisa sobreviver ao reinício: %d", got)
	}
	if summary := restarted.PipelineSummary("org-a"); summary.OpenCount != 1 {
		t.Fatalf("o funil precisa sobreviver ao reinício: %+v", summary)
	}
	// A busca funciona por nome, telefone e etiqueta.
	if got := len(restarted.ListContacts("org-a", "da a")); got != 1 {
		t.Fatalf("busca por nome: %d", got)
	}
	if got := len(restarted.ListContacts("org-a", "955554444")); got != 1 {
		t.Fatalf("busca por telefone: %d", got)
	}
}
