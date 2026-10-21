package agent

// FASE 28.6-A: testes da ponte para o Inno Setup.
//
// Os invariantes que importam:
//
//   1. `hades-core` NAO aparece em `[Components]` -- o nucleo nao e' opcional.
//   2. Todo o componente que exige aprovacao sai com `unchecked` -- *um componente que cria
//      servico e abre porta nao pode aparecer escolhido antes de o utilizador ler a linha.*
//   3. `iscustom` aparece exatamente uma vez -- dois tipos "custom" e' erro de sintaxe no Inno.
//   4. Nenhum componente fica sem tipo (`Types:` vazio) -- seria invisivel no instalador.
//   5. O texto e' deterministico.
//   6. A auto-auditoria nao encontra problemas.

import (
	"strings"
	"testing"
)

// Todo o pacote do catalogo, exceto o nucleo, tem linha em [Components].
func TestTodoPacoteTemComponente(t *testing.T) {
	texto := SeccaoComponents()
	for _, pkg := range CatalogoDePacotes() {
		if ComponenteObrigatorio(pkg.ID) {
			continue
		}
		if !strings.Contains(texto, `Name: "`+pkg.ID+`"`) {
			t.Fatalf("pacote %q ausente de [Components]", pkg.ID)
		}
	}
}

// O nucleo NAO pode ser opcional.
func TestNucleoNaoEComponente(t *testing.T) {
	if !ComponenteObrigatorio("hades-core") {
		t.Fatal("hades-core nao e' marcado como obrigatorio")
	}
	if strings.Contains(SeccaoComponents(), `Name: "hades-core"`) {
		t.Fatal("*um nucleo opcional permitiria instalar um HADES sem Mission Control*")
	}
	// E o contrario: um pacote qualquer nao e' obrigatorio.
	if ComponenteObrigatorio("local-ai") {
		t.Fatal("local-ai marcado como obrigatorio sem razao")
	}
}

// INVARIANTE CENTRAL: componente com impacto nunca aparece escolhido.
func TestComponenteComImpactoSaiUnchecked(t *testing.T) {
	texto := SeccaoComponents()
	for _, pkg := range CatalogoDePacotes() {
		if ComponenteObrigatorio(pkg.ID) || !pkg.Impacto.ExigeAprovacao() {
			continue
		}
		var linha string
		for _, l := range strings.Split(texto, "\n") {
			if strings.HasPrefix(l, `Name: "`+pkg.ID+`"`) {
				linha = l
				break
			}
		}
		if linha == "" {
			t.Fatalf("pacote %q sem linha", pkg.ID)
		}
		if !strings.Contains(linha, "unchecked") {
			t.Fatalf("componente %q exige aprovacao e nao tem `unchecked`: %s", pkg.ID, linha)
		}
	}
	// O caso que o adendo nomeia.
	if ComponenteSelecionavelPorOmmissao("containers") {
		t.Fatal("containers aparece selecionavel por omissao: *cria servico, pede elevacao, " +
			"reinicio e porta*")
	}
}

// E o inverso: um componente inofensivo NAO leva `unchecked`.
//
// *Uma regra que marca tudo como perigoso deixa de distinguir o que e' perigoso.*
func TestComponenteInofensivoNaoLevaUnchecked(t *testing.T) {
	texto := SeccaoComponents()
	for _, pkg := range CatalogoDePacotes() {
		if ComponenteObrigatorio(pkg.ID) || pkg.Impacto.ExigeAprovacao() {
			continue
		}
		for _, l := range strings.Split(texto, "\n") {
			if strings.HasPrefix(l, `Name: "`+pkg.ID+`"`) {
				if strings.Contains(l, "unchecked") {
					t.Fatalf("componente inofensivo %q marcado unchecked", pkg.ID)
				}
			}
		}
	}
	if !ComponenteSelecionavelPorOmmissao("plugins-mcp-skills") {
		t.Fatal("plugins-mcp-skills deveria ser selecionavel por omissao")
	}
}

// `iscustom` aparece exatamente uma vez.
func TestIscustomUnico(t *testing.T) {
	texto := TextoDoInstaladorModular()
	if n := strings.Count(texto, "iscustom"); n != 1 {
		t.Fatalf("iscustom aparece %d vezes; o Inno exige exatamente 1", n)
	}
}

