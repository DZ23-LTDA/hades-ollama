package agent

import (
	"errors"
	"strings"
	"testing"
)

// E-972: OS TESTES DO FORMATO `hades-plugin.json`.
//
// **O que cada teste discrimina**: cada um falharia se a garantia correspondente fosse
// removida. *Nao ha' aqui um teste que passe porque o JSON e' bonito.*

func politicaParaFormato() CapabilityPolicy { return DefaultCapabilityPolicy() }

// TestFormatoAceitaManifestoMinimo e' o controlo: *sem ele, os testes negativos abaixo
// poderiam passar por o parser recusar TUDO.*
//
// **O controlo e' obrigatorio**: um teste que so' prova recusas nao distingue «valida
// bem» de «nao valida nada».
func TestFormatoAceitaManifestoMinimo(t *testing.T) {
	data := []byte(`{
	  "$schema": "https://hades.dev/schemas/hades-plugin-1.0.json",
	  "format_version": "1.0",
	  "id": "estoque-widget",
	  "version": "1.0.0",
	  "name": "Widget de estoque",
	  "kind": "bundle"
	}`)
	manifest, err := ParseHadesPluginManifest(data, politicaParaFormato())
	if err != nil {
		t.Fatalf("o manifesto minimo tem de ser aceito, veio: %v", err)
	}
	if manifest.ID != "estoque-widget" || manifest.FormatVersion != "1.0" {
		t.Fatalf("o manifesto aceito perdeu campos: %+v", manifest)
	}
}

// TestFormatoAceitaSkillsEInterface e' o caso COMPLETO: *o que o formato existe para
// suportar.*
func TestFormatoAceitaSkillsEInterface(t *testing.T) {
	data := []byte(`{
	  "format_version": "1.0",
	  "id": "agent-skills-import",
	  "version": "0.6.13",
	  "name": "Agent Skills (importado)",
	  "kind": "skill",
	  "license": "MIT",
	  "scopes": ["workspace:read"],
	  "content_sha256": "` + strings.Repeat("a", 64) + `",
	  "skills": [
	    {"name": "test-driven-development", "path": "skills/test-driven-development/SKILL.md"},
	    {"name": "code-review-and-quality", "path": "skills/code-review-and-quality/SKILL.md"}
	  ],
	  "interface": {
	    "display_name": "Agent Skills",
	    "category": "Productivity",
	    "capabilities": ["Read", "Write"],
	    "default_prompt": ["Escolha o fluxo de engenharia adequado"]
	  }
	}`)
	manifest, err := ParseHadesPluginManifest(data, politicaParaFormato())
	if err != nil {
		t.Fatalf("o manifesto completo tem de ser aceito, veio: %v", err)
	}
	if len(manifest.Skills) != 2 {
		t.Fatalf("esperadas 2 skills, vieram %d", len(manifest.Skills))
	}
	if manifest.Interface == nil || manifest.Interface.DisplayName != "Agent Skills" {
		t.Fatalf("a interface nao foi lida: %+v", manifest.Interface)
	}
	if manifest.License != "MIT" {
		t.Fatalf("a licenca tem de sobreviver ao parse: %q", manifest.License)
	}
}

// TestFormatoRecusaVersaoDeFormatoDesconhecida e' a evolucao do schema: *uma versao que
// este runtime nao conhece nao pode ser aceita por omissao.*
func TestFormatoRecusaVersaoDeFormatoDesconhecida(t *testing.T) {
	data := []byte(`{"format_version":"9.9","id":"x","version":"1","name":"X","kind":"bundle"}`)
	_, err := ParseHadesPluginManifest(data, politicaParaFormato())
	if !errors.Is(err, ErrHadesPluginVersion) {
		t.Fatalf("versao desconhecida tem de dar ErrHadesPluginVersion, veio: %v", err)
	}
}

// TestFormatoRecusaCampoDesconhecido e' o DISCRIMINADOR do DisallowUnknownFields.
//
// *Se alguem remover o `DisallowUnknownFields`, este teste falha* — e o efeito seria
// aceitar manifestos cujo efeito o runtime nao sabe prever.
func TestFormatoRecusaCampoDesconhecido(t *testing.T) {
	data := []byte(`{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"bundle","shell":"rm -rf /"}`)
	_, err := ParseHadesPluginManifest(data, politicaParaFormato())
	if !errors.Is(err, ErrHadesPluginFormat) {
		t.Fatalf("campo desconhecido tem de ser recusado, veio: %v", err)
	}
}

// TestFormatoRecusaKindInvalido prova que o formato NAO afrouxa o nucleo: *os cinco kinds
// continuam a ser cinco.*
func TestFormatoRecusaKindInvalido(t *testing.T) {
	data := []byte(`{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"super-poder"}`)
	_, err := ParseHadesPluginManifest(data, politicaParaFormato())
	if err == nil {
		t.Fatal("kind fora dos cinco tem de ser recusado")
	}
}

// TestFormatoRecusaIdComSeparador prova que o id continua a ser slug: *a regra do
// `validatePluginManifest` (`:188`) tem de valer tambem pelo formato novo.*
//
// **E-972, e um erro MEU que este teste apanhou**: a primeira versao usava `"a\b"` — e em
// **Go** o `\b` e' o caractere BACKSPACE (`\x08`), nao `\` + `b`. *O id que chegava ao
// validador nao continha separador nenhum, e o teste passava a nao testar o que dizia.*
// A cura e' `"a\\\\b"` em JSON, que descodifica para `a\b`.
func TestFormatoRecusaIdComSeparador(t *testing.T) {
	casos := map[string]string{
		"barra":           `"a/b"`,
		"dois-pontos":     `"a:b"`,
		"espaco":          `"a b"`,
		"barra-invertida": `"a\\b"`, // JSON: descodifica para a\b
	}
	for nome, idJSON := range casos {
		data := []byte(`{"format_version":"1.0","id":` + idJSON + `,"version":"1","name":"X","kind":"bundle"}`)
		if _, err := ParseHadesPluginManifest(data, politicaParaFormato()); err == nil {
			t.Fatalf("id com %s (%s) tem de ser recusado", nome, idJSON)
		}
	}
}

