package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPushSubscriptionQuotasBoundPerOrganizationCountAndBytes(t *testing.T) {
	tooMany := make(map[string]PushSubscription, maxPushSubscriptionsPerOrg+1)
	for index := 0; index <= maxPushSubscriptionsPerOrg; index++ {
		id := "push-count-" + strconv.Itoa(index)
		tooMany[id] = PushSubscription{ID: id, Token: "token", TokenCiphertext: "ciphertext", UserID: "user", OrganizationID: "org-a"}
	}
	if err := validatePushSubscriptionQuotas(tooMany); !errors.Is(err, errPushSubscriptionQuota) {
		t.Fatalf("per-organization count error=%v, want quota error", err)
	}

	tooManyBytes := make(map[string]PushSubscription, 150)
	for index := 0; index < 150; index++ {
		id := "push-bytes-" + strconv.Itoa(index)
		tooManyBytes[id] = PushSubscription{ID: id, Token: "token", TokenCiphertext: strings.Repeat("x", 15_000), UserID: "user", OrganizationID: "org-a"}
	}
	if err := validatePushSubscriptionQuotas(tooManyBytes); !errors.Is(err, errPushSubscriptionQuota) {
		t.Fatalf("per-organization byte quota error=%v, want quota error", err)
	}
}

func TestPushSubscriptionQuotaMatchesIndentedPersistedRepresentation(t *testing.T) {
	items := make(map[string]PushSubscription)
	records := make(map[string]pushSubscriptionRecord)
	compactBytes := 0
	for index := 0; index < maxPushSubscriptionsPerOrg; index++ {
		id := "push-indent-" + strconv.Itoa(index)
		item := PushSubscription{ID: id, Token: "token", TokenCiphertext: strings.Repeat("x", 15_000), Platform: "ios", UserID: "user", OrganizationID: "org-a"}
		items[id] = item
		record := pushSubscriptionRecord{ID: item.ID, TokenCiphertext: item.TokenCiphertext, Platform: item.Platform, UserID: item.UserID, OrganizationID: item.OrganizationID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, RevokedAt: item.RevokedAt}
		records[id] = record
		compact, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		compactBytes += len(compact)
		persisted, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if compactBytes < maxPushSubscriptionBytesPerOrg && len(persisted)+1 > maxPushSubscriptionBytesPerOrg {
			if err := validatePushSubscriptionQuotas(items); !errors.Is(err, errPushSubscriptionQuota) {
				t.Fatalf("indented persisted representation over quota accepted: %v", err)
			}
			return
		}
	}
	t.Fatal("fixture did not straddle the compact-versus-indented organization byte limit")
}

func TestPushServiceRejectsNonLoopbackHTTPAndEndpointUserinfo(t *testing.T) {
	for _, endpoint := range []string{
		"http://localhost.evil.example/push",
		"http://127.0.0.1.evil.example/push",
		"http://localhost:secret@evil.example/push",
		"http://127.0.0.1@evil.example/push",
		"ftp://localhost/push",
	} {
		t.Run(endpoint, func(t *testing.T) {
			if _, err := NewPushService("", endpoint); err == nil {
				t.Fatalf("unsafe push endpoint was accepted: %q", endpoint)
			}
		})
	}
	for _, endpoint := range []string{"http://localhost/push", "http://127.0.0.1:43123/push", "http://[::1]/push", "https://push.example.test/send"} {
		t.Run("allowed/"+endpoint, func(t *testing.T) {
			if _, err := NewPushService("", endpoint); err != nil {
				t.Fatalf("valid push endpoint was rejected: %v", err)
			}
		})
	}
}

func TestPushServiceDoesNotEchoProviderErrorBodyOrFollowRedirects(t *testing.T) {
	const echoedSecret = "opaque-provider-diagnostic-secret-do-not-leak"
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusTemporaryRedirect)
		_, _ = w.Write([]byte(echoedSecret))
	}))
	defer source.Close()
	push, err := NewPushService("", source.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := push.Register("synthetic-token", "ios", "user", "org"); err != nil {
		t.Fatal(err)
	}
	err = push.NotifyOrganization(context.Background(), "org", "title", "body", nil)
	if err == nil || strings.Contains(err.Error(), echoedSecret) {
		t.Fatalf("provider error leaked response content or succeeded: %v", err)
	}
	if redirected.Load() != 0 {
		t.Fatalf("push client followed redirect to another endpoint %d times", redirected.Load())
	}
}

func TestPushServiceRegisterRollsBackOnPersistenceFailure(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-store-test-key-with-sufficient-length")
	root := t.TempDir()
	service, err := NewPushService(root, "http://127.0.0.1:43123")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Register("push-token", "android", "user-a", "org-a"); err == nil {
		t.Fatal("register unexpectedly succeeded without a writable root")
	}
	if got := service.ListOrganization("org-a"); len(got) != 0 {
		t.Fatalf("subscription remained in memory after persistence failure: %+v", got)
	}
}

