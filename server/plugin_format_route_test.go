package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// E-976: A SUPERFICIE DO FORMATO ABERTO.
//
// **O que faltava**: o formato existia (`E-972`) e instalava no registo (`E-974`), mas
// **so' por dentro do processo**. Estes testes provam a **rota** - e o discriminante
// central e' que a organizacao vem do **contexto**, nunca do corpo.

func formatoRouteContext(body, organization, role string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/plugins/format", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if organization != "" {
		ctx.Set("agent.organization", agent.Organization{ID: organization})
	}
	if role != "" {
		ctx.Set("agent.membership", agent.Membership{Role: agent.Role(role), OrganizationID: organization, UserID: "user-1"})
	}
	return ctx, recorder
}

func formatoRouteAPI(t *testing.T) *agentAPI {
	t.Helper()
	gin.SetMode(gin.TestMode)
	registry, err := agent.NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &agentAPI{plugins: registry, authRequired: true}
}

// TestFormatoRouteInstalaPeloFormatoAberto e' o controlo: *a rota instala um
// `hades-plugin.json` e devolve 201 com a instalacao.*
func TestFormatoRouteInstalaPeloFormatoAberto(t *testing.T) {
	api := formatoRouteAPI(t)
	body := `{
	  "format_version": "1.0",
	  "id": "agent-skills",
	  "version": "0.6.13",
	  "name": "Agent Skills",
	  "kind": "skill",
	  "license": "MIT",
	  "skills": [{"name": "test-driven-development", "path": "skills/test-driven-development/SKILL.md"}],
	  "interface": {"display_name": "Agent Skills", "category": "Productivity"}
	}`
	ctx, recorder := formatoRouteContext(body, "org-a", "admin")
	api.installHadesPlugin(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("esperado 201, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	// *E as skills tem de vir na RESPOSTA* - se vierem vazias, a persistencia falhou.
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("a resposta tem de ser JSON: %v", err)
	}
	skills, ok := payload["skills"].([]any)
	if !ok || len(skills) != 1 {
		t.Fatalf("as skills tem de vir na resposta, veio: %v", payload["skills"])
	}
}

// TestFormatoRouteExigeAdmin e' a porta: *sem papel de admin, nao se instala.*
func TestFormatoRouteExigeAdmin(t *testing.T) {
	api := formatoRouteAPI(t)
	ctx, recorder := formatoRouteContext(`{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"bundle"}`, "org-a", "member")
	api.installHadesPlugin(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("sem admin tem de dar 403, veio %d", recorder.Code)
	}
}

// TestFormatoRouteNaoAceitaTenantDoCorpo e' O DISCRIMINANTE CENTRAL.
//
// **A garantia**: a organizacao vem do **contexto autenticado**, e o corpo **nao** a pode
// escolher. *E o `DisallowUnknownFields` torna isto **mais forte** que «ignorado»: o campo
// e' **recusado na entrada**.*
func TestFormatoRouteNaoAceitaTenantDoCorpo(t *testing.T) {
	api := formatoRouteAPI(t)
	body := `{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"bundle","organization_id":"org-INTRUSA"}`
	ctx, recorder := formatoRouteContext(body, "org-a", "admin")
	api.installHadesPlugin(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("o corpo nao pode escolher tenant; esperado 400, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "unknown field") {
		t.Fatalf("a recusa tem de nomear o campo desconhecido: %s", recorder.Body.String())
	}
}

// TestFormatoRouteRecusaVersaoDeFormatoInvalida prova que a versao do formato e' a porta
// de evolucao: *um `format_version` que o runtime nao conhece nao instala.*
func TestFormatoRouteRecusaVersaoDeFormatoInvalida(t *testing.T) {
	api := formatoRouteAPI(t)
	ctx, recorder := formatoRouteContext(`{"format_version":"9.9","id":"x","version":"1","name":"X","kind":"bundle"}`, "org-a", "admin")
	api.installHadesPlugin(ctx)
	if recorder.Code == http.StatusCreated {
		t.Fatalf("versao desconhecida nao pode criar instalacao: %s", recorder.Body.String())
	}
}

// TestFormatoRouteRecusaTravessia prova que a defesa do formato vale pela rota: *um
// caminho com `..` e' recusado antes de tocar o registo.*
func TestFormatoRouteRecusaTravessia(t *testing.T) {
	api := formatoRouteAPI(t)
	body := `{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"skill","skills":[{"name":"s","path":"../../etc/passwd"}]}`
	ctx, recorder := formatoRouteContext(body, "org-a", "admin")
	api.installHadesPlugin(ctx)
	if recorder.Code == http.StatusCreated {
		t.Fatalf("travessia nao pode instalar: %s", recorder.Body.String())
	}
}

// TestFormatoRouteNaoInstalaDuasVezes prova que a regra de versao do registo chega a'
// superficie: *a segunda instalacao do mesmo id@versao e' recusada.*
func TestFormatoRouteNaoInstalaDuasVezes(t *testing.T) {
	api := formatoRouteAPI(t)
	body := `{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"bundle"}`
	ctx, recorder := formatoRouteContext(body, "org-a", "admin")
	api.installHadesPlugin(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("a primeira tem de instalar, veio %d", recorder.Code)
	}
	ctx2, recorder2 := formatoRouteContext(body, "org-a", "admin")
	api.installHadesPlugin(ctx2)
	if recorder2.Code == http.StatusCreated {
		t.Fatalf("a segunda tem de ser recusada: %s", recorder2.Body.String())
	}
}

// TestFormatoRouteDescreveOFormato prova a leitura pura: *o `$schema`, a versao, os cinco
// `kind` e os runtimes.*
//
// **E ela importa ao autor do plugin**: *declarar o que o runtime aceita **antes** de o
// escrever evita um manifesto que falha na instalacao por um motivo que nao se ve.*
func TestFormatoRouteDescreveOFormato(t *testing.T) {
	api := formatoRouteAPI(t)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/plugins/format", nil)
	api.hadesPluginFormat(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("esperado 200, veio %d", recorder.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("JSON invalido: %v", err)
	}
	if payload["format_version"] != agent.HadesPluginFormatVersion {
		t.Fatalf("a versao do formato tem de ser a do runtime: %v", payload["format_version"])
	}
	if !strings.Contains(payload["schema"].(string), "hades-plugin") {
		t.Fatalf("o schema tem de ser o do HADES: %v", payload["schema"])
	}
	kinds, _ := payload["kinds"].([]any)
	if len(kinds) != 5 {
		t.Fatalf("os cinco kind do validador canonico, vieram %d", len(kinds))
	}
	runtimes, _ := payload["runtimes"].([]any)
	if len(runtimes) == 0 {
		t.Fatal("os runtimes suportados tem de ser declarados")
	}
}

// TestFormatoRouteFalhaFechadoSemRegisto segue o padrao do `E-449` e do `E-966`:
// *sem registo nao ha' superficie.*
func TestFormatoRouteFalhaFechadoSemRegisto(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &agentAPI{authRequired: false}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/plugins/format", strings.NewReader(`{"format_version":"1.0","id":"x","version":"1","name":"X","kind":"bundle"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	api.installHadesPlugin(ctx)
	// Sem registo, `a.plugins` e' nil: *o panic do nil-pointer e' o pior resultado; o
	// esperado e' um erro HTTP.*
	if recorder.Code == http.StatusCreated {
		t.Fatal("sem registo nao pode criar instalacao")
	}
}
