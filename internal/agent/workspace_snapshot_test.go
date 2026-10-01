package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

type snapshotFixtureState struct {
	Mode    os.FileMode
	Content string
	Link    string
}

func TestCreateWorkspaceSnapshotCopiesSafeWorkingTreeWithoutMutatingSource(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	source := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, source, ".gitignore", "ignored.txt\n")
	writeSnapshotFixture(t, source, "tracked.txt", "committed base\n")
	runSnapshotFixtureGit(t, source, "add", ".gitignore", "tracked.txt")
	runSnapshotFixtureGit(t, source, "commit", "-m", "snapshot base")

	writeSnapshotFixture(t, source, "tracked.txt", "modified working copy\n")
	writeSnapshotFixture(t, source, "untracked.txt", "untracked working copy\n")
	writeSnapshotFixture(t, source, ".env.local", "SECRET_ENV_SHOULD_NOT_APPEAR=1\n")
	writeSnapshotFixture(t, source, "credentials.json", "SECRET_FILE_SHOULD_NOT_APPEAR\n")
	writeSnapshotFixture(t, source, "config.txt", "api_key=abcdef0123456789\n")
	writeSnapshotFixture(t, source, "token.txt", "xai-abcdefghijklmnopqrstuvwxyz123456\n")
	if runtime.GOOS != "windows" {
		writeSnapshotFixture(t, source, "run.sh", "#!/bin/sh\necho safe\n")
		if err := os.Chmod(filepath.Join(source, "run.sh"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeSnapshotFixture(t, source, "ignored.txt", "IGNORED_CONTENT_SHOULD_NOT_APPEAR\n")
	paths := []string{".gitignore", "tracked.txt", "untracked.txt", ".env.local", "credentials.json", "config.txt", "token.txt", "ignored.txt"}
	if runtime.GOOS != "windows" {
		paths = append(paths, "run.sh")
		if err := os.Symlink(filepath.Join(source, "tracked.txt"), filepath.Join(source, "link.txt")); err != nil {
			t.Fatalf("create symlink fixture: %v", err)
		}
		paths = append(paths, "link.txt")
	}
	sourceBefore := captureSnapshotFixtureState(t, source, paths)

	statusArgs := []string{"status", "--porcelain=v2", "--branch", "-z", "--untracked-files=all", "--ignore-submodules=all"}
	statusBefore := runSnapshotFixtureGitOutput(t, source, statusArgs...)
	indexBefore, err := os.ReadFile(filepath.Join(source, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	headBefore := strings.TrimSpace(string(runSnapshotFixtureGitOutput(t, source, "rev-parse", "HEAD")))
	branchBefore := strings.TrimSpace(string(runSnapshotFixtureGitOutput(t, source, "branch", "--show-current")))
	dataRoot := filepath.Join(t.TempDir(), "private-data")
	manifest, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: dataRoot, MissionID: "mission_safe_1", OrganizationID: "org_test", ProjectID: "project_test",
	})
	if err != nil {
		t.Fatalf("CreateWorkspaceSnapshot: %v", err)
	}

	wantContents := map[string]string{
		".gitignore":    "ignored.txt\n",
		"tracked.txt":   "modified working copy\n",
		"untracked.txt": "untracked working copy\n",
	}
	if runtime.GOOS != "windows" {
		wantContents["run.sh"] = "#!/bin/sh\necho safe\n"
	}
	wantManifestPaths := map[string]bool{
		".gitignore": true, "tracked.txt": true, "untracked.txt": true,
		".env.local": true, "credentials.json": true, "config.txt": true, "token.txt": true,
	}
	if runtime.GOOS != "windows" {
		wantManifestPaths["run.sh"] = true
		wantManifestPaths["link.txt"] = true
	}
	if len(manifest.Files) != len(wantManifestPaths) {
		t.Fatalf("manifest files=%d, want exactly %d: %+v", len(manifest.Files), len(wantManifestPaths), manifest.Files)
	}
	seenManifestPaths := map[string]bool{}
	for _, entry := range manifest.Files {
		path := filepath.ToSlash(entry.Path)
		if !wantManifestPaths[path] || seenManifestPaths[path] {
			t.Fatalf("unexpected or duplicate manifest path %q", path)
		}
		seenManifestPaths[path] = true
		if !entry.Present {
			t.Errorf("fixture path %q is unexpectedly missing from source", path)
		}
		if entry.Source != "tracked" && entry.Source != "untracked" {
			t.Errorf("manifest source for %q = %q", path, entry.Source)
		}
	}
	for path := range wantManifestPaths {
		if !seenManifestPaths[path] {
			t.Errorf("manifest omitted expected source path %q", path)
		}
	}
	for _, path := range []string{".gitignore", "tracked.txt"} {
		entry, _ := findSnapshotManifestFile(t, manifest.Files, path)
		if entry.Source != "tracked" {
			t.Errorf("manifest source for tracked file %q = %q", path, entry.Source)
		}
	}
	for _, path := range []string{"untracked.txt", ".env.local", "credentials.json", "config.txt", "token.txt", "run.sh"} {
		if path == "run.sh" && runtime.GOOS == "windows" {
			continue
		}
		entry, _ := findSnapshotManifestFile(t, manifest.Files, path)
		if entry.Source != "untracked" {
			t.Errorf("manifest source for untracked file %q = %q", path, entry.Source)
		}
	}
	if runtime.GOOS != "windows" {
		entry, _ := findSnapshotManifestFile(t, manifest.Files, "link.txt")
		if entry.Source != "untracked" {
			t.Errorf("manifest source for symlink = %q", entry.Source)
		}
	}
	wantTotalBytes := int64(0)
	for path, want := range wantContents {
		wantTotalBytes += int64(len(want))
		entry, ok := findSnapshotManifestFile(t, manifest.Files, path)
		if !ok || !entry.Included || !entry.Present || entry.Kind != "file" || entry.Size != int64(len(want)) || entry.SHA256 != snapshotFixtureSHA256([]byte(want)) {
			t.Fatalf("manifest entry %q = %+v, present=%t", path, entry, ok)
		}
		data, err := os.ReadFile(filepath.Join(manifest.TreeRoot, filepath.FromSlash(path)))
		if err != nil || string(data) != want {
			t.Fatalf("snapshot %q content=%q err=%v, want %q", path, data, err, want)
		}
		info, err := os.Stat(filepath.Join(manifest.TreeRoot, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		wantMode := os.FileMode(0o600)
		if path == "run.sh" {
			wantMode = 0o700
		}
		if info.Mode().Perm() != wantMode {
			t.Errorf("snapshot file %q mode = %04o, want %04o", path, info.Mode().Perm(), wantMode)
		}
	}
	if manifest.CopiedFiles != len(wantContents) || manifest.TotalBytes != wantTotalBytes {
		t.Fatalf("copied=%d total=%d, want copied=%d total=%d", manifest.CopiedFiles, manifest.TotalBytes, len(wantContents), wantTotalBytes)
	}
	wantTreePaths := make([]string, 0, len(wantContents))
	for path := range wantContents {
		wantTreePaths = append(wantTreePaths, path)
	}
	sort.Strings(wantTreePaths)
	gotTreePaths := collectSnapshotTreeFiles(t, manifest.TreeRoot)
	if strings.Join(gotTreePaths, "\x00") != strings.Join(wantTreePaths, "\x00") {
		t.Fatalf("snapshot tree paths=%q, want exactly %q", gotTreePaths, wantTreePaths)
	}

	for _, path := range []string{".env.local", "credentials.json", "config.txt", "token.txt"} {
		entry, ok := findSnapshotManifestFile(t, manifest.Files, path)
		if !ok || entry.Included || entry.SkipReason == "" {
			t.Errorf("expected sensitive path %q to be explicitly omitted, entry=%+v present=%t", path, entry, ok)
		}
		assertSnapshotPathAbsent(t, manifest.TreeRoot, path)
	}
	for _, secret := range []string{"SECRET_ENV_SHOULD_NOT_APPEAR", "SECRET_FILE_SHOULD_NOT_APPEAR", "abcdef0123456789", "abcdefghijklmnopqrstuvwxyz123456", "IGNORED_CONTENT_SHOULD_NOT_APPEAR"} {
		if strings.Contains(string(manifestBytesForTest(t, manifest.SnapshotRoot)), secret) {
			t.Errorf("manifest leaked excluded content %q", secret)
		}
	}
	if _, ok := findSnapshotManifestFile(t, manifest.Files, "ignored.txt"); ok {
		t.Error("Git-ignored file unexpectedly appeared in manifest")
	}
	assertSnapshotPathAbsent(t, manifest.TreeRoot, "ignored.txt")
	if runtime.GOOS != "windows" {
		entry, ok := findSnapshotManifestFile(t, manifest.Files, "link.txt")
		if !ok || entry.Included || entry.Kind != "symlink" || entry.SkipReason != "symlink_not_copied" {
			t.Errorf("symlink policy = %+v, present=%t", entry, ok)
		}
		assertSnapshotPathAbsent(t, manifest.TreeRoot, "link.txt")
	}
	if _, err := os.Lstat(filepath.Join(manifest.TreeRoot, ".git")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("snapshot must not contain .git metadata, lstat error = %v", err)
	}

	wantStatus, err := splitSnapshotNULRecords(statusBefore)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.GitHead != headBefore || manifest.GitBranch != branchBefore || manifest.GitIndexSHA256 != snapshotFixtureSHA256(indexBefore) || strings.Join(manifest.GitStatus, "\x00") != strings.Join(wantStatus, "\x00") {
		t.Fatalf("manifest Git state differs from source: head=%q branch=%q index=%q status=%q", manifest.GitHead, manifest.GitBranch, manifest.GitIndexSHA256, manifest.GitStatus)
	}
	if len(manifest.IndexEntries) != 2 {
		t.Fatalf("index entries=%+v, want exactly .gitignore and tracked.txt", manifest.IndexEntries)
	}
	for _, entry := range manifest.IndexEntries {
		if entry.Stage != 0 || entry.Mode != 0o100644 || (entry.Path != ".gitignore" && entry.Path != "tracked.txt") {
			t.Errorf("unexpected index entry: %+v", entry)
		}
	}
	if sameOrWithin(source, manifest.SnapshotRoot) || sameOrWithin(manifest.SnapshotRoot, source) {
		t.Fatalf("snapshot is not isolated from source: source=%q snapshot=%q", source, manifest.SnapshotRoot)
	}
	manifestBytes := manifestBytesForTest(t, manifest.SnapshotRoot)
	var persisted WorkspaceSnapshotManifest
	if err := json.Unmarshal(manifestBytes, &persisted); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
	wantPersisted := manifest
	wantPersisted.SnapshotRoot = ""
	wantPersisted.TreeRoot = ""
	if !reflect.DeepEqual(persisted, wantPersisted) {
		t.Fatalf("persisted manifest differs from returned manifest:\npersisted=%+v\nwant=%+v", persisted, wantPersisted)
	}

	statusAfter := runSnapshotFixtureGitOutput(t, source, statusArgs...)
	indexAfter, err := os.ReadFile(filepath.Join(source, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	headAfter := strings.TrimSpace(string(runSnapshotFixtureGitOutput(t, source, "rev-parse", "HEAD")))
	branchAfter := strings.TrimSpace(string(runSnapshotFixtureGitOutput(t, source, "branch", "--show-current")))
	if string(statusAfter) != string(statusBefore) || string(indexAfter) != string(indexBefore) || headAfter != headBefore || branchAfter != branchBefore {
		t.Fatal("snapshot mutated source Git status, index, HEAD, or branch")
	}
	assertSnapshotFixtureStateEqual(t, source, sourceBefore)
}

func TestCreateWorkspaceSnapshotRejectsInvalidLimitsBeforeCreatingDataRoot(t *testing.T) {
	source := initWorkspaceSnapshotGit(t)
	dataRoot := filepath.Join(t.TempDir(), "should-not-exist")
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: dataRoot, MissionID: "mission_limits_1", MaxFiles: -1,
	})
	if err == nil {
		t.Fatal("expected invalid snapshot limits to fail")
	}
	if _, statErr := os.Lstat(dataRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("invalid configuration created the data root: %v", statErr)
	}
}

func TestCreateWorkspaceSnapshotRejectsNonGitSourceBeforeCreatingDataRoot(t *testing.T) {
	source := t.TempDir()
	dataRoot := filepath.Join(t.TempDir(), "must-not-be-created")
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: dataRoot, MissionID: "mission_non_git_1",
	})
	if err == nil {
		t.Fatal("expected non-Git source to be rejected")
	}
	if _, statErr := os.Lstat(dataRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("rejected non-Git source created DataRoot: %v", statErr)
	}
}

func TestCreateWorkspaceSnapshotRejectsConfiguredGitFilterWithoutExecutingIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("filter execution probe uses a POSIX shell script")
	}
	source := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, source, "tracked.txt", "base\n")
	runSnapshotFixtureGit(t, source, "add", "tracked.txt")
	runSnapshotFixtureGit(t, source, "commit", "-m", "initial")
	filterScript := filepath.Join(t.TempDir(), "filter.sh")
	sentinel := filepath.Join(t.TempDir(), "filter-was-executed")
	script := "#!/bin/sh\ntouch " + sentinel + "\ncat\n"
	if err := os.WriteFile(filterScript, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	runSnapshotFixtureGit(t, source, "config", "filter.probe.clean", filterScript)
	writeSnapshotFixture(t, source, ".gitattributes", "*.txt filter=probe\n")
	runSnapshotFixtureGit(t, source, "add", ".gitattributes")
	runSnapshotFixtureGit(t, source, "commit", "-m", "configure filter attribute")
	if err := os.Remove(sentinel); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	writeSnapshotFixture(t, source, "tracked.txt", "modified\n")
	dataRoot := filepath.Join(t.TempDir(), "must-not-be-created")
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: dataRoot, MissionID: "mission_filter_1",
	})
	if err == nil || !strings.Contains(err.Error(), "external filter") {
		t.Fatalf("snapshot error = %v, want external Git filter rejection", err)
	}
	if _, statErr := os.Lstat(sentinel); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("repository-configured filter executed: %v", statErr)
	}
	if _, statErr := os.Lstat(dataRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("rejected repository created DataRoot: %v", statErr)
	}
	configPath := filepath.Join(source, ".git", "config")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	configText := strings.Replace(string(config), `[filter "probe"]`, `[filter.probe]`, 1)
	if configText == string(config) {
		t.Fatalf("test fixture did not contain Git filter subsection: %s", config)
	}
	if err := os.WriteFile(configPath, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	dotFormDataRoot := filepath.Join(t.TempDir(), "must-not-be-created")
	_, err = CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{SourceRoot: source, DataRoot: dotFormDataRoot, MissionID: "mission_filter_dot_1"})
	if err == nil || !strings.Contains(err.Error(), "external filter") {
		t.Fatalf("dot-form snapshot filter error=%v, want fail-closed rejection", err)
	}
	if _, statErr := os.Lstat(sentinel); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("dot-form repository filter command executed: %v", statErr)
	}
	if _, statErr := os.Lstat(dotFormDataRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("dot-form rejected repository created DataRoot: %v", statErr)
	}
}

