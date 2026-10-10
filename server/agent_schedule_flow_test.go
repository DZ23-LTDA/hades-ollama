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

const scheduleFlowBody = `{"objective":"Fluxo diário","interval_seconds":3600,"steps":[` +
	`{"id":"gatilho","kind":"trigger.interval"},` +
	`{"id":"coletar","kind":"action.mission","depends_on":["gatilho"],"objective":"Coletar métricas"},` +
	`{"id":"decidir","kind":"condition.if","depends_on":["coletar"],"expect":"succeeded"}]}`

func decodeSchedules(t *testing.T, recorder *httptest.ResponseRecorder) []agent.Schedule {
	t.Helper()
	var payload struct {
		Schedules []agent.Schedule `json:"schedules"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decodificar listagem de agendamentos: %v (corpo %s)", err, recorder.Body.String())
	}
	return payload.Schedules
}

func TestScheduleRoutePersistsFlowGraph(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, _, _ := newObjectScopeTestAPI(t)

	ctx, recorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/v1/schedules", scheduleFlowBody, agent.LocalOrganizationID)
	api.createSchedule(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("esperava 201, obteve %d: %s", recorder.Code, recorder.Body.String())
	}
	var created agent.Schedule
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decodificar agendamento criado: %v", err)
	}
	if len(created.Steps) != 3 {
		t.Fatalf("esperava 3 passos persistidos, obteve %d: %s", len(created.Steps), recorder.Body.String())
	}
	if created.Steps[2].Kind != agent.ScheduleStepConditionIf || created.Steps[2].Expect != agent.ScheduleExpectSucceeded {
		t.Fatalf("passo condicional perdido no round-trip: %+v", created.Steps[2])
	}

	listCtx, listRecorder := scopedObjectRequest(t, http.MethodGet, "/api/agent/v1/schedules", "", agent.LocalOrganizationID)
	api.schedules(listCtx)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("esperava 200 na listagem, obteve %d", listRecorder.Code)
	}
	schedules := decodeSchedules(t, listRecorder)
	if len(schedules) != 1 || len(schedules[0].Steps) != 3 {
		t.Fatalf("grafo não sobreviveu à listagem: %+v", schedules)
	}
	if schedules[0].Steps[1].Objective != "Coletar métricas" {
		t.Fatalf("objetivo do passo não persistiu: %+v", schedules[0].Steps[1])
	}
}

func TestScheduleRouteRejectsUnknownStepKind(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, _, _ := newObjectScopeTestAPI(t)

	body := `{"objective":"Fluxo inválido","interval_seconds":3600,"steps":[` +
		`{"id":"gatilho","kind":"trigger.interval"},` +
		`{"id":"shell","kind":"action.shell","depends_on":["gatilho"],"objective":"rodar"}]}`
	ctx, recorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/v1/schedules", body, agent.LocalOrganizationID)
	api.createSchedule(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400, obteve %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), agent.ErrUnknownScheduleStep.Error()) {
		t.Fatalf("erro deveria citar o vocabulário fechado, obteve: %s", recorder.Body.String())
	}

	listCtx, listRecorder := scopedObjectRequest(t, http.MethodGet, "/api/agent/v1/schedules", "", agent.LocalOrganizationID)
	api.schedules(listCtx)
	if schedules := decodeSchedules(t, listRecorder); len(schedules) != 0 {
		t.Fatalf("fluxo inválido não deveria persistir, obteve %d agendamento(s)", len(schedules))
	}
}

func TestScheduleRouteUpdatePersistsAndValidatesFlowGraph(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api, _, _ := newObjectScopeTestAPI(t)

	createCtx, createRecorder := scopedObjectRequest(t, http.MethodPost, "/api/agent/v1/schedules", scheduleFlowBody, agent.LocalOrganizationID)
	api.createSchedule(createCtx)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("esperava 201 na criação, obteve %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created agent.Schedule
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decodificar agendamento criado: %v", err)
	}

	widened := `{"objective":"Fluxo diário ampliado","interval_seconds":3600,"steps":[` +
		`{"id":"gatilho","kind":"trigger.interval"},` +
		`{"id":"coletar","kind":"action.mission","depends_on":["gatilho"],"objective":"Coletar métricas"},` +
		`{"id":"decidir","kind":"condition.if","depends_on":["coletar"],"expect":"succeeded"},` +
		`{"id":"publicar","kind":"action.mission","depends_on":["decidir"],"objective":"Publicar relatório"}]}`

	updateCtx, updateRecorder := scopedObjectRequest(t, http.MethodPut, "/api/agent/v1/schedules/"+created.ID, widened, agent.LocalOrganizationID)
	updateCtx.Params = gin.Params{gin.Param{Key: "id", Value: created.ID}}
	api.updateSchedule(updateCtx)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("esperava 200 no update, obteve %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated agent.Schedule
	if err := json.Unmarshal(updateRecorder.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decodificar agendamento atualizado: %v", err)
	}
	if len(updated.Steps) != 4 {
		t.Fatalf("esperava 4 passos após update, obteve %d", len(updated.Steps))
	}

	broken := `{"objective":"Fluxo diário ampliado","interval_seconds":3600,"steps":[` +
		`{"id":"gatilho","kind":"trigger.interval"},` +
		`{"id":"a","kind":"action.mission","depends_on":["b"],"objective":"a"},` +
		`{"id":"b","kind":"action.mission","depends_on":["a"],"objective":"b"}]}`
	brokenCtx, brokenRecorder := scopedObjectRequest(t, http.MethodPut, "/api/agent/v1/schedules/"+created.ID, broken, agent.LocalOrganizationID)
	brokenCtx.Params = gin.Params{gin.Param{Key: "id", Value: created.ID}}
	api.updateSchedule(brokenCtx)
	if brokenRecorder.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 para grafo cíclico, obteve %d: %s", brokenRecorder.Code, brokenRecorder.Body.String())
	}

	listCtx, listRecorder := scopedObjectRequest(t, http.MethodGet, "/api/agent/v1/schedules", "", agent.LocalOrganizationID)
	api.schedules(listCtx)
	schedules := decodeSchedules(t, listRecorder)
	if len(schedules) != 1 || len(schedules[0].Steps) != 4 {
		t.Fatalf("grafo válido não deveria ser sobrescrito por um inválido: %+v", schedules)
	}
}
