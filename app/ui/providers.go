//go:build windows || darwin

package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/ollama/ollama/app/secrets"
	"github.com/ollama/ollama/internal/multillm"
)

// ProviderStatus is what the UI may know about a configured AI provider.
// It never contains a credential value.
type ProviderStatus struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	BaseURL    string `json:"base_url"`
	APIKeyEnv  string `json:"api_key_env,omitempty"`
	Models     int    `json:"models"`
	Enabled    bool   `json:"enabled"`
	Configured bool   `json:"configured"`
	NeedsKey   bool   `json:"needs_key"`
}

func providerConfigPath() string {
	return strings.TrimSpace(os.Getenv("OLLAMA_DZ23_CONFIG"))
}

func providerSecretsDir() (string, error) {
	dir := multillm.DefaultCredentialDir()
	if dir == "" {
		return "", errors.New("no directory for saved credentials")
	}
	return dir, nil
}

func loadProviderConfig() (multillm.Config, error) {
	var cfg multillm.Config
	path := providerConfigPath()
	if path == "" {
		return cfg, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// The installer defines OLLAMA_DZ23_CONFIG but does not create the
			// file; treat a missing file as an empty config so the UI shows the
			// empty state plus the "add provider" form, and the first create
			// writes the file.
			return cfg, nil
		}
		return cfg, fmt.Errorf("read provider config: %w", err)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("decode provider config: %w", err)
	}
	return cfg, nil
}

func providerStatuses(cfg multillm.Config) []ProviderStatus {
	out := make([]ProviderStatus, 0, len(cfg.Providers))
	for _, p := range cfg.Providers {
		needsKey := p.APIKeyEnv != ""
		out = append(out, ProviderStatus{
			Name:       p.Name,
			Type:       p.Type,
			BaseURL:    p.BaseURL,
			APIKeyEnv:  p.APIKeyEnv,
			Models:     len(p.Models),
			Enabled:    p.Enabled == nil || *p.Enabled,
			Configured: !needsKey || secrets.Configured(p.APIKeyEnv),
			NeedsKey:   needsKey,
		})
	}
	return out
}

func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) error {
	AdoptProviderCredentials()
	cfg, err := loadProviderConfig()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(map[string]any{
		"config_path": providerConfigPath(),
		"providers":   providerStatuses(cfg),
	})
}

// providerEnvNamePattern mirrors the accepted credential variable names in
// package secrets so the UI can reject or generate a valid name up front.
var providerEnvNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)

// deriveProviderEnvName builds a deterministic credential variable name from a
// provider name, e.g. "OpenAI" -> "OPENAI_API_KEY". The result always matches
// providerEnvNamePattern. It returns "" only if no valid name can be built.
func deriveProviderEnvName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	// A variable must start with a letter.
	base := strings.TrimLeft(b.String(), "_0123456789")
	base = strings.Trim(base, "_")
	if base == "" {
		base = "PROVEDOR"
	}
	// Keep room for the "_API_KEY" suffix within the 64-char limit.
	if len(base) > 55 {
		base = strings.Trim(base[:55], "_")
	}
	env := base + "_API_KEY"
	if !providerEnvNamePattern.MatchString(env) {
		return ""
	}
	return env
}