func TestPushServiceEncryptsTokensAtRestAndReloads(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-store-test-key-with-sufficient-length")
	root := t.TempDir()
	service, err := NewPushService(root, "https://push.example.test")
	if err != nil {
		t.Fatal(err)
	}
	const token = "ExponentPushToken[synthetic-test-token]"
	item, err := service.Register(token, "android", "user-a", "org-a")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "subscriptions.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), token) || !strings.Contains(string(data), "token_ciphertext") {
		t.Fatalf("push token was not encrypted at rest: %s", data)
	}
	reloaded, err := NewPushService(root, "https://push.example.test")
	if err != nil {
		t.Fatal(err)
	}
	items := reloaded.ListOrganization("org-a")
	if len(items) != 1 || items[0].ID != item.ID || items[0].Token != token {
		t.Fatalf("reloaded subscription=%+v", items)
	}
	encoded, err := json.Marshal(items[0])
	if err != nil || strings.Contains(string(encoded), token) || strings.Contains(string(encoded), `"token"`) {
		t.Fatalf("push subscription JSON exposed token material: err=%v json=%s", err, encoded)
	}
}

func TestPushServiceMigratesLegacyPlaintextTokenFile(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-store-test-key-with-sufficient-length")
	root := t.TempDir()
	const token = "ExponentPushToken[legacy-synthetic-token]"
	now := time.Now().UTC()
	legacy := map[string]pushSubscriptionRecord{
		"push_legacy": {ID: "push_legacy", Token: token, Platform: "ios", UserID: "user-a", OrganizationID: "org-a", CreatedAt: now, UpdatedAt: now},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "subscriptions.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := NewPushService(root, "https://push.example.test")
	if err != nil {
		t.Fatal(err)
	}
	items := service.ListOrganization("org-a")
	if len(items) != 1 || items[0].Token != token {
		t.Fatalf("migrated subscriptions=%+v", items)
	}
	migrated, err := os.ReadFile(filepath.Join(root, "subscriptions.json"))
	if err != nil || strings.Contains(string(migrated), token) || !strings.Contains(string(migrated), "token_ciphertext") {
		t.Fatalf("legacy file was not migrated to encrypted token storage: err=%v", err)
	}
}

func TestPushServiceSubscriptionIDBindsOrganizationAndUser(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-store-test-key-with-sufficient-length")
	root := t.TempDir()
	service, err := NewPushService(root, "https://push.example.test")
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Register("same-token", "android", "user-a", "org-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Register("same-token", "android", "user-a", "org-b")
	if err != nil {
		t.Fatal(err)
	}
	third, err := service.Register("same-token", "android", "user-b", "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || first.ID == third.ID || second.ID == third.ID {
		t.Fatalf("subscription IDs are not tenant/user bound: %q %q %q", first.ID, second.ID, third.ID)
	}
	if got := service.ListOrganization("org-a"); len(got) != 2 {
		t.Fatalf("org-a subscriptions=%+v", got)
	}
	if got := service.ListOrganization("org-b"); len(got) != 1 || got[0].ID != second.ID {
		t.Fatalf("org-b subscriptions=%+v", got)
	}
}

func TestPushServiceConcurrentInstancesDoNotLoseRegistrations(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-store-test-key-with-sufficient-length")
	root := t.TempDir()
	first, err := NewPushService(root, "https://push.example.test")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewPushService(root, "https://push.example.test")
	if err != nil {
		t.Fatal(err)
	}
	errs := make(chan error, 2)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		_, err := first.Register("token-a", "android", "user-a", "org-a")
		errs <- err
	}()
	go func() {
		defer group.Done()
		_, err := second.Register("token-b", "ios", "user-b", "org-a")
		errs <- err
	}()
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := first.ListOrganization("org-a"); len(got) != 2 {
		t.Fatalf("registrations lost across instances: %+v", got)
	}
}

func TestPushServiceSubscriptionCopiesRevocationPointer(t *testing.T) {
	stored := time.Now().UTC()
	item := PushSubscription{RevokedAt: &stored}
	copy := clonePushSubscription(item)
	if copy.RevokedAt == item.RevokedAt {
		t.Fatal("subscription clone reused mutable revocation pointer")
	}
	copy.RevokedAt = nil
	if item.RevokedAt == nil {
		t.Fatal("subscription clone mutation changed source revocation pointer")
	}
}

func TestPushDialRejectsDisallowedResolvedAddressesBeforeTCP(t *testing.T) {
	for _, tc := range []struct {
		name          string
		allowLoopback bool
		ips           []net.IP
	}{
		{name: "localhost resolving public", allowLoopback: true, ips: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("203.0.113.8")}},
		{name: "public endpoint resolving private", ips: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("203.0.113.8")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			accepted := make(chan struct{}, 1)
			go func() {
				conn, acceptErr := listener.Accept()
				if acceptErr == nil {
					_ = conn.Close()
				}
				accepted <- struct{}{}
			}()
			lookup := func(context.Context, string) ([]net.IPAddr, error) {
				result := make([]net.IPAddr, 0, len(tc.ips))
				for _, ip := range tc.ips {
					result = append(result, net.IPAddr{IP: ip})
				}
				return result, nil
			}
			_, err = pushDialContextWithResolver(context.Background(), "tcp", "provider.example:"+portOf(listener.Addr().String()), tc.allowLoopback, lookup)
			if err == nil {
				t.Fatal("push dial accepted a mixed or private resolution")
			}
			select {
			case <-accepted:
				t.Fatal("disallowed resolution reached the TCP listener")
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
}

func TestPushDialPinsExactLoopbackAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			accepted <- conn
		}
	}()
	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
	}
	conn, err := pushDialContextWithResolver(context.Background(), "tcp", "localhost:"+portOf(listener.Addr().String()), true, lookup)
	if err != nil {
		t.Fatalf("verified loopback dial failed: %v", err)
	}
	_ = conn.Close()
	select {
	case acceptedConn := <-accepted:
		_ = acceptedConn.Close()
	case <-time.After(time.Second):
		t.Fatal("pinned loopback address was not dialed")
	}
}

