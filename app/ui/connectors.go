//go:build windows || darwin

package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ollama/ollama/app/secrets"
	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/internal/multillm"
)

// errAgentLoginRequired mirrors the agent API's answer when the server
// listens beyond loopback and demands a bearer token.
var errAgentLoginRequired = errors.New(`o Ollama está exposto na rede, então conectar exige login no Workspace; desative "Expose Ollama to the network" em Configurações para conectar localmente`)

func quickConnectEntry(id string) (agent.ConnectorCatalogEntry, bool) {
	for _, entry := range agent.ConnectorCatalog() {
		if entry.ID == id && entry.QuickConnect {
			return entry, true
		}
	}
	return agent.ConnectorCatalogEntry{}, false
}

// callAgentAPI sends a request to the local Ollama server's agent API.
func callAgentAPI(ctx context.Context, method, path string, body any, authorization, organization string) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(envconfig.ConnectableHost().String(), "/")+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if authorization = strings.TrimSpace(authorization); authorization != "" {
		if len(authorization) > 8192 || len(authorization) < len("Bearer ") || !strings.EqualFold(authorization[:len("Bearer ")], "Bearer ") || strings.TrimSpace(authorization[len("Bearer "):]) == "" {
			return errors.New("invalid agent authorization header")
		}
		req.Header.Set("Authorization", authorization)
	}
	if organization = strings.TrimSpace(organization); organization != "" {
		if len(organization) > 256 || strings.ContainsAny(organization, "\r\n\x00") {
			return errors.New("invalid agent organization header")
		}
		req.Header.Set("X-Ollama-Organization", organization)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return errAgentLoginRequired
	}
	if resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		var parsed struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(detail, &parsed) == nil && parsed.Error != "" {
			return fmt.Errorf("agent API %d: %s", resp.StatusCode, parsed.Error)
		}
		return fmt.Errorf("agent API %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	}
	return nil
}

// selfHostedBaseURL validates the address of a user's own instance.
func selfHostedBaseURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return "", errors.New("informe a URL https da sua instância, por exemplo https://meu-servidor.com/api")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

// connectConnector saves a pasted API key for a catalog service and
// registers the connector in the agent runtime in one step.
func (s *Server) connectConnector(w http.ResponseWriter, r *http.Request) error {
	entry, ok := quickConnectEntry(r.PathValue("id"))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return fmt.Errorf("connector %q cannot be connected with an API key", r.PathValue("id"))
	}
	var body struct {
		Key     string `json:"key"`
		BaseURL string `json:"base_url"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, secrets.MaxKeyBytes+1024)).Decode(&body); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	var err error
	baseURL := entry.APIBaseURL
	if entry.APISelfHosted {
		baseURL, err = selfHostedBaseURL(body.BaseURL)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return err
		}
	}
	dir, err := providerSecretsDir()
	if err != nil {
		return err
	}
	envName := agent.ConnectorTokenEnv(entry.ID)
	authorization := r.Header.Get("Authorization")
	organization := r.Header.Get("X-Ollama-Organization")
	operations := agent.QuickConnectOperations(entry.ID)
	if len(operations) == 0 {
		w.WriteHeader(http.StatusForbidden)
		return errors.New("connector has no approved quick-connect policy")
	}
	// Check auth and API availability before persisting a user credential.
	if err := callAgentAPI(r.Context(), http.MethodGet, "/api/agent/v1/connectors", nil, authorization, organization); err != nil {
		if errors.Is(err, errAgentLoginRequired) {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusBadGateway)
		}
		return err
	}
	previousCredential := multillm.CredentialValue(envName)
	_, defaultCredentialErr := os.Stat(secrets.Path(dir, envName))
	if defaultCredentialErr != nil && !errors.Is(defaultCredentialErr, os.ErrNotExist) {
		w.WriteHeader(http.StatusConflict)
		return fmt.Errorf("inspect existing connector credential safely: %w", defaultCredentialErr)
	}
	previousConfigured := secrets.Configured(envName) || defaultCredentialErr == nil
	if previousConfigured && previousCredential == "" {
		w.WriteHeader(http.StatusConflict)
		return errors.New("existing connector credential cannot be read safely; refusing to overwrite")
	}
	restoreCredential := func() error {
		if previousConfigured && previousCredential != "" {
			_, restoreErr := secrets.Save(dir, envName, previousCredential)
			return restoreErr
		}
		return secrets.Remove(dir, envName)
	}
	if savedPath, saveErr := secrets.Save(dir, envName, body.Key); saveErr != nil {
		if savedPath != "" {
			if rollbackErr := restoreCredential(); rollbackErr != nil {
				return fmt.Errorf("save connector credential: %w; credential rollback failed: %v", saveErr, rollbackErr)
			}
		}
		if errors.Is(saveErr, secrets.ErrInvalid) {
			w.WriteHeader(http.StatusBadRequest)
		}
		return saveErr
	}
	err = callAgentAPI(r.Context(), http.MethodPost, "/api/agent/v1/connectors", agent.ConnectorConfig{
		ID:         entry.ID,
		Provider:   entry.ID,
		BaseURL:    baseURL,
		TokenEnv:   envName,
		AuthHeader: entry.APIAuthHeader,
		AuthScheme: entry.APIAuthScheme,
		Operations: operations,
	}, authorization, organization)
	if err != nil {
		if rollbackErr := restoreCredential(); rollbackErr != nil {
			return fmt.Errorf("register connector: %w; credential rollback failed: %v", err, rollbackErr)
		}
		if errors.Is(err, errAgentLoginRequired) {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusBadGateway)
		}
		return err
	}
	s.log().Info("connector connected", "connector", entry.ID)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// disconnectConnector removes the connector registration and its key.
func (s *Server) disconnectConnector(w http.ResponseWriter, r *http.Request) error {
	entry, ok := quickConnectEntry(r.PathValue("id"))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return fmt.Errorf("unknown connector %q", r.PathValue("id"))
	}
	if err := callAgentAPI(r.Context(), http.MethodDelete, "/api/agent/v1/connectors/"+entry.ID, nil, r.Header.Get("Authorization"), r.Header.Get("X-Ollama-Organization")); err != nil && !strings.Contains(err.Error(), "404") {
		if errors.Is(err, errAgentLoginRequired) {
			w.WriteHeader(http.StatusForbidden)
		}
		return err
	}
	dir, err := providerSecretsDir()
	if err != nil {
		return err
	}
	if err := secrets.Remove(dir, agent.ConnectorTokenEnv(entry.ID)); err != nil {
		return err
	}
	s.log().Info("connector disconnected", "connector", entry.ID)
	w.WriteHeader(http.StatusNoContent)
	return nil
}
