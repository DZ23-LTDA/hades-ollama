package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// ESTAGIO 47 — o teste de HANDLER da rota de leitura de versoes.
//
// *Os testes puros do round 119 provam o mapa de erros e a busca do manifesto; este prova o
// que so' a via HTTP diz: que a rota responde 404 a um artefacto que nao e' desta missao e que,
// quando responde 200, diz QUAL e' o escopo do historico (`scope`) em vez de o prometer global.*

func montarMissaoComArtefacto(t *testing.T) (*agent.Runtime, agent.Mission, agent.ArtifactManifest) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	workspace := t.TempDir()
	artifactPath := filepath.Join(workspace, "doc.md")
	if err := os.WriteFile(artifactPath, []byte("conteudo do artefacto"), 0o600); err != nil {
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
		Objective: "versoes de artefacto", Workspace: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := agent.BuildArtifactManifest(workspace, mission.ID, "step_1", "doc.md", "doc.md")
	if err != nil {
		t.Fatal(err)
	}
	mission.Artifacts = []agent.ArtifactManifest{manifest}
	mission.Version++
	if err := store.PutMission(mission); err != nil {
		t.Fatal(err)
	}
	return runtime, mission, manifest
}

func TestArtifactVersionsResponde404ParaArtefactoAusente(t *testing.T) {
	runtime, mission, _ := montarMissaoComArtefacto(t)
	api := &agentAPI{runtime: runtime}
	router := gin.New()
	router.GET("/missions/:id/artifacts/:artifact_id/versions", api.artifactVersions)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/missions/"+mission.ID+"/artifacts/art_que_nao_existe/versions", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("artefacto ausente devia ser 404, veio %d (corpo=%q) - *um 200 aqui seria um historico inventado*",
			recorder.Code, recorder.Body.String())
	}
}

func TestArtifactVersionsDizOEscopoDoHistorico(t *testing.T) {
	runtime, mission, manifest := montarMissaoComArtefacto(t)
	api := &agentAPI{runtime: runtime}
	router := gin.New()
	router.GET("/missions/:id/artifacts/:artifact_id/versions", api.artifactVersions)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/missions/"+mission.ID+"/artifacts/"+manifest.ID+"/versions", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("artefacto real devia ser 200, veio %d (corpo=%q)", recorder.Code, recorder.Body.String())
	}
	var corpo map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta nao e' JSON: %v", err)
	}
	if corpo["mission_id"] != mission.ID {
		t.Fatalf("mission_id devia ser %q, veio %v", mission.ID, corpo["mission_id"])
	}
	// *O historico de versoes e' POR MISSAO (o indice e' `mission.Artifacts`); a resposta tem
	// de o declarar, senao o cliente le^ um historico parcial como se fosse global.*
	if corpo["scope"] != "missao do chamador" {
		t.Fatalf("scope devia declarar o limite real do historico, veio %v", corpo["scope"])
	}
	if _, existe := corpo["versions"]; !existe {
		t.Fatalf("a resposta tem de trazer `versions` (mesmo vazio): %v", corpo)
	}
}
