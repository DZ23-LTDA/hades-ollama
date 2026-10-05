package grok

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	StatusCataloged     Status = "cataloged"
	StatusAuthenticated Status = "authenticated"
	StatusHealthy       Status = "healthy"
	StatusUnavailable   Status = "unavailable"
	StatusCircuitOpen   Status = "circuit_open"
)

type Tool struct {
	Type     string         `json:"type"`
	Name     string         `json:"name,omitempty"`
	Function map[string]any `json:"function,omitempty"`
}

type ResponseRequest struct {
	Model          string         `json:"model,omitempty"`
	Input          any            `json:"input"`
	Tools          []Tool         `json:"tools,omitempty"`
	Stream         bool           `json:"stream,omitempty"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
	Temperature    *float64       `json:"temperature,omitempty"`
}

type Citation struct {
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	Excerpt   string `json:"excerpt,omitempty"`
	Publisher string `json:"publisher,omitempty"`
}

type Usage struct {
	InputTokens  int64 `json:"input_tokens,omitempty"`
	OutputTokens int64 `json:"output_tokens,omitempty"`
	TotalTokens  int64 `json:"total_tokens,omitempty"`
}

type Response struct {
	ID         string           `json:"id,omitempty"`
	Model      string           `json:"model,omitempty"`
	OutputText string           `json:"output_text,omitempty"`
	Output     []map[string]any `json:"output,omitempty"`
	Citations  []Citation       `json:"citations,omitempty"`
	Usage      Usage            `json:"usage,omitempty"`
	Raw        json.RawMessage  `json:"-"`
}

type ProviderStatus struct {
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	State         Status    `json:"state"`
	Authenticated bool      `json:"authenticated"`
	Healthy       bool      `json:"healthy"`
	LastLatencyMS int64     `json:"last_latency_ms,omitempty"`
	LastError     string    `json:"last_error,omitempty"`
	CheckedAt     time.Time `json:"checked_at"`
}

type Client struct {
	BaseURL               string
	APIKey                string
	Model                 string
	AllowedModels         []string
	HTTPClient            *http.Client
	AllowInsecureLoopback bool
	MaxRetries            int
	Backoff               time.Duration
	mu                    sync.Mutex
	failures              int
	openedUntil           time.Time
	lastStatus            ProviderStatus
}

var (
	ErrInvalidBaseURL       = errors.New("grok base URL must use HTTPS; insecure HTTP is allowed only for loopback tests")
	ErrCircuitOpen          = errors.New("grok provider circuit is open")
	ErrModelNotAllowed      = errors.New("grok model is not allowlisted")
	ErrStreamingUnsupported = errors.New("grok HTTP route does not expose streaming")
)

// newHardenedHTTPClient returns an HTTP client that never routes provider
// traffic through an environment proxy (which would leak the API key carried in
// the Authorization header to HTTP_PROXY/HTTPS_PROXY) and enforces TLS 1.2+.
func newHardenedHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		transport.TLSClientConfig.MinVersion = tls.VersionTLS12
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

func NewClient(baseURL, apiKey, model string) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopback(parsed.Hostname()))) {
		return nil, ErrInvalidBaseURL
	}
	model = strings.TrimSpace(model)
	client := &Client{BaseURL: baseURL, APIKey: strings.TrimSpace(apiKey), Model: model, HTTPClient: newHardenedHTTPClient(60 * time.Second), MaxRetries: 2, Backoff: 150 * time.Millisecond}
	if model != "" {
		client.AllowedModels = []string{model}
	}
	return client, nil
}

func (c *Client) SetAllowedModels(models ...string) error {
	allowed := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		allowed = append(allowed, model)
	}
	if len(allowed) == 0 {
		return ErrModelNotAllowed
	}
	c.AllowedModels = allowed
	return nil
}

func (c *Client) validateModel(model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		model = strings.TrimSpace(c.Model)
	}
	if model == "" {
		return errors.New("grok model is required")
	}
	allowed := c.AllowedModels
	if len(allowed) == 0 {
		allowed = []string{c.Model}
	}
	for _, candidate := range allowed {
		if model == strings.TrimSpace(candidate) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrModelNotAllowed, model)
}

func (c *Client) endpoint(path string) string {
	base := strings.TrimRight(c.BaseURL, "/")
	path = "/" + strings.TrimLeft(path, "/")
	if strings.HasSuffix(base, "/v1") && strings.HasPrefix(path, "/v1/") {
		path = strings.TrimPrefix(path, "/v1")
	}
	return base + path
}

func (c *Client) Responses(ctx context.Context, request ResponseRequest) (Response, error) {
	if request.Stream {
		return Response{}, ErrStreamingUnsupported
	}
	if strings.TrimSpace(request.Model) == "" {
		request.Model = c.Model
	}
	if err := c.validateModel(request.Model); err != nil {
		return Response{}, err
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return Response{}, err
	}
	var response Response
	err = c.withRetry(ctx, func() error {
		started := time.Now()
		body, statusCode, callErr := c.do(ctx, http.MethodPost, c.endpoint("/v1/responses"), payload, "application/json")
		c.record(statusCode, time.Since(started), callErr)
		if callErr != nil {
			return callErr
		}
		if statusCode/100 != 2 {
			return fmt.Errorf("grok responses returned HTTP %d: %s", statusCode, truncate(string(body), 500))
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("decode grok response: %w", err)
		}
		response.Raw = append(json.RawMessage(nil), body...)
		return nil
	})
	return response, err
}

func (c *Client) StreamResponses(ctx context.Context, request ResponseRequest, onDelta func(string) error) error {
	if onDelta == nil {
		return errors.New("stream callback is required")
	}
	if strings.TrimSpace(request.Model) == "" {
		request.Model = c.Model
	}
	if err := c.validateModel(request.Model); err != nil {
		return err
	}
	request.Stream = true
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if err := c.checkCircuit(); err != nil {
		return err
	}
	started := time.Now()
	requestHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/v1/responses"), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	c.authorize(requestHTTP)
	requestHTTP.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient().Do(requestHTTP)
	if err != nil {
		c.record(0, time.Since(started), err)
		return err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		err := fmt.Errorf("grok stream returned HTTP %d: %s", response.StatusCode, truncate(string(body), 500))
		c.record(response.StatusCode, time.Since(started), err)
		return err
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 4096), 2<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			c.record(response.StatusCode, time.Since(started), err)
			return fmt.Errorf("decode grok stream event: %w", err)
		}
		if delta := stringField(event, "delta"); delta != "" {
			if err := onDelta(delta); err != nil {
				return err
			}
			continue
		}
		if text := stringField(event, "text"); text != "" {
			if err := onDelta(text); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		c.record(response.StatusCode, time.Since(started), err)
		return err
	}
	c.record(response.StatusCode, time.Since(started), nil)
	return nil
}

func (c *Client) Probe(ctx context.Context) ProviderStatus {
	status := ProviderStatus{Provider: "xai/grok", Model: c.Model, State: StatusCataloged, Authenticated: c.APIKey != "", CheckedAt: time.Now().UTC()}
	if c.APIKey == "" {
		status.LastError = "credential not configured"
		return status
	}
	status.State = StatusAuthenticated
	if err := c.checkCircuit(); err != nil {
		status.State = StatusCircuitOpen
		status.LastError = err.Error()
		return status
	}
	started := time.Now()
	body, code, err := c.do(ctx, http.MethodGet, c.endpoint("/v1/models"), nil, "")
	status.LastLatencyMS = time.Since(started).Milliseconds()
	if err != nil {
		status.State, status.LastError = StatusUnavailable, err.Error()
		return status
	}
	if code/100 != 2 {
		status.State, status.LastError = StatusUnavailable, fmt.Sprintf("HTTP %d: %s", code, truncate(string(body), 300))
		return status
	}
	if err := c.validateCatalogModel(body); err != nil {
		status.State, status.LastError = StatusUnavailable, err.Error()
		return status
	}
	status.Healthy = true
	status.State = StatusHealthy
	return status
}

func (c *Client) validateCatalogModel(body []byte) error {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("decode Grok model catalog: %w", err)
	}
	allowed := c.AllowedModels
	if len(allowed) == 0 {
		allowed = []string{c.Model}
	}
	available := make(map[string]struct{}, len(payload.Data))
	for _, item := range payload.Data {
		available[strings.TrimSpace(item.ID)] = struct{}{}
	}
	for _, model := range allowed {
		if _, ok := available[strings.TrimSpace(model)]; ok {
			return nil
		}
	}
	return fmt.Errorf("configured Grok model is not present in provider catalog")
}

func (c *Client) LastStatus() ProviderStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastStatus
}

func (c *Client) withRetry(ctx context.Context, operation func() error) error {
	attempts := c.MaxRetries + 1
	if attempts < 1 || attempts > 5 {
		attempts = 3
	}
	var last error
	for attempt := range attempts {
		if err := c.checkCircuit(); err != nil {
			return err
		}
		if attempt > 0 {
			delay := c.Backoff
			if delay <= 0 {
				delay = 100 * time.Millisecond
			}
			delay *= time.Duration(1 << (attempt - 1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		last = operation()
		if last == nil || !retryable(last) {
			return last
		}
	}
	return last
}

func (c *Client) do(ctx context.Context, method, endpoint string, payload []byte, contentType string) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, 0, err
	}
	c.authorize(request)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := c.httpClient().Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	return data, response.StatusCode, readErr
}

func (c *Client) authorize(request *http.Request) {
	if c.APIKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
}

func (c *Client) checkCircuit() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Now().Before(c.openedUntil) {
		return ErrCircuitOpen
	}
	return nil
}

func (c *Client) record(code int, latency time.Duration, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Count transport errors, 5xx, and the transient rate-limit/timeout codes
	// (429/408) as failures so repeated rate limiting opens the circuit and
	// backs off instead of hammering the provider; other 4xx are caller errors
	// that should not trip the breaker.
	if err != nil || code >= 500 || code == 0 || code == http.StatusTooManyRequests || code == http.StatusRequestTimeout {
		c.failures++
		if c.failures >= 3 {
			c.openedUntil = time.Now().Add(10 * time.Second)
		}
	} else if code/100 == 2 {
		c.failures = 0
		c.openedUntil = time.Time{}
	}
	status := ProviderStatus{Provider: "xai/grok", Model: c.Model, Authenticated: c.APIKey != "", Healthy: err == nil && code/100 == 2, LastLatencyMS: latency.Milliseconds(), CheckedAt: time.Now().UTC()}
	if status.Healthy {
		status.State = StatusHealthy
	} else if c.openedUntil.After(time.Now()) {
		status.State = StatusCircuitOpen
	} else if status.Authenticated {
		status.State = StatusUnavailable
	} else {
		status.State = StatusCataloged
	}
	if err != nil {
		status.LastError = err.Error()
	}
	c.lastStatus = status
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}}
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func retryable(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrCircuitOpen) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "http 5") || strings.Contains(message, "timeout") || strings.Contains(message, "connection") || strings.Contains(message, "temporary")
}

func stringField(event map[string]any, key string) string {
	value, _ := event[key].(string)
	return value
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}
