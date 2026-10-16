package agent

import (
	"errors"
	"testing"
)

// E-973: A LIGACAO DO FORMATO AO REGISTO.
//
// **O que cada teste discrimina**: o `E-972` criou o formato; estes testes provam que ele
// **instala**, e que as `skills`/`interface` **sobrevivem a gravacao**. *Sem eles, o
// formato seria uma especificacao com testes de parser - e o encaixe, uma promessa.*

func registoParaFormato(t *testing.T) *PluginRegistry {
	t.Helper()
	registry, err := NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func manifestoAgentSkills() HadesPluginManifest {
	return HadesPluginManifest{
		FormatVersion: HadesPluginFormatVersion,
		PluginManifest: PluginManifest{
			ID:      "agent-skills",
			Version: "0.6.13",
			Name:    "Agent Skills",
			Kind:    PluginKindSkill,
			License: "MIT",
		},
		Skills: []HadesPluginSkill{
			{Name: "test-driven-development", Path: "skills/test-driven-development/SKILL.md"},
			{Name: "code-review-and-quality", Path: "skills/code-review-and-quality/SKILL.md"},
		},
		Interface: &HadesPluginInterface{
			DisplayName: "Agent Skills",
			Category:    "Productivity",
		},
	}
}

// TestLigacaoInstalaPeloCaminhoCanonico e' o controlo: *o formato instala no registo, e o
// nucleo aparece.*
func TestLigacaoInstalaPeloCaminhoCanonico(t *testing.T) {
	registry := registoParaFormato(t)
	installation, err := registry.InstallHadesPluginForOrganization("org-a", manifestoAgentSkills(), politicaParaFormato())
	if err != nil {
		t.Fatalf("a instalacao pelo formato tem de funcionar, veio: %v", err)
	}
	if installation.Manifest.ID != "agent-skills" || installation.Manifest.Version != "0.6.13" {
		t.Fatalf("o nucleo nao foi instalado: %+v", installation.Manifest)
	}
	// *E a instalacao entra DESCONFIADA e sem scopes concedidos* - a governanca do nucleo
	// continua a valer pelo formato novo.
	if installation.Trusted {
		t.Fatal("uma instalacao nova nao pode nascer Trusted")
	}
	if len(installation.GrantedScopes) != 0 {
		t.Fatalf("nenhum scope pode ser CONCEDIDO pela instalacao: %v", installation.GrantedScopes)
	}
}

// TestLigacaoAsSkillsSobrevivemAoRegisto e' O DISCRIMINADOR DA ARMADILHA DO CLONE.
//
// **O achado**: o `InstallForOrganization` termina com
// `return clonePluginInstallation(*installation)` (`plugin_registry.go:383`). *Mutar o
// valor devolvido alteraria um CLONE, e o registo ficaria sem as skills* — o manifesto
// seria aceito e a informacao perdida no momento de gravar.
//
// Este teste le do registo **noutra chamada** (`GetForOrganization`), e nao do valor
// devolvido: *e' a unica forma de provar que a gravacao aconteceu.*
func TestLigacaoAsSkillsSobrevivemAoRegisto(t *testing.T) {
	registry := registoParaFormato(t)
	if _, err := registry.InstallHadesPluginForOrganization("org-a", manifestoAgentSkills(), politicaParaFormato()); err != nil {
		t.Fatal(err)
	}
	lido, err := registry.GetForOrganization("org-a", "agent-skills")
	if err != nil {
		t.Fatalf("a instalacao tem de estar no registo: %v", err)
	}
	if len(lido.Skills) != 2 {
		t.Fatalf("as 2 skills declaradas têm de sobreviver ao registo, vieram %d", len(lido.Skills))
	}
	if lido.Skills[0].Name != "test-driven-development" {
		t.Fatalf("a skill lida nao e' a declarada: %+v", lido.Skills[0])
	}
	if lido.Interface == nil || lido.Interface.DisplayName != "Agent Skills" {
		t.Fatalf("a interface nao sobreviveu ao registo: %+v", lido.Interface)
	}
	if lido.NomeVisivel() != "Agent Skills" {
		t.Fatalf("o nome visivel tem de vir da interface: %q", lido.NomeVisivel())
	}
}

// TestLigacaoNomeVisivelUsaONucleoComoFallback prova a outra metade: *sem `interface`, a
// resposta honesta e' o nome do nucleo.*
func TestLigacaoNomeVisivelUsaONucleoComoFallback(t *testing.T) {
	manifesto := manifestoAgentSkills()
	manifesto.Interface = nil
	registry := registoParaFormato(t)
	if _, err := registry.InstallHadesPluginForOrganization("org-a", manifesto, politicaParaFormato()); err != nil {
		t.Fatal(err)
	}
	lido, err := registry.GetForOrganization("org-a", "agent-skills")
	if err != nil {
		t.Fatal(err)
	}
	if lido.NomeVisivel() != "Agent Skills" {
		t.Fatalf("sem interface, o nome visivel e' o do nucleo: %q", lido.NomeVisivel())
	}
}

// TestLigacaoRecusaFormatoInvalidoAntesDeInstalar prova que o formato **nao** afrouxa: *um
// manifesto invalido nao chega a tocar o registo.*
func TestLigacaoRecusaFormatoInvalidoAntesDeInstalar(t *testing.T) {
	registry := registoParaFormato(t)
	manifesto := manifestoAgentSkills()
	manifesto.FormatVersion = "9.9"
	if _, err := registry.InstallHadesPluginForOrganization("org-a", manifesto, politicaParaFormato()); !errors.Is(err, ErrHadesPluginVersion) {
		t.Fatalf("versao de formato invalida tem de ser recusada, veio: %v", err)
	}
	if _, err := registry.GetForOrganization("org-a", "agent-skills"); !errors.Is(err, ErrPluginNotFound) {
		t.Fatalf("nada pode ter sido instalado, veio: %v", err)
	}
}

// TestLigacaoRecusaSkillComTravessiaAntesDeInstalar prova que a defesa do formato vale no
// caminho real: *um caminho com `..` nao chega ao registo.*
func TestLigacaoRecusaSkillComTravessiaAntesDeInstalar(t *testing.T) {
	registry := registoParaFormato(t)
	manifesto := manifestoAgentSkills()
	manifesto.Skills[0].Path = "../../etc/passwd"
	if _, err := registry.InstallHadesPluginForOrganization("org-a", manifesto, politicaParaFormato()); !errors.Is(err, ErrHadesPluginSkillPath) {
		t.Fatalf("travessia tem de ser recusada, veio: %v", err)
	}
	if _, err := registry.GetForOrganization("org-a", "agent-skills"); !errors.Is(err, ErrPluginNotFound) {
		t.Fatal("nada pode ter sido instalado")
	}
}

// TestLigacaoPreservaOTenantIsolationDoNucleo prova que a ligacao **nao** contorna a
// regra de organizacao: *um id instalado numa organizacao nao entra noutra.*
//
// **E o discriminador de que nao ha' segunda instalacao**: se eu tivesse reimplementado a
// instalacao neste ficheiro, esta recusa poderia ter sido esquecida.
func TestLigacaoPreservaOTenantIsolationDoNucleo(t *testing.T) {
	registry := registoParaFormato(t)
	if _, err := registry.InstallHadesPluginForOrganization("org-a", manifestoAgentSkills(), politicaParaFormato()); err != nil {
		t.Fatal(err)
	}
	_, err := registry.InstallHadesPluginForOrganization("org-b", manifestoAgentSkills(), politicaParaFormato())
	if !errors.Is(err, ErrPluginOrganizationScope) {
		t.Fatalf("o mesmo id noutra organizacao tem de dar ErrPluginOrganizationScope, veio: %v", err)
	}
}

// TestLigacaoRecusaReinstalacaoDaMesmaVersao prova que a regra de versao do nucleo vale
// pelo formato.
func TestLigacaoRecusaReinstalacaoDaMesmaVersao(t *testing.T) {
	registry := registoParaFormato(t)
	if _, err := registry.InstallHadesPluginForOrganization("org-a", manifestoAgentSkills(), politicaParaFormato()); err != nil {
		t.Fatal(err)
	}
	_, err := registry.InstallHadesPluginForOrganization("org-a", manifestoAgentSkills(), politicaParaFormato())
	if !errors.Is(err, ErrPluginVersionConflict) {
		t.Fatalf("a mesma versao tem de dar ErrPluginVersionConflict, veio: %v", err)
	}
}

// TestLigacaoSkillsDeclaradasNaoPartilhamSlice prova que a leitura nao entrega o slice
// interno: *mutar o que se leu nao pode alterar o registo.*
func TestLigacaoSkillsDeclaradasNaoPartilhamSlice(t *testing.T) {
	registry := registoParaFormato(t)
	if _, err := registry.InstallHadesPluginForOrganization("org-a", manifestoAgentSkills(), politicaParaFormato()); err != nil {
		t.Fatal(err)
	}
	lido, err := registry.GetForOrganization("org-a", "agent-skills")
	if err != nil {
		t.Fatal(err)
	}
	skills := lido.SkillsDeclaradas()
	skills[0].Name = "ADULTERADA"
	outra, err := registry.GetForOrganization("org-a", "agent-skills")
	if err != nil {
		t.Fatal(err)
	}
	if outra.Skills[0].Name == "ADULTERADA" {
		t.Fatal("mutar o que se leu alterou o registo: o slice interno foi entregue")
	}
}
