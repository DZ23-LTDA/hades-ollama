package agent

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrOAuthCredentialConflict = errors.New("oauth credential changed during refresh")
	ErrOAuthCredentialRevoked  = errors.New("oauth credential is revoked")
)

type User struct {
	ID                      string    `json:"id"`
	Email                   string    `json:"email"`
	Name                    string    `json:"name"`
	MFAEnabled              bool      `json:"mfa_enabled,omitempty"`
	MFASecretCiphertext     string    `json:"mfa_secret_ciphertext,omitempty"`
	RecoveryCodesCiphertext string    `json:"recovery_codes_ciphertext,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
}

func (u User) Public() User {
	u.MFASecretCiphertext = ""
	u.RecoveryCodesCiphertext = ""
	return u
}

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Role string

const (
	RoleOwner    Role = "owner"
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
	RoleAuditor  Role = "auditor"
)

type Membership struct {
	UserID         string    `json:"user_id"`
	OrganizationID string    `json:"organization_id"`
	Role           Role      `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}

type AccessToken struct {
	Hash           string     `json:"hash"`
	UserID         string     `json:"user_id"`
	OrganizationID string     `json:"organization_id"`
	ExpiresAt      time.Time  `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

type OAuthState struct {
	Hash         string     `json:"hash"`
	Provider     string     `json:"provider"`
	RedirectURI  string     `json:"redirect_uri"`
	CodeVerifier string     `json:"code_verifier"`
	Nonce        string     `json:"nonce,omitempty"`
	UserID       string     `json:"user_id,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	ConsumedAt   *time.Time `json:"consumed_at,omitempty"`
}

type OAuthCredential struct {
	ID                     string     `json:"id"`
	UserID                 string     `json:"user_id"`
	OrganizationID         string     `json:"organization_id"`
	Provider               string     `json:"provider"`
	AccessTokenCiphertext  string     `json:"access_token_ciphertext"`
	RefreshTokenCiphertext string     `json:"refresh_token_ciphertext,omitempty"`
	ExpiresAt              time.Time  `json:"expires_at,omitempty"`
	UpdatedAt              time.Time  `json:"updated_at"`
	RevokedAt              *time.Time `json:"revoked_at,omitempty"`
}

type AuthStore struct {
	mu            sync.RWMutex
	root          string
	users         map[string]User
	organizations map[string]Organization
	memberships   map[string]Membership
	tokens        map[string]AccessToken
	oauthStates   map[string]OAuthState
	credentials   map[string]OAuthCredential
	mfaAttempts   map[string]MFAAttemptState
	// identities binds a federated sign-in ("provider|subject") to a local
	// user ID. Email alone is never the identity key (SEC-04): it can be
	// reassigned and is not unique across identity providers.
	identities map[string]string
}

const (
	mfaFailureLimit   = 5
	mfaFailureWindow  = 5 * time.Minute
	mfaLockoutPeriod  = 15 * time.Minute
	mfaStateRetention = 24 * time.Hour
)

type MFAAttemptState struct {
	Failures      int       `json:"failures"`
	WindowStarted time.Time `json:"window_started"`
	LockedUntil   time.Time `json:"locked_until,omitempty"`
	LastAttemptAt time.Time `json:"last_attempt_at"`
}

type MFAThrottleError struct {
	RetryAfter time.Duration
}

func (e *MFAThrottleError) Error() string {
	return "mfa verification temporarily locked"
}

func NewAuthStore(root string) (*AuthStore, error) {
	store := &AuthStore{root: strings.TrimSpace(root), users: map[string]User{}, organizations: map[string]Organization{}, memberships: map[string]Membership{}, tokens: map[string]AccessToken{}, oauthStates: map[string]OAuthState{}, credentials: map[string]OAuthCredential{}, mfaAttempts: map[string]MFAAttemptState{}, identities: map[string]string{}}
	if store.root == "" {
		return store, nil
	}
	if err := os.MkdirAll(store.root, 0o700); err != nil {
		return nil, err
	}
	for name, target := range map[string]any{"users": &store.users, "organizations": &store.organizations, "memberships": &store.memberships, "tokens": &store.tokens, "oauth-states": &store.oauthStates, "credentials": &store.credentials, "mfa-attempts": &store.mfaAttempts, "identities": &store.identities} {
		path := filepathJoin(store.root, name+".json")
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err := readJSON(path, target); err != nil {
			return nil, err
		}
	}
	if store.mfaAttempts == nil {
		store.mfaAttempts = map[string]MFAAttemptState{}
	}
	if store.identities == nil {
		store.identities = map[string]string{}
	}
	return store, nil
}

