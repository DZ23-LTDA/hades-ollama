package agent

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

var (
	ErrSkillSignatureRequired   = errors.New("skill signature is required")
	ErrSkillSignatureInvalid    = errors.New("skill signature is invalid")
	ErrSkillKeyUnauthorized     = errors.New("skill signing key is not authorized")
	ErrArtifactSignatureInvalid = errors.New("artifact signature is invalid")
)

// SkillManifestSignatureBytes is the canonical, unsigned representation used
// for both signing and verification. Trusted is deliberately excluded from the
// signed claim so a signer cannot grant trust by editing a boolean.
func SkillManifestSignatureBytes(manifest SkillManifest) ([]byte, error) {
	manifest.Trusted = false
	manifest.Signature = ""
	manifest.ContentSHA256 = ""
	manifest.Scopes = append([]string(nil), manifest.Scopes...)
	manifest.Tools = append([]string(nil), manifest.Tools...)
	sort.Strings(manifest.Scopes)
	sort.Strings(manifest.Tools)
	return json.Marshal(manifest)
}

func SkillManifestDigest(manifest SkillManifest) (string, error) {
	data, err := SkillManifestSignatureBytes(manifest)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func SignSkillManifest(manifest SkillManifest, privateKey ed25519.PrivateKey, keyID string) (SkillManifest, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return SkillManifest{}, errors.New("invalid skill signing key")
	}
	manifest.SigningKeyID = strings.TrimSpace(keyID)
	data, err := SkillManifestSignatureBytes(manifest)
	if err != nil {
		return SkillManifest{}, err
	}
	digest := sha256.Sum256(data)
	manifest.ContentSHA256 = hex.EncodeToString(digest[:])
	manifest.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, data))
	manifest.Trusted = false
	return manifest, nil
}

func VerifySkillManifestSignature(manifest SkillManifest, publicKey ed25519.PublicKey) error {
	if strings.TrimSpace(manifest.Signature) == "" || strings.TrimSpace(manifest.SigningKeyID) == "" {
		return ErrSkillSignatureRequired
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return ErrSkillSignatureInvalid
	}
	data, err := SkillManifestSignatureBytes(manifest)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if !strings.EqualFold(manifest.ContentSHA256, hex.EncodeToString(digest[:])) {
		return ErrSkillSignatureInvalid
	}
	signature, err := base64.RawStdEncoding.DecodeString(manifest.Signature)
	if err != nil || !ed25519.Verify(publicKey, data, signature) {
		return ErrSkillSignatureInvalid
	}
	return nil
}

// SignedArtifact is a detached signature over the exact SHA-256 digest of an
// artifact. It is intentionally independent of a release transport.
type SignedArtifact struct {
	Artifact     string `json:"artifact"`
	SHA256       string `json:"sha256"`
	SigningKeyID string `json:"signing_key_id"`
	Signature    string `json:"signature"`
}

func SignArtifact(path string, privateKey ed25519.PrivateKey, keyID string) (SignedArtifact, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return SignedArtifact{}, errors.New("invalid artifact signing key")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return SignedArtifact{}, err
	}
	digest := sha256.Sum256(data)
	digestHex := hex.EncodeToString(digest[:])
	return SignedArtifact{Artifact: path, SHA256: digestHex, SigningKeyID: strings.TrimSpace(keyID), Signature: base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(digestHex)))}, nil
}

func VerifyArtifactSignature(path string, signed SignedArtifact, publicKey ed25519.PublicKey) error {
	if len(publicKey) != ed25519.PublicKeySize || signed.Signature == "" || signed.SHA256 == "" {
		return ErrArtifactSignatureInvalid
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	actual := hex.EncodeToString(digest[:])
	if !strings.EqualFold(actual, signed.SHA256) {
		return fmt.Errorf("%w: checksum mismatch", ErrArtifactSignatureInvalid)
	}
	sig, err := base64.RawStdEncoding.DecodeString(signed.Signature)
	if err != nil || !ed25519.Verify(publicKey, []byte(signed.SHA256), sig) {
		return ErrArtifactSignatureInvalid
	}
	return nil
}
