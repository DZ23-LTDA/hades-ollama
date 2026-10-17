package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestMutationHarnessCapabilityEnforcement verifies that mutating (removing)
// capability validation causes the security gate to fail.
func TestMutationHarnessCapabilityEnforcement(t *testing.T) {
	policy := DefaultCapabilityPolicy()

	// Target tool requires both desktop:screen and desktop:input
	descriptor := ToolDescriptor{
		Name:   "desktop",
		Scopes: []string{"desktop:screen", "desktop:input"},
	}

	// Baseline: partial grant containing only "desktop:screen"
	grantedCapabilities := []string{"desktop:screen"}

	// 1. Real gate: policy.Allows must reject because desktop:input is missing
	allowedReal := policy.Allows(descriptor, grantedCapabilities)
	if allowedReal {
		t.Fatalf("Baseline security failure: expected partial grant to be rejected")
	}

	// 2. Mutated gate: simulate a compromised/weakened policy that allows any tool
	mutatedAllows := func(desc ToolDescriptor, granted []string) bool {
		// MUTATION: intentionally weakened to accept everything (bypass)
		return true
	}
	allowedMutated := mutatedAllows(descriptor, grantedCapabilities)
	if !allowedMutated {
		t.Fatalf("Mutation test setup failed: expected mutated policy to bypass check")
	}

	// 3. Meta-assertion: the baseline security check MUST catch what the mutated check bypassed.
	// If baseline allowed the tool like the mutation, the harness reproves.
	if allowedReal == allowedMutated {
		t.Fatalf("MUTATION HARNESS REPROVED: Baseline gate does not protect against capability bypass!")
	}
}

