package agent

import (
	"errors"
	"strings"
	"testing"
)

func TestSlashRegistryCoversSection6AndIsHonest(t *testing.T) {
	registry := DefaultSlashRegistry()
	byName := map[string]SlashCommandSpec{}
	for _, spec := range registry {
		if _, duplicated := byName[spec.Name]; duplicated {
			t.Fatalf("duplicated command %s", spec.Name)
		}
		byName[spec.Name] = spec
		if !strings.HasPrefix(spec.Name, "/") || spec.Name != strings.ToLower(spec.Name) {
			t.Fatalf("invalid command name %q", spec.Name)
		}
		if strings.TrimSpace(spec.Summary) == "" {
			t.Fatalf("command %s has no pt-BR summary", spec.Name)
		}
		// Invariante central: comando disponível TEM rota; indisponível TEM motivo.
		if spec.Available {
			if !strings.HasPrefix(spec.Backend, "GET /") && !strings.HasPrefix(spec.Backend, "POST /") &&
				!strings.HasPrefix(spec.Backend, "PATCH /") && !strings.HasPrefix(spec.Backend, "PUT /") &&
				!strings.HasPrefix(spec.Backend, "DELETE /") {
				t.Fatalf("available command %s has no real backend path: %q", spec.Name, spec.Backend)
			}
			if strings.TrimSpace(spec.UnavailableReason) != "" {
				t.Fatalf("available command %s must not carry an unavailable reason", spec.Name)
			}
			continue
		}
		if strings.TrimSpace(spec.UnavailableReason) == "" {
			t.Fatalf("unavailable command %s must explain why", spec.Name)
		}
		if strings.TrimSpace(spec.Backend) != "" {
			t.Fatalf("unavailable command %s must not declare a backend", spec.Name)
		}
	}

	// Todos os comandos da seção 6 do mandato, mais os exigidos pela §11.1.
	required := []string{
		"/goal", "/plan", "/build", "/code", "/review", "/research", "/company", "/store", "/sell",
		"/marketing", "/social", "/phone", "/support", "/agents", "/mcp", "/skills", "/automate",
		"/memory", "/deploy", "/status", "/resume", "/whatsapp", "/telegram", "/inbox", "/voice",
		"/calls", "/channels", "/crm", "/orders", "/handoff", "/campaign", "/business-status",
		"/cancel", "/approve", "/deny",
	}
	for _, name := range required {
		if _, ok := byName[name]; !ok {
			t.Fatalf("command %s missing from the registry", name)
		}
	}
	if len(registry) != len(required) {
		t.Fatalf("registry has %d commands, mandate lists %d", len(registry), len(required))
	}

	// §11.1 nomeia explicitamente estes seis: precisam estar disponíveis de fato.
	for _, name := range []string{"/goal", "/status", "/resume", "/cancel", "/approve", "/deny"} {
		if !byName[name].Available {
			t.Fatalf("command %s is required by §11.1 and must be available", name)
		}
	}
	// E o registro não pode mentir sobre o que não existe.
	for _, name := range []string{"/research", "/support", "/telegram", "/inbox", "/crm", "/handoff"} {
		if byName[name].Available {
			t.Fatalf("command %s must be marked unavailable while it has no route", name)
		}
	}
}

