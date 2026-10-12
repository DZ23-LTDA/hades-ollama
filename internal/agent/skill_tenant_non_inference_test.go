package agent

import (
	"testing"
)

// NÃO INFERÊNCIA NA FAMÍLIA DE SKILLS (E-195, continuação do B-17).
//
// O scanner estendido (E-195) passou a modelar busca por MAPA — a lacuna que fazia o instrumento achar 1 de 7
// instâncias. Ele apontou três métodos de skill que devolviam `skill %q is not registered` para "não existe" e
// `ErrPluginOrganizationScope` para "é de outro tenant": recusas diferentes para a mesma pergunta do tenant.
//
// Este teste compara as duas recusas. Se voltarem a divergir, ele falha.
func TestSkillTenantOperationsDoNotRevealExistence(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	const intrusa = "org-intrusa"
	const inexistente = "skill_que_nao_existe"

	if err := store.RegisterSkillForOrganization("org-a", SkillManifest{ID: "skill_da_org_a", Version: "1.0.0", Description: "skill de teste", Enabled: true}); err != nil {
		t.Fatalf("registrar skill da org-a: %v", err)
	}

	casos := []struct {
		nome     string
		executar func(id string) error
	}{
		{"SetSkillEnabledForOrganization", func(id string) error {
			return store.SetSkillEnabledForOrganization(intrusa, id, false)
		}},
		{"PromoteSkillTrustedForOrganization", func(id string) error {
			return store.PromoteSkillTrustedForOrganization(intrusa, id, CapabilityPolicy{})
		}},
		{"RemoveSkillForOrganization", func(id string) error {
			return store.RemoveSkillForOrganization(intrusa, id)
		}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			errOutro := caso.executar("skill_da_org_a")
			errInexistente := caso.executar(inexistente)
			if errOutro == nil || errInexistente == nil {
				t.Fatalf("%s: as duas negações precisam existir (outro=%v, inexistente=%v)", caso.nome, errOutro, errInexistente)
			}
			if errOutro.Error() != errInexistente.Error() {
				t.Fatalf(
					"%s: ORÁCULO DE ENUMERAÇÃO — recusa de outro tenant (%q) difere da recusa de inexistente (%q)",
					caso.nome, errOutro.Error(), errInexistente.Error())
			}
		})
	}
}

// CONTROLE POSITIVO: o dono continua operando a própria skill.
func TestOwnerStillOperatesOwnSkills(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkillForOrganization("org-a", SkillManifest{ID: "skill_da_org_a", Version: "1.0.0", Description: "skill de teste", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetSkillEnabledForOrganization("org-a", "skill_da_org_a", false); err != nil {
		t.Fatalf("o dono precisa desativar a própria skill: %v", err)
	}
	if err := store.RemoveSkillForOrganization("org-a", "skill_da_org_a"); err != nil {
		t.Fatalf("o dono precisa remover a própria skill: %v", err)
	}
}