func (s *AuthStore) CreateUser(email, name string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	if email == "" || !strings.Contains(email, "@") {
		return User{}, errors.New("valid email is required")
	}
	if len(email) > 320 || len(name) > 200 {
		return User{}, errors.New("user field is too long")
	}
	now := time.Now().UTC()
	user := User{ID: "usr_" + uuid.NewString(), Email: email, Name: name, CreatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.users {
		if existing.Email == email {
			return existing, nil
		}
	}
	s.users[user.ID] = user
	return user, s.persistLocked()
}

func (s *AuthStore) CreateOrganization(name string, owner User) (Organization, Membership, error) {
	name = strings.TrimSpace(name)
	if name == "" || owner.ID == "" {
		return Organization{}, Membership{}, errors.New("organization name and owner are required")
	}
	now := time.Now().UTC()
	organization := Organization{ID: "org_" + uuid.NewString(), Name: name, CreatedAt: now}
	membership := Membership{UserID: owner.ID, OrganizationID: organization.ID, Role: RoleOwner, CreatedAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.organizations[organization.ID] = organization
	s.memberships[membershipKey(owner.ID, organization.ID)] = membership
	return organization, membership, s.persistLocked()
}

func (s *AuthStore) AddMembership(userID, organizationID string, role Role) (Membership, error) {
	if !validRole(role) {
		return Membership{}, errors.New("invalid organization role")
	}
	membership := Membership{UserID: strings.TrimSpace(userID), OrganizationID: strings.TrimSpace(organizationID), Role: role, CreatedAt: time.Now().UTC()}
	if membership.UserID == "" || membership.OrganizationID == "" {
		return Membership{}, errors.New("user and organization are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[membership.UserID]; !ok {
		return Membership{}, os.ErrNotExist
	}
	if _, ok := s.organizations[membership.OrganizationID]; !ok {
		return Membership{}, os.ErrNotExist
	}
	s.memberships[membershipKey(membership.UserID, membership.OrganizationID)] = membership
	return membership, s.persistLocked()
}

func (s *AuthStore) IssueToken(userID, organizationID string, ttl time.Duration) (string, AccessToken, error) {
	if ttl <= 0 || ttl > 30*24*time.Hour {
		ttl = 24 * time.Hour
	}
	if _, err := s.Authorize(userID, organizationID, "read"); err != nil {
		return "", AccessToken{}, err
	}
	raw, err := randomSecret(32)
	if err != nil {
		return "", AccessToken{}, err
	}
	now := time.Now().UTC()
	token := AccessToken{Hash: hashSecret(raw), UserID: userID, OrganizationID: organizationID, CreatedAt: now, ExpiresAt: now.Add(ttl)}
	s.mu.Lock()
	s.tokens[token.Hash] = token
	err = s.persistLocked()
	s.mu.Unlock()
	return raw, token, err
}

func (s *AuthStore) Authenticate(raw string) (User, Organization, Membership, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return User{}, Organization{}, Membership{}, errors.New("token is required")
	}
	s.mu.RLock()
	token, ok := s.tokens[hashSecret(raw)]
	user := s.users[token.UserID]
	organization := s.organizations[token.OrganizationID]
	membership := s.memberships[membershipKey(token.UserID, token.OrganizationID)]
	s.mu.RUnlock()
	if !ok || token.RevokedAt != nil || time.Now().UTC().After(token.ExpiresAt) || user.ID == "" || organization.ID == "" || membership.UserID == "" {
		return User{}, Organization{}, Membership{}, errors.New("invalid or expired token")
	}
	return user, organization, membership, nil
}

func (s *AuthStore) EnableMFA(userID, secret string) (User, error) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	secret = strings.TrimRight(secret, "=")
	if secret == "" {
		return User{}, errors.New("mfa secret is required")
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil || len(decoded) < 10 {
		return User{}, errors.New("mfa secret must be a valid base32 value")
	}
	ciphertext, err := encryptCredential(secret)
	if err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return User{}, os.ErrNotExist
	}
	user.MFAEnabled = true
	user.MFASecretCiphertext = ciphertext
	s.users[userID] = user
	return user, s.persistLocked()
}

func (s *AuthStore) DisableMFA(userID string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return User{}, os.ErrNotExist
	}
	user.MFAEnabled = false
	user.MFASecretCiphertext = ""
	s.users[userID] = user
	return user, s.persistLocked()
}

func (s *AuthStore) VerifyMFA(userID, code string, now time.Time) error {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return errors.New("mfa code must have six digits")
	}
	if _, err := strconv.Atoi(code); err != nil {
		return errors.New("mfa code must contain digits")
	}
	s.mu.RLock()
	user, ok := s.users[userID]
	s.mu.RUnlock()
	if !ok {
		return os.ErrNotExist
	}
	if !user.MFAEnabled {
		return nil
	}
	secret, err := decryptCredential(user.MFASecretCiphertext)
	if err != nil {
		return err
	}
	counter := now.UTC().Unix() / 30
	for offset := int64(-1); offset <= 1; offset++ {
		if hmac.Equal([]byte(code), []byte(totpCode(secret, counter+offset))) {
			return nil
		}
	}
	return errors.New("invalid mfa code")
}

func (s *AuthStore) VerifyMFAWithThrottle(userID, code, clientKey string, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	key := mfaAttemptKey(userID, clientKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	if throttle := s.mfaThrottleLocked(key, now); throttle != nil {
		return throttle
	}
	user, ok := s.users[userID]
	if !ok {
		return os.ErrNotExist
	}
	if !user.MFAEnabled {
		return nil
	}
	secret, err := decryptCredential(user.MFASecretCiphertext)
	if err == nil {
		err = verifyTOTP(secret, code, now)
	}
	if err != nil {
		return s.recordMFAFailureLocked(key, now, err)
	}
	if err := s.clearMFAFailureLocked(key); err != nil {
		return err
	}
	return nil
}

func verifyTOTP(secret, code string, now time.Time) error {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return errors.New("mfa code must have six digits")
	}
	if _, err := strconv.Atoi(code); err != nil {
		return errors.New("mfa code must contain digits")
	}
	counter := now.Unix() / 30
	for offset := int64(-1); offset <= 1; offset++ {
		if hmac.Equal([]byte(code), []byte(totpCode(secret, counter+offset))) {
			return nil
		}
	}
	return errors.New("invalid mfa code")
}

func mfaAttemptKey(userID, clientKey string) string {
	return strings.TrimSpace(userID) + ":" + hashSecret(strings.TrimSpace(clientKey))
}

func (s *AuthStore) mfaThrottleLocked(key string, now time.Time) *MFAThrottleError {
	state, ok := s.mfaAttempts[key]
	if !ok {
		return nil
	}
	if state.LockedUntil.After(now) {
		return &MFAThrottleError{RetryAfter: state.LockedUntil.Sub(now)}
	}
	if !state.LockedUntil.IsZero() || (!state.LastAttemptAt.IsZero() && now.Sub(state.LastAttemptAt) > mfaStateRetention) {
		delete(s.mfaAttempts, key)
	}
	if !state.WindowStarted.IsZero() && now.Sub(state.WindowStarted) >= mfaFailureWindow {
		delete(s.mfaAttempts, key)
	}
	return nil
}