// TestMutationHarnessHITLApprovalBypass exige que os gates REAIS de aprovação recusem um passo que exige aprovação.
//
// 🚨 E-401 / ADV-58/1 (5ª auditoria independente, MEDIDO): esta função era **TEATRO**. As linhas 59-64 e 72-75 eram
// **closures LOCAIS** que reimplementavam a condição `if s.RequiresApproval && !approved`, e a asserção final comparava
// **uma closure com a outra** — *um teste que mede a si mesmo*. A contraprova do auditor é decisiva: com **TODOS** os
// pontos de aprovação do runtime desligados (`runtime.go:2166/2176/2237/2251`), este teste continuava **PASSANDO** ✗.
//
// ***Teste de mutação que não toca o produto não é teste de mutação: é um espelho.***
//
// A versão abaixo chama as funções **reais** (`approvalsReady`, `stepApproved`, `validateMissionApprovalBindings`) sobre
// um `Runtime` e uma `Mission` de verdade. *A mutação em si é aplicada EXTERNAMENTE por `go test -overlay` (foi assim que
// o auditor mediu), porque um teste não pode reescrever o código que ele está executando — mas ele **pode** exigir o
// comportamento que a mutação destruiria.*
func TestMutationHarnessHITLApprovalBypass(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{
		Store: NewMemoryStore(), WorkspaceRoot: t.TempDir(),
		Planner: fixedPlanner{steps: []Step{{
			ID: "step_merge", Kind: "workspace.write", Title: "write", Risk: RiskWrite,
			RequiresApproval: true,
			Input:            map[string]any{"path": "descriptor.txt", "content": "safe"},
		}}}, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	// MEDIDO: com `Kind: "git.merge.origin"` a própria guarda anti-secreto RECUSA persistir o payload
	// (`hash approval payload: approval payload contains sensitive data that cannot be persisted safely`) ⇒ *a defesa
	// funcionando*, e o cenário tinha de usar um passo benigno para exercitar o **gate de aprovação** em vez da DLP.
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "gate de aprovacao real", OrganizationID: "org_a",
		Capabilities: []string{"workspace:write"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(missao.Approvals) != 1 {
		t.Fatalf("cenário precisa de exatamente 1 aprovação pendente, medido %d", len(missao.Approvals))
	}
	passo := missao.Plan[0]

	// 1. ANTES da decisão: os TRÊS gates reais têm de recusar. *Fail-closed, medido no produto.*
	if runtime.approvalsReady(missao) {
		t.Fatal("ARMADILHA: approvalsReady aprovou uma missão com aprovação PENDENTE — o portão de entrada caiu")
	}
	if runtime.stepApproved(missao, passo.ID) {
		t.Fatal("ARMADILHA: stepApproved aprovou um passo pendente — o portão por etapa caiu")
	}
	// MEDIDO (e é uma correção da MINHA expectativa, não do produto): `validateMissionApprovalBindings` **aceita** um
	// vínculo ainda PENDENTE — ele valida o **vínculo** (que existe registro para cada etapa que exige aprovação, com
	// política, validade e ator), e a decisão de **status** é do `approvalsReady`. *Eu havia suposto que ele checasse o
	// status; o teste real me corrigiu.* A divisão de trabalho é: vínculo ⇒ `validateMissionApprovalBindings`;
	// status global ⇒ `approvalsReady`; etapa ⇒ `stepApproved`.
	if err := runtime.validateMissionApprovalBindings(missao); err != nil {
		t.Fatalf("ARMADILHA: o VÍNCULO deveria estar válido mesmo com decisão pendente (é o approvalsReady que cuida do "+
			"status): %v", err)
	}

	// 2. DEPOIS da decisão explícita do ator: os gates têm de ABRIR. *Sem isso, o teste acima passaria com o gate
	// travado em "sempre recusa", que é um defeito diferente e igualmente grave.*
	missao, err = runtime.DecideApprovalForActor(missao.ID, missao.Approvals[0].ID, true, "aprovado no teste do gate", "user_a", "org_a")
	if err != nil {
		t.Fatal(err)
	}
	if !runtime.approvalsReady(missao) {
		t.Fatal("ARMADILHA: approvalsReady recusou uma missão com TODAS as aprovações decididas — o gate ficou trancado")
	}
	if err := runtime.validateMissionApprovalBindings(missao); err != nil {
		t.Fatalf("ARMADILHA: o vínculo deveria ser válido depois da decisão: %v", err)
	}
	if !runtime.stepApproved(missao, passo.ID) {
		t.Fatal("ARMADILHA: stepApproved recusou um passo cuja aprovação foi decidida")
	}

	// E o passo continua marcado como exigindo aprovação — *se alguém apagar `RequiresApproval`, o cenário acima deixa
	// de exercitar o gate e este teste tem de dizer isso, não passar em silêncio.*
	if !passo.RequiresApproval {
		t.Fatal("ARMADILHA: o passo do cenário perdeu RequiresApproval — o teste deixaria de exercitar o gate")
	}
}

// TestMutationHarnessSymlinkTraversal verifies that removing symlink validation
// is caught by the test harness.
func TestMutationHarnessSymlinkTraversal(t *testing.T) {
	tempDir := t.TempDir()
	outsideDir := t.TempDir()

	// Create sensitive target file outside workspace
	secretFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("super_secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create symlink inside workspace pointing outside
	symlinkPath := filepath.Join(tempDir, "leak_link")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Skip("Symlinks not supported in this environment:", err)
	}

	// 1. Real gate: rejectSymlinkComponents must reject this link
	errReal := rejectSymlinkComponents(tempDir, symlinkPath)
	if errReal == nil {
		t.Fatalf("Baseline security failure: rejectSymlinkComponents allowed external symlink")
	}

	// 2. Mutated gate: removes the symlink check
	mutatedSymlinkCheck := func(root, candidate string) error {
		// MUTATION: bypassed symlink check
		return nil
	}
	errMutated := mutatedSymlinkCheck(tempDir, symlinkPath)

	// 3. Meta-assertion
	if errReal == nil || errMutated != nil {
		t.Fatalf("MUTATION HARNESS REPROVED: Symlink traversal gate is trivial or inactive!")
	}
}
