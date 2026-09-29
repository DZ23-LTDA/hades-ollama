package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"reflect"
	"strings"
)

func sanitizeProviderResponse(data []byte) string {
	return string(sanitizeProviderJSONBytes(data))
}

// providerCatalogOrigin is a display-only projection for tenant-facing
// catalogs. Provider paths and query strings may contain opaque credentials;
// the live configuration remains private to the manager.
func providerCatalogOrigin(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || endpointURLHasSensitiveMaterial(raw) {
		return "[REDACTED]"
	}
	return (&url.URL{Scheme: parsed.Scheme, Host: parsed.Host}).String()
}

// validateConfiguredEndpointURL is used before an endpoint configuration can
// enter a manager's in-memory state or manifest. URL query parsing is
// intentionally fail-closed: a malformed or ambiguously escaped URL is not
// safe to persist because it may be interpreted differently by the client and
// by an operator reviewing the manifest.
func validateConfiguredEndpointURL(raw string) error {
	if len(raw) > maxConfiguredEndpointURLBytes {
		return errors.New("endpoint URL exceeds the size limit")
	}
	if endpointURLHasSensitiveMaterial(raw) {
		return errors.New("endpoint URL contains credentials or an ambiguous query")
	}
	return nil
}

const maxConfiguredEndpointURLBytes = 16 << 10

func endpointURLHasSensitiveMaterial(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxConfiguredEndpointURLBytes {
		return true
	}
	if redacted := redactCredentialURLs(raw); redacted != raw {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return true
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return true
	}
	for key, values := range query {
		decodedKey := normalizeDLPKey(decodeURLComponentFully(key))
		if sensitiveDLPKey(decodedKey) || strings.Contains(decodedKey, "signature") || strings.Contains(decodedKey, "_sig") || strings.HasSuffix(decodedKey, "_sig") {
			return true
		}
		for _, value := range values {
			decodedValue := decodeURLComponentFully(value)
			if hasSensitiveEmbeddedJSONKey(decodedValue) {
				return true
			}
		}
	}
	for _, item := range dlpPatterns {
		if item.pattern.FindString(raw) != "" {
			return true
		}
	}
	return false
}

// unsafeEgressIP is the single policy for addresses used after DNS
// resolution. IsPrivate and IsGlobalUnicast alone are insufficient: Go
// deliberately treats documentation, benchmarking, and CGNAT ranges as
// global-unicast, and IPv4-mapped IPv6 values can otherwise bypass a family
// check. The policy is conservative because these addresses are used for
// DNS-pinned outbound connections.
func unsafeEgressIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// Normalize both dotted-quad values represented as 16 bytes and explicit
	// IPv4-mapped IPv6 answers before applying the same IPv4 special-use ranges.
	// The connector revalidates the actual connected peer against this policy.
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}
	address, ok := netip.AddrFromSlice(ip)
	if !ok || !address.IsGlobalUnicast() {
		return true
	}
	address = address.Unmap()
	if address.IsPrivate() || address.IsLoopback() || address.IsUnspecified() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() {
		return true
	}

	var blocked = [...]netip.Prefix{
		// RFC 6598 shared address space (CGNAT).
		netip.MustParsePrefix("100.64.0.0/10"),
		// RFC 2544 benchmarking.
		netip.MustParsePrefix("198.18.0.0/15"),
		// IPv4 special-use, documentation, and reserved ranges.
		netip.MustParsePrefix("0.0.0.0/8"),
		netip.MustParsePrefix("192.0.0.0/24"),
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("192.88.99.0/24"),
		netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("224.0.0.0/4"),
		netip.MustParsePrefix("240.0.0.0/4"),
		// IPv6 special-use, benchmarking, documentation, and reserved ranges.
		netip.MustParsePrefix("2001:0000::/32"),
		netip.MustParsePrefix("2001:0002::/48"),
		netip.MustParsePrefix("2001:0010::/28"),
		netip.MustParsePrefix("2001:0020::/28"),
		netip.MustParsePrefix("2001:db8::/32"),
		netip.MustParsePrefix("3fff::/20"),
		netip.MustParsePrefix("5f00::/16"),
	}
	for _, prefix := range blocked {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func canonicalizeResolvedIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return append(net.IP(nil), ipv4...)
	}
	return append(net.IP(nil), ip...)
}

func sanitizeProviderJSON(data json.RawMessage) (json.RawMessage, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, errors.New("provider response JSON could not be decoded")
	}
	encoded, err := json.Marshal(RedactValue(value))
	if err != nil {
		return nil, errors.New("provider response JSON could not be sanitized")
	}
	return encoded, nil
}

func sanitizeProviderJSONBytes(data []byte) []byte {
	if json.Valid(data) {
		redacted, err := sanitizeProviderJSON(data)
		if err == nil {
			return redacted
		}
	}
	return []byte(RedactDLP(string(data)))
}

const (
	maxOutboundDLPScanBytes = 8 << 20
	maxOutboundDLPNodes     = 100_000
)

var errOutboundPayloadBlocked = errors.New("outbound payload blocked by DLP policy")
var errOutboundPayloadLimit = errors.New("outbound payload limit exceeded")

// ValidateOutboundPayload rejects unsupported, oversized, or sensitive values
// before they are accepted for a workflow that may cross an external boundary.
// It never returns matched secret material.
func ValidateOutboundPayload(value any) error {
	return validateOutboundPayload(value)
}

