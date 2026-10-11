package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Aprovação HUMANA para ferramentas de escrita (estágio 35). O que estes testes travam:
//
//  1. SEM aprovação, a escrita NÃO roda e NADA é gravado no store real;
//  2. o pedido pendente NÃO se duplica quando o turno se repete;
//  3. a aprovação é de USO ÚNICO: depois de executar, exige decisão nova;
//  4. a aprovação é AMARRADA AOS PARÂMETROS da ação: aprovar um valor não autoriza outro;
//  5. NEGADO é decidido, não ignorado — a tentativa seguinte é BLOQUEADA com quem negou;
//  6. sem ator humano não há decisão (não se aprova no anonimato);
//  7. aprovações SOBREVIVEM AO REINÍCIO e são ISOLADAS por organização;
//  8. sem store de aprovações, a escrita é RECUSADA (o padrão é negar).
func approvalFixture(t *testing.T) (*ServiceAgentRuntime, *CRMStore, string, time.Time) {
	t.Helper()
	runtime, _, crm, _, conversationID := serviceAgentRuntime(t, true)
	return runtime, crm, conversationID, time.Now().UTC()
}

func salesWriteTurn(conversationID, title string, valueCents int64) ServiceAgentTurnRequest {
	return ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSales, ConversationID: conversationID,
		Intent: "quero o plano", RequestedTools: []string{"crm.opportunity.write"},
		ToolArguments: ServiceAgentToolRequest{Title: title, ValueCents: valueCents, OwnerID: "ana"},
	}
}

func TestServiceAgentWriteRequiresApprovalAndWritesNothing(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	antes := len(crm.ListOpportunities("org-a", ""))

	turn, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano avançado", 79900), now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("escrita sem aprovação precisa ficar PENDENTE: %+v", turn.ToolCalls)
	}
	if call.ApprovalID == "" || !strings.Contains(call.Reason, "aprovação humana") {
		t.Fatalf("o pendente precisa ser identificável e explicado: %+v", call)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes {
		t.Fatalf("NADA pode ser gravado antes da aprovação: antes=%d depois=%d", antes, depois)
	}
	// E o pedido está registrado para o humano decidir.
	pending, err := runtime.ApprovalStore().ListApprovals("org-a", ServiceAgentApprovalPending)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != call.ApprovalID {
		t.Fatalf("o pedido pendente precisa estar registrado: %+v", pending)
	}
}

func TestServiceAgentPendingApprovalIsNotDuplicated(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	pedido := salesWriteTurn(conversationID, "Plano avançado", 79900)

	primeiro, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := runtime.RunTurn(pedido, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	// Repetir o turno enquanto o humano decide NÃO pode inundar a caixa de aprovações: um pedido
	// duplicado esconde quantas decisões realmente faltam.
	if callByName(primeiro.ToolCalls, "crm.opportunity.write").ApprovalID != callByName(segundo.ToolCalls, "crm.opportunity.write").ApprovalID {
		t.Fatal("o mesmo pedido precisa ser reusado, não duplicado")
	}
	pending, err := runtime.ApprovalStore().ListApprovals("org-a", ServiceAgentApprovalPending)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Fatalf("precisa existir UM pedido pendente: %+v", pending)
	}
}

func TestServiceAgentApprovalIsSingleUse(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	pedido := salesWriteTurn(conversationID, "Plano avançado", 79900)

	if _, err := runtime.RunTurn(pedido, now); err != nil {
		t.Fatal(err)
	}
	approvePendingForTool(t, runtime, "org-a", "crm.opportunity.write", "ana")

	// Primeira execução: autorizada.
	turn, err := runtime.RunTurn(pedido, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "crm.opportunity.write"); call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("com aprovação precisa executar: %+v", turn.ToolCalls)
	}
	// Segunda execução com os MESMOS parâmetros: exige decisão NOVA — aprovação pontual não vira
	// autorização permanente.
	turn, err = runtime.RunTurn(pedido, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("aprovação consumida não pode autorizar de novo: %+v", turn.ToolCalls)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != 2 {
		t.Fatalf("a oportunidade aprovada precisa existir uma vez: %d", depois)
	}
}

func TestServiceAgentApprovalIsBoundToArguments(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)

	// Aprova a criação de 100 centavos...
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano pequeno", 100), now); err != nil {
		t.Fatal(err)
	}
	approvePendingForTool(t, runtime, "org-a", "crm.opportunity.write", "ana")

	// ...e tenta criar um valor MUITO maior: a aprovação de um valor não pode valer para outra decisão.
	turn, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano gigante", 9999900), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolApprovalRequired {
		t.Fatalf("aprovação de um valor não pode autorizar outro: %+v", turn.ToolCalls)
	}
	for _, opportunity := range crm.ListOpportunities("org-a", "") {
		if opportunity.ValueCents == 9999900 {
			t.Fatal("a escrita NÃO aprovada foi gravada")
		}
	}
}

