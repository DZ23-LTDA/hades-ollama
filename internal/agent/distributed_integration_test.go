//go:build integration

package agent

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func expireRedisLeaseForTest(t *testing.T, ctx context.Context, queue *RedisQueue, jobID string) {
	t.Helper()
	job, err := queue.get(jobID)
	if err != nil {
		t.Fatal(err)
	}
	expired := time.Now().UTC().Add(-time.Second)
	job.LeaseUntil = expired
	job.LeaseUntilUnixMilli = expired.UnixMilli()
	if err := queue.put(job); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "ZADD", queue.leaseKey(), strconv.FormatInt(job.LeaseUntilUnixMilli, 10), jobID); err != nil {
		t.Fatal(err)
	}
}

func captureRedisEnqueueState(t *testing.T, ctx context.Context, queue *RedisQueue, missionID, jobID string) map[string]any {
	t.Helper()
	commands := map[string][]string{
		"mission":       {"GET", queue.missionKey(missionID)},
		"job":           {"GET", queue.jobKey(jobID)},
		"pending":       {"LRANGE", queue.pendingKey(), "0", "-1"},
		"pending_count": {"LLEN", queue.pendingKey()},
		"delayed":       {"ZRANGE", queue.delayedKey(), "0", "-1", "WITHSCORES"},
		"delayed_score": {"ZSCORE", queue.delayedKey(), jobID},
		"delayed_count": {"ZCARD", queue.delayedKey()},
		"leases":        {"ZRANGE", queue.leaseKey(), "0", "-1", "WITHSCORES"},
		"lease_score":   {"ZSCORE", queue.leaseKey(), jobID},
		"lease_count":   {"ZCARD", queue.leaseKey()},
		"dead":          {"LRANGE", queue.deadKey(), "0", "-1"},
		"sequence":      {"GET", queue.key("sequence")},
	}
	state := make(map[string]any, len(commands))
	for name, command := range commands {
		args := append([]string{command[0]}, command[1:]...)
		value, err := queue.do(ctx, args...)
		if err != nil {
			t.Fatalf("capture Redis state %s: %v", name, err)
		}
		state[name] = value
	}
	return state
}

func openRedisTestQueue(t *testing.T, ctx context.Context, redisURL, prefix string) (*RedisQueue, error) {
	t.Helper()
	if !strings.HasPrefix(prefix, "ollama:") || strings.ContainsAny(prefix, "*?[]\\") {
		return nil, errors.New("Redis integration-test prefix is outside the safe namespace")
	}
	prefix += ":" + uuid.NewString()
	queue, err := OpenRedisQueue(ctx, redisURL, prefix)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { cleanupRedisTestQueue(t, queue) })
	return queue, nil
}

func cleanupRedisTestQueue(t *testing.T, queue *RedisQueue) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor := "0"
	keys := make([]string, 0)
	for iterations := 0; iterations < 10000; iterations++ {
		value, err := queue.do(ctx, "SCAN", cursor, "MATCH", queue.prefix+":*", "COUNT", "100")
		if err != nil {
			t.Errorf("scan Redis test namespace: %v", err)
			return
		}
		response, ok := value.([]any)
		if !ok || len(response) != 2 {
			t.Errorf("invalid Redis SCAN response: %#v", value)
			return
		}
		cursor, ok = response[0].(string)
		if !ok {
			t.Errorf("invalid Redis SCAN cursor: %#v", response[0])
			return
		}
		batch, ok := response[1].([]any)
		if !ok {
			t.Errorf("invalid Redis SCAN keys: %#v", response[1])
			return
		}
		for _, item := range batch {
			key, ok := item.(string)
			if !ok || !strings.HasPrefix(key, queue.prefix+":") {
				t.Errorf("Redis SCAN escaped test namespace: %#v", item)
				return
			}
			keys = append(keys, key)
		}
		if cursor == "0" {
			break
		}
		if iterations == 9999 {
			t.Errorf("Redis SCAN exceeded iteration limit for test namespace")
			return
		}
	}
	for _, key := range keys {
		if _, err := queue.do(ctx, "DEL", key); err != nil {
			t.Errorf("delete Redis test key %q: %v", key, err)
		}
	}
}