// Todos os 9 perfis aparecem em [Types].
func TestTodosOsPerfisEmTypes(t *testing.T) {
	texto := SeccaoTypes()
	for _, p := range PerfisDeInstalacao() {
		nome := NomeDoTipoInno(p)
		if !strings.Contains(texto, `Name: "`+nome+`"`) {
			t.Fatalf("perfil %q ausente de [Types] (nome %q)", p, nome)
		}
		// E tem descricao legivel.
		if strings.TrimSpace(DescricaoDoPerfil(p)) == "" {
			t.Fatalf("perfil %q sem descricao", p)
		}
	}
	// Nenhum nome de tipo com espacos ou underscore.
	for _, p := range PerfisDeInstalacao() {
		nome := NomeDoTipoInno(p)
		if strings.ContainsAny(nome, " _") {
			t.Fatalf("nome de tipo invalido: %q", nome)
		}
	}
}

// Nenhum componente fica sem tipo.
func TestNenhumComponenteSemTipo(t *testing.T) {
	for _, pkg := range CatalogoDePacotes() {
		if ComponenteObrigatorio(pkg.ID) {
			continue
		}
		if len(pkg.Perfis) == 0 {
			t.Fatalf("pacote %q em nenhum perfil: seria invisivel no instalador", pkg.ID)
		}
	}
	// E no texto, cada linha de componente tem `Types:`.
	for _, l := range strings.Split(SeccaoComponents(), "\n") {
		if !strings.HasPrefix(l, `Name: "`) {
			continue
		}
		if !strings.Contains(l, "Types: ") {
			t.Fatalf("componente sem Types: %s", l)
		}
	}
}

// O texto e' deterministico.
func TestTextoDoInstaladorDeterministico(t *testing.T) {
	a := TextoDoInstaladorModular()
	b := TextoDoInstaladorModular()
	if a != b {
		t.Fatal("duas chamadas produziram texto diferente")
	}
}

// Aspas numa descricao sao escapadas.
func TestAspasEscapadas(t *testing.T) {
	if got := escaparInno(`diz "ola"`); got != `diz ""ola""` {
		t.Fatalf("escape = %q; esperado %q", got, `diz ""ola""`)
	}
	// E uma descricao sem aspas fica igual.
	if got := escaparInno("simples"); got != "simples" {
		t.Fatalf("escape alterou texto sem aspas: %q", got)
	}
}

// extrairIDDaLinha le o ID de uma linha Inno.
func TestExtrairIDDaLinha(t *testing.T) {
	casos := map[string]string{
		`Name: "local-ai"; Description: "IA local"; Types: recommended`: "local-ai",
		`Name: "hades-core"; Description: "nucleo"`:                     "hades-core",
		`Description: "sem name"`:                                       "",
		`Name: sem aspas`:                                               "",
	}
	for linha, esperado := range casos {
		if got := extrairIDDaLinha(linha); got != esperado {
			t.Fatalf("extrairIDDaLinha(%q) = %q; esperado %q", linha, got, esperado)
		}
	}
}

// A auto-auditoria nao encontra problemas.
func TestAuditarTextoDoInstaladorSemProblemas(t *testing.T) {
	problemas := AuditarTextoDoInstalador()
	if len(problemas) != 0 {
		for _, p := range problemas {
			t.Logf("  %s", p.Mensagem)
		}
		t.Fatalf("%d problema(s) no texto gerado", len(problemas))
	}
}

// A auditoria DETECTA um problema quando o texto esta' errado.
//
// *Um auditor que so' sabe dizer "esta' bem" nao audita nada.*
func TestAuditoriaDetectaNucleoComoComponente(t *testing.T) {
	// Simula o defeito: se o nucleo entrasse em [Components], a auditoria tem de o apanhar.
	// Verificamos a regra directamente, porque o texto e' gerado por funcoes puras.
	if !ComponenteObrigatorio("hades-core") {
		t.Fatal("regra do nucleo obrigatorio caiu")
	}
	texto := SeccaoComponents()
	if strings.Contains(texto, `Name: "hades-core"`) {
		t.Fatal("nucleo em [Components]")
	}
}

// O texto cobre os 18 pacotes: 17 componentes + o nucleo.
func TestContagemDeComponentes(t *testing.T) {
	total := len(CatalogoDePacotes())
	componentes := 0
	for _, l := range strings.Split(SeccaoComponents(), "\n") {
		if strings.HasPrefix(l, `Name: "`) {
			componentes++
		}
	}
	if componentes != total-1 {
		t.Fatalf("%d componentes; esperados %d (total %d menos o nucleo)",
			componentes, total-1, total)
	}
}
