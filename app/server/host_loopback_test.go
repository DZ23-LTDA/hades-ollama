//go:build windows || darwin

package server

import "testing"

func TestHostSpecIsLoopback(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:11434":        true,
		"127.0.0.1":              true,
		"localhost:11434":        true,
		"http://127.0.0.1:11434": true,
		"[::1]:11434":            true,
		"0.0.0.0:11434":          false,
		"0.0.0.0":                false,
		"http://0.0.0.0:11434":   false,
		"192.168.1.10:11434":     false,
		"[::]:11434":             false,
	}
	for spec, want := range cases {
		if got := hostSpecIsLoopback(spec); got != want {
			t.Errorf("hostSpecIsLoopback(%q) = %v, want %v", spec, got, want)
		}
	}
}

func TestForceLoopbackHostSpec(t *testing.T) {
	cases := map[string]string{
		"0.0.0.0:11434":        "127.0.0.1:11434",
		"http://0.0.0.0:11434": "http://127.0.0.1:11434",
		"192.168.1.10:11434":   "127.0.0.1:11434",
		"0.0.0.0":              "127.0.0.1",
	}
	for spec, want := range cases {
		if got := forceLoopbackHostSpec(spec); got != want {
			t.Errorf("forceLoopbackHostSpec(%q) = %q, want %q", spec, got, want)
		}
	}
}

func TestAgentTLSConfigured(t *testing.T) {
	if agentTLSConfigured(map[string]string{}) {
		t.Fatal("empty env must not be considered TLS-configured")
	}
	if agentTLSConfigured(map[string]string{"OLLAMA_AGENT_TLS_CERT_FILE": "c.pem"}) {
		t.Fatal("cert without key must not be considered TLS-configured")
	}
	if !agentTLSConfigured(map[string]string{"OLLAMA_AGENT_TLS_CERT_FILE": "c.pem", "OLLAMA_AGENT_TLS_KEY_FILE": "k.pem"}) {
		t.Fatal("cert and key must be considered TLS-configured")
	}
}
