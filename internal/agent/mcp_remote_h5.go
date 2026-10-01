package agent

// This file is intentionally kept separate from the transport implementation so
// the protocol state machine can be audited and tested without a live IdP.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func randomRemoteMCPValue(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func remoteMCPAuthContext(ctx context.Context, rawURL string) context.Context {
	parsed, err := url.Parse(rawURL)
	if err == nil && remoteMCPLoopback(parsed.Hostname()) {
		return context.WithValue(ctx, remoteMCPLoopbackContextKey{}, true)
	}
	return ctx
}

// BeginOAuth creates a PKCE authorization-code request. The verifier is kept
// only in memory and the state is one-time/short-lived.
func (m *RemoteMCPManager) BeginOAuth(serverID, organizationID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	config, ok := m.servers[strings.TrimSpace(serverID)]
	if !ok || config.OAuth == nil {
		return "", ErrRemoteMCPNotConfigured
	}
	if !pluginAccessibleByOrganization(config.OrganizationID, strings.TrimSpace(organizationID)) {
		return "", ErrPluginOrganizationScope
	}
	verifier, err := randomRemoteMCPValue(32)
	if err != nil {
		return "", err
	}
	state, err := randomRemoteMCPValue(24)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	m.oauthStates[state] = remoteMCPAuthState{ServerID: serverID, OrganizationID: organizationID, Verifier: verifier, State: state, ExpiresAt: now.Add(10 * time.Minute)}
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	parsed, err := url.Parse(config.OAuth.AuthorizationURL)
	if err != nil {
		return "", err
	}
	values := parsed.Query()
	values.Set("response_type", "code")
	values.Set("client_id", os.Getenv(config.OAuth.ClientIDEnv))
	values.Set("redirect_uri", config.OAuth.RedirectURI)
	values.Set("state", state)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	if len(config.OAuth.Scopes) > 0 {
		values.Set("scope", strings.Join(config.OAuth.Scopes, " "))
	}
	parsed.RawQuery = values.Encode()
	return parsed.String(), nil
}

