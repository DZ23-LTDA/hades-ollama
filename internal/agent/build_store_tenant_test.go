package agent

import (
	"errors"
	"testing"
)

// O DEFEITO MEDIDO (audit_tenant_guards): cinco handlers de build liam o registo SO' pelo id, e o
// `RegistoDeBuild` nao tinha dono. Qualquer sessao autenticada que adivinhasse um id lia estado,
// saida e artefactos de um build de OUTRA organizacao.
func TestRegistoDeBuildTemDonoEIsolaLeitura(t *testing.T) {
	s := NovoBuildStore()
	id, err := s.RegistarDoTenant("alvo-a", "raiz-a", "destino-a", "org-1")
	if err != nil {
		t.Fatalf("registar: %v", err)
	}

	// POSITIVO: o dono le.
	r, err := s.ObterDoTenant(id, "org-1")
	if err != nil {
		t.Fatalf("o dono devia ler o proprio build: %v", err)
	}
	if r.Tenant != "org-1" {
		t.Fatalf("o registo devia guardar o tenant, veio %q", r.Tenant)
	}

	// NEGATIVO: outro tenant recebe o MESMO erro de um id inexistente (sem oracle de existencia).
	_, errOutro := s.ObterDoTenant(id, "org-2")
	_, errInexistente := s.ObterDoTenant("nao-existe", "org-2")
	if !errors.Is(errOutro, ErrBuildDesconhecido) {
		t.Fatalf("outro tenant devia receber ErrBuildDesconhecido, veio %v", errOutro)
	}
	if errOutro.Error() != errInexistente.Error() {
		t.Fatalf("a negacao de outro tenant (%q) tem de ser IDENTICA a de id inexistente (%q)",
			errOutro.Error(), errInexistente.Error())
	}

	// NEGATIVO: cancelar build alheio nao muda nada.
	if err := s.CancelarDoTenant(id, "org-2"); !errors.Is(err, ErrBuildDesconhecido) {
		t.Fatalf("cancelar build alheio devia dar ErrBuildDesconhecido, veio %v", err)
	}
	depois, err := s.ObterDoTenant(id, "org-1")
	if err != nil {
		t.Fatalf("obter depois: %v", err)
	}
	if depois.Estado != r.Estado {
		t.Fatalf("o estado mudou apos cancelamento negado: %q -> %q", r.Estado, depois.Estado)
	}
}

func TestListagemDeBuildsIsoladaPorTenant(t *testing.T) {
	s := NovoBuildStore()
	if _, err := s.RegistarDoTenant("a1", "r1", "d1", "org-1"); err != nil {
		t.Fatalf("registar org-1: %v", err)
	}
	if _, err := s.RegistarDoTenant("a2", "r2", "d2", "org-2"); err != nil {
		t.Fatalf("registar org-2: %v", err)
	}
	if n := len(s.ListarDoTenant("org-1")); n != 1 {
		t.Fatalf("org-1 devia ver 1 build, viu %d", n)
	}
	if n := len(s.ListarDoTenant("org-2")); n != 1 {
		t.Fatalf("org-2 devia ver 1 build, viu %d", n)
	}
	if n := len(s.ListarDoTenant("org-3")); n != 0 {
		t.Fatalf("org-3 nao devia ver builds, viu %d", n)
	}
	// Um registo sem dono (caminho antigo) nao aparece para ninguem com tenant.
	if _, err := s.registarSemDono("a3", "r3", "d3"); err != nil {
		t.Fatalf("registar sem dono: %v", err)
	}
	if n := len(s.ListarDoTenant("")); n != 1 {
		t.Fatalf("o tenant vazio devia ver so' o build sem dono, viu %d", n)
	}
}
