package server

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// stubVetorial é um embedder determinístico para a rota de re-embed: identifica o espaço vetorial (o carimbo
// depende disso) e devolve sempre o mesmo vetor.
type stubVetorial struct{}

func (stubVetorial) Embed(_ context.Context, _ string) ([]float32, error) {
	return []float32{1, 0, 0}, nil
}

func (stubVetorial) EmbeddingModel() string { return "modelo-da-rota" }

// A ROTA de re-embed é o gatilho de operador: sem ela, a busca só sabia DECLARAR que parte do conhecimento
// ficou fora do espaço vetorial, sem remédio acionável.
func TestAgentMemoryReembedRouteRecalculatesVectors(t *testing.T) {
	api, projectA, _ := newObjectScopeTestAPI(t)
	// As memórias são gravadas ANTES do embedder: sem embedder elas ficam SEM vetor, que é exatamente o
	// estado de quem trocou o modelo de embedding depois de já ter conhecimento acumulado.
	for _, conteudo := range []string{"política de troca", "prazo de entrega"} {
		if _, err := api.context.AddMemory(agent.Memory{ProjectID: projectA.ID, Kind: "fact", Content: conteudo, Source: "manual"}); err != nil {
			t.Fatal(err)
		}
	}
	api.context.SetEmbedder(stubVetorial{})

	ctx, recorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/projects/"+projectA.ID+"/memories/reembed", "", "org_a")
	ctx.Params = gin.Params{{Key: "id", Value: projectA.ID}}
	api.reembedProjectMemories(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("re-embed precisa responder 200, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	corpo := recorder.Body.String()
	for _, esperado := range []string{`"reembedded":2`, `"failed":0`, `"remaining":0`, `"limit":200`} {
		if !strings.Contains(corpo, esperado) {
			t.Fatalf("a resposta precisa trazer %s: %s", esperado, corpo)
		}
	}
	// E o conhecimento saiu do limbo: nada mais fora do espaço.
	if fora := api.context.CountMemoriesOutsideVectorSpace(projectA.ID); fora != 0 {
		t.Fatalf("depois do re-embed nada pode ficar fora do espaço: %d", fora)
	}
}

func TestAgentMemoryReembedRouteRespectsTheLimit(t *testing.T) {
	api, projectA, _ := newObjectScopeTestAPI(t)
	for _, conteudo := range []string{"um", "dois", "três"} {
		if _, err := api.context.AddMemory(agent.Memory{ProjectID: projectA.ID, Kind: "fact", Content: conteudo, Source: "manual"}); err != nil {
			t.Fatal(err)
		}
	}
	api.context.SetEmbedder(stubVetorial{})

	ctx, recorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/projects/"+projectA.ID+"/memories/reembed?limit=1", "", "org_a")
	ctx.Params = gin.Params{{Key: "id", Value: projectA.ID}}
	api.reembedProjectMemories(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("re-embed precisa responder 200, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	corpo := recorder.Body.String()
	// Operação LIMITADA precisa dizer o que ficou: sem `remaining`, o operador acharia que terminou.
	for _, esperado := range []string{`"reembedded":1`, `"remaining":2`, `"limit":1`} {
		if !strings.Contains(corpo, esperado) {
			t.Fatalf("a resposta precisa trazer %s: %s", esperado, corpo)
		}
	}
	// E os vetores ainda não recalculados continuam fora do espaço (nada foi inventado).
	if fora := api.context.CountMemoriesOutsideVectorSpace(projectA.ID); fora != 2 {
		t.Fatalf("o limite precisa deixar o resto fora do espaço: %d", fora)
	}
}

func TestAgentMemoryReembedRouteIsScopedToTheProjectOrganization(t *testing.T) {
	api, projectA, _ := newObjectScopeTestAPI(t)
	api.context.SetEmbedder(stubVetorial{})
	if _, err := api.context.AddMemory(agent.Memory{ProjectID: projectA.ID, Kind: "fact", Content: "segredo da org A", Source: "manual"}); err != nil {
		t.Fatal(err)
	}

	// A organização B pede re-embed no projeto da A: a checagem de projeto tem de recusar.
	ctx, recorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/projects/"+projectA.ID+"/memories/reembed", "", "org_b")
	ctx.Params = gin.Params{{Key: "id", Value: projectA.ID}}
	api.reembedProjectMemories(ctx)

	if recorder.Code == http.StatusOK {
		t.Fatalf("outra organização não pode operar o projeto alheio: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "segredo da org A") {
		t.Fatalf("a recusa não pode devolver conteúdo do projeto: %s", recorder.Body.String())
	}
}

func TestAgentMemoryReembedRouteWithoutEmbedderIsDeclared(t *testing.T) {
	api, projectA, _ := newObjectScopeTestAPI(t)
	// Sem embedder configurado não há espaço de destino: a rota precisa DIZER isso, não fingir sucesso.
	ctx, recorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/projects/"+projectA.ID+"/memories/reembed", "", "org_a")
	ctx.Params = gin.Params{{Key: "id", Value: projectA.ID}}
	api.reembedProjectMemories(ctx)

	if recorder.Code == http.StatusOK {
		t.Fatalf("sem embedder não há re-embed a declarar como feito: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "embedder") {
		t.Fatalf("a falha precisa nomear o que falta: %s", recorder.Body.String())
	}
}
