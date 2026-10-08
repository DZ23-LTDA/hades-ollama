package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type failingMissionPutStore struct{ Store }

func snapshotManifestDigest(t *testing.T, snapshotRoot string) string {
	t.Helper()
	manifest, err := os.ReadFile(filepath.Join(snapshotRoot, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(manifest)
	return hex.EncodeToString(digest[:])
}

func (s failingMissionPutStore) PutMission(Mission) error {
	return errors.New("mission persistence unavailable")
}

func (s failingMissionPutStore) CreateMission(Mission) error {
	return errors.New("mission persistence unavailable")
}

func TestRuntimePersistsAndRunsReadMission(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(filepath.Join(root, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Planner: RulePlanner{}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "inspecionar o workspace"})
	if err != nil {
		t.Fatal(err)
	}
	if mission.State != MissionReady {
		t.Fatalf("state = %s, want READY", mission.State)
	}
	if err := runtime.Run(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
	completed, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != MissionCompleted || completed.Plan[0].State != StepSucceeded {
		t.Fatalf("mission = %+v", completed)
	}
	reloaded, err := NewJSONStore(filepath.Join(root, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := reloaded.GetMission(mission.ID)
	if err != nil || persisted.State != MissionCompleted {
		t.Fatalf("persisted = %+v err=%v", persisted, err)
	}
	events, err := runtime.Events(mission.ID)
	if err != nil || len(events) < 3 {
		t.Fatalf("events=%d err=%v", len(events), err)
	}
}

func TestRuntimeCreatesTenantScopedIsolatedWorkspaceSnapshot(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	repo := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, repo, "app.txt", "source version")
	runSnapshotFixtureGit(t, repo, "add", "app.txt")
	runSnapshotFixtureGit(t, repo, "commit", "-m", "initial")
	contextStore, err := NewContextStore(filepath.Join(t.TempDir(), "context"))
	if err != nil {
		t.Fatal(err)
	}
	if err := contextStore.SetWorkspaceRoot(repo); err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("isolated", repo, "org_snapshot_test")
	if err != nil {
		t.Fatal(err)
	}
	dataRoot := t.TempDir()
	missionStore, err := NewJSONStore(filepath.Join(dataRoot, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: missionStore, Planner: RulePlanner{}, WorkspaceRoot: repo, DataRoot: dataRoot, Context: contextStore})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:        "inspect isolated project",
		ProjectID:        project.ID,
		OrganizationID:   "org_snapshot_test",
		IsolateWorkspace: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !mission.WorkspaceIsolated || mission.WorkspaceSnapshotID == "" || len(mission.WorkspaceSnapshotSHA256) != 64 {
		t.Fatalf("snapshot metadata missing from mission: %+v", mission)
	}
	if samePath(mission.Workspace, repo) || !isWithin(filepath.Join(dataRoot, ".agent-workspace-snapshots"), mission.Workspace) {
		t.Fatalf("mission workspace is not isolated: source=%q mission=%q", repo, mission.Workspace)
	}
	snapshotBytes, err := os.ReadFile(filepath.Join(mission.Workspace, "app.txt"))
	if err != nil || string(snapshotBytes) != "source version" {
		t.Fatalf("snapshot file=%q err=%v", snapshotBytes, err)
	}
	if err := os.WriteFile(filepath.Join(mission.Workspace, "app.txt"), []byte("agent edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	if root, err := openMissionWorkspaceSnapshot(dataRoot, mission); err == nil {
		_ = root.Close()
		t.Fatal("tampered snapshot opened without a persisted artifact manifest")
	}
	sourceBytes, err := os.ReadFile(filepath.Join(repo, "app.txt"))
	if err != nil || string(sourceBytes) != "source version" {
		t.Fatalf("source changed through isolated workspace: %q err=%v", sourceBytes, err)
	}
	reloadedStore, err := NewJSONStore(filepath.Join(dataRoot, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	recoveredRuntime, err := NewRuntime(RuntimeConfig{Store: reloadedStore, Planner: RulePlanner{}, WorkspaceRoot: repo, DataRoot: dataRoot, Context: contextStore})
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := recoveredRuntime.GetMission(mission.ID)
	if err != nil || persisted.WorkspaceSnapshotID != mission.WorkspaceSnapshotID || persisted.WorkspaceSnapshotSHA256 != mission.WorkspaceSnapshotSHA256 {
		t.Fatalf("snapshot metadata did not persist: %+v err=%v", persisted, err)
	}
	if err := validateMissionWorkspaceSnapshot(dataRoot, persisted); err != nil {
		t.Fatalf("persisted snapshot failed integrity validation: %v", err)
	}
	manifestPath := filepath.Join(dataRoot, ".agent-workspace-snapshots", "snapshots", "*")
	manifestPaths, err := filepath.Glob(filepath.Join(manifestPath, "mis_*", mission.WorkspaceSnapshotID, "manifest.json"))
	if err != nil || len(manifestPaths) != 1 {
		t.Fatalf("snapshot manifest paths = %v, err=%v", manifestPaths, err)
	}
	if err := os.WriteFile(manifestPaths[0], []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateMissionWorkspaceSnapshot(dataRoot, persisted); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("tampered manifest validation error = %v, want digest mismatch", err)
	}
}

func TestNonIsolatedMissionWorkspaceIdentityRejectsPathReplacement(t *testing.T) {
	workspace := t.TempDir()
	identity, err := workspaceDirectoryIdentity(workspace)
	if err != nil {
		t.Fatal(err)
	}
	mission := Mission{ID: "mis_workspace_identity_test", Workspace: workspace, WorkspaceIdentity: identity}
	root, err := openMissionWorkspaceSnapshot(t.TempDir(), mission)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	moved := workspace + ".authorized"
	if err := os.Rename(workspace, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := openMissionWorkspaceSnapshot(t.TempDir(), mission); err == nil || !strings.Contains(err.Error(), "changed after authorization") {
		t.Fatalf("replaced workspace error = %v, want identity mismatch", err)
	}
	mission.WorkspaceIdentity = ""
	if _, err := openMissionWorkspaceSnapshot(t.TempDir(), mission); err == nil || !strings.Contains(err.Error(), "identity is missing") {
		t.Fatalf("legacy workspace error = %v, want fail-closed identity requirement", err)
	}
}

func TestIsolatedWorkspaceToolsStayAnchoredAfterPathReplacement(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	repo := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, repo, "app.txt", "approved source")
	runSnapshotFixtureGit(t, repo, "add", "app.txt")
	runSnapshotFixtureGit(t, repo, "commit", "-m", "initial")
	contextStore, err := NewContextStore(filepath.Join(t.TempDir(), "context"))
	if err != nil {
		t.Fatal(err)
	}
	if err := contextStore.SetWorkspaceRoot(repo); err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("anchored", repo, "org_anchor_test")
	if err != nil {
		t.Fatal(err)
	}
	dataRoot := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: repo, DataRoot: dataRoot, Context: contextStore})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "read isolated snapshot", ProjectID: project.ID, OrganizationID: "org_anchor_test", IsolateWorkspace: true})
	if err != nil {
		t.Fatal(err)
	}
	root, err := openMissionWorkspaceSnapshot(dataRoot, mission)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	originalPath := mission.Workspace + ".original"
	if err := os.Rename(mission.Workspace, originalPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(mission.Workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mission.Workspace, "app.txt"), []byte("other tenant data"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := (workspaceReadTool{}).Execute(context.Background(), ToolContext{Workspace: mission.Workspace, WorkspaceRoot: root}, map[string]any{"path": "app.txt"})
	if err != nil {
		t.Fatal(err)
	}
	value, ok := result.Value.(map[string]any)
	if !ok || value["content"] != "approved source" {
		t.Fatalf("anchored read=%+v, want original snapshot content", result.Value)
	}
	if _, err := (workspaceReadTool{}).Execute(context.Background(), ToolContext{Workspace: mission.Workspace, WorkspaceRoot: root}, map[string]any{"path": "../../etc/passwd"}); err == nil {
		t.Fatal("anchored read accepted traversal")
	}
	writeResult, err := writeWorkspaceFile(ToolContext{MissionID: mission.ID, StepID: "step_anchor_write", Workspace: mission.Workspace, WorkspaceRoot: root}, map[string]any{
		"path": "agent.txt", "content": "anchored output", "expected_sha256": "", "expected_absent": true,
	})
	if err != nil {
		t.Fatalf("anchored write failed: %v", err)
	}
	if len(writeResult.Artifacts) != 1 {
		t.Fatalf("anchored write artifacts=%+v, want one verified artifact", writeResult.Artifacts)
	}
	originalOutput, err := os.ReadFile(filepath.Join(originalPath, "agent.txt"))
	if err != nil || string(originalOutput) != "anchored output" {
		t.Fatalf("anchored write target=%q err=%v", originalOutput, err)
	}
	if _, err := os.Stat(filepath.Join(mission.Workspace, "agent.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("anchored write modified replacement workspace: err=%v", err)
	}
}

func TestRuntimeSnapshotsGitRootForNestedProject(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	repo := initWorkspaceSnapshotGit(t)
	projectRoot := filepath.Join(repo, "apps", "service")
	if err := os.MkdirAll(projectRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	writeSnapshotFixture(t, projectRoot, "app.txt", "nested project source")
	runSnapshotFixtureGit(t, repo, "add", "apps/service/app.txt")
	runSnapshotFixtureGit(t, repo, "commit", "-m", "nested project")
	contextStore, err := NewContextStore(filepath.Join(t.TempDir(), "context"))
	if err != nil {
		t.Fatal(err)
	}
	if err := contextStore.SetWorkspaceRoot(repo); err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("nested service", projectRoot, "org_nested_snapshot")
	if err != nil {
		t.Fatal(err)
	}
	dataRoot := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: repo, DataRoot: dataRoot, Context: contextStore})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "inspect nested project", ProjectID: project.ID, OrganizationID: "org_nested_snapshot", IsolateWorkspace: true})
	if err != nil {
		t.Fatal(err)
	}
	if !isWithin(filepath.Join(dataRoot, ".agent-workspace-snapshots"), mission.Workspace) || !strings.HasSuffix(filepath.Clean(mission.Workspace), filepath.Join("tree", "apps", "service")) {
		t.Fatalf("nested mission workspace = %q", mission.Workspace)
	}
	content, err := os.ReadFile(filepath.Join(mission.Workspace, "app.txt"))
	if err != nil || string(content) != "nested project source" {
		t.Fatalf("nested snapshot content = %q, err=%v", content, err)
	}
}

func TestRuntimeRequiresProjectAndOrganizationForIsolation(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: root, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "isolated without project", IsolateWorkspace: true}); err == nil || !strings.Contains(err.Error(), "project and organization") {
		t.Fatalf("isolation without project/tenant error = %v", err)
	}
	projectRoot := t.TempDir()
	contextStore, err := NewContextStore(filepath.Join(t.TempDir(), "context"))
	if err != nil {
		t.Fatal(err)
	}
	if err := contextStore.SetWorkspaceRoot(projectRoot); err != nil {
		t.Fatal(err)
	}
	legacyProject, err := contextStore.CreateProject("unscoped legacy", projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	legacyRuntime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: projectRoot, Context: contextStore, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacyRuntime.CreateMission(context.Background(), CreateMissionRequest{Objective: "isolate legacy", ProjectID: legacyProject.ID, OrganizationID: "org_test", IsolateWorkspace: true}); err == nil || !strings.Contains(err.Error(), "owned by the active organization") {
		t.Fatalf("isolation of unscoped project error = %v", err)
	}
}

func TestRuntimePreservesWorkspaceSnapshotWhenPersistenceOutcomeIsUnknown(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	repo := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, repo, "app.txt", "source version")
	runSnapshotFixtureGit(t, repo, "add", "app.txt")
	runSnapshotFixtureGit(t, repo, "commit", "-m", "initial")
	contextStore, err := NewContextStore(filepath.Join(t.TempDir(), "context"))
	if err != nil {
		t.Fatal(err)
	}
	if err := contextStore.SetWorkspaceRoot(repo); err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("isolated", repo, "org_snapshot_test")
	if err != nil {
		t.Fatal(err)
	}
	dataRoot := t.TempDir()
	store := failingMissionPutStore{Store: NewMemoryStore()}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Planner: RulePlanner{}, WorkspaceRoot: repo, DataRoot: dataRoot, Context: contextStore})
	if err != nil {
		t.Fatal(err)
	}
	_, err = runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:        "snapshot cleanup",
		ProjectID:        project.ID,
		OrganizationID:   "org_snapshot_test",
		IsolateWorkspace: true,
	})
	if err == nil || !strings.Contains(err.Error(), "mission persistence unavailable") {
		t.Fatalf("CreateMission error = %v", err)
	}
	manifests, err := filepath.Glob(filepath.Join(dataRoot, ".agent-workspace-snapshots", "snapshots", "*", "mis_*", "snp_*", "manifest.json"))
	if err != nil || len(manifests) != 1 {
		t.Fatalf("snapshot manifests = %v, err=%v; ambiguous persistence errors must preserve the snapshot for reconciliation", manifests, err)
	}
}

