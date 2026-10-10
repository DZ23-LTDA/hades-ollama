package agent

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
)

func skillActivationStore(t *testing.T, manifests ...SkillManifest) *ContextStore {
	t.Helper()
	return skillActivationStoreAt(t, t.TempDir(), manifests...)
}

// skillActivationStoreAt permite reabrir o MESMO root para provar persistência.
func skillActivationStoreAt(t *testing.T, root string, manifests ...SkillManifest) *ContextStore {
	t.Helper()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	// Trust é decidido pelo SERVIDOR: assinatura verificada + chave autorizada.
	// O fixture usa o mesmo caminho da produção (assinar e promover), em vez de
	// ligar o campo Trusted à mão.
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultCapabilityPolicy().WithTrustedSkillKey("release-key", publicKey)
	for _, manifest := range manifests {
		manifest.OrganizationID = "org-skills"
		wantTrusted := manifest.Trusted
		// O booleano não faz parte da assinatura: zerar antes de assinar deixa o
		// manifesto honesto e prova que a confiança vem da verificação.
		manifest.Trusted = false
		signed, err := SignSkillManifest(manifest, privateKey, "release-key")
		if err != nil {
			t.Fatalf("sign skill %s: %v", manifest.ID, err)
		}
		signed.OrganizationID = "org-skills"
		if err := store.RegisterSkillForOrganization("org-skills", signed); err != nil {
			t.Fatalf("register skill %s: %v", manifest.ID, err)
		}
		if wantTrusted {
			if err := store.PromoteSkillTrustedForOrganization("org-skills", manifest.ID, policy); err != nil {
				t.Fatalf("promote skill %s: %v", manifest.ID, err)
			}
		}
	}
	return store
}

func trustedSkill(id, category string, budget int) SkillManifest {
	return SkillManifest{
		ID:          id,
		Version:     "1.0.0",
		Description: "skill de teste",
		Category:    category,
		Trusted:     true,
		Enabled:     true,
		Scopes:      []string{"workspace:read"},
		TokenBudget: budget,
	}
}

func TestSkillActivationRequiresTrustedSkill(t *testing.T) {
	untrusted := trustedSkill("skill-nao-confiavel", "dados", 100)
	untrusted.Trusted = false
	store := skillActivationStore(t, untrusted)

	_, err := store.ActivateSkillForOrganization("org-skills", SkillActivation{SkillID: untrusted.ID})
	if !errors.Is(err, ErrSkillNotTrusted) {
		t.Fatalf("skill não confiável deveria recusar ativação, got %v", err)
	}
	if activations := store.SkillActivationsForOrganization("org-skills"); len(activations) != 0 {
		t.Fatalf("nada deveria ter sido ativado: %+v", activations)
	}
}