func (s *AuthStore) recordMFAFailureLocked(key string, now time.Time, cause error) error {
	previous, hadPrevious := s.mfaAttempts[key]
	state := previous
	if state.WindowStarted.IsZero() || now.Sub(state.WindowStarted) >= mfaFailureWindow {
		state = MFAAttemptState{WindowStarted: now}
	}
	state.Failures++
	state.LastAttemptAt = now
	if state.Failures >= mfaFailureLimit {
		state.LockedUntil = now.Add(mfaLockoutPeriod)
	}
	s.mfaAttempts[key] = state
	if err := s.persistLocked(); err != nil {
		if hadPrevious {
			s.mfaAttempts[key] = previous
		} else {
			delete(s.mfaAttempts, key)
		}
		return err
	}
	if state.LockedUntil.After(now) {
		return &MFAThrottleError{RetryAfter: state.LockedUntil.Sub(now)}
	}
	return cause
}

func (s *AuthStore) clearMFAFailureLocked(key string) error {
	previous, ok := s.mfaAttempts[key]
	if !ok {
		return nil
	}
	delete(s.mfaAttempts, key)
	if err := s.persistLocked(); err != nil {
		s.mfaAttempts[key] = previous
		return err
	}
	return nil
}

func (s *AuthStore) GenerateRecoveryCodes(userID string) (User, []string, error) {
	codes := make([]string, 10)
	for index := range codes {
		raw, err := randomSecret(6)
		if err != nil {
			return User{}, nil, err
		}
		codes[index] = strings.ToUpper(raw[:4] + "-" + raw[4:])
	}
	ciphertext, err := encryptCredential(strings.Join(codes, "\n"))
	if err != nil {
		return User{}, nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return User{}, nil, os.ErrNotExist
	}
	if !user.MFAEnabled {
		return User{}, nil, errors.New("mfa must be enabled before generating recovery codes")
	}
	user.RecoveryCodesCiphertext = ciphertext
	s.users[userID] = user
	if err := s.persistLocked(); err != nil {
		return User{}, nil, err
	}
	return user.Public(), codes, nil
}

func (s *AuthStore) VerifyRecoveryCode(userID, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.verifyRecoveryCodeLocked(userID, code); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *AuthStore) VerifyRecoveryCodeWithThrottle(userID, code, clientKey string, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	key := mfaAttemptKey(userID, clientKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	if throttle := s.mfaThrottleLocked(key, now); throttle != nil {
		return throttle
	}
	previousUser, userExists := s.users[userID]
	if err := s.verifyRecoveryCodeLocked(userID, code); err != nil {
		return s.recordMFAFailureLocked(key, now, err)
	}
	previousAttempt, hadAttempt := s.mfaAttempts[key]
	delete(s.mfaAttempts, key)
	if err := s.persistLocked(); err != nil {
		if userExists {
			s.users[userID] = previousUser
		}
		if hadAttempt {
			s.mfaAttempts[key] = previousAttempt
		} else {
			delete(s.mfaAttempts, key)
		}
		return err
	}
	return nil
}

func (s *AuthStore) verifyRecoveryCodeLocked(userID, code string) error {
	code = normalizeRecoveryCode(code)
	if code == "" {
		return errors.New("recovery code is required")
	}
	user, ok := s.users[userID]
	if !ok {
		return os.ErrNotExist
	}
	plain, err := decryptCredential(user.RecoveryCodesCiphertext)
	if err != nil {
		return errors.New("recovery code is unavailable")
	}
	remaining := make([]string, 0, 10)
	matched := false
	for _, candidate := range strings.Split(plain, "\n") {
		candidate = normalizeRecoveryCode(candidate)
		if candidate == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(code)) == 1 && !matched {
			matched = true
			continue
		}
		remaining = append(remaining, candidate)
	}
	if !matched {
		return errors.New("invalid recovery code")
	}
	if len(remaining) == 0 {
		user.RecoveryCodesCiphertext = ""
	} else {
		user.RecoveryCodesCiphertext, err = encryptCredential(strings.Join(remaining, "\n"))
		if err != nil {
			return err
		}
	}
	s.users[userID] = user
	return nil
}

func normalizeRecoveryCode(value string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), "-", ""))
}

func totpCode(secret string, counter int64) string {
	decoded, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.TrimRight(strings.ToUpper(strings.TrimSpace(secret)), "="))
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], uint64(counter))
	digest := hmac.New(sha1.New, decoded)
	_, _ = digest.Write(message[:])
	sum := digest.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 | uint32(sum[offset+1])<<16 | uint32(sum[offset+2])<<8 | uint32(sum[offset+3])
	return fmt.Sprintf("%06d", value%1000000)
}

func (s *AuthStore) RevokeToken(raw string) error {
	hash := hashSecret(strings.TrimSpace(raw))
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[hash]
	if !ok {
		return os.ErrNotExist
	}
	token.RevokedAt = &now
	s.tokens[hash] = token
	return s.persistLocked()
}

func (s *AuthStore) Authorize(userID, organizationID, action string) (Membership, error) {
	s.mu.RLock()
	membership, ok := s.memberships[membershipKey(userID, organizationID)]
	s.mu.RUnlock()
	if !ok {
		return Membership{}, errors.New("user is not a member of organization")
	}
	if !roleAllows(membership.Role, action) {
		return Membership{}, fmt.Errorf("role %s cannot perform %s", membership.Role, action)
	}
	return membership, nil
}

func (s *AuthStore) CreateOAuthState(provider, redirectURI, codeVerifier, userID string, ttl time.Duration) (string, OAuthState, error) {
	return s.CreateOAuthStateWithNonce(provider, redirectURI, codeVerifier, "", userID, ttl)
}

