package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func webhookTestContext(t *testing.T, organizationID, scheduleID, secret, idempotencyKey string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/webhooks/"+scheduleID, bytes.NewBufferString(`{"event":"created"}`))
	ctx.Request.Header.Set("X-Ollama-Agent-Secret", secret)
	if idempotencyKey != "" {
		ctx.Request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	ctx.Params = gin.Params{{Key: "schedule_id", Value: scheduleID}}
	ctx.Set("agent.organization", agent.Organization{ID: organizationID})
	return ctx, recorder
}

func TestWebhookBindsScheduleOrganizationAndDeduplicates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("WEBHOOK_TEST_SECRET", "webhook-secret")
	contextStore, err := agent.NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	if err := contextStore.SetWorkspaceRoot(workspace); err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("webhook", workspace, "org_a")
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := contextStore.CreateSchedule(agent.Schedule{Objective: "process webhook", ProjectID: project.ID, OrganizationID: "org_a", IntervalSeconds: 60, Enabled: true, WebhookSecretEnv: "WEBHOOK_TEST_SECRET"})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Context: contextStore, Planner: agent.RulePlanner{}, WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, context: contextStore, authRequired: true}

	crossTenant, crossTenantRecorder := webhookTestContext(t, "org_b", schedule.ID, "webhook-secret", "evt-1")
	api.webhook(crossTenant)
	if crossTenantRecorder.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant webhook status=%d body=%s", crossTenantRecorder.Code, crossTenantRecorder.Body.String())
	}

	missingKey, missingKeyRecorder := webhookTestContext(t, "org_a", schedule.ID, "webhook-secret", "")
	api.webhook(missingKey)
	if missingKeyRecorder.Code != http.StatusBadRequest {
		t.Fatalf("missing key status=%d body=%s", missingKeyRecorder.Code, missingKeyRecorder.Body.String())
	}

	first, firstRecorder := webhookTestContext(t, "org_a", schedule.ID, "webhook-secret", "evt-1")
	api.webhook(first)
	if firstRecorder.Code != http.StatusAccepted {
		t.Fatalf("first webhook status=%d body=%s", firstRecorder.Code, firstRecorder.Body.String())
	}
	var firstMission agent.Mission
	if err := json.Unmarshal(firstRecorder.Body.Bytes(), &firstMission); err != nil {
		t.Fatal(err)
	}

	replay, replayRecorder := webhookTestContext(t, "org_a", schedule.ID, "webhook-secret", "evt-1")
	api.webhook(replay)
	if replayRecorder.Code != http.StatusAccepted {
		t.Fatalf("replay webhook status=%d body=%s", replayRecorder.Code, replayRecorder.Body.String())
	}
	var replayMission agent.Mission
	if err := json.Unmarshal(replayRecorder.Body.Bytes(), &replayMission); err != nil {
		t.Fatal(err)
	}
	if replayMission.ID != firstMission.ID {
		t.Fatalf("replay mission id=%q, want original %q", replayMission.ID, firstMission.ID)
	}
	brokenSchedule := schedule
	brokenSchedule.ProjectID = "proj_missing_for_retry_test"
	if _, err := contextStore.UpdateSchedule(schedule.ID, brokenSchedule); err != nil {
		t.Fatal(err)
	}
	failed, failedRecorder := webhookTestContext(t, "org_a", schedule.ID, "webhook-secret", "evt-retry-after-failure")
	api.webhook(failed)
	if failedRecorder.Code != http.StatusBadRequest {
		t.Fatalf("failed mission creation status=%d body=%s", failedRecorder.Code, failedRecorder.Body.String())
	}
	if _, err := contextStore.UpdateSchedule(schedule.ID, schedule); err != nil {
		t.Fatal(err)
	}
	retryAfterFailure, retryAfterFailureRecorder := webhookTestContext(t, "org_a", schedule.ID, "webhook-secret", "evt-retry-after-failure")
	api.webhook(retryAfterFailure)
	if retryAfterFailureRecorder.Code != http.StatusAccepted {
		t.Fatalf("retry after pre-persistence failure status=%d body=%s", retryAfterFailureRecorder.Code, retryAfterFailureRecorder.Body.String())
	}
}

func TestWebhookRejectsSensitivePayloadBeforeReplayClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("WEBHOOK_TEST_SECRET", "webhook-secret")
	contextStore, err := agent.NewContextStore("")
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	if err := contextStore.SetWorkspaceRoot(workspace); err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("webhook", workspace, "org_a")
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := contextStore.CreateSchedule(agent.Schedule{Objective: "process webhook", ProjectID: project.ID, OrganizationID: "org_a", IntervalSeconds: 60, Enabled: true, WebhookSecretEnv: "WEBHOOK_TEST_SECRET"})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := agent.NewRuntime(agent.RuntimeConfig{Context: contextStore, Planner: agent.RulePlanner{}, WorkspaceRoot: workspace})
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{runtime: runtime, context: contextStore, authRequired: true}

	sensitive, sensitiveRecorder := webhookTestContext(t, "org_a", schedule.ID, "webhook-secret", "evt-sensitive")
	sensitive.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/webhooks/"+schedule.ID, bytes.NewBufferString(`{"access_token":"ghp_abcdefghijklmnopqrstuvwxyz123456"}`))
	sensitive.Request.Header.Set("X-Ollama-Agent-Secret", "webhook-secret")
	sensitive.Request.Header.Set("Idempotency-Key", "evt-sensitive")
	api.webhook(sensitive)
	if sensitiveRecorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("sensitive webhook status=%d body=%s", sensitiveRecorder.Code, sensitiveRecorder.Body.String())
	}
	quotedKey, quotedKeyRecorder := webhookTestContext(t, "org_a", schedule.ID, "webhook-secret", "evt-quoted-key")
	quotedKey.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/webhooks/"+schedule.ID, bytes.NewBufferString(`{"payload":{"refresh_token":"ordinary-secret-value"}}`))
	quotedKey.Request.Header.Set("X-Ollama-Agent-Secret", "webhook-secret")
	quotedKey.Request.Header.Set("Idempotency-Key", "evt-quoted-key")
	api.webhook(quotedKey)
	if quotedKeyRecorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("quoted sensitive key webhook status=%d body=%s", quotedKeyRecorder.Code, quotedKeyRecorder.Body.String())
	}

	// A safe retry with the same provider event ID succeeds because the DLP
	// rejection happened before the durable replay claim.
	retry, retryRecorder := webhookTestContext(t, "org_a", schedule.ID, "webhook-secret", "evt-sensitive")
	api.webhook(retry)
	if retryRecorder.Code != http.StatusAccepted {
		t.Fatalf("safe retry status=%d body=%s", retryRecorder.Code, retryRecorder.Body.String())
	}
}
