package server

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// TestSlashRegistryBackendsAreActuallyRegistered é a prova de honestidade do
// registro de comandos: todo comando marcado como DISPONÍVEL precisa apontar para
// um caminho que o roteador realmente registra. Se alguém inventar uma rota, este
// teste falha — nenhum comando pode existir apenas no papel.
func TestSlashRegistryBackendsAreActuallyRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)

	registered := map[string]gin.RouteInfo{}
	for _, route := range engine.Routes() {
		registered[route.Method+" "+route.Path] = route
	}
	if len(registered) == 0 {
		t.Fatal("router registered no routes")
	}

	available := 0
	for _, spec := range agent.DefaultSlashRegistry() {
		if !spec.Available {
			continue
		}
		available++
		method, path, ok := strings.Cut(spec.Backend, " ")
		if !ok {
			t.Fatalf("command %s has a malformed backend %q", spec.Name, spec.Backend)
		}
		if _, ok := registered[method+" "+path]; !ok {
			t.Fatalf("command %s declares %s %s but the router does not register it", spec.Name, method, path)
		}
	}
	if available == 0 {
		t.Fatal("registry has no available command")
	}
	// ***O NÚMERO NÃO SE REPETE AQUI. E-550.***
	//
	// *Este teste contava `len(agent.DefaultSlashRegistry()) != 38`. O `E-547` mediu o custo dessa cópia: quando o
	// `E-533` levou o registo de 35 para 38, ele corrigiu a lista do `internal/agent` e **esqueceu-se desta** — e
	// quatro jobs do CI ficaram vermelhos por um número.*
	//
	// ⇒ ***Duas cópias de um número não são uma verificação: são uma coincidência à espera de divergir.***
	//
	// **A contagem já é verificada onde pertence:** `internal/agent/slash_registry_test.go` compara
	// `len(registry) == len(required)`, e a lista `required` é o **mandato** — *a verificação bidireccional de que
	// acrescentar um comando exige uma **decisão**.*
	//
	// **O que ESTE teste verifica — e que só ele pode verificar** — é a propriedade que depende do **router**: que
	// cada `Backend` declarado por um comando disponível está **de facto registado** (o laço acima). *Isso exige
	// construir o engine do `gin`, e é por isso que vive aqui e não no `internal/agent`.*
	//
	// *Removido o número, o teste fica **mais** forte: passa a verificar **uma propriedade** em vez de repetir
	// **uma contagem** que outro pacote já faz.*
}

// TestSlashCommandRoutesExistForSection11MandatoryCommands garante que os seis
// comandos exigidos nominalmente pela §11.1 têm backend de verdade e que o
// caminho declarado responde ao método correto.
func TestSlashCommandRoutesExistForSection11MandatoryCommands(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	registered := map[string]bool{}
	for _, route := range engine.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, name := range []string{"/goal", "/status", "/resume", "/cancel", "/approve", "/deny"} {
		spec, ok := agent.SlashCommandSpecByName(name)
		if !ok {
			t.Fatalf("command %s missing", name)
		}
		if !spec.Available {
			t.Fatalf("command %s required by §11.1 is not available", name)
		}
		if !registered[spec.Backend] {
			t.Fatalf("command %s backend %q is not registered", name, spec.Backend)
		}
	}
}