func TestCreateWorkspaceSnapshotRejectsGitMetadataPointersBeforeAnyGitCommand(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "alternates", path: filepath.Join(".git", "objects", "info", "alternates")},
		{name: "http alternates", path: filepath.Join(".git", "objects", "info", "http-alternates")},
		{name: "commondir", path: filepath.Join(".git", "commondir")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := initWorkspaceSnapshotGit(t)
			path := filepath.Join(source, tc.path)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(t.TempDir()+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			commands := 0
			workspaceSnapshotGitCommandHook = func(string, []string) { commands++ }
			defer func() { workspaceSnapshotGitCommandHook = nil }()
			dataRoot := filepath.Join(t.TempDir(), "must-not-be-created")
			_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{SourceRoot: source, DataRoot: dataRoot, MissionID: "mis_git_metadata"})
			if err == nil || !strings.Contains(err.Error(), "Git metadata pointer") {
				t.Fatalf("unsafe Git metadata error=%v, want pointer rejection", err)
			}
			if commands != 0 {
				t.Fatalf("snapshot invoked Git %d times before rejecting metadata pointer", commands)
			}
			if _, statErr := os.Lstat(dataRoot); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("rejected Git metadata created DataRoot: %v", statErr)
			}
		})
	}
}

func TestCreateWorkspaceSnapshotRejectsGitMetadataSymlinksBeforeAnyGitCommand(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Git metadata symlink regression requires Linux symlink support")
	}
	source := initWorkspaceSnapshotGit(t)
	head := filepath.Join(source, ".git", "HEAD")
	if err := os.Remove(head); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "outside-head"), head); err != nil {
		t.Fatal(err)
	}
	commands := 0
	workspaceSnapshotGitCommandHook = func(string, []string) { commands++ }
	defer func() { workspaceSnapshotGitCommandHook = nil }()
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{SourceRoot: source, DataRoot: filepath.Join(t.TempDir(), "must-not-be-created"), MissionID: "mis_git_symlink"})
	if err == nil || !strings.Contains(err.Error(), "not a regular file or directory") {
		t.Fatalf("symlinked Git metadata error=%v, want fail-closed rejection", err)
	}
	if commands != 0 {
		t.Fatalf("snapshot invoked Git %d times before rejecting Git metadata symlink", commands)
	}
}