func TestRuntimePushOutboxRechecksDLPBeforeNetworkDelivery(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	push, err := NewPushService("", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := push.Register("push-token", "ios", "user", "org_push_test"); err != nil {
		t.Fatal(err)
	}
	outbox, err := NewPushOutbox(filepath.Join(t.TempDir(), "outbox"))
	if err != nil {
		t.Fatal(err)
	}
	item, err := outbox.Enqueue("org_push_test", "Notice", "Body", map[string]any{"message": "safe"})
	if err != nil {
		t.Fatal(err)
	}
	outbox.mu.Lock()
	corrupted := outbox.items[item.ID]
	outbox.mu.Unlock()
	corrupted.Data = map[string]any{"details": map[string]any{"api_key": "abcdef0123456789abcdef"}}
	encoded, err := json.Marshal(map[string]PushOutboxItem{item.ID: corrupted})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outbox.path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(outbox.path)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir(), Push: push, PushOutbox: outbox})
	if err != nil {
		t.Fatal(err)
	}
	runtime.flushPushOutbox(context.Background())
	if got := requests.Load(); got != 0 {
		t.Fatalf("push provider received %d request(s) with DLP-blocked outbox data", got)
	}
	if runtime.Metrics().PushOutboxFailures != 1 {
		t.Fatalf("corrupted legacy record should fail closed during claim: %+v", runtime.Metrics())
	}
	after, err := os.ReadFile(outbox.path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("corrupted record changed during failed delivery claim: err=%v", err)
	}
}

func TestRuntimeRejectsUnconfiguredMissionProvider(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(filepath.Join(root, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Planner: RulePlanner{}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "testar provider", Provider: "claude"}); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("provider error = %v", err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "usar provider local"})
	if err != nil {
		t.Fatal(err)
	}
	if mission.Provider != "ollama-local" {
		t.Fatalf("provider = %q", mission.Provider)
	}
}