func (s *AuthStore) CreateOAuthStateWithNonce(provider, redirectURI, codeVerifier, nonce, userID string, ttl time.Duration) (string, OAuthState, error) {
	provider = strings.TrimSpace(provider)
	canonicalRedirectURI, err := validateOAuthRedirectSyntax(redirectURI, false)
	if err != nil {
		return "", OAuthState{}, err
	}
	if len(codeVerifier) < 43 || len(codeVerifier) > 128 {
		return "", OAuthState{}, errors.New("code_verifier must be PKCE length")
	}
	if ttl <= 0 || ttl > 10*time.Minute {
		ttl = 5 * time.Minute
	}
	raw, err := randomSecret(32)
	if err != nil {
		return "", OAuthState{}, err
	}
	state := OAuthState{Hash: hashSecret(raw), Provider: provider, RedirectURI: canonicalRedirectURI, CodeVerifier: codeVerifier, Nonce: strings.TrimSpace(nonce), UserID: userID, ExpiresAt: time.Now().UTC().Add(ttl)}
	s.mu.Lock()
	// Drop stale/consumed states and cap the total so an attacker cannot grow
	// the persisted store without bound by repeatedly starting OAuth (SEC-06).
	s.sweepOAuthStatesLocked(time.Now().UTC())
	if len(s.oauthStates) >= maxPendingOAuthStates {
		s.mu.Unlock()
		return "", OAuthState{}, errors.New("too many pending oauth states; try again later")
	}
	s.oauthStates[state.Hash] = state
	err = s.persistLocked()
	s.mu.Unlock()
	return raw, state, err
}

func (s *AuthStore) ConsumeOAuthState(raw, provider, redirectURI string) (OAuthState, error) {
	hash := hashSecret(strings.TrimSpace(raw))
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.oauthStates[hash]
	if !ok || state.Provider != provider || state.RedirectURI != redirectURI || state.ConsumedAt != nil || time.Now().UTC().After(state.ExpiresAt) {
		return OAuthState{}, errors.New("invalid, expired, or already consumed oauth state")
	}
	now := time.Now().UTC()
	state.ConsumedAt = &now
	// Delete on consume (atomic single-use) so consumed states never accumulate
	// in the persisted store. Reuse is still rejected because the hash is gone
	// (SEC-06).
	delete(s.oauthStates, hash)
	if err := s.persistLocked(); err != nil {
		return OAuthState{}, err
	}
	return state, nil
}

const maxPendingOAuthStates = 10000

// sweepOAuthStatesLocked removes expired or already-consumed OAuth states so the
// persisted store cannot grow without bound. Callers must hold s.mu (SEC-06).
func (s *AuthStore) sweepOAuthStatesLocked(now time.Time) {
	for h, st := range s.oauthStates {
		if st.ConsumedAt != nil || now.After(st.ExpiresAt) {
			delete(s.oauthStates, h)
		}
	}
}

func (s *AuthStore) StoreOAuthCredential(provider, userID, organizationID string, payload map[string]any) (OAuthCredential, error) {
	// The trusted single-user local scope has no users or memberships (the
	// unauthenticated local runtime is already a single operator). Every other
	// (multi-tenant) organization must prove a write membership before a
	// credential may be attached to it.
	if strings.TrimSpace(organizationID) != LocalOrganizationID {
		if _, err := s.Authorize(userID, organizationID, "write"); err != nil {
			return OAuthCredential{}, err
		}
	}
	access, _ := payload["access_token"].(string)
	refresh, _ := payload["refresh_token"].(string)
	if strings.TrimSpace(access) == "" {
		return OAuthCredential{}, errors.New("oauth response has no access_token")
	}
	accessCipher, err := encryptCredential(access)
	if err != nil {
		return OAuthCredential{}, err
	}
	refreshCipher := ""
	if refresh != "" {
		refreshCipher, err = encryptCredential(refresh)
		if err != nil {
			return OAuthCredential{}, err
		}
	}
	expires := time.Time{}
	if seconds, ok := payload["expires_in"].(float64); ok && seconds > 0 {
		expires = time.Now().UTC().Add(time.Duration(seconds) * time.Second)
	}
	credential := OAuthCredential{ID: "cred_" + uuid.NewString(), UserID: userID, OrganizationID: organizationID, Provider: provider, AccessTokenCiphertext: accessCipher, RefreshTokenCiphertext: refreshCipher, ExpiresAt: expires, UpdatedAt: time.Now().UTC()}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[credential.ID] = credential
	return credential, s.persistLocked()
}

// OAuthAccessTokenForOrganization decrypts a currently valid credential only for
// the requested tenant and provider. The plaintext token never enters a public
// response or is persisted back to disk.
func (s *AuthStore) OAuthAccessTokenForOrganization(organizationID, provider string) (string, OAuthCredential, error) {
	organizationID = strings.TrimSpace(organizationID)
	provider = strings.TrimSpace(provider)
	if organizationID == "" || provider == "" {
		return "", OAuthCredential{}, errors.New("organization and provider are required")
	}
	s.mu.RLock()
	var selected OAuthCredential
	for _, credential := range s.credentials {
		if credential.RevokedAt == nil && credential.OrganizationID == organizationID && credential.Provider == provider && credential.UpdatedAt.After(selected.UpdatedAt) {
			selected = credential
		}
	}
	s.mu.RUnlock()
	if selected.ID == "" {
		return "", OAuthCredential{}, os.ErrNotExist
	}
	if !selected.ExpiresAt.IsZero() && time.Now().UTC().Add(30*time.Second).After(selected.ExpiresAt) {
		return "", selected, errors.New("oauth credential is expired or near expiry")
	}
	token, err := decryptCredential(selected.AccessTokenCiphertext)
	if err != nil {
		return "", OAuthCredential{}, err
	}
	return token, selected, nil
}

func (s *AuthStore) HasOAuthCredentialForOrganization(organizationID, provider string) bool {
	organizationID = strings.TrimSpace(organizationID)
	provider = strings.TrimSpace(provider)
	if s == nil || organizationID == "" || provider == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, credential := range s.credentials {
		if credential.OrganizationID == organizationID && credential.Provider == provider && credential.RevokedAt == nil {
			return true
		}
	}
	return false
}

func (s *AuthStore) RefreshOAuthCredential(ctx context.Context, provider OAuthProvider, credentialID string, client *http.Client) (OAuthCredential, error) {
	return s.refreshOAuthCredential(ctx, "", provider, credentialID, client)
}