// TestFormatoRecusaSkillComTravessia e' a defesa contra travessia, no manifesto.
//
// **Onde ela actua importa**: o `preparePluginWorkspace` (`plugin_runtime.go:212`) protege
// na EXECUCAO; aqui protege-se na INSTALACAO. *Recusar cedo e' melhor que recusar tarde.*
func TestFormatoRecusaSkillComTravessia(t *testing.T) {
	for _, path := range []string{"../../etc/passwd", "/etc/passwd", `..\..\windows\system32`, "a/../../b"} {
		data := []byte(`{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"skill","skills":[{"name":"s","path":"` + strings.ReplaceAll(path, `\`, `\\`) + `"}]}`)
		if _, err := ParseHadesPluginManifest(data, politicaParaFormato()); err == nil {
			t.Fatalf("caminho %q tem de ser recusado", path)
		}
	}
}

// TestFormatoRecusaSkillDuplicada e' a coerencia do catalogo: *duas entradas com o mesmo
// nome sao uma ambiguidade, nao uma repeticao.*
func TestFormatoRecusaSkillDuplicada(t *testing.T) {
	data := []byte(`{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"skill","skills":[{"name":"s","path":"a.md"},{"name":"s","path":"b.md"}]}`)
	_, err := ParseHadesPluginManifest(data, politicaParaFormato())
	if !errors.Is(err, ErrHadesPluginSkillPath) {
		t.Fatalf("skill duplicada tem de ser recusada, veio: %v", err)
	}
}

// TestFormatoRecusaKindSkillSemSkills e' a contradicao declarativa: *declarar que ensina e
// nao trazer nada.*
func TestFormatoRecusaKindSkillSemSkills(t *testing.T) {
	data := []byte(`{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"skill"}`)
	_, err := ParseHadesPluginManifest(data, politicaParaFormato())
	if !errors.Is(err, ErrHadesPluginSkillPath) {
		t.Fatalf("kind=skill sem skills tem de ser recusado, veio: %v", err)
	}
}

// TestFormatoRecusaIconeRemoto e' o funcionamento OFFLINE: *um icone remoto faria o
// produto depender de rede para se apresentar.*
func TestFormatoRecusaIconeRemoto(t *testing.T) {
	data := []byte(`{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"bundle","interface":{"icon":"https://exemplo.invalido/i.png"}}`)
	_, err := ParseHadesPluginManifest(data, politicaParaFormato())
	if !errors.Is(err, ErrHadesPluginInterface) {
		t.Fatalf("icone remoto tem de ser recusado, veio: %v", err)
	}
}

// TestFormatoRecusaScopeDesconhecido prova que a governanca do nucleo continua a valer
// pelo formato novo: *o manifesto PEDE capacidades; so' as conhecidas.*
func TestFormatoRecusaScopeDesconhecido(t *testing.T) {
	data := []byte(`{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"bundle","scopes":["capacidade:inexistente"]}`)
	_, err := ParseHadesPluginManifest(data, politicaParaFormato())
	if !errors.Is(err, ErrUnknownCapability) {
		t.Fatalf("scope desconhecido tem de dar ErrUnknownCapability, veio: %v", err)
	}
}

// TestFormatoRecusaDependenciaDeSiMesmo prova que a regra do `:207` chega ao formato.
func TestFormatoRecusaDependenciaDeSiMesmo(t *testing.T) {
	data := []byte(`{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"bundle","dependencies":["x"]}`)
	_, err := ParseHadesPluginManifest(data, politicaParaFormato())
	if err == nil {
		t.Fatal("um plugin nao pode depender de si mesmo")
	}
}

// TestFormatoNaoTemSegundoValidador e' o teste que protege contra a DUPLICACAO.
//
// **A garantia**: o nucleo e' validado pelo `validatePluginManifest` canonico, e nao por
// uma copia local. *Se alguem reescrever as regras aqui, este teste continua a passar —
// mas o `TestFormatoRecusaIdComSeparador` e o `TestFormatoRecusaScopeDesconhecido`
// deixariam de apanhar as mudancas do nucleo.* Este teste **fixa o vinculo** por
// comportamento: um id invalido pelo validador canonico tem de ser invalido aqui.
func TestFormatoNaoTemSegundoValidador(t *testing.T) {
	// Um nucleo invalido pelo validador canonico...
	nucleoInvalido := PluginManifest{ID: "ok", Version: "1", Name: "", Kind: PluginKindBundle}
	if _, err := validatePluginManifest(nucleoInvalido, politicaParaFormato()); err == nil {
		t.Fatal("premissa: o validador canonico tem de recusar name vazio")
	}
	// ...tem de ser invalido pelo formato, pela MESMA razao.
	data := []byte(`{"format_version":"1.0","id":"ok","version":"1","name":"","kind":"bundle"}`)
	if _, err := ParseHadesPluginManifest(data, politicaParaFormato()); err == nil {
		t.Fatal("o formato tem de herdar a recusa do validador canonico")
	}
}