func TestPushRevocationFencesLaterDeliveryInBatch(t *testing.T) {
	var service *PushService
	var sends atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sends.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	var err error
	service, err = NewPushService("", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Register("token-first", "ios", "user-a", "org-a")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Register("token-second", "ios", "user-a", "org-a")
	if err != nil {
		t.Fatal(err)
	}
	var preSendCalls int32
	oldHook := pushBeforeDeliveryHook
	pushBeforeDeliveryHook = func(item PushSubscription) {
		if atomic.AddInt32(&preSendCalls, 1) == 2 {
			if err := service.Revoke(item.ID, "org-a", "user-a"); err != nil {
				t.Errorf("revoke pending subscription: %v", err)
			}
		}
	}
	defer func() { pushBeforeDeliveryHook = oldHook }()
	if err := service.NotifyOrganization(context.Background(), "org-a", "title", "body", nil); err != nil {
		t.Fatal(err)
	}
	if got := sends.Load(); got != 1 {
		t.Fatalf("notifications sent after revocation=%d, want 1", got)
	}
	if err := service.Revoke(first.ID, "org-b", "user-a"); !errors.Is(err, ErrPushSubscriptionNotFound) {
		t.Fatalf("cross-organization revoke error=%v", err)
	}
}

func TestPushRevocationFencesPendingSendAfterInitialListing(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-revocation-race-test-key")
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	push, err := NewPushService(t.TempDir(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	push.client = server.Client()
	subscription, err := push.Register("ExponentPushToken[revoked]", "android", "user_race", "org_race")
	if err != nil {
		t.Fatal(err)
	}
	oldHook := pushBeforeDeliveryHook
	pushBeforeDeliveryHook = func(item PushSubscription) {
		pushBeforeDeliveryHook = nil
		if item.ID != subscription.ID {
			t.Errorf("hook subscription=%q, want %q", item.ID, subscription.ID)
		}
		if err := push.Revoke(item.ID, "org_race", "user_race"); err != nil {
			t.Errorf("revoke before send: %v", err)
		}
	}
	defer func() { pushBeforeDeliveryHook = oldHook }()
	if err := push.NotifyOrganization(context.Background(), "org_race", "title", "body", nil); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 0 {
		t.Fatalf("revoked pending subscription received %d provider request(s)", requests.Load())
	}
}

func TestPushProviderRequestDoesNotHoldSubscriptionLock(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "push-lock-release-test-key")
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		started <- struct{}{}
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	push, err := NewPushService(t.TempDir(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	push.client = server.Client()
	subscription, err := push.Register("ExponentPushToken[lock-release]", "android", "user_lock", "org_lock")
	if err != nil {
		t.Fatal(err)
	}
	notifyDone := make(chan error, 1)
	go func() { notifyDone <- push.NotifyOrganization(context.Background(), "org_lock", "title", "body", nil) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("provider request did not start")
	}
	revoked := make(chan error, 1)
	go func() { revoked <- push.Revoke(subscription.ID, "org_lock", "user_lock") }()
	select {
	case err := <-revoked:
		if err != nil {
			close(release)
			t.Fatalf("revoke while provider request was blocked: %v", err)
		}
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("provider network I/O blocked subscription revocation")
	}
	close(release)
	select {
	case err := <-notifyDone:
		if err != nil {
			t.Fatalf("notify result: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("notification did not finish after provider was released")
	}
}
