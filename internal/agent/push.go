package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type PushSubscription struct {
	ID              string     `json:"id"`
	Token           string     `json:"-"`
	TokenCiphertext string     `json:"-"`
	Platform        string     `json:"platform"`
	UserID          string     `json:"user_id"`
	OrganizationID  string     `json:"organization_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
}

// pushBeforeDeliveryHook is used by package tests to force revocation in the final pre-send window.
var pushBeforeDeliveryHook func(PushSubscription)

type PushService struct {
	mu       sync.Mutex
	root     string
	endpoint string
	client   *http.Client
	items    map[string]PushSubscription
}

var ErrPushSubscriptionNotFound = errors.New("push subscription is not available")

const (
	maxPushSubscriptions           = 10_000
	maxPushSubscriptionsPerOrg     = 1_000
	maxPushSubscriptionBytesPerOrg = 2 << 20
	maxPushSubscriptionRecordBytes = 16 << 10
	maxPushSubscriptionFileBytes   = 20 << 20
)

var errPushSubscriptionQuota = errors.New("push subscription capacity limit reached")

func NewPushService(root, endpoint string) (*PushService, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, nil
	}
	validatedEndpoint, err := validatePushEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	endpoint = validatedEndpoint
	parsedEndpoint, _ := url.Parse(endpoint)
	allowLoopback := isPushLoopbackHost(parsedEndpoint.Hostname())
	root = strings.TrimSpace(root)
	if root != "" {
		root, err = canonicalPushPersistentRoot(root)
		if err != nil {
			return nil, err
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return pushDialContextWithResolver(ctx, network, address, allowLoopback, net.DefaultResolver.LookupIPAddr)
	}
	service := &PushService{root: root, endpoint: endpoint, client: &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, items: map[string]PushSubscription{}}
	if service.root != "" {
		if err := os.MkdirAll(service.root, 0o700); err != nil {
			return nil, err
		}
		if err := service.loadSubscriptions(); err != nil {
			return nil, err
		}
	}
	return service, nil
}

func pushDialContextWithResolver(ctx context.Context, network, address string, allowLoopback bool, lookup func(context.Context, string) ([]net.IPAddr, error)) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return nil, errors.New("push destination address is invalid")
	}
	var addresses []net.IPAddr
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		addresses = []net.IPAddr{{IP: ip}}
	} else {
		if lookup == nil {
			return nil, errors.New("push destination resolver is unavailable")
		}
		addresses, err = lookup(ctx, host)
		if err != nil {
			return nil, errors.New("push destination lookup failed")
		}
	}
	if len(addresses) == 0 {
		return nil, errors.New("push destination has no addresses")
	}
	for _, resolved := range addresses {
		if resolved.IP == nil || (allowLoopback && !resolved.IP.IsLoopback()) || (!allowLoopback && unsafeEgressIP(resolved.IP)) {
			return nil, errors.New("push destination resolves to a disallowed address")
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var lastErr error
	for _, resolved := range addresses {
		if network == "tcp4" && resolved.IP.To4() == nil || network == "tcp6" && resolved.IP.To4() != nil {
			continue
		}
		target := net.JoinHostPort(resolved.IP.String(), port)
		if resolved.Zone != "" {
			target = net.JoinHostPort(resolved.IP.String()+"%"+resolved.Zone, port)
		}
		conn, dialErr := dialer.DialContext(ctx, network, target)
		if dialErr != nil {
			lastErr = dialErr
			continue
		}
		remoteHost, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
		remoteIP := net.ParseIP(strings.Trim(remoteHost, "[]"))
		if splitErr != nil || remoteIP == nil || !remoteIP.Equal(resolved.IP) || (allowLoopback && !remoteIP.IsLoopback()) || (!allowLoopback && unsafeEgressIP(remoteIP)) {
			_ = conn.Close()
			lastErr = errors.New("push connected address was not approved")
			continue
		}
		return conn, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("push destination has no address for requested network")
}

func validatePushEndpoint(raw string) (string, error) {
	if endpointURLHasSensitiveMaterial(raw) {
		return "", errors.New("push endpoint must not contain credential-shaped URL data")
	}
	endpoint, err := url.Parse(raw)
	if err != nil || endpoint == nil || endpoint.Opaque != "" || endpoint.Host == "" || endpoint.Hostname() == "" || endpoint.User != nil {
		return "", errors.New("push endpoint must be an absolute HTTPS URL or exact loopback HTTP URL without userinfo")
	}
	endpoint.Scheme = strings.ToLower(endpoint.Scheme)
	switch endpoint.Scheme {
	case "https":
	case "http":
		if !isPushLoopbackHost(endpoint.Hostname()) {
			return "", errors.New("push endpoint must use HTTPS or exact loopback HTTP")
		}
	default:
		return "", errors.New("push endpoint must use HTTPS or exact loopback HTTP")
	}
	if endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return "", errors.New("push endpoint must not contain query or fragment credentials")
	}
	return endpoint.String(), nil
}

func isPushLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (p *PushService) Register(token, platform, userID, organizationID string) (PushSubscription, error) {
	token = strings.TrimSpace(token)
	platform = strings.ToLower(strings.TrimSpace(platform))
	userID = strings.TrimSpace(userID)
	organizationID = strings.TrimSpace(organizationID)
	if token == "" || len(token) > 1024 || organizationID == "" || len(organizationID) > 256 || len(userID) > 256 {
		return PushSubscription{}, errors.New("push token and organization are required")
	}
	if platform != "android" && platform != "ios" {
		return PushSubscription{}, errors.New("push platform must be android or ios")
	}
	ciphertext := ""
	if p.root != "" {
		var err error
		ciphertext, err = encryptCredential(token)
		if err != nil {
			return PushSubscription{}, fmt.Errorf("encrypt push token: %w", err)
		}
	}
	now := time.Now().UTC()
	id := pushSubscriptionID(token, userID, organizationID)
	item := PushSubscription{ID: id, Token: token, TokenCiphertext: ciphertext, Platform: platform, UserID: userID, OrganizationID: organizationID, CreatedAt: now, UpdatedAt: now}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.withFileStateLocked(true, func() error {
		p.items[id] = item
		if err := validatePushSubscriptionQuotas(p.items); err != nil {
			delete(p.items, id)
			return err
		}
		return nil
	}); err != nil {
		return PushSubscription{}, err
	}
	return clonePushSubscription(item), nil
}

func (p *PushService) ListOrganization(organizationID string) []PushSubscription {
	items, err := p.listOrganization(organizationID)
	if err != nil {
		return nil
	}
	return items
}

func (p *PushService) listOrganization(organizationID string) ([]PushSubscription, error) {
	if p == nil {
		return nil, nil
	}
	organizationID = strings.TrimSpace(organizationID)
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.withFileStateLocked(false, nil); err != nil {
		return nil, err
	}
	items := make([]PushSubscription, 0)
	for _, item := range p.items {
		if item.OrganizationID == organizationID && item.RevokedAt == nil {
			items = append(items, clonePushSubscription(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt.Before(items[j].UpdatedAt)
	})
	return items, nil
}

// Revoke marks one subscription inactive only when both its organization and
// owning user match. It never returns token material to the caller.
func (p *PushService) Revoke(id, organizationID, userID string) error {
	if p == nil {
		return ErrPushSubscriptionNotFound
	}
	id, organizationID, userID = strings.TrimSpace(id), strings.TrimSpace(organizationID), strings.TrimSpace(userID)
	if id == "" || organizationID == "" || userID == "" {
		return ErrPushSubscriptionNotFound
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.withFileStateLocked(true, func() error {
		item, ok := p.items[id]
		if !ok || item.OrganizationID != organizationID || item.UserID != userID || item.RevokedAt != nil {
			return ErrPushSubscriptionNotFound
		}
		now := time.Now().UTC()
		item.RevokedAt = &now
		item.UpdatedAt = now
		p.items[id] = item
		return nil
	})
}

func (p *PushService) subscriptionStillActive(snapshot PushSubscription, organizationID string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.withFileStateLocked(false, nil); err != nil {
		return false, err
	}
	current, ok := p.items[snapshot.ID]
	return ok && current.OrganizationID == strings.TrimSpace(organizationID) && current.RevokedAt == nil && current.Token == snapshot.Token && current.UpdatedAt.Equal(snapshot.UpdatedAt), nil
}

func (p *PushService) NotifyOrganization(ctx context.Context, organizationID, title, body string, data map[string]any) error {
	if p == nil || strings.TrimSpace(p.endpoint) == "" {
		return nil
	}
	normalized, err := normalizedOutboundPayload(map[string]any{"title": title, "body": body, "data": data}, maxOutboundDLPScanBytes)
	if err != nil {
		return err
	}
	validated := normalized.(map[string]any)
	title, _ = validated["title"].(string)
	body, _ = validated["body"].(string)
	data, _ = validated["data"].(map[string]any)
	items, err := p.listOrganization(organizationID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if pushBeforeDeliveryHook != nil {
			pushBeforeDeliveryHook(clonePushSubscription(item))
		}
		var encoded []byte
		shouldSend := false
		p.mu.Lock()
		stateErr := p.withFileStateLocked(false, func() error {
			current, ok := p.items[item.ID]
			if !ok || current.OrganizationID != strings.TrimSpace(organizationID) || current.RevokedAt != nil || current.Token != item.Token || !current.UpdatedAt.Equal(item.UpdatedAt) {
				return nil
			}
			payload := map[string]any{"to": current.Token, "title": title, "body": body, "data": data, "sound": "default"}
			var err error
			encoded, err = json.Marshal(payload)
			if err != nil {
				return errors.New("push provider request could not be encoded")
			}
			shouldSend = true
			return nil
		})
		p.mu.Unlock()
		if stateErr != nil {
			return errors.New("push subscription state or provider delivery could not be verified")
		}
		if !shouldSend {
			continue
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(encoded))
		if err != nil {
			return errors.New("push provider request could not be created")
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := p.client.Do(request)
		if err != nil {
			return errors.New("push provider request failed")
		}
		_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		_ = response.Body.Close()
		if readErr != nil {
			return errors.New("push provider response could not be read")
		}
		if response.StatusCode/100 != 2 {
			return fmt.Errorf("push provider returned status %d", response.StatusCode)
		}
	}

	return nil
}

func (p *PushService) subscriptionsPath() string {
	return filepath.Join(p.root, "subscriptions.json")
}

func (p *PushService) subscriptionsLockPath() string {
	return filepath.Join(p.root, ".subscriptions.lock")
}

// withFileStateLocked refreshes the complete on-disk snapshot while holding a
// stable cross-process lock, then applies one read-modify-write operation.
// Callers hold p.mu while invoking this helper.
func (p *PushService) withFileStateLocked(persist bool, run func() error) error {
	if p.root == "" {
		if run == nil {
			return nil
		}
		return run()
	}
	previous := clonePushSubscriptions(p.items)
	return withFileLock(p.subscriptionsLockPath(), func() error {
		migrated, err := p.loadSubscriptionsLocked()
		if err != nil {
			p.items = previous
			return err
		}
		if run != nil {
			if err := run(); err != nil {
				p.items = previous
				return err
			}
		}
		if !persist && !migrated {
			return nil
		}
		if err := p.persistLocked(); err != nil {
			p.items = previous
			return err
		}
		return nil
	})
}

func (p *PushService) persistLocked() error {
	if p.root == "" {
		return nil
	}
	if len(p.items) > maxPushSubscriptions {
		return errors.New("push subscription count exceeds persistence limit")
	}
	if err := validatePushSubscriptionQuotas(p.items); err != nil {
		return err
	}
	stored := make(map[string]pushSubscriptionRecord, len(p.items))
	for id, item := range p.items {
		if item.ID == "" || id != item.ID {
			return errors.New("push subscription id is inconsistent")
		}
		if strings.TrimSpace(item.TokenCiphertext) == "" {
			return errors.New("push subscription token is not encrypted")
		}
		record := pushSubscriptionRecord{ID: item.ID, TokenCiphertext: item.TokenCiphertext, Platform: item.Platform, UserID: item.UserID, OrganizationID: item.OrganizationID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, RevokedAt: clonePushTime(item.RevokedAt)}
		encoded, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("marshal push subscription record: %w", err)
		}
		if len(encoded) > maxPushSubscriptionRecordBytes {
			return errors.New("push subscription record exceeds persistence limit")
		}
		stored[id] = record
	}
	return writeJSONAtomicBounded(p.subscriptionsPath(), stored, maxPushSubscriptionFileBytes, "push subscription file")
}

type pushSubscriptionRecord struct {
	ID              string     `json:"id"`
	Token           string     `json:"token,omitempty"`
	TokenCiphertext string     `json:"token_ciphertext,omitempty"`
	Platform        string     `json:"platform"`
	UserID          string     `json:"user_id"`
	OrganizationID  string     `json:"organization_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
}