func TestSkillActivationIsScopedByProjectAgentAndModel(t *testing.T) {
	store := skillActivationStore(t,
		trustedSkill("skill-global", "geral", 10),
		trustedSkill("skill-projeto", "projeto", 20),
		trustedSkill("skill-modelo", "modelo", 30),
	)
	if _, err := store.ActivateSkillForOrganization("org-skills", SkillActivation{SkillID: "skill-global"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ActivateSkillForOrganization("org-skills", SkillActivation{SkillID: "skill-projeto", ProjectID: "proj-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ActivateSkillForOrganization("org-skills", SkillActivation{SkillID: "skill-modelo", Model: "qwen2.5:7b"}); err != nil {
		t.Fatal(err)
	}

	// Contexto do projeto 1 com o modelo declarado: as três se aplicam.
	selection, err := store.SelectSkillsForMission("org-skills", SkillContext{ProjectID: "proj-1", Model: "qwen2.5:7b"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Selected) != 3 {
		t.Fatalf("esperava 3 skills ativas, veio %+v", selection.Selected)
	}

	// Outro projeto e outro modelo: só a global (curinga) permanece.
	selection, err = store.SelectSkillsForMission("org-skills", SkillContext{ProjectID: "proj-2", Model: "llama3.1:8b"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Selected) != 1 || selection.Selected[0].ID != "skill-global" {
		t.Fatalf("esperava apenas a skill global, veio %+v", selection.Selected)
	}
}

func TestSkillSelectionRespectsTokenBudgetAndReportsSkips(t *testing.T) {
	store := skillActivationStore(t,
		trustedSkill("skill-a-curta", "a", 40),
		trustedSkill("skill-b-longa", "b", 100),
	)
	for _, id := range []string{"skill-a-curta", "skill-b-longa"} {
		if _, err := store.ActivateSkillForOrganization("org-skills", SkillActivation{SkillID: id}); err != nil {
			t.Fatal(err)
		}
	}

	// Orçamento cabe só a primeira (ordem determinística por categoria).
	selection, err := store.SelectSkillsForMission("org-skills", SkillContext{}, 50)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.BudgetDefined || selection.UsedTokens != 40 {
		t.Fatalf("orçamento usado = %d (definido=%v)", selection.UsedTokens, selection.BudgetDefined)
	}
	if len(selection.Selected) != 1 || selection.Selected[0].ID != "skill-a-curta" {
		t.Fatalf("selecionadas = %+v", selection.Selected)
	}
	if len(selection.Skipped) != 1 || selection.Skipped[0].ID != "skill-b-longa" {
		t.Fatalf("a exclusão precisa aparecer com motivo: %+v", selection.Skipped)
	}
	if selection.Skipped[0].Reason == "" {
		t.Fatal("motivo da exclusão não pode ser vazio")
	}

	// Sem orçamento declarado, tudo entra (e o usado não é inventado).
	selection, err = store.SelectSkillsForMission("org-skills", SkillContext{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if selection.BudgetDefined || len(selection.Selected) != 2 || len(selection.Skipped) != 0 {
		t.Fatalf("sem orçamento: %+v", selection)
	}
}

func TestSkillSelectionSkipsDisabledAndKeepsProvenance(t *testing.T) {
	store := skillActivationStore(t, trustedSkill("skill-ok", "a", 5), trustedSkill("skill-off", "b", 5))
	for _, id := range []string{"skill-ok", "skill-off"} {
		if _, err := store.ActivateSkillForOrganization("org-skills", SkillActivation{SkillID: id}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SetSkillEnabledForOrganization("org-skills", "skill-off", false); err != nil {
		t.Fatal(err)
	}

	selection, err := store.SelectSkillsForMission("org-skills", SkillContext{ProjectID: "proj-9"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(selection.Selected) != 1 || selection.Selected[0].ID != "skill-ok" {
		t.Fatalf("skill desabilitada não pode entrar: %+v", selection.Selected)
	}
	if len(selection.Skipped) != 1 || selection.Skipped[0].ID != "skill-off" {
		t.Fatalf("a skill desabilitada precisa aparecer como excluída: %+v", selection.Skipped)
	}
	// Proveniência: a ativação que sustenta a escolha vem junto.
	if selection.Selected[0].Activation.SkillID != "skill-ok" {
		t.Fatalf("proveniência ausente: %+v", selection.Selected[0])
	}
}

func TestSkillActivationPersistsAndCanBeDeactivated(t *testing.T) {
	root := t.TempDir()
	store := skillActivationStoreAt(t, root, trustedSkill("skill-persistente", "dados", 7))
	activation, err := store.ActivateSkillForOrganization("org-skills", SkillActivation{SkillID: "skill-persistente", ProjectID: "proj-1"})
	if err != nil {
		t.Fatal(err)
	}

	// Reabrir o store precisa trazer a ativação do disco.
	reopened, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	persisted := reopened.SkillActivationsForOrganization("org-skills")
	if len(persisted) != 1 || persisted[0].Key() != activation.Key() {
		t.Fatalf("ativação não persistiu: %+v", persisted)
	}

	if err := reopened.DeactivateSkillForOrganization("org-skills", activation.Key()); err != nil {
		t.Fatal(err)
	}
	if len(reopened.SkillActivationsForOrganization("org-skills")) != 0 {
		t.Fatal("desativar deveria remover a ativação")
	}
	if err := reopened.DeactivateSkillForOrganization("org-skills", activation.Key()); !errors.Is(err, ErrSkillActivationNotFound) {
		t.Fatalf("desativar duas vezes deveria devolver ErrSkillActivationNotFound, got %v", err)
	}
}

func TestSkillActivationRejectsUnknownSkillAndEmptyOrganization(t *testing.T) {
	store := skillActivationStore(t, trustedSkill("skill-existente", "a", 1))
	if _, err := store.ActivateSkillForOrganization("org-skills", SkillActivation{SkillID: "skill-fantasma"}); !errors.Is(err, ErrSkillNotFound) {
		t.Fatalf("skill inexistente deveria recusar, got %v", err)
	}
	if _, err := store.ActivateSkillForOrganization("", SkillActivation{SkillID: "skill-existente"}); !errors.Is(err, ErrSkillActivationInvalid) {
		t.Fatalf("organização vazia deveria recusar, got %v", err)
	}
	if _, err := store.SelectSkillsForMission("", SkillContext{}, 0); !errors.Is(err, ErrSkillActivationInvalid) {
		t.Fatalf("seleção sem organização deveria recusar, got %v", err)
	}
}
