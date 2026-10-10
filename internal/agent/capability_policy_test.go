package agent

import (
	"errors"
	"reflect"
	"testing"
)

func TestCapabilityPolicyRejectsUnknownMissionGrant(t *testing.T) {
	policy := DefaultCapabilityPolicy()
	if _, err := policy.ValidateMissionCapabilities([]string{"workspace:read", "payments:charge"}); !errors.Is(err, ErrUnknownCapability) {
		t.Fatalf("err=%v, want unknown capability", err)
	}
}

func TestCapabilityPolicyRejectsUndeclaredOrUnknownToolScopes(t *testing.T) {
	policy := DefaultCapabilityPolicy()
	cases := []ToolDescriptor{
		{Name: "missing-scopes"},
		{Name: "unknown-scope", Scopes: []string{"payments:charge"}},
		{Name: "empty-scope", Scopes: []string{""}},
	}
	for _, descriptor := range cases {
		if err := policy.ValidateToolDescriptor(descriptor); !errors.Is(err, ErrToolCapabilityDeclaration) {
			t.Errorf("descriptor=%+v err=%v, want declaration error", descriptor, err)
		}
	}
}

func TestCapabilityPolicyRequiresEveryToolScope(t *testing.T) {
	policy := DefaultCapabilityPolicy()
	descriptor := ToolDescriptor{Name: "desktop", Scopes: []string{"desktop:screen", "desktop:input"}}
	if policy.Allows(descriptor, []string{"desktop:screen"}) {
		t.Fatal("partial grant unexpectedly authorized tool")
	}
	if !policy.Allows(descriptor, []string{"desktop:input", "desktop:screen"}) {
		t.Fatal("complete grant did not authorize tool")
	}
}

func TestCapabilityPolicyKnownScopesAreDeterministic(t *testing.T) {
	got := DefaultCapabilityPolicy().KnownScopes()
	want := []string{
		"browser:files", "browser:navigate", "browser:takeover", "connector:external",
		"desktop:clipboard", "desktop:input", "desktop:process", "desktop:screen",
		"harness:cli",
		"mcp:call", "mcp:remote:call", "media:execute", "repo:read", "sandbox:execute", "terminal:allowlisted",
		"workspace:read", "workspace:write",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("known scopes=%v, want %v", got, want)
	}
}

func TestToolApprovalPolicyIncludesCapabilities(t *testing.T) {
	got := toolApprovalPolicy(ToolDescriptor{Scopes: []string{"workspace:write", "terminal:allowlisted"}}, RiskWrite)
	want := "capabilities:terminal:allowlisted,workspace:write;risk:write"
	if got != want {
		t.Fatalf("approval policy=%q, want %q", got, want)
	}
}
