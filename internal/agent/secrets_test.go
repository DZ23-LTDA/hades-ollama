package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretStoreEncryptsAtRestAndListsMetadata(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "test-only-credential-key")
	root := t.TempDir()
	store, err := NewSecretStore(filepath.Join(root, "secrets"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("github_token", "super-secret-value"); err != nil {
		t.Fatal(err)
	}
	value, err := store.Get("github_token")
	if err != nil || value != "super-secret-value" {
		t.Fatalf("value=%q err=%v", value, err)
	}
	data, err := os.ReadFile(filepath.Join(root, "secrets", "secrets.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "super-secret-value") {
		t.Fatal("plaintext secret was persisted")
	}
	if len(store.List()) != 1 || store.List()[0].Name != "github_token" {
		t.Fatalf("unexpected metadata: %+v", store.List())
	}
}

func TestDLPRedactsKnownCredentialShapes(t *testing.T) {
	input := "Authorization: Bearer abcdefghijklmnop1234 and key ghp_abcdefghijklmnopqrstuvwxyz123456"
	findings := ScanDLP(input)
	if len(findings) < 2 {
		t.Fatalf("expected at least two DLP findings, got %+v", findings)
	}
	redacted := RedactDLP(input)
	if strings.Contains(redacted, "abcdefghijklmnop1234") || strings.Contains(redacted, "ghp_abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatalf("DLP did not redact: %s", redacted)
	}
}

func TestDLPRedactsCredentialQueryURLsRecursively(t *testing.T) {
	input := "provider returned https://example.test/deploy?x.sig=provider-secret&view=1"
	if got := RedactDLP(input); strings.Contains(got, "provider-secret") || !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("credential-bearing URL was not redacted: %q", got)
	}
	if findings := ScanDLP(input); len(findings) == 0 {
		t.Fatal("credential-bearing URL was not detected by outbound DLP")
	}
	value := RedactValue(map[string]any{"nested": []any{map[string]any{"url": "https://example.test/deploy?x.sig=provider-secret"}}})
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "provider-secret") {
		t.Fatalf("nested signed URL survived recursive redaction: %s", encoded)
	}
}

func TestDLPRedactsAmbiguousEncodedURLCandidates(t *testing.T) {
	for _, input := range []string{
		"nested URL https%3A%2F%2Fexample.test%2Fdeploy%3Fx.sig%3Dencoded-secret",
		"double encoded https%253A%252F%252Fexample.test%252Fdeploy%253Fx.sig%253Ddeep-secret",
		`escaped URL https:\/\/example.test\/deploy?x.sig=slash-secret`,
		"malformed query https://example.test/deploy?safe=1;sig=semicolon-secret",
	} {
		got := RedactDLP(input)
		if got == input || strings.Contains(got, "encoded-secret") || strings.Contains(got, "deep-secret") || strings.Contains(got, "slash-secret") || strings.Contains(got, "semicolon-secret") {
			t.Fatalf("ambiguous credential URL survived redaction: input=%q output=%q", input, got)
		}
		if !strings.Contains(got, "[REDACTED]") {
			t.Fatalf("ambiguous credential URL was not replaced as a complete candidate: %q", got)
		}
	}
}

func TestDLPRedactsSignatureAliasesRecursively(t *testing.T) {
	value := map[string]any{
		"nested": map[string]any{
			"x.sig":              "signature-query-secret",
			"providerSignature":  "provider-signature-secret",
			"x-signature":        "hyphenated-signature-secret",
			"provider_signature": "provider-underscore-signature-secret",
			"raw":                `{"authSig":"embedded-signature-secret","auth_signature":"embedded-auth-signature-secret"}`,
		},
	}
	if err := ValidateOutboundPayload(value); err == nil {
		t.Fatal("signature alias payload was accepted for outbound delivery")
	}
	encoded, err := json.Marshal(RedactValue(value))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"signature-query-secret", "provider-signature-secret", "hyphenated-signature-secret", "provider-underscore-signature-secret", "embedded-signature-secret", "embedded-auth-signature-secret"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("signature alias %q survived recursive DLP: %s", secret, encoded)
		}
	}
	for _, key := range []string{"x.sig", "providerSignature", "authSig", "x-signature", "provider_signature", "auth_signature"} {
		if !sensitiveDLPKey(key) {
			t.Errorf("sensitiveDLPKey(%q) = false", key)
		}
	}
}

