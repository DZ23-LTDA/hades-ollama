package multillm

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"
)

// The guided presets are configuration templates: they never carry a
// credential, only the name of the environment variable that holds it. These
// tests prove that every preset produces a Provider accepted by the same
// validation used for configuration coming from disk, so a guided setup cannot
// bypass the HTTPS/loopback/private-address rules.

func TestBuiltInProviderPresetsAreValidAndUnique(t *testing.T) {
	presets := BuiltInProviderPresets()
	if len(presets) == 0 {
		t.Fatal("no built-in provider presets")
	}
	seen := map[string]bool{}
	for _, preset := range presets {
		if preset.ID == "" || strings.TrimSpace(preset.ID) != preset.ID {
			t.Fatalf("preset has invalid id %q", preset.ID)
		}
		if seen[preset.ID] {
			t.Fatalf("duplicate preset id %q", preset.ID)
		}
		seen[preset.ID] = true
		if strings.TrimSpace(preset.Name) == "" {
			t.Fatalf("preset %q has no display name", preset.ID)
		}
		if len(preset.Paths) == 0 {
			t.Fatalf("preset %q declares no protocol path", preset.ID)
		}
		for _, path := range preset.Paths {
			if !supportedPath(path) {
				t.Fatalf("preset %q declares unsupported path %q", preset.ID, path)
			}
		}
		if _, err := preset.Provider(nil); err != nil {
			t.Fatalf("preset %q does not build a valid provider: %v", preset.ID, err)
		}
	}
	if !sort.SliceIsSorted(presets, func(i, j int) bool { return presets[i].ID < presets[j].ID }) {
		t.Fatal("presets are not sorted by id")
	}
}

func TestPresetsCoverRequiredProviderFamilies(t *testing.T) {
	required := []string{
		"ollama", "vllm", "llamacpp", "lmstudio",
		"openai", "anthropic", "gemini", "deepseek", "groq", "mistral", "openrouter", "xai",
	}
	for _, id := range required {
		if _, ok := ProviderPresetByID(id); !ok {
			t.Errorf("missing preset for required provider family %q", id)
		}
	}
}

func TestLocalPresetsStayOnLoopbackWithoutCredential(t *testing.T) {
	local := 0
	for _, preset := range BuiltInProviderPresets() {
		if !preset.Local {
			continue
		}
		local++
		provider, err := preset.Provider(nil)
		if err != nil {
			t.Fatalf("preset %q: %v", preset.ID, err)
		}
		if !provider.AllowPrivate || !provider.AllowInsecureLoopback {
			t.Fatalf("local preset %q must opt in to private/loopback HTTP explicitly", preset.ID)
		}
		if preset.RequiresKey || preset.APIKeyEnv != "" {
			t.Fatalf("local preset %q must not require a credential", preset.ID)
		}
		if !isLoopbackHostname(hostnameOf(t, preset.BaseURL)) {
			t.Fatalf("local preset %q must point at loopback, got %q", preset.ID, preset.BaseURL)
		}
	}
	if local == 0 {
		t.Fatal("expected at least one local preset")
	}
}

func TestRemotePresetsUseHTTPSAndDeclareEnvName(t *testing.T) {
	remote := 0
	for _, preset := range BuiltInProviderPresets() {
		if preset.Local || preset.ID == "openai-compatible" {
			continue
		}
		remote++
		if !strings.HasPrefix(preset.BaseURL, "https://") {
			t.Fatalf("remote preset %q must use HTTPS, got %q", preset.ID, preset.BaseURL)
		}
		if !preset.RequiresKey || strings.TrimSpace(preset.APIKeyEnv) == "" {
			t.Fatalf("remote preset %q must declare the credential env var", preset.ID)
		}
		if strings.TrimSpace(preset.AuthStyle) == "" {
			t.Fatalf("remote preset %q must declare an auth style", preset.ID)
		}
	}
	if remote == 0 {
		t.Fatal("expected at least one remote preset")
	}
}

func TestProviderPresetByIDIgnoresCaseAndSpacing(t *testing.T) {
	if _, ok := ProviderPresetByID("  OLLAMA  "); !ok {
		t.Fatal("expected OLLAMA to resolve")
	}
	if _, ok := ProviderPresetByID("nao-existe"); ok {
		t.Fatal("unknown preset must not resolve")
	}
}

func TestPresetProviderUsesEnvironmentForCredentialOnly(t *testing.T) {
	preset, ok := ProviderPresetByID("xai")
	if !ok {
		t.Fatal("xai preset missing")
	}
	provider, err := preset.Provider(nil)
	if err != nil {
		t.Fatalf("xai preset: %v", err)
	}
	// The provider carries the variable NAME only; no literal key anywhere.
	for _, value := range []string{provider.APIKeyEnv, provider.BaseURL, provider.Name} {
		if strings.Contains(value, "sk-") {
			t.Fatalf("preset leaked a literal credential in %q", value)
		}
	}
	t.Setenv(provider.APIKeyEnv, "chave-de-teste")
	request, err := http.NewRequest(http.MethodPost, provider.BaseURL+"/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	applyProviderAuth(request, provider)
	if got := request.Header.Get("Authorization"); got != "Bearer chave-de-teste" {
		t.Fatalf("unexpected Authorization header %q", got)
	}
}

func TestAnthropicPresetUsesAnthropicHeaders(t *testing.T) {
	preset, ok := ProviderPresetByID("anthropic")
	if !ok {
		t.Fatal("anthropic preset missing")
	}
	provider, err := preset.Provider(nil)
	if err != nil {
		t.Fatalf("anthropic preset: %v", err)
	}
	t.Setenv(provider.APIKeyEnv, "chave-de-teste")
	request, err := http.NewRequest(http.MethodPost, provider.BaseURL+"/v1/messages", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	applyProviderAuth(request, provider)
	if got := request.Header.Get("X-API-Key"); got != "chave-de-teste" {
		t.Fatalf("unexpected X-API-Key header %q", got)
	}
	if got := request.Header.Get("Anthropic-Version"); got == "" {
		t.Fatal("anthropic preset must send Anthropic-Version")
	}
}

func TestGenericPresetNeedsOperatorBaseURL(t *testing.T) {
	preset, ok := ProviderPresetByID("openai-compatible")
	if !ok {
		t.Fatal("generic preset missing")
	}
	if preset.BaseURL != GenericPresetBaseURL {
		t.Fatalf("generic preset must ship a reserved placeholder, got %q", preset.BaseURL)
	}
	if !strings.Contains(strings.ToLower(preset.Notes), "base_url") {
		t.Fatal("generic preset must tell the operator to edit base_url")
	}
	if _, err := preset.Provider(nil); err != nil {
		t.Fatalf("generic preset must still be structurally valid: %v", err)
	}
}

func TestPresetProviderRejectsUnsupportedType(t *testing.T) {
	broken := ProviderPreset{
		ID:      "quebrado",
		Name:    "Quebrado",
		Type:    "protocolo-inexistente",
		BaseURL: "https://api.exemplo.invalid/v1",
		Paths:   []string{"/v1/chat/completions"},
	}
	if _, err := broken.Provider(nil); err == nil {
		t.Fatal("an unsupported provider type must fail validation")
	}
}

func hostnameOf(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse %q: %v", rawURL, err)
	}
	return parsed.Hostname()
}