func (s *AuthStore) RefreshOAuthCredentialForOrganization(ctx context.Context, organizationID string, provider OAuthProvider, credentialID string, client *http.Client) (OAuthCredential, error) {
	return s.refreshOAuthCredential(ctx, strings.TrimSpace(organizationID), provider, credentialID, client)
}

func (s *AuthStore) refreshOAuthCredential(ctx context.Context, organizationID string, provider OAuthProvider, credentialID string, client *http.Client) (OAuthCredential, error) {
	s.mu.RLock()
	credential, ok := s.credentials[credentialID]
	s.mu.RUnlock()
	if !ok {
		return OAuthCredential{}, os.ErrNotExist
	}
	if credential.Provider != provider.Name {
		return OAuthCredential{}, errors.New("oauth credential provider mismatch")
	}
	if organizationID != "" && credential.OrganizationID != organizationID {
		return OAuthCredential{}, errors.New("oauth credential is outside the active organization")
	}
	if credential.RevokedAt != nil {
		return OAuthCredential{}, ErrOAuthCredentialRevoked
	}
	originalUpdatedAt := credential.UpdatedAt
	refresh, err := decryptCredential(credential.RefreshTokenCiphertext)
	if err != nil {
		return OAuthCredential{}, err
	}
	if err := provider.Validate(); err != nil {
		return OAuthCredential{}, err
	}
	if client == nil {
		client = NewSafeEgressHTTPClient(EgressOptions{Callsite: "oauth", Timeout: 30 * time.Second})
	}
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}, "client_id": {os.Getenv(provider.ClientIDEnv)}, "client_secret": {os.Getenv(provider.SecretEnv)}}
	client = oauthClient(client)
	request, err := oauthRequest(ctx, http.MethodPost, provider.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return OAuthCredential{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return OAuthCredential{}, err
	}
	defer response.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return OAuthCredential{}, err
	}
	if response.StatusCode/100 != 2 {
		return OAuthCredential{}, fmt.Errorf("oauth refresh failed with status %d", response.StatusCode)
	}
	access, _ := payload["access_token"].(string)
	if access == "" {
		return OAuthCredential{}, errors.New("oauth refresh has no access_token")
	}
	credential.AccessTokenCiphertext, err = encryptCredential(access)
	if err != nil {
		return OAuthCredential{}, err
	}
	if next, _ := payload["refresh_token"].(string); next != "" {
		credential.RefreshTokenCiphertext, err = encryptCredential(next)
		if err != nil {
			return OAuthCredential{}, err
		}
	}
	if seconds, ok := payload["expires_in"].(float64); ok && seconds > 0 {
		credential.ExpiresAt = time.Now().UTC().Add(time.Duration(seconds) * time.Second)
	}
	credential.UpdatedAt = time.Now().UTC()
	s.mu.Lock()
	current, currentOK := s.credentials[credential.ID]
	if !currentOK || !current.UpdatedAt.Equal(originalUpdatedAt) {
		s.mu.Unlock()
		return OAuthCredential{}, ErrOAuthCredentialConflict
	}
	if current.RevokedAt != nil {
		s.mu.Unlock()
		return OAuthCredential{}, ErrOAuthCredentialRevoked
	}
	s.credentials[credential.ID] = credential
	err = s.persistLocked()
	s.mu.Unlock()
	return credential, err
}

func (s *AuthStore) RevokeOAuthCredentialForOrganization(ctx context.Context, organizationID, credentialID string, provider OAuthProvider, client *http.Client) error {
	organizationID = strings.TrimSpace(organizationID)
	credentialID = strings.TrimSpace(credentialID)
	s.mu.RLock()
	credential, ok := s.credentials[credentialID]
	s.mu.RUnlock()
	if !ok {
		return os.ErrNotExist
	}
	if credential.OrganizationID != organizationID || credential.Provider != provider.Name {
		return errors.New("oauth credential is outside the active organization")
	}
	if credential.RevokedAt != nil {
		return ErrOAuthCredentialRevoked
	}
	originalUpdatedAt := credential.UpdatedAt
	if strings.TrimSpace(provider.RevocationURL) != "" {
		access, err := decryptCredential(credential.AccessTokenCiphertext)
		if err != nil {
			return err
		}
		if err := provider.Revoke(ctx, client, access); err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	s.mu.Lock()
	current, currentOK := s.credentials[credential.ID]
	if !currentOK || !current.UpdatedAt.Equal(originalUpdatedAt) {
		s.mu.Unlock()
		return ErrOAuthCredentialConflict
	}
	current.RevokedAt = &now
	current.UpdatedAt = now
	s.credentials[current.ID] = current
	err := s.persistLocked()
	s.mu.Unlock()
	return err
}

func (s *AuthStore) Users() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]User, 0, len(s.users))
	for _, user := range s.users {
		result = append(result, user)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Email < result[j].Email })
	return result
}

// Organizations returns a stable snapshot of known organizations. Background
// PostgreSQL recovery uses these trusted auth records to build one scoped
// runtime per tenant rather than granting SQL-level global visibility.
func (s *AuthStore) Organizations() []Organization {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Organization, 0, len(s.organizations))
	for _, organization := range s.organizations {
		result = append(result, organization)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s *AuthStore) FirstOrganization(userID string) (Organization, Membership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, membership := range s.memberships {
		if membership.UserID == userID {
			if organization, ok := s.organizations[membership.OrganizationID]; ok {
				return organization, membership, nil
			}
		}
	}
	return Organization{}, Membership{}, os.ErrNotExist
}

