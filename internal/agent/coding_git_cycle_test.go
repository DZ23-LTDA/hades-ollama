package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupSampleGitRepo creates a real local git repository for testing.
func setupSampleGitRepo(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()

	gitExec, err := trustedGitExecutable()
	if err != nil {
		gitExec, err = exec.LookPath("git")
		if err != nil {
			t.Skipf("git executable not found: %v", err)
		}
	}

	runGit := func(args ...string) {
		cmd := exec.Command(gitExec, args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Tester",
			"GIT_AUTHOR_EMAIL=tester@example.com",
			"GIT_COMMITTER_NAME=Tester",
			"GIT_COMMITTER_EMAIL=tester@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %v (output: %s)", strings.Join(args, " "), err, out)
		}
	}

	runGit("init", "-b", "main")
	runGit("config", "user.name", "Tester")
	runGit("config", "user.email", "tester@example.com")

	goMod := "module example.com/calc\n\ngo 1.22\n"
	calcGo := "package calc\n\nfunc Add(a, b int) int { return a - b }\n"
	calcTest := "package calc\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fatalf(\"expected 5, got %d\", Add(2, 3))\n\t}\n}\n"

	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "calc.go"), []byte(calcGo), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "calc_test.go"), []byte(calcTest), 0o600); err != nil {
		t.Fatal(err)
	}

	runGit("add", "-A")
	runGit("commit", "-m", "initial commit with buggy Add function")

	return repoDir
}

