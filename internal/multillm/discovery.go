package multillm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

const maxModelListBytes = 4 << 20

// ErrNoCredential is returned when a provider needs a key that is not set.
var ErrNoCredential = errors.New("provider credential is not configured")

// ListUpstreamModels asks a provider which models the configured credential
// can use, through the provider's OpenAI-style GET {base_url}/models listing
// (Anthropic and Gemini's OpenAI endpoint use the same shape). Authentication
// is applied exactly as for proxied requests.
func ListUpstreamModels(ctx context.Context, provider Provider, client *http.Client) ([]string, error) {
	if provider.APIKeyEnv != "" && credentialValue(provider.APIKeyEnv) == "" {
		return nil, ErrNoCredential
	}
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/") + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	applyProviderAuth(req, provider)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxModelListBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxModelListBytes {
		return nil, errors.New("provider model list is too large")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider returned %d: %.200s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var listing struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, fmt.Errorf("decode provider model list: %w", err)
	}
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		id = strings.TrimSpace(strings.TrimPrefix(id, "models/"))
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, item := range listing.Data {
		add(item.ID)
	}
	for _, item := range listing.Models {
		if item.ID != "" {
			add(item.ID)
		} else {
			add(item.Name)
		}
	}
	sort.Strings(ids)
	return ids, nil
}