func TestCreateWorkspaceSnapshotRejectsSpecialGitMetadataBeforeAnyGitCommand(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("special Git metadata regression requires Unix sockets")
	}
	source := initWorkspaceSnapshotGit(t)
	socketPath := filepath.Join(source, ".git", "s")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	commands := 0
	workspaceSnapshotGitCommandHook = func(string, []string) { commands++ }
	defer func() { workspaceSnapshotGitCommandHook = nil }()
	_, err = CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{SourceRoot: source, DataRoot: filepath.Join(t.TempDir(), "must-not-be-created"), MissionID: "mis_git_special"})
	if err == nil || !strings.Contains(err.Error(), "not a regular file or directory") {
		t.Fatalf("special Git metadata error=%v, want fail-closed rejection", err)
	}
	if commands != 0 {
		t.Fatalf("snapshot invoked Git %d times before rejecting special Git metadata", commands)
	}
}

func TestCreateWorkspaceSnapshotRejectsAliasesAndWorktreeRedirectBeforeAnyGitCommand(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	marker := filepath.Join(t.TempDir(), "git-alias-executed")
	outside := t.TempDir()
	for _, tc := range []struct {
		name   string
		config string
		want   string
	}{
		{name: "alias", config: "[alias]\n\tstatus = !touch " + marker + "\n", want: "Git aliases"},
		{name: "core worktree", config: "[core]\n\tworktree = " + outside + "\n", want: "core.worktree"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := initWorkspaceSnapshotGit(t)
			if err := os.WriteFile(filepath.Join(source, ".git", "config"), []byte("[core]\n\trepositoryformatversion = 0\n"+tc.config), 0o600); err != nil {
				t.Fatal(err)
			}
			commands := 0
			workspaceSnapshotGitCommandHook = func(string, []string) { commands++ }
			defer func() { workspaceSnapshotGitCommandHook = nil }()
			dataRoot := filepath.Join(t.TempDir(), "must-not-be-created")
			_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{SourceRoot: source, DataRoot: dataRoot, MissionID: "mis_git_config"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unsafe Git config error=%v, want rejection containing %q", err, tc.want)
			}
			if commands != 0 {
				t.Fatalf("snapshot invoked Git %d times before rejecting config", commands)
			}
			if _, statErr := os.Lstat(dataRoot); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("rejected Git config created DataRoot: %v", statErr)
			}
		})
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("repository-local alias command executed: %v", err)
	}
}

