package agent

// Testes do `PublishingProvider` (E-1269) -- e com ele fecham os DEZ providers do adendo 28.1.
//
// O que estes testes travam:
//
//  1. `LOCAL_ONLY` **nao e' canal publico** -- e `pronto` significa "esta' no disco", nao "publicado";
//  2. **`Publicado` e' sempre false** -- nao ha' `submit` neste provider;
//  3. canal publico **exige aprovacao** mesmo com o preflight completo;
//  4. canal publico **nunca e' declarado disponivel** para executar nesta missao;
//  5. sem artefacto, o preflight nao passa;
//  6. sem credencial, canal publico nao passa;
//  7. a credencial e' verificada por EXISTENCIA e o detalhe diz que o conteudo NAO foi lido;
//  8. canal desconhecido nao e' declarado suportado nem recusado por omissao;
//  9. o catalogo classifica o `LOCAL_ONLY` como nao-publico.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// artefactoParaPreflight escreve um ficheiro e devolve o caminho.
func artefactoParaPreflight(t *testing.T) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "HadesSetup.exe")
	if err := os.WriteFile(caminho, []byte("conteudo do instalador"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	return caminho
}

// LOCAL_ONLY NAO E' DEPLOY PUBLICO.
func TestPublishingProviderLocalNaoEPublico(t *testing.T) {
	artefacto := artefactoParaPreflight(t)
	provider := ProviderDePublicacao{}
	preflight, err := provider.Preflight(context.Background(), PublicacaoLocal, artefacto, "")
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if preflight.Publico {
		t.Fatal("LOCAL_ONLY declarado publico")
	}
	if !preflight.Pronto {
		t.Fatalf("preflight local devia passar: %+v", preflight)
	}
	if preflight.Publicado {
		t.Fatal("publicado = true")
	}
	// A explicacao tem de negar a equivalencia com deploy publico.
	if !strings.Contains(preflight.Explicacao, "NAO que foi publicado") {
		t.Fatalf("explicacao nao nega o deploy publico: %q", preflight.Explicacao)
	}
	// Canal local nao exige aprovacao (nada saiu da maquina).
	if preflight.RequerAprovacao {
		t.Fatal("LOCAL_ONLY marcado como requerendo aprovacao de publicacao")
	}
}

// O PROVIDER NAO PUBLICA: `Publicado` e' sempre false.
func TestPublishingProviderNuncaPublica(t *testing.T) {
	artefacto := artefactoParaPreflight(t)
	credencial := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(credencial, []byte(`{"type":"service_account"}`), 0o600); err != nil {
		t.Fatalf("escrever credencial: %v", err)
	}
	provider := ProviderDePublicacao{}
	for _, canal := range []CanalDePublicacao{
		PublicacaoGooglePlay, PublicacaoAppStore, PublicacaoEASUpdate,
		PublicacaoGitHubRelease, PublicacaoOCI, PublicacaoLocal,
	} {
		preflight, err := provider.Preflight(context.Background(), canal, artefacto, credencial)
		if err != nil {
			t.Fatalf("%s: erro inesperado: %v", canal, err)
		}
		if preflight.Publicado {
			t.Fatalf("%s: publicado = true", canal)
		}
		if !strings.Contains(preflight.Aviso, "NAO publicou nada") {
			t.Fatalf("%s: aviso nao declara que nada foi publicado: %q", canal, preflight.Aviso)
		}
	}
}

// Canal publico exige aprovacao, e NUNCA e' declarado disponivel para executar nesta missao.
func TestPublishingProviderCanalPublicoExigeAprovacao(t *testing.T) {
	artefacto := artefactoParaPreflight(t)
	credencial := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(credencial, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	provider := ProviderDePublicacao{}
	preflight, err := provider.Preflight(context.Background(), PublicacaoGooglePlay, artefacto, credencial)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if !preflight.RequerAprovacao {
		t.Fatal("canal publico sem requer_aprovacao")
	}
	// A condicao "canal disponivel" tem de estar INSATISFEITA para canal publico.
	var encontrada bool
	for _, condicao := range preflight.Condicoes {
		if condicao.Nome != "canal disponivel" {
			continue
		}
		encontrada = true
		if condicao.Satisfeita {
			t.Fatal("canal publico declarado disponivel para executar nesta missao")
		}
		if !strings.Contains(condicao.Detalhe, "proibe publicar") {
			t.Fatalf("detalhe nao declara a proibicao: %q", condicao.Detalhe)
		}
	}
	if !encontrada {
		t.Fatal("a condicao `canal disponivel` nao foi verificada")
	}
	if preflight.Pronto {
		t.Fatal("preflight publico declarado pronto")
	}
}

func TestPublishingProviderSemArtefactoNaoPassa(t *testing.T) {
	provider := ProviderDePublicacao{}
	preflight, err := provider.Preflight(context.Background(), PublicacaoLocal, "", "")
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if preflight.Pronto {
		t.Fatal("pronto = true sem artefacto")
	}
	if !strings.Contains(preflight.Explicacao, "NAO passou") {
		t.Fatalf("explicacao nao declara a falha: %q", preflight.Explicacao)
	}
}

func TestPublishingProviderArtefactoInexistente(t *testing.T) {
	provider := ProviderDePublicacao{}
	preflight, err := provider.Preflight(
		context.Background(), PublicacaoLocal,
		filepath.Join(t.TempDir(), "nao-existe.exe"), "",
	)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if preflight.Pronto {
		t.Fatal("pronto = true para um artefacto inexistente")
	}
}

// A credencial e' verificada por EXISTENCIA; o conteudo NAO e' lido.
func TestPublishingProviderVerificaCredencialSemLer(t *testing.T) {
	artefacto := artefactoParaPreflight(t)
	credencial := filepath.Join(t.TempDir(), "service-account.json")
	// Conteudo que NAO pode aparecer em nenhum campo do resultado.
	const sentinela = "SEGREDO-QUE-NAO-PODE-VAZAR"
	if err := os.WriteFile(credencial, []byte(sentinela), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	provider := ProviderDePublicacao{}
	preflight, err := provider.Preflight(
		context.Background(), PublicacaoGooglePlay, artefacto, credencial,
	)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	var detalheDaCredencial string
	for _, condicao := range preflight.Condicoes {
		if condicao.Nome != "credencial presente" {
			continue
		}
		detalheDaCredencial = condicao.Detalhe
		if !condicao.Satisfeita {
			t.Fatal("credencial presente mas condicao insatisfeita")
		}
		if !strings.Contains(condicao.Detalhe, "conteudo NAO lido") {
			t.Fatalf("detalhe nao declara que o conteudo nao foi lido: %q", condicao.Detalhe)
		}
	}
	if detalheDaCredencial == "" {
		t.Fatal("a condicao `credencial presente` nao foi verificada")
	}
	// E a sentinela nao pode ter vazado para o resultado.
	if strings.Contains(detalheDaCredencial, sentinela) {
		t.Fatal("o conteudo da credencial vazou para o resultado")
	}
	if strings.Contains(preflight.Explicacao, sentinela) {
		t.Fatal("o conteudo da credencial vazou para a explicacao")
	}
}

func TestPublishingProviderCanalDesconhecido(t *testing.T) {
	provider := ProviderDePublicacao{}
	preflight, err := provider.Preflight(
		context.Background(), CanalDePublicacao("INVENTADO"), "", "",
	)
	if !errors.Is(err, ErrRollbackNaoSuportado) {
		t.Fatalf("err = %v; esperado ErrRollbackNaoSuportado", err)
	}
	if preflight.Pronto {
		t.Fatal("canal desconhecido declarado pronto")
	}
	if !strings.Contains(preflight.Explicacao, "nao afirma que serve nem que nao serve") {
		t.Fatalf("explicacao nao declara a ignorancia: %q", preflight.Explicacao)
	}
}

func TestPublishingProviderCatalogo(t *testing.T) {
	catalogo := CanaisDePublicacao()
	// *"Publicar localmente" nao pode ser apresentado como deploy publico.*
	if publico, existe := catalogo[PublicacaoLocal]; !existe || publico {
		t.Fatal("LOCAL_ONLY ausente ou declarado publico no catalogo")
	}
	// Os canais externos sao publicos.
	for _, canal := range []CanalDePublicacao{
		PublicacaoGooglePlay, PublicacaoAppStore, PublicacaoEASUpdate,
		PublicacaoGitHubRelease, PublicacaoOCI,
	} {
		publico, existe := catalogo[canal]
		if !existe {
			t.Fatalf("%q ausente do catalogo", canal)
		}
		if !publico {
			t.Fatalf("%q declarado nao-publico", canal)
		}
	}
}
