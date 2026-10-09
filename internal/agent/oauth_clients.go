package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// OAuthClientStore keeps the OAuth *application* credentials (client_id and
// client_secret) that the operator registers per provider so the server can
// run the OAuth flow without the values living in process environment at
// startup. It is a small JSON file in the data root, written 0600. The secret
// is never logged or returned in any API response.
type OAuthClientStore struct {
	mu      sync.RWMutex
	path    string
	entries map[string]oauthClientEntry
}

type oauthClientEntry struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// NewOAuthClientStore loads (or lazily creates) the client-credential file at
// <root>/oauth-clients.json. An empty root keeps the store in memory only.
func NewOAuthClientStore(root string) (*OAuthClientStore, error) {
	store := &OAuthClientStore{entries: map[string]oauthClientEntry{}}
	root = strings.TrimSpace(root)
	if root == "" {
		return store, nil
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	store.path = filepath.Join(root, "oauth-clients.json")
	data, err := os.ReadFile(store.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return store, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return store, nil
	}
	loaded := map[string]oauthClientEntry{}
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, err
	}
	for name, entry := range loaded {
		store.entries[normalizeOAuthProvider(name)] = entry
	}
	return store, nil
}

func normalizeOAuthProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

// Save stores the application client_id/client_secret for a provider. Both
// values are required. The file is rewritten atomically with mode 0600.
func (s *OAuthClientStore) Save(provider, clientID, clientSecret string) error {
	if s == nil {
		return errors.New("oauth client store is unavailable")
	}
	provider = normalizeOAuthProvider(provider)
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	if provider == "" {
		return errors.New("oauth provider is required")
	}
	if clientID == "" || clientSecret == "" {
		return errors.New("client_id and client_secret are required")
	}
	if strings.ContainsAny(clientID, "\r\n\x00") || strings.ContainsAny(clientSecret, "\r\n\x00") {
		return errors.New("client_id and client_secret must be single-line values")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, existed := s.entries[provider]
	s.entries[provider] = oauthClientEntry{ClientID: clientID, ClientSecret: clientSecret}
	if err := s.persistLocked(); err != nil {
		if existed {
			s.entries[provider] = previous
		} else {
			delete(s.entries, provider)
		}
		return err
	}
	return nil
}

// Get returns the stored client_id/client_secret for a provider. The boolean
// reports whether a configured entry exists.
func (s *OAuthClientStore) Get(provider string) (clientID, clientSecret string, ok bool) {
	if s == nil {
		return "", "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, found := s.entries[normalizeOAuthProvider(provider)]
	if !found || entry.ClientID == "" || entry.ClientSecret == "" {
		return "", "", false
	}
	return entry.ClientID, entry.ClientSecret, true
}

// Configured reports whether a provider has a stored client credential.
func (s *OAuthClientStore) Configured(provider string) bool {
	_, _, ok := s.Get(provider)
	return ok
}

// Providers returns the sorted list of configured provider names. It never
// includes secrets.
func (s *OAuthClientStore) Providers() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.entries))
	for name, entry := range s.entries {
		if entry.ClientID != "" && entry.ClientSecret != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (s *OAuthClientStore) persistLocked() error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".oauth-clients-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, s.path)
}