func TestCreateWorkspaceSnapshotRejectsIncludesBeforeAnyGitCommand(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	for _, tc := range []struct {
		name           string
		config         string
		worktreeConfig string
	}{
		{name: "local config", config: "[include]\n\tpath = /tmp/attacker-config\n"},
		{name: "worktree config", config: "[extensions]\n\tworktreeConfig = true\n", worktreeConfig: "[includeIf \"gitdir:/tmp/\"]\n\tpath = /tmp/attacker-worktree-config\n"},
		{name: "worktree config yes", config: "[extensions]\n\tworktreeConfig = yes\n", worktreeConfig: "[include]\n\tpath = /tmp/attacker-worktree-config\n"},
		{name: "worktree config on", config: "[extensions]\n\tworktreeConfig = on\n", worktreeConfig: "[include]\n\tpath = /tmp/attacker-worktree-config\n"},
		{name: "worktree config one", config: "[extensions]\n\tworktreeConfig = 1\n", worktreeConfig: "[include]\n\tpath = /tmp/attacker-worktree-config\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := initWorkspaceSnapshotGit(t)
			if err := os.WriteFile(filepath.Join(source, ".git", "config"), []byte(tc.config), 0o600); err != nil {
				t.Fatal(err)
			}
			if tc.worktreeConfig != "" {
				if err := os.WriteFile(filepath.Join(source, ".git", "config.worktree"), []byte(tc.worktreeConfig), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			commands := 0
			workspaceSnapshotGitCommandHook = func(string, []string) { commands++ }
			defer func() { workspaceSnapshotGitCommandHook = nil }()
			_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{SourceRoot: source, DataRoot: filepath.Join(t.TempDir(), "snapshot"), MissionID: "mis_git_include"})
			if err == nil || !strings.Contains(err.Error(), "include/includeIf") {
				t.Fatalf("unsafe config error=%v, want include rejection", err)
			}
			if commands != 0 {
				t.Fatalf("snapshot invoked Git %d times before rejecting local includes", commands)
			}
		})
	}
}

func TestValidateWorkspaceSnapshotPathRejectsNonPortableComponents(t *testing.T) {
	for _, path := range []string{"CON", "nul.txt", "LPT1.log", "COM9", "has:ads", "bad.", "trailing ", "question?mark", "control\x01char"} {
		t.Run(strings.ReplaceAll(path, "\x01", "control"), func(t *testing.T) {
			if err := validateWorkspaceSnapshotPath(path, workspaceSnapshotDefaultDepth); err == nil {
				t.Fatalf("expected non-portable path %q to fail", path)
			}
		})
	}
	for _, path := range []string{"src/main.go", ".config/settings.json", "COM10/ordinary.txt", "readme.md"} {
		t.Run("accept_"+strings.ReplaceAll(path, "/", "_"), func(t *testing.T) {
			if err := validateWorkspaceSnapshotPath(path, workspaceSnapshotDefaultDepth); err != nil {
				t.Fatalf("portable path %q rejected: %v", path, err)
			}
		})
	}
}