func (s *AuthStore) ProvisionOAuthUser(payload map[string]any, provider string) (User, Organization, Membership, error) {
	provider = strings.TrimSpace(provider)
	email := strings.ToLower(strings.TrimSpace(oauthClaim(payload, "email", "email_address", "preferred_username", "login")))
	if email == "" || !strings.Contains(email, "@") {
		return User{}, Organization{}, Membership{}, errors.New("oauth userinfo did not provide a valid email")
	}
	// SEC-04: bind the local account to the provider's stable subject, never
	// to the email alone. An email address is reassignable and is not unique
	// across identity providers, so linking purely by email lets a second IdP
	// (or an unverified-email account at a federatable provider) silently take
	// over an account created through a different sign-in.
	subject := oauthClaim(payload, "sub", "oid", "user_id", "uid", "id")
	if subject == "" {
		return User{}, Organization{}, Membership{}, errors.New("oauth userinfo did not provide a stable subject identifier")
	}
	if provider == "" {
		return User{}, Organization{}, Membership{}, errors.New("oauth provider is required")
	}
	identityKey := provider + "|" + subject
	name := oauthClaim(payload, "name", "preferred_username", "login")
	if name == "" {
		name = email
	}
	if len(email) > 320 || len(name) > 200 {
		return User{}, Organization{}, Membership{}, errors.New("user field is too long")
	}

	s.mu.Lock()
	// A known federated identity always resolves to the same local user.
	if uid, ok := s.identities[identityKey]; ok {
		if user, ok := s.users[uid]; ok {
			s.mu.Unlock()
			organization, membership, err := s.FirstOrganization(user.ID)
			if err == nil {
				return user, organization, membership, nil
			}
			organization, membership, err = s.CreateOrganization(provider+" — "+email, user)
			return user, organization, membership, err
		}
		// The bound user no longer exists; drop the stale binding and re-provision.
		delete(s.identities, identityKey)
	}
	// A brand-new identity must not silently attach to an account that already
	// exists under a different sign-in — that is the account-takeover vector.
	for _, existing := range s.users {
		if existing.Email == email {
			s.mu.Unlock()
			return User{}, Organization{}, Membership{}, errors.New("an account with this email already exists under a different sign-in; link it from account settings instead")
		}
	}
	now := time.Now().UTC()
	user := User{ID: "usr_" + uuid.NewString(), Email: email, Name: name, CreatedAt: now}
	s.users[user.ID] = user
	s.identities[identityKey] = user.ID
	if err := s.persistLocked(); err != nil {
		s.mu.Unlock()
		return User{}, Organization{}, Membership{}, err
	}
	s.mu.Unlock()

	organization, membership, err := s.CreateOrganization(provider+" — "+email, user)
	return user, organization, membership, err
}

func oauthClaim(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	// Fall back to the nested claim maps the server attaches during the OAuth
	// and OIDC flows (userinfo endpoint response and validated id_token
	// claims). The subject identifier (SEC-04) usually lives there.
	for _, nestedKey := range []string{"userinfo", "id_token_claims"} {
		if nested, ok := payload[nestedKey].(map[string]any); ok {
			if value := oauthClaim(nested, keys...); value != "" {
				return value
			}
		}
	}
	return ""
}

func (s *AuthStore) persistLocked() error {
	if s.root == "" {
		return nil
	}
	for name, value := range map[string]any{
		"users":         s.users,
		"organizations": s.organizations,
		"memberships":   s.memberships,
		"tokens":        s.tokens,
		"oauth-states":  s.oauthStates,
		"credentials":   s.credentials,
		"mfa-attempts":  s.mfaAttempts,
		"identities":    s.identities,
	} {
		if err := writeJSONAtomic(filepathJoin(s.root, name+".json"), value); err != nil {
			return err
		}
	}
	return nil
}

func validRole(role Role) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleOperator, RoleViewer, RoleAuditor:
		return true
	default:
		return false
	}
}

func roleAllows(role Role, action string) bool {
	switch strings.TrimSpace(action) {
	case "read":
		return role == RoleOwner || role == RoleAdmin || role == RoleOperator || role == RoleViewer || role == RoleAuditor
	case "execute":
		return role == RoleOwner || role == RoleAdmin || role == RoleOperator
	case "write":
		return role == RoleOwner || role == RoleAdmin || role == RoleOperator
	case "share", "manage":
		return role == RoleOwner || role == RoleAdmin
	default:
		return false
	}
}

func membershipKey(userID, organizationID string) string { return userID + ":" + organizationID }
func hashSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func credentialKey() ([]byte, error) {
	value := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_CREDENTIAL_KEY"))
	if len(value) < 16 {
		return nil, errors.New("OLLAMA_AGENT_CREDENTIAL_KEY must be configured with at least 16 characters")
	}
	sum := sha256.Sum256([]byte(value))
	return sum[:], nil
}

