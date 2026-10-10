package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func bootstrapInput() ContextBootstrapInput {
	return ContextBootstrapInput{
		Version:             "2.1.0",
		OrganizationID:      "org-a",
		ProjectID:           "proj-1",
		MissionID:           "mission-1",
		Workspace:           "C:/work/repo",
		Objective:           "Corrigir o bug de ordenação",
		PlanTitles:          []string{"Inspecionar", "Corrigir", "Testar"},
		Checkpoints:         []string{"baseline verde"},
		GrantedCapabilities: []string{"workspace:read", "workspace:write", "terminal:allowlisted"},
		Policy:              DefaultCapabilityPolicy(),
		UntrustedSources:    []string{"web", "whatsapp"},
	}
}

func TestBuildContextBootstrapRequiresTenantScope(t *testing.T) {
	_, err := BuildContextBootstrap(ContextBootstrapInput{})
	if !errors.Is(err, ErrContextBootstrapEmpty) {
		t.Fatalf("expected ErrContextBootstrapEmpty, got %v", err)
	}
}

func TestBuildContextBootstrapDeclaresPermissionStates(t *testing.T) {
	input := bootstrapInput()
	input.GrantedCapabilities = append(input.GrantedCapabilities, "harness:cli")
	bootstrap, err := BuildContextBootstrap(input)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	expectations := map[string]string{
		"workspace:read":       PermissionRead,
		"workspace:write":      PermissionApproval,
		"terminal:allowlisted": PermissionApproval,
		"repo:read":            PermissionForbidden,
		"mcp:call":             PermissionForbidden,
		"harness:cli":          PermissionUnavailable,
	}
	for scope, want := range expectations {
		if got := bootstrap.Permissions[scope]; got != want {
			t.Errorf("scope %q: want %q, got %q", scope, want, got)
		}
	}
	if bootstrap.Delegation != PermissionForbidden {
		t.Errorf("delegation must stay forbidden without an explicit grant, got %q", bootstrap.Delegation)
	}
	if bootstrap.Identity != "Hades" || bootstrap.Version != "2.1.0" {
		t.Fatalf("identity/version missing: %+v", bootstrap)
	}
}

func TestBuildContextBootstrapIsDeterministic(t *testing.T) {
	first, err := BuildContextBootstrap(bootstrapInput())
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := BuildContextBootstrap(bootstrapInput())
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.SystemPreamble != second.SystemPreamble {
		t.Fatal("context bootstrap must be deterministic for the same input")
	}
	readIndex := strings.Index(first.SystemPreamble, "workspace:read")
	writeIndex := strings.Index(first.SystemPreamble, "workspace:write")
	if readIndex < 0 || writeIndex < 0 || readIndex > writeIndex {
		t.Fatalf("permission lines must be sorted: read=%d write=%d", readIndex, writeIndex)
	}
}

func TestBuildContextBootstrapFiltersMemoriesByProject(t *testing.T) {
	input := bootstrapInput()
	input.Memories = []Memory{
		{ID: "m-2", ProjectID: "proj-1", Kind: "fact", Content: "segunda", CreatedAt: time.Now()},
		{ID: "m-1", ProjectID: "proj-1", Kind: "fact", Content: "primeira", CreatedAt: time.Now()},
		{ID: "m-9", ProjectID: "proj-outro", Kind: "fact", Content: "vazamento", CreatedAt: time.Now()},
	}
	bootstrap, err := BuildContextBootstrap(input)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(bootstrap.MemoryRefs) != 2 {
		t.Fatalf("expected 2 memories of proj-1, got %d (%+v)", len(bootstrap.MemoryRefs), bootstrap.MemoryRefs)
	}
	if bootstrap.MemoryRefs[0].ID != "m-1" || bootstrap.MemoryRefs[1].ID != "m-2" {
		t.Fatalf("memories must be ordered by id, got %+v", bootstrap.MemoryRefs)
	}
	if strings.Contains(bootstrap.SystemPreamble, "vazamento") {
		t.Fatal("context leaked a memory from another project")
	}
	if bootstrap.Limits.MemoriesConsidered != 2 {
		t.Fatalf("considered = %d", bootstrap.Limits.MemoriesConsidered)
	}
}

func TestBuildContextBootstrapWithoutProjectInjectsNoMemory(t *testing.T) {
	input := bootstrapInput()
	input.ProjectID = ""
	input.Memories = []Memory{
		{ID: "m-1", ProjectID: "proj-1", Kind: "fact", Content: "dado sensível do projeto"},
	}
	bootstrap, err := BuildContextBootstrap(input)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(bootstrap.MemoryRefs) != 0 {
		t.Fatalf("no project means no memory injection, got %+v", bootstrap.MemoryRefs)
	}
	if strings.Contains(bootstrap.SystemPreamble, "dado sensível do projeto") {
		t.Fatal("memory leaked without a project scope")
	}
}

func TestBuildContextBootstrapRedactsCredentialShapedContent(t *testing.T) {
	const raw = "sk-live-abcdefghijklmnopqrstuvwxyz012345"
	input := bootstrapInput()
	input.Memories = []Memory{
		{ID: "m-1", ProjectID: "proj-1", Kind: "fact", Content: "OPENAI_API_KEY=" + raw},
	}
	bootstrap, err := BuildContextBootstrap(input)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if strings.Contains(bootstrap.SystemPreamble, raw) {
		t.Fatal("context bootstrap leaked a credential-shaped value")
	}
	for _, memory := range bootstrap.MemoryRefs {
		if strings.Contains(memory.Snippet, raw) {
			t.Fatal("memory snippet leaked a credential-shaped value")
		}
	}
}