func TestCreateWorkspaceSnapshotRecordsButDoesNotCopySubmodule(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	submodule := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, submodule, "private.txt", "submodule content must not be copied\n")
	runSnapshotFixtureGit(t, submodule, "add", "private.txt")
	runSnapshotFixtureGit(t, submodule, "commit", "-m", "submodule base")

	source := initWorkspaceSnapshotGit(t)
	runSnapshotFixtureGit(t, source, "-c", "protocol.file.allow=always", "submodule", "add", "-q", submodule, "vendor/local")
	runSnapshotFixtureGit(t, source, "commit", "-m", "submodule reference")
	manifest, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: filepath.Join(t.TempDir(), "private-data"), MissionID: "mission_submodule_1",
	})
	if err != nil {
		t.Fatalf("CreateWorkspaceSnapshot with submodule: %v", err)
	}
	entry, ok := findSnapshotManifestFile(t, manifest.Files, "vendor/local")
	if !ok || entry.Kind != "submodule" || entry.Included || entry.SkipReason != "submodule_not_copied" {
		t.Fatalf("submodule manifest entry=%+v found=%t", entry, ok)
	}
	assertSnapshotPathAbsent(t, manifest.TreeRoot, "vendor/local")
	if _, ok := findSnapshotManifestFile(t, manifest.Files, "vendor/local/private.txt"); ok {
		t.Fatal("submodule contents unexpectedly appeared as independent snapshot files")
	}
}

func TestCreateWorkspaceSnapshotRejectsParentSymlinkEscape(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	source := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, source, "linked/private.txt", "tracked base\n")
	runSnapshotFixtureGit(t, source, "add", "linked/private.txt")
	runSnapshotFixtureGit(t, source, "commit", "-m", "nested tracked file")

	outside := t.TempDir()
	writeSnapshotFixture(t, outside, "private.txt", "OUTSIDE_MUST_NOT_BE_READ\n")
	if err := os.Remove(filepath.Join(source, "linked", "private.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(source, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(source, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	dataRoot := filepath.Join(t.TempDir(), "private-data")
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: dataRoot, MissionID: "mission_parent_symlink_1",
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink-parent rejection, got %v", err)
	}
	outsideAfter, readErr := os.ReadFile(filepath.Join(outside, "private.txt"))
	if readErr != nil || string(outsideAfter) != "OUTSIDE_MUST_NOT_BE_READ\n" {
		t.Fatalf("outside target changed/read unexpectedly: %q err=%v", outsideAfter, readErr)
	}
	assertEmptySnapshotMissionDirectory(t, dataRoot, "mission_parent_symlink_1")
}

func TestCreateWorkspaceSnapshotRejectsNestedDataRootWithoutMutatingSource(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	source := initWorkspaceSnapshotGit(t)
	dataRoot := filepath.Join(source, "runtime-data", "snapshots")
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: dataRoot, MissionID: "mission_nested_root_1",
	})
	if err == nil || !strings.Contains(err.Error(), "must be separate") {
		t.Fatalf("nested data-root error = %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(source, "runtime-data")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("rejected nested DataRoot mutated source workspace: %v", statErr)
	}
}

func TestVerifyWorkspaceSnapshotRootIdentityRejectsReplacedSource(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	parent := t.TempDir()
	rootPath := filepath.Join(parent, "repo")
	if err := os.Mkdir(rootPath, 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := os.Rename(rootPath, filepath.Join(parent, "repo-original")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(rootPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := verifyWorkspaceSnapshotRootIdentity(rootPath, root); !errors.Is(err, ErrWorkspaceSnapshotChanged) {
		t.Fatalf("replacement identity error = %v, want ErrWorkspaceSnapshotChanged", err)
	}
}

func TestCreateWorkspaceSnapshotRejectsReplacedAuthorizedProjectRoot(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	source := initWorkspaceSnapshotGit(t)
	projectRoot := filepath.Join(source, "apps", "approved")
	if err := os.MkdirAll(projectRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	approvedInfo, err := os.Lstat(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(projectRoot, projectRoot+".original"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, ExpectedProjectRoot: projectRoot, ExpectedProjectRootInfo: approvedInfo,
		DataRoot: filepath.Join(t.TempDir(), "snapshots"), MissionID: "mission_project_replacement_1",
		OrganizationID: "org_project_replacement", ProjectID: "proj_project_replacement",
	})
	if !errors.Is(err, ErrWorkspaceSnapshotChanged) {
		t.Fatalf("replaced authorized project root error = %v, want ErrWorkspaceSnapshotChanged", err)
	}
}

func TestSweepOrphanedWorkspaceSnapshotsPreservesReferencesAndTenantScope(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	source := initWorkspaceSnapshotGit(t)
	dataRoot := t.TempDir()
	create := func(missionID, organizationID string) WorkspaceSnapshotManifest {
		t.Helper()
		manifest, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{SourceRoot: source, DataRoot: filepath.Join(dataRoot, ".agent-workspace-snapshots"), MissionID: missionID, OrganizationID: organizationID, ProjectID: "proj_" + missionID})
		if err != nil {
			t.Fatal(err)
		}
		return manifest
	}
	kept := create("mis_keep_1", "org_keep")
	orphan := create("mis_orphan_1", "org_keep")
	terminal := create("mis_terminal_1", "org_keep")
	active := create("mis_active_1", "org_keep")
	otherTenant := create("mis_other_1", "org_other")
	now := time.Now().UTC()
	old := now.Add(-workspaceSnapshotRetention - time.Minute)
	for _, manifest := range []WorkspaceSnapshotManifest{orphan, terminal, active, otherTenant} {
		if err := os.Chtimes(manifest.SnapshotRoot, old, old); err != nil {
			t.Fatal(err)
		}
	}
	mission := Mission{ID: kept.MissionID, OrganizationID: kept.OrganizationID, ProjectID: kept.ProjectID, WorkspaceIsolated: true, WorkspaceSnapshotID: kept.SnapshotID}
	terminalMission := Mission{ID: terminal.MissionID, OrganizationID: terminal.OrganizationID, ProjectID: terminal.ProjectID, WorkspaceIsolated: true, WorkspaceSnapshotID: terminal.SnapshotID, State: MissionCompleted, UpdatedAt: old}
	activeMission := Mission{ID: active.MissionID, OrganizationID: active.OrganizationID, ProjectID: active.ProjectID, WorkspaceIsolated: true, WorkspaceSnapshotID: active.SnapshotID, State: MissionRunning, UpdatedAt: old}
	if err := sweepOrphanedWorkspaceSnapshots(dataRoot, "org_keep", []Mission{mission, terminalMission, activeMission}, now); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		path string
		want bool
	}{{"referenced nonterminal", kept.SnapshotRoot, true}, {"orphan same tenant", orphan.SnapshotRoot, false}, {"old terminal retention", terminal.SnapshotRoot, false}, {"old active reference", active.SnapshotRoot, true}, {"other tenant scope", otherTenant.SnapshotRoot, true}} {
		_, err := os.Lstat(test.path)
		if test.want && err != nil {
			t.Errorf("%s snapshot was removed: %v", test.name, err)
		}
		if !test.want && !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s snapshot still exists or stat failed: %v", test.name, err)
		}
	}
}

func TestCreateWorkspaceSnapshotEnforcesFileLimitAndRemovesPartialSnapshot(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	source := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, source, "a-small.txt", "ok\n")
	writeSnapshotFixture(t, source, "z-large.txt", "more than four bytes")
	runSnapshotFixtureGit(t, source, "add", "a-small.txt", "z-large.txt")
	dataRoot := filepath.Join(t.TempDir(), "private-data")
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: dataRoot, MissionID: "mission_limit_2", MaxTotalBytes: 3,
	})
	if !errors.Is(err, ErrWorkspaceSnapshotLimit) {
		t.Fatalf("limit error = %v, want ErrWorkspaceSnapshotLimit", err)
	}
	assertEmptySnapshotMissionDirectory(t, dataRoot, "mission_limit_2")
}

