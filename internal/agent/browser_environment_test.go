package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var errNoBinary = errors.New("not found")

func TestBrowserProcessEnvironmentDoesNotInheritSecrets(t *testing.T) {
	t.Setenv("PATH", "safe-path")
	t.Setenv("OLLAMA_AGENT_BROWSER_EXECUTABLE", "chromium")
	t.Setenv("OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE", "1")
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "secret-credential")
	t.Setenv("AWS_ACCESS_KEY_ID", "secret-cloud-key")
	t.Setenv("HTTP_PROXY", "http://user:password@proxy.example")

	workspace := filepath.Join(os.TempDir(), "workspace")
	got := browserProcessEnvironment(workspace)
	joined := strings.Join(got, "\n")
	for _, secret := range []string{"secret-credential", "secret-cloud-key", "user:password", "OLLAMA_AGENT_CREDENTIAL_KEY", "AWS_ACCESS_KEY_ID", "HTTP_PROXY="} {
		if strings.Contains(joined, secret) {
			t.Fatalf("browser process environment contains forbidden value %q: %q", secret, joined)
		}
	}
	if !strings.Contains(joined, "PATH=safe-path") {
		t.Fatalf("browser process environment omitted PATH: %q", joined)
	}
	if !strings.Contains(joined, "OLLAMA_AGENT_BROWSER_EXECUTABLE=chromium") {
		t.Fatalf("browser process environment omitted configured executable: %q", joined)
	}
	if !strings.Contains(joined, "OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE=1") {
		t.Fatalf("browser process environment omitted loopback test setting: %q", joined)
	}
	if !strings.Contains(joined, "OLLAMA_AGENT_BROWSER_ROOT="+filepath.Join(workspace, ".browser")) {
		t.Fatalf("browser process environment omitted browser root: %q", joined)
	}
}

func TestCheckBrowserEnvironmentNoPython(t *testing.T) {
	lookPath := func(string) (string, error) { return "", errNoBinary }
	probe := func(string, string) (string, error) { return "", errNoBinary }
	st := checkBrowserEnvironment(lookPath, probe)
	if st.PythonOK || st.Ready || st.CanAutoSetup {
		t.Fatalf("expected not-ready status without python: %+v", st)
	}
	if len(st.Guidance) == 0 || !strings.Contains(st.Guidance[0], "Python") {
		t.Fatalf("expected python guidance, got %+v", st.Guidance)
	}
}

func TestCheckBrowserEnvironmentPythonOnly(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "python3" {
			return "/usr/bin/python3", nil
		}
		return "", errNoBinary
	}
	probe := func(_, code string) (string, error) {
		return "", errNoBinary // playwright import fails
	}
	st := checkBrowserEnvironment(lookPath, probe)
	if !st.PythonOK || !st.CanAutoSetup {
		t.Fatalf("expected python detected: %+v", st)
	}
	if st.PlaywrightOK || st.Ready {
		t.Fatalf("expected playwright missing: %+v", st)
	}
	if !containsSubstr(st.Guidance, "pip install playwright") {
		t.Fatalf("expected pip guidance, got %+v", st.Guidance)
	}
}

func TestCheckBrowserEnvironmentPlaywrightNoChromium(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "python3" {
			return "/usr/bin/python3", nil
		}
		return "", errNoBinary
	}
	probe := func(_, code string) (string, error) {
		if strings.Contains(code, "__version__") {
			return "1.47.0", nil
		}
		return "missing", nil // chromium path check returns missing
	}
	st := checkBrowserEnvironment(lookPath, probe)
	if !st.PlaywrightOK || st.PlaywrightVersion != "1.47.0" {
		t.Fatalf("expected playwright detected with version: %+v", st)
	}
	if st.ChromiumOK || st.Ready {
		t.Fatalf("expected chromium missing: %+v", st)
	}
	if !containsSubstr(st.Guidance, "playwright install chromium") {
		t.Fatalf("expected chromium guidance, got %+v", st.Guidance)
	}
}

func TestCheckBrowserEnvironmentReady(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "python3" {
			return "/usr/bin/python3", nil
		}
		return "", errNoBinary
	}
	probe := func(_, code string) (string, error) {
		if strings.Contains(code, "__version__") {
			return "1.47.0", nil
		}
		return "ok", nil
	}
	st := checkBrowserEnvironment(lookPath, probe)
	if !st.Ready || !st.ChromiumOK || len(st.Guidance) != 0 {
		t.Fatalf("expected fully ready status: %+v", st)
	}
}

func TestSetupBrowserEnvironmentRunsMissingStepsOnly(t *testing.T) {
	status := BrowserEnvironmentStatus{PythonOK: true, PythonPath: "/usr/bin/python3", PlaywrightOK: true, ChromiumOK: false, CanAutoSetup: true}
	var calls [][]string
	run := func(_ context.Context, name string, args ...string) (string, error) {
		calls = append(calls, append([]string{name}, args...))
		return "done", nil
	}
	ready := BrowserEnvironmentStatus{Ready: true, PythonOK: true, PlaywrightOK: true, ChromiumOK: true}
	res := setupBrowserEnvironment(context.Background(), status, run, func() BrowserEnvironmentStatus { return ready })
	// Playwright already OK → only the chromium step must run.
	if len(calls) != 1 {
		t.Fatalf("expected only the chromium step, ran %d: %v", len(calls), calls)
	}
	if !containsSubstr(calls[0], "chromium") {
		t.Fatalf("expected chromium install, got %v", calls[0])
	}
	if !res.Status.Ready {
		t.Fatalf("expected recheck to report ready: %+v", res.Status)
	}
}

func TestSetupBrowserEnvironmentNoPythonIsNoop(t *testing.T) {
	status := BrowserEnvironmentStatus{PythonOK: false}
	run := func(_ context.Context, name string, args ...string) (string, error) {
		t.Fatalf("setup must not run commands without python")
		return "", nil
	}
	res := setupBrowserEnvironment(context.Background(), status, run, func() BrowserEnvironmentStatus { return status })
	if len(res.Steps) != 0 {
		t.Fatalf("expected no steps without python: %+v", res.Steps)
	}
}

func containsSubstr(list []string, want string) bool {
	for _, item := range list {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}