func TestBuildContextBootstrapBoundsMemoryCount(t *testing.T) {
	input := bootstrapInput()
	for _, id := range []string{
		"m-01", "m-02", "m-03", "m-04", "m-05", "m-06",
		"m-07", "m-08", "m-09", "m-10", "m-11", "m-12",
	} {
		input.Memories = append(input.Memories, Memory{ID: id, ProjectID: "proj-1", Kind: "fact", Content: "conteúdo " + id})
	}
	bootstrap, err := BuildContextBootstrap(input)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(bootstrap.MemoryRefs) != MaxContextBootstrapMemories {
		t.Fatalf("expected the memory cap of %d, got %d", MaxContextBootstrapMemories, len(bootstrap.MemoryRefs))
	}
	if !bootstrap.Limits.Truncated {
		t.Fatal("dropping memories must be reported as truncation")
	}
	if bootstrap.Limits.MemoriesConsidered != 12 {
		t.Fatalf("considered = %d", bootstrap.Limits.MemoriesConsidered)
	}
}

func TestBuildContextBootstrapKeepsPreambleWithinBudget(t *testing.T) {
	input := bootstrapInput()
	plan := make([]string, 0, 200)
	for range 200 {
		plan = append(plan, strings.Repeat("passo de plano muito longo ", 10))
	}
	input.PlanTitles = plan
	bootstrap, err := BuildContextBootstrap(input)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(bootstrap.SystemPreamble) > MaxContextBootstrapBytes {
		t.Fatalf("preamble exceeded the budget: %d bytes", len(bootstrap.SystemPreamble))
	}
	if !bootstrap.Limits.Truncated || !strings.Contains(bootstrap.SystemPreamble, contextTruncationMarker) {
		t.Fatal("truncation must be explicit in the rendered context")
	}
}

func TestBuildContextBootstrapMarksLocalOnlyAndUntrustedSources(t *testing.T) {
	input := bootstrapInput()
	input.LocalOnly = true
	input.DelegationAllowed = true
	bootstrap, err := BuildContextBootstrap(input)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.Contains(bootstrap.SystemPreamble, "LOCAL_ONLY") {
		t.Fatal("LOCAL_ONLY must be visible to the model")
	}
	if !strings.Contains(bootstrap.SystemPreamble, "NÃO CONFIÁVEL") {
		t.Fatal("untrusted sources must be declared to the model")
	}
	if !strings.Contains(bootstrap.SystemPreamble, "web") || !strings.Contains(bootstrap.SystemPreamble, "whatsapp") {
		t.Fatal("untrusted source names must be listed")
	}
	if bootstrap.Delegation != PermissionDelegate {
		t.Fatalf("explicit delegation grant must be declared, got %q", bootstrap.Delegation)
	}
	if bootstrap.SystemMessage() == "" {
		t.Fatal("SystemMessage must render the preamble")
	}
}

func TestBuildContextBootstrapNeverLeaksIdentifiersIntoThePreamble(t *testing.T) {
	input := bootstrapInput()
	input.Workspace = "C:/srv/privado/cliente-x/repo"
	input.ProjectID = "project-secret-id"
	input.OrganizationID = "org-secret-id"
	input.MissionID = "mission-secret-id"
	bootstrap, err := BuildContextBootstrap(input)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	for _, secret := range []string{
		"C:/srv/privado/cliente-x/repo",
		"project-secret-id",
		"org-secret-id",
		"mission-secret-id",
	} {
		if strings.Contains(bootstrap.SystemPreamble, secret) {
			t.Fatalf("preamble leaked %q", secret)
		}
	}
	// Os campos estruturados seguem disponíveis para o lado autorizado.
	if bootstrap.ProjectID != "project-secret-id" || bootstrap.Workspace != "C:/srv/privado/cliente-x/repo" {
		t.Fatalf("structured fields must keep the server-side scope: %+v", bootstrap)
	}
	if !strings.Contains(bootstrap.SystemPreamble, "Escopo autenticado") {
		t.Fatal("preamble must still declare that a scope is applied")
	}
}

func TestResolveContextPermissionNeverInventsAccess(t *testing.T) {
	known := map[string]struct{}{"workspace:read": {}}
	if got := ResolveContextPermission("workspace:write", map[string]struct{}{}, known); got != PermissionUnavailable {
		t.Fatalf("scope unknown to the policy must be unavailable, got %q", got)
	}
	if got := ResolveContextPermission("workspace:read", map[string]struct{}{}, known); got != PermissionForbidden {
		t.Fatalf("known but ungranted scope must be forbidden, got %q", got)
	}
	if got := ResolveContextPermission("", map[string]struct{}{}, known); got != PermissionUnavailable {
		t.Fatalf("empty scope must be unavailable, got %q", got)
	}
	if got := ResolveContextPermission("repo:read", map[string]struct{}{"repo:read": {}}, map[string]struct{}{"repo:read": {}}); got != PermissionRead {
		t.Fatalf("granted read scope must be PODE_LER, got %q", got)
	}
	if got := ResolveContextPermission("sandbox:execute", map[string]struct{}{"sandbox:execute": {}}, map[string]struct{}{"sandbox:execute": {}}); got != PermissionApproval {
		t.Fatalf("granted execution scope must require approval, got %q", got)
	}
}