func initWorkspaceSnapshotGit(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runSnapshotFixtureGit(t, root, "init", "-q")
	runSnapshotFixtureGit(t, root, "config", "user.name", "Ollama Full snapshot test")
	runSnapshotFixtureGit(t, root, "config", "user.email", "snapshot-test@example.invalid")
	return root
}

func writeSnapshotFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func captureSnapshotFixtureState(t *testing.T, root string, paths []string) map[string]snapshotFixtureState {
	t.Helper()
	result := make(map[string]snapshotFixtureState, len(paths))
	for _, relative := range paths {
		path := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		state := snapshotFixtureState{Mode: info.Mode()}
		if info.Mode()&os.ModeSymlink != 0 {
			state.Link, err = os.Readlink(path)
		} else if info.Mode().IsRegular() {
			var data []byte
			data, err = os.ReadFile(path)
			state.Content = string(data)
		}
		if err != nil {
			t.Fatal(err)
		}
		result[relative] = state
	}
	return result
}

func assertSnapshotFixtureStateEqual(t *testing.T, root string, before map[string]snapshotFixtureState) {
	t.Helper()
	paths := make([]string, 0, len(before))
	for path := range before {
		paths = append(paths, path)
	}
	after := captureSnapshotFixtureState(t, root, paths)
	for path, expected := range before {
		if got := after[path]; got != expected {
			t.Errorf("source fixture %q changed: before=%+v after=%+v", path, expected, got)
		}
	}
}

func findSnapshotManifestFile(t *testing.T, files []WorkspaceSnapshotFile, path string) (WorkspaceSnapshotFile, bool) {
	t.Helper()
	var found WorkspaceSnapshotFile
	count := 0
	for _, file := range files {
		if filepath.ToSlash(file.Path) == path {
			found = file
			count++
		}
	}
	if count > 1 {
		t.Fatalf("manifest contains duplicate path %q", path)
	}
	return found, count == 1
}

func collectSnapshotTreeFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

func assertSnapshotPathAbsent(t *testing.T, root, relative string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative))); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("excluded path %q exists in snapshot tree (lstat error: %v)", relative, err)
	}
}

func assertEmptySnapshotMissionDirectory(t *testing.T, dataRoot, missionID string) {
	t.Helper()
	snapshotsRoot := filepath.Join(dataRoot, "snapshots")
	tenants, err := os.ReadDir(snapshotsRoot)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, tenant := range tenants {
		missionRoot := filepath.Join(snapshotsRoot, tenant.Name(), missionID)
		entries, readErr := os.ReadDir(missionRoot)
		if errors.Is(readErr, os.ErrNotExist) {
			continue
		}
		if readErr != nil {
			t.Fatal(readErr)
		}
		if len(entries) != 0 {
			t.Fatalf("snapshot mission directory contains partial artifacts: %v", entries)
		}
	}
}

