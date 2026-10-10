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
	// O registro cobre os 32 comandos da seção 6 mais /cancel, /approve e /deny
	// exigidos pela §11.1.
	if got := len(agent.DefaultSlashRegistry()); got != 35 {
		t.Fatalf("registry has %d commands, want 35", got)
	}
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