func (p *PushService) loadSubscriptions() error {
	if p.root == "" {
		return nil
	}
	return withFileLock(p.subscriptionsLockPath(), func() error {
		migrated, err := p.loadSubscriptionsLocked()
		if err != nil {
			return err
		}
		if migrated {
			if err := p.persistLocked(); err != nil {
				return fmt.Errorf("persist encrypted push subscription migration: %w", err)
			}
		}
		return nil
	})
}

// loadSubscriptionsLocked validates and normalizes every record before
// replacing the in-memory snapshot. It must be called while the file lock is
// held so legacy encryption and ID rekeying cannot race another process.
func (p *PushService) loadSubscriptionsLocked() (bool, error) {
	path := p.subscriptionsPath()
	if info, err := os.Stat(path); err == nil {
		if info.Size() > maxPushSubscriptionFileBytes {
			return false, errors.New("push subscription file exceeds persistence limit")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	var stored map[string]pushSubscriptionRecord
	if err := readJSONBounded(path, &stored, maxPushSubscriptionFileBytes); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			p.items = map[string]PushSubscription{}
			return false, nil
		}
		return false, err
	}
	if stored == nil {
		stored = map[string]pushSubscriptionRecord{}
	}
	if len(stored) > maxPushSubscriptions {
		return false, errors.New("push subscription count exceeds persistence limit")
	}
	items := make(map[string]PushSubscription, len(stored))
	migrated := false
	for key, record := range stored {
		encoded, err := json.Marshal(record)
		if err != nil {
			return false, fmt.Errorf("marshal push subscription record: %w", err)
		}
		if len(encoded) > maxPushSubscriptionRecordBytes {
			return false, errors.New("push subscription record exceeds persistence limit")
		}
		storageID := strings.TrimSpace(key)
		if storageID == "" {
			return false, errors.New("push subscription has an empty storage id")
		}
		recordID := strings.TrimSpace(record.ID)
		if recordID != "" && recordID != storageID {
			return false, fmt.Errorf("push subscription %s has mismatched record id", storageID)
		}
		if recordID == "" {
			recordID = storageID
			migrated = true
		}
		ciphertext := strings.TrimSpace(record.TokenCiphertext)
		legacyToken := strings.TrimSpace(record.Token)
		var token string
		if ciphertext == "" {
			if legacyToken == "" {
				return false, fmt.Errorf("push subscription %s has no token", storageID)
			}
			var err error
			ciphertext, err = encryptCredential(legacyToken)
			if err != nil {
				return false, fmt.Errorf("encrypt legacy push token for subscription %s: %w", storageID, err)
			}
			token = legacyToken
			migrated = true
		} else {
			var err error
			token, err = decryptCredential(ciphertext)
			if err != nil {
				return false, fmt.Errorf("decrypt push token for subscription %s: %w", storageID, err)
			}
		}
		token = strings.TrimSpace(token)
		if token == "" || len(token) > 1024 {
			return false, fmt.Errorf("push subscription %s has an invalid token", storageID)
		}
		if legacyToken != "" && legacyToken != token {
			return false, fmt.Errorf("legacy push token does not match encrypted token for subscription %s", storageID)
		}
		platform := strings.ToLower(strings.TrimSpace(record.Platform))
		if platform != "android" && platform != "ios" {
			return false, fmt.Errorf("push subscription %s has an invalid platform", storageID)
		}
		organizationID := strings.TrimSpace(record.OrganizationID)
		if organizationID == "" {
			return false, fmt.Errorf("push subscription %s has no organization", storageID)
		}
		userID := strings.TrimSpace(record.UserID)
		id := pushSubscriptionID(token, userID, organizationID)
		if _, exists := items[id]; exists {
			return false, fmt.Errorf("duplicate push subscription for organization %s and user %s", organizationID, userID)
		}
		items[id] = PushSubscription{ID: id, Token: token, TokenCiphertext: ciphertext, Platform: platform, UserID: userID, OrganizationID: organizationID, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt, RevokedAt: clonePushTime(record.RevokedAt)}
		if storageID != id || recordID != id || legacyToken != "" || platform != record.Platform || userID != record.UserID || organizationID != record.OrganizationID {
			migrated = true
		}
	}
	if err := validatePushSubscriptionQuotas(items); err != nil {
		return false, err
	}
	p.items = items
	return migrated, nil
}