func manifestBytesForTest(t *testing.T, snapshotRoot string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(snapshotRoot, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func runSnapshotFixtureGit(t *testing.T, root string, args ...string) {
	t.Helper()
	if _, err := runSnapshotFixtureGitCommand(root, args...); err != nil {
		t.Fatal(err)
	}
}

func runSnapshotFixtureGitOutput(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	output, err := runSnapshotFixtureGitCommand(root, args...)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func runSnapshotFixtureGitCommand(root string, args ...string) ([]byte, error) {
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=", "GIT_TERMINAL_PROMPT=0")
	return command.CombinedOutput()
}

func snapshotFixtureSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func TestCopyWorkspaceSnapshotFileRemainsAnchoredAfterDestinationSwap(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	source := t.TempDir()
	const content = "source content remains inside the pinned tree\n"
	if err := os.WriteFile(filepath.Join(source, "src.txt"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceRoot, err := os.OpenRoot(source)
	if err != nil {
		t.Fatal(err)
	}
	defer sourceRoot.Close()

	dataRoot := t.TempDir()
	treePath := filepath.Join(dataRoot, "tree")
	if err := os.Mkdir(treePath, 0o700); err != nil {
		t.Fatal(err)
	}
	treeRoot, err := os.OpenRoot(treePath)
	if err != nil {
		t.Fatal(err)
	}
	defer treeRoot.Close()
	detachedTree := filepath.Join(dataRoot, "detached-tree")
	if err := os.Rename(treePath, detachedTree); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	if err := os.Symlink(external, treePath); err != nil {
		t.Fatal(err)
	}

	limits := workspaceSnapshotLimits{maxFiles: 1, maxDepth: 64, maxFileBytes: 1024, maxTotal: 1024}
	entry := WorkspaceSnapshotFile{Path: "src.txt", Source: "tracked"}
	var total int64
	if err := copyWorkspaceSnapshotFile(context.Background(), source, sourceRoot, treeRoot, treePath, "src.txt", 0o100644, limits, &total, &entry); err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(filepath.Join(detachedTree, "src.txt"))
	if err != nil || string(copied) != content {
		t.Fatalf("copy did not stay within pinned original tree: content=%q err=%v", copied, err)
	}
	if _, err := os.Lstat(filepath.Join(external, "src.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("destination replacement received snapshot data: err=%v", err)
	}
}

func TestPrepareWorkspaceSnapshotHandoffRejectsReplacedSnapshotPath(t *testing.T) {
	requireDescriptorBoundWorkspaceIsolation(t)
	source := initWorkspaceSnapshotGit(t)
	dataRoot := filepath.Join(t.TempDir(), "private-data")
	manifest, handles, err := createWorkspaceSnapshotWithHandles(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: dataRoot, MissionID: "mis_handoff_replace", OrganizationID: "org_handoff", ProjectID: "proj_handoff",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer handles.Close()
	if runtime.GOOS == "windows" {
		t.Skip("snapshot replacement symlink fixture requires Unix symlink privileges")
	}
	moved := manifest.SnapshotRoot + ".moved"
	if err := os.Rename(manifest.SnapshotRoot, moved); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	if err := os.Symlink(external, manifest.SnapshotRoot); err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareWorkspaceSnapshotHandoff(manifest, handles, "."); !errors.Is(err, ErrWorkspaceSnapshotChanged) {
		t.Fatalf("replaced snapshot path error = %v, want ErrWorkspaceSnapshotChanged", err)
	}
	if _, err := os.Lstat(filepath.Join(external, "tree")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("handoff wrote through replacement symlink: %v", err)
	}
}

func TestSweepOrphanedWorkspaceSnapshotsRejectsReplacedStorageParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("snapshot storage symlink fixture requires Unix symlink privileges")
	}
	source := initWorkspaceSnapshotGit(t)
	dataRoot := filepath.Join(t.TempDir(), "private-data")
	snapshotDataRoot := filepath.Join(dataRoot, ".agent-workspace-snapshots")
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: snapshotDataRoot, MissionID: "mis_sweep_replace", OrganizationID: "org_sweep", ProjectID: "proj_sweep",
	})
	if err != nil {
		t.Fatal(err)
	}
	storagePath := snapshotDataRoot
	movedStorage := storagePath + ".moved"
	if err := os.Rename(storagePath, movedStorage); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	if err := os.Symlink(external, storagePath); err != nil {
		t.Fatal(err)
	}
	err = sweepOrphanedWorkspaceSnapshots(dataRoot, "org_sweep", nil, time.Now())
	if err == nil {
		t.Fatal("sweeper accepted a symlink replacement in its storage parent")
	}
	if _, err := os.Stat(filepath.Join(movedStorage, "snapshots")); err != nil {
		t.Fatalf("sweeper damaged pinned original storage: %v", err)
	}
	entries, err := os.ReadDir(external)
	if err != nil || len(entries) != 0 {
		t.Fatalf("sweeper touched external symlink target: entries=%v err=%v", entries, err)
	}
}

func TestCreateWorkspaceSnapshotGitUsesPinnedSourceAfterPathReplacement(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("descriptor-bound Git cwd regression is Linux-specific")
	}
	source := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, source, "original.txt", "pinned original repository\n")
	runSnapshotFixtureGit(t, source, "add", "original.txt")
	runSnapshotFixtureGit(t, source, "commit", "-m", "original repository")
	decoy := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, decoy, "decoy.txt", "replacement repository must not be read\n")
	runSnapshotFixtureGit(t, decoy, "add", "decoy.txt")
	runSnapshotFixtureGit(t, decoy, "commit", "-m", "replacement repository")

	movedOriginal := source + ".pinned"
	swapped := false
	gitCommands := 0
	workspaceSnapshotGitCommandHook = func(root string, args []string) {
		gitCommands++
		if swapped || gitCommands < 1 || len(args) == 0 || args[0] != "rev-parse" {
			return
		}
		if err := os.Rename(source, movedOriginal); err != nil {
			t.Fatalf("move opened source path: %v", err)
		}
		if err := os.Rename(decoy, source); err != nil {
			t.Fatalf("install replacement source path: %v", err)
		}
		swapped = true
	}
	defer func() { workspaceSnapshotGitCommandHook = nil }()
	dataRoot := filepath.Join(t.TempDir(), "snapshot-data")
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: dataRoot, MissionID: "mis_source_swap", OrganizationID: "org_source_swap", ProjectID: "proj_source_swap",
	})
	if !swapped {
		t.Fatal("Git command hook did not replace the source pathname")
	}
	if !errors.Is(err, ErrWorkspaceSnapshotChanged) {
		t.Fatalf("snapshot after source replacement error = %v, want ErrWorkspaceSnapshotChanged", err)
	}
	if _, err := os.Stat(filepath.Join(movedOriginal, "original.txt")); err != nil {
		t.Fatalf("pinned original repository was not preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(source, "decoy.txt")); err != nil {
		t.Fatalf("replacement repository fixture is missing: %v", err)
	}
	if _, err := os.Stat(dataRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed pre-copy snapshot created a data root: %v", err)
	}
}