// isLoopbackHost reports whether hostname is localhost or a loopback IP. HTTP
// endpoints are accepted only for loopback; everything external must be HTTPS.
func isLoopbackHost(hostname string) bool {
	h := strings.Trim(strings.TrimSpace(hostname), "[]")
	if strings.EqualFold(h, "localhost") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// createProviderRequest is the JSON body for POST /api/v1/providers. The API
// key, when present, is stored only in the OS credential vault and never in the
// provider config file, in the response, or in logs.
type createProviderRequest struct {
	Name      string   `json:"name"`
	BaseURL   string   `json:"base_url"`
	Type      string   `json:"type"`
	APIKeyEnv string   `json:"api_key_env"`
	Models    []string `json:"models"`
	APIKey    string   `json:"api_key"`
}

// createProvider registers a new provider from the UI: it validates the
// request, appends the provider to the config, validates the whole config with
// multillm.LoadBytes before writing, persists it, optionally stores the API key
// in the credential vault, and restarts the router so the change takes effect.
func (s *Server) createProvider(w http.ResponseWriter, r *http.Request) error {
	path := providerConfigPath()
	if path == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return errors.New("OLLAMA_DZ23_CONFIG não está definido; instale novamente o aplicativo ou defina essa variável para cadastrar provedores")
	}

	var body createProviderRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("corpo da requisição inválido: %w", err)
	}

	name := strings.TrimSpace(body.Name)
	if name == "" || strings.ContainsAny(name, "/: \\") {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("o nome do provedor é obrigatório e não pode conter espaços nem os caracteres / : \\")
	}

	providerType := strings.TrimSpace(body.Type)
	if providerType == "" {
		providerType = multillm.ProviderTypeOpenAICompatible
	}

	provider := multillm.Provider{
		Name:    name,
		Type:    providerType,
		BaseURL: strings.TrimSpace(body.BaseURL),
	}

	// Validate the endpoint: external providers must use HTTPS; plain HTTP is
	// accepted only for loopback services, and then only with the explicit
	// local flags multillm requires.
	if providerType != multillm.ProviderTypeCLI {
		u, err := url.Parse(provider.BaseURL)
		if err != nil || u.Host == "" || u.User != nil {
			w.WriteHeader(http.StatusBadRequest)
			return errors.New("o endpoint precisa ser uma URL válida, sem usuário/senha embutidos")
		}
		switch u.Scheme {
		case "https":
		case "http":
			if !isLoopbackHost(u.Hostname()) {
				w.WriteHeader(http.StatusBadRequest)
				return errors.New("endpoints HTTP só são aceitos para serviços locais (localhost/127.0.0.1); use HTTPS para provedores externos")
			}
			provider.AllowInsecureLoopback = true
			provider.AllowPrivate = true
		default:
			w.WriteHeader(http.StatusBadRequest)
			return errors.New("o endpoint precisa usar HTTPS (ou HTTP apenas para serviços locais)")
		}
	}

	// Build a unique, non-empty model list capped at 100 ids.
	seen := map[string]bool{}
	for _, raw := range body.Models {
		id := strings.TrimSpace(raw)
		if id == "" || strings.ContainsAny(id, "\r\n\x00") || seen[id] {
			continue
		}
		seen[id] = true
		provider.Models = append(provider.Models, multillm.ModelConfig{ID: id, Capabilities: []string{"chat"}})
	}
	if len(provider.Models) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("informe ao menos um ID de modelo")
	}
	if len(provider.Models) > 100 {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("máximo de 100 modelos por provedor")
	}

	cfg, err := loadProviderConfig()
	if err != nil {
		return err
	}
	if _, exists := findProvider(cfg, name); exists {
		w.WriteHeader(http.StatusConflict)
		return fmt.Errorf("já existe um provedor chamado %q", name)
	}

	// Resolve the credential variable. When a key is supplied (or an explicit
	// variable is requested) we need a valid, non-ambiguous name.
	apiKey := strings.TrimSpace(body.APIKey)
	envName := strings.TrimSpace(body.APIKeyEnv)
	if envName == "" && apiKey != "" {
		envName = deriveProviderEnvName(name)
		if envName == "" {
			w.WriteHeader(http.StatusBadRequest)
			return errors.New("não foi possível gerar um nome de variável para a chave; informe api_key_env")
		}
	}
	if envName != "" {
		if !providerEnvNamePattern.MatchString(envName) {
			w.WriteHeader(http.StatusBadRequest)
			return errors.New("nome de variável de chave inválido (use letras maiúsculas, dígitos e _, começando por letra)")
		}
		for _, p := range cfg.Providers {
			if p.APIKeyEnv == envName {
				w.WriteHeader(http.StatusConflict)
				return fmt.Errorf("a variável de chave %q já é usada por outro provedor; informe api_key_env diferente", envName)
			}
		}
		provider.APIKeyEnv = envName
	}

	// Reject an invalid key before touching the config so a bad paste never
	// leaves a half-written provider; secrets.Save re-validates on write.
	if apiKey != "" && (len(apiKey) > secrets.MaxKeyBytes || strings.ContainsAny(apiKey, "\r\n\x00")) {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("a chave de API precisa ser uma única linha não vazia")
	}

	// Validate the entire resulting config BEFORE writing anything.
	next := cfg
	next.Providers = append(append([]multillm.Provider(nil), cfg.Providers...), provider)
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if _, err := multillm.LoadBytes(data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("configuração de provedor inválida: %w", err)
	}
	if err := writeProviderConfig(path, next); err != nil {
		return err
	}

	// Store the key only after the config is persisted; never log its value.
	if apiKey != "" {
		dir, err := providerSecretsDir()
		if err != nil {
			return err
		}
		if _, err := secrets.Save(dir, envName, apiKey); err != nil {
			if errors.Is(err, secrets.ErrInvalid) {
				w.WriteHeader(http.StatusBadRequest)
			}
			return err
		}
	}
	s.log().Info("provider created", "provider", name, "type", providerType, "models", len(provider.Models), "needs_key", envName != "")
	s.restartForProviders()

	// Report the created provider without ever echoing the key.
	status := ProviderStatus{
		Name:       provider.Name,
		Type:       provider.Type,
		BaseURL:    provider.BaseURL,
		APIKeyEnv:  provider.APIKeyEnv,
		Models:     len(provider.Models),
		Enabled:    true,
		Configured: provider.APIKeyEnv == "" || secrets.Configured(provider.APIKeyEnv),
		NeedsKey:   provider.APIKeyEnv != "",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(status)
}