func validatePushSubscriptionQuotas(items map[string]PushSubscription) error {
	if len(items) > maxPushSubscriptions {
		return errors.New("push subscription count exceeds persistence limit")
	}
	tenantCounts := make(map[string]int)
	tenantRecords := make(map[string]map[string]pushSubscriptionRecord)
	storedRecords := make(map[string]pushSubscriptionRecord, len(items))
	for id, item := range items {
		if id == "" || item.ID != id || strings.TrimSpace(item.OrganizationID) == "" || len(item.OrganizationID) > 256 || len(item.UserID) > 256 {
			return errors.New("push subscription identity exceeds persistence limits")
		}
		ciphertext := item.TokenCiphertext
		if ciphertext == "" {
			// Memory-only records do not retain ciphertext; estimate the persisted
			// AES-GCM/base64 representation conservatively for quota parity.
			ciphertext = strings.Repeat("x", len(item.Token)*2+96)
		}
		record := pushSubscriptionRecord{ID: item.ID, TokenCiphertext: ciphertext, Platform: item.Platform, UserID: item.UserID, OrganizationID: item.OrganizationID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, RevokedAt: clonePushTime(item.RevokedAt)}
		encoded, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("marshal push subscription record: %w", err)
		}
		if len(encoded) > maxPushSubscriptionRecordBytes {
			return errors.New("push subscription record exceeds persistence limit")
		}
		organizationID := item.OrganizationID
		tenantCounts[organizationID]++
		if tenantCounts[organizationID] > maxPushSubscriptionsPerOrg {
			return errPushSubscriptionQuota
		}
		if tenantRecords[organizationID] == nil {
			tenantRecords[organizationID] = make(map[string]pushSubscriptionRecord)
		}
		tenantRecords[organizationID][id] = record
		storedRecords[id] = record
	}
	for _, records := range tenantRecords {
		encoded, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal organization push subscriptions: %w", err)
		}
		if len(encoded)+1 > maxPushSubscriptionBytesPerOrg {
			return errPushSubscriptionQuota
		}
	}
	encoded, err := json.MarshalIndent(storedRecords, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal push subscription file: %w", err)
	}
	if len(encoded)+1 > maxPushSubscriptionFileBytes {
		return errPushSubscriptionQuota
	}
	return nil
}

func pushSubscriptionID(token, userID, organizationID string) string {
	return "push_" + hashSecret(strings.TrimSpace(organizationID) + "\x00" + strings.TrimSpace(userID) + "\x00" + strings.TrimSpace(token))[:24]
}

func clonePushTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func clonePushSubscription(item PushSubscription) PushSubscription {
	item.RevokedAt = clonePushTime(item.RevokedAt)
	return item
}

func clonePushSubscriptions(items map[string]PushSubscription) map[string]PushSubscription {
	cloned := make(map[string]PushSubscription, len(items))
	for id, item := range items {
		cloned[id] = clonePushSubscription(item)
	}
	return cloned
}

func canonicalPushPersistentRoot(root string) (string, error) {
	absolute, err := filepath.Abs(filepath.Clean(strings.TrimSpace(root)))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}
