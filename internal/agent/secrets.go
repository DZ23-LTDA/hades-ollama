package agent

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type SecretMetadata struct {
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SecretStore struct {
	mu     sync.RWMutex
	root   string
	values map[string]SecretValue
}

type SecretValue struct {
	Ciphertext string    `json:"ciphertext"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewSecretStore(root string) (*SecretStore, error) {
	store := &SecretStore{root: strings.TrimSpace(root), values: map[string]SecretValue{}}
	if store.root != "" {
		if err := os.MkdirAll(store.root, 0o700); err != nil {
			return nil, err
		}
		if err := withFileLock(filepath.Join(store.root, ".secrets.lock"), func() error {
			return store.loadLocked()
		}); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *SecretStore) loadLocked() error {
	values := map[string]SecretValue{}
	if err := readJSON(filepathJoin(s.root, "secrets.json"), &values); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if values == nil {
		values = map[string]SecretValue{}
	}
	s.values = values
	return nil
}

func (s *SecretStore) withStateLocked(run func() error) error {
	if s.root == "" {
		return run()
	}
	return withFileLock(filepath.Join(s.root, ".secrets.lock"), func() error {
		if err := s.loadLocked(); err != nil {
			return err
		}
		return run()
	})
}

func (s *SecretStore) Set(name, value string) error {
	name = normalizeSecretName(name)
	if name == "" || strings.TrimSpace(value) == "" {
		return errors.New("secret name and value are required")
	}
	ciphertext, err := encryptCredential(value)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withStateLocked(func() error {
		previous := s.values
		next := make(map[string]SecretValue, len(previous)+1)
		for key, entry := range previous {
			next[key] = entry
		}
		next[name] = SecretValue{Ciphertext: ciphertext, UpdatedAt: time.Now().UTC()}
		if s.root != "" {
			if err := writeJSONAtomic(filepathJoin(s.root, "secrets.json"), next); err != nil {
				return err
			}
		}
		s.values = next
		return nil
	})
}

func (s *SecretStore) Get(name string) (string, error) {
	name = normalizeSecretName(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	var result string
	err := s.withStateLocked(func() error {
		value, ok := s.values[name]
		if !ok {
			return os.ErrNotExist
		}
		decrypted, err := decryptCredential(value.Ciphertext)
		if err != nil {
			return err
		}
		result = decrypted
		return nil
	})
	return result, err
}

func (s *SecretStore) Delete(name string) error {
	name = normalizeSecretName(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withStateLocked(func() error {
		previous := s.values
		if _, ok := previous[name]; !ok {
			return os.ErrNotExist
		}
		next := make(map[string]SecretValue, len(previous)-1)
		for key, value := range previous {
			if key != name {
				next[key] = value
			}
		}
		if s.root != "" {
			if err := writeJSONAtomic(filepathJoin(s.root, "secrets.json"), next); err != nil {
				return err
			}
		}
		s.values = next
		return nil
	})
}

func (s *SecretStore) List() []SecretMetadata {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.withStateLocked(func() error { return nil }); err != nil {
		return nil
	}
	items := make([]SecretMetadata, 0, len(s.values))
	for name, value := range s.values {
		items = append(items, SecretMetadata{Name: name, UpdatedAt: value.UpdatedAt})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (s *SecretStore) persistLocked() error {
	if s.root == "" {
		return nil
	}
	return writeJSONAtomic(filepathJoin(s.root, "secrets.json"), s.values)
}

func normalizeSecretName(name string) string {
	name = strings.TrimSpace(name)
	for _, character := range name {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' && character != '-' && character != '.' {
			return ""
		}
	}
	return name
}

type DLPFinding struct {
	Kind     string `json:"kind"`
	Redacted string `json:"redacted"`
}

var dlpPatterns = []struct {
	kind    string
	pattern *regexp.Regexp
}{
	{"private_key", regexp.MustCompile(`(?s)-----BEGIN [A-Z ]+PRIVATE KEY-----.*?-----END [A-Z ]+PRIVATE KEY-----`)},
	{"github_token", regexp.MustCompile(`(?i)\b(?:ghp|github_pat)_[A-Za-z0-9_]{20,}\b`)},
	{"openrouter_token", regexp.MustCompile(`\bsk-or-v1-[A-Za-z0-9_-]{20,}\b`)},
	{"openai_token", regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}\b`)},
	{"xai_token", regexp.MustCompile(`\bxai-[A-Za-z0-9_-]{20,}\b`)},
	{"aws_access_key", regexp.MustCompile(`\bAKIA[A-Z0-9]{16}\b`)},
	{"slack_token", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{16,}\b`)},
	{"bearer_token", regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{16,}`)},
	{"basic_authorization", regexp.MustCompile(`(?i)\b(?:authorization\s*:\s*)?basic\s+[A-Za-z0-9+/=_-]{8,}`)},
	{"cookie_credential", regexp.MustCompile(`(?i)\b(?:cookie|set-cookie)\s*:\s*[^\s;=]+=\s*[^\s;]+|\b(?:session|sid|auth|jwt|access_token|refresh_token|phpsessid)=\s*[^\s;,&]+`)},
	{"jwt_token", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`)},
	{"credential_assignment", regexp.MustCompile(`(?i)(?:password|passwd|secret|credential|credentials|token|api[_-]?key|access[_-]?token|refresh[_-]?token)\s*[:=]\s*["']?[A-Za-z0-9._~+/=-]{1,}["']?`)},
}

// embeddedJSONStringKeyPattern recognizes both ordinary JSON property names
// and property names embedded in escaped JSON strings. The captured content is
// decoded with encoding/json before sensitiveDLPKey checks it, so e.g.
// `api\u005fkey` is treated exactly like `api_key`.
var embeddedJSONStringKeyPattern = regexp.MustCompile(`\\*"((?:\\.|[^"\\])+?)\\*"\s*:`)
var embeddedSingleQuotedKeyPattern = regexp.MustCompile(`\\*'((?:\\.|[^'\\])+?)\\*'\s*:`)
var embeddedUnicodeEscapePattern = regexp.MustCompile(`\\u[0-9a-fA-F]{4}`)
var dlpEscapeNormalizer = strings.NewReplacer(`\\`, `\`, `\"`, `"`, `\'`, `'`)

func normalizeDLPEscapes(text string) string {
	var out strings.Builder
	last := 0
	for _, match := range embeddedUnicodeEscapePattern.FindAllStringIndex(text, -1) {
		precedingSlashes := 0
		for index := match[0] - 1; index >= 0 && text[index] == '\\'; index-- {
			precedingSlashes++
		}
		if precedingSlashes%2 == 1 {
			continue
		}
		escape := text[match[0]:match[1]]
		var decoded string
		if err := json.Unmarshal([]byte(`"`+escape+`"`), &decoded); err != nil {
			continue
		}
		out.WriteString(text[last:match[0]])
		out.WriteString(decoded)
		last = match[1]
	}
	if last > 0 {
		out.WriteString(text[last:])
		text = out.String()
	}
	return dlpEscapeNormalizer.Replace(text)
}

func hasSensitiveEmbeddedJSONKey(text string) bool {
	// Tool/provider output can contain JSON nested inside one or more JSON
	// strings. Peel a small, fixed number of escape layers and decode JSON
	// Unicode escapes so escaped quotes/braces cannot conceal property names.
	for depth := 0; depth < 8; depth++ {
		if hasSensitiveQuotedKey(text, embeddedJSONStringKeyPattern) || hasSensitiveQuotedKey(text, embeddedSingleQuotedKeyPattern) {
			return true
		}
		next := normalizeDLPEscapes(text)
		if next == text {
			return false
		}
		if depth == 7 {
			// We cannot prove an over-depth escaped value safe within the
			// bounded inspection budget, so fail closed rather than leak it.
			return true
		}
		text = next
	}
	return true
}

func hasSensitiveQuotedKey(text string, pattern *regexp.Regexp) bool {
	for offset := 0; offset < len(text); {
		match := pattern.FindStringSubmatchIndex(text[offset:])
		if match == nil {
			return false
		}
		key := text[offset+match[2] : offset+match[3]]
		if len(key) > 256 {
			// Sensitive-key normalization is intentionally bounded. A property
			// name larger than the inspection budget is ambiguous; withhold it.
			return true
		}
		keyJSON := `"` + key + `"`
		if json.Unmarshal([]byte(keyJSON), &key) != nil {
			key = ""
		}
		if sensitiveDLPKey(key) {
			return true
		}
		next := offset + match[1]
		if next <= offset {
			return false
		}
		offset = next
	}
	return false
}

// dlpURLPattern deliberately matches the whole URL-like token, including
// escaped and percent-encoded separators. Validation below decodes a bounded
// number of layers and redacts the complete candidate when it cannot prove it
// is safe.
var dlpURLPattern = regexp.MustCompile(`(?i)\bhttps?(?::|\\:|%3a|%253a)[^\s"'<>]+`)
var dlpURLTokenPattern = regexp.MustCompile(`[^\s"'<>]+`)

func redactCredentialURLs(text string) string {
	text = dlpURLPattern.ReplaceAllStringFunc(text, func(candidate string) string {
		if credentialURLCandidate(candidate) {
			return "[REDACTED]"
		}
		return candidate
	})
	// A URL whose separators are percent encoded does not match the prefix
	// pattern. Inspect all non-quoted tokens so nested encodings such as
	// %2568%2574... cannot bypass the URL scanner. Redacting the complete token
	// is intentional: its boundaries are ambiguous after decoding.
	return dlpURLTokenPattern.ReplaceAllStringFunc(text, func(candidate string) string {
		if credentialURLCandidate(candidate) {
			return "[REDACTED]"
		}
		return candidate
	})
}

func credentialURLCandidate(candidate string) bool {
	current := candidate
	for depth := 0; depth < 8; depth++ {
		current = normalizeURLEscapes(current)
		if start := urlSchemeStart(current); start >= 0 {
			urlCandidate := current[start:]
			if credentialURLValue(urlCandidate) {
				return true
			}
			decoded, err := url.PathUnescape(urlCandidate)
			if err != nil {
				return true
			}
			if decoded == urlCandidate {
				return false
			}
			current = current[:start] + decoded
			continue
		}
		decoded, err := url.PathUnescape(current)
		if err != nil {
			// A URL-like candidate with a malformed escape cannot be safely
			// interpreted by all consumers; withhold the whole candidate.
			if strings.Contains(strings.ToLower(current), "http") {
				return true
			}
			return false
		}
		if decoded == current {
			return false
		}
		current = decoded
	}
	// An over-depth nested URL is ambiguous within the bounded inspection
	// budget. If it looked URL-like at any point, fail closed.
	return strings.Contains(strings.ToLower(current), "http")
}

func normalizeURLEscapes(value string) string {
	for depth := 0; depth < 8; depth++ {
		next := strings.ReplaceAll(value, `\/`, `/`)
		next = strings.ReplaceAll(next, `\:`, `:`)
		if next == value {
			return value
		}
		value = next
	}
	return value
}

func urlSchemeStart(value string) int {
	lower := strings.ToLower(value)
	for _, scheme := range []string{"http://", "https://"} {
		if index := strings.Index(lower, scheme); index >= 0 {
			return index
		}
	}
	return -1
}

func credentialURLValue(candidate string) bool {
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return true
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	// url.URL.Query silently discards ParseQuery errors. Use ParseQuery
	// directly so semicolon-separated or otherwise malformed queries are
	// treated as ambiguous and the complete URL is redacted.
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return true
	}
	for key := range query {
		key = decodeURLComponentFully(key)
		normalized := normalizeDLPKey(key)
		if sensitiveDLPKey(normalized) || strings.Contains(normalized, "signature") {
			return true
		}
		for _, part := range strings.Split(normalized, "_") {
			if part == "sig" {
				return true
			}
		}
	}
	for _, values := range query {
		for _, value := range values {
			value = decodeURLComponentFully(value)
			if urlSchemeStart(value) >= 0 && credentialURLCandidate(value) {
				return true
			}
		}
	}
	return false
}

func decodeURLComponentFully(value string) string {
	for depth := 0; depth < 8; depth++ {
		next, err := url.QueryUnescape(value)
		if err != nil || next == value {
			return value
		}
		value = next
	}
	return value
}

func ScanDLP(text string) []DLPFinding {
	findings := []DLPFinding{}
	if hasSensitiveEmbeddedJSONKey(text) {
		findings = append(findings, DLPFinding{Kind: "sensitive_json_key", Redacted: "[REDACTED]"})
	}
	if redactCredentialURLs(text) != text {
		findings = append(findings, DLPFinding{Kind: "credential_url", Redacted: "[REDACTED]"})
	}
	for _, item := range dlpPatterns {
		if match := item.pattern.FindString(text); match != "" {
			findings = append(findings, DLPFinding{Kind: item.kind, Redacted: item.pattern.ReplaceAllString(match, "[REDACTED]")})
		}
	}
	return findings
}

func RedactDLP(text string) string {
	if hasSensitiveEmbeddedJSONKey(text) {
		// The value may be arbitrary and the JSON may be escaped or nested;
		// withhold the whole string rather than risk leaking an unrecognized
		// credential representation in an error or free-form tool result.
		return "[REDACTED]"
	}
	text = redactCredentialURLs(text)
	for _, item := range dlpPatterns {
		text = item.pattern.ReplaceAllString(text, "[REDACTED]")
	}
	return text
}

// RedactValue recursively removes credential-shaped strings from values that
// are about to be persisted, emitted as events, or serialized externally.
// JSON-compatible values are normalized to JSON-compatible maps/slices so
// nested tool responses cannot bypass the string redaction path.
func RedactValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return RedactDLP(typed)
	case []byte:
		return RedactDLP(string(typed))
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if sensitiveDLPKey(key) {
				result[key] = "[REDACTED]"
				continue
			}
			result[key] = RedactValue(item)
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = RedactValue(item)
		}
		return result
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
		return value
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return value
		}
		var normalized any
		if err := json.Unmarshal(encoded, &normalized); err != nil {
			return value
		}
		return RedactValue(normalized)
	}
}

