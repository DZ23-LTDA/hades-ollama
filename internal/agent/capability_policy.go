package agent

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrUnknownCapability         = errors.New("unknown mission capability")
	ErrToolCapabilityDeclaration = errors.New("tool must declare known capabilities")
)

// CapabilityPolicy is the runtime boundary between a mission grant and a tool.
// It is intentionally deny-by-default: a descriptor with no scopes, or a grant
// outside the known vocabulary, cannot authorize execution.
type CapabilityPolicy struct {
	known            map[string]struct{}
	trustedSkillKeys map[string]ed25519.PublicKey
}

func DefaultCapabilityPolicy() CapabilityPolicy {
	return NewCapabilityPolicy([]string{
		"workspace:read",
		"workspace:write",
		"terminal:allowlisted",
		"sandbox:execute",
		"browser:navigate",
		"browser:files",
		"browser:takeover",
		"desktop:screen",
		"desktop:input",
		"desktop:clipboard",
		"desktop:process",
		"mcp:call",
		"mcp:remote:call",
		"connector:external",
		"media:execute",
		"repo:read",
	})
}

func NewCapabilityPolicy(scopes []string) CapabilityPolicy {
	known := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		if normalized := strings.TrimSpace(scope); normalized != "" {
			known[normalized] = struct{}{}
		}
	}
	return CapabilityPolicy{known: known, trustedSkillKeys: map[string]ed25519.PublicKey{}}
}

// WithTrustedSkillKey adds one explicitly authorized signer to the policy.
// The key is copied so callers cannot mutate the trust root after construction.
func (p CapabilityPolicy) WithTrustedSkillKey(keyID string, key ed25519.PublicKey) CapabilityPolicy {
	if p.trustedSkillKeys == nil {
		p.trustedSkillKeys = map[string]ed25519.PublicKey{}
	}
	if len(key) == ed25519.PublicKeySize && strings.TrimSpace(keyID) != "" {
		p.trustedSkillKeys[strings.TrimSpace(keyID)] = append(ed25519.PublicKey(nil), key...)
	}
	return p
}

func (p CapabilityPolicy) VerifySkillAttestation(manifest SkillManifest) error {
	key, ok := p.trustedSkillKeys[strings.TrimSpace(manifest.SigningKeyID)]
	if !ok {
		return ErrSkillKeyUnauthorized
	}
	return VerifySkillManifestSignature(manifest, key)
}

// PromoteSkillTrusted is the only capability-policy path that can elevate a
// skill. A persisted trusted boolean is never accepted as authority.
func (p CapabilityPolicy) PromoteSkillTrusted(manifest SkillManifest) (SkillManifest, error) {
	if err := p.VerifySkillAttestation(manifest); err != nil {
		manifest.Trusted = false
		return manifest, err
	}
	manifest.Trusted = true
	return manifest, nil
}

func (p CapabilityPolicy) ValidateMissionCapabilities(capabilities []string) ([]string, error) {
	normalized := normalizeMissionCapabilities(capabilities)
	for _, capability := range normalized {
		if _, ok := p.known[capability]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownCapability, capability)
		}
	}
	return normalized, nil
}

func (p CapabilityPolicy) ValidateToolDescriptor(descriptor ToolDescriptor) error {
	if strings.TrimSpace(descriptor.Name) == "" {
		return fmt.Errorf("%w: tool name is required", ErrToolCapabilityDeclaration)
	}
	if len(descriptor.Scopes) == 0 {
		return fmt.Errorf("%w: %s has no scopes", ErrToolCapabilityDeclaration, descriptor.Name)
	}
	seen := make(map[string]struct{}, len(descriptor.Scopes))
	for _, scope := range descriptor.Scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			return fmt.Errorf("%w: %s has an empty scope", ErrToolCapabilityDeclaration, descriptor.Name)
		}
		if _, duplicate := seen[scope]; duplicate {
			return fmt.Errorf("%w: %s repeats scope %q", ErrToolCapabilityDeclaration, descriptor.Name, scope)
		}
		seen[scope] = struct{}{}
		if _, known := p.known[scope]; !known {
			return fmt.Errorf("%w: %s declares %q", ErrToolCapabilityDeclaration, descriptor.Name, scope)
		}
	}
	return nil
}

func (p CapabilityPolicy) Allows(descriptor ToolDescriptor, granted []string) bool {
	if p.ValidateToolDescriptor(descriptor) != nil {
		return false
	}
	validated, err := p.ValidateMissionCapabilities(granted)
	if err != nil {
		return false
	}
	allowed := make(map[string]struct{}, len(validated))
	for _, scope := range validated {
		allowed[scope] = struct{}{}
	}
	for _, scope := range descriptor.Scopes {
		if _, ok := allowed[strings.TrimSpace(scope)]; !ok {
			return false
		}
	}
	return true
}

func (p CapabilityPolicy) KnownScopes() []string {
	result := make([]string, 0, len(p.known))
	for scope := range p.known {
		result = append(result, scope)
	}
	sort.Strings(result)
	return result
}

func toolApprovalPolicy(descriptor ToolDescriptor, risk RiskClass) string {
	scopes := append([]string(nil), descriptor.Scopes...)
	sort.Strings(scopes)
	return "capabilities:" + strings.Join(scopes, ",") + ";risk:" + string(risk)
}

func toolDescriptorSHA256(descriptor ToolDescriptor) (string, error) {
	descriptor.Scopes = append([]string(nil), descriptor.Scopes...)
	sort.Strings(descriptor.Scopes)
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

type capabilityPolicyValidator interface {
	ValidateToolDescriptor(ToolDescriptor) error
}

var _ capabilityPolicyValidator = DefaultCapabilityPolicy()
