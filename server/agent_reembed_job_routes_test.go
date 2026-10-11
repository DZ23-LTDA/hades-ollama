package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// extrairCampoJSON lê um campo string do corpo da resposta sem depender de uma struct completa.
func extrairCampoJSON(t *testing.T, corpo, campo string) string {
	t.Helper()
	var dados map[string]any
	if err := json.Unmarshal([]byte(corpo), &dados); err != nil {
		t.Fatalf("corpo não é JSON: %v (%s)", err, corpo)
	}
	valor, ok := dados[campo].(string)
	if !ok || valor == "" {
		t.Fatalf("campo %q ausente no corpo: %s", campo, corpo)
	}
	return valor
}

// RE-EMBED ASSÍNCRONO pela rota (estágio 35): estimativa, job com progresso e cancelamento.
//
// O limite declarado em E-159 era "re-embed SÍNCRONO, 200 por chamada e por projeto, sem estimativa prévia do
// volume". Aqui a operação vira job observável — e a rota precisa dizer a verdade em cada passo.
func TestAgentReembedJobRouteRunsToCompletion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, projectA, _ := newObjectScopeTestAPI(t)
	// As memórias nascem SEM embedder (o estado de quem trocou de modelo depois de acumular conhecimento).
	for i := range 60 {
		if _, err := api.context.AddMemory(agent.Memory{ProjectID: projectA.ID, Kind: "fact", Content: fmt.Sprintf("memória %d", i), Source: "manual"}); err != nil {
			t.Fatal(err)
		}
	}
	api.context.SetEmbedder(stubVetorial{})

	// 1. ESTIMATIVA antes de começar, com o modelo de destino.
	ctx, recorder := scopedObjectRequest(t, http.MethodGet, "/api/agent/projects/"+projectA.ID+"/memories/reembed/estimate", "", "org_a")
	ctx.Params = gin.Params{{Key: "id", Value: projectA.ID}}
	api.reembedEstimate(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("estimativa precisa responder 200, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"pendentes":`) || !strings.Contains(recorder.Body.String(), `"modelo_atual":"modelo-da-rota"`) {
		t.Fatalf("a estimativa precisa dizer quanto falta e para qual modelo: %s", recorder.Body.String())
	}

	// 2. JOB aceito com 202 e id próprio.
	ctx, recorder = scopedObjectRequest(t, http.MethodPost, "/api/agent/projects/"+projectA.ID+"/memories/reembed/jobs", "", "org_a")
	ctx.Params = gin.Params{{Key: "id", Value: projectA.ID}}
	api.startReembedJob(ctx)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("iniciar job precisa responder 202, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	corpo := recorder.Body.String()
	if !strings.Contains(corpo, `"state":"running"`) && !strings.Contains(corpo, `"state":"done"`) {
		t.Fatalf("a resposta precisa trazer o estado do job: %s", corpo)
	}
	jobID := extrairCampoJSON(t, corpo, "id")

	// 3. PROGRESSO consultável até concluir.
	final := ""
	limite := time.Now().Add(10 * time.Second)
	for time.Now().Before(limite) {
		ctx, recorder = scopedObjectRequest(t, http.MethodGet, "/api/agent/projects/"+projectA.ID+"/memories/reembed/jobs/"+jobID, "", "org_a")
		ctx.Params = gin.Params{{Key: "id", Value: projectA.ID}, {Key: "job_id", Value: jobID}}
		api.reembedJobStatus(ctx)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status do job precisa responder 200, veio %d: %s", recorder.Code, recorder.Body.String())
		}
		final = recorder.Body.String()
		if strings.Contains(final, `"state":"done"`) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !strings.Contains(final, `"state":"done"`) {
		t.Fatalf("o job precisa terminar: %s", final)
	}
	// Contadores conferem com o trabalho pedido — nada de "concluído" sem número.
	if !strings.Contains(final, `"reembedded":60`) || !strings.Contains(final, `"remaining":0`) {
		t.Fatalf("o job precisa relatar o que fez: %s", final)
	}
	if api.context.EstimateReembed(projectA.ID) != 0 {
		t.Fatal("depois do job não pode haver pendência")
	}
}

func TestAgentReembedJobRouteRefusesSecondJobWith409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, projectA, _ := newObjectScopeTestAPI(t)
	for i := range 200 {
		if _, err := api.context.AddMemory(agent.Memory{ProjectID: projectA.ID, Kind: "fact", Content: fmt.Sprintf("memória %d", i), Source: "manual"}); err != nil {
			t.Fatal(err)
		}
	}
	api.context.SetEmbedder(stubVetorial{})

	ctx, _ := scopedObjectRequest(t, http.MethodPost, "/api/agent/projects/"+projectA.ID+"/memories/reembed/jobs", "", "org_a")
	ctx.Params = gin.Params{{Key: "id", Value: projectA.ID}}
	api.startReembedJob(ctx)

	// Segundo pedido no mesmo projeto: 409 (dois jobs competiriam pelo mesmo arquivo).
	ctx, recorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/projects/"+projectA.ID+"/memories/reembed/jobs", "", "org_a")
	ctx.Params = gin.Params{{Key: "id", Value: projectA.ID}}
	api.startReembedJob(ctx)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("segundo job precisa responder 409, veio %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestAgentReembedJobRouteDoesNotRevealAnotherProjectsJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, projectA, projectB := newObjectScopeTestAPI(t)
	if _, err := api.context.AddMemory(agent.Memory{ProjectID: projectA.ID, Kind: "fact", Content: "segredo da org A", Source: "manual"}); err != nil {
		t.Fatal(err)
	}
	api.context.SetEmbedder(stubVetorial{})
	ctx, recorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/projects/"+projectA.ID+"/memories/reembed/jobs", "", "org_a")
	ctx.Params = gin.Params{{Key: "id", Value: projectA.ID}}
	api.startReembedJob(ctx)
	jobID := extrairCampoJSON(t, recorder.Body.String(), "id")

	// Mesma organização, OUTRO projeto: o job não pertence a ele e não pode ser consultado por ali.
	ctx, recorder = scopedObjectRequest(t, http.MethodGet, "/api/agent/projects/"+projectB.ID+"/memories/reembed/jobs/"+jobID, "", "org_a")
	ctx.Params = gin.Params{{Key: "id", Value: projectB.ID}, {Key: "job_id", Value: jobID}}
	api.reembedJobStatus(ctx)
	// A recusa acontece na checagem de PROJETO (403) antes da checagem de dono do job (404): defesa em
	// profundidade. O que importa é: não entrega o job e não vaza conteúdo.
	if recorder.Code == http.StatusOK {
		t.Fatalf("job de outro projeto não pode ser consultado: %s", recorder.Body.String())
	}
	if recorder.Code != http.StatusForbidden && recorder.Code != http.StatusNotFound {
		t.Fatalf("a recusa precisa ser 403 (projeto) ou 404 (job), veio %d: %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "segredo da org A") {
		t.Fatalf("a recusa não pode devolver conteúdo: %s", recorder.Body.String())
	}
}
