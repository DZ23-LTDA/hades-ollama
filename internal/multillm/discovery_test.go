package multillm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestListUpstreamModelsOpenAIShapeWithBearer(t *testing.T) {
	const env = "DZ23_DISCOVERY_TEST_KEY"
	t.Setenv(env, "sk-discovery")
	t.Setenv(env+"_FILE", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer sk-discovery" {
			http.Error(w, "unexpected "+r.URL.Path+" "+r.Header.Get("Authorization"), http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"data":[{"id":"b-model"},{"id":"a-model"},{"id":"b-model"}]}`))
	}))
	defer server.Close()

	got, err := ListUpstreamModels(context.Background(), Provider{Name: "p", Type: ProviderTypeOpenAICompatible, BaseURL: server.URL + "/v1/", APIKeyEnv: env}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a-model", "b-model"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("models = %v, want %v", got, want)
	}
}

func TestListUpstreamModelsAnthropicHeadersAndGeminiNames(t *testing.T) {
	const env = "DZ23_DISCOVERY_TEST_KEY"
	t.Setenv(env, "ak")
	t.Setenv(env+"_FILE", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "ak" || r.Header.Get("Anthropic-Version") == "" {
			http.Error(w, "missing anthropic headers", http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"models":[{"name":"models/gemini-x"}],"data":[{"id":"claude-y"}]}`))
	}))
	defer server.Close()

	got, err := ListUpstreamModels(context.Background(), Provider{Name: "a", Type: ProviderTypeAnthropic, BaseURL: server.URL, APIKeyEnv: env}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"claude-y", "gemini-x"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("models = %v, want %v", got, want)
	}
}

func TestListUpstreamModelsErrors(t *testing.T) {
	const env = "DZ23_DISCOVERY_TEST_KEY"
	t.Setenv(env, "")
	t.Setenv(env+"_FILE", "")
	if _, err := ListUpstreamModels(context.Background(), Provider{BaseURL: "https://x", APIKeyEnv: env}, nil); !errors.Is(err, ErrNoCredential) {
		t.Fatalf("err = %v, want ErrNoCredential", err)
	}

	t.Setenv(env, "k")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid key"}`, http.StatusUnauthorized)
	}))
	defer server.Close()
	_, err := ListUpstreamModels(context.Background(), Provider{BaseURL: server.URL, APIKeyEnv: env}, server.Client())
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v, want 401", err)
	}
}