func TestRuntimeQueueJobsOrganizationScope(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "fila tenant", OrganizationID: "org_a"})
	if err != nil {
		t.Fatal(err)
	}
	job, err := runtime.EnqueueMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.OrganizationID != "org_a" {
		t.Fatalf("queue job organization=%q, want org_a", job.OrganizationID)
	}
	jobs, err := runtime.QueueJobsForOrganization("org_a", "")
	if err != nil || len(jobs) != 1 || jobs[0].ID != job.ID {
		t.Fatalf("org_a jobs=%+v err=%v", jobs, err)
	}
	jobs, err = runtime.QueueJobsForOrganization("org_b", "")
	if err != nil || len(jobs) != 0 {
		t.Fatalf("org_b jobs=%+v err=%v", jobs, err)
	}
	if _, err := runtime.ReplayJobForOrganization(job.ID, "org_b"); !errors.Is(err, ErrQueueJobForbidden) {
		t.Fatalf("cross-tenant replay error=%v", err)
	}
	viewB := runtime.WithOrganization("org_b")
	if jobs := viewB.QueueJobs(""); len(jobs) != 0 {
		t.Fatalf("org_b scoped queue leaked tenant A jobs: %+v", jobs)
	}
	if _, err := viewB.ReplayJobForOrganization(job.ID, "org_b"); !errors.Is(err, ErrQueueJobForbidden) {
		t.Fatalf("scoped cross-tenant replay error=%v, want forbidden", err)
	}
	jobs, err = runtime.QueueJobsForOrganization("org_a", "")
	if err != nil || len(jobs) != 1 || jobs[0].Status != QueuePending {
		t.Fatalf("job mutated after rejected replay: %+v err=%v", jobs, err)
	}
}

func TestRuntimeQueueListingsHideJobsWithMismatchedOwnerMetadata(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "queue metadata leak", OrganizationID: "org-a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.queue.EnqueueForOrganization("org-b", mission.ID, 1); err != nil {
		t.Fatal(err)
	}
	if jobs, err := runtime.QueueJobsForOrganization("org-a", ""); err != nil || len(jobs) != 0 {
		t.Fatalf("org-a listing exposed owner-mismatched job: jobs=%+v err=%v", jobs, err)
	}
	if jobs := runtime.WithOrganization("org-a").QueueJobs(""); len(jobs) != 0 {
		t.Fatalf("scoped QueueJobs exposed owner-mismatched job: %+v", jobs)
	}
	if _, err := runtime.QueueJobsForOrganization("", ""); !errors.Is(err, ErrQueueJobForbidden) {
		t.Fatalf("unscoped listing of tenant mission error=%v, want forbidden", err)
	}
}

func TestNewRuntimeFailsClosedForPostgresStoresAndDecorators(t *testing.T) {
	for name, store := range map[string]Store{
		"direct":              &PostgresStore{},
		"organization-scoped": organizationScopedStore{store: &PostgresStore{}, organizationID: "org-a"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewRuntime(RuntimeConfig{Store: store, WorkspaceRoot: t.TempDir()}); !errors.Is(err, ErrPostgresTenantIsolationUnavailable) {
				t.Fatalf("Postgres runtime error=%v, want fail-closed tenant-isolation error", err)
			}
		})
	}
}

func TestRuntimeQueueWorkerRejectsOwnerlessAndForeignJobs(t *testing.T) {
	store := NewMemoryStore()
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Planner: RulePlanner{}, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "queue tenant boundary", OrganizationID: "org-a"})
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range []QueueJob{{MissionID: mission.ID}, {MissionID: mission.ID, OrganizationID: "org-b"}} {
		if err := runtime.runQueueJob(context.Background(), job); !errors.Is(err, ErrQueueJobForbidden) {
			t.Fatalf("base worker accepted ownerless/foreign job %+v: %v", job, err)
		}
	}
	if err := runtime.WithOrganization("org-a").runQueueJob(context.Background(), QueueJob{MissionID: mission.ID, OrganizationID: "org-b"}); !errors.Is(err, ErrQueueJobForbidden) {
		t.Fatalf("scoped worker accepted mismatched organization job: %v", err)
	}
	after, err := store.GetMission(mission.ID)
	if err != nil || after.Version != mission.Version || after.State != mission.State {
		t.Fatalf("rejected queue job mutated mission: after=%+v err=%v", after, err)
	}
}

func TestRuntimeWithOrganizationScopesMemoryStore(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	missionA, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "tenant A", OrganizationID: "org_a"})
	if err != nil {
		t.Fatal(err)
	}
	missionB, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "tenant B", OrganizationID: "org_b"})
	if err != nil {
		t.Fatal(err)
	}
	viewA := runtime.WithOrganization("org_a")
	if got, err := viewA.GetMission(missionA.ID); err != nil || got.OrganizationID != "org_a" {
		t.Fatalf("org_a mission=%+v err=%v", got, err)
	}
	if _, err := viewA.GetMission(missionB.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cross-tenant get error=%v, want not found", err)
	}
	if _, err := viewA.EnqueueMission(missionB.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cross-tenant enqueue error=%v, want not found", err)
	}
	if _, err := viewA.Events(missionB.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cross-tenant events error=%v, want not found", err)
	}
	missions, err := viewA.ListMissions()
	if err != nil || len(missions) != 1 || missions[0].ID != missionA.ID {
		t.Fatalf("org_a missions=%+v err=%v", missions, err)
	}
	if _, err := viewA.CreateMission(context.Background(), CreateMissionRequest{Objective: "spoof", OrganizationID: "org_b"}); err == nil {
		t.Fatal("organization-scoped runtime accepted a forged tenant")
	}
	if _, err := viewA.CreateMission(context.Background(), CreateMissionRequest{Objective: "shared workspace IDOR"}); err == nil || !strings.Contains(err.Error(), "require a project") {
		t.Fatalf("organization-scoped mission without project error=%v, want project ownership rejection", err)
	}
}

func TestRuntimeScopedRecoveryDoesNotSweepOtherTenantSnapshots(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	source := initWorkspaceSnapshotGit(t)
	dataRoot := t.TempDir()
	create := func(missionID, organizationID string) WorkspaceSnapshotManifest {
		t.Helper()
		manifest, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
			SourceRoot: source, DataRoot: filepath.Join(dataRoot, ".agent-workspace-snapshots"),
			MissionID: missionID, OrganizationID: organizationID, ProjectID: "proj_" + missionID,
		})
		if err != nil {
			t.Fatal(err)
		}
		return manifest
	}
	kept := create("mis_scope_keep", "org_a")
	otherTenantOrphan := create("mis_scope_other", "org_b")
	old := time.Now().Add(-workspaceSnapshotOrphanGrace - time.Minute)
	if err := os.Chtimes(otherTenantOrphan.SnapshotRoot, old, old); err != nil {
		t.Fatal(err)
	}
	store := NewMemoryStore()
	mission := Mission{
		ID: kept.MissionID, Version: 1, Objective: "kept", Workspace: kept.TreeRoot,
		WorkspaceIsolated: true, WorkspaceSnapshotID: kept.SnapshotID,
		WorkspaceSnapshotSHA256: snapshotManifestDigest(t, kept.SnapshotRoot),
		ProjectID:               kept.ProjectID, OrganizationID: kept.OrganizationID,
		State: MissionCompleted, Plan: []Step{}, Approvals: []Approval{}, Artifacts: []ArtifactManifest{},
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := store.PutMission(mission); err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Planner: RulePlanner{}, WorkspaceRoot: source, DataRoot: dataRoot})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.WithOrganization("org_a").resumePending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(otherTenantOrphan.SnapshotRoot); err != nil {
		t.Fatalf("scoped org_a recovery removed org_b snapshot: %v", err)
	}
}

func TestRuntimeRequiresApprovalBeforeWritingAndBuildsArtifact(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: fixedPlanner{steps: []Step{{ID: "step_1", Kind: "workspace.write", Title: "write", Risk: RiskWrite, RequiresApproval: true, State: StepPending, Input: map[string]any{"path": "result.txt", "content": "hello"}}}}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "escrever resultado", Capabilities: []string{"workspace:read", "workspace:write"}})
	if err != nil {
		t.Fatal(err)
	}
	if mission.State != MissionAwaitingApproval || len(mission.Approvals) != 1 {
		t.Fatalf("mission before approval = %+v", mission)
	}
	if err := runtime.Run(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "result.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("write happened before approval: %v", err)
	}
	mission, err = runtime.DecideApproval(mission.ID, mission.Approvals[0].ID, true, "approved for test")
	if err != nil || mission.State != MissionReady {
		t.Fatalf("approval result = %+v err=%v", mission, err)
	}
	if err := runtime.Run(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
	completed, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != MissionCompleted || len(completed.Artifacts) != 1 || completed.Artifacts[0].SHA256 == "" {
		t.Fatalf("completed = %+v", completed)
	}
	data, err := os.ReadFile(filepath.Join(root, "result.txt"))
	if err != nil || string(data) != "hello" {
		t.Fatalf("artifact content=%q err=%v", data, err)
	}
}

