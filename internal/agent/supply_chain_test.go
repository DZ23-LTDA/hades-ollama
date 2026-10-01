package agent

import (
	"crypto/ed25519"
	"errors"
	"os"
	"testing"
)

func TestSignedSkillRequiresAuthorizedKeyAndRejectsTamper(t *testing.T) {
	public, private, _ := ed25519.GenerateKey(nil)
	manifest, err := SignSkillManifest(SkillManifest{ID: "signed", Version: "1", Description: "safe", Scopes: []string{"workspace:read"}}, private, "release-key")
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultCapabilityPolicy().WithTrustedSkillKey("release-key", public)
	trusted, err := policy.PromoteSkillTrusted(manifest)
	if err != nil || !trusted.Trusted {
		t.Fatalf("promotion err=%v manifest=%+v", err, trusted)
	}
	manifest.Description = "tampered"
	if _, err := policy.PromoteSkillTrusted(manifest); !errors.Is(err, ErrSkillSignatureInvalid) {
		t.Fatalf("tamper err=%v", err)
	}
	if _, err := DefaultCapabilityPolicy().PromoteSkillTrusted(manifest); !errors.Is(err, ErrSkillKeyUnauthorized) {
		t.Fatalf("unauthorized err=%v", err)
	}
}

func TestUnsignedSkillNeverTrusted(t *testing.T) {
	_, err := DefaultCapabilityPolicy().PromoteSkillTrusted(SkillManifest{ID: "unsigned", Version: "1", Trusted: true})
	if !errors.Is(err, ErrSkillKeyUnauthorized) {
		t.Fatalf("err=%v", err)
	}
}

func TestArtifactSignatureVerifiesAndRejectsTamper(t *testing.T) {
	public, private, _ := ed25519.GenerateKey(nil)
	path := t.TempDir() + "/artifact.zip"
	if err := os.WriteFile(path, []byte("artifact-v1"), 0o600); err != nil {
		t.Fatal(err)
	}
	signed, err := SignArtifact(path, private, "release-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyArtifactSignature(path, signed, public); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("artifact-v2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(VerifyArtifactSignature(path, signed, public), ErrArtifactSignatureInvalid) {
		t.Fatal("tampered artifact verified")
	}
}