func TestDLPRedactsCredentialAndTokenAssignmentsInMetadata(t *testing.T) {
	for _, input := range []string{
		"reports/token=artifact-path-secret",
		"credential=legacy-name-secret",
		"application/x-token=legacy-media-secret",
		"credential=/media",
	} {
		got := RedactDLP(input)
		for _, secret := range []string{"artifact-path-secret", "legacy-name-secret", "legacy-media-secret", "/media"} {
			if strings.Contains(got, secret) {
				t.Fatalf("credential assignment survived redaction: input=%q output=%q", input, got)
			}
		}
	}
}

func TestRedactValueRecursesThroughToolPayloads(t *testing.T) {
	value := map[string]any{
		"nested":     map[string]any{"token": "xai-abcdefghijklmnopqrstuvwxyz123456"},
		"items":      []any{"sk-or-v1-abcdefghijklmnopqrstuvwxyz123456", "AKIA1234567890ABCDEF"},
		"credential": "api_key=super-secret-token-value",
	}
	redacted, ok := RedactValue(value).(map[string]any)
	if !ok {
		t.Fatalf("redacted type=%T", RedactValue(value))
	}
	if strings.Contains(strings.Join([]string{stringValue(redacted["credential"]), redactedString(redacted["nested"]), redactedString(redacted["items"])}, " "), "super-secret-token-value") {
		t.Fatalf("credential survived redaction: %+v", redacted)
	}
	if strings.Contains(redactedString(redacted["nested"]), "xai-") || strings.Contains(redactedString(redacted["items"]), "sk-or-v1-") || strings.Contains(redactedString(redacted["items"]), "AKIA") {
		t.Fatalf("token survived redaction: %+v", redacted)
	}
}

func TestDLPRedactsEmbeddedSensitiveJSONFromStrings(t *testing.T) {
	inputs := []string{
		`provider error: {"access_token":"ordinary-secret-value"}`,
		`nested escaped: {\"refresh_token\":\"plain-secret-value\"}`,
		`provider rejected request: {"api\u005fkey":"unicode-escaped-secret"}`,
		`provider error: {"status":"failed","customCredential":"secret-after-benign-key"}`,
		`"{\"api\\u005fkey\":\"nested-unicode-secret\"}"`,
		`{'custom\u0043redential':'escaped-single-quote-secret'}`,
		`payload: \\u007b\\u0022api_key\\u0022:\\u0022unicode-delimiter-secret\\u0022\\u007d`,
	}
	for _, input := range inputs {
		got := RedactDLP(input)
		if got != "[REDACTED]" {
			t.Fatalf("embedded sensitive JSON survived redaction: %q", got)
		}
	}
	deep := `{"api_key":"over-depth-secret"}`
	for range 10 {
		encoded, err := json.Marshal(deep)
		if err != nil {
			t.Fatal(err)
		}
		deep = string(encoded)
	}
	if got := RedactDLP(deep); got != "[REDACTED]" {
		t.Fatalf("over-depth embedded JSON must fail closed, got %q", got)
	}
	longKey := strings.Repeat("a", 300) + "customCredential"
	longKeyPayload := `{"` + longKey + `":"long-key-secret"}`
	if got := RedactDLP(longKeyPayload); got != "[REDACTED]" {
		t.Fatalf("overlong embedded property must fail closed, got %q", got)
	}
	result := RedactValue(map[string]any{"response": `{"api_key":"plain-secret-value"}`})
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "plain-secret-value") || strings.Contains(string(encoded), "api_key") {
		t.Fatalf("tool result serialized embedded secret: %s", encoded)
	}
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func redactedString(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func TestSecretStoreRefreshesAndPersistsMutationsAcrossInstances(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "shared-process-secret-test-key")
	root := filepath.Join(t.TempDir(), "secrets")
	first, err := NewSecretStore(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewSecretStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Set("from_first", "first-secret-value"); err != nil {
		t.Fatal(err)
	}
	if err := second.Set("from_second", "second-secret-value"); err != nil {
		t.Fatal(err)
	}
	if got, err := first.Get("from_second"); err != nil || got != "second-secret-value" {
		t.Fatalf("cross-instance Get returned %q err=%v", got, err)
	}
	if err := first.Delete("from_second"); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Get("from_second"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted cross-instance secret remained visible: %v", err)
	}
}
