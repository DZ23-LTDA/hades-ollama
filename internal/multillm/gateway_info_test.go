package multillm

import (
	"encoding/json"
	"strings"
	"testing"
)

func boolPtr(value bool) *bool { return &value }

func TestRotationEffectiveResolvesDefaultsAndLimits(t *testing.T) {
	cases := []struct {
		name     string
		declared *RotationConfig
		want     GatewayRotation
	}{
		{name: "ausente liga a rotacao padrao", declared: nil, want: GatewayRotation{Enabled: true, MaxAttempts: 3}},
		{name: "bloco vazio usa os padroes", declared: &RotationConfig{}, want: GatewayRotation{Enabled: true, MaxAttempts: 3}},
		{name: "desligado continua desligado", declared: &RotationConfig{Enabled: boolPtr(false)}, want: GatewayRotation{Enabled: false, MaxAttempts: 3}},
		{name: "cross provider opt in", declared: &RotationConfig{CrossProvider: boolPtr(true)}, want: GatewayRotation{Enabled: true, MaxAttempts: 3, CrossProvider: true}},
		{name: "tentativas abaixo do minimo sobem para um", declared: &RotationConfig{MaxAttempts: -4}, want: GatewayRotation{Enabled: true, MaxAttempts: 3}},
		{name: "tentativas acima do teto descem para o teto", declared: &RotationConfig{MaxAttempts: 99}, want: GatewayRotation{Enabled: true, MaxAttempts: maxRotationAttempts}},
		{name: "tentativas validas sao respeitadas", declared: &RotationConfig{MaxAttempts: 2}, want: GatewayRotation{Enabled: true, MaxAttempts: 2}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.declared.Effective(); got != testCase.want {
				t.Fatalf("Effective() = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestRegistryRotationPolicyMatchesEffective(t *testing.T) {
	registry := &Registry{rotation: &RotationConfig{Enabled: boolPtr(false), MaxAttempts: 4, CrossProvider: boolPtr(true)}}
	if got := registry.RotationPolicy(); got != (GatewayRotation{Enabled: false, MaxAttempts: 4, CrossProvider: true}) {
		t.Fatalf("RotationPolicy() = %+v", got)
	}
	if got := (*Registry)(nil).RotationPolicy(); got != (GatewayRotation{Enabled: true, MaxAttempts: 3}) {
		t.Fatalf("nil RotationPolicy() = %+v", got)
	}
}

func TestDescribeGatewayHidesTheKeyUnlessTheCallerIsLocal(t *testing.T) {
	t.Setenv(DefaultGatewayKeyEnv, "gateway-secret")
	cfg := Config{
		GatewayAPIKeyEnv: DefaultGatewayKeyEnv,
		Rotation:         &RotationConfig{MaxAttempts: 2},
		Providers: []Provider{
			{Name: "anthropic", Type: ProviderTypeAnthropic, BaseURL: "https://api.anthropic.com"},
		},
	}

	local := DescribeGateway(cfg, nil, "http://127.0.0.1:11434/", "/etc/hades/providers.json", true)
	if local.BaseURL != "http://127.0.0.1:11434" {
		t.Fatalf("BaseURL = %q", local.BaseURL)
	}
	if !local.KeyPresent || local.Key != "gateway-secret" {
		t.Fatalf("local caller must see the key: present=%v key=%q", local.KeyPresent, local.Key)
	}
	if local.Loopback || local.Providers != 1 || local.Rotation.MaxAttempts != 2 {
		t.Fatalf("unexpected local info: %+v", local)
	}
	if local.KeyEnv != DefaultGatewayKeyEnv {
		t.Fatalf("KeyEnv = %q", local.KeyEnv)
	}

	remote := DescribeGateway(cfg, nil, "http://127.0.0.1:11434", "/etc/hades/providers.json", false)
	if remote.Key != "" {
		t.Fatalf("remote caller must not receive the key, got %q", remote.Key)
	}
	if !remote.KeyPresent {
		t.Fatal("remote caller must still learn that a key exists")
	}
}

func TestDescribeGatewayDefaultsToLoopbackAndTheStandardEnv(t *testing.T) {
	cfg := Config{Providers: []Provider{{Name: "openai", Type: ProviderTypeOpenAICompatible, BaseURL: "https://api.openai.com"}}}
	info := DescribeGateway(cfg, nil, "", "", false)
	if !info.Loopback {
		t.Fatal("without gateway_api_key_env the gateway is for this computer only")
	}
	if info.KeyEnv != DefaultGatewayKeyEnv {
		t.Fatalf("KeyEnv = %q", info.KeyEnv)
	}
	if info.KeyPresent || info.Key != "" {
		t.Fatal("no key may be reported when none exists")
	}
	if len(info.Protocols) != 4 {
		t.Fatalf("Protocols = %d", len(info.Protocols))
	}
}

func TestGatewayInfoSerialisationOmitsTheKeyWhenAbsent(t *testing.T) {
	payload, err := json.Marshal(GatewayInfo{KeyEnv: DefaultGatewayKeyEnv})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(payload), "gateway_key\"") {
		t.Fatalf("empty key must be omitted: %s", payload)
	}
	if !strings.Contains(string(payload), `"gateway_key_env"`) {
		t.Fatalf("env name must be present: %s", payload)
	}
}