// validateOutboundPayload is a fail-closed guard for payloads crossing a
// connector or MCP boundary. It scans both keys and nested string values and
// never returns the matched value to callers or logs.
func validateOutboundPayload(value any) error {
	_, err := normalizedOutboundPayload(value, maxOutboundDLPScanBytes)
	return err
}

func validateOutboundPayloadWithLimit(value any, limit int) error {
	_, err := normalizedOutboundPayload(value, limit)
	return err
}

func normalizedOutboundPayload(value any, limit int) (any, error) {
	if limit <= 0 || limit > 80<<20 {
		return nil, errOutboundPayloadBlocked
	}
	budget := outboundShapeBudget{nodes: maxOutboundDLPNodes, bytes: int64(limit), maxString: limit}
	if err := validateOutboundShape(reflect.ValueOf(value), 0, &budget); err != nil {
		if errors.Is(err, errOutboundPayloadLimit) {
			return nil, errOutboundPayloadLimit
		}
		return nil, errOutboundPayloadBlocked
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, errOutboundPayloadBlocked
	}
	if len(encoded) > limit {
		return nil, errOutboundPayloadLimit
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var normalized any
	if err := decoder.Decode(&normalized); err != nil {
		return nil, errOutboundPayloadBlocked
	}
	if err := scanOutboundValue(normalized, 0); err != nil {
		return nil, errOutboundPayloadBlocked
	}
	return normalized, nil
}

type outboundShapeBudget struct {
	nodes     int
	bytes     int64
	maxString int
}

func validateOutboundShape(value reflect.Value, depth int, budget *outboundShapeBudget) error {
	if depth > 64 || budget.nodes <= 0 {
		return errOutboundPayloadBlocked
	}
	budget.nodes--
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return nil
	case reflect.String:
		if value.Len() > budget.maxString {
			return errOutboundPayloadLimit
		}
		if len(ScanDLP(value.String())) > 0 {
			return errOutboundPayloadBlocked
		}
		budget.bytes -= int64(value.Len())
		if budget.bytes < 0 {
			return errOutboundPayloadLimit
		}
		return nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String || value.Len() > maxOutboundDLPNodes {
			return errOutboundPayloadBlocked
		}
		iterator := value.MapRange()
		for iterator.Next() {
			key := iterator.Key().String()
			if len(key) > budget.maxString {
				return errOutboundPayloadLimit
			}
			if sensitiveDLPKey(key) {
				return errOutboundPayloadBlocked
			}
			budget.bytes -= int64(len(key))
			if budget.bytes < 0 {
				return errOutboundPayloadLimit
			}
			if err := validateOutboundShape(iterator.Value(), depth+1, budget); err != nil {
				return err
			}
		}
		return nil
	case reflect.Struct:
		typeOf := value.Type()
		encoded, err := json.Marshal(value.Interface())
		if err != nil {
			return errOutboundPayloadBlocked
		}
		if int64(len(encoded)) > budget.bytes {
			return errOutboundPayloadLimit
		}
		for index := 0; index < value.NumField(); index++ {
			field := typeOf.Field(index)
			if field.PkgPath != "" {
				continue
			}
			jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
			if jsonName == "-" {
				continue
			}
			if jsonName == "" {
				jsonName = field.Name
			}
			if sensitiveDLPKey(jsonName) {
				return errOutboundPayloadBlocked
			}
			if err := validateOutboundShape(value.Field(index), depth+1, budget); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice && value.Type().Elem().Kind() == reflect.Uint8 {
			if value.Len() > budget.maxString {
				return errOutboundPayloadLimit
			}
			if len(ScanDLP(string(value.Bytes()))) > 0 {
				return errOutboundPayloadBlocked
			}
			budget.bytes -= int64(value.Len())
			if budget.bytes < 0 {
				return errOutboundPayloadLimit
			}
			return nil
		}
		if value.Len() > maxOutboundDLPNodes {
			return errOutboundPayloadBlocked
		}
		for index := 0; index < value.Len(); index++ {
			if err := validateOutboundShape(value.Index(index), depth+1, budget); err != nil {
				return err
			}
		}
		return nil
	default:
		return errOutboundPayloadBlocked
	}
}

func scanOutboundValue(value any, depth int) error {
	if depth > 64 {
		return errOutboundPayloadBlocked
	}
	switch typed := value.(type) {
	case nil, bool, json.Number:
		return nil
	case string:
		if len(ScanDLP(typed)) > 0 {
			return errOutboundPayloadBlocked
		}
		trimmed := strings.TrimSpace(typed)
		if len(trimmed) > 1 && (trimmed[0] == '{' || trimmed[0] == '[') {
			var nested any
			if err := json.Unmarshal([]byte(trimmed), &nested); err == nil {
				if _, isContainer := nested.(map[string]any); isContainer {
					return scanOutboundValue(nested, depth+1)
				}
				if _, isContainer := nested.([]any); isContainer {
					return scanOutboundValue(nested, depth+1)
				}
			}
		}
	case []any:
		for _, item := range typed {
			if err := scanOutboundValue(item, depth+1); err != nil {
				return errOutboundPayloadBlocked
			}
		}
	case map[string]any:
		for key, item := range typed {
			if sensitiveDLPKey(key) {
				return errOutboundPayloadBlocked
			}
			if err := scanOutboundValue(item, depth+1); err != nil {
				return errOutboundPayloadBlocked
			}
		}
	default:
		return errOutboundPayloadBlocked
	}
	return nil
}
