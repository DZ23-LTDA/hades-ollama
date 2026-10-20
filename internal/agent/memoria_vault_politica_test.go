package agent

import (
	"errors"
	"strings"
	"testing"
)

// NEGATIVO: configuracao vazia nega tudo, e negar tudo nao pode ser confundido com erro.
func TestPoliticaVaziaNegaTudoSemErro(t *testing.T) {
	politica, err := ParsearPoliticaDeAcesso("   ")
	if err != nil {
		t.Fatalf("vazio devia ser aceite como politica vazia, veio %v", err)
	}
	if len(politica.Concessoes) != 0 {
		t.Fatalf("politica vazia devia nao ter concessoes, tem %d", len(politica.Concessoes))
	}
	if err := politica.Autorizar("user-1", EscopoTenant, PermissaoEscrever); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("politica vazia devia negar, veio %v", err)
	}
}

// NEGATIVO: entrada malformada e' ERRO, nunca "politica vazia".
func TestPoliticaMalformadaDaErroNaoPoliticaVazia(t *testing.T) {
	casos := map[string]string{
		"json invalido":          `{isto nao e' json`,
		"campo desconhecido":     `[{"identidade":"u","escopo":"TENANT","permissoes":["ler"],"org_id":"x"}]`,
		"lista vazia":            `[]`,
		"sem identidade":         `[{"escopo":"TENANT","permissoes":["ler"]}]`,
		"escopo desconhecido":    `[{"identidade":"u","escopo":"PLANETA","permissoes":["ler"]}]`,
		"sem permissoes":         `[{"identidade":"u","escopo":"TENANT","permissoes":[]}]`,
		"permissao desconhecida": `[{"identidade":"u","escopo":"TENANT","permissoes":["administrar"]}]`,
	}
	for nome, texto := range casos {
		if _, err := ParsearPoliticaDeAcesso(texto); !errors.Is(err, ErrPoliticaDeAcessoInvalida) {
			t.Fatalf("%s: esperava ErrPoliticaDeAcessoInvalida, veio %v", nome, err)
		}
	}
}

// POSITIVO: configuracao valida autoriza exactamente a identidade, o escopo e a permissao dados.
func TestPoliticaValidaAutorizaSoOQueDeclara(t *testing.T) {
	politica, err := ParsearPoliticaDeAcesso(`[
		{"identidade":"user-1","escopo":"TENANT","permissoes":["ler","escrever"]},
		{"identidade":"user-2","escopo":"TENANT","permissoes":["ler"]}
	]`)
	if err != nil {
		t.Fatalf("configuracao valida recusada: %v", err)
	}

	if err := politica.Autorizar("user-1", EscopoTenant, PermissaoEscrever); err != nil {
		t.Fatalf("user-1 devia poder escrever no TENANT: %v", err)
	}
	if err := politica.Autorizar("user-2", EscopoTenant, PermissaoLer); err != nil {
		t.Fatalf("user-2 devia poder ler no TENANT: %v", err)
	}

	// NEGATIVO: ler nao implica escrever.
	if err := politica.Autorizar("user-2", EscopoTenant, PermissaoEscrever); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("user-2 nao podia escrever, veio %v", err)
	}
	// NEGATIVO: identidade sem concessao.
	if err := politica.Autorizar("intruso", EscopoTenant, PermissaoLer); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("intruso nao podia ler, veio %v", err)
	}
	// NEGATIVO: concessao noutro escopo nao serve.
	if err := politica.Autorizar("user-1", EscopoProject, PermissaoEscrever); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("escopo diferente devia negar, veio %v", err)
	}
}

// POSITIVO: "ler+escrever" e' aceite como nome unico.
func TestPoliticaAceitaLerMaisEscrever(t *testing.T) {
	politica, err := ParsearPoliticaDeAcesso(`[{"identidade":"u","escopo":"tenant","permissoes":["ler+escrever"]}]`)
	if err != nil {
		t.Fatalf("nome combinado recusado: %v", err)
	}
	if err := politica.Autorizar("u", EscopoTenant, PermissaoEscrever); err != nil {
		t.Fatalf("devia poder escrever: %v", err)
	}
}

// A politica configurada alimenta o servico de append -- nao e' um contrato paralelo.
func TestPoliticaConfiguradaAlimentaOAnexarGovernado(t *testing.T) {
	raiz := t.TempDir()
	politica, err := ParsearPoliticaDeAcesso(`[{"identidade":"user-1","escopo":"TENANT","permissoes":["ler","escrever"]}]`)
	if err != nil {
		t.Fatalf("politica invalida: %v", err)
	}
	servico := ServicoDeAppendGovernado{Raiz: raiz, Politica: politica}

	if _, err := servico.Anexar(pedidoDeTeste("user-1", "org-1", "notas/da-politica.md")); err != nil {
		t.Fatalf("devia escrever com a politica configurada: %v", err)
	}
	if _, err := servico.Anexar(pedidoDeTeste("outro", "org-1", "notas/negado.md")); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("outra identidade devia ser negada, veio %v", err)
	}
	if n := contarFicheiros(t, raiz); n != 1 {
		t.Fatalf("esperava exactamente 1 ficheiro (o autorizado), ha' %d", n)
	}
}

// O erro nao pode conter o texto da configuracao (pode trazer identidades).
func TestErroDaPoliticaNaoEcoaAConfiguracao(t *testing.T) {
	segredo := "identidade-secreta-42"
	_, err := ParsearPoliticaDeAcesso(`[{"identidade":"` + segredo + `","escopo":"PLANETA","permissoes":["ler"]}]`)
	if err == nil {
		t.Fatal("esperava erro")
	}
	if strings.Contains(err.Error(), segredo) {
		t.Fatalf("o erro ecoou a identidade: %v", err)
	}
}