func (m *RemoteMCPManager) CompleteOAuth(ctx context.Context, state, code string) error {
	m.mu.Lock()
	authState, ok := m.oauthStates[strings.TrimSpace(state)]
	if ok {
		delete(m.oauthStates, state)
	}
	config, configOK := m.servers[authState.ServerID]
	client := m.client
	m.mu.Unlock()
	if !ok || !configOK || time.Now().UTC().After(authState.ExpiresAt) || strings.TrimSpace(code) == "" {
		return ErrRemoteMCPOAuthStateInvalid
	}
	if config.OAuth == nil {
		return ErrRemoteMCPNotConfigured
	}
	clientID, ok := os.LookupEnv(config.OAuth.ClientIDEnv)
	if !ok || strings.TrimSpace(clientID) == "" {
		return ErrRemoteMCPNotConfigured
	}
	values := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {clientID}, "redirect_uri": {config.OAuth.RedirectURI}, "code_verifier": {authState.Verifier}}
	if config.OAuth.ClientSecretEnv != "" {
		values.Set("client_secret", os.Getenv(config.OAuth.ClientSecretEnv))
	}
	req, err := http.NewRequestWithContext(remoteMCPAuthContext(ctx, config.OAuth.TokenURL), http.MethodPost, config.OAuth.TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(req)
	if err != nil {
		return errors.New("remote MCP OAuth token request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("remote MCP OAuth token endpoint returned HTTP status %d", response.StatusCode)
	}
	var token struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&token); err != nil || strings.TrimSpace(token.AccessToken) == "" {
		return errors.New("remote MCP OAuth token response is invalid")
	}
	ttl := time.Duration(token.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	m.mu.Lock()
	m.oauthTokens[authState.ServerID] = remoteMCPOAuthToken{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, ExpiresAt: time.Now().UTC().Add(ttl)}
	m.mu.Unlock()
	return nil
}

func (m *RemoteMCPManager) refreshOAuth(ctx context.Context, serverID string, config RemoteMCPServerConfig, token remoteMCPOAuthToken) (remoteMCPOAuthToken, error) {
	if config.OAuth == nil || strings.TrimSpace(token.RefreshToken) == "" {
		return remoteMCPOAuthToken{}, ErrRemoteMCPNotConfigured
	}
	clientID, ok := os.LookupEnv(config.OAuth.ClientIDEnv)
	if !ok || strings.TrimSpace(clientID) == "" {
		return remoteMCPOAuthToken{}, ErrRemoteMCPNotConfigured
	}
	values := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token.RefreshToken}, "client_id": {clientID}}
	if config.OAuth.ClientSecretEnv != "" {
		values.Set("client_secret", os.Getenv(config.OAuth.ClientSecretEnv))
	}
	req, err := http.NewRequestWithContext(remoteMCPAuthContext(ctx, config.OAuth.TokenURL), http.MethodPost, config.OAuth.TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return remoteMCPOAuthToken{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := m.client.Do(req)
	if err != nil {
		return remoteMCPOAuthToken{}, errors.New("remote MCP OAuth refresh request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return remoteMCPOAuthToken{}, fmt.Errorf("remote MCP OAuth refresh returned HTTP status %d", response.StatusCode)
	}
	var refreshed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&refreshed); err != nil || strings.TrimSpace(refreshed.AccessToken) == "" {
		return remoteMCPOAuthToken{}, errors.New("remote MCP OAuth refresh response is invalid")
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = token.RefreshToken
	}
	ttl := time.Duration(refreshed.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	updated := remoteMCPOAuthToken{AccessToken: refreshed.AccessToken, RefreshToken: refreshed.RefreshToken, ExpiresAt: time.Now().UTC().Add(ttl)}
	m.mu.Lock()
	m.oauthTokens[serverID] = updated
	m.mu.Unlock()
	return updated, nil
}

func (m *RemoteMCPManager) setOAuthTokenForTest(serverID string, token remoteMCPOAuthToken) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.oauthTokens[serverID] = token
}

func (m *RemoteMCPManager) accessToken(ctx context.Context, serverID string, config RemoteMCPServerConfig) (string, error) {
	if config.OAuth == nil {
		if config.TokenEnv == "" {
			return "", nil
		}
		value, ok := os.LookupEnv(config.TokenEnv)
		if !ok || strings.TrimSpace(value) == "" {
			return "", ErrRemoteMCPNotConfigured
		}
		return strings.TrimSpace(value), nil
	}
	m.mu.RLock()
	token, ok := m.oauthTokens[serverID]
	m.mu.RUnlock()
	if !ok || strings.TrimSpace(token.AccessToken) == "" {
		return "", ErrRemoteMCPNotConfigured
	}
	if time.Now().UTC().Before(token.ExpiresAt.Add(-30 * time.Second)) {
		return token.AccessToken, nil
	}
	updated, err := m.refreshOAuth(ctx, serverID, config, token)
	if err != nil {
		return "", err
	}
	return updated.AccessToken, nil
}

func (m *RemoteMCPManager) rememberSession(serverID, organizationID, id string, expiry time.Time) {
	if strings.TrimSpace(id) == "" {
		return
	}
	if expiry.IsZero() {
		expiry = time.Now().UTC().Add(30 * time.Minute)
	}
	m.mu.Lock()
	m.sessions[id] = remoteMCPSession{ID: id, ServerID: serverID, OrganizationID: organizationID, ExpiresAt: expiry}
	m.mu.Unlock()
}

func (m *RemoteMCPManager) InvalidateSession(id string) {
	m.mu.Lock()
	delete(m.sessions, strings.TrimSpace(id))
	m.mu.Unlock()
}

func (m *RemoteMCPManager) Status(id string) GateStatus {
	m.mu.RLock()
	config, ok := m.servers[strings.TrimSpace(id)]
	token, hasToken := m.oauthTokens[strings.TrimSpace(id)]
	m.mu.RUnlock()
	if !ok || config.Disabled {
		return GateStatusNotConfigured
	}
	if config.OAuth != nil && (!hasToken || strings.TrimSpace(token.AccessToken) == "") {
		return GateStatusNotConfigured
	}
	if config.TokenEnv != "" {
		if value, exists := os.LookupEnv(config.TokenEnv); !exists || strings.TrimSpace(value) == "" {
			return GateStatusNotConfigured
		}
	}
	return GateStatusPass
}

func (m *RemoteMCPManager) sessionFor(serverID, organizationID, id string) (remoteMCPSession, error) {
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok || session.ServerID != serverID || session.OrganizationID != organizationID || time.Now().UTC().After(session.ExpiresAt) {
		return remoteMCPSession{}, ErrRemoteMCPSessionInvalid
	}
	return session, nil
}

func (m *RemoteMCPManager) latestSession(serverID, organizationID string) (remoteMCPSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var latest remoteMCPSession
	for _, session := range m.sessions {
		if session.ServerID == serverID && session.OrganizationID == organizationID && time.Now().UTC().Before(session.ExpiresAt) && (latest.ExpiresAt.IsZero() || session.ExpiresAt.After(latest.ExpiresAt)) {
			latest = session
		}
	}
	return latest, !latest.ExpiresAt.IsZero()
}

func (m *RemoteMCPManager) BeginPairing(serverID, organizationID string) (string, error) {
	m.mu.RLock()
	config, ok := m.servers[serverID]
	m.mu.RUnlock()
	if !ok || config.PairingTokenEnv == "" {
		return "", ErrRemoteMCPNotConfigured
	}
	if !pluginAccessibleByOrganization(config.OrganizationID, organizationID) {
		return "", ErrPluginOrganizationScope
	}
	value, err := randomRemoteMCPValue(24)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.pairings[serverID+"\x00"+organizationID] = remoteMCPPairing{Challenge: value, ExpiresAt: time.Now().UTC().Add(5 * time.Minute)}
	m.mu.Unlock()
	return value, nil
}

func (m *RemoteMCPManager) CompletePairing(serverID, organizationID, presented, challenge string) error {
	m.mu.Lock()
	key := serverID + "\x00" + organizationID
	pairing, ok := m.pairings[key]
	if ok {
		delete(m.pairings, key)
	}
	m.mu.Unlock()
	if !ok || time.Now().UTC().After(pairing.ExpiresAt) {
		return ErrRemoteMCPPairingUnauthorized
	}
	m.mu.RLock()
	config, configOK := m.servers[serverID]
	m.mu.RUnlock()
	if !configOK || !pluginAccessibleByOrganization(config.OrganizationID, organizationID) {
		return ErrRemoteMCPPairingUnauthorized
	}
	expected, ok := os.LookupEnv(config.PairingTokenEnv)
	if !ok || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(expected)), []byte(strings.TrimSpace(presented))) != 1 || subtle.ConstantTimeCompare([]byte(pairing.Challenge), []byte(strings.TrimSpace(challenge))) != 1 {
		return ErrRemoteMCPPairingUnauthorized
	}
	return nil
}