func TestParseSlashInvocationValidatesArguments(t *testing.T) {
	invocation, err := ParseSlashInvocation("/goal auditar")
	if err != nil {
		t.Fatalf("parse /goal: %v", err)
	}
	if invocation.Spec.Name != "/goal" || invocation.Args["objetivo"] != "auditar" {
		t.Fatalf("parsed invocation = %+v", invocation)
	}
	// O objetivo é UM parâmetro por posição: o restante é recusado em vez de
	// ignorado em silêncio (schema validado, não "engolido").
	if _, err := ParseSlashInvocation("/goal auditar o repositório inteiro"); !errors.Is(err, ErrSlashCommandArgsInvalid) {
		t.Fatalf("extra arguments must be refused, got %v", err)
	}
	// O segundo argumento é opcional (workspace), então dois campos são aceitos.
	if _, err := ParseSlashInvocation("/goal auditar C:/tmp/ws"); err != nil {
		t.Fatalf("optional argument must be accepted: %v", err)
	}
	// Aceita sem barra inicial (mesma ergonomia do parser histórico).
	if _, err := ParseSlashInvocation("mcp"); err != nil {
		t.Fatalf("bare command must parse: %v", err)
	}
	// Faltando argumento obrigatório: erro claro com o nome do argumento.
	if _, err := ParseSlashInvocation("/status"); !errors.Is(err, ErrSlashCommandArgsInvalid) || !strings.Contains(err.Error(), "missao") {
		t.Fatalf("missing argument error = %v", err)
	}
	// Comando desconhecido.
	if _, err := ParseSlashInvocation("/nao-existe x"); !errors.Is(err, ErrUnknownSlashCommand) {
		t.Fatalf("unknown command error = %v", err)
	}
	// Comando indisponível: recusado COM o motivo, nunca executado.
	_, err = ParseSlashInvocation("/inbox")
	if !errors.Is(err, ErrSlashCommandUnavailable) {
		t.Fatalf("unavailable command error = %v", err)
	}
	if !strings.Contains(err.Error(), "caixa de entrada") {
		t.Fatalf("unavailable error must carry the reason: %v", err)
	}
}

func TestParseSlashInvocationValidatesArgKinds(t *testing.T) {
	// id com espaço/caractere inválido é recusado.
	if _, err := ParseSlashInvocation(`/status "meu id"`); !errors.Is(err, ErrSlashCommandArgsInvalid) {
		t.Fatalf("invalid id error = %v", err)
	}
	if _, err := ParseSlashInvocation("/status missao-1"); err != nil {
		t.Fatalf("valid id must parse: %v", err)
	}
	// URL de repositório precisa ser http(s).
	if _, err := ParseSlashInvocation("/code git@github.com:DZ23-LTDA/hades-ollama.git"); !errors.Is(err, ErrSlashCommandArgsInvalid) {
		t.Fatalf("non-http url must be refused, got %v", err)
	}
	invocation, err := ParseSlashInvocation("/code https://github.com/DZ23-LTDA/hades-ollama")
	if err != nil {
		t.Fatalf("valid url must parse: %v", err)
	}
	if invocation.Args["repositorio"] != "https://github.com/DZ23-LTDA/hades-ollama" {
		t.Fatalf("url argument = %+v", invocation.Args)
	}
	// Aprovação exige missão E aprovação.
	if _, err := ParseSlashInvocation("/approve missao-1"); !errors.Is(err, ErrSlashCommandArgsInvalid) {
		t.Fatalf("missing approval id error = %v", err)
	}
	if _, err := ParseSlashInvocation("/approve missao-1 apr-9"); err != nil {
		t.Fatalf("approve with both ids: %v", err)
	}
}

func TestSlashHelpAndAutocomplete(t *testing.T) {
	help := SlashHelp()
	for _, spec := range DefaultSlashRegistry() {
		if !strings.Contains(help, spec.Name) {
			t.Fatalf("help omits %s", spec.Name)
		}
	}
	if !strings.Contains(help, "Comandos indisponíveis nesta versão (não executam nada)") {
		t.Fatalf("help must state that unavailable commands do nothing: %s", help)
	}
	if !strings.Contains(help, "/inbox —") {
		t.Fatalf("help must list /inbox among unavailable: %s", help)
	}

	matches := CompleteSlashCommand("/go")
	if len(matches) != 1 || matches[0].Name != "/goal" {
		t.Fatalf("autocomplete /go = %+v", matches)
	}
	if got := CompleteSlashCommand("/zzz"); len(got) != 0 {
		t.Fatalf("autocomplete unknown prefix = %+v", got)
	}
	if got := CompleteSlashCommand(""); len(got) != len(DefaultSlashRegistry()) {
		t.Fatalf("empty prefix must list everything, got %d", len(got))
	}
	if got := CompleteSlashCommand("me"); len(got) != 1 || got[0].Name != "/memory" {
		t.Fatalf("autocomplete without slash = %+v", got)
	}
	// Ordenado e com nomes únicos: a UI pode renderizar direto.
	previous := ""
	for _, spec := range CompleteSlashCommand("") {
		if spec.Name <= previous {
			t.Fatalf("autocomplete must be sorted: %s after %s", spec.Name, previous)
		}
		previous = spec.Name
	}
}
