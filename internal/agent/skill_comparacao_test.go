package agent

import (
	"errors"
	"testing"
)

// COMPARAR VERSÕES DE SKILL (E-648) — o §9 do Paperclip pede «comparar revisões».
//
// E o teste que interessa não é o do caso feliz: é o de ***não sugerir que uma versão antiga é mais
// confiável do que é*** (restaurar não promove confiança, E-631) e o de ***não acusar mudança quando
// só a ORDEM das permissões mudou***.

func storeComDuasVersoesDeSkill(t *testing.T) (*ContextStore, string) {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkillForOrganization("org-a", SkillManifest{
		ID: "skill-x", Version: "1.0", Description: "leitura de dados",
		Scopes: []string{"research:read"}, Tools: []string{"search"},
		Category: "dados", ContentSHA256: "hash-1", Signature: "sig-1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkillForOrganization("org-a", SkillManifest{
		ID: "skill-x", Version: "2.0", Description: "leitura e escrita de dados",
		// MESMAS permissões, ORDEM diferente: não é mudança.
		Scopes: []string{"research:write", "research:read"}, Tools: []string{"search", "write"},
		Category: "dados", ContentSHA256: "hash-2", Signature: "sig-2",
	}); err != nil {
		t.Fatal(err)
	}
	return store, "skill-x"
}

func TestCompararVersoesDeSkillMostraOQueMuda(t *testing.T) {
	store, id := storeComDuasVersoesDeSkill(t)
	diferenca, err := store.CompararVersoesDeSkill("org-a", id, "1.0", "2.0")
	if err != nil {
		t.Fatal(err)
	}
	if diferenca.De != "1.0" || diferenca.Para != "2.0" {
		t.Fatalf("par errado: %+v", diferenca)
	}
	campos := map[string]SkillMudanca{}
	for _, mudanca := range diferenca.Mudancas {
		campos[mudanca.Campo] = mudanca
	}
	if _, existe := campos["description"]; !existe {
		t.Fatalf("a descrição mudou e devia aparecer: %+v", diferenca.Mudancas)
	}
	if mudanca, existe := campos["scopes"]; !existe {
		t.Fatalf("os escopos mudaram e deviam aparecer: %+v", diferenca.Mudancas)
	} else if mudanca.De != "research:read" || mudanca.Para != "research:read,research:write" {
		t.Fatalf("escopos deviam vir ORDENADOS: %+v", mudanca)
	}
	// INTEGRIDADE DITA À PARTE, e não como «conteúdo de trabalho».
	if !diferenca.ConteudoMudou || !diferenca.AssinaturaMudou {
		t.Fatalf("hash e assinatura mudaram: %+v", diferenca)
	}
	// E `trusted`/`enabled` NÃO são reportados como mudança de conteúdo: são estado derivado, e
	// mostrá-los sugeriria que a versão antiga é mais confiável do que é.
	for _, campo := range []string{"trusted", "enabled"} {
		if _, existe := campos[campo]; existe {
			t.Fatalf("*** %q não pode ser reportado como mudança ***: %+v", campo, diferenca.Mudancas)
		}
	}
}

// A ORDEM das permissões não é uma mudança: a mesma permissão escrita noutra ordem é a mesma
// permissão. Acusar mudança aqui ensinaria quem revê a desconfiar de diferenças que não existem.
func TestCompararVersoesNaoAcusaMudancaPorOrdem(t *testing.T) {
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, versao := range []struct {
		v      string
		scopes []string
	}{{"1.0", []string{"b:read", "a:read"}}, {"2.0", []string{"a:read", "b:read"}}} {
		if err := store.RegisterSkillForOrganization("org-a", SkillManifest{
			ID: "skill-ordem", Version: versao.v, Description: "igual", Scopes: versao.scopes,
		}); err != nil {
			t.Fatal(err)
		}
	}
	diferenca, err := store.CompararVersoesDeSkill("org-a", "skill-ordem", "1.0", "2.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, mudanca := range diferenca.Mudancas {
		if mudanca.Campo == "scopes" || mudanca.Campo == "tools" {
			t.Fatalf("*** não houve mudança de conteúdo: só a ordem ***: %+v", mudanca)
		}
	}
}

// Versão inexistente e outra organização: a MESMA resposta «não existe» — distinguir seria uma
// pista sobre o que existe noutra organização.
func TestCompararVersoesInexistenteEOutraOrganizacao(t *testing.T) {
	store, id := storeComDuasVersoesDeSkill(t)
	if _, err := store.CompararVersoesDeSkill("org-a", id, "1.0", "9.9"); !errors.Is(err, ErrVersaoInexistente) {
		t.Fatalf("versão inexistente devia dar ErrVersaoInexistente: %v", err)
	}
	if _, err := store.CompararVersoesDeSkill("org-b", id, "1.0", "2.0"); !errors.Is(err, ErrVersaoInexistente) {
		t.Fatalf("*** outra organização não pode comparar a skill alheia ***: %v", err)
	}
	if _, err := store.CompararVersoesDeSkill("org-a", "skill-que-nao-existe", "1.0", "2.0"); !errors.Is(err, ErrVersaoInexistente) {
		t.Fatalf("skill inexistente devia dar ErrVersaoInexistente: %v", err)
	}
}

// Travessia de caminho continua recusada pela leitura partilhada — o refactor não abriu a porta.
func TestCompararVersoesRecusaTravessia(t *testing.T) {
	store, id := storeComDuasVersoesDeSkill(t)
	for _, versao := range []string{"../../../etc/passwd", "1.0/../../x", "..", "a/b"} {
		if _, err := store.CompararVersoesDeSkill("org-a", id, "1.0", versao); err == nil {
			t.Fatalf("*** travessia aceite na versão %q ***", versao)
		}
	}
}