func TestCreateWorkspaceSnapshotRejectsGitMetadataReplacementBetweenCommands(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("descriptor-bound Git metadata replacement regression is Linux-specific")
	}
	source := initWorkspaceSnapshotGit(t)
	writeSnapshotFixture(t, source, "tracked.txt", "committed content\n")
	runSnapshotFixtureGit(t, source, "add", "tracked.txt")
	runSnapshotFixtureGit(t, source, "commit", "-m", "metadata replacement base")
	externalHead := filepath.Join(t.TempDir(), "HEAD")
	if err := os.WriteFile(externalHead, []byte("ref: refs/heads/main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	swapped := false
	workspaceSnapshotGitCommandHook = func(_ string, args []string) {
		if swapped || len(args) == 0 || args[0] != "rev-parse" {
			return
		}
		head := filepath.Join(source, ".git", "HEAD")
		if err := os.Remove(head); err != nil {
			t.Fatalf("remove original Git HEAD: %v", err)
		}
		if err := os.Symlink(externalHead, head); err != nil {
			t.Fatalf("replace Git HEAD with symlink: %v", err)
		}
		swapped = true
	}
	defer func() { workspaceSnapshotGitCommandHook = nil }()
	dataRoot := filepath.Join(t.TempDir(), "snapshot-data")
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{
		SourceRoot: source, DataRoot: dataRoot, MissionID: "mis_git_meta_swap", OrganizationID: "org_git_meta_swap", ProjectID: "proj_git_meta_swap",
	})
	if !swapped {
		t.Fatal("Git command hook did not replace repository metadata")
	}
	if err == nil || !strings.Contains(err.Error(), "Git metadata changed or became unsafe") {
		t.Fatalf("snapshot after Git metadata replacement error = %v, want fail-closed metadata validation", err)
	}
	if _, err := os.Stat(dataRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed pre-copy snapshot created a data root: %v", err)
	}
}

func TestWorkspaceSnapshotRepositoryDiscoveryPinsAncestors(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("descriptor-parent discovery regression requires Linux")
	}
	repository := initWorkspaceSnapshotGit(t)
	source := filepath.Join(repository, "nested")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	movedRepository := repository + "-moved"
	commands := 0
	workspaceSnapshotGitCommandHook = func(string, []string) { commands++ }
	defer func() { workspaceSnapshotGitCommandHook = nil }()
	workspaceSnapshotRepositoryDiscoveryHook = func(openedPath string) {
		if openedPath != source {
			t.Errorf("discovery opened %q, want %q", openedPath, source)
			return
		}
		if err := os.Rename(repository, movedRepository); err != nil {
			t.Errorf("rename pinned repository ancestor: %v", err)
			return
		}
		if err := os.Mkdir(repository, 0o700); err != nil {
			t.Errorf("replace repository ancestor pathname: %v", err)
		}
	}
	defer func() { workspaceSnapshotRepositoryDiscoveryHook = nil }()
	dataRoot := filepath.Join(t.TempDir(), "must-not-be-created")
	_, err := CreateWorkspaceSnapshot(context.Background(), WorkspaceSnapshotRequest{SourceRoot: source, DataRoot: dataRoot, MissionID: "mis_parent_swap"})
	if !errors.Is(err, ErrWorkspaceSnapshotChanged) {
		t.Fatalf("ancestor replacement error=%v, want %v", err, ErrWorkspaceSnapshotChanged)
	}
	if commands != 0 {
		t.Fatalf("snapshot invoked Git %d times after ancestor replacement", commands)
	}
	if _, statErr := os.Lstat(dataRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed ancestor discovery created DataRoot: %v", statErr)
	}
}

func TestCreateWorkspaceSnapshotDirectoryCleansCreatedParentsOnLeafFailure(t *testing.T) {
	dataRoot := t.TempDir()
	root, err := os.OpenRoot(dataRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	tenantKey := "0123456789abcdef01234567"
	_, _, _, _, err = createWorkspaceSnapshotDirectory(root, dataRoot, tenantKey, "mis_cleanup_test", "invalid/snapshot")
	if err == nil {
		t.Fatal("invalid snapshot leaf unexpectedly succeeded")
	}
	if _, err := os.Lstat(filepath.Join(dataRoot, "snapshots")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed snapshot left newly created parent directories behind: %v", err)
	}
}

func TestCopyWorkspaceSnapshotRejectsSameSizeSameMtimeMutation(t *testing.T) {
	sourcePath := t.TempDir()
	content := []byte("original bytes with fixed width")
	filePath := filepath.Join(sourcePath, "src.txt")
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	originalInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceRoot, err := os.OpenRoot(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer sourceRoot.Close()
	treePath := t.TempDir()
	treeRoot, err := os.OpenRoot(treePath)
	if err != nil {
		t.Fatal(err)
	}
	defer treeRoot.Close()
	oldHook := workspaceSnapshotAfterFileReadHook
	workspaceSnapshotAfterFileReadHook = func(path string) {
		workspaceSnapshotAfterFileReadHook = nil
		changed := append([]byte(nil), content...)
		changed[0] = 'M'
		if err := os.WriteFile(path, changed, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, originalInfo.ModTime(), originalInfo.ModTime()); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { workspaceSnapshotAfterFileReadHook = oldHook }()
	limits := workspaceSnapshotLimits{maxFiles: 1, maxDepth: 64, maxFileBytes: 1024, maxTotal: 1024}
	entry := WorkspaceSnapshotFile{Path: "src.txt", Source: "tracked"}
	var total int64
	err = copyWorkspaceSnapshotFile(context.Background(), sourcePath, sourceRoot, treeRoot, treePath, "src.txt", 0o100600, limits, &total, &entry)
	if !errors.Is(err, ErrWorkspaceSnapshotChanged) {
		t.Fatalf("copy error=%v, want ErrWorkspaceSnapshotChanged", err)
	}
	if _, err := os.Stat(filepath.Join(treePath, "src.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("mutated source was committed into snapshot: stat err=%v", err)
	}
}
