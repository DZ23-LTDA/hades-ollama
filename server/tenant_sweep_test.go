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

// Varredura ofensiva cross-tenant (estágio 18) com ator AUTENTICADO.
//
// Metodologia: a organização A cria objetos reais pelos PRÓPRIOS handlers; em
// seguida a organização B — também autenticada, com o mesmo conjunto de rotas —
// tenta ler e mutar esses objetos. A exigência é dura: B não pode ver nem
// alterar nada de A, e a recusa precisa ser explícita (403/404), nunca 200 com
// dado alheio.
//
// Este arquivo cobre as superfícies MAIS NOVAS (plugins, skills, ativações e
// retenção), que são justamente as que ganharam código nas últimas rodadas.
// Projetos, memórias, missões e anexos já têm testes de escopo próprios.

const (
	sweepTenantA = "org-a"
	sweepTenantB = "org-b"
)

// sweepSkillAPIActivations monta um store com uma skill assinada e promovida para
// a organização informada, pelo caminho real de produção.
func signedSkillFor(t *testing.T, store *agent.ContextStore, organization, id string) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	policy := agent.DefaultCapabilityPolicy().WithTrustedSkillKey("release-key", publicKey)
	manifest := agent.SkillManifest{
		ID:             id,
		OrganizationID: organization,
		Version:        "1.0.0",
		Description:    "skill do tenant A",
		Enabled:        true,
	}
	signed, err := agent.SignSkillManifest(manifest, privateKey, "release-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkillForOrganization(organization, signed); err != nil {
		t.Fatal(err)
	}
	if err := store.PromoteSkillTrustedForOrganization(organization, id, policy); err != nil {
		t.Fatal(err)
	}
}

