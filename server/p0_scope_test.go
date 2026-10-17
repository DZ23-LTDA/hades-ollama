package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func orgContext(t *testing.T, method, path string, organizationID string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, nil)
	ctx.Set("agent.organization", agent.Organization{ID: organizationID})
	return ctx, recorder
}

func TestAgentP0StoresRejectCrossTenantReadsAndMutations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	traces, err := agent.NewTraceStore("")
	if err != nil {
		t.Fatal(err)
	}
	devices, err := agent.NewDeviceStore("")
	if err != nil {
		t.Fatal(err)
	}
	runner := func(context.Context, agent.AgentTask) (agent.AgentResult, error) {
		return agent.AgentResult{Output: "ok"}, nil
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: t.TempDir(), Traces: traces, Devices: devices, Planner: agent.RulePlanner{}, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	_ = runner
	api := &agentAPI{runtime: runtime}

	job, err := runtime.Orchestrator().PlanForOrganization("org-a", "tenant A", t.TempDir(), "", []agent.AgentRole{agent.RoleTesting}, agent.AgentBudget{})
	if err != nil {
		t.Fatal(err)
	}
	getContext, getRecorder := orgContext(t, http.MethodGet, "/orchestration/jobs/"+job.ID, "org-b")
	getContext.Params = gin.Params{{Key: "id", Value: job.ID}}
	api.getOrchestration(getContext)
	if getRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant orchestration read status=%d body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	cancelContext, cancelRecorder := orgContext(t, http.MethodPost, "/orchestration/jobs/"+job.ID+"/cancel", "org-b")
	cancelContext.Params = gin.Params{{Key: "id", Value: job.ID}}
	api.cancelOrchestration(cancelContext)
	if cancelRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant orchestration cancel status=%d body=%s", cancelRecorder.Code, cancelRecorder.Body.String())
	}
	stored, err := runtime.Orchestrator().GetForOrganization(job.ID, "org-a")
	if err != nil || stored.State != agent.OrchestrationPlanned {
		t.Fatalf("cross-tenant orchestration mutation changed job=%+v err=%v", stored, err)
	}

	span := traces.StartForOrganization("org-a", "tr_scope", "", "test", nil)
	span.End("ok", nil)
	traceContext, traceRecorder := orgContext(t, http.MethodGet, "/traces", "org-b")
	api.allTraces(traceContext)
	var tracePayload struct {
		Spans []agent.TraceSpan `json:"spans"`
	}
	if err := json.Unmarshal(traceRecorder.Body.Bytes(), &tracePayload); err != nil {
		t.Fatal(err)
	}
	if traceRecorder.Code != http.StatusOK || len(tracePayload.Spans) != 0 {
		t.Fatalf("cross-tenant traces code=%d spans=%+v", traceRecorder.Code, tracePayload.Spans)
	}

	code, _, err := devices.StartPairing("user-a", "org-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	device, token, err := devices.CompletePairing(code, "Org A desktop", "linux", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	deviceListContext, deviceListRecorder := orgContext(t, http.MethodGet, "/devices", "org-b")
	api.devices(deviceListContext)
	var devicePayload struct {
		Devices []agent.Device `json:"devices"`
	}
	if err := json.Unmarshal(deviceListRecorder.Body.Bytes(), &devicePayload); err != nil {
		t.Fatal(err)
	}
	if deviceListRecorder.Code != http.StatusOK || len(devicePayload.Devices) != 0 {
		t.Fatalf("cross-tenant devices code=%d devices=%+v", deviceListRecorder.Code, devicePayload.Devices)
	}
	revokeContext, revokeRecorder := orgContext(t, http.MethodPost, "/devices/"+device.ID+"/revoke", "org-b")
	revokeContext.Params = gin.Params{{Key: "id", Value: device.ID}}
	api.revokeDevice(revokeContext)
	if revokeRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant device revoke status=%d body=%s", revokeRecorder.Code, revokeRecorder.Body.String())
	}
	unchanged, err := devices.GetForOrganization(device.ID, "org-a")
	if err != nil || unchanged.Status == agent.DeviceRevoked {
		t.Fatalf("cross-tenant device mutation changed device=%+v err=%v", unchanged, err)
	}
	heartbeatContext, heartbeatRecorder := orgContext(t, http.MethodPost, "/devices/"+device.ID+"/heartbeat", "org-b")
	heartbeatContext.Params = gin.Params{{Key: "id", Value: device.ID}}
	heartbeatContext.Request = httptest.NewRequest(http.MethodPost, "/devices/"+device.ID+"/heartbeat", bytes.NewBufferString(`{}`))
	heartbeatContext.Request.Header.Set("Authorization", "Bearer "+token)
	api.deviceHeartbeat(heartbeatContext)
	if heartbeatRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant heartbeat status=%d body=%s", heartbeatRecorder.Code, heartbeatRecorder.Body.String())
	}

	mission, err := runtime.CreateMission(context.Background(), agent.CreateMissionRequest{Objective: "queued tenant", OrganizationID: "org-a"})
	if err != nil {
		t.Fatal(err)
	}
	queued, err := runtime.EnqueueMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	jobsContext, jobsRecorder := orgContext(t, http.MethodGet, "/jobs", "org-b")
	api.jobs(jobsContext)
	var jobsPayload struct {
		Jobs []agent.QueueJob `json:"jobs"`
	}
	if err := json.Unmarshal(jobsRecorder.Body.Bytes(), &jobsPayload); err != nil {
		t.Fatal(err)
	}
	if jobsRecorder.Code != http.StatusOK || len(jobsPayload.Jobs) != 0 {
		t.Fatalf("cross-tenant jobs code=%d jobs=%+v", jobsRecorder.Code, jobsPayload.Jobs)
	}
	replayContext, replayRecorder := orgContext(t, http.MethodPost, "/jobs/"+queued.ID+"/replay", "org-b")
	replayContext.Params = gin.Params{{Key: "id", Value: queued.ID}}
	api.replayJob(replayContext)
	if replayRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant job replay status=%d body=%s", replayRecorder.Code, replayRecorder.Body.String())
	}
}