func TestCreateAndRemoveGitWorktree(t *testing.T) {
	repoDir := setupSampleGitRepo(t)
	dataDir := t.TempDir()
	ctx := context.Background()

	session, err := CreateGitWorktree(ctx, repoDir, dataDir, "test_mis_01", "agent/feature_x")
	if err != nil {
		t.Fatalf("CreateGitWorktree failed: %v", err)
	}
	defer RemoveGitWorktree(ctx, session)

	if session.BranchName != "agent/feature_x" {
		t.Errorf("expected branch agent/feature_x, got %s", session.BranchName)
	}
	if _, err := os.Stat(session.WorktreeDir); err != nil {
		t.Errorf("worktree directory does not exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(session.WorktreeDir, "calc.go")); err != nil {
		t.Errorf("calc.go not found in worktree: %v", err)
	}

	// Verify origin is still at base commit and clean
	if err := RemoveGitWorktree(ctx, session); err != nil {
		t.Fatalf("RemoveGitWorktree failed: %v", err)
	}
	if session.Status != "removed" {
		t.Errorf("expected status removed, got %s", session.Status)
	}
}

func TestProjectTestRunnerGo(t *testing.T) {
	repoDir := setupSampleGitRepo(t)
	ctx := context.Background()

	// Initial tests should fail because Add(2, 3) returns -1 instead of 5
	result, err := RunProjectTests(ctx, repoDir, TestRunOptions{Framework: "go"})
	if err != nil {
		t.Fatalf("RunProjectTests error: %v", err)
	}
	if result.Passed {
		t.Fatalf("expected initial test to fail, but it passed")
	}
	if len(result.FailedTests) == 0 || result.FailedTests[0] != "TestAdd" {
		t.Errorf("expected failed test TestAdd, got: %v", result.FailedTests)
	}

	// Fix the bug and test again
	fixedGo := "package calc\n\nfunc Add(a, b int) int { return a + b }\n"
	if err := os.WriteFile(filepath.Join(repoDir, "calc.go"), []byte(fixedGo), 0o600); err != nil {
		t.Fatal(err)
	}

	result2, err := RunProjectTests(ctx, repoDir, TestRunOptions{Framework: "go"})
	if err != nil {
		t.Fatalf("RunProjectTests error after fix: %v", err)
	}
	if !result2.Passed {
		t.Fatalf("expected test to pass after fix, but output: %s", result2.Output)
	}
}

func TestRepairLoopSucceeds(t *testing.T) {
	repoDir := setupSampleGitRepo(t)
	ctx := context.Background()

	mission := Mission{
		ID:        "mis_repair_01",
		Workspace: repoDir,
	}
	step := &Step{ID: "step_repair"}

	repairFn := func(attempt int, lastOutput string) error {
		fixedGo := "package calc\n\nfunc Add(a, b int) int { return a + b }\n"
		return os.WriteFile(filepath.Join(repoDir, "calc.go"), []byte(fixedGo), 0o600)
	}

	result, err := ExecuteRepairLoop(ctx, nil, mission, step, TestRunOptions{Framework: "go"}, repairFn, 3)
	if err != nil {
		t.Fatalf("ExecuteRepairLoop failed: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected test to pass after repair loop")
	}
}

// TestMissionGitWorktreeCycle runs the complete E2E cycle:
// plan -> worktree isolation -> edit -> test -> repair -> approval diff -> merge to origin
func TestMissionGitWorktreeCycle(t *testing.T) {
	repoDir := setupSampleGitRepo(t)
	dataDir := t.TempDir()
	ctx := context.Background()

	runtime, err := NewRuntime(RuntimeConfig{
		Planner:       RulePlanner{},
		WorkspaceRoot: repoDir,
		DataRoot:      dataDir,
	})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}

	// Step 1: Create mission with IsolateWorktree and AutoRepair
	missionReq := CreateMissionRequest{
		Objective:       "corrigir bug na funcao Add e testar",
		Capabilities:    []string{"workspace:read", "workspace:write", "repo:read"},
		Workspace:       repoDir,
		IsolateWorktree: true,
		WorktreeBranch:  "agent/fix-calc-add",
		AutoRepair:      true,
		MaxRepairTries:  3,
	}

	mission, err := runtime.CreateMission(ctx, missionReq)
	if err != nil {
		t.Fatalf("CreateMission with IsolateWorktree failed: %v", err)
	}

	if !mission.GitWorktreeActive {
		t.Fatalf("expected mission.GitWorktreeActive to be true")
	}
	if mission.GitBranch != "agent/fix-calc-add" {
		t.Errorf("expected branch agent/fix-calc-add, got %s", mission.GitBranch)
	}
	if mission.GitRepoRoot != repoDir {
		t.Errorf("expected GitRepoRoot %s, got %s", repoDir, mission.GitRepoRoot)
	}

	worktreeDir := mission.GitWorktreePath
	if _, err := os.Stat(worktreeDir); err != nil {
		t.Fatalf("worktree directory not found on disk: %v", err)
	}

	// Step 2: Confirm origin repo still has the bug
	originContent, err := os.ReadFile(filepath.Join(repoDir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(originContent), "return a - b") {
		t.Fatalf("origin repo was unexpectedly modified before merge")
	}

	// Step 3: Run tests in worktree (fails initially)
	initTest, err := RunProjectTests(ctx, worktreeDir, TestRunOptions{Framework: "go"})
	if err != nil {
		t.Fatalf("initial test error: %v", err)
	}
	if initTest.Passed {
		t.Fatalf("expected initial test in worktree to fail")
	}

	// Step 4: Repair loop fixes the bug in worktree
	repairFn := func(attempt int, lastOutput string) error {
		fixed := "package calc\n\nfunc Add(a, b int) int { return a + b }\n"
		return os.WriteFile(filepath.Join(worktreeDir, "calc.go"), []byte(fixed), 0o600)
	}
	repairResult, err := ExecuteRepairLoop(ctx, runtime, mission, &Step{ID: "step_repair"}, TestRunOptions{Framework: "go"}, repairFn, 3)
	if err != nil {
		t.Fatalf("repair loop failed: %v", err)
	}
	if !repairResult.Passed {
		t.Fatalf("expected repaired code to pass tests in worktree")
	}

	// Step 5: Check diff and approval payload before merge
	diffApproval, err := runtime.GetMissionWorktreeDiff(ctx, mission.ID)
	if err != nil {
		t.Fatalf("GetMissionWorktreeDiff failed: %v", err)
	}
	if diffApproval.Diff == "" {
		t.Fatalf("expected non-empty diff")
	}
	if diffApproval.DiffSHA256 == "" {
		t.Fatalf("expected non-empty DiffSHA256")
	}
	if !strings.Contains(diffApproval.Diff, "+func Add(a, b int) int { return a + b }") {
		t.Errorf("diff does not contain expected fix: %s", diffApproval.Diff)
	}

	// Confirm origin repo is STILL untouched before merge approval
	originContentBeforeMerge, _ := os.ReadFile(filepath.Join(repoDir, "calc.go"))
	if !strings.Contains(string(originContentBeforeMerge), "return a - b") {
		t.Fatalf("origin repo was prematurely modified before approval")
	}

	// Step 6: Approval granted -> Merge to origin
	mergeCommit, err := runtime.MergeMissionWorktree(ctx, mission.ID)
	if err != nil {
		t.Fatalf("MergeMissionWorktree failed: %v", err)
	}
	if mergeCommit == "" {
		t.Fatalf("expected valid merge commit SHA")
	}

	// Step 7: Verify origin repo now has the fix merged!
	originContentAfterMerge, err := os.ReadFile(filepath.Join(repoDir, "calc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(originContentAfterMerge), "return a + b") {
		t.Fatalf("expected origin repo to have merged fix, but got: %s", string(originContentAfterMerge))
	}

	// Run tests on origin repo: must pass now!
	originTests, err := RunProjectTests(ctx, repoDir, TestRunOptions{Framework: "go"})
	if err != nil {
		t.Fatalf("origin test error: %v", err)
	}
	if !originTests.Passed {
		t.Fatalf("expected origin tests to pass after merge, output: %s", originTests.Output)
	}

	t.Logf("Mission git worktree cycle completed successfully: merge commit %s", mergeCommit)
}
