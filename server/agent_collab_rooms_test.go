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

// Complemento 48 / FASE 2 — os testes das ROTAS da sala e dos membros.
//
// *Sem estes testes as rotas eram `IMPLEMENTED_UNTESTED`: compilavam, passavam o lint e não quebravam nada -
// **e ninguém provava que faziam o que dizem**.* Aqui ficam as duas regras que dão valor às rotas:
//  1. **a origem do membro é o ATOR AUTENTICADO**, nunca o corpo do pedido (*o cliente não escolhe quem declara*);
//  2. **sala inexistente é 404**, e **nunca** uma lista vazia — *«não há ninguém» ≠ «essa sala não existe»*.

func collabTestContext(t *testing.T, projectID, actor, organizationID string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	ctx, recorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/v1/collab/"+projectID, "{}", organizationID)
	ctx.Params = params
	if actor != "" {
		ctx.Set("agent.user", agent.User{ID: actor})
	}
	return ctx, recorder
}

func TestCollabJoinProjectUsesTheAuthenticatedActorAsOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, projectA, _ := newObjectScopeTestAPI(t)
	params := gin.Params{{Key: "project_id", Value: projectA.ID}}

	// (1) O corpo TENTA declarar a origem. **E é RECUSADO**: o `decodeJSON` do projeto usa
	// `DisallowUnknownFields`, então o pedido cai com 400 em vez de ser silenciosamente ignorado.
	// *Medido, não suposto: eu esperava «ignora em silêncio» e o servidor foi mais forte - recusou.*
	ctx, recorder := collabTestContext(t, projectA.ID, "ana", projectA.OrganizationID, params)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/collab/"+projectA.ID,
		strings.NewReader(`{"id":"bob","tipo":"agente","nome":"Pesquisador","origem":"falsificador"}`))
	api.collabJoinProject(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("origem no corpo TEM de ser recusada (400), veio %d: %s", recorder.Code, recorder.Body.String())
	}

	// (2) Corpo limpo: a origem é o ATOR AUTENTICADO, e o tipo exigido é respeitado.
	ctx, recorder = collabTestContext(t, projectA.ID, "ana", projectA.OrganizationID, params)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/collab/"+projectA.ID,
		strings.NewReader(`{"id":"bob","tipo":"agente","nome":"Pesquisador"}`))
	api.collabJoinProject(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("esperava 201, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	var membro agent.Participant
	if err := json.Unmarshal(recorder.Body.Bytes(), &membro); err != nil {
		t.Fatal(err)
	}
	if membro.Origin != "ana" {
		t.Fatalf("A ORIGEM TEM DE SER O ATOR: esperava %q, veio %q", "ana", membro.Origin)
	}
	if membro.Kind != agent.ParticipantAgent {
		t.Fatalf("esperava tipo agente, veio %q", membro.Kind)
	}
}

func TestCollabMembersListsHumansAndAgentsTogether(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, projectA, _ := newObjectScopeTestAPI(t)
	params := gin.Params{{Key: "project_id", Value: projectA.ID}}

	for _, entrada := range []string{
		`{"id":"ana","tipo":"humano","nome":"Ana"}`,
		`{"id":"agente-1","tipo":"agente","nome":"Pesquisador"}`,
	} {
		ctx, recorder := collabTestContext(t, projectA.ID, "ana", projectA.OrganizationID, params)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/collab/"+projectA.ID, strings.NewReader(entrada))
		api.collabJoinProject(ctx)
		if recorder.Code != http.StatusCreated {
			t.Fatalf("esperava 201 em %s, veio %d: %s", entrada, recorder.Code, recorder.Body.String())
		}
	}

	ctx, recorder := collabTestContext(t, projectA.ID, "ana", projectA.OrganizationID, params)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/collab/"+projectA.ID+"/members", nil)
	api.collabMembers(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d", recorder.Code)
	}
	var resposta struct {
		Membros []agent.Participant `json:"membros"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resposta); err != nil {
		t.Fatal(err)
	}
	if len(resposta.Membros) != 2 {
		t.Fatalf("esperava 2 membros (humano + agente), veio %d", len(resposta.Membros))
	}
	types := map[agent.ParticipantKind]bool{}
	for _, membro := range resposta.Membros {
		types[membro.Kind] = true
	}
	if !types[agent.ParticipantHuman] || !types[agent.ParticipantAgent] {
		t.Fatalf("a lista tem de trazer os dois tipos: %+v", resposta.Membros)
	}
}

func TestCollabRoomMembersMissingRoomIsNotFoundNotAnEmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, projectA, _ := newObjectScopeTestAPI(t)

	ctx, recorder := collabTestContext(t, projectA.ID, "ana", projectA.OrganizationID, gin.Params{
		{Key: "project_id", Value: projectA.ID},
		{Key: "room_id", Value: "sala-que-nao-existe"},
	})
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/collab/"+projectA.ID+"/rooms/inexistente/members", nil)
	api.collabRoomMembers(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("sala inexistente TEM de ser 404, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	// ***A DISTINÇÃO QUE IMPORTA:*** *404 sem lista - e não 200 com «membros: []», que faria a tela dizer
	// «ainda não há membros» sobre uma sala que não existe.*
	if strings.Contains(recorder.Body.String(), `"membros"`) {
		t.Fatalf("404 não pode trazer lista de membros: %s", recorder.Body.String())
	}
}

func TestCollabCreateRoomRequiresProjectMembership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, projectA, _ := newObjectScopeTestAPI(t)

	// `ninguem` não é membro do projeto: a regra vive no domínio e a rota NÃO a contorna.
	ctx, recorder := collabTestContext(t, projectA.ID, "ninguem",
		projectA.OrganizationID, gin.Params{{Key: "project_id", Value: projectA.ID}})
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/collab/"+projectA.ID+"/rooms",
		strings.NewReader(`{"id":"sala-1","nome":"Sala 1"}`))

	api.collabCreateRoom(ctx)

	if recorder.Code == http.StatusCreated {
		t.Fatal("quem NÃO é membro do projeto criou uma sala")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400, veio %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCollabCreateRoomAndJoinRoomRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, projectA, _ := newObjectScopeTestAPI(t)
	params := gin.Params{{Key: "project_id", Value: projectA.ID}}

	// 1. `ana` entra no projeto (é o que a torna elegível a criar a sala).
	ctx, recorder := collabTestContext(t, projectA.ID, "ana", projectA.OrganizationID, params)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/collab/"+projectA.ID,
		strings.NewReader(`{"id":"ana","tipo":"humano","nome":"Ana"}`))
	api.collabJoinProject(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("entrada no projeto falhou: %d %s", recorder.Code, recorder.Body.String())
	}

	// 2. Cria a sala.
	ctx, recorder = collabTestContext(t, projectA.ID, "ana", projectA.OrganizationID, params)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/collab/"+projectA.ID+"/rooms",
		strings.NewReader(`{"id":"sala-1","nome":"Sala 1"}`))
	api.collabCreateRoom(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("criação da sala falhou: %d %s", recorder.Code, recorder.Body.String())
	}

	// 3. A sala existe ⇒ 200 com a lista (o criador entra automaticamente: uma lista VAZIA aqui seria defeito).
	ctx, recorder = collabTestContext(t, projectA.ID, "ana", projectA.OrganizationID, gin.Params{
		{Key: "project_id", Value: projectA.ID},
		{Key: "room_id", Value: "sala-1"},
	})
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/agent/v1/collab/"+projectA.ID+"/rooms/sala-1/members", nil)
	api.collabRoomMembers(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("sala existente TEM de ser 200, veio %d: %s", recorder.Code, recorder.Body.String())
	}
	var resposta struct {
		Membros []agent.Participant `json:"membros"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resposta); err != nil {
		t.Fatal(err)
	}
	if len(resposta.Membros) == 0 {
		t.Fatal("a sala existe e o criador entra nela: lista vazia contradiz a regra do domínio")
	}
}
