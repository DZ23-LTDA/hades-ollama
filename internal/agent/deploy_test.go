package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeploymentManagerGenericProvider(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<h1>DZ23</h1>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SYNTHETIC_PRIVATE=1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DZ23_DEPLOY_TOKEN", "deploy-test-token")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/deploy" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer deploy-test-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		var payload struct {
			Files []struct {
				File string `json:"file"`
				Data string `json:"data"`
			} `json:"files"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Files) != 1 || payload.Files[0].File != "index.html" {
			t.Fatalf("files=%+v", payload.Files)
		}
		decoded, err := base64.StdEncoding.DecodeString(payload.Files[0].Data)
		if err != nil || string(decoded) != "<h1>DZ23</h1>" {
			t.Fatalf("decoded payload=%q err=%v", decoded, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"dep_123","url":"https://example.test/d/123","status":"ready"}`))
	}))
	defer server.Close()
	manager := NewDeploymentManager()
	manager.client = server.Client()
	if err := manager.Register(DeployConfig{ID: "self", Provider: "generic", BaseURL: server.URL, TokenEnv: "DZ23_DEPLOY_TOKEN"}); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Deploy(context.Background(), "self", DeploymentRequest{Name: "site", Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.DeploymentID != "dep_123" || result.URL == "" || result.Files != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestDeploymentManagerRejectsExternalHTTP(t *testing.T) {
	manager := NewDeploymentManager()
	if err := manager.Register(DeployConfig{ID: "unsafe", Provider: "generic", BaseURL: "http://example.com"}); err == nil {
		t.Fatal("external HTTP deployment unexpectedly accepted")
	}
}

func TestDeploymentManagerConcurrentRegisterAndList(t *testing.T) {
	manager := NewDeploymentManager()
	var group sync.WaitGroup
	for worker := range 8 {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			for index := range 80 {
				if worker == 0 {
					id := fmt.Sprintf("provider-%d", index)
					if err := manager.Register(DeployConfig{ID: id, OrganizationID: "org-a", Provider: "generic", BaseURL: "https://example.test"}); err != nil {
						t.Errorf("Register(%s): %v", id, err)
						return
					}
					continue
				}
				_, _ = manager.Deploy(context.Background(), "provider-0", DeploymentRequest{})
				_ = manager.List()
				_ = manager.ListForOrganization("org-a")
			}
		}(worker)
	}
	group.Wait()
	if got := len(manager.ListForOrganization("org-a")); got != 80 {
		t.Fatalf("registered providers=%d, want 80", got)
	}
}

func TestDeploymentBlocksSecretContentBeforeProviderSideEffects(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), []byte("password: \"example-secret-value\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	manager := NewDeploymentManager()
	manager.client = server.Client()
	if err := manager.Register(DeployConfig{ID: "provider", Provider: "netlify", BaseURL: server.URL}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Deploy(context.Background(), "provider", DeploymentRequest{Name: "safe-name", Root: root}); err == nil || !strings.Contains(err.Error(), "DLP") {
		t.Fatalf("deployment error = %v, want DLP block", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("deployment made %d provider request(s) with secret content", requests.Load())
	}
}

func TestDeploymentManagerReportsNetlifyPartialState(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{"a.txt": "first", "b.txt": "second"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var uploaded int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"site_1"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sites/site_1/deploys":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"dep_1","url":"https://example.test/site_1","state":"building"}`))
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/a.txt"):
			uploaded++
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/b.txt"):
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`provider upload failed`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	manager := NewDeploymentManager()
	manager.client = server.Client()
	if err := manager.Register(DeployConfig{ID: "netlify", Provider: "netlify", BaseURL: server.URL}); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Deploy(context.Background(), "netlify", DeploymentRequest{Name: "site", Root: root})
	if err == nil {
		t.Fatal("partial Netlify deploy unexpectedly succeeded")
	}
	var deploymentErr *DeploymentError
	if !errors.As(err, &deploymentErr) {
		t.Fatalf("error=%T %v", err, err)
	}
	if result.Provider != "netlify" || result.Status != "partial" || result.DeploymentID != "dep_1" || result.Files != uploaded || uploaded != 1 {
		t.Fatalf("partial result=%+v uploaded=%d", result, uploaded)
	}
}

func TestDeploymentRejectsSymlinkRoot(t *testing.T) {
	parent := t.TempDir()
	realRoot := filepath.Join(parent, "real")
	if err := os.Mkdir(realRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	linkRoot := filepath.Join(parent, "link")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := collectDeployFiles(linkRoot); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink root rejection, got %v", err)
	}
}

func TestDeploymentDialRejectsPrivateConnectedAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan bool, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil && conn != nil {
			_ = conn.Close()
			accepted <- true
			return
		}
		accepted <- false
	}()
	_, err = deploymentDialContext(context.Background(), "tcp", listener.Addr().String())
	if err == nil || !strings.Contains(err.Error(), "private") {
		t.Fatalf("expected private deployment dial rejection, got %v", err)
	}
	select {
	case contacted := <-accepted:
		if contacted {
			t.Fatal("private deployment destination received TCP before refusal")
		}
	case <-time.After(200 * time.Millisecond):
	}
}

func TestDeploymentPackageExcludesPrivateFiles(t *testing.T) {
	root := t.TempDir()
	fixtures := map[string]string{
		"index.html":      "<h1>public</h1>",
		".env":            "SYNTHETIC_PRIVATE=1",
		".env.production": "SYNTHETIC_PRIVATE=2",
		"server.key":      "SYNTHETIC_PRIVATE=3",
		"database.backup": "SYNTHETIC_PRIVATE=4",
		"runtime.log":     "SYNTHETIC_PRIVATE=5",
	}
	for name, content := range fixtures {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := collectDeployFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "index.html" {
		t.Fatalf("public deployment files=%+v", files)
	}
}

func TestBuildDeploymentManifestReportsIncludedExcludedAndHash(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"index.html":     "<h1>public</h1>",
		"logo.svg":       "<svg></svg>",
		"tokenizer.json": "{\"version\":1}",
		".env":           "SYNTHETIC_PRIVATE=1",
		"runtime.log":    "SYNTHETIC_PRIVATE=2",
		"state.sqlite":   "SYNTHETIC_PRIVATE=3",
		"api-token.json": "SYNTHETIC_PRIVATE=4",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest, err := BuildDeploymentManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SHA256 == "" || len(manifest.Included) != 3 {
		t.Fatalf("manifest included=%+v hash=%q", manifest.Included, manifest.SHA256)
	}
	if manifest.Included[0].Path != "index.html" || manifest.Included[1].Path != "logo.svg" || manifest.Included[2].Path != "tokenizer.json" {
		t.Fatalf("manifest included order=%+v", manifest.Included)
	}
	if len(manifest.Excluded) != 5 {
		t.Fatalf("manifest excluded=%+v", manifest.Excluded)
	}
	for _, entry := range manifest.Excluded {
		if entry.Reason == "" {
			t.Fatalf("excluded entry has no reason: %+v", entry)
		}
	}
	files, err := collectDeployFiles(root)
	if err != nil || len(files) != 3 {
		t.Fatalf("collect files=%+v err=%v", files, err)
	}
}

func TestDeploymentManagerRejectsChangedManifestBeforeProvider(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "index.html")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := BuildDeploymentManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	var requests int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"dep_should_not_exist"}`))
	}))
	defer server.Close()
	manager := NewDeploymentManager()
	manager.client = server.Client()
	if err := manager.Register(DeployConfig{ID: "self", Provider: "generic", BaseURL: server.URL}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = manager.Deploy(context.Background(), "self", DeploymentRequest{Name: "site", Root: root, ManifestSHA256: manifest.SHA256})
	if err == nil || !strings.Contains(err.Error(), "changed after approval") {
		t.Fatalf("expected changed-manifest rejection, got %v", err)
	}
	if requests != 0 {
		t.Fatalf("provider requests=%d, want zero", requests)
	}
}

func TestDeploymentDialRejectsPrivateResolvedAddressBeforeTCP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan bool, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil && conn != nil {
			_ = conn.Close()
			accepted <- true
			return
		}
		accepted <- false
	}()
	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}, {IP: net.ParseIP("203.0.113.8")}}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = deploymentDialContextWithResolver(ctx, "tcp", "public.example:"+portOf(listener.Addr().String()), lookup)
	if err == nil || !strings.Contains(err.Error(), "private") {
		t.Fatalf("expected pre-resolution private rejection, got %v", err)
	}
	select {
	case contacted := <-accepted:
		if contacted {
			t.Fatal("resolved private deployment destination received TCP")
		}
	case <-time.After(200 * time.Millisecond):
	}
}

func portOf(address string) string {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		panic(err)
	}
	return port
}

func TestDeploymentAllowsLocalhostHTTPConfiguration(t *testing.T) {
	manager := NewDeploymentManager()
	if err := manager.Register(DeployConfig{ID: "local", Provider: "generic", BaseURL: "http://localhost:43123"}); err != nil {
		t.Fatalf("localhost deployment config rejected: %v", err)
	}
}

func TestDeploymentProviderErrorDoesNotEchoResponseBody(t *testing.T) {
	const sentinel = "provider-opaque-diagnostic-not-for-users"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(sentinel))
	}))
	defer server.Close()
	manager := NewDeploymentManager()
	manager.client = server.Client()
	_, err := manager.request(context.Background(), DeployConfig{BaseURL: server.URL, TimeoutSeconds: 5}, http.MethodPost, "/deploy", []byte(`{}`), "application/json")
	if err == nil || !strings.Contains(err.Error(), "502") || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("deployment error=%v; expected status only and no provider body", err)
	}
}

type deploymentRoundTripFunc func(*http.Request) (*http.Response, error)

func (f deploymentRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestDeploymentTransportErrorDoesNotEchoProviderQuery(t *testing.T) {
	const sentinel = "deployment-provider-query-sentinel"
	manager := NewDeploymentManager()
	manager.client = &http.Client{Transport: deploymentRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		return nil, errors.New("transport failed for " + request.URL.String())
	})}
	_, err := manager.request(context.Background(), DeployConfig{BaseURL: "https://provider.invalid", TimeoutSeconds: 5}, http.MethodPost, "/deploy?signature="+sentinel, []byte(`{}`), "application/json")
	if err == nil || err.Error() != "deployment provider request failed" || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("deployment transport error=%v; expected stable sanitized failure", err)
	}
}

func TestDeploymentProvidersAreOrganizationScoped(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	manager := NewDeploymentManager()
	for _, config := range []DeployConfig{
		{ID: "org-a-provider", OrganizationID: "org-a", Provider: "generic", BaseURL: server.URL},
		{ID: "org-b-provider", OrganizationID: "org-b", Provider: "generic", BaseURL: server.URL},
		{ID: "legacy-global", Provider: "generic", BaseURL: server.URL},
	} {
		if err := manager.Register(config); err != nil {
			t.Fatal(err)
		}
	}
	providers := manager.ListForOrganization("org-a")
	if len(providers) != 1 || providers[0].ID != "org-a-provider" || providers[0].OrganizationID != "org-a" {
		t.Fatalf("org-a provider list=%+v", providers)
	}
	if _, err := manager.DeployForOrganization(context.Background(), "org-a", "org-b-provider", DeploymentRequest{Root: t.TempDir()}); !errors.Is(err, ErrDeploymentProviderNotFound) {
		t.Fatalf("foreign provider error=%v", err)
	}
	if _, err := manager.DeployForOrganization(context.Background(), "org-a", "legacy-global", DeploymentRequest{Root: t.TempDir()}); !errors.Is(err, ErrDeploymentProviderNotFound) {
		t.Fatalf("unscoped provider error=%v", err)
	}
	if _, err := manager.Deploy(context.Background(), "org-a-provider", DeploymentRequest{Root: t.TempDir()}); !errors.Is(err, ErrDeploymentProviderNotFound) {
		t.Fatalf("unscoped API reached tenant-owned provider: %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("foreign provider caused %d network request(s)", requests.Load())
	}
}

func TestDeploymentTenantCatalogSanitizesProviderURL(t *testing.T) {
	manager := NewDeploymentManager()
	if err := manager.Register(DeployConfig{
		ID:             "private-provider",
		OrganizationID: "org-a",
		Provider:       "generic",
		BaseURL:        "https://deploy.example.test/private-path",
		TokenEnv:       "DEPLOYMENT_TOKEN_ENV_SECRET",
	}); err != nil {
		t.Fatal(err)
	}
	providers := manager.ListForOrganization("org-a")
	if len(providers) != 1 || providers[0].BaseURL != "https://deploy.example.test" || providers[0].TokenEnv != "" {
		t.Fatalf("sanitized provider catalog=%+v", providers)
	}
	encoded, err := json.Marshal(providers)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-path", "DEPLOYMENT_TOKEN_ENV_SECRET"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("tenant deployment catalog exposed %q: %s", secret, encoded)
		}
	}
}

func TestDeploymentRejectsCredentialBearingEndpointURL(t *testing.T) {
	manager := NewDeploymentManager()
	err := manager.Register(DeployConfig{ID: "signed", Provider: "generic", BaseURL: "https://deploy.example.test/private?x.sig=provider-url-secret"})
	if err == nil {
		t.Fatal("accepted credential-bearing deployment endpoint URL")
	}
}

func TestDeploymentRequestRejectsMalformedAndOversizedSuccessfulResponses(t *testing.T) {
	oversized := strings.Repeat("x", (4<<20)+1)
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "truncated-json", body: `{"id":"partial`},
		{name: "non-object-json", body: `[]`},
		{name: "oversized", body: oversized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			manager := NewDeploymentManager()
			manager.client = server.Client()
			_, err := manager.request(context.Background(), DeployConfig{BaseURL: server.URL, TimeoutSeconds: 5}, http.MethodGet, "/result", nil, "")
			if err == nil {
				t.Fatalf("successful provider response %q was accepted", tc.name)
			}
		})
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	manager := NewDeploymentManager()
	manager.client = server.Client()
	if _, err := manager.deployGeneric(context.Background(), DeployConfig{BaseURL: server.URL, TimeoutSeconds: 5}, DeploymentRequest{}, nil); err == nil {
		t.Fatal("generic deployment accepted a success response without deployment identity")
	}
}

func TestVercelDeploymentRejectsSuccessfulResponseWithoutIdentity(t *testing.T) {
	manager := NewDeploymentManager()
	manager.client = &http.Client{Transport: deploymentRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"readyState":"READY"}`)), Request: request}, nil
	})}
	_, err := manager.deployVercel(context.Background(), DeployConfig{BaseURL: "https://127.0.0.1", TimeoutSeconds: 5}, DeploymentRequest{Name: "test", Root: t.TempDir()}, nil)
	if err == nil || !strings.Contains(err.Error(), "omitted deployment identity") {
		t.Fatalf("Vercel result without deployment identity error = %v", err)
	}
}

func TestVercelDeploymentRejectsEmptySuccessfulResponse(t *testing.T) {
	manager := NewDeploymentManager()
	manager.client = &http.Client{Transport: deploymentRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
	})}
	_, err := manager.deployVercel(context.Background(), DeployConfig{BaseURL: "https://127.0.0.1", TimeoutSeconds: 5}, DeploymentRequest{Name: "test", Root: t.TempDir()}, nil)
	if err == nil || !strings.Contains(err.Error(), "omitted deployment identity") {
		t.Fatalf("empty Vercel success error = %v, want missing identity", err)
	}
}

func TestDeploymentRejectsSecretBearingEndpointBeforePersistence(t *testing.T) {
	for _, rawURL := range []string{
		"https://example.test/deploy?x.sig=provider-secret",
		"https://example.test/deploy?safe=1;sig=provider-secret",
		"https://example.test/deploy?next=https%253A%252F%252Fexample.test%252F?token%253Dprovider-secret",
	} {
		manager := NewDeploymentManager()
		if err := manager.Register(DeployConfig{ID: "secret", Provider: "generic", BaseURL: rawURL}); err == nil {
			t.Fatalf("secret-bearing deployment endpoint %q was accepted", rawURL)
		}
		if len(manager.List()) != 0 {
			t.Fatalf("rejected endpoint %q entered manager state", rawURL)
		}
	}
}

func TestDeploymentLoopbackHostnameCannotResolveToPublicAddress(t *testing.T) {
	ctx := context.WithValue(context.Background(), deploymentLoopbackContextKey{}, true)
	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}, {IP: net.ParseIP("203.0.113.8")}}, nil
	}
	if _, err := deploymentDialContextWithResolver(ctx, "tcp", "localhost:443", lookup); err == nil || !strings.Contains(err.Error(), "outside loopback") {
		t.Fatalf("deployment loopback accepted a public DNS answer: %v", err)
	}
}

func TestDeploymentRedactsCredentialBearingSuccessfulURL(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"dep_safe","url":"https://example.test/deploy?x.sig=provider-secret&view=1","status":"ready"}`))
	}))
	defer server.Close()
	manager := NewDeploymentManager()
	manager.client = server.Client()
	if err := manager.Register(DeployConfig{ID: "self", Provider: "generic", BaseURL: server.URL}); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Deploy(context.Background(), "self", DeploymentRequest{Name: "site", Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.URL, "provider-secret") || result.URL != "[REDACTED]" {
		t.Fatalf("credential-bearing URL was exposed: %q", result.URL)
	}
}