func sensitiveDLPKey(key string) bool {
	key = normalizeDLPKey(key)
	switch key {
	case "token", "access_token", "refresh_token", "id_token", "auth_token", "bearer_token",
		"secret", "client_secret", "app_secret", "password", "passwd", "passphrase",
		"api_key", "apikey", "access_key", "secret_key", "private_key", "credential", "credentials",
		"authorization", "proxy_authorization", "cookie", "set_cookie", "session_key", "session_id", "signature", "sig",
		"aws_secret_access_key", "aws_session_token":
		return true
	}
	for _, suffix := range []string{"_token", "_secret", "_password", "_passwd", "_api_key", "_apikey", "_access_key", "_private_key", "_credential", "_credentials", "_session_key", "_session_id", "_signature", "_sig"} {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}

func normalizeDLPKey(key string) string {
	var out strings.Builder
	runes := []rune(strings.TrimSpace(key))
	separator := true
	for index, r := range runes {
		previous := rune(0)
		if index > 0 {
			previous = runes[index-1]
		}
		next := rune(0)
		if index+1 < len(runes) {
			next = runes[index+1]
		}
		if r >= 'A' && r <= 'Z' {
			prevLowerOrDigit := (previous >= 'a' && previous <= 'z') || (previous >= '0' && previous <= '9')
			prevUpper := previous >= 'A' && previous <= 'Z'
			nextLower := next >= 'a' && next <= 'z'
			if index > 0 && (prevLowerOrDigit || (prevUpper && nextLower)) && !separator {
				out.WriteByte('_')
				separator = true
			}
			r += 'a' - 'A'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			separator = false
		} else if out.Len() > 0 && !separator {
			out.WriteByte('_')
			separator = true
		}
	}
	return strings.Trim(out.String(), "_")
}
