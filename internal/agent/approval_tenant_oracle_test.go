package agent

import (
	"context"
	"testing"
	"time"
)

// ORÁCULO DE EXISTÊNCIA ENTRE TENANTS NA DECISÃO DE APROVAÇÃO (E-253).
//
// `decideApprovalForActor` percorre as aprovações da missão e, DENTRO do laço, checa estado ANTES de checar a
// organização:
//
//	L20  aprovação EXPIRADA  -> "approval has expired"
//	L23  outra ORGANIZAÇÃO   -> "approval organization mismatch"
//	L91  não existe          -> "approval not found or already decided"
//
// Com isso, um ator de OUTRA organização distingue três situações que deveriam ser indistinguíveis:
// (a) "existe e está pendente", (b) "existe e está expirada", (c) "não existe" — e enumera aprovações alheias.
// É a mesma família de E-198 (fila) e E-252 (refresh de OAuth): a checagem de tenant tem de vir ANTES das de estado.
//
// O teste compara as três recusas; se voltarem a divergir, ele falha.
func TestDecisaoDeAprovacaoNaoRevelaExistenciaEntreTenants(t *testing.T) {
	root := t.TempDir()
	planner := fixedPlanner{steps: []Step{
		{
			ID: "step_1", Kind: "workspace.write", Title: "escrever a.txt", Risk: RiskWrite, RequiresApproval: true, State: StepPending,
			Input: map[string]any{"path": "a.txt", "content": "conteudo\n"},
		},
	}}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: planner, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	missao, err := runtime.CreateMission(ctx, CreateMissionRequest{
		Objective:      "aprovar escrita",
		Capabilities:   []string{"workspace:read", "workspace:write"},
		OrganizationID: "org-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if missao.OrganizationID != "org-a" {
		t.Fatalf("a missão de teste precisa pertencer a org-a, pertence a %q", missao.OrganizationID)
	}
	if len(missao.Approvals) != 1 {
		t.Fatalf("a missão de teste precisa de UMA aprovação, veio %d", len(missao.Approvals))
	}
	const intrusa = "org-intrusa"

	decidir := func(approvalID string) error {
		_, err := runtime.DecideApprovalForActor(missao.ID, approvalID, true, "tentativa de outro tenant", "local", intrusa)
		return err
	}

	errPendente := decidir(missao.Approvals[0].ID)
	errInexistente := decidir("aprovacao_que_nao_existe")
	if errPendente == nil || errInexistente == nil {
		t.Fatalf("as duas recusas precisam existir (pendente=%v, inexistente=%v)", errPendente, errInexistente)
	}
	if errPendente.Error() != "approval not found or already decided" {
		t.Fatalf("a recusa entre tenants precisa ser INDISTINGUÍVEL de \"não encontrado\", veio %q", errPendente.Error())
	}
	if errPendente.Error() != errInexistente.Error() {
		t.Fatalf(
			"ORÁCULO DE EXISTÊNCIA ENTRE TENANTS: recusa de aprovação EXISTENTE de outro tenant (%q) difere da recusa de INEXISTENTE (%q)",
			errPendente.Error(), errInexistente.Error())
	}

	// (b) aprovação EXPIRADA de outro tenant: a checagem de expiração vinha ANTES da de organização.
	atual, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatal(err)
	}
	passado := time.Now().UTC().Add(-time.Hour)
	atual.Approvals[0].ExpiresAt = &passado
	// A store exige que a versão avance EXATAMENTE uma vez: incrementar e informar a anterior.
	atual.Version++
	if err := runtime.store.PutMissionIfVersion(atual, atual.Version-1); err != nil {
		t.Fatalf("gravar aprovação expirada: %v", err)
	}
	errExpirada := decidir(missao.Approvals[0].ID)
	if errExpirada == nil {
		t.Fatal("a recusa de aprovação expirada de outro tenant precisa existir")
	}
	if errExpirada.Error() != errInexistente.Error() {
		t.Fatalf(
			"ORÁCULO POR ORDEM DE CHECAGEM: recusa de aprovação EXPIRADA de outro tenant (%q) difere da recusa de INEXISTENTE (%q)",
			errExpirada.Error(), errInexistente.Error())
	}
}

// CONTROLE POSITIVO: o DONO da organização decide a própria aprovação e a decisão FUNCIONA — a checagem de tenant
// antecipada não pode bloquear quem é do tenant.
func TestDonoAindaDecideAPropriaAprovacao(t *testing.T) {
	root := t.TempDir()
	planner := fixedPlanner{steps: []Step{
		{
			ID: "step_1", Kind: "workspace.write", Title: "escrever a.txt", Risk: RiskWrite, RequiresApproval: true, State: StepPending,
			Input: map[string]any{"path": "a.txt", "content": "conteudo\n"},
		},
	}}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: planner, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:      "aprovar escrita",
		Capabilities:   []string{"workspace:read", "workspace:write"},
		OrganizationID: "org-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	decidida, err := runtime.DecideApprovalForActor(missao.ID, missao.Approvals[0].ID, true, "decisao do dono", "local", "org-a")
	if err != nil {
		t.Fatalf("o dono precisa conseguir decidir a PRÓPRIA aprovação: %v", err)
	}
	if decidida.Approvals[0].Status != ApprovalApproved {
		t.Fatalf("a aprovação do dono precisa ficar aprovada, veio %q", decidida.Approvals[0].Status)
	}
}
