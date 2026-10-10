package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// newSkillActivationAPI monta a API com um store real e uma skill ASSINADA e
// promovida (o caminho de produção), mais uma skill apenas registrada.
func newSkillActivationAPI(t *testing.T) *agentAPI {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	policy := agent.DefaultCapabilityPolicy().WithTrustedSkillKey("release-key", publicKey)
	trusted := agent.SkillManifest{
		ID:             "skill-runtime",
		OrganizationID: "org-skills",
		Version:        "1.0.0",
		Description:    "skill assinada",
		Category:       "dados",
		Scopes:         []string{"workspace:read"},
		TokenBudget:    30,
		// Enabled precisa estar definido ANTES de assinar: o registro força a
		// habilitação e isso mudaria os bytes canônicos do manifesto.
		Enabled: true,
	}
	signed, err := agent.SignSkillManifest(trusted, privateKey, "release-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkillForOrganization("org-skills", signed); err != nil {
		t.Fatal(err)
	}
	if err := store.PromoteSkillTrustedForOrganization("org-skills", trusted.ID, policy); err != nil {
		t.Fatal(err)
	}
	// Segunda skill: registrada, mas SEM assinatura autorizada.
	if err := store.RegisterSkillForOrganization("org-skills", agent.SkillManifest{
		ID:          "skill-sem-assinatura",
		Version:     "1.0.0",
		Description: "não confiável",
		Category:    "dados",
	}); err != nil {
		t.Fatal(err)
	}
	return &agentAPI{context: store, authRequired: true}
}

func skillRequest(t *testing.T, method, path, body, skillID, organization string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if skillID != "" {
		ctx.Params = gin.Params{{Key: "id", Value: skillID}}
	}
	ctx.Set("agent.organization", agent.Organization{ID: organization})
	return ctx, recorder
}

func TestSkillActivationRouteRefusesUntrustedSkill(t *testing.T) {
	api := newSkillActivationAPI(t)
	ctx, recorder := skillRequest(t, http.MethodPost, "/api/agent/v1/skills/skill-sem-assinatura/activate", "{}", "skill-sem-assinatura", "org-skills")
	api.activateSkill(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("skill não confiável deveria ser 403, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "not trusted") {
		t.Fatalf("o motivo precisa ser explícito: %s", recorder.Body.String())
	}
}

func TestSkillActivationRouteActivatesAndSelectsWithBudget(t *testing.T) {
	api := newSkillActivationAPI(t)

	// Ativação com escopo de projeto.
	ctx, recorder := skillRequest(t, http.MethodPost, "/api/agent/v1/skills/skill-runtime/activate",
		`{"project_id":"proj-1"}`, "skill-runtime", "org-skills")
	api.activateSkill(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("ativação falhou: %d %s", recorder.Code, recorder.Body.String())
	}

	// Seleção no contexto certo: entra e consome 30 tokens.
	ctx, recorder = skillRequest(t, http.MethodGet, "/api/agent/v1/skills/active?project_id=proj-1&token_budget=100", "", "", "org-skills")
	api.activeSkills(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("seleção falhou: %d %s", recorder.Code, recorder.Body.String())
	}
	var selection agent.SkillSelection
	if err := json.Unmarshal(recorder.Body.Bytes(), &selection); err != nil {
		t.Fatal(err)
	}
	if len(selection.Selected) != 1 || selection.Selected[0].ID != "skill-runtime" {
		t.Fatalf("selecionadas = %+v", selection.Selected)
	}
	if selection.UsedTokens != 30 || !selection.BudgetDefined {
		t.Fatalf("orçamento = %+v", selection)
	}
	if len(selection.Selected[0].Activation.SkillID) == 0 {
		t.Fatal("proveniência da ativação ausente")
	}

	// Contexto de outro projeto: nada entra.
	ctx, recorder = skillRequest(t, http.MethodGet, "/api/agent/v1/skills/active?project_id=proj-2", "", "", "org-skills")
	api.activeSkills(ctx)
	if err := json.Unmarshal(recorder.Body.Bytes(), &selection); err != nil {
		t.Fatal(err)
	}
	if len(selection.Selected) != 0 {
		t.Fatalf("contexto de outro projeto não deveria receber a skill: %+v", selection.Selected)
	}

	// Desativação remove a ativação.
	ctx, recorder = skillRequest(t, http.MethodPost, "/api/agent/v1/skills/skill-runtime/deactivate",
		`{"project_id":"proj-1"}`, "skill-runtime", "org-skills")
	api.deactivateSkill(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("desativação falhou: %d %s", recorder.Code, recorder.Body.String())
	}
	ctx, recorder = skillRequest(t, http.MethodGet, "/api/agent/v1/skills/active?project_id=proj-1", "", "", "org-skills")
	api.activeSkills(ctx)
	if err := json.Unmarshal(recorder.Body.Bytes(), &selection); err != nil {
		t.Fatal(err)
	}
	if len(selection.Selected) != 0 {
		t.Fatalf("após desativar, nada deveria ser selecionado: %+v", selection.Selected)
	}
}

func TestSkillActivationRouteValidatesBudgetAndOrganization(t *testing.T) {
	api := newSkillActivationAPI(t)
	ctx, recorder := skillRequest(t, http.MethodGet, "/api/agent/v1/skills/active?token_budget=abc", "", "", "org-skills")
	api.activeSkills(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("orçamento inválido deveria ser 400, veio %d", recorder.Code)
	}

	// Organização de outro tenant não vê a skill de org-skills.
	ctx, recorder = skillRequest(t, http.MethodGet, "/api/agent/v1/skills/active", "", "", "org-outra")
	api.activeSkills(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("seleção de outra org deveria responder 200 vazio, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	var selection agent.SkillSelection
	if err := json.Unmarshal(recorder.Body.Bytes(), &selection); err != nil {
		t.Fatal(err)
	}
	if len(selection.Selected) != 0 {
		t.Fatalf("isolamento entre organizações violado: %+v", selection.Selected)
	}

	// Ativação de skill inexistente responde 404, não 500.
	ctx, recorder = skillRequest(t, http.MethodPost, "/api/agent/v1/skills/nao-existe/activate", "{}", "nao-existe", "org-skills")
	api.activateSkill(ctx)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("skill inexistente deveria ser 404, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
}