func TestServiceAgentDeniedWriteIsBlockedWithActor(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	pedido := salesWriteTurn(conversationID, "Plano avançado", 79900)
	antes := len(crm.ListOpportunities("org-a", ""))

	if _, err := runtime.RunTurn(pedido, now); err != nil {
		t.Fatal(err)
	}
	store := runtime.ApprovalStore()
	pending, err := store.ListApprovals("org-a", ServiceAgentApprovalPending)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pedido pendente ausente: %+v (%v)", pending, err)
	}
	if _, err := store.Decide("org-a", pending[0].ID, false, "gerente-ana", "valor fora da política", now); err != nil {
		t.Fatal(err)
	}

	turn, err := runtime.RunTurn(pedido, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	// Negado é DECIDIDO: o estado é bloqueado, e o motivo nomeia quem negou e por quê.
	if call == nil || call.Status != ServiceAgentToolBlocked {
		t.Fatalf("escrita negada precisa ficar BLOQUEADA: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Reason, "gerente-ana") || !strings.Contains(call.Reason, "fora da política") {
		t.Fatalf("a recusa precisa nomear o ator e o motivo: %q", call.Reason)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes {
		t.Fatalf("nada pode ser gravado numa escrita negada: antes=%d depois=%d", antes, depois)
	}
}

func TestServiceAgentApprovalDecideRequiresHumanActor(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano avançado", 79900), now); err != nil {
		t.Fatal(err)
	}
	store := runtime.ApprovalStore()
	pending, err := store.ListApprovals("org-a", ServiceAgentApprovalPending)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pedido pendente ausente: %+v (%v)", pending, err)
	}
	// Aprovar no anonimato não é aprovação: quem autoriza precisa ser identificável.
	if _, err := store.Decide("org-a", pending[0].ID, true, "   ", "", now); err == nil {
		t.Fatal("decidir sem ator humano precisa ser RECUSADO")
	}
	// Decidir sobre pedido de OUTRA organização não encontra nada.
	if _, err := store.Decide("org-b", pending[0].ID, true, "ana", "", now); err == nil {
		t.Fatal("decisão escopada por organização precisa recusar pedido de outra")
	}
}

func TestServiceAgentApprovalSurvivesRestartAndIsScoped(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	pedido := salesWriteTurn(conversationID, "Plano avançado", 79900)
	if _, err := runtime.RunTurn(pedido, now); err != nil {
		t.Fatal(err)
	}
	root := runtime.ApprovalStore().Root()
	approvePendingForTool(t, runtime, "org-a", "crm.opportunity.write", "ana")

	// Reinício: novo runtime lendo o MESMO diretório (o humano não pode ser obrigado a decidir de novo).
	depois, _, _, _, conversationDepois := serviceAgentRuntime(t, true)
	_ = conversationDepois
	depois.SetApprovalStore(NewServiceAgentApprovalStore(root))
	store := depois.ApprovalStore()

	approved, err := store.ListApprovals("org-a", ServiceAgentApprovalApproved)
	if err != nil {
		t.Fatal(err)
	}
	if len(approved) != 1 {
		t.Fatalf("a aprovação precisa sobreviver ao reinício: %+v", approved)
	}
	// Isolamento: a organização B não vê a aprovação de A.
	outra, err := store.ListApprovals("org-b", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(outra) != 0 {
		t.Fatalf("aprovação de outra organização não pode aparecer: %+v", outra)
	}
	// E o arquivo é por organização, no disco.
	if _, err := os.Stat(filepath.Join(root, sanitizeIDComponent("org-b")+".json")); err == nil {
		t.Fatal("organização sem aprovação não pode ter arquivo")
	}
}

func TestServiceAgentWriteWithoutApprovalStoreIsRefused(t *testing.T) {
	runtime, _, crm, _, conversationID := serviceAgentRuntime(t, true)
	// Sem store de aprovações, o runtime NÃO sabe pedir autorização — então não executa efeito.
	runtime.SetApprovalStore(nil)
	antes := len(crm.ListOpportunities("org-a", ""))

	turn, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano avançado", 79900), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolBlocked {
		t.Fatalf("sem store de aprovações a escrita precisa ser RECUSADA: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Reason, "store de aprovações") {
		t.Fatalf("o motivo precisa dizer o que falta: %q", call.Reason)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes {
		t.Fatalf("nada pode ser gravado: antes=%d depois=%d", antes, depois)
	}
}

func TestServiceAgentApprovalCapKeepsPendingDecisions(t *testing.T) {
	store := NewServiceAgentApprovalStore(serviceAgentApprovalRoot(t.TempDir()))
	store.MaxPerOrganization = 3
	now := time.Now().UTC()
	// Três decisões já tomadas...
	for index := range 3 {
		decidida, err := store.Request("org-a", ServiceAgentApproval{
			OrganizationID: "org-a", ToolName: "crm.opportunity.write", Fingerprint: fingerprintDeTeste(index),
			RequestedAt: now.Add(time.Duration(index) * time.Minute),
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Decide("org-a", decidida.ID, true, "ana", "", now); err != nil {
			t.Fatal(err)
		}
	}
	// ...e um pedido PENDENTE, que não pode ser descartado por causa de volume: descartá-lo silenciaria
	// uma decisão que alguém precisa tomar.
	if _, err := store.Request("org-a", ServiceAgentApproval{
		OrganizationID: "org-a", ToolName: "inbox.assign", Fingerprint: "fp-pendente", RequestedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Request("org-a", ServiceAgentApproval{
		OrganizationID: "org-a", ToolName: "crm.opportunity.write", Fingerprint: "fp-extra",
		RequestedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	pending, err := store.ListApprovals("org-a", ServiceAgentApprovalPending)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 {
		t.Fatalf("os pedidos pendentes precisam sobreviver ao teto: %+v", pending)
	}
	todas, err := store.ListApprovals("org-a", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(todas) != store.MaxPerOrganization {
		t.Fatalf("o teto precisa valer: %d", len(todas))
	}
}

func fingerprintDeTeste(index int) string {
	return "fp-" + string(rune('a'+index))
}
