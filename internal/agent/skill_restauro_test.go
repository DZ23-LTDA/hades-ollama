package agent

import (
	"testing"
)

// RESTAURAR É RECUPERAR CONTEÚDO, NÃO CONFIANÇA (E-631).
//
// Uma versão arquivada que alguma vez tenha sido promovida voltaria como NÃO confiável. Se o
// restauro herdasse `Trusted`, seria uma ESCALADA SILENCIOSA: bastava ressuscitar uma versão
// antiga para contornar a política que governa `PromoteSkillTrustedForOrganization`.
func TestRestaurarVersaoDevolveConteudoSemHerdarConfianca(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkill(SkillManifest{ID: "skill-x", Version: "1.0", Description: "v1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkill(SkillManifest{ID: "skill-x", Version: "2.0", Description: "v2"}); err != nil {
		t.Fatal(err)
	}

	// (1) O HISTÓRICO LISTA O QUE FOI ARQUIVADO.
	versoes, err := store.SkillVersionsForOrganization("", "skill-x")
	if err != nil {
		t.Fatal(err)
	}
	if len(versoes) != 1 || versoes[0] != "1.0" {
		t.Fatalf("o histórico devia listar a 1.0: %v", versoes)
	}

	// (2) RESTAURAR traz o conteúdo de volta — e fá-lo ficar ACTIVO.
	restaurado, err := store.RestoreSkillVersionForOrganization("", "skill-x", "1.0")
	if err != nil {
		t.Fatal(err)
	}
	if restaurado.Version != "1.0" || restaurado.Description != "v1" {
		t.Fatalf("restaurou conteúdo errado: %+v", restaurado)
	}
	activas := store.Skills()
	if len(activas) != 1 || activas[0].Version != "1.0" {
		t.Fatalf("a versão restaurada devia ficar activa: %+v", activas)
	}

	// (3) ***E NÃO HERDA CONFIANÇA.*** É este o ponto do teste.
	if activas[0].Trusted {
		t.Fatal("*** restaurar NÃO pode herdar confiança — seria escalada silenciosa ***")
	}

	// (4) A TROCA É SIMÉTRICA: a 2.0, que saiu, passou a ser histórico.
	depois, err := store.SkillVersionsForOrganization("", "skill-x")
	if err != nil {
		t.Fatal(err)
	}
	if len(depois) != 2 {
		t.Fatalf("a versão substituída devia ir para o histórico: %v", depois)
	}
}

// A VERSÃO E O ID ENTRAM NUM CAMINHO DE FICHEIRO: travessia tem de ser recusada.
// (O `validateSkillManifest` valida o ID para travessia, mas NÃO a versão — só o comprimento.)
func TestRestaurarVersaoRecusaTravessiaDeCaminho(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, versao := range []string{"../../etc/passwd", "..", ".", "a/b", `a\b`, "", "com espaço"} {
		if _, err := store.RestoreSkillVersionForOrganization("", "skill-x", versao); err == nil {
			t.Fatalf("versão %q devia ser recusada", versao)
		}
	}
	for _, id := range []string{"../fora", "a/b", `a\b`, ""} {
		if _, err := store.RestoreSkillVersionForOrganization("", id, "1.0"); err == nil {
			t.Fatalf("id %q devia ser recusado", id)
		}
	}
	// E o mesmo para a listagem.
	if _, err := store.SkillVersionsForOrganization("", "../fora"); err == nil {
		t.Fatal("a listagem devia recusar id com travessia")
	}
}

// Restaurar uma versão que não existe é um ERRO, e não um restauro silencioso de nada.
func TestRestaurarVersaoInexistenteEhErro(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkill(SkillManifest{ID: "skill-x", Version: "1.0", Description: "v1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RestoreSkillVersionForOrganization("", "skill-x", "9.9"); err == nil {
		t.Fatal("restaurar versão inexistente devia falhar")
	}
}