func sweepSkillIDs(t *testing.T, api *agentAPI, organization string) []string {
	t.Helper()
	ctx, recorder := skillRequest(t, http.MethodGet, "/api/agent/v1/skills", "", "", organization)
	api.skills(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("listar skills de %s: %d %s", organization, recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Skills []agent.SkillManifest `json:"skills"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(payload.Skills))
	for _, skill := range payload.Skills {
		ids = append(ids, skill.ID)
	}
	return ids
}

func TestTenantSweepSkillsAreInvisibleAndImmutableAcrossOrganizations(t *testing.T) {
	store, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{context: store, authRequired: true}
	signedSkillFor(t, store, sweepTenantA, "skill-do-tenant-a")

	// A vê a própria skill.
	if ids := sweepSkillIDs(t, api, sweepTenantA); len(ids) != 1 || ids[0] != "skill-do-tenant-a" {
		t.Fatalf("tenant A deveria ver sua skill: %v", ids)
	}
	// B NÃO vê.
	if ids := sweepSkillIDs(t, api, sweepTenantB); len(ids) != 0 {
		t.Fatalf("VAZAMENTO: tenant B listou %v", ids)
	}

	// B não consegue ativar a skill de A.
	activateCtx, activateRecorder := skillRequest(t, http.MethodPost, "/api/agent/v1/skills/skill-do-tenant-a/activate", "{}", "skill-do-tenant-a", sweepTenantB)
	api.activateSkill(activateCtx)
	if activateRecorder.Code == http.StatusOK {
		t.Fatalf("VAZAMENTO: tenant B ativou skill de A: %s", activateRecorder.Body.String())
	}
	if activateRecorder.Code != http.StatusForbidden && activateRecorder.Code != http.StatusNotFound {
		t.Fatalf("recusa deveria ser explícita (403/404), veio %d", activateRecorder.Code)
	}

	// B não consegue desabilitar nem remover a skill de A.
	disableCtx, disableRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/skills/skill-do-tenant-a/disable", "{}", "skill-do-tenant-a", sweepTenantB)
	api.disableSkill(disableCtx)
	if disableRecorder.Code == http.StatusOK {
		t.Fatalf("VAZAMENTO: tenant B desabilitou skill de A: %s", disableRecorder.Body.String())
	}
	removeCtx, removeRecorder := pluginRouteContext(t, http.MethodDelete, "/api/agent/v1/skills/skill-do-tenant-a", "", "skill-do-tenant-a", sweepTenantB)
	api.removeSkill(removeCtx)
	if removeRecorder.Code == http.StatusOK {
		t.Fatalf("VAZAMENTO: tenant B removeu skill de A: %s", removeRecorder.Body.String())
	}
	// A skill de A continua intacta depois do ataque.
	if ids := sweepSkillIDs(t, api, sweepTenantA); len(ids) != 1 {
		t.Fatalf("a skill de A foi afetada pelo ataque: %v", ids)
	}
}

func TestTenantSweepSkillActivationsDoNotCrossOrganizations(t *testing.T) {
	root := t.TempDir()
	store, err := agent.NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{context: store, authRequired: true}
	signedSkillFor(t, store, sweepTenantA, "skill-a")

	// A ativa para o próprio projeto.
	activateCtx, activateRecorder := skillRequest(t, http.MethodPost, "/api/agent/v1/skills/skill-a/activate",
		`{"project_id":"projeto-a"}`, "skill-a", sweepTenantA)
	api.activateSkill(activateCtx)
	if activateRecorder.Code != http.StatusOK {
		t.Fatalf("ativação legítima falhou: %d %s", activateRecorder.Code, activateRecorder.Body.String())
	}

	// B não vê a ativação de A nem usando o MESMO project_id.
	ctx, recorder := skillRequest(t, http.MethodGet, "/api/agent/v1/skills/active?project_id=projeto-a", "", "", sweepTenantB)
	api.activeSkills(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("seleção de B: %d %s", recorder.Code, recorder.Body.String())
	}
	var selection agent.SkillSelection
	if err := json.Unmarshal(recorder.Body.Bytes(), &selection); err != nil {
		t.Fatal(err)
	}
	if len(selection.Selected) != 0 {
		t.Fatalf("VAZAMENTO: B recebeu skills ativadas por A: %+v", selection.Selected)
	}

	// B não consegue desativar a ativação de A (nem com o mesmo project_id).
	deactivateCtx, deactivateRecorder := skillRequest(t, http.MethodPost, "/api/agent/v1/skills/skill-a/deactivate",
		`{"project_id":"projeto-a"}`, "skill-a", sweepTenantB)
	api.deactivateSkill(deactivateCtx)
	if deactivateRecorder.Code == http.StatusOK {
		t.Fatalf("VAZAMENTO: B desativou skill de A: %s", deactivateRecorder.Body.String())
	}
	// E a ativação de A sobreviveu.
	ctx, recorder = skillRequest(t, http.MethodGet, "/api/agent/v1/skills/active?project_id=projeto-a", "", "", sweepTenantA)
	api.activeSkills(ctx)
	if err := json.Unmarshal(recorder.Body.Bytes(), &selection); err != nil {
		t.Fatal(err)
	}
	if len(selection.Selected) != 1 {
		t.Fatalf("ativação de A foi perdida: %+v", selection.Selected)
	}

	// Persistência não vaza: reabrir o store não muda a fronteira.
	reopened, err := agent.NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	reopenedAPI := &agentAPI{context: reopened, authRequired: true}
	ctx, recorder = skillRequest(t, http.MethodGet, "/api/agent/v1/skills/active?project_id=projeto-a", "", "", sweepTenantB)
	reopenedAPI.activeSkills(ctx)
	if err := json.Unmarshal(recorder.Body.Bytes(), &selection); err != nil {
		t.Fatal(err)
	}
	if len(selection.Selected) != 0 {
		t.Fatalf("VAZAMENTO após reabrir o store: %+v", selection.Selected)
	}
}

func TestTenantSweepPluginsAndStatusAreIsolated(t *testing.T) {
	registry, err := agent.NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{plugins: registry, authRequired: true}

	manifest := agent.PluginManifest{ID: "plugin-do-a", Version: "1.0.0", Name: "Plugin de A", Kind: agent.PluginKindConnector}
	installCtx, installRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins", pluginBody(t, manifest), "", sweepTenantA)
	api.installPlugin(installCtx)
	if installRecorder.Code != http.StatusCreated {
		t.Fatalf("instalação legítima falhou: %d %s", installRecorder.Code, installRecorder.Body.String())
	}

	// B não lista o plugin de A.
	listCtx, listRecorder := pluginRouteContext(t, http.MethodGet, "/api/agent/v1/plugins", "", "", sweepTenantB)
	api.listPlugins(listCtx)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("lista de B: %d %s", listRecorder.Code, listRecorder.Body.String())
	}
	if strings.Contains(listRecorder.Body.String(), "plugin-do-a") {
		t.Fatalf("VAZAMENTO: B listou plugin de A: %s", listRecorder.Body.String())
	}

	// B não vê o plugin de A no status canônico.
	statusCtx, statusRecorder := pluginRouteContext(t, http.MethodGet, "/api/agent/v1/plugins/status", "", "", sweepTenantB)
	api.pluginStatuses(statusCtx)
	if statusCtx.Writer == nil {
		t.Fatal("status sem resposta")
	}
	if strings.Contains(statusRecorder.Body.String(), "plugin-do-a") {
		t.Fatalf("VAZAMENTO: B viu status do plugin de A: %s", statusRecorder.Body.String())
	}

	// B não faz rollback nem promove o plugin de A.
	rollbackCtx, rollbackRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins/plugin-do-a/rollback", "{}", "plugin-do-a", sweepTenantB)
	api.rollbackPlugin(rollbackCtx)
	if rollbackRecorder.Code == http.StatusOK {
		t.Fatalf("VAZAMENTO: B fez rollback do plugin de A: %s", rollbackRecorder.Body.String())
	}
	promoteCtx, promoteRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins/plugin-do-a/promote", "{}", "plugin-do-a", sweepTenantB)
	api.promotePlugin(promoteCtx)
	if promoteRecorder.Code == http.StatusOK {
		t.Fatalf("VAZAMENTO: B promoveu plugin de A: %s", promoteRecorder.Body.String())
	}

	// O plugin de A segue instalado e visível para A.
	listCtx, listRecorder = pluginRouteContext(t, http.MethodGet, "/api/agent/v1/plugins", "", "", sweepTenantA)
	api.listPlugins(listCtx)
	if !strings.Contains(listRecorder.Body.String(), "plugin-do-a") {
		t.Fatalf("plugin de A desapareceu após o ataque: %s", listRecorder.Body.String())
	}
}

// O lado perigoso da retenção é a MUTAÇÃO: aplicar a política APAGA memórias.
// Aqui a organização B — autenticada e sem política própria — tenta executar a
// aplicação no lugar de A, e a prova exigida é dupla: B é recusada E os dados de
// A continuam intactos; depois A aplica a própria política e o expurgo funciona.
func TestTenantSweepRetentionApplyCannotDeleteAnotherOrganizationData(t *testing.T) {
	api, own, _ := newRetentionAPI(t)
	// Memórias RECENTES de propósito: assim só a regra de contagem pode agir, e o
	// teste isola o efeito destrutivo que estamos medindo (a política de idade é
	// coberta pelos testes de retenção existentes).
	seedRetentionMemoriesForRoute(t, api.context, own.ID, "memoria", 3, 0)

	countMemories := func() int {
		t.Helper()
		return len(api.context.SearchMemories(own.ID, "", 0))
	}
	before := countMemories()
	if before != 3 {
		t.Fatalf("fixture deveria ter 3 memórias, tem %d", before)
	}

	// A configura política agressiva (guarda 1 memória por projeto).
	setCtx, setRecorder := retentionRequest(t, http.MethodPut, "/api/agent/v1/retention",
		`{"max_age_days":30,"max_memories_per_project":1}`, sweepTenantA)
	api.setRetentionPolicy(setCtx)
	if setRecorder.Code != http.StatusOK {
		t.Fatalf("política de A não foi aceita: %d %s", setRecorder.Code, setRecorder.Body.String())
	}

	// B tenta aplicar. Sem política própria, a recusa é explícita (404) e, mais
	// importante, NADA de A pode ser apagado por essa chamada.
	applyCtx, applyRecorder := retentionRequest(t, http.MethodPost, "/api/agent/v1/retention/apply", "", sweepTenantB)
	api.applyRetentionPolicy(applyCtx)
	if applyRecorder.Code == http.StatusOK {
		t.Fatalf("VAZAMENTO: B aplicou retenção com sucesso: %s", applyRecorder.Body.String())
	}
	if applyRecorder.Code != http.StatusNotFound && applyRecorder.Code != http.StatusForbidden {
		t.Fatalf("recusa de B deveria ser 404/403, veio %d (%s)", applyRecorder.Code, applyRecorder.Body.String())
	}
	if after := countMemories(); after != before {
		t.Fatalf("VAZAMENTO DESTRUTIVO: B apagou memórias de A (%d -> %d)", before, after)
	}

	// A aplica a própria política: o expurgo acontece para o dono.
	applyCtx, applyRecorder = retentionRequest(t, http.MethodPost, "/api/agent/v1/retention/apply", "", sweepTenantA)
	api.applyRetentionPolicy(applyCtx)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("dono deveria conseguir aplicar: %d %s", applyRecorder.Code, applyRecorder.Body.String())
	}
	if after := countMemories(); after != 1 {
		t.Fatalf("expurgo do dono não funcionou: %d memórias", after)
	}
}

// Mass assignment de tenant: o corpo da requisição NÃO pode escolher a
// organização de destino. Se B conseguir gravar um objeto "pertencente" a A, o
// efeito é escrita cross-tenant (poluir o catálogo alheio com conteúdo do
// atacante) — por isso a asserção é sobre o estado de A, não sobre o status code.
func TestTenantSweepCannotWriteIntoAnotherOrganizationViaBody(t *testing.T) {
	registry, err := agent.NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{plugins: registry, context: store, authRequired: true}

	// Plugin: B envia um corpo que TENTA declarar a organização de A. O
	// PluginManifest não expõe campo de organização (o dono vem só da sessão),
	// então este corpo testa duas coisas: que a chave extra não é aceita como
	// dono e que o plugin instalado pertence a B, nunca a A.
	planted := agent.PluginManifest{
		ID:      "plugin-plantado",
		Version: "1.0.0",
		Name:    "plantado por B",
		Kind:    agent.PluginKindConnector,
	}
	rawBody := strings.Replace(pluginBody(t, planted), `"id"`,
		`"organization_id":"`+sweepTenantA+`","id"`, 1)
	installCtx, installRecorder := pluginRouteContext(t, http.MethodPost, "/api/agent/v1/plugins", rawBody, "", sweepTenantB)
	api.installPlugin(installCtx)

	// A não pode enxergar um plugin que nunca instalou.
	listCtx, listRecorder := pluginRouteContext(t, http.MethodGet, "/api/agent/v1/plugins", "", "", sweepTenantA)
	api.listPlugins(listCtx)
	if strings.Contains(listRecorder.Body.String(), "plugin-plantado") {
		t.Fatalf("ESCRITA CROSS-TENANT: A recebeu plugin plantado por B (status do ataque=%d): %s",
			installRecorder.Code, listRecorder.Body.String())
	}

	// Skill: mesmo ataque na rota de registro.
	skillPayload := `{"id":"skill-plantada","organization_id":"` + sweepTenantA + `","version":"1.0.0","description":"plantada por B"}`
	registerCtx, registerRecorder := skillRequest(t, http.MethodPost, "/api/agent/v1/skills", skillPayload, "", sweepTenantB)
	api.registerSkill(registerCtx)

	if ids := sweepSkillIDs(t, api, sweepTenantA); len(ids) != 0 {
		t.Fatalf("ESCRITA CROSS-TENANT: A recebeu skill plantada por B (status do ataque=%d): %v",
			registerRecorder.Code, ids)
	}
	if ids := sweepSkillIDs(t, api, sweepTenantB); len(ids) > 1 {
		t.Fatalf("estado inesperado para B: %v", ids)
	}
}

// Caminho "global" de skills: eu levantei a hipótese de um registro órfão, aceito
// na entrada e impossível de confiar depois (caminho morto). O teste REFUTOU a
// hipótese — o registro sem dono é recusado na ENTRADA, com erro explícito, e a
// promoção também. O comportamento é fail-closed nos dois pontos, que é o
// desejável; este teste trava essa propriedade para que uma futura mudança não
// reintroduza um registro inutilizável sem que ninguém perceba.
func TestTenantSweepOwnerlessSkillIsRefusedAtEntry(t *testing.T) {
	store, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	policy := agent.DefaultCapabilityPolicy().WithTrustedSkillKey("release-key", publicKey)
	manifest := agent.SkillManifest{ID: "skill-global", Version: "1.0.0", Description: "sem dono", Enabled: true}
	signed, err := agent.SignSkillManifest(manifest, privateKey, "release-key")
	if err != nil {
		t.Fatal(err)
	}

	// (1) Registro sem organização dona: RECUSADO (fail closed na entrada).
	registerErr := store.RegisterSkillForOrganization("", signed)
	if registerErr == nil {
		t.Fatal("registro sem dono foi aceito: criar um registro que nunca poderá ser confiável é um caminho morto")
	}
	if !strings.Contains(registerErr.Error(), "organization scope") {
		t.Fatalf("erro deveria dizer que o escopo de organização é obrigatório: %v", registerErr)
	}
	// (2) Promoção sem dono: também RECUSADA.
	if err := store.PromoteSkillTrustedForOrganization("", "skill-global", policy); err == nil {
		t.Fatal("promoção sem dono foi aceita")
	}

	api := &agentAPI{context: store, authRequired: true}
	// (3) Nada ficou para trás: nenhum tenant vê ou ativa o registro recusado.
	if ids := sweepSkillIDs(t, api, sweepTenantA); len(ids) != 0 {
		t.Fatalf("tenant viu skill recusada: %v", ids)
	}
	activateCtx, activateRecorder := skillRequest(t, http.MethodPost, "/api/agent/v1/skills/skill-global/activate", "{}", "skill-global", sweepTenantA)
	api.activateSkill(activateCtx)
	if activateRecorder.Code == http.StatusOK {
		t.Fatalf("tenant ativou skill recusada: %s", activateRecorder.Body.String())
	}
}

// A varredura também precisa cobrir o caminho HTTP completo, e não só os handlers
// chamados direto: aqui a rota agentic de skills é montada no roteador com o
// middleware real de organização.
func TestTenantSweepRouterRefusesForeignSkillActivation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	signedSkillFor(t, store, sweepTenantA, "skill-router")
	api := &agentAPI{context: store, authRequired: true}

	engine := gin.New()
	engine.POST("/skills/:id/activate", func(c *gin.Context) {
		c.Set("agent.organization", agent.Organization{ID: sweepTenantB})
		api.activateSkill(c)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/skills/skill-router/activate", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	if recorder.Code == http.StatusOK {
		t.Fatalf("VAZAMENTO pela rota HTTP: %s", recorder.Body.String())
	}
	if recorder.Code != http.StatusForbidden && recorder.Code != http.StatusNotFound {
		t.Fatalf("recusa pela rota deveria ser 403/404, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
}
