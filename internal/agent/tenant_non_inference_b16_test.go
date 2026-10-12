package agent

import (
	"context"
	"errors"
	"testing"
	"time"
)

// NÃO INFERÊNCIA NOS QUATRO MÓDULOS DO B-16 (E-192).
//
// O scanner calibrado (E-191) apontou estes quatro métodos como a mesma forma do B-15: "não existe" devolvia o erro
// CRU da busca e "é de outro tenant" devolvia um erro PRÓPRIO — e o mapeamento de status transformava isso em 404 ×
// 403, revelando a existência do ID. A correção unificou as recusas; estes testes provam a propriedade E impedem a
// regressão, comparando as duas negações.
//
// A comparação é feita por IDENTIDADE DO ERRO (errors.Is) e pela mensagem: duas recusas diferentes — ainda que ambas
// "erradas" — já bastam para o oráculo.

func negacoesIguais(t *testing.T, nome string, errOutro, errInexistente error) {
	t.Helper()
	if errOutro == nil {
		t.Fatalf("%s: leitura cross-tenant não pode ter sucesso", nome)
	}
	if errInexistente == nil {
		t.Fatalf("%s: leitura de recurso inexistente não pode ter sucesso", nome)
	}
	if errOutro.Error() != errInexistente.Error() {
		t.Fatalf(
			"%s: ORÁCULO DE ENUMERAÇÃO — a recusa de outro tenant (%q) difere da recusa de inexistente (%q)",
			nome, errOutro.Error(), errInexistente.Error())
	}
}

// builder: BuilderService.GetForOrganization
func TestBuilderGetForOrganizationDoesNotRevealExistence(t *testing.T) {
	serviço, err := NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := serviço.Create(context.Background(), BuilderSpec{Name: "Loja", Kind: BuilderWebsite})
	if err != nil {
		t.Fatal(err)
	}
	// O projeto criado pertence à organização do serviço/store; usamos uma organização intrusa que não é dona.
	_, errOutro := serviço.GetForOrganization(projeto.ID, "org-intrusa")
	_, errInexistente := serviço.GetForOrganization("proj_que_nao_existe", "org-intrusa")
	negacoesIguais(t, "builder.GetForOrganization", errOutro, errInexistente)
	if !errors.Is(errOutro, ErrBuilderForbidden) {
		t.Fatalf("a recusa unificada precisa ser ErrBuilderForbidden: %v", errOutro)
	}
}

// devices: DeviceStore.GetForOrganization
func TestDeviceGetForOrganizationDoesNotRevealExistence(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	code, _, err := store.StartPairing("user-1", "org-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	device, _, err := store.CompletePairing(code, "Notebook", "windows", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, errOutro := store.GetForOrganization(device.ID, "org-intrusa")
	_, errInexistente := store.GetForOrganization("dev_que_nao_existe", "org-intrusa")
	negacoesIguais(t, "devices.GetForOrganization", errOutro, errInexistente)
	if !errors.Is(errOutro, ErrDeviceForbidden) {
		t.Fatalf("a recusa unificada precisa ser ErrDeviceForbidden: %v", errOutro)
	}
}

// swarm: AgentOrchestrator.GetForOrganization
func TestOrchestrationGetForOrganizationDoesNotRevealExistence(t *testing.T) {
	runner := func(ctx context.Context, task AgentTask) (AgentResult, error) {
		return AgentResult{Output: "ok"}, nil
	}
	orquestrador, err := NewAgentOrchestrator(t.TempDir(), runner)
	if err != nil {
		t.Fatal(err)
	}
	job, err := orquestrador.PlanForOrganization("org-a", "objetivo", ".", "proj-1", []AgentRole{RoleSecurity}, AgentBudget{MaxAgents: 1, MaxRetries: 1, MaxOutputBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	_, errOutro := orquestrador.GetForOrganization(job.ID, "org-intrusa")
	_, errInexistente := orquestrador.GetForOrganization("job_que_nao_existe", "org-intrusa")
	negacoesIguais(t, "swarm.GetForOrganization", errOutro, errInexistente)
	if !errors.Is(errOutro, ErrOrchestrationForbidden) {
		t.Fatalf("a recusa unificada precisa ser ErrOrchestrationForbidden: %v", errOutro)
	}
}

// tel_agent_session: CompanyStore.TelAgentSessionForOrganization
func TestTelAgentSessionForOrganizationDoesNotRevealExistence(t *testing.T) {
	store, err := NewCompanyStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	empresa, err := store.Create(Company{OrganizationID: "org-a", Name: "Suporte"})
	if err != nil {
		t.Fatal(err)
	}
	_, sessão, err := store.StartTelAgentSession(empresa.ID, "org-a", "cliente-1", "whatsapp")
	if err != nil {
		t.Fatal(err)
	}
	_, errOutro := store.TelAgentSessionForOrganization(empresa.ID, "org-intrusa", sessão.ID)
	_, errInexistente := store.TelAgentSessionForOrganization("cmp_que_nao_existe", "org-intrusa", "sess_que_nao_existe")
	negacoesIguais(t, "tel_agent_session.TelAgentSessionForOrganization", errOutro, errInexistente)
	if !errors.Is(errOutro, ErrTelAgentOrganizationMismatch) {
		t.Fatalf("a recusa unificada precisa ser ErrTelAgentOrganizationMismatch: %v", errOutro)
	}
}

// CONTROLE POSITIVO do conjunto: o dono precisa continuar conseguindo ler. Sem isto, "sempre negar" passaria nos
// quatro testes acima e a suíte estaria provando nada.
func TestOwnersStillReadTheirOwnResources(t *testing.T) {
	serviço, err := NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := serviço.Create(context.Background(), BuilderSpec{Name: "Loja", Kind: BuilderWebsite})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := serviço.GetForOrganization(projeto.ID, projeto.OrganizationID); err != nil {
		t.Fatalf("o dono precisa ler o próprio projeto: %v", err)
	}

	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	code, _, err := store.StartPairing("user-1", "org-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	device, _, err := store.CompletePairing(code, "Notebook", "windows", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetForOrganization(device.ID, device.OrganizationID); err != nil {
		t.Fatalf("o dono precisa ler o próprio dispositivo: %v", err)
	}
}
