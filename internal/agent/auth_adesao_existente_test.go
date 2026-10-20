package agent

import "testing"

// O DEFEITO MEDIDO: a rota de dev chamava `AddMembership(..., RoleViewer)` sempre que alguem
// voltava a entrar no tenant -- e o dono do tenant ficou com papel de viewer, perdendo o
// direito de escrever ("role viewer cannot perform execute").
func TestAdesaoExistenteDevolveOPapelReal(t *testing.T) {
	store, err := NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewAuthStore: %v", err)
	}
	dono, err := store.CreateUser("dono@local", "Dono")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	organizacao, adesaoDono, err := store.CreateOrganization("Org", dono)
	if err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}

	encontrada, ok := store.AdesaoExistente(dono.ID, organizacao.ID)
	if !ok {
		t.Fatal("o dono devia ter adesao")
	}
	if encontrada.Role != adesaoDono.Role {
		t.Fatalf("papel mudou: %q -> %q", adesaoDono.Role, encontrada.Role)
	}

	outro, err := store.CreateUser("outro@local", "Outro")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, ok := store.AdesaoExistente(outro.ID, organizacao.ID); ok {
		t.Fatal("utilizador sem adesao nao pode aparecer como membro")
	}
}