func assertRedisEnqueueStateUnchanged(t *testing.T, before, after map[string]any) {
	t.Helper()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("Enqueue mutated Redis state:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestDistributedPostgresRuntimeReadinessForLegacyRole(t *testing.T) {
	runtimeDSN := os.Getenv("OLLAMA_AGENT_TEST_POSTGRES_RUNTIME_URL")
	key, keyErr := hex.DecodeString(os.Getenv("OLLAMA_AGENT_TEST_POSTGRES_TENANT_KEY"))
	if runtimeDSN == "" || keyErr != nil || len(key) < 32 {
		t.Skip("runtime test DSN and a 64+ byte hex tenant key are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	store, err := OpenPostgresRuntimeStore(ctx, runtimeDSN, key)
	wantLegacyActive := strings.EqualFold(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_TEST_EXPECT_LEGACY_ROLE_ACTIVE")), "true")
	if wantLegacyActive {
		if err == nil {
			_ = store.Close()
			t.Fatal("PostgreSQL runtime opened while the legacy ollama_agent privileged login remains active")
		}
		if !strings.Contains(err.Error(), "active legacy ollama_agent login") {
			t.Fatalf("runtime readiness error=%v, want active legacy role refusal", err)
		}
		return
	}
	if err != nil {
		t.Fatalf("PostgreSQL runtime did not become ready after legacy-role retirement: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDistributedPostgresRuntimeReadinessForStaleLegacySession(t *testing.T) {
	runtimeDSN := os.Getenv("OLLAMA_AGENT_TEST_POSTGRES_RUNTIME_URL")
	adminDSN := os.Getenv("OLLAMA_AGENT_TEST_POSTGRES_ADMIN_URL")
	legacyDSN := os.Getenv("OLLAMA_AGENT_TEST_POSTGRES_LEGACY_URL")
	key, keyErr := hex.DecodeString(os.Getenv("OLLAMA_AGENT_TEST_POSTGRES_TENANT_KEY"))
	if runtimeDSN == "" || adminDSN == "" || legacyDSN == "" || keyErr != nil || len(key) < 32 {
		t.Skip("runtime/admin/legacy PostgreSQL test DSNs and a 64+ byte hex tenant key are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	legacyDB, err := sql.Open("pgx", legacyDSN)
	if err != nil {
		t.Fatal(err)
	}
	legacyDB.SetMaxIdleConns(0)
	defer legacyDB.Close()
	legacyConn, err := legacyDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer legacyConn.Close()
	if _, err := legacyConn.ExecContext(ctx, `SELECT 1`); err != nil {
		t.Fatal(err)
	}
	adminDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer adminDB.Close()
	if _, err := adminDB.ExecContext(ctx, `ALTER ROLE ollama_agent NOLOGIN NOSUPERUSER NOBYPASSRLS`); err != nil {
		t.Fatal(err)
	}
	store, err := OpenPostgresRuntimeStore(ctx, runtimeDSN, key)
	if err == nil {
		_ = store.Close()
		t.Fatal("PostgreSQL runtime opened while a privileged legacy session remained connected")
	}
	if !strings.Contains(err.Error(), "active legacy ollama_agent login") {
		t.Fatalf("runtime readiness error=%v, want stale legacy session refusal", err)
	}
	if err := legacyConn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := adminDB.ExecContext(ctx, `ALTER ROLE ollama_agent LOGIN SUPERUSER`); err != nil {
		t.Fatalf("restore legacy fixture role for retirement gate: %v", err)
	}
}

func TestDistributedPostgresRLSAndEvents(t *testing.T) {
	runtimeDSN := os.Getenv("OLLAMA_AGENT_TEST_POSTGRES_RUNTIME_URL")
	migratorDSN := os.Getenv("OLLAMA_AGENT_TEST_POSTGRES_MIGRATOR_URL")
	key, keyErr := hex.DecodeString(os.Getenv("OLLAMA_AGENT_TEST_POSTGRES_TENANT_KEY"))
	if runtimeDSN == "" || migratorDSN == "" || keyErr != nil || len(key) < 32 {
		t.Skip("OLLAMA_AGENT_TEST_POSTGRES_RUNTIME_URL, OLLAMA_AGENT_TEST_POSTGRES_MIGRATOR_URL, and a 64+ byte hex OLLAMA_AGENT_TEST_POSTGRES_TENANT_KEY are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := MigratePostgresAgentSchema(ctx, migratorDSN, key); err != nil {
		t.Fatal(err)
	}
	store, err := OpenPostgresRuntimeStore(ctx, runtimeDSN, key)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	wrongKey := append([]byte(nil), key...)
	wrongKey[0] ^= 0xff
	if wrongKeyStore, err := OpenPostgresRuntimeStore(ctx, runtimeDSN, wrongKey); err == nil {
		_ = wrongKeyStore.Close()
		t.Fatal("PostgreSQL runtime opened with a signing key different from the migrated database")
	}
	if err := MigratePostgresAgentSchema(ctx, migratorDSN, wrongKey); err == nil || !strings.Contains(err.Error(), "ordinary migrations do not rotate keys") {
		t.Fatalf("migration with a different key error=%v, want explicit rotation refusal", err)
	}

	orgA := "org_test_a_" + uuid.NewString()
	orgB := "org_test_b_" + uuid.NewString()
	missionWorkspace := t.TempDir()
	missionWorkspaceIdentity, err := workspaceDirectoryIdentity(missionWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	mission := Mission{ID: "mis_" + uuid.NewString(), Version: 1, Objective: "tenant RLS", Provider: "ollama-local", Model: "qwen3-coder", Capabilities: []string{"workspace:read", "workspace:write"}, OrganizationID: orgA, ProjectID: "proj_test", Workspace: missionWorkspace, WorkspaceIdentity: missionWorkspaceIdentity, WorkspaceIsolated: true, WorkspaceSnapshotID: "snp_" + uuid.NewString(), WorkspaceSnapshotSHA256: strings.Repeat("a", 64), State: MissionReady, Plan: []Step{}, Approvals: []Approval{}, Artifacts: []ArtifactManifest{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.WithOrganization(orgA).PutMission(mission); err != nil {
		t.Fatal(err)
	}
	duplicateCreate := mission
	duplicateCreate.Objective = "create-only must not overwrite"
	if err := store.WithOrganization(orgA).CreateMission(duplicateCreate); !errors.Is(err, ErrMissionAlreadyExists) {
		t.Fatalf("duplicate Postgres create error=%v, want ErrMissionAlreadyExists", err)
	}
	if got, err := store.WithOrganization(orgA).GetMission(mission.ID); err != nil || got.OrganizationID != orgA || got.Provider != mission.Provider || got.Model != mission.Model || strings.Join(got.Capabilities, ",") != strings.Join(mission.Capabilities, ",") || !got.WorkspaceIsolated || got.WorkspaceSnapshotID != mission.WorkspaceSnapshotID || got.WorkspaceSnapshotSHA256 != mission.WorkspaceSnapshotSHA256 {
		t.Fatalf("same-tenant read failed: got=%+v err=%v", got, err)
	}
	blocker, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	blockerContext, err := store.signedTenantContext(orgA, time.Now())
	if err != nil {
		_ = blocker.Rollback()
		t.Fatal(err)
	}
	if _, err := blocker.ExecContext(ctx, `SELECT set_config('app.tenant_context',$1,true)`, blockerContext); err != nil {
		_ = blocker.Rollback()
		t.Fatal(err)
	}
	var lockedMissionID string
	if err := blocker.QueryRowContext(ctx, `SELECT id FROM agent_missions WHERE id=$1 FOR UPDATE`, mission.ID).Scan(&lockedMissionID); err != nil {
		_ = blocker.Rollback()
		t.Fatal(err)
	}
	requestCtx, cancelRequest := context.WithTimeout(ctx, 150*time.Millisecond)
	blockedUpdate := mission
	blockedUpdate.Version++
	blockedUpdate.UpdatedAt = time.Now().UTC()
	blockedErr := store.WithOrganizationContext(requestCtx, orgA).PutMission(blockedUpdate)
	cancelRequest()
	_ = blocker.Rollback()
	if !errors.Is(blockedErr, context.DeadlineExceeded) {
		t.Fatalf("canceled tenant store operation error=%v, want context deadline exceeded", blockedErr)
	}
	foreignMission := Mission{ID: "mis_" + uuid.NewString(), Version: 1, Objective: "raw RLS foreign row", OrganizationID: orgB, State: MissionCompleted, Plan: []Step{}, Approvals: []Approval{}, Artifacts: []ArtifactManifest{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.WithOrganization(orgB).PutMission(foreignMission); err != nil {
		t.Fatal(err)
	}
	var superuser, bypassRLS bool
	if err := store.db.QueryRowContext(ctx, `SELECT rolsuper,rolbypassrls FROM pg_roles WHERE rolname=current_user`).Scan(&superuser, &bypassRLS); err != nil {
		t.Fatal(err)
	}
	if superuser || bypassRLS {
		t.Fatalf("integration role is privileged: superuser=%v bypassRLS=%v", superuser, bypassRLS)
	}
	rawTx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	validContextA, err := store.signedTenantContext(orgA, time.Now())
	if err != nil {
		_ = rawTx.Rollback()
		t.Fatal(err)
	}
	if _, err := rawTx.ExecContext(ctx, `SELECT set_config('app.tenant_context',$1,true),set_config('app.current_organization_id',$2,true),set_config('app.system_access','1',true)`, validContextA, orgB); err != nil {
		_ = rawTx.Rollback()
		t.Fatal(err)
	}
	var visibleA, visibleB int
	if err := rawTx.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_missions WHERE id=$1`, mission.ID).Scan(&visibleA); err != nil {
		_ = rawTx.Rollback()
		t.Fatal(err)
	}
	if err := rawTx.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_missions WHERE id=$1`, foreignMission.ID).Scan(&visibleB); err != nil {
		_ = rawTx.Rollback()
		t.Fatal(err)
	}
	if visibleA != 1 || visibleB != 0 {
		_ = rawTx.Rollback()
		t.Fatalf("signed-context RLS visibility mismatch: orgA=%d orgB=%d", visibleA, visibleB)
	}
	if _, err := rawTx.ExecContext(ctx, `SAVEPOINT cross_tenant_event_attempt`); err != nil {
		_ = rawTx.Rollback()
		t.Fatal(err)
	}
	_, crossTenantEventErr := rawTx.ExecContext(ctx, `INSERT INTO agent_events (id,mission_id,organization_id,type,payload,created_at) VALUES ($1,$2,$3,'raw.cross_tenant',NULL,NOW())`, "evt_"+uuid.NewString(), foreignMission.ID, orgA)
	if crossTenantEventErr == nil {
		_ = rawTx.Rollback()
		t.Fatal("runtime SQL attached an orgA event to an orgB mission")
	}
	if _, err := rawTx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT cross_tenant_event_attempt`); err != nil {
		_ = rawTx.Rollback()
		t.Fatalf("restore transaction after expected cross-tenant event rejection: %v", err)
	}
	fakeContextB := orgB + "|" + strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10) + "|" + strings.Repeat("0", 64)
	if _, err := rawTx.ExecContext(ctx, `SELECT set_config('app.tenant_context',$1,true)`, fakeContextB); err != nil {
		_ = rawTx.Rollback()
		t.Fatal(err)
	}
	if err := rawTx.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_missions WHERE id=$1`, foreignMission.ID).Scan(&visibleB); err != nil {
		_ = rawTx.Rollback()
		t.Fatal(err)
	}
	if visibleB != 0 {
		_ = rawTx.Rollback()
		t.Fatalf("forged tenant HMAC exposed foreign mission: count=%d", visibleB)
	}
	_, insertErr := rawTx.ExecContext(ctx, `INSERT INTO agent_missions (id,version,objective,organization_id,state,plan,approvals,artifacts,created_at,updated_at) VALUES ($1,1,'raw cross-tenant insert',$2,'READY','[]'::jsonb,'[]'::jsonb,'[]'::jsonb,NOW(),NOW())`, "mis_"+uuid.NewString(), orgB)
	if insertErr == nil {
		_ = rawTx.Rollback()
		t.Fatal("raw PostgreSQL insert crossed tenant RLS policy")
	}
	if err := rawTx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		t.Fatal(err)
	}
	for _, attempt := range []struct {
		name  string
		query string
	}{
		{name: "read signing key", query: `SELECT secret FROM agent_tenant_context_key`},
		{name: "assume migrator role", query: `SET ROLE ollama_agent_migrator`},
		{name: "create schema object", query: `CREATE TABLE agent_runtime_forbidden_ddl(id INT)`},
	} {
		t.Run(attempt.name, func(t *testing.T) {
			tx, err := store.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tx.ExecContext(ctx, attempt.query); err == nil {
				_ = tx.Rollback()
				t.Fatalf("runtime unexpectedly succeeded at: %s", attempt.name)
			}
			_ = tx.Rollback()
		})
	}
	t.Run("row security off does not bypass policies", func(t *testing.T) {
		tx, err := store.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, `SET row_security = off`); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_missions`).Scan(&count); err == nil {
			t.Fatal("runtime bypassed RLS with row_security=off")
		}
	})
	if _, err := store.WithOrganization(orgB).GetMission(mission.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cross-tenant read should be hidden, got err=%v", err)
	}
	foreignCollision := mission
	foreignCollision.OrganizationID = orgB
	foreignCollision.Objective = "must not overwrite another tenant"
	if err := store.WithOrganization(orgB).PutMission(foreignCollision); err == nil {
		t.Fatal("cross-tenant PutMission overwrote an existing mission ID")
	}
	if got, err := store.WithOrganization(orgA).GetMission(mission.ID); err != nil || got.Objective != mission.Objective {
		t.Fatalf("cross-tenant upsert changed the owner mission: got=%+v err=%v", got, err)
	}
	updatedMission := mission
	updatedMission.Version++
	updatedMission.Objective = "tenant RLS CAS update"
	updatedMission.Capabilities = []string{"workspace:read"}
	if err := store.WithOrganization(orgA).PutMissionIfVersion(updatedMission, mission.Version); err != nil {
		t.Fatalf("same-tenant snapshot metadata CAS update failed: %v", err)
	}
	invalidAdvance := updatedMission
	invalidAdvance.Version += 2
	if err := store.WithOrganization(orgA).PutMissionIfVersion(invalidAdvance, updatedMission.Version); err == nil {
		t.Fatal("Postgres CAS accepted a version jump larger than one")
	}
	if got, err := store.WithOrganization(orgA).GetMission(mission.ID); err != nil || got.Version != updatedMission.Version || got.Provider != updatedMission.Provider || strings.Join(got.Capabilities, ",") != strings.Join(updatedMission.Capabilities, ",") || got.WorkspaceSnapshotSHA256 != mission.WorkspaceSnapshotSHA256 {
		t.Fatalf("same-tenant CAS read failed: got=%+v err=%v", got, err)
	}
	staleMission := mission
	staleMission.Objective = "stale unconditional write"
	if err := store.WithOrganization(orgA).PutMission(staleMission); !errors.Is(err, ErrMissionVersionConflict) {
		t.Fatalf("stale PostgreSQL PutMission error=%v, want version conflict", err)
	}
	if got, err := store.WithOrganization(orgA).GetMission(mission.ID); err != nil || got.Version != updatedMission.Version || got.Objective != updatedMission.Objective {
		t.Fatalf("stale PutMission changed newer PostgreSQL row: got=%+v err=%v", got, err)
	}
	event := Event{ID: "evt_" + uuid.NewString(), MissionID: mission.ID, OrganizationID: orgA, Type: "integration.test", CreatedAt: time.Now().UTC()}
	if err := store.WithOrganization(orgA).AppendEvent(event); err != nil {
		t.Fatal(err)
	}
	if err := store.WithOrganization(orgA).AppendEvent(event); err != nil {
		t.Fatalf("identical event replay should be idempotent: %v", err)
	}
	conflictingEvent := event
	conflictingEvent.Type = "integration.conflicting-replay"
	if err := store.WithOrganization(orgA).AppendEvent(conflictingEvent); err == nil || !strings.Contains(err.Error(), "different content") {
		t.Fatalf("conflicting event replay error = %v, want conflict", err)
	}
	wrongTenantEvent := Event{ID: "evt_" + uuid.NewString(), MissionID: mission.ID, OrganizationID: orgB, Type: "integration.wrong-tenant", CreatedAt: time.Now().UTC()}
	if err := store.WithOrganization(orgB).AppendEvent(wrongTenantEvent); err == nil {
		t.Fatal("Postgres accepted an event for a mission owned by another tenant")
	}
	invalidPayloadEvent := event
	invalidPayloadEvent.ID = "evt_" + uuid.NewString()
	invalidPayloadEvent.Payload = map[string]any{"unsupported": func() {}}
	if err := store.WithOrganization(orgA).AppendEvent(invalidPayloadEvent); err == nil {
		t.Fatal("Postgres accepted a non-JSON event payload")
	}
	if events, err := store.WithOrganization(orgA).ListEvents(mission.ID); err != nil || len(events) != 1 || events[0].OrganizationID != orgA {
		t.Fatalf("same-tenant events failed: events=%+v err=%v", events, err)
	}
	if events, err := store.WithOrganization(orgB).ListEvents(mission.ID); err != nil || len(events) != 0 {
		t.Fatalf("cross-tenant events should be empty: events=%+v err=%v", events, err)
	}
	const legacySecret = "legacy-db-credential-not-for-response"
	seedLegacyRows := func() {
		t.Helper()
		tx, err := store.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		tenantContext, contextErr := store.signedTenantContext(orgA, time.Now())
		if contextErr != nil {
			_ = tx.Rollback()
			t.Fatal(contextErr)
		}
		if _, err := tx.ExecContext(ctx, `SELECT set_config('app.tenant_context',$1,true)`, tenantContext); err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE agent_missions SET objective=$1 WHERE id=$2`, `{"api_key":"`+legacySecret+`"}`, mission.ID); err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE agent_events SET payload=$1::jsonb WHERE id=$2 AND mission_id=$3`, `{"nested":{"api_key":"`+legacySecret+`"}}`, event.ID, mission.ID); err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	seedLegacyRows()
	legacyMission, err := store.WithOrganization(orgA).GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	legacyMissionJSON, err := json.Marshal(legacyMission)
	if err != nil || strings.Contains(string(legacyMissionJSON), legacySecret) {
		t.Fatalf("GetMission exposed legacy credential: err=%v", err)
	}
	seedLegacyRows()
	listedMissions, err := store.WithOrganization(orgA).ListMissions()
	if err != nil || len(listedMissions) == 0 {
		t.Fatalf("ListMissions failed after legacy row seed: count=%d err=%v", len(listedMissions), err)
	}
	listedJSON, err := json.Marshal(listedMissions)
	if err != nil || strings.Contains(string(listedJSON), legacySecret) {
		t.Fatalf("ListMissions exposed legacy credential: err=%v", err)
	}
	seedLegacyRows()
	legacyEvents, err := store.WithOrganization(orgA).ListEvents(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	legacyEventsJSON, err := json.Marshal(legacyEvents)
	if err != nil || strings.Contains(string(legacyEventsJSON), legacySecret) {
		t.Fatalf("ListEvents exposed legacy credential: err=%v", err)
	}

	// The migrator must refuse ambiguous legacy ownership instead of guessing.
	ownerlessID := "mis_" + uuid.NewString()
	adminDB, err := sql.Open("pgx", migratorDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer adminDB.Close()
	seedTx, err := adminDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seedTx.ExecContext(ctx, `ALTER TABLE agent_missions DISABLE ROW LEVEL SECURITY`); err != nil {
		_ = seedTx.Rollback()
		t.Fatal(err)
	}
	if _, err := seedTx.ExecContext(ctx, `INSERT INTO agent_missions (id,version,objective,state,plan,approvals,artifacts,created_at,updated_at) VALUES ($1,1,'ambiguous legacy row','READY','[]'::jsonb,'[]'::jsonb,'[]'::jsonb,NOW(),NOW())`, ownerlessID); err != nil {
		_ = seedTx.Rollback()
		t.Fatal(err)
	}
	if _, err := seedTx.ExecContext(ctx, `ALTER TABLE agent_missions ENABLE ROW LEVEL SECURITY`); err != nil {
		_ = seedTx.Rollback()
		t.Fatal(err)
	}
	if _, err := seedTx.ExecContext(ctx, `ALTER TABLE agent_missions FORCE ROW LEVEL SECURITY`); err != nil {
		_ = seedTx.Rollback()
		t.Fatal(err)
	}
	if err := seedTx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := MigratePostgresAgentSchema(ctx, migratorDSN, key); err == nil || !strings.Contains(err.Error(), "valid owner=1") {
		t.Fatalf("migration error=%v, want fail-closed ownerless-row refusal", err)
	}
	cleanupTx, err := adminDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cleanupTx.ExecContext(ctx, `ALTER TABLE agent_missions DISABLE ROW LEVEL SECURITY`); err != nil {
		_ = cleanupTx.Rollback()
		t.Fatal(err)
	}
	if _, err := cleanupTx.ExecContext(ctx, `DELETE FROM agent_missions WHERE id=$1`, ownerlessID); err != nil {
		_ = cleanupTx.Rollback()
		t.Fatal(err)
	}
	if _, err := cleanupTx.ExecContext(ctx, `ALTER TABLE agent_missions ENABLE ROW LEVEL SECURITY`); err != nil {
		_ = cleanupTx.Rollback()
		t.Fatal(err)
	}
	if _, err := cleanupTx.ExecContext(ctx, `ALTER TABLE agent_missions FORCE ROW LEVEL SECURITY`); err != nil {
		_ = cleanupTx.Rollback()
		t.Fatal(err)
	}
	if err := cleanupTx.Commit(); err != nil {
		t.Fatal(err)
	}
	workspaceIdentityMissingID := "mis_" + uuid.NewString()
	seedWorkspaceTx, err := adminDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seedWorkspaceTx.ExecContext(ctx, `ALTER TABLE agent_missions DISABLE ROW LEVEL SECURITY`); err != nil {
		_ = seedWorkspaceTx.Rollback()
		t.Fatal(err)
	}
	if _, err := seedWorkspaceTx.ExecContext(ctx, `INSERT INTO agent_missions (id,version,objective,organization_id,state,plan,approvals,artifacts,created_at,updated_at) VALUES ($1,1,'legacy runnable workspace',$2,'READY','[]'::jsonb,'[]'::jsonb,'[]'::jsonb,NOW(),NOW())`, workspaceIdentityMissingID, orgA); err != nil {
		_ = seedWorkspaceTx.Rollback()
		t.Fatal(err)
	}
	if _, err := seedWorkspaceTx.ExecContext(ctx, `ALTER TABLE agent_missions ENABLE ROW LEVEL SECURITY`); err != nil {
		_ = seedWorkspaceTx.Rollback()
		t.Fatal(err)
	}
	if _, err := seedWorkspaceTx.ExecContext(ctx, `ALTER TABLE agent_missions FORCE ROW LEVEL SECURITY`); err != nil {
		_ = seedWorkspaceTx.Rollback()
		t.Fatal(err)
	}
	if err := seedWorkspaceTx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := MigratePostgresAgentSchema(ctx, migratorDSN, key); err == nil || !strings.Contains(err.Error(), "without a persisted workspace identity") {
		t.Fatalf("migration error=%v, want fail-closed workspace-identity refusal", err)
	}
	cleanupWorkspaceTx, err := adminDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cleanupWorkspaceTx.ExecContext(ctx, `ALTER TABLE agent_missions DISABLE ROW LEVEL SECURITY`); err != nil {
		_ = cleanupWorkspaceTx.Rollback()
		t.Fatal(err)
	}
	if _, err := cleanupWorkspaceTx.ExecContext(ctx, `DELETE FROM agent_missions WHERE id=$1`, workspaceIdentityMissingID); err != nil {
		_ = cleanupWorkspaceTx.Rollback()
		t.Fatal(err)
	}
	if _, err := cleanupWorkspaceTx.ExecContext(ctx, `ALTER TABLE agent_missions ENABLE ROW LEVEL SECURITY`); err != nil {
		_ = cleanupWorkspaceTx.Rollback()
		t.Fatal(err)
	}
	if _, err := cleanupWorkspaceTx.ExecContext(ctx, `ALTER TABLE agent_missions FORCE ROW LEVEL SECURITY`); err != nil {
		_ = cleanupWorkspaceTx.Rollback()
		t.Fatal(err)
	}
	if err := cleanupWorkspaceTx.Commit(); err != nil {
		t.Fatal(err)
	}
	t.Run("postgres recovery uses trusted tenant enumeration", func(t *testing.T) {
		redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
		if redisURL == "" {
			t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
		}
		queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:postgres-recovery")
		if err != nil {
			t.Fatal(err)
		}
		authStore, err := NewAuthStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		_, organization, _, err := authStore.ProvisionOAuthUser(map[string]any{"email": "pg-recovery@example.test", "name": "Recovery Test"}, "integration")
		if err != nil {
			t.Fatal(err)
		}
		recoveryStore, err := OpenPostgresRuntimeStore(ctx, runtimeDSN, key)
		if err != nil {
			t.Fatal(err)
		}
		runtime, err := NewRuntime(RuntimeConfig{Store: recoveryStore, RedisQueue: queue, Planner: RulePlanner{}, WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
		if err != nil {
			_ = recoveryStore.Close()
			t.Fatal(err)
		}
		defer runtime.Close(context.Background())
		runtime.SetAuthStore(authStore)
		workspace := t.TempDir()
		workspaceIdentity, err := workspaceDirectoryIdentity(workspace)
		if err != nil {
			t.Fatal(err)
		}
		pending := Mission{ID: "mis_" + uuid.NewString(), Version: 1, Objective: "recover tenant work", Workspace: workspace, WorkspaceIdentity: workspaceIdentity, OrganizationID: organization.ID, AutoRun: true, State: MissionReady, Plan: []Step{}, Approvals: []Approval{}, Artifacts: []ArtifactManifest{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		if err := recoveryStore.WithOrganization(organization.ID).PutMission(pending); err != nil {
			t.Fatal(err)
		}
		if err := runtime.resumePending(ctx); err != nil {
			t.Fatalf("tenant-aware recovery failed: %v", err)
		}
		found := false
		for _, job := range queue.List(QueuePending) {
			if job.MissionID == pending.ID && job.OrganizationID == organization.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("tenant-aware recovery did not enqueue mission %s for organization %s", pending.ID, organization.ID)
		}
	})
	for _, missingColumn := range []struct{ table, column string }{
		{"agent_missions", "id"}, {"agent_missions", "version"}, {"agent_missions", "objective"}, {"agent_missions", "model"},
		{"agent_missions", "workspace"}, {"agent_missions", "project_id"}, {"agent_missions", "auto_run"}, {"agent_missions", "state"},
		{"agent_missions", "plan"}, {"agent_missions", "approvals"}, {"agent_missions", "artifacts"}, {"agent_missions", "last_error"},
		{"agent_missions", "created_at"}, {"agent_missions", "updated_at"}, {"agent_missions", "completed_at"},
		{"agent_events", "id"}, {"agent_events", "mission_id"}, {"agent_events", "type"}, {"agent_events", "step_id"},
		{"agent_events", "payload"}, {"agent_events", "created_at"},
	} {
		legacyColumn := "legacy_missing_" + missingColumn.column
		if _, err := adminDB.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE public.%s RENAME COLUMN %s TO %s`, missingColumn.table, missingColumn.column, legacyColumn)); err != nil {
			t.Fatalf("rename %s.%s for schema preflight test: %v", missingColumn.table, missingColumn.column, err)
		}
		migrationErr := MigratePostgresAgentSchema(ctx, migratorDSN, key)
		if migrationErr == nil || !strings.Contains(migrationErr.Error(), "older than the supported baseline") || !strings.Contains(migrationErr.Error(), missingColumn.table) || !strings.Contains(migrationErr.Error(), missingColumn.column) {
			t.Fatalf("missing %s.%s migration error=%v, want explicit unsupported-baseline refusal", missingColumn.table, missingColumn.column, migrationErr)
		}
		if _, err := adminDB.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE public.%s RENAME COLUMN %s TO %s`, missingColumn.table, legacyColumn, missingColumn.column)); err != nil {
			t.Fatalf("restore %s.%s after schema preflight test: %v", missingColumn.table, missingColumn.column, err)
		}
		if err := MigratePostgresAgentSchema(ctx, migratorDSN, key); err != nil {
			t.Fatalf("migration after restoring supported schema shape %s.%s failed: %v", missingColumn.table, missingColumn.column, err)
		}
	}
}

func TestDistributedRedisRetriesDeadLetterReplay(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:integration:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 2)
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := queue.Claim("integration-worker", time.Now().UTC())
	if err != nil || !ok || claimed.ID != job.ID {
		t.Fatalf("first claim failed: job=%+v ok=%v err=%v", claimed, ok, err)
	}
	if _, err := queue.Nack(claimed, errors.New("first failure")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	claimed, ok, err = queue.Claim("integration-worker", time.Now().UTC().Add(2*time.Second))
	if err != nil || !ok || claimed.Attempts != 2 {
		t.Fatalf("second claim failed: job=%+v ok=%v err=%v", claimed, ok, err)
	}
	dead, err := queue.Nack(claimed, errors.New("second failure"))
	if err != nil || dead.Status != QueueDeadLetter {
		t.Fatalf("dead-letter transition failed: job=%+v err=%v", dead, err)
	}
	if _, err := queue.Replay(job.ID); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err = queue.Claim("integration-worker-replay", time.Now().UTC())
	if err != nil || !ok || claimed.Attempts != 1 {
		t.Fatalf("replay claim failed: job=%+v ok=%v err=%v", claimed, ok, err)
	}
	if err := queue.Ack(claimed); err != nil {
		t.Fatal(err)
	}
	jobs := queue.List(QueueSucceeded)
	found := false
	for _, candidate := range jobs {
		if candidate.ID == job.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("acknowledged job was not listed as succeeded")
	}
}

func TestDistributedRedisClaimIsIdempotentAndReclaimsExpiredLease(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:lease-integration:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	missionID := "mis_" + uuid.NewString()
	first, err := queue.Enqueue(missionID, 3)
	if err != nil {
		t.Fatal(err)
	}
	if first.AvailableAtUnixMilli <= 0 || first.UpdatedAtUnixMilli <= 0 || first.AvailableAt.After(time.Now().UTC().Add(2*time.Second)) {
		t.Fatalf("Redis enqueue did not expose server-authoritative timestamps: %+v", first)
	}
	second, err := queue.Enqueue(missionID, 3)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("duplicate Redis enqueue: first=%+v second=%+v", first, second)
	}
	claimed, ok, err := queue.Claim("worker-a", time.Now().UTC())
	if err != nil || !ok || claimed.ID != first.ID {
		t.Fatalf("first FIFO claim failed: job=%+v ok=%v err=%v", claimed, ok, err)
	}
	skewJob, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	clientFuture := time.Now().UTC().Add(24 * time.Hour)
	skewClaim, ok, err := queue.Claim("clock-skew-claim", clientFuture)
	if err != nil || !ok || skewClaim.ID != skewJob.ID {
		t.Fatalf("future-clock claim failed: job=%+v ok=%v err=%v", skewClaim, ok, err)
	}
	if skewClaim.LockedAt == nil || skewClaim.LockedAt.After(time.Now().UTC().Add(2*time.Second)) || skewClaim.UpdatedAt.After(time.Now().UTC().Add(2*time.Second)) {
		t.Fatalf("Redis claim exposed client-future timestamps: %+v", skewClaim)
	}
	rawValue, err := queue.do(context.Background(), "GET", queue.jobKey(skewJob.ID))
	if err != nil {
		t.Fatal(err)
	}
	rawJSON, ok := rawValue.(string)
	if !ok {
		t.Fatalf("Redis raw job is not a string: %T", rawValue)
	}
	var rawJob map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &rawJob); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"created_at", "available_at", "locked_at", "updated_at"} {
		if rawJob[key] != nil {
			t.Fatalf("raw Redis field %s retained a client-clock timestamp: %v", key, rawJob[key])
		}
	}
	for _, key := range []string{"created_at_ms", "available_at_ms", "locked_at_ms", "updated_at_ms"} {
		value, ok := rawJob[key].(float64)
		if !ok || value <= 0 {
			t.Fatalf("raw Redis field %s is missing authoritative epoch-ms: %v", key, rawJob[key])
		}
	}
	if err := queue.Ack(skewClaim); err != nil {
		t.Fatal(err)
	}
	if early, ok, err := queue.Claim("clock-skew-worker", time.Now().UTC().Add(24*time.Hour)); err != nil || ok {
		t.Fatalf("client clock skew reclaimed an unexpired lease: job=%+v ok=%v err=%v", early, ok, err)
	}
	expireRedisLeaseForTest(t, context.Background(), queue, first.ID)
	reclaimed, ok, err := queue.Claim("worker-b", time.Now().UTC().Add(time.Minute))
	if err != nil || !ok || reclaimed.ID != first.ID || reclaimed.Attempts != 2 || reclaimed.WorkerID != "worker-b" {
		t.Fatalf("expired lease was not reclaimed: job=%+v ok=%v err=%v", reclaimed, ok, err)
	}
	if reclaimed.AvailableAtUnixMilli <= 0 || reclaimed.AvailableAt.After(time.Now().UTC().Add(2*time.Second)) {
		t.Fatalf("reclaim exposed client-future available_at: %+v", reclaimed)
	}
}

func TestDistributedOTLPCollector(t *testing.T) {
	endpoint := os.Getenv("OLLAMA_AGENT_TEST_OTLP_ENDPOINT")
	if endpoint == "" {
		t.Skip("OLLAMA_AGENT_TEST_OTLP_ENDPOINT is not configured")
	}
	if len(endpoint) >= 7 && endpoint[:7] == "http://" {
		t.Setenv("OLLAMA_AGENT_OTLP_ALLOW_INSECURE", "1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	telemetry, err := NewTelemetry(ctx, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	_, span := telemetry.Start(ctx, "integration.collector", map[string]string{"test": "distributed"})
	span.End()
	if err := telemetry.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestDistributedRedisHeartbeatReclaimAndAtomicMoveDue(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:heartbeat-integration:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	queue.leaseDuration = 120 * time.Millisecond

	job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	first, ok, err := queue.Claim("worker-first", time.Now().UTC())
	if err != nil || !ok || first.ID != job.ID {
		t.Fatalf("initial claim: job=%+v ok=%v err=%v", first, ok, err)
	}
	if err := queue.renewLease(first); err != nil {
		t.Fatal(err)
	}
	if err := queue.reclaimExpired(ctx, time.Now().UTC().Add(40*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if current, err := queue.get(job.ID); err != nil || current.Status != QueueRunning || current.LeaseToken != first.LeaseToken {
		t.Fatalf("heartbeat lease was incorrectly reclaimed: job=%+v err=%v", current, err)
	}
	expireRedisLeaseForTest(t, ctx, queue, job.ID)
	if err := queue.reclaimExpired(ctx, time.Now().UTC().Add(200*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if current, err := queue.get(job.ID); err != nil || current.Status != QueuePending || current.LeaseToken != "" {
		t.Fatalf("expired claim was not safely reclaimed: job=%+v err=%v", current, err)
	}
	second, ok, err := queue.Claim("worker-second", time.Now().UTC().Add(300*time.Millisecond))
	if err != nil || !ok || second.ID != job.ID || second.LeaseToken == first.LeaseToken {
		t.Fatalf("reclaimed claim: job=%+v ok=%v err=%v", second, ok, err)
	}
	if err := queue.Ack(first); !errors.Is(err, ErrQueueLeaseLost) {
		t.Fatalf("stale worker ack error = %v, want lease lost", err)
	}
	if err := queue.Ack(second); err != nil {
		t.Fatal(err)
	}

	queue.leaseDuration = 30 * time.Millisecond
	expiring, err := queue.Enqueue("mis_"+uuid.NewString(), 1)
	if err != nil {
		t.Fatal(err)
	}
	expiringClaim, ok, err := queue.Claim("worker-expiring", time.Now().UTC())
	if err != nil || !ok || expiringClaim.ID != expiring.ID {
		t.Fatalf("expiring claim: job=%+v ok=%v err=%v", expiringClaim, ok, err)
	}
	time.Sleep(40 * time.Millisecond)
	if err := queue.renewLease(expiringClaim); !errors.Is(err, ErrQueueLeaseLost) {
		t.Fatalf("expired Redis heartbeat error = %v, want lease lost", err)
	}
	if err := queue.Ack(expiringClaim); !errors.Is(err, ErrQueueLeaseLost) {
		t.Fatalf("expired Redis ack error = %v, want lease lost", err)
	}
	if _, err := queue.Nack(expiringClaim, ErrQueueNonRetryable); !errors.Is(err, ErrQueueLeaseLost) {
		t.Fatalf("expired Redis nack error = %v, want lease lost", err)
	}
	if _, ok, err := queue.Claim("worker-expiry-cleanup", time.Now().UTC().Add(100*time.Millisecond)); err != nil || ok {
		t.Fatalf("expired max-attempt claim was not dead-lettered before next scenario: ok=%v err=%v", ok, err)
	}

	delayed, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	delayedClaim, ok, err := queue.Claim("worker-delayed", time.Now().UTC())
	if err != nil || !ok || delayedClaim.ID != delayed.ID {
		t.Fatalf("delayed source claim: job=%+v ok=%v err=%v", delayedClaim, ok, err)
	}
	if _, err := queue.Nack(delayedClaim, errors.New("retry once")); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "ZADD", queue.delayedKey(), "0", delayed.ID); err != nil {
		t.Fatal(err)
	}
	future := time.Now().UTC().Add(3 * time.Second)
	if err := queue.moveDue(ctx, future); err != nil {
		t.Fatal(err)
	}
	if err := queue.moveDue(ctx, future); err != nil {
		t.Fatal(err)
	}
	moved, ok, err := queue.Claim("worker-delayed-retry", future)
	if err != nil || !ok || moved.ID != delayed.ID {
		t.Fatalf("delayed job was not moved atomically: job=%+v ok=%v err=%v", moved, ok, err)
	}
	if _, ok, err := queue.Claim("worker-no-duplicate", future); err != nil || ok {
		t.Fatalf("atomic move produced a duplicate pending item: ok=%v err=%v", ok, err)
	}
	if err := queue.Ack(moved); err != nil {
		t.Fatal(err)
	}
}

func TestDistributedRedisQueueDrainWaitsForUncooperativeHandler(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:drain-integration:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	queue.leaseDuration = 90 * time.Millisecond
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	claim, ok, err := queue.Claim("worker-drain", time.Now().UTC())
	if err != nil || !ok || claim.ID != job.ID || claim.LeaseUntil.IsZero() || claim.LeaseUntilUnixMilli == 0 {
		t.Fatalf("claim/deadline = %+v ok=%v err=%v", claim, ok, err)
	}
	if listed, err := queue.get(job.ID); err != nil || listed.LeaseUntilUnixMilli != claim.LeaseUntilUnixMilli {
		t.Fatalf("persisted lease deadline = %+v err=%v", listed, err)
	}
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- queue.runWithHeartbeat(workerCtx, claim, func(context.Context, QueueJob) error {
			close(started)
			<-release // models a provider call that ignores cancellation
			return nil
		})
	}()
	<-started
	cancelWorker()
	select {
	case err := <-done:
		t.Fatalf("Redis worker returned before handler stopped: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	if current, err := queue.get(job.ID); err != nil || current.Status != QueueRunning {
		t.Fatalf("Redis job was released while handler remained active: %+v err=%v", current, err)
	}
	close(release)
	runErr := <-done
	if !errors.Is(runErr, ErrQueueNonRetryable) {
		t.Fatalf("drained Redis cancellation error = %v, want terminal/non-retryable", runErr)
	}
	failed, err := queue.Nack(claim, runErr)
	if err != nil || failed.Status != QueueFailed || failed.LeaseUntilUnixMilli != 0 {
		t.Fatalf("Redis terminal Nack = %+v err=%v", failed, err)
	}
}

func TestDistributedRedisDrainRenewsLeaseImmediatelyNearExpiry(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:drain-near-expiry:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	queue.leaseDuration = 100 * time.Millisecond
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	claim, ok, err := queue.Claim("worker-drain-near-expiry", time.Now().UTC())
	if err != nil || !ok {
		t.Fatalf("claim = %+v ok=%v err=%v", claim, ok, err)
	}
	queue.leaseDuration = 300 * time.Millisecond
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- queue.runWithHeartbeat(workerCtx, claim, func(context.Context, QueueJob) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	time.Sleep(80 * time.Millisecond)
	cancelWorker()
	time.Sleep(30 * time.Millisecond)
	if contender, ok, err := queue.Claim("worker-contender", time.Now().UTC()); err != nil || ok {
		t.Fatalf("Redis contender reclaimed lease during drain: job=%+v ok=%v err=%v", contender, ok, err)
	}
	close(release)
	runErr := <-done
	if !errors.Is(runErr, ErrQueueNonRetryable) {
		t.Fatalf("drain result = %v, want terminal non-retryable", runErr)
	}
	failed, err := queue.Nack(claim, runErr)
	if err != nil || failed.Status != QueueFailed {
		t.Fatalf("terminal Nack = %+v err=%v", failed, err)
	}
	if _, err := queue.Replay(job.ID); err == nil {
		t.Fatal("QueueFailed job unexpectedly became replayable")
	}
}

func TestDistributedRedisCancellationDominatesHandlerCompletion(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:cancel-completion:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	claim, ok, err := queue.Claim("worker-cancel-race", time.Now().UTC())
	if err != nil || !ok {
		t.Fatalf("claim = %+v ok=%v err=%v", claim, ok, err)
	}
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- queue.runWithHeartbeat(workerCtx, claim, func(handlerCtx context.Context, _ QueueJob) error {
			close(started)
			<-handlerCtx.Done()
			return nil
		})
	}()
	<-started
	cancelWorker()
	runErr := <-done
	if !errors.Is(runErr, ErrQueueNonRetryable) {
		t.Fatalf("cancellation result = %v, want terminal non-retryable", runErr)
	}
	failed, err := queue.Nack(claim, runErr)
	if err != nil || failed.Status != QueueFailed {
		t.Fatalf("cancelled job was acknowledged/retried: %+v err=%v", failed, err)
	}
	if _, err := queue.Replay(job.ID); err == nil {
		t.Fatal("QueueFailed job unexpectedly became replayable")
	}
}

func TestDistributedRedisStartCancellationDominatesNilHandler(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:start-cancel-race:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	started := make(chan struct{})
	queue.Start(workerCtx, "worker-start-cancel-race", func(handlerCtx context.Context, _ QueueJob) error {
		close(started)
		<-handlerCtx.Done()
		return nil
	})
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("Redis queue Start did not dispatch the test job")
	}
	cancelWorker()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, candidate := range queue.List(QueueFailed) {
			if candidate.ID == job.ID {
				return
			}
		}
		for _, candidate := range queue.List(QueueSucceeded) {
			if candidate.ID == job.ID {
				t.Fatalf("cancelled handler was acknowledged as succeeded: %+v", candidate)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("cancelled handler was not terminalized: pending=%+v running=%+v", queue.List(QueuePending), queue.List(QueueRunning))
}

func TestDistributedRedisWrongTypesDoNotPartiallyMutateQueue(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	newQueue := func(t *testing.T, name string) *RedisQueue {
		t.Helper()
		queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:wrongtype:"+name+":"+uuid.NewString())
		if err != nil {
			t.Fatal(err)
		}
		return queue
	}
	getRaw := func(t *testing.T, queue *RedisQueue, id string) map[string]any {
		t.Helper()
		value, err := queue.do(ctx, "GET", queue.jobKey(id))
		if err != nil {
			t.Fatal(err)
		}
		text, ok := value.(string)
		if !ok {
			t.Fatalf("raw job is %T, want string", value)
		}
		var job map[string]any
		if err := json.Unmarshal([]byte(text), &job); err != nil {
			t.Fatal(err)
		}
		return job
	}
	t.Run("delayed-move-preserves-sorted-set-on-wrong-pending-type", func(t *testing.T) {
		queue := newQueue(t, "move")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "ZADD", queue.delayedKey(), "0", job.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "SET", queue.pendingKey(), "wrong-type"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := queue.Claim("worker", time.Now()); err == nil {
			t.Fatal("Claim succeeded despite pending list WRONGTYPE")
		}
		if score, err := queue.do(ctx, "ZSCORE", queue.delayedKey(), job.ID); err != nil || score == nil {
			t.Fatalf("delayed job was lost after error: score=%v err=%v", score, err)
		}
		if status := getRaw(t, queue, job.ID)["status"]; status != string(QueuePending) {
			t.Fatalf("job status changed on failed move: %v", status)
		}
	})
	t.Run("claim-preserves-pending-id-on-wrong-lease-type", func(t *testing.T) {
		queue := newQueue(t, "claim")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "SET", queue.leaseKey(), "wrong-type"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := queue.Claim("worker", time.Now()); err == nil {
			t.Fatal("Claim succeeded despite lease index WRONGTYPE")
		}
		if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(1) {
			t.Fatalf("pending ID was consumed: length=%v err=%v", length, err)
		}
		if status := getRaw(t, queue, job.ID)["status"]; status != string(QueuePending) {
			t.Fatalf("job status changed on failed claim: %v", status)
		}
	})
	t.Run("nack-preserves-running-claim-on-wrong-delayed-type", func(t *testing.T) {
		queue := newQueue(t, "nack")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
		if err != nil {
			t.Fatal(err)
		}
		claim, ok, err := queue.Claim("worker", time.Now())
		if err != nil || !ok || claim.ID != job.ID {
			t.Fatalf("claim failed: %+v ok=%v err=%v", claim, ok, err)
		}
		if _, err := queue.do(ctx, "SET", queue.delayedKey(), "wrong-type"); err != nil {
			t.Fatal(err)
		}
		if _, err := queue.Nack(claim, errors.New("retry")); err == nil {
			t.Fatal("Nack succeeded despite delayed zset WRONGTYPE")
		}
		if raw := getRaw(t, queue, job.ID); raw["status"] != string(QueueRunning) || raw["lease_token"] != claim.LeaseToken {
			t.Fatalf("running claim was partially mutated: %#v", raw)
		}
		if score, err := queue.do(ctx, "ZSCORE", queue.leaseKey(), job.ID); err != nil || score == nil {
			t.Fatalf("lease index was removed after failed Nack: score=%v err=%v", score, err)
		}
	})
	t.Run("reclaim-preserves-expired-lease-on-wrong-dead-letter-type", func(t *testing.T) {
		queue := newQueue(t, "reclaim")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
		if err != nil {
			t.Fatal(err)
		}
		claim, ok, err := queue.Claim("worker", time.Now())
		if err != nil || !ok || claim.ID != job.ID {
			t.Fatalf("claim failed: %+v ok=%v err=%v", claim, ok, err)
		}
		expireRedisLeaseForTest(t, ctx, queue, job.ID)
		if _, err := queue.do(ctx, "SET", queue.deadKey(), "wrong-type"); err != nil {
			t.Fatal(err)
		}
		if _, _, err := queue.Claim("new-worker", time.Now()); err == nil {
			t.Fatal("Claim succeeded despite dead-letter list WRONGTYPE during reclaim")
		}
		if score, err := queue.do(ctx, "ZSCORE", queue.leaseKey(), job.ID); err != nil || score == nil {
			t.Fatalf("expired lease was removed on failed reclaim: score=%v err=%v", score, err)
		}
		if raw := getRaw(t, queue, job.ID); raw["status"] != string(QueueRunning) || raw["lease_token"] != claim.LeaseToken {
			t.Fatalf("running claim was partially mutated: %#v", raw)
		}
	})
	t.Run("enqueue-does-not-create-partial-job-on-wrong-pending-type", func(t *testing.T) {
		queue := newQueue(t, "enqueue")
		if _, err := queue.do(ctx, "SET", queue.pendingKey(), "wrong-type"); err != nil {
			t.Fatal(err)
		}
		missionID := "mis_" + uuid.NewString()
		if _, err := queue.Enqueue(missionID, 3); err == nil {
			t.Fatal("Enqueue succeeded despite pending list WRONGTYPE")
		}
		if value, err := queue.do(ctx, "GET", queue.missionKey(missionID)); err != nil || value != nil {
			t.Fatalf("mission index was partially created: value=%v err=%v", value, err)
		}
		if value, err := queue.do(ctx, "KEYS", queue.key("job:*")); err != nil || len(value.([]any)) != 0 {
			t.Fatalf("partial job record was created: value=%v err=%v", value, err)
		}
	})
	t.Run("replay-preserves-dead-letter-on-wrong-pending-type", func(t *testing.T) {
		queue := newQueue(t, "replay")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 1)
		if err != nil {
			t.Fatal(err)
		}
		claim, ok, err := queue.Claim("worker", time.Now())
		if err != nil || !ok || claim.ID != job.ID {
			t.Fatalf("claim failed: %+v ok=%v err=%v", claim, ok, err)
		}
		if _, err := queue.Nack(claim, errors.New("terminal")); err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "SET", queue.pendingKey(), "wrong-type"); err != nil {
			t.Fatal(err)
		}
		if _, err := queue.Replay(job.ID); err == nil {
			t.Fatal("Replay succeeded despite pending list WRONGTYPE")
		}
		if raw := getRaw(t, queue, job.ID); raw["status"] != string(QueueDeadLetter) {
			t.Fatalf("dead-letter status changed on failed replay: %#v", raw)
		}
		if members, err := queue.do(ctx, "LRANGE", queue.deadKey(), "0", "-1"); err != nil || len(members.([]any)) != 1 {
			t.Fatalf("dead-letter index changed on failed replay: members=%v err=%v", members, err)
		}
	})
}

func TestDistributedRedisDelayedJobsWithEqualScoreRetainRetryOrder(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:delayed-fifo:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	first, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	second, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	claim, ok, err := queue.Claim("fifo-worker", time.Now())
	if err != nil || !ok || claim.ID != first.ID {
		t.Fatalf("first queue claim failed: %+v ok=%v err=%v", claim, ok, err)
	}
	if _, err := queue.Nack(claim, errors.New("retry first")); err != nil {
		t.Fatal(err)
	}
	claim, ok, err = queue.Claim("fifo-worker", time.Now())
	if err != nil || !ok || claim.ID != second.ID {
		t.Fatalf("second queue claim failed: %+v ok=%v err=%v", claim, ok, err)
	}
	if _, err := queue.Nack(claim, errors.New("retry second")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{first.ID, second.ID} {
		if _, err := queue.do(ctx, "ZADD", queue.delayedKey(), "0", id); err != nil {
			t.Fatal(err)
		}
	}
	claim, ok, err = queue.Claim("fifo-worker", time.Now())
	if err != nil || !ok || claim.ID != first.ID {
		t.Fatalf("equal-score delayed jobs did not preserve enqueue/retry order: got=%+v ok=%v err=%v; want=%s", claim, ok, err, first.ID)
	}
}

func TestDistributedRedisRejectsOrphanAndMismatchedJobRecords(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	newQueue := func(t *testing.T, name string) *RedisQueue {
		t.Helper()
		queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:orphan:"+name+":"+uuid.NewString())
		if err != nil {
			t.Fatal(err)
		}
		return queue
	}
	t.Run("pending-id-without-job-is-not-consumed", func(t *testing.T) {
		queue := newQueue(t, "claim")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "DEL", queue.jobKey(job.ID)); err != nil {
			t.Fatal(err)
		}
		if _, _, err := queue.Claim("worker", time.Now()); err == nil {
			t.Fatal("Claim accepted a pending ID whose job record is missing")
		}
		if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(1) {
			t.Fatalf("orphan pending ID was consumed: length=%v err=%v", length, err)
		}
	})
	t.Run("pending-record-with-mismatched-id-is-not-consumed", func(t *testing.T) {
		queue := newQueue(t, "mismatch")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := queue.do(ctx, "GET", queue.jobKey(job.ID))
		if err != nil {
			t.Fatal(err)
		}
		var stored map[string]any
		if err := json.Unmarshal([]byte(raw.(string)), &stored); err != nil {
			t.Fatal(err)
		}
		stored["id"] = "different-job-id"
		encoded, err := json.Marshal(stored)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "SET", queue.jobKey(job.ID), string(encoded)); err != nil {
			t.Fatal(err)
		}
		if _, _, err := queue.Claim("worker", time.Now()); err == nil {
			t.Fatal("Claim accepted a job record whose embedded ID does not match its index")
		}
		if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(1) {
			t.Fatalf("mismatched pending ID was consumed: length=%v err=%v", length, err)
		}
	})
	t.Run("delayed-orphan-is-not-partially-moved", func(t *testing.T) {
		queue := newQueue(t, "delayed")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "LREM", queue.pendingKey(), "0", job.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "ZADD", queue.delayedKey(), "0", job.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "DEL", queue.jobKey(job.ID)); err != nil {
			t.Fatal(err)
		}
		if _, _, err := queue.Claim("worker", time.Now()); err == nil {
			t.Fatal("move-due accepted a delayed ID whose job record is missing")
		}
		if score, err := queue.do(ctx, "ZSCORE", queue.delayedKey(), job.ID); err != nil || score == nil {
			t.Fatalf("orphan delayed ID was removed after error: score=%v err=%v", score, err)
		}
		if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(0) {
			t.Fatalf("orphan delayed ID was partially moved: pending length=%v err=%v", length, err)
		}
	})
}

func TestDistributedRedisRejectsQueueIndexStatusMismatch(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	newQueue := func(t *testing.T, name string) *RedisQueue {
		t.Helper()
		queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:index-status:"+name+":"+uuid.NewString())
		if err != nil {
			t.Fatal(err)
		}
		return queue
	}
	setStatus := func(t *testing.T, queue *RedisQueue, jobID string, status QueueStatus) {
		t.Helper()
		value, err := queue.do(ctx, "GET", queue.jobKey(jobID))
		if err != nil {
			t.Fatal(err)
		}
		var stored map[string]any
		if err := json.Unmarshal([]byte(value.(string)), &stored); err != nil {
			t.Fatal(err)
		}
		stored["status"] = string(status)
		encoded, err := json.Marshal(stored)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "SET", queue.jobKey(jobID), string(encoded)); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("pending-index-does-not-consume-nonpending-job", func(t *testing.T) {
		queue := newQueue(t, "pending")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
		if err != nil {
			t.Fatal(err)
		}
		setStatus(t, queue, job.ID, QueueSucceeded)
		if _, _, err := queue.Claim("worker", time.Now()); err == nil {
			t.Fatal("Claim accepted a pending index pointing to a succeeded job")
		}
		if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(1) {
			t.Fatalf("pending index was consumed on status mismatch: length=%v err=%v", length, err)
		}
	})
	t.Run("delayed-index-does-not-move-nonpending-job", func(t *testing.T) {
		queue := newQueue(t, "delayed")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "LREM", queue.pendingKey(), "0", job.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "ZADD", queue.delayedKey(), "0", job.ID); err != nil {
			t.Fatal(err)
		}
		setStatus(t, queue, job.ID, QueueSucceeded)
		if _, _, err := queue.Claim("worker", time.Now()); err == nil {
			t.Fatal("move-due accepted a delayed index pointing to a succeeded job")
		}
		if score, err := queue.do(ctx, "ZSCORE", queue.delayedKey(), job.ID); err != nil || score == nil {
			t.Fatalf("delayed index was removed on status mismatch: score=%v err=%v", score, err)
		}
		if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(0) {
			t.Fatalf("mismatched delayed job was partially moved: pending=%v err=%v", length, err)
		}
	})
	t.Run("lease-index-does-not-reclaim-nonrunning-job", func(t *testing.T) {
		queue := newQueue(t, "lease")
		job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
		if err != nil {
			t.Fatal(err)
		}
		claim, ok, err := queue.Claim("worker", time.Now())
		if err != nil || !ok || claim.ID != job.ID {
			t.Fatalf("claim failed: %+v ok=%v err=%v", claim, ok, err)
		}
		expireRedisLeaseForTest(t, ctx, queue, job.ID)
		setStatus(t, queue, job.ID, QueueSucceeded)
		if _, _, err := queue.Claim("new-worker", time.Now()); err == nil {
			t.Fatal("reclaim accepted a lease index pointing to a succeeded job")
		}
		if score, err := queue.do(ctx, "ZSCORE", queue.leaseKey(), job.ID); err != nil || score == nil {
			t.Fatalf("lease index was removed on status mismatch: score=%v err=%v", score, err)
		}
	})
}

func TestDistributedRedisRejectsInvalidArgumentsBeforeMutation(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:argv-preflight:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "EVAL", redisClaimScript, "5", queue.pendingKey(), queue.delayedKey(), queue.leaseKey(), queue.key("job:"), queue.deadKey(), "worker", "token", "0"); err == nil {
		t.Fatal("Claim accepted zero lease duration")
	}
	if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(1) {
		t.Fatalf("invalid Claim arguments consumed pending job: length=%v err=%v", length, err)
	}
	claim, ok, err := queue.Claim("worker", time.Now())
	if err != nil || !ok || claim.ID != job.ID {
		t.Fatalf("valid claim failed: %+v ok=%v err=%v", claim, ok, err)
	}
	sequenceBefore, err := queue.do(ctx, "GET", queue.key("sequence"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "EVAL", redisNackScript, "6", queue.jobKey(job.ID), queue.leaseKey(), queue.delayedKey(), queue.key("sequence"), queue.pendingKey(), queue.deadKey(), claim.WorkerID, claim.LeaseToken, "bad retry", "unknown-mode", job.ID, "0"); err == nil {
		t.Fatal("Nack accepted an unknown mode")
	}
	sequenceAfter, err := queue.do(ctx, "GET", queue.key("sequence"))
	if err != nil || sequenceAfter != sequenceBefore {
		t.Fatalf("invalid Nack arguments changed sequence: before=%v after=%v err=%v", sequenceBefore, sequenceAfter, err)
	}
	current, err := queue.get(job.ID)
	if err != nil || current.Status != QueueRunning || current.LeaseToken != claim.LeaseToken {
		t.Fatalf("invalid Nack arguments mutated the running claim: job=%+v err=%v", current, err)
	}
	if _, err := queue.do(ctx, "EVAL", redisClaimScript, "5", queue.pendingKey(), queue.pendingKey(), queue.leaseKey(), queue.key("job:"), queue.deadKey(), "worker", "token", "1000"); err == nil {
		t.Fatal("Claim accepted aliased pending/delayed keys")
	}
	if score, err := queue.do(ctx, "ZSCORE", queue.leaseKey(), job.ID); err != nil || score == nil {
		t.Fatalf("aliased-key Claim disturbed the existing lease: score=%v err=%v", score, err)
	}
}

func TestDistributedRedisRejectsPendingAttemptsAtLimit(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, delayed := range []bool{false, true} {
		name := "pending"
		if delayed {
			name = "delayed"
		}
		t.Run(name, func(t *testing.T) {
			queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:attempt-limit:"+name+":"+uuid.NewString())
			if err != nil {
				t.Fatal(err)
			}
			job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
			if err != nil {
				t.Fatal(err)
			}
			job.Attempts = job.MaxAttempts
			encoded, err := json.Marshal(job)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := queue.do(ctx, "SET", queue.jobKey(job.ID), string(encoded)); err != nil {
				t.Fatal(err)
			}
			if delayed {
				if _, err := queue.do(ctx, "LREM", queue.pendingKey(), "0", job.ID); err != nil {
					t.Fatal(err)
				}
				if _, err := queue.do(ctx, "ZADD", queue.delayedKey(), "0", job.ID); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := queue.Claim("worker", time.Now()); err == nil {
				t.Fatal("Claim accepted pending/delayed job already at max attempts")
			}
			stored, err := queue.get(job.ID)
			if err != nil || stored.Status != QueuePending || stored.Attempts != stored.MaxAttempts {
				t.Fatalf("invalid attempts record was mutated: job=%+v err=%v", stored, err)
			}
			if delayed {
				if score, err := queue.do(ctx, "ZSCORE", queue.delayedKey(), job.ID); err != nil || score == nil {
					t.Fatalf("invalid delayed job was moved: score=%v err=%v", score, err)
				}
			} else if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(1) {
				t.Fatalf("invalid pending job was consumed: length=%v err=%v", length, err)
			}
		})
	}
}

func TestDistributedRedisReplayRequiresDeadLetterIndex(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:replay-origin:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 1)
	if err != nil {
		t.Fatal(err)
	}
	claim, ok, err := queue.Claim("worker", time.Now())
	if err != nil || !ok || claim.ID != job.ID {
		t.Fatalf("claim failed: %+v ok=%v err=%v", claim, ok, err)
	}
	if _, err := queue.Nack(claim, errors.New("retry limit")); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "LREM", queue.deadKey(), "0", job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.Replay(job.ID); err == nil {
		t.Fatal("Replay accepted a dead-letter job missing its source index")
	}
	stored, err := queue.get(job.ID)
	if err != nil || stored.Status != QueueDeadLetter {
		t.Fatalf("failed replay changed job status: job=%+v err=%v", stored, err)
	}
	if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(0) {
		t.Fatalf("failed replay inserted a pending ID: length=%v err=%v", length, err)
	}
}

func TestDistributedRedisEnqueueRejectsMissionIndexCollision(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:mission-index:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	missionA, missionB := "mis_"+uuid.NewString(), "mis_"+uuid.NewString()
	if _, err := queue.Enqueue(missionA, 3); err != nil {
		t.Fatal(err)
	}
	jobB, err := queue.Enqueue(missionB, 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "SET", queue.missionKey(missionA), jobB.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.Enqueue(missionA, 3); err == nil {
		t.Fatal("Enqueue accepted a mission index pointing to another mission's job")
	}
	index, err := queue.do(ctx, "GET", queue.missionKey(missionA))
	if err != nil || index != jobB.ID {
		t.Fatalf("mismatched mission index was overwritten: value=%v err=%v", index, err)
	}
}

func TestDistributedRedisEnqueueRejectsStaleMissionIndex(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	newQueue := func(t *testing.T) *RedisQueue {
		t.Helper()
		prefix := "ollama:stale-mission-index:" + uuid.NewString()
		queue, err := openRedisTestQueue(t, ctx, redisURL, prefix)
		if err != nil {
			t.Fatal(err)
		}
		return queue
	}

	t.Run("pending job missing queue membership", func(t *testing.T) {
		queue := newQueue(t)
		missionID := "mis_" + uuid.NewString()
		job, err := queue.Enqueue(missionID, 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "LREM", queue.pendingKey(), "0", job.ID); err != nil {
			t.Fatal(err)
		}
		before := captureRedisEnqueueState(t, ctx, queue, missionID, job.ID)
		_, err = queue.Enqueue(missionID, 3)
		if err == nil {
			t.Fatal("Enqueue accepted a pending mission job missing from queue indexes")
		}
		assertRedisEnqueueStateUnchanged(t, before, captureRedisEnqueueState(t, ctx, queue, missionID, job.ID))
		index, err := queue.do(ctx, "GET", queue.missionKey(missionID))
		if err != nil || index != job.ID {
			t.Fatalf("failed Enqueue changed mission index: value=%v err=%v", index, err)
		}
	})

	t.Run("running job missing lease membership", func(t *testing.T) {
		queue := newQueue(t)
		missionID := "mis_" + uuid.NewString()
		job, err := queue.Enqueue(missionID, 3)
		if err != nil {
			t.Fatal(err)
		}
		claim, ok, err := queue.Claim("worker", time.Now())
		if err != nil || !ok || claim.ID != job.ID {
			t.Fatalf("Claim failed: job=%+v ok=%v err=%v", claim, ok, err)
		}
		if _, err := queue.do(ctx, "ZREM", queue.leaseKey(), job.ID); err != nil {
			t.Fatal(err)
		}
		before := captureRedisEnqueueState(t, ctx, queue, missionID, job.ID)
		_, err = queue.Enqueue(missionID, 3)
		if err == nil {
			t.Fatal("Enqueue accepted a running mission job missing lease membership")
		}
		assertRedisEnqueueStateUnchanged(t, before, captureRedisEnqueueState(t, ctx, queue, missionID, job.ID))
		index, err := queue.do(ctx, "GET", queue.missionKey(missionID))
		if err != nil || index != job.ID {
			t.Fatalf("failed Enqueue changed mission index: value=%v err=%v", index, err)
		}
	})

	t.Run("pending job present only in delayed is idempotent", func(t *testing.T) {
		queue := newQueue(t)
		missionID := "mis_" + uuid.NewString()
		job, err := queue.Enqueue(missionID, 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "LREM", queue.pendingKey(), "0", job.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "ZADD", queue.delayedKey(), "0", job.ID); err != nil {
			t.Fatal(err)
		}
		before := captureRedisEnqueueState(t, ctx, queue, missionID, job.ID)
		got, err := queue.Enqueue(missionID, 3)
		if err != nil || got.ID != job.ID {
			t.Fatalf("Enqueue did not return valid delayed job idempotently: got=%+v err=%v", got, err)
		}
		assertRedisEnqueueStateUnchanged(t, before, captureRedisEnqueueState(t, ctx, queue, missionID, job.ID))
		if score, err := queue.do(ctx, "ZSCORE", queue.delayedKey(), job.ID); err != nil || score == nil {
			t.Fatalf("idempotent Enqueue changed delayed membership: score=%v err=%v", score, err)
		}
		if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(0) {
			t.Fatalf("idempotent Enqueue created pending membership: length=%v err=%v", length, err)
		}
	})

	t.Run("pending job in both pending and delayed is rejected", func(t *testing.T) {
		queue := newQueue(t)
		missionID := "mis_" + uuid.NewString()
		job, err := queue.Enqueue(missionID, 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "ZADD", queue.delayedKey(), "0", job.ID); err != nil {
			t.Fatal(err)
		}
		before := captureRedisEnqueueState(t, ctx, queue, missionID, job.ID)
		_, err = queue.Enqueue(missionID, 3)
		if err == nil {
			t.Fatal("Enqueue accepted a pending job in both pending and delayed")
		}
		assertRedisEnqueueStateUnchanged(t, before, captureRedisEnqueueState(t, ctx, queue, missionID, job.ID))
		if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(1) {
			t.Fatalf("failed Enqueue changed pending membership: length=%v err=%v", length, err)
		}
		if score, err := queue.do(ctx, "ZSCORE", queue.delayedKey(), job.ID); err != nil || score == nil {
			t.Fatalf("failed Enqueue changed delayed membership: score=%v err=%v", score, err)
		}
	})

	t.Run("pending job in dead-letter is rejected", func(t *testing.T) {
		queue := newQueue(t)
		missionID := "mis_" + uuid.NewString()
		job, err := queue.Enqueue(missionID, 3)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := queue.do(ctx, "LPUSH", queue.deadKey(), job.ID); err != nil {
			t.Fatal(err)
		}
		before := captureRedisEnqueueState(t, ctx, queue, missionID, job.ID)
		_, err = queue.Enqueue(missionID, 3)
		if err == nil {
			t.Fatal("Enqueue accepted a pending job also present in dead-letter")
		}
		assertRedisEnqueueStateUnchanged(t, before, captureRedisEnqueueState(t, ctx, queue, missionID, job.ID))
		if length, err := queue.do(ctx, "LLEN", queue.deadKey()); err != nil || length != int64(1) {
			t.Fatalf("failed Enqueue changed dead-letter membership: length=%v err=%v", length, err)
		}
		if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(1) {
			t.Fatalf("failed Enqueue changed pending membership: length=%v err=%v", length, err)
		}
	})

	t.Run("running job with mismatched lease score is rejected", func(t *testing.T) {
		queue := newQueue(t)
		missionID := "mis_" + uuid.NewString()
		job, err := queue.Enqueue(missionID, 3)
		if err != nil {
			t.Fatal(err)
		}
		claim, ok, err := queue.Claim("worker", time.Now())
		if err != nil || !ok || claim.ID != job.ID {
			t.Fatalf("Claim failed: job=%+v ok=%v err=%v", claim, ok, err)
		}
		if _, err := queue.do(ctx, "ZADD", queue.leaseKey(), strconv.FormatInt(claim.LeaseUntilUnixMilli+1, 10), job.ID); err != nil {
			t.Fatal(err)
		}
		before := captureRedisEnqueueState(t, ctx, queue, missionID, job.ID)
		_, err = queue.Enqueue(missionID, 3)
		if err == nil {
			t.Fatal("Enqueue accepted a running job with a mismatched lease score")
		}
		assertRedisEnqueueStateUnchanged(t, before, captureRedisEnqueueState(t, ctx, queue, missionID, job.ID))
		if score, err := queue.do(ctx, "ZSCORE", queue.leaseKey(), job.ID); err != nil || score == nil {
			t.Fatalf("failed Enqueue changed lease membership: score=%v err=%v", score, err)
		}
	})
}

func TestDistributedRedisClaimValidatesLeaseBeforeHousekeeping(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:claim-lease-preflight:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "LREM", queue.pendingKey(), "0", job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "ZADD", queue.delayedKey(), "0", job.ID); err != nil {
		t.Fatal(err)
	}

	queue.leaseDuration = 24*time.Hour + time.Millisecond
	before := captureRedisEnqueueState(t, ctx, queue, job.MissionID, job.ID)
	if _, _, err := queue.Claim("worker", time.Now()); err == nil {
		t.Fatal("Claim accepted a lease duration above 24 hours")
	}
	assertRedisEnqueueStateUnchanged(t, before, captureRedisEnqueueState(t, ctx, queue, job.MissionID, job.ID))
	if score, err := queue.do(ctx, "ZSCORE", queue.delayedKey(), job.ID); err != nil || score == nil {
		t.Fatalf("invalid lease duration allowed housekeeping to move delayed job: score=%v err=%v", score, err)
	}
	if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(0) {
		t.Fatalf("invalid lease duration mutated pending queue: length=%v err=%v", length, err)
	}
}

func TestDistributedRedisLegacyUnstampedJobIsInspectOnly(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:legacy-job:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	value, err := queue.do(ctx, "GET", queue.jobKey(job.ID))
	if err != nil {
		t.Fatal(err)
	}
	var legacy map[string]any
	if err := json.Unmarshal([]byte(value.(string)), &legacy); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"created_at_ms", "available_at_ms", "updated_at_ms", "available_seq"} {
		delete(legacy, field)
	}
	legacy["created_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	legacy["available_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	legacy["updated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	encoded, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "SET", queue.jobKey(job.ID), string(encoded)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := queue.Claim("worker", time.Now()); err == nil {
		t.Fatal("Claim silently accepted an unstamped legacy job")
	}
	stored, err := queue.do(ctx, "GET", queue.jobKey(job.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored != string(encoded) {
		t.Fatalf("legacy job was mutated despite inspection-only policy: before=%s after=%v", encoded, stored)
	}
	if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(1) {
		t.Fatalf("legacy pending ID was consumed: length=%v err=%v", length, err)
	}
}

func TestDistributedRedisMoveDueRejectsDuplicatePendingIndex(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:duplicate-index:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "ZADD", queue.delayedKey(), "0", job.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := queue.Claim("worker", time.Now()); err == nil {
		t.Fatal("move-due accepted a job indexed in both delayed and pending")
	}
	if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(1) {
		t.Fatalf("duplicate pending index was added: length=%v err=%v", length, err)
	}
	if score, err := queue.do(ctx, "ZSCORE", queue.delayedKey(), job.ID); err != nil || score == nil {
		t.Fatalf("conflicting delayed index was removed: score=%v err=%v", score, err)
	}
}

func TestDistributedRedisEnqueueRejectsMismatchedPayloadMission(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:payload-mission:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	missionID := "mis_" + uuid.NewString()
	jobID := "job_" + uuid.NewString()
	now := time.Now().UTC()
	payload, err := json.Marshal(QueueJob{ID: jobID, MissionID: "mis_" + uuid.NewString(), Status: QueuePending, MaxAttempts: 3, AvailableAt: now, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	before := captureRedisEnqueueState(t, ctx, queue, missionID, jobID)
	if _, err := queue.do(ctx, "EVAL", redisEnqueueScript, "7", queue.missionKey(missionID), queue.key("job:"), queue.pendingKey(), queue.delayedKey(), queue.leaseKey(), queue.deadKey(), queue.key("sequence"), string(payload), missionID, ""); err == nil || !strings.Contains(err.Error(), "invalid enqueue payload identity or initial state") {
		t.Fatalf("Enqueue script did not reject payload for the expected mission identity reason: %v", err)
	}
	assertRedisEnqueueStateUnchanged(t, before, captureRedisEnqueueState(t, ctx, queue, missionID, jobID))
	if value, err := queue.do(ctx, "GET", queue.missionKey(missionID)); err != nil || value != nil {
		t.Fatalf("mismatched payload created mission index: value=%v err=%v", value, err)
	}
	if length, err := queue.do(ctx, "LLEN", queue.pendingKey()); err != nil || length != int64(0) {
		t.Fatalf("mismatched payload created pending entry: length=%v err=%v", length, err)
	}
}

func TestDistributedRedisLegacyRunningLeaseIsInspectOnly(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:legacy-running:"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	job, err := queue.Enqueue("mis_"+uuid.NewString(), 3)
	if err != nil {
		t.Fatal(err)
	}
	claim, ok, err := queue.Claim("worker-old", time.Now())
	if err != nil || !ok || claim.ID != job.ID {
		t.Fatalf("claim failed: %+v ok=%v err=%v", claim, ok, err)
	}
	expireRedisLeaseForTest(t, ctx, queue, job.ID)
	value, err := queue.do(ctx, "GET", queue.jobKey(job.ID))
	if err != nil {
		t.Fatal(err)
	}
	var legacy map[string]any
	if err := json.Unmarshal([]byte(value.(string)), &legacy); err != nil {
		t.Fatal(err)
	}
	delete(legacy, "lease_until_ms")
	encoded, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "SET", queue.jobKey(job.ID), string(encoded)); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.do(ctx, "ZADD", queue.leaseKey(), "0", job.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := queue.Claim("worker-new", time.Now()); err == nil {
		t.Fatal("reclaim silently resumed a legacy running job without authoritative lease_until_ms")
	}
	stored, err := queue.do(ctx, "GET", queue.jobKey(job.ID))
	if err != nil || stored != string(encoded) {
		t.Fatalf("legacy running job was mutated: got=%v want=%s err=%v", stored, encoded, err)
	}
	if score, err := queue.do(ctx, "ZSCORE", queue.leaseKey(), job.ID); err != nil || score == nil {
		t.Fatalf("legacy lease index was removed: score=%v err=%v", score, err)
	}
}

func TestDistributedRedisQueueOrganizationBindingAndReplay(t *testing.T) {
	redisURL := os.Getenv("OLLAMA_AGENT_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("OLLAMA_AGENT_TEST_REDIS_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	queue, err := openRedisTestQueue(t, ctx, redisURL, "ollama:tenant-owner")
	if err != nil {
		t.Fatal(err)
	}
	legacyMission := "mis_legacy_" + uuid.NewString()
	legacy, err := queue.Enqueue(legacyMission, 3)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := queue.EnqueueForOrganization("org-a", legacyMission, 3)
	if err != nil || bound.ID != legacy.ID || bound.OrganizationID != "org-a" {
		t.Fatalf("legacy binding=%+v err=%v", bound, err)
	}
	duplicate, err := queue.EnqueueForOrganization("org-a", legacyMission, 3)
	if err != nil || duplicate.ID != bound.ID {
		t.Fatalf("same-owner duplicate=%+v err=%v", duplicate, err)
	}
	if _, err := queue.EnqueueForOrganization("org-b", legacyMission, 3); err == nil {
		t.Fatal("Redis queue accepted cross-organization mission dedupe")
	}
	legacyClaim, ok, err := queue.Claim("tenant-owner-worker", time.Now().UTC())
	if err != nil || !ok || legacyClaim.ID != bound.ID || legacyClaim.OrganizationID != "org-a" {
		t.Fatalf("legacy claim=%+v ok=%v err=%v", legacyClaim, ok, err)
	}
	if err := queue.Ack(legacyClaim); err != nil {
		t.Fatal(err)
	}

	replayMission := "mis_replay_" + uuid.NewString()
	job, err := queue.EnqueueForOrganization("org-a", replayMission, 1)
	if err != nil {
		t.Fatal(err)
	}
	claim, ok, err := queue.Claim("tenant-owner-worker", time.Now().UTC())
	if err != nil || !ok || claim.ID != job.ID || claim.OrganizationID != "org-a" {
		t.Fatalf("claim=%+v ok=%v err=%v", claim, ok, err)
	}
	dead, err := queue.Nack(claim, errors.New("synthetic terminal failure"))
	if err != nil || dead.Status != QueueDeadLetter {
		t.Fatalf("dead-letter=%+v err=%v", dead, err)
	}
	if _, err := queue.ReplayForOrganization("org-b", job.ID); err == nil {
		t.Fatal("Redis replay accepted a foreign organization")
	}
	replayed, err := queue.ReplayForOrganization("org-a", job.ID)
	if err != nil || replayed.Status != QueuePending || replayed.OrganizationID != "org-a" {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}
}