// providerKeyEnv resolves the credential variable for a provider declared in
// the config; arbitrary variable names are never accepted from the client.
func providerKeyEnv(name string) (string, error) {
	cfg, err := loadProviderConfig()
	if err != nil {
		return "", err
	}
	for _, p := range cfg.Providers {
		if p.Name == name {
			if p.APIKeyEnv == "" {
				return "", fmt.Errorf("provider %q does not use an API key", name)
			}
			return p.APIKeyEnv, nil
		}
	}
	return "", fmt.Errorf("unknown provider %q", name)
}

func (s *Server) setProviderKey(w http.ResponseWriter, r *http.Request) error {
	envName, err := providerKeyEnv(r.PathValue("name"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return err
	}
	var body struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, secrets.MaxKeyBytes+1024)).Decode(&body); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	dir, err := providerSecretsDir()
	if err != nil {
		return err
	}
	if _, err := secrets.Save(dir, envName, body.Key); err != nil {
		if errors.Is(err, secrets.ErrInvalid) {
			w.WriteHeader(http.StatusBadRequest)
		}
		return err
	}
	s.log().Info("provider credential saved", "provider", r.PathValue("name"))
	s.restartForProviders()
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) removeProviderKey(w http.ResponseWriter, r *http.Request) error {
	envName, err := providerKeyEnv(r.PathValue("name"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return err
	}
	dir, err := providerSecretsDir()
	if err != nil {
		return err
	}
	if err := secrets.Remove(dir, envName); err != nil {
		return err
	}
	s.log().Info("provider credential removed", "provider", r.PathValue("name"))
	s.restartForProviders()
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// restartForProviders reloads the Ollama server so the router re-reads
// credentials; it inherits the updated <ENV>_FILE from this process.
func (s *Server) restartForProviders() {
	if s.Restart != nil {
		go s.Restart()
	}
}

// AdoptProviderCredentials publishes <ENV>_FILE for keys saved by this app
// before the Ollama server starts, so the router sees them even when the
// app was launched without the user's updated environment.
func AdoptProviderCredentials() {
	cfg, err := loadProviderConfig()
	if err != nil {
		return
	}
	dir, err := providerSecretsDir()
	if err != nil {
		return
	}
	for _, p := range cfg.Providers {
		if p.APIKeyEnv != "" {
			if !secrets.Adopt(dir, p.APIKeyEnv) {
				// Keys saved by older builds used an OLLAMA_DZ23_ prefix.
				secrets.AdoptFrom(dir, "OLLAMA_DZ23_"+p.APIKeyEnv, p.APIKeyEnv)
			}
		}
	}
}

func findProvider(cfg multillm.Config, name string) (int, bool) {
	for i, p := range cfg.Providers {
		if p.Name == name {
			return i, true
		}
	}
	return -1, false
}

// listProviderModels returns the models configured for a provider and the
// models its API reports as available for the saved credential.
func (s *Server) listProviderModels(w http.ResponseWriter, r *http.Request) error {
	cfg, err := loadProviderConfig()
	if err != nil {
		return err
	}
	i, ok := findProvider(cfg, r.PathValue("name"))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return fmt.Errorf("unknown provider %q", r.PathValue("name"))
	}
	provider := cfg.Providers[i]
	configured := make([]string, 0, len(provider.Models))
	for _, m := range provider.Models {
		configured = append(configured, m.ID)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	available, err := multillm.ListUpstreamModels(ctx, provider, &http.Client{Timeout: 20 * time.Second})
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{"configured": configured, "available": available}
	if err != nil {
		resp["available"] = []string{}
		resp["error"] = err.Error()
	}
	return json.NewEncoder(w).Encode(resp)
}

// setProviderModels replaces the provider's model list in the config file,
// keeping the settings of models that stay selected, and restarts the server.
func (s *Server) setProviderModels(w http.ResponseWriter, r *http.Request) error {
	path := providerConfigPath()
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("OLLAMA_DZ23_CONFIG is not set")
	}
	var body struct {
		Models []string `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	cfg, err := loadProviderConfig()
	if err != nil {
		return err
	}
	i, ok := findProvider(cfg, r.PathValue("name"))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return fmt.Errorf("unknown provider %q", r.PathValue("name"))
	}
	models, err := mergeProviderModels(cfg.Providers[i].Models, body.Models)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	cfg.Providers[i].Models = models
	if err := writeProviderConfig(path, cfg); err != nil {
		return err
	}
	s.log().Info("provider models updated", "provider", r.PathValue("name"), "count", len(models))
	s.restartForProviders()
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// mergeProviderModels keeps existing entries (capabilities, priority, cost)
// for ids that remain selected and adds new ids as chat models.
func mergeProviderModels(current []multillm.ModelConfig, selected []string) ([]multillm.ModelConfig, error) {
	if len(selected) > 500 {
		return nil, errors.New("too many models selected")
	}
	byID := make(map[string]multillm.ModelConfig, len(current))
	for _, m := range current {
		byID[m.ID] = m
	}
	out := make([]multillm.ModelConfig, 0, len(selected))
	seen := map[string]bool{}
	for _, raw := range selected {
		id := strings.TrimSpace(raw)
		if id == "" || len(id) > 200 || strings.ContainsAny(id, "\r\n\x00") || seen[id] {
			continue
		}
		seen[id] = true
		if existing, ok := byID[id]; ok {
			out = append(out, existing)
		} else {
			out = append(out, multillm.ModelConfig{ID: id, Capabilities: []string{"chat"}})
		}
	}
	if len(out) == 0 {
		return nil, errors.New("select at least one model")
	}
	slices.SortStableFunc(out, func(a, b multillm.ModelConfig) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

func writeProviderConfig(path string, cfg multillm.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if _, err := multillm.LoadBytes(data); err != nil {
		return fmt.Errorf("refusing to write an invalid provider config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