func encryptCredential(value string) (string, error) {
	key, err := credentialKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(value), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func decryptCredential(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("oauth credential has no refresh token")
	}
	key, err := credentialKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	sealed, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	if len(sealed) < aead.NonceSize() {
		return "", errors.New("invalid oauth ciphertext")
	}
	plaintext, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func randomSecret(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func filepathJoin(root, name string) string {
	return strings.TrimRight(root, string(os.PathSeparator)) + string(os.PathSeparator) + name
}

// OAuthProvider validates the provider configuration and exchanges an authorization code
// through a caller-supplied HTTP client. Secrets are read only from environment variables.
type OAuthProvider struct {
	Name                  string   `json:"name"`
	AuthorizeURL          string   `json:"authorize_url"`
	TokenURL              string   `json:"token_url"`
	RevocationURL         string   `json:"revocation_url,omitempty"`
	UserInfoURL           string   `json:"userinfo_url,omitempty"`
	IssuerURL             string   `json:"issuer_url,omitempty"`
	Audience              string   `json:"audience,omitempty"`
	ClientIDEnv           string   `json:"client_id_env"`
	SecretEnv             string   `json:"secret_env"`
	RedirectURIs          []string `json:"redirect_uris,omitempty"`
	AllowLoopbackRedirect bool     `json:"allow_loopback_redirect,omitempty"`
}

func (p OAuthProvider) ValidateRedirectURI(raw string) error {
	_, err := p.NormalizeRedirectURI(raw)
	return err
}

func (p OAuthProvider) NormalizeRedirectURI(raw string) (string, error) {
	canonical, err := validateOAuthRedirectSyntax(raw, p.AllowLoopbackRedirect)
	if err != nil {
		return "", err
	}
	// A desktop app receives the OAuth callback on a loopback address it opens
	// locally (127.0.0.1/localhost/[::1]); that URI cannot be pinned in a fixed
	// allowlist because the port is chosen at runtime. When loopback redirects
	// are enabled, accept a validated loopback URI without requiring the
	// allowlist. This does not loosen the security binding: CreateOAuthState
	// stores the exact redirect_uri and ConsumeOAuthState demands the identical
	// value at the callback, so a tampered redirect still fails. Anything that
	// is not loopback continues to require the allowlist below.
	if p.AllowLoopbackRedirect {
		if parsed, parseErr := url.Parse(canonical); parseErr == nil && isLoopbackHost(parsed.Hostname()) {
			return canonical, nil
		}
	}
	if len(p.RedirectURIs) == 0 {
		return "", errors.New("oauth redirect URI allowlist is required")
	}
	for _, allowed := range p.RedirectURIs {
		allowedCanonical, allowedErr := validateOAuthRedirectSyntax(allowed, p.AllowLoopbackRedirect)
		if allowedErr == nil && allowedCanonical == canonical {
			return canonical, nil
		}
	}
	return "", errors.New("oauth redirect URI is not allowlisted")
}

func validateOAuthRedirectSyntax(raw string, allowLoopbackHTTP bool) (string, error) {
	value := strings.TrimSpace(raw)
	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return "", errors.New("oauth redirect URI must be an absolute URL without userinfo or fragment")
	}
	if u.Scheme != "https" && !(allowLoopbackHTTP && u.Scheme == "http" && isLoopbackHost(u.Hostname())) {
		return "", errors.New("oauth redirect URI must use HTTPS or explicitly allowed loopback HTTP")
	}
	return u.String(), nil
}

func (p OAuthProvider) Validate() error {
	for _, redirectURI := range p.RedirectURIs {
		if _, err := validateOAuthRedirectSyntax(redirectURI, p.AllowLoopbackRedirect); err != nil {
			return err
		}
	}
	for _, raw := range []string{p.AuthorizeURL, p.TokenURL, p.RevocationURL} {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if err := validateOAuthEndpointURL(raw); err != nil {
			return err
		}
	}
	if strings.TrimSpace(p.IssuerURL) != "" {
		if err := validateOAuthEndpointURL(p.IssuerURL); err != nil {
			return err
		}
	}
	if strings.TrimSpace(p.IssuerURL) == "" && (strings.TrimSpace(p.AuthorizeURL) == "" || strings.TrimSpace(p.TokenURL) == "") {
		return errors.New("oauth authorize and token endpoints or an issuer are required")
	}
	if p.UserInfoURL != "" {
		if err := validateOAuthEndpointURL(p.UserInfoURL); err != nil {
			return err
		}
	}
	if os.Getenv(p.ClientIDEnv) == "" || os.Getenv(p.SecretEnv) == "" {
		return errors.New("oauth client credentials are not configured")
	}
	return nil
}

func validateOAuthEndpointURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return errors.New("oauth endpoints must use HTTPS without credentials or fragment")
	}
	return nil
}

type oauthLoopbackContextKey struct{}

func oauthClient(base *http.Client) *http.Client {
	if base == nil {
		base = NewSafeEgressHTTPClient(EgressOptions{Callsite: "oauth", Timeout: 30 * time.Second})
	}
	client := *base
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return errors.New("oauth redirects are disabled") }
	switch transport := base.Transport.(type) {
	case nil:
		safe := http.DefaultTransport.(*http.Transport).Clone()
		safe.Proxy = nil
		safe.DialContext = oauthDialContext
		client.Transport = safe
	case *http.Transport:
		safe := transport.Clone()
		safe.Proxy = nil
		safe.DialContext = oauthDialContext
		client.Transport = safe
	default:
		safe := http.DefaultTransport.(*http.Transport).Clone()
		safe.Proxy = nil
		safe.DialContext = oauthDialContext
		client.Transport = safe
	}
	return &client
}

func oauthRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return nil, errors.New("oauth endpoint URL is invalid")
	}
	requestContext := context.WithValue(ctx, oauthLoopbackContextKey{}, isLoopbackHost(parsed.Hostname()))
	return http.NewRequestWithContext(requestContext, method, parsed.String(), body)
}

func oauthDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	if loopback, _ := ctx.Value(oauthLoopbackContextKey{}).(bool); loopback {
		return conn, nil
	}
	remote, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
	if splitErr != nil {
		_ = conn.Close()
		return nil, errors.New("oauth connected address is invalid")
	}
	if ip := net.ParseIP(strings.Trim(remote, "[]")); ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()) {
		_ = conn.Close()
		return nil, errors.New("oauth destination connected to a private address")
	}
	return conn, nil
}

type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

func (p OAuthProvider) Discover(ctx context.Context, client *http.Client) (OIDCDiscovery, error) {
	issuer := strings.TrimRight(strings.TrimSpace(p.IssuerURL), "/")
	if issuer == "" {
		return OIDCDiscovery{}, errors.New("oidc issuer is not configured")
	}
	client = oauthClient(client)
	request, err := oauthRequest(ctx, http.MethodGet, issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		return OIDCDiscovery{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return OIDCDiscovery{}, err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return OIDCDiscovery{}, fmt.Errorf("oidc discovery failed with status %d", response.StatusCode)
	}
	var discovery OIDCDiscovery
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&discovery); err != nil {
		return OIDCDiscovery{}, err
	}
	if strings.TrimRight(discovery.Issuer, "/") != issuer || discovery.AuthorizationEndpoint == "" || discovery.TokenEndpoint == "" || discovery.JWKSURI == "" {
		return OIDCDiscovery{}, errors.New("oidc discovery issuer or required endpoints are invalid")
	}
	for _, endpoint := range []string{discovery.AuthorizationEndpoint, discovery.TokenEndpoint, discovery.JWKSURI} {
		if err := validateOAuthEndpointURL(endpoint); err != nil {
			return OIDCDiscovery{}, err
		}
	}
	if discovery.UserInfoEndpoint != "" {
		if err := validateOAuthEndpointURL(discovery.UserInfoEndpoint); err != nil {
			return OIDCDiscovery{}, err
		}
	}
	return discovery, nil
}

