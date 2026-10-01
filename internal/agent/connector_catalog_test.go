package agent

import (
	"net/http"
	"strings"
	"testing"
)

func TestConnectorCatalogIncludesOperationalIntegrations(t *testing.T) {
	entries := ConnectorCatalog()
	if len(entries) < 20 {
		t.Fatalf("expected broad connector catalog, got %d entries", len(entries))
	}
	wanted := map[string]bool{
		"composio":         false,
		"google-workspace": false,
		"github":           false,
		"woovi-openpix":    false,
		"fiscal-invoicing": false,
		"shopify":          false,
		"tiktok-business":  false,
		"vercel":           false,
	}
	for _, entry := range entries {
		if _, ok := wanted[entry.ID]; ok {
			wanted[entry.ID] = true
		}
		if entry.Status == "available" && entry.Auth == "" {
			t.Fatalf("available catalog entry %q has no auth contract", entry.ID)
		}
	}
	for id, found := range wanted {
		if !found {
			t.Errorf("catalog is missing %q", id)
		}
	}
}

func TestConnectorCatalogDoesNotClaimFiscalIssuerByDefault(t *testing.T) {
	for _, entry := range ConnectorCatalog() {
		if entry.ID == "fiscal-invoicing" && entry.Status != "provider_selection_required" {
			t.Fatalf("fiscal catalog entry must require provider selection, got %q", entry.Status)
		}
	}
}

func TestConnectorCatalogIDsAreUniqueAndSetupIsExplicit(t *testing.T) {
	seen := map[string]bool{}
	for _, entry := range ConnectorCatalog() {
		if seen[entry.ID] {
			t.Fatalf("duplicate connector id %q", entry.ID)
		}
		seen[entry.ID] = true
		if entry.Name == "" || entry.Description == "" || entry.Auth == "" {
			t.Fatalf("connector %q is missing name, description or auth", entry.ID)
		}
	}
	for _, id := range []string{"pinterest", "youtube", "linkedin", "telegram", "gmail", "mercado-pago", "resend", "twilio", "firebase", "sentry", "aws"} {
		if !seen[id] {
			t.Errorf("catalog is missing %q", id)
		}
	}
}

func TestQuickConnectEntriesExistWithHTTPS(t *testing.T) {
	byID := map[string]ConnectorCatalogEntry{}
	for _, entry := range ConnectorCatalog() {
		byID[entry.ID] = entry
	}
	for id, qc := range quickConnects {
		if len(QuickConnectOperations(id)) == 0 {
			continue
		}
		entry, ok := byID[id]
		if !ok {
			t.Errorf("quick connect %q is not in the catalog", id)
			continue
		}
		if !entry.QuickConnect {
			t.Errorf("%s should be quick-connect", id)
		}
		if qc.selfHosted != (qc.base == "") {
			t.Errorf("%s: self-hosted entries have no base URL and hosted ones must have one", id)
		}
		if qc.base != "" && !strings.HasPrefix(qc.base, "https://") {
			t.Errorf("%s base %q must be https", id, qc.base)
		}
		if qc.header != "" && !validAuthHeader(qc.header) {
			t.Errorf("%s header %q is not accepted by connector validation", id, qc.header)
		}
		if qc.scheme != "" && !validAuthScheme(qc.scheme) {
			t.Errorf("%s scheme %q is not accepted", id, qc.scheme)
		}
	}
	for id, operations := range quickConnectOperations {
		entry, ok := byID[id]
		if !ok || !entry.QuickConnect {
			t.Errorf("policy %q is not exposed as a quick-connect catalog entry", id)
		}
		if len(operations) == 0 {
			t.Errorf("policy %q is empty", id)
		}
		for _, operation := range operations {
			if operation.Name != "read" || len(operation.Methods) == 0 {
				t.Errorf("%s operation must be named read and have methods: %+v", id, operation)
			}
			for _, method := range operation.Methods {
				if method != "GET" {
					t.Errorf("%s allows non-read method %q", id, method)
				}
			}
			for _, prefix := range operation.PathPrefixes {
				if prefix == "" || prefix == "/" {
					t.Errorf("%s has broad path prefix %q", id, prefix)
				}
			}
		}
	}
	if got := ConnectorTokenEnv("mercado-pago"); got != "OLLAMA_CONNECTOR_MERCADO_PAGO_TOKEN" {
		t.Fatalf("token env = %q", got)
	}
}

func TestQuickConnectOperationsReturnsDefensiveCopy(t *testing.T) {
	operations := QuickConnectOperations("resend")
	if len(operations) != 1 || len(operations[0].Methods) != 1 || len(operations[0].PathPrefixes) != 1 {
		t.Fatalf("unexpected resend policy: %+v", operations)
	}
	operations[0].Name = "write"
	operations[0].Methods[0] = "DELETE"
	operations[0].PathPrefixes[0] = "/"

	stored := QuickConnectOperations("resend")
	if stored[0].Name != "read" || stored[0].Methods[0] != "GET" || stored[0].PathPrefixes[0] != "/domains" {
		t.Fatalf("caller mutated stored quick-connect policy: %+v", stored)
	}
}

func TestConnectorAuthHeader(t *testing.T) {
	for _, tc := range []struct {
		header, scheme, wantHeader, wantValue string
	}{
		{"", "", "Authorization", "Bearer k"},
		{"", "raw", "Authorization", "k"},
		{"", "Token", "Authorization", "Token k"},
		{"X-Appwrite-Key", "", "X-Appwrite-Key", "k"},
	} {
		h, v := connectorAuth(ConnectorConfig{AuthHeader: tc.header, AuthScheme: tc.scheme}, "k")
		if h != tc.wantHeader || v != tc.wantValue {
			t.Errorf("connectorAuth(%q,%q) = %s: %s", tc.header, tc.scheme, h, v)
		}
	}
	for _, bad := range []string{"Host", "Cookie", "X Bad", "Content-Length", ""} {
		if validAuthHeader(bad) {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

func TestEveryCatalogConnectorUsesSharedSafeEgressClient(t *testing.T) {
	manager := NewConnectorManager()
	if manager.client == nil || manager.client.Transport == nil || manager.client.CheckRedirect == nil {
		t.Fatal("connector catalog has no hardened egress client")
	}
	transport, ok := manager.client.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil || transport.DialContext == nil {
		t.Fatal("connector catalog manager is not backed by the zero-trust transport")
	}
	entries := ConnectorCatalog()
	if len(entries) == 0 {
		t.Fatal("connector catalog is empty")
	}
	for _, entry := range entries {
		if manager.client.Transport == nil {
			t.Fatalf("catalog connector %q lost its shared egress transport", entry.ID)
		}
	}
}