func TestWorkspaceToolsRejectTraversal(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "ler arquivo"})
	if err != nil {
		t.Fatal(err)
	}
	tool, ok := runtime.tools.Get("workspace.read")
	if !ok {
		t.Fatal("workspace.read not registered")
	}
	_, err = tool.Execute(context.Background(), ToolContext{MissionID: mission.ID, StepID: "step_1", Workspace: mission.Workspace}, map[string]any{"path": "../secret"})
	if err == nil || !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("error = %v, want traversal rejection", err)
	}
}

func TestWorkspaceToolsRejectSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable on this platform: %v", err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "ler arquivo"})
	if err != nil {
		t.Fatal(err)
	}
	tool, ok := runtime.tools.Get("workspace.read")
	if !ok {
		t.Fatal("workspace.read not registered")
	}
	if _, err := tool.Execute(context.Background(), ToolContext{MissionID: mission.ID, StepID: "step_1", Workspace: mission.Workspace}, map[string]any{"path": "linked/secret.txt"}); err == nil {
		t.Fatal("expected symlink escape rejection")
	}
}

type fixedPlanner struct {
	steps []Step
}

func (p fixedPlanner) Plan(_ context.Context, _ Mission) ([]Step, error) {
	return append([]Step(nil), p.steps...), nil
}

type cancellationProbeTool struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (t *cancellationProbeTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "workspace.read", Version: "test", Description: "cancellation probe", Risk: RiskRead, Scopes: []string{"workspace:read"}}
}

func (t *cancellationProbeTool) Execute(ctx context.Context, _ ToolContext, _ map[string]any) (ToolResult, error) {
	call := t.calls.Add(1)
	if call == 1 {
		close(t.started)
		select {
		case <-ctx.Done():
			return ToolResult{}, ctx.Err()
		case <-t.release:
		}
	}
	return ToolResult{Value: map[string]any{"calls": call}}, nil
}

func TestRuntimeCancelInterruptsActiveToolAndPreventsNextStep(t *testing.T) {
	probe := &cancellationProbeTool{started: make(chan struct{}), release: make(chan struct{})}
	registry := NewRegistry()
	registry.Register(probe)
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Tools:         registry,
		WorkspaceRoot: t.TempDir(),
		Planner: fixedPlanner{steps: []Step{
			{ID: "step_1", Kind: "workspace.read", Title: "blocking", Risk: RiskRead, State: StepPending},
			{ID: "step_2", Kind: "workspace.read", Title: "must not run", Risk: RiskRead, State: StepPending},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "cancel active tool"})
	if err != nil {
		t.Fatal(err)
	}
	runDone := make(chan error, 1)
	go func() { runDone <- runtime.Run(context.Background(), mission.ID) }()
	select {
	case <-probe.started:
	case <-time.After(2 * time.Second):
		t.Fatal("tool did not start")
	}
	if _, err := runtime.Cancel(mission.ID); err != nil {
		t.Fatal(err)
	}
	close(probe.release)
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("cancelled run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled tool did not stop")
	}
	cancelled, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.State != MissionCancelled {
		t.Fatalf("state = %s, want CANCELLED", cancelled.State)
	}
	if calls := probe.calls.Load(); calls != 1 {
		t.Fatalf("tool calls = %d, want exactly one", calls)
	}
}

type dlpResultTool struct{}

func (dlpResultTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "workspace.read", Version: "test", Description: "test tool", Risk: RiskRead, Scopes: []string{"workspace:read"}}
}

func (dlpResultTool) Execute(context.Context, ToolContext, map[string]any) (ToolResult, error) {
	return ToolResult{Value: map[string]any{
		"safe":        "visible",
		"nested":      map[string]any{"token": "xai-abcdefghijklmnopqrstuvwxyz123456"},
		"nested_json": `provider error: {"api_key":"plain-secret-value"}`,
		"credential":  "api_key=super-secret-token-value",
	}}, nil
}

