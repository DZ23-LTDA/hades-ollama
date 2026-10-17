package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// ESTAGIO 47 — a ESCRITA (`restore`): as duas recusas que a rota TEM de fazer.
//
// *O CAS da missao e' opt-in pelo cabecalho `If-Match` (medido em `missionVersionMatches`:
// sem cabecalho passa, com versao diferente devolve 409). Estes dois testes fixam esse
// comportamento na rota nova - sem eles, ninguem saberia se o `restore` respeita o If-Match
// ou se o ignora em silencio.*

func TestArtifactRestoreRecusaVersaoDeMissaoDesactualizada(t *testing.T) {
	runtime, mission, manifest := montarMissaoComArtefacto(t)
	api := &agentAPI{runtime: runtime}
	router := gin.New()
	router.POST("/missions/:id/artifacts/:artifact_id/restore", api.artifactRestore)

	corpo := strings.NewReader(`{"version":1,"expected_version":1}`)
	requisicao := httptest.NewRequest(http.MethodPost,
		"/missions/"+mission.ID+"/artifacts/"+manifest.ID+"/restore", corpo)
	requisicao.Header.Set("Content-Type", "application/json")
	requisicao.Header.Set("If-Match", "999999") // versao que NAO e' a da missao

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, requisicao)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("If-Match desactualizado devia ser 409, veio %d (corpo=%q) - *um 200 aqui seria escrever sobre uma versao que o cliente nao viu*",
			recorder.Code, recorder.Body.String())
	}
	var resposta map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("resposta nao e' JSON: %v", err)
	}
	if resposta["error"] != "mission version conflict" {
		t.Fatalf("o erro devia nomear o conflito de versao da missao, veio %v", resposta["error"])
	}
	if resposta["mission_id"] != mission.ID {
		t.Fatalf("o conflito tem de dizer QUAL missao, veio %v", resposta["mission_id"])
	}
}

func TestArtifactRestoreResponde404ParaArtefactoAusente(t *testing.T) {
	runtime, mission, _ := montarMissaoComArtefacto(t)
	api := &agentAPI{runtime: runtime}
	router := gin.New()
	router.POST("/missions/:id/artifacts/:artifact_id/restore", api.artifactRestore)

	// Sem `If-Match`: o CAS da missao passa (e' opt-in). O que tem de travar aqui e' o
	// artefacto - *restaurar um artefacto que nao e' desta missao nao pode ser 200 nem 5xx.*
	requisicao := httptest.NewRequest(http.MethodPost,
		"/missions/"+mission.ID+"/artifacts/art_que_nao_existe/restore",
		strings.NewReader(`{"version":1,"expected_version":1}`))
	requisicao.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, requisicao)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("artefacto ausente devia ser 404, veio %d (corpo=%q)", recorder.Code, recorder.Body.String())
	}
}
