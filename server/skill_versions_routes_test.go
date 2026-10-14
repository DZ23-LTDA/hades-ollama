package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// TESTE DE ROTA DO HISTORICO DE VERSOES (E-643).
//
// As garantias do historico estao provadas no store (E-629/E-631). O que se prova AQUI e' o que
// so' a rota pode provar: que o ESCOPO do chamador chega ao store, e que restaurar pela rota NAO
// promove confianca. *As outras rotas de skill tiveram este teste (tenant_sweep_test.go); esta
// nao ia ser a excepcao.*

func novoStoreComDuasVersoes(t *testing.T) *agent.ContextStore {
	t.Helper()
	store, err := agent.NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Registar a MESMA skill duas vezes: o E-629 arquiva a versao que sai.
	if err := store.RegisterSkillForOrganization("org-a", agent.SkillManifest{ID: "skill-x", Version: "1.0", Description: "v1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterSkillForOrganization("org-a", agent.SkillManifest{ID: "skill-x", Version: "2.0", Description: "v2"}); err != nil {
		t.Fatal(err)
	}
	return store
}

func versoesDaResposta(t *testing.T, corpo []byte) []string {
	t.Helper()
	var payload struct {
		Versions []string `json:"versions"`
	}
	if err := json.Unmarshal(corpo, &payload); err != nil {
		t.Fatalf("resposta ilegivel: %v (%s)", err, corpo)
	}
	return payload.Versions
}

func TestHistóricoDeSkillListaParaODonoENadaParaOutroInquilino(t *testing.T) {
	store := novoStoreComDuasVersoes(t)
	api := &agentAPI{context: store, authRequired: true}

	// (1) O DONO ve' a versao arquivada.
	ctx, recorder := skillRequest(t, http.MethodGet, "/api/agent/v1/skill-versions/skill-x", "", "skill-x", "org-a")
	api.skillVersions(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("dono devia receber 200, veio %d (%s)", recorder.Code, recorder.Body.String())
	}
	versoes := versoesDaResposta(t, recorder.Body.Bytes())
	if len(versoes) != 1 || versoes[0] != "1.0" {
		t.Fatalf("o dono devia ver a 1.0 arquivada: %v", versoes)
	}

	// (2) ***OUTRO INQUILINO NAO VE NADA*** — e a resposta e' 200 com lista vazia, nao erro:
	// distinguir "nao existe" de "nao e' sua" seria uma pista sobre a existencia da skill.
	ctxB, recorderB := skillRequest(t, http.MethodGet, "/api/agent/v1/skill-versions/skill-x", "", "skill-x", "org-b")
	api.skillVersions(ctxB)
	if recorderB.Code != http.StatusOK {
		t.Fatalf("o outro inquilino devia receber 200 com lista vazia, veio %d", recorderB.Code)
	}
	if versoesB := versoesDaResposta(t, recorderB.Body.Bytes()); len(versoesB) != 0 {
		t.Fatalf("*** VAZAMENTO: outro inquilino viu o historico ***: %v", versoesB)
	}
}

func TestRestaurarVersaoPelaRotaNaoPromoveConfianca(t *testing.T) {
	store := novoStoreComDuasVersoes(t)
	api := &agentAPI{context: store, authRequired: true}

	// O dono restaura a 1.0 (que fica ACTIVA), e ela volta NAO confiavel.
	ctx, recorder := skillRequest(t, http.MethodPost, "/api/agent/v1/skill-versions/skill-x/restore/1.0", "{}", "skill-x", "org-a")
	ctx.Params = gin.Params{{Key: "id", Value: "skill-x"}, {Key: "version", Value: "1.0"}}
	api.restoreSkillVersion(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("restauro devia dar 200, deu %d (%s)", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Skill agent.SkillManifest `json:"skill"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Skill.Version != "1.0" {
		t.Fatalf("restaurou a versao errada: %+v", payload.Skill)
	}
	if payload.Skill.Trusted {
		t.Fatal("*** a rota PROMOVEU confianca ao restaurar — escalada silenciosa ***")
	}

	// E outro inquilino NAO consegue restaurar a skill alheia.
	ctxB, recorderB := skillRequest(t, http.MethodPost, "/api/agent/v1/skill-versions/skill-x/restore/2.0", "{}", "skill-x", "org-b")
	ctxB.Params = gin.Params{{Key: "id", Value: "skill-x"}, {Key: "version", Value: "2.0"}}
	api.restoreSkillVersion(ctxB)
	if recorderB.Code == http.StatusOK {
		t.Fatalf("*** VAZAMENTO: outro inquilino restaurou a skill alheia ***: %s", recorderB.Body.String())
	}
}

// Versao inexistente e' recusada — e nao um restauro silencioso de nada.
func TestRestaurarVersaoInexistentePelaRotaEhRecusada(t *testing.T) {
	store := novoStoreComDuasVersoes(t)
	api := &agentAPI{context: store, authRequired: true}
	ctx, recorder := skillRequest(t, http.MethodPost, "/api/agent/v1/skill-versions/skill-x/restore/9.9", "{}", "skill-x", "org-a")
	ctx.Params = gin.Params{{Key: "id", Value: "skill-x"}, {Key: "version", Value: "9.9"}}
	api.restoreSkillVersion(ctx)
	if recorder.Code == http.StatusOK {
		t.Fatalf("versao inexistente devia ser recusada: %s", recorder.Body.String())
	}
}