func (p OAuthProvider) ValidateIDToken(ctx context.Context, client *http.Client, rawToken, expectedNonce string) (map[string]any, error) {
	client = oauthClient(client)
	parts := strings.Split(strings.TrimSpace(rawToken), ".")
	if len(parts) != 3 {
		return nil, errors.New("id_token must be a compact JWT")
	}
	decode := func(value string, target any) error {
		data, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, target)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := decode(parts[0], &header); err != nil || header.Alg != "RS256" || header.Kid == "" {
		return nil, errors.New("unsupported or incomplete id_token header")
	}
	var claims map[string]any
	if err := decode(parts[1], &claims); err != nil {
		return nil, err
	}
	discovery, err := p.Discover(ctx, client)
	if err != nil {
		return nil, err
	}
	request, err := oauthRequest(ctx, http.MethodGet, discovery.JWKSURI, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("oidc jwks failed with status %d", response.StatusCode)
	}
	var jwks struct {
		Keys []struct {
			KID string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&jwks); err != nil {
		return nil, err
	}
	var publicKey *rsa.PublicKey
	for _, key := range jwks.Keys {
		if key.KID != header.Kid || key.Kty != "RSA" {
			continue
		}
		nBytes, nErr := base64.RawURLEncoding.DecodeString(key.N)
		eBytes, eErr := base64.RawURLEncoding.DecodeString(key.E)
		if nErr != nil || eErr != nil || len(eBytes) == 0 {
			continue
		}
		exponent := 0
		for _, value := range eBytes {
			exponent = exponent<<8 | int(value)
		}
		publicKey = &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}
		break
	}
	if publicKey == nil || publicKey.N.Sign() <= 0 || publicKey.E < 2 {
		return nil, errors.New("oidc signing key was not found")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return nil, errors.New("invalid id_token signature")
	}
	issuer, _ := claims["iss"].(string)
	if strings.TrimRight(issuer, "/") != strings.TrimRight(discovery.Issuer, "/") {
		return nil, errors.New("id_token issuer mismatch")
	}
	expectedAudience := strings.TrimSpace(p.Audience)
	if expectedAudience == "" {
		expectedAudience = strings.TrimSpace(os.Getenv(p.ClientIDEnv))
	}
	if !jwtAudienceContains(claims["aud"], expectedAudience) {
		return nil, errors.New("id_token audience mismatch")
	}
	if strings.TrimSpace(expectedNonce) == "" || claims["nonce"] != expectedNonce {
		return nil, errors.New("id_token nonce mismatch")
	}
	exp, ok := claims["exp"].(float64)
	if !ok || time.Now().UTC().Unix() >= int64(exp) {
		return nil, errors.New("id_token is expired")
	}
	return claims, nil
}

func jwtAudienceContains(value any, expected string) bool {
	if expected == "" {
		return false
	}
	if single, ok := value.(string); ok {
		return single == expected
	}
	if multiple, ok := value.([]any); ok {
		for _, item := range multiple {
			if item == expected {
				return true
			}
		}
	}
	return false
}

func (p OAuthProvider) FetchUserInfo(ctx context.Context, client *http.Client, accessToken string) (map[string]any, error) {
	endpoint := strings.TrimSpace(p.UserInfoURL)
	if endpoint == "" && strings.TrimSpace(p.IssuerURL) != "" {
		discovery, err := p.Discover(ctx, client)
		if err != nil {
			return nil, err
		}
		endpoint = discovery.UserInfoEndpoint
	}
	if endpoint == "" {
		return nil, errors.New("oauth userinfo endpoint is not configured")
	}
	client = oauthClient(client)
	request, err := oauthRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("oauth userinfo failed with status %d", response.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (p OAuthProvider) AuthorizationURL(state string, scopes []string) (string, error) {
	return p.AuthorizationURLWithNonce(state, "", scopes)
}

func (p OAuthProvider) AuthorizationURLWithNonce(state, nonce string, scopes []string) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	u, err := url.Parse(p.AuthorizeURL)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set("client_id", os.Getenv(p.ClientIDEnv))
	query.Set("response_type", "code")
	query.Set("state", state)
	if strings.TrimSpace(nonce) != "" {
		query.Set("nonce", nonce)
	}
	query.Set("scope", strings.Join(scopes, " "))
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func (p OAuthProvider) ExchangeCode(ctx context.Context, client *http.Client, code, redirectURI, codeVerifier string) (map[string]any, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	client = oauthClient(client)
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirectURI}, "client_id": {os.Getenv(p.ClientIDEnv)}, "client_secret": {os.Getenv(p.SecretEnv)}, "code_verifier": {codeVerifier}}
	request, err := oauthRequest(ctx, http.MethodPost, p.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("oauth token exchange failed with status %d", response.StatusCode)
	}
	return payload, nil
}

func (p OAuthProvider) Revoke(ctx context.Context, client *http.Client, accessToken string) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(p.RevocationURL) == "" {
		return errors.New("oauth revocation endpoint is not configured")
	}
	if strings.TrimSpace(accessToken) == "" {
		return errors.New("oauth access token is required for revocation")
	}
	if client == nil {
		client = NewSafeEgressHTTPClient(EgressOptions{Callsite: "oauth", Timeout: 30 * time.Second})
	}
	client = oauthClient(client)
	form := url.Values{
		"token":           {accessToken},
		"token_type_hint": {"access_token"},
		"client_id":       {os.Getenv(p.ClientIDEnv)},
		"client_secret":   {os.Getenv(p.SecretEnv)},
	}
	request, err := oauthRequest(ctx, http.MethodPost, p.RevocationURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("oauth revocation failed with status %d", response.StatusCode)
	}
	return nil
}