func TestRuntimeRedactsStepResultsEventsTracesAndPersistence(t *testing.T) {
	root := t.TempDir()
	store, err := NewJSONStore(filepath.Join(root, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	registry.Register(dlpResultTool{})
	traces, err := NewTraceStore("")
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, Tools: registry, Traces: traces, Planner: fixedPlanner{steps: []Step{{ID: "step_secret", Kind: "workspace.read", Title: "secret result", Risk: RiskRead, State: StepPending}}}, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "redaction test", OrganizationID: "org_a"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Run(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
	completed, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	encodedResult, _ := json.Marshal(completed.Plan[0].Result)
	if strings.Contains(string(encodedResult), "xai-") || strings.Contains(string(encodedResult), "super-secret-token-value") || strings.Contains(string(encodedResult), "plain-secret-value") || strings.Contains(string(encodedResult), `"api_key"`) {
		t.Fatalf("step result leaked credential: %s", encodedResult)
	}
	if err := runtime.event(completed, "test.secret", completed.Plan[0].ID, map[string]any{"token": "Bearer abcdefghijklmnop1234"}); err != nil {
		t.Fatal(err)
	}
	events, err := runtime.Events(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	encodedEvents, _ := json.Marshal(events)
	if strings.Contains(string(encodedEvents), "abcdefghijklmnop1234") {
		t.Fatalf("event leaked bearer token: %s", encodedEvents)
	}
	span := traces.StartForOrganization("org_a", "trace_secret", "", "test", map[string]any{"api_key": "super-secret-token-value"})
	span.End("error", errors.New("provider failed with xai-abcdefghijklmnopqrstuvwxyz123456"))
	encodedTraces, _ := json.Marshal(traces.ListForOrganization("org_a", "trace_secret", 10))
	if strings.Contains(string(encodedTraces), "super-secret-token-value") || strings.Contains(string(encodedTraces), "xai-") {
		t.Fatalf("trace leaked credential: %s", encodedTraces)
	}
	reloaded, err := NewJSONStore(filepath.Join(root, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := reloaded.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	persistedResult, _ := json.Marshal(persisted.Plan[0].Result)
	if strings.Contains(string(persistedResult), "super-secret-token-value") || strings.Contains(string(persistedResult), "xai-") {
		t.Fatalf("persisted result leaked credential: %s", persistedResult)
	}
}

func TestRuntimeEventReportsPushOutboxFailureWithoutFailingMissionEvent(t *testing.T) {
	root := t.TempDir()
	outboxPath := filepath.Join(root, "outbox.json")
	if err := os.Mkdir(outboxPath, 0o700); err != nil {
		t.Fatal(err)
	}
	store := NewMemoryStore()
	mission := Mission{ID: "mission_push_failure", Version: 1, OrganizationID: "org_push_failure", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := store.CreateMission(mission); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{
		store:      store,
		metrics:    &RuntimeMetrics{},
		push:       &PushService{endpoint: "https://push.example.test/send"},
		pushOutbox: &PushOutbox{path: outboxPath, items: map[string]PushOutboxItem{}},
	}
	err := runtime.event(mission, "mission.failed", "", map[string]any{"state": "failed"})
	if err != nil {
		t.Fatalf("optional notification failure changed persisted mission event result: %v", err)
	}
	if got := runtime.Metrics().PushOutboxFailures; got != 1 {
		t.Fatalf("push outbox failure metric=%d, want 1", got)
	}
}

func TestContextStorePersistsProjectAndMemory(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(filepath.Join(root, "context"))
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("DZ23", filepath.Join(root, "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemory(Memory{ProjectID: project.ID, Kind: "decision", Content: "usar aprovação antes de publicar", Confidence: 1, Source: "test"}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewContextStore(filepath.Join(root, "context"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reloaded.GetProject(project.ID); err != nil {
		t.Fatal(err)
	}
	memories := reloaded.SearchMemories(project.ID, "aprovação", 10)
	if len(memories) != 1 || memories[0].Content == "" {
		t.Fatalf("memories = %+v", memories)
	}
}

func TestSandboxExecRunsIsolatedPython(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_SANDBOX_MODE", "best-effort")
	t.Setenv("OLLAMA_AGENT_SANDBOX_ALLOW_BEST_EFFORT", "true")
	if _, err := exec.LookPath("unshare"); err != nil {
		t.Skip("sandbox test requires unshare")
	}
	probe := exec.Command("unshare", "--user", "--map-root-user", "--mount", "--pid", "--fork", "--mount-proc", "--net", "/bin/true")
	if output, err := probe.CombinedOutput(); err != nil {
		t.Skipf("user namespace sandbox unavailable: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: root, Planner: fixedPlanner{steps: []Step{{ID: "step_1", Kind: "sandbox.exec", Title: "run", Risk: RiskWrite, RequiresApproval: true, State: StepPending, Input: map[string]any{"language": "python", "code": "print(2 + 2)"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "executar código", Capabilities: []string{"sandbox:execute"}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err = runtime.DecideApproval(mission.ID, mission.Approvals[0].ID, true, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Run(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
	completed, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, ok := completed.Plan[0].Result.(map[string]any)
	stdout, stdoutOK := result["stdout"].(string)
	if completed.State != MissionCompleted || !ok || !stdoutOK || !strings.Contains(stdout, "4") {
		t.Fatalf("completed = %+v", completed)
	}
}

func TestSandboxIsolationModeIsBoundToApproval(t *testing.T) {
	requireLinuxSandboxExecutor(t)
	t.Setenv("OLLAMA_AGENT_SANDBOX_MODE", "best-effort")
	t.Setenv("OLLAMA_AGENT_SANDBOX_ALLOW_BEST_EFFORT", "true")
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: root, Planner: fixedPlanner{steps: []Step{{ID: "step_1", Kind: "sandbox.exec", Title: "run", Risk: RiskWrite, RequiresApproval: true, State: StepPending, Input: map[string]any{"language": "python", "code": "print('must not run')", "sandbox_mode": "strict"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "approve sandbox mode", Capabilities: []string{"sandbox:execute"}})
	if err != nil {
		t.Fatal(err)
	}
	if mode := stringInput(mission.Plan[0].Input, "sandbox_mode", ""); mode != "best-effort" {
		t.Fatalf("planner was able to select sandbox mode %q; want authoritative best-effort", mode)
	}
	mission, err = runtime.DecideApproval(mission.ID, mission.Approvals[0].ID, true, "approved")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("OLLAMA_AGENT_SANDBOX_MODE", "strict")
	if err := runtime.Run(context.Background(), mission.ID); err == nil || !strings.Contains(err.Error(), ErrApprovalPayloadChanged.Error()) {
		t.Fatalf("sandbox mode drift error=%v, want approval payload mismatch", err)
	}
	completed, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != MissionFailed || completed.Plan[0].State != StepFailed {
		t.Fatalf("mode drift did not fail closed: mission=%s step=%s", completed.State, completed.Plan[0].State)
	}
}

func TestRuntimeFailsClosedWhenApprovalRecordIsMissing(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_SANDBOX_MODE", "best-effort")
	t.Setenv("OLLAMA_AGENT_SANDBOX_ALLOW_BEST_EFFORT", "true")
	store := NewMemoryStore()
	runtime, err := NewRuntime(RuntimeConfig{Store: store, WorkspaceRoot: t.TempDir(), Planner: fixedPlanner{steps: []Step{{ID: "step_1", Kind: "sandbox.exec", Title: "run", Risk: RiskWrite, RequiresApproval: true, Input: map[string]any{"language": "python", "code": "print('must not run')"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "missing approval regression", Capabilities: []string{"sandbox:execute"}})
	if err != nil {
		t.Fatal(err)
	}
	mission.Approvals = nil
	mission.State = MissionReady
	mission.Version++
	if err := store.PutMission(mission); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Run(context.Background(), mission.ID); !errors.Is(err, ErrApprovalPayloadChanged) {
		t.Fatalf("missing approval execution error=%v, want fail-closed approval error", err)
	}
	stored, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != MissionFailed || stored.Plan[0].State == StepSucceeded {
		t.Fatalf("mission executed or did not fail closed: state=%s step=%s", stored.State, stored.Plan[0].State)
	}
}

func TestSandboxExecUsesAnchoredWorkspaceAfterPathReplacement(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("anchored namespace mount test requires Linux")
	}
	if _, err := exec.LookPath("unshare"); err != nil {
		t.Skip("sandbox test requires unshare")
	}
	probe := exec.Command("unshare", "--user", "--map-root-user", "--mount", "--pid", "--fork", "--mount-proc", "--net", "/bin/true")
	if output, err := probe.CombinedOutput(); err != nil {
		t.Skipf("user namespace sandbox unavailable: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	t.Setenv("OLLAMA_AGENT_SANDBOX_MODE", "best-effort")
	t.Setenv("OLLAMA_AGENT_SANDBOX_ALLOW_BEST_EFFORT", "true")
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "marker.txt"), []byte("original workspace"), 0o600); err != nil {
		t.Fatal(err)
	}
	siblingSecret := workspace + ".secret"
	if err := os.WriteFile(siblingSecret, []byte("OUTSIDE_SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(siblingSecret)
	root, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	originalPath := workspace + ".original"
	if err := os.Rename(workspace, originalPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "marker.txt"), []byte("replacement workspace"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := (sandboxExecTool{}).Execute(context.Background(), ToolContext{
		StepID: "step_anchored_sandbox", Workspace: workspace, WorkspaceRoot: root,
	}, map[string]any{"language": "python", "code": "print(open('marker.txt', encoding='utf-8').read())\nfor p in ['/proc/self/fd/3/../secret.txt', '" + siblingSecret + "', '/etc/passwd']:\n try:\n  print('ESCAPED:' + open(p).read())\n except Exception:\n  print('blocked')", "sandbox_mode": "best-effort", "sandbox_gate_status": string(GateStatusNotConfigured), "sandbox_approval_notice": sandboxBestEffortApprovalNotice})
	if err != nil {
		t.Fatalf("anchored sandbox execution: %v; result=%+v", err, result.Value)
	}
	value, ok := result.Value.(map[string]any)
	stdout, stdoutOK := value["stdout"].(string)
	if !ok || !stdoutOK || !strings.Contains(stdout, "original workspace") || strings.Contains(stdout, "replacement workspace") || strings.Contains(stdout, "OUTSIDE_SECRET") || strings.Contains(stdout, "root:x:") || strings.Count(stdout, "blocked") != 3 {
		t.Fatalf("sandbox output=%+v, want workspace-only content", result.Value)
	}
}

func TestApprovalBindsActorOrganizationAndReason(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir(), Planner: fixedPlanner{steps: []Step{{ID: "step_1", Kind: "workspace.write", Title: "write", Risk: RiskWrite, RequiresApproval: true, Input: map[string]any{"path": "approval.txt", "content": "ok"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "aprovar operação", OrganizationID: "org_a", Capabilities: []string{"workspace:write"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.DecideApprovalForActor(mission.ID, mission.Approvals[0].ID, true, "approved", "user_a", "org_b"); err == nil {
		t.Fatal("expected organization mismatch")
	}
	if _, err := runtime.DecideApprovalForActor(mission.ID, mission.Approvals[0].ID, true, "", "user_a", "org_a"); err == nil {
		t.Fatal("expected empty reason rejection")
	}
	mission, err = runtime.DecideApprovalForActor(mission.ID, mission.Approvals[0].ID, true, "approved", "user_a", "org_a")
	if err != nil {
		t.Fatal(err)
	}
	approval := mission.Approvals[0]
	if approval.ActorID != "user_a" || approval.OrganizationID != "org_a" || approval.Nonce == "" || approval.Policy != "capabilities:workspace:write;risk:write" || approval.ExpiresAt == nil {
		t.Fatalf("approval metadata = %+v", approval)
	}
}

func TestApprovalValidationBindsPolicyExpiryAndDecisionActor(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir(), Planner: fixedPlanner{steps: []Step{{ID: "step_1", Kind: "workspace.write", Title: "write", Risk: RiskWrite, RequiresApproval: true, Input: map[string]any{"path": "approval-validation.txt", "content": "ok"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "approval record validation", OrganizationID: "org_a", Capabilities: []string{"workspace:write"}})
	if err != nil {
		t.Fatal(err)
	}
	assertInvalid := func(name string, mutate func(*Approval)) {
		t.Run(name, func(t *testing.T) {
			copy := cloneMission(mission)
			mutate(&copy.Approvals[0])
			if err := runtime.validateMissionApprovalBindings(copy); err == nil {
				t.Fatal("tampered approval record was accepted")
			}
		})
	}
	assertInvalid("policy", func(approval *Approval) { approval.Policy = "capabilities:;risk:read" })
	assertInvalid("missing expiry", func(approval *Approval) { approval.ExpiresAt = nil })
	assertInvalid("expired", func(approval *Approval) {
		expired := time.Now().UTC().Add(-time.Second)
		approval.ExpiresAt = &expired
	})
	assertInvalid("approved without actor", func(approval *Approval) {
		approval.Status = ApprovalApproved
		approval.ActorID = ""
		approval.Reason = "approved"
	})
	assertInvalid("approved without reason", func(approval *Approval) {
		approval.Status = ApprovalApproved
		approval.ActorID = "admin_a"
		approval.Reason = ""
	})
}

func TestApprovalCASAndNonceAreSingleUse(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir(), Planner: fixedPlanner{steps: []Step{{ID: "step_1", Kind: "workspace.write", Title: "write", Risk: RiskWrite, RequiresApproval: true, Input: map[string]any{"path": "approval-cas.txt", "content": "ok"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "approval CAS", OrganizationID: "org_a", Capabilities: []string{"workspace:write"}})
	if err != nil {
		t.Fatal(err)
	}
	approval := mission.Approvals[0]
	if _, err := runtime.DecideApprovalForActorCAS(mission.ID, approval.ID, true, "approved", "admin_a", "org_a", mission.Version, "wrong"); !errors.Is(err, ErrApprovalNonceMismatch) {
		t.Fatalf("wrong nonce err=%v", err)
	}
	if _, err := runtime.DecideApprovalForActorCAS(mission.ID, approval.ID, true, strings.Repeat("x", 2049), "admin_a", "org_a", mission.Version, approval.Nonce); !errors.Is(err, ErrApprovalReasonTooLong) {
		t.Fatalf("oversized reason err=%v", err)
	}
	decided, err := runtime.DecideApprovalForActorCAS(mission.ID, approval.ID, true, "approved", "admin_a", "org_a", mission.Version, approval.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	if decided.Approvals[0].Status != ApprovalApproved {
		t.Fatalf("decided=%+v", decided.Approvals[0])
	}
	if _, err := runtime.DecideApprovalForActorCAS(mission.ID, approval.ID, true, "replay", "admin_a", "org_a", decided.Version, approval.Nonce); err == nil {
		t.Fatal("approval nonce was reusable")
	}
}

func TestApprovalReasonDLPIsRedactedBeforeMissionAndEvent(t *testing.T) {
	const secret = "ghp_abcdefghijklmnopqrstuvwxyz123456"
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir(), Planner: fixedPlanner{steps: []Step{{ID: "step_1", Kind: "workspace.write", Title: "write", Risk: RiskWrite, RequiresApproval: true, Input: map[string]any{"path": "approval-dlp.txt", "content": "ok"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "approval DLP", OrganizationID: "org_a", Capabilities: []string{"workspace:write"}})
	if err != nil {
		t.Fatal(err)
	}
	approval := mission.Approvals[0]
	reason := `{"api_key":"` + secret + `"}`
	decided, err := runtime.DecideApprovalForActorCAS(mission.ID, approval.ID, true, reason, "admin_a", "org_a", mission.Version, approval.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	if decided.Approvals[0].Reason != "[REDACTED]" {
		t.Fatalf("returned approval reason = %q, want DLP redaction", decided.Approvals[0].Reason)
	}
	stored, err := runtime.GetMission(mission.ID)
	if err != nil || stored.Approvals[0].Reason != "[REDACTED]" {
		t.Fatalf("stored approval reason = %q err=%v", stored.Approvals[0].Reason, err)
	}
	events, err := runtime.Events(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Type != "approval.decided" {
			continue
		}
		encoded, marshalErr := json.Marshal(event.Payload)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("approval event contains raw credential: %s", encoded)
		}
		return
	}
	t.Fatal("approval decision event was not recorded")
}

func TestApprovalCASConcurrentDecisionsHaveOneWinner(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir(), Planner: fixedPlanner{steps: []Step{{ID: "step_1", Kind: "workspace.write", Title: "write", Risk: RiskWrite, RequiresApproval: true, Input: map[string]any{"path": "approval-concurrent.txt", "content": "ok"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "approval concurrent CAS", OrganizationID: "org_a", Capabilities: []string{"workspace:write"}})
	if err != nil {
		t.Fatal(err)
	}
	approval := mission.Approvals[0]
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, actor := range []string{"admin_a", "admin_b"} {
		group.Add(1)
		go func(actor string) {
			defer group.Done()
			<-start
			_, decideErr := runtime.DecideApprovalForActorCAS(mission.ID, approval.ID, true, "approved", actor, "org_a", mission.Version, approval.Nonce)
			results <- decideErr
		}(actor)
	}
	close(start)
	group.Wait()
	close(results)
	winners := 0
	conflicts := 0
	for decideErr := range results {
		if decideErr == nil {
			winners++
		} else if errors.Is(decideErr, ErrApprovalVersionConflict) {
			conflicts++
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("winners=%d conflicts=%d", winners, conflicts)
	}
	decided, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if decided.Approvals[0].Status != ApprovalApproved {
		t.Fatalf("approval status = %s, want APPROVED", decided.Approvals[0].Status)
	}
}

func TestCapabilityPolicyDefaultsToLocalScopes(t *testing.T) {
	if !capabilityAllowed(ToolDescriptor{Name: "workspace.read", Scopes: []string{"workspace:read"}}, []string{"workspace:read", "workspace:write"}) {
		t.Fatal("local workspace read should be allowed by the default policy")
	}
	if capabilityAllowed(ToolDescriptor{Name: "browser.operator", Scopes: []string{"browser:navigate"}}, []string{"workspace:read", "workspace:write"}) {
		t.Fatal("browser capability must require explicit grant")
	}
	if !capabilityAllowed(ToolDescriptor{Name: "browser.operator", Scopes: []string{"browser:navigate"}}, []string{"browser:navigate"}) {
		t.Fatal("explicit browser capability should be accepted")
	}
	got := normalizeMissionCapabilities([]string{" browser:navigate ", "workspace:read", "browser:navigate"})
	if strings.Join(got, ",") != "browser:navigate,workspace:read" {
		t.Fatalf("normalized capabilities = %v", got)
	}
}

func TestRuntimeRejectsUnknownMissionCapability(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "invalid grant", Capabilities: []string{"workspace:read", "payments:charge"}}); !errors.Is(err, ErrUnknownCapability) {
		t.Fatalf("err=%v, want unknown capability", err)
	}
}

func TestScheduleClaimIsIdempotent(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(filepath.Join(root, "context"))
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := store.CreateSchedule(Schedule{Objective: "verificar status", IntervalSeconds: 60})
	if err != nil {
		t.Fatal(err)
	}
	checkAt := time.Now().UTC().Add(2 * time.Minute)
	due := store.ClaimDueSchedules(checkAt)
	if len(due) != 1 || due[0].ID != schedule.ID {
		t.Fatalf("due = %+v", due)
	}
	if again := store.ClaimDueSchedules(checkAt); len(again) != 0 {
		t.Fatalf("schedule claimed twice: %+v", again)
	}
}

func TestBrowserOperatorNavigateAndSnapshot(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE", "1")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><head><title>DZ23</title></head><body><h1>Hello Browser</h1></body></html>"))
	}))
	defer server.Close()
	root := t.TempDir()
	tool := browserOperatorTool{}
	toolContext := ToolContext{MissionID: "browser-test", StepID: "step_1", Workspace: root, OrganizationID: LocalOrganizationID}
	if _, err := tool.Execute(context.Background(), toolContext, map[string]any{"action": "navigate", "url": server.URL}); err != nil {
		if browserDependencyUnavailable(err) {
			t.Skipf("Playwright browser dependency unavailable; skipping integration test: %v", err)
		}
		t.Fatal(err)
	}
	result, err := tool.Execute(context.Background(), toolContext, map[string]any{"action": "snapshot"})
	if err != nil {
		t.Fatal(err)
	}
	content, ok := result.Value.(map[string]any)["content"].(string)
	if !ok || !strings.Contains(content, "Hello Browser") {
		t.Fatalf("browser result = %+v", result.Value)
	}
}

// browserDependencyUnavailable reports whether the error is caused by the
// Python interpreter or the Playwright module being absent from the dev
// machine, rather than by a defect in the browser operator. CI installs the
// dependency and exercises the full path; locally we skip instead of failing.
func browserDependencyUnavailable(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	for _, marker := range []string{
		"No module named 'playwright'",
		"requires python3 or python on PATH",
		"executable file not found",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func TestBrowserOperatorRejectsOrganizationScopeWithoutStrictSandbox(t *testing.T) {
	tool := browserOperatorTool{}
	for _, organizationID := range []string{"", "org_example"} {
		_, err := tool.Execute(context.Background(), ToolContext{OrganizationID: organizationID}, map[string]any{
			"action": "navigate",
			"url":    "https://example.com",
		})
		if err == nil || !strings.Contains(err.Error(), "strict OS and network sandbox") {
			t.Fatalf("organization %q error=%v, want explicit sandbox requirement", organizationID, err)
		}
	}
}

type plannerResolverStub struct {
	provider string
	model    string
	planner  Planner
}

func (s *plannerResolverStub) ResolvePlanner(provider, model string) (Planner, error) {
	s.provider = provider
	s.model = model
	return s.planner, nil
}

func TestRuntimeUsesExplicitPlannerResolverForRemoteProvider(t *testing.T) {
	resolver := &plannerResolverStub{planner: RulePlanner{}}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:           NewMemoryStore(),
		PlannerResolver: resolver,
		WorkspaceRoot:   t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "inspect remote provider",
		Provider:  "anthropic",
		Model:     "claude-sonnet-4-5",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolver.provider != "anthropic" || resolver.model != "claude-sonnet-4-5" {
		t.Fatalf("resolver request = %q/%q", resolver.provider, resolver.model)
	}
	if mission.Provider != "anthropic" || mission.Model != "claude-sonnet-4-5" {
		t.Fatalf("mission provider/model = %q/%q", mission.Provider, mission.Model)
	}
}

func TestRuntimeRejectsRemoteProviderWithoutResolver(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "remote", Provider: "codex", Model: "codex-mini"}); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err = %v, want explicit unavailable provider", err)
	}
}

type countingPlanner struct{ calls atomic.Int32 }

func (p *countingPlanner) Plan(_ context.Context, _ Mission) ([]Step, error) {
	p.calls.Add(1)
	return RulePlanner{}.Plan(context.Background(), Mission{Objective: "inspect"})
}

func TestRuntimeBlocksSensitiveObjectiveBeforeRemotePlanner(t *testing.T) {
	store := NewMemoryStore()
	planner := &countingPlanner{}
	resolver := &plannerResolverStub{planner: planner}
	runtime, err := NewRuntime(RuntimeConfig{Store: store, PlannerResolver: resolver, WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "analyze incident token=ghp_abcdefghijklmnopqrstuvwxyz123456",
		Provider:  "anthropic",
		Model:     "claude-sonnet-4-5",
	})
	if err == nil || !strings.Contains(err.Error(), "data-egress policy") {
		t.Fatalf("sensitive external mission error=%v", err)
	}
	if planner.calls.Load() != 0 {
		t.Fatalf("remote planner was called %d times", planner.calls.Load())
	}
	missions, err := store.ListMissions()
	if err != nil {
		t.Fatal(err)
	}
	if len(missions) != 0 {
		t.Fatalf("sensitive mission was persisted: %+v", missions)
	}
	_, err = runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective: `review incident payload {"api_key":"ordinary-secret-value"}`,
		Provider:  "anthropic",
		Model:     "claude-sonnet-4-5",
	})
	if err == nil || !strings.Contains(err.Error(), "data-egress policy") {
		t.Fatalf("embedded JSON credential objective error=%v", err)
	}
	if planner.calls.Load() != 0 {
		t.Fatalf("remote planner received embedded JSON credential; calls=%d", planner.calls.Load())
	}
	missions, err = store.ListMissions()
	if err != nil || len(missions) != 0 {
		t.Fatalf("embedded JSON credential mission persisted: missions=%+v err=%v", missions, err)
	}
}

type descriptorMutationTool struct{ executed atomic.Bool }

func (t *descriptorMutationTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "workspace.write", Version: "changed", Risk: RiskWrite, RequiresApproval: true, Scopes: []string{"workspace:write"}}
}

func (t *descriptorMutationTool) Execute(context.Context, ToolContext, map[string]any) (ToolResult, error) {
	t.executed.Store(true)
	return ToolResult{}, nil
}

func TestRuntimeInvalidatesApprovalWhenToolDescriptorChanges(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir(), Planner: fixedPlanner{steps: []Step{{ID: "step_1", Kind: "workspace.write", Title: "write", Risk: RiskWrite, RequiresApproval: true, Input: map[string]any{"path": "descriptor.txt", "content": "safe"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "descriptor binding", OrganizationID: "org_a", Capabilities: []string{"workspace:write"}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err = runtime.DecideApprovalForActor(mission.ID, mission.Approvals[0].ID, true, "approved for original tool", "user_a", "org_a")
	if err != nil {
		t.Fatal(err)
	}
	changed := &descriptorMutationTool{}
	runtime.tools.Register(changed)
	err = runtime.Run(context.Background(), mission.ID)
	if !errors.Is(err, ErrApprovalPayloadChanged) {
		t.Fatalf("run error=%v, want ErrApprovalPayloadChanged", err)
	}
	if changed.executed.Load() {
		t.Fatal("tool executed despite descriptor changing after approval")
	}
}

type readOnlyDescriptorMutationTool struct{ executed atomic.Bool }

func (t *readOnlyDescriptorMutationTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "git.repo.inspect", Version: "mutated", Description: "unexpected replacement", Risk: RiskRead, Scopes: []string{"repo:read"}}
}

func (t *readOnlyDescriptorMutationTool) Execute(context.Context, ToolContext, map[string]any) (ToolResult, error) {
	t.executed.Store(true)
	return ToolResult{Value: "executed"}, nil
}

func TestRuntimeInvalidatesReadOnlyToolWhenDescriptorChanges(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir(), Planner: fixedPlanner{steps: []Step{{ID: "step_read", Kind: "git.repo.inspect", Title: "inspect", Risk: RiskRead}}}})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: "descriptor drift", Capabilities: []string{"repo:read"}})
	if err != nil {
		t.Fatal(err)
	}
	changed := &readOnlyDescriptorMutationTool{}
	runtime.tools.Register(changed)
	if err := runtime.Run(context.Background(), mission.ID); !errors.Is(err, ErrApprovalPayloadChanged) {
		t.Fatalf("run error=%v, want descriptor mismatch", err)
	}
	if changed.executed.Load() {
		t.Fatal("replacement read-only tool executed despite descriptor drift")
	}
}

func TestRuntimeGitRepoInspectUsesIsolatedSnapshotBaseline(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	repo := initWorkspaceSnapshotGit(t)
	projectRoot := filepath.Join(repo, "service")
	if err := os.MkdirAll(projectRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	writeSnapshotFixture(t, repo, "service/app.txt", "source version\n")
	runSnapshotFixtureGit(t, repo, "add", "service/app.txt")
	runSnapshotFixtureGit(t, repo, "commit", "-m", "initial")

	contextStore, err := NewContextStore(filepath.Join(t.TempDir(), "context"))
	if err != nil {
		t.Fatal(err)
	}
	if err := contextStore.SetWorkspaceRoot(repo); err != nil {
		t.Fatal(err)
	}
	project, err := contextStore.CreateProject("nested service", projectRoot, "org_git_snapshot")
	if err != nil {
		t.Fatal(err)
	}
	dataRoot := t.TempDir()
	missionStore, err := NewJSONStore(filepath.Join(dataRoot, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         missionStore,
		WorkspaceRoot: repo,
		DataRoot:      dataRoot,
		Context:       contextStore,
		Planner: fixedPlanner{steps: []Step{{
			ID:    "step_repo_inspect",
			Kind:  "git.repo.inspect",
			Title: "Inspect isolated repository changes",
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:        "inspect repository status after an approved edit",
		ProjectID:        project.ID,
		OrganizationID:   "org_git_snapshot",
		Capabilities:     []string{"repo:read", "workspace:read", "workspace:write"},
		IsolateWorkspace: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(mission.Workspace), "/tree/service") {
		t.Fatalf("nested mission workspace=%q; want snapshot tree/service", mission.Workspace)
	}
	root, err := openMissionWorkspaceSnapshot(dataRoot, mission)
	if err != nil {
		t.Fatal(err)
	}
	input, err := prepareWorkspaceWriteApproval(mission.Workspace, map[string]any{
		"path":    "app.txt",
		"content": "agent version\n",
	})
	if err != nil {
		_ = root.Close()
		t.Fatal(err)
	}
	writeResult, err := writeWorkspaceFile(ToolContext{
		MissionID:     mission.ID,
		StepID:        "step_approved_edit",
		Workspace:     mission.Workspace,
		WorkspaceRoot: root,
	}, input)
	_ = root.Close()
	if err != nil {
		t.Fatal(err)
	}
	mission.Artifacts = appendUniqueArtifacts(mission.Artifacts, writeResult.Artifacts)
	oldVersion := mission.Version
	mission.Version++
	mission.UpdatedAt = time.Now().UTC()
	if err := missionStore.PutMissionIfVersion(mission, oldVersion); err != nil {
		t.Fatal(err)
	}

	if err := runtime.Run(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
	completed, err := runtime.GetMission(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != MissionCompleted || len(completed.Plan) != 1 || completed.Plan[0].State != StepSucceeded {
		t.Fatalf("mission did not complete inspection: state=%s plan=%+v", completed.State, completed.Plan)
	}
	result, ok := completed.Plan[0].Result.(map[string]any)
	if !ok {
		t.Fatalf("Git result type=%T value=%#v", completed.Plan[0].Result, completed.Plan[0].Result)
	}
	if result["repository_root"] != "." || result["baseline_is_mission_start"] != true || result["read_only"] != true || result["change_scope"] != "persisted_mission_artifacts_only" || result["changes_complete"] != false {
		t.Fatalf("unsafe/missing snapshot metadata: %#v", result)
	}
	changes, ok := result["changes"].([]any)
	if !ok || len(changes) != 1 {
		t.Fatalf("snapshot changes=%#v", result["changes"])
	}
	change, ok := changes[0].(map[string]any)
	if !ok || change["change"] != "modified" || change["path"] != "app.txt" {
		t.Fatalf("nested project change=%#v", changes[0])
	}
	if result["staged_diff_available"] != false {
		t.Fatalf("snapshot must not claim access to source index: %#v", result)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), repo) || strings.Contains(string(encoded), dataRoot) {
		t.Fatalf("Git inspection exposed a host path: %s", encoded)
	}
	original, err := os.ReadFile(filepath.Join(repo, "service", "app.txt"))
	if err != nil || string(original) != "source version\n" {
		t.Fatalf("source repository changed: content=%q err=%v", original, err)
	}
}

func TestRuntimeCreateMissionReturnsRedactedMissionValues(t *testing.T) {
	const secret = "example-secret-value"
	planner := fixedPlanner{steps: []Step{{ID: "step_1", Kind: "workspace.list", Title: "token=ghp_abcdefghijklmnopqrstuvwxyz123456", Risk: RiskRead, Input: map[string]any{"api_key": secret, "path": "."}}}}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir(), Planner: planner})
	if err != nil {
		t.Fatal(err)
	}
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{Objective: `Investigate {"api_key":"` + secret + `"}`})
	if err != nil {
		t.Fatal(err)
	}
	for _, current := range []Mission{mission, func() Mission {
		loaded, loadErr := runtime.GetMission(mission.ID)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		return loaded
	}()} {
		encoded, err := json.Marshal(current)
		if err != nil || strings.Contains(string(encoded), secret) || strings.Contains(string(encoded), "ghp_abcdefghijklmnopqrstuvwxyz123456") {
			t.Fatalf("mission response/cache exposed sensitive data: err=%v", err)
		}
	}
}

type mockBrowserTool struct{}

func (mockBrowserTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "browser.operator", Version: "1", Risk: RiskRead, Scopes: []string{"browser:navigate"}}
}

func (mockBrowserTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	return ToolResult{
		Value: map[string]any{
			"url":        "https://example.com",
			"title":      "Example Domain",
			"screenshot": "data:image/jpeg;base64,ZmFrZXNjcmVlbnNob3Q=",
		},
	}, nil
}

func TestRuntimeEmitsBrowserFrameEvent(t *testing.T) {
	store := NewMemoryStore()
	workspace := t.TempDir()
	planner := fixedPlanner{
		steps: []Step{
			{
				ID:    "step_browser_1",
				Kind:  "browser.operator",
				Title: "Navigate to example.com",
				Risk:  RiskRead,
				Input: map[string]any{"action": "navigate", "url": "https://example.com"},
			},
		},
	}
	registry := NewRegistry()
	registry.Register(mockBrowserTool{})
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         store,
		WorkspaceRoot: workspace,
		Planner:       planner,
		Tools:         registry,
	})
	if err != nil {
		t.Fatal(err)
	}

	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:    "Test browser frame event",
		AutoRun:      false,
		Capabilities: []string{"browser:navigate"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// browser.operator tem efeito externo e exige approval; o planner não pode
	// rebaixar esse risco. Aprovar o passo antes de executar reflete o fluxo real.
	if len(mission.Approvals) != 1 {
		t.Fatalf("expected one approval-gated browser step, got %d", len(mission.Approvals))
	}
	mission, err = runtime.DecideApproval(mission.ID, mission.Approvals[0].ID, true, "approved for test")
	if err != nil {
		t.Fatal(err)
	}

	if err := runtime.Run(context.Background(), mission.ID); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	events, err := runtime.Events(mission.ID)
	if err != nil {
		t.Fatal(err)
	}

	var foundFrame bool
	for _, evt := range events {
		if evt.Type == "browser.frame" {
			foundFrame = true
			payload, ok := evt.Payload.(map[string]any)
			if !ok {
				t.Fatalf("expected map[string]any payload, got %T", evt.Payload)
			}
			if payload["url"] != "https://example.com" {
				t.Errorf("expected url https://example.com, got %v", payload["url"])
			}
			if payload["screenshot"] != "data:image/jpeg;base64,ZmFrZXNjcmVlbnNob3Q=" {
				t.Errorf("unexpected screenshot: %v", payload["screenshot"])
			}
		}
	}
	if !foundFrame {
		t.Fatalf("expected browser.frame event to be emitted in mission events, got: %+v", events)
	}
}
