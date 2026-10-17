package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// ESTAGIO 47 — o CAMINHO DE SUCESSO do `restore` (a prova que faltava).
//
// *A primeira tentativa (round 125) falhou por ERRO MEU de escrituracao: mutei a missao em
// memoria sem `PutMission` e construi o `If-Match` da copia mutada, enquanto o servidor lia do
// store ⇒ 409 legitimamente. Este teste faz o que faltava: **grava** a missao com o historico
// antes de pedir o restauro, e **omite `If-Match`** (que e' opt-in) para isolar o caminho de
// sucesso do controlo de versao - cada teste prova uma coisa.*

func TestArtifactRestoreProduzVersaoNova(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspace := t.TempDir()
	caminho := filepath.Join(workspace, "doc.md")
	if err := os.WriteFile(caminho, []byte("conteudo"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := agent.NewMemoryStore()
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{
		Store: store, Planner: agent.RulePlanner{}, WorkspaceRoot: workspace, DataRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{
		Objective: "restauro de versao", Workspace: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	base, err := agent.BuildArtifactManifest(workspace, mission.ID, "step_1", "doc.md", "doc.md")
	if err != nil {
		t.Fatal(err)
	}
	anterior := base
	anterior.Version = 1
	anterior.SHA256 = "sha-da-versao-1"
	corrente := base
	corrente.Version = 2
	corrente.SHA256 = "sha-da-versao-2"
	// ***ACHADO (round 126):*** sem TENANT o dominio recusa com 400 «artifact search requires a
	// tenant» - *o invariante de ACL do Stage 47 a' superficie*. O fixture sem organizacao nao
	// consegue exercer o caminho de sucesso; um artefacto real de missao traz o tenant.
	anterior.TenantID = "org-de-teste"
	corrente.TenantID = "org-de-teste"

	// ***O PASSO QUE FALTAVA:*** o servidor le' a missao do STORE. Mutar a copia em memoria
	// nao chega - se nao gravar, o handler ve' outra coisa (foi o 409 do round 125).
	mission.Artifacts = []agent.ArtifactManifest{anterior, corrente}
	// O STORE tem CAS proprio: escrever sem AVANCAR a versao devolve
	// «mission version conflict» (medido no round 126, linha 60) - *nao era o handler, era o store.*
	mission.Version++
	if err := store.PutMission(mission); err != nil {
		t.Fatal(err)
	}

	api := &agentAPI{runtime: runtime}
	router := gin.New()
	router.POST("/missions/:id/artifacts/:artifact_id/restore", api.artifactRestore)

	requisicao := httptest.NewRequest(http.MethodPost,
		"/missions/"+mission.ID+"/artifacts/"+corrente.ID+"/restore",
		strings.NewReader(`{"version":1,"expected_version":2}`))
	requisicao.Header.Set("Content-Type", "application/json")
	// Sem `If-Match`: o CAS da missao e' opt-in; este teste mede o restauro, nao o controlo de versao.

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, requisicao)

	if recorder.Code != http.StatusOK {
		t.Fatalf("restauro legitimo devia ser 200, veio %d (corpo=%q)", recorder.Code, recorder.Body.String())
	}
	var resposta map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("resposta nao e' JSON: %v", err)
	}
	restaurado, ok := resposta["restored"].(map[string]any)
	if !ok {
		t.Fatalf("a resposta tem de trazer o artefacto restaurado: %v", resposta)
	}
	// *Restaurar NAO sobrescreve: produz a versao SEGUINTE a' corrente (2 -> >2).*
	if versao, _ := restaurado["version"].(float64); int(versao) <= 2 {
		t.Fatalf("restaurar devia produzir versao NOVA (>2), veio %v - *voltar atras nao pode apagar a versao actual*",
			restaurado["version"])
	}
	if resposta["scope"] != "missao do chamador" {
		t.Fatalf("o restauro tem de declarar o escopo, veio %v", resposta["scope"])
	}
}
