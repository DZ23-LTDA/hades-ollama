package secrets

import (
	"errors"
	"os"
	"testing"
)

func TestValidateRejectsBadInput(t *testing.T) {
	for _, tc := range []struct{ env, key string }{
		{"lower_case", "k"},
		{"A", "k"},
		{"OPENAI_API_KEY", ""},
		{"OPENAI_API_KEY", "two\nlines"},
		{"OPENAI_API_KEY", string(make([]byte, MaxKeyBytes+1))},
	} {
		if err := validate(tc.env, tc.key); !errors.Is(err, ErrInvalid) {
			t.Errorf("validate(%q, len %d) = %v, want ErrInvalid", tc.env, len(tc.key), err)
		}
	}
	if err := validate("OPENAI_API_KEY", "  sk-test  "); err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}
}

func TestProtectDoesNotStoreKeyInClear(t *testing.T) {
	payload, err := protect("sk-unit-test-value")
	if err != nil {
		t.Fatal(err)
	}
	if fileExtension == ".dpapi" && string(payload) == "sk-unit-test-value" {
		t.Fatal("windows payload must be DPAPI-protected")
	}
}

func TestConfiguredReadsFileVariable(t *testing.T) {
	const env = "DZ23_SECRETS_TEST_KEY"
	t.Setenv(env, "")
	path := t.TempDir() + "/k" + fileExtension
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(env+"_FILE", path)
	if !Configured(env) {
		t.Fatal("expected configured via _FILE")
	}
	t.Setenv(env+"_FILE", path+".missing")
	if Configured(env) {
		t.Fatal("missing file must not count as configured")
	}
}

func TestAdoptOnlyFillsMissingVariables(t *testing.T) {
	const env = "DZ23_SECRETS_ADOPT_KEY"
	dir := t.TempDir()
	t.Setenv(env, "")
	t.Setenv(env+"_FILE", "")
	if Adopt(dir, env) {
		t.Fatal("nothing saved yet, must not adopt")
	}
	if err := os.WriteFile(Path(dir, env), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !Adopt(dir, env) || os.Getenv(env+"_FILE") != Path(dir, env) {
		t.Fatal("expected adoption of the saved file")
	}
	t.Setenv(env+"_FILE", "/elsewhere")
	if Adopt(dir, env) || os.Getenv(env+"_FILE") != "/elsewhere" {
		t.Fatal("must not override an explicit _FILE")
	}
}

func TestAdoptFromLegacyName(t *testing.T) {
	const env = "DZ23_SECRETS_LEGACY_KEY"
	dir := t.TempDir()
	t.Setenv(env, "")
	t.Setenv(env+"_FILE", "")
	if err := os.WriteFile(Path(dir, "OLLAMA_DZ23_"+env), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if Adopt(dir, env) {
		t.Fatal("no file under the current name")
	}
	if !AdoptFrom(dir, "OLLAMA_DZ23_"+env, env) || os.Getenv(env+"_FILE") != Path(dir, "OLLAMA_DZ23_"+env) {
		t.Fatal("expected the legacy file to be adopted")
	}
}
