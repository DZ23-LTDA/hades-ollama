package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type MediaProvider struct {
	Name               string
	BaseURL            string
	APIKey             string
	ImageModel         string
	VideoModel         string
	SpeechModel        string
	TranscriptionModel string
}

type MediaManager struct {
	Provider MediaProvider
	Client   *http.Client
}

type MediaResult struct {
	Path      string           `json:"path"`
	MediaType string           `json:"media_type"`
	Artifact  ArtifactManifest `json:"artifact"`
	Text      string           `json:"text,omitempty"`
}

func (p MediaProvider) Validate() error {
	if strings.TrimSpace(p.BaseURL) == "" {
		return errors.New("media provider base URL is required")
	}
	u, err := url.Parse(p.BaseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && isLoopbackHost(u.Hostname()))) {
		return errors.New("media provider must use HTTPS or a loopback HTTP endpoint")
	}
	if u.Scheme == "https" && strings.TrimSpace(p.APIKey) == "" {
		return errors.New("media provider API key is required")
	}
	return nil
}

func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func safeMediaWorkspaceRoot(workspace string) (string, error) {
	if strings.TrimSpace(workspace) == "" {
		return "", errors.New("media workspace is required")
	}
	root, err := filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", errors.New("media workspace must be a directory")
	}
	linkInfo, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("media workspace must not be a symlink")
	}
	return root, nil
}

func openMediaWorkspaceRoot(workspace string) (*os.Root, string, error) {
	root, err := safeMediaWorkspaceRoot(workspace)
	if err != nil {
		return nil, "", err
	}
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, "", err
	}
	return rootHandle, root, nil
}

func mediaRelativePath(root, path string) (string, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("media path escapes workspace")
	}
	return filepath.ToSlash(relative), nil
}

func safeMediaInputPath(workspace, inputPath string) (string, os.FileInfo, error) {
	root, err := safeMediaWorkspaceRoot(workspace)
	if err != nil {
		return "", nil, err
	}
	input, err := filepath.Abs(inputPath)
	if err != nil {
		return "", nil, err
	}
	if !isWithin(root, input) {
		return "", nil, errors.New("media input escapes workspace")
	}
	if err := rejectSymlinkComponents(root, input); err != nil {
		return "", nil, fmt.Errorf("media input is not safe: %w", err)
	}
	info, err := os.Stat(input)
	if err != nil {
		return "", nil, err
	}
	if !info.Mode().IsRegular() {
		return "", nil, errors.New("media input must be a regular file")
	}
	return input, info, nil
}

func openSafeMediaInput(workspace, inputPath string) (*os.Root, *os.File, string, os.FileInfo, error) {
	rootHandle, root, err := openMediaWorkspaceRoot(workspace)
	if err != nil {
		return nil, nil, "", nil, err
	}
	input, _, err := safeMediaInputPath(root, inputPath)
	if err != nil {
		_ = rootHandle.Close()
		return nil, nil, "", nil, err
	}
	relative, err := mediaRelativePath(root, input)
	if err != nil {
		_ = rootHandle.Close()
		return nil, nil, "", nil, err
	}
	file, err := rootHandle.Open(relative)
	if err != nil {
		_ = rootHandle.Close()
		return nil, nil, "", nil, err
	}
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() {
		_ = file.Close()
		_ = rootHandle.Close()
		if err != nil {
			return nil, nil, "", nil, err
		}
		return nil, nil, "", nil, errors.New("media input must be a regular file")
	}
	return rootHandle, file, relative, openedInfo, nil
}

func writeMediaFile(workspace, relativePath string, data []byte, limit int64, pinnedRoots ...*os.Root) (string, error) {
	if len(pinnedRoots) > 0 {
		return writeMediaFileFromRoot(workspace, pinnedRoots[0], relativePath, data, limit)
	}
	if int64(len(data)) > limit {
		return "", errors.New("media payload exceeds limit")
	}
	root, rootPath, err := openMediaWorkspaceRoot(workspace)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := rejectMediaOutputSymlinks(root, relativePath); err != nil {
		return "", err
	}
	if err := root.MkdirAll(filepath.ToSlash(filepath.Dir(relativePath)), 0o700); err != nil {
		return "", err
	}
	file, err := root.OpenFile(filepath.ToSlash(relativePath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return filepath.Join(rootPath, filepath.FromSlash(relativePath)), nil
}

func rejectMediaOutputSymlinks(root *os.Root, relativePath string) error {
	clean := filepath.ToSlash(filepath.Clean(relativePath))
	parts := strings.Split(clean, "/")
	current := ""
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if current == "" {
			current = part
		} else {
			current += "/" + part
		}
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("media output path contains a symlink: %s", current)
		}
	}
	return nil
}

func validateMediaOutputDirectory(workspace, relativePath string, pinnedRoots ...*os.Root) error {
	if len(pinnedRoots) > 0 {
		return validateMediaOutputDirectoryFromRoot(pinnedRoots[0], relativePath)
	}
	root, _, err := openMediaWorkspaceRoot(workspace)
	if err != nil {
		return err
	}
	defer root.Close()
	info, err := root.Lstat(filepath.ToSlash(relativePath))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("media output directory contains a symlink: %s", relativePath)
	}
	if !info.IsDir() {
		return fmt.Errorf("media output path is not a directory: %s", relativePath)
	}
	return nil
}

func readMediaFile(ctx context.Context, reader io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, errors.New("media payload limit is invalid")
	}
	var data bytes.Buffer
	buffer := make([]byte, 32<<10)
	remaining := limit + 1
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		chunkSize := int64(len(buffer))
		if chunkSize > remaining {
			chunkSize = remaining
		}
		n, err := reader.Read(buffer[:chunkSize])
		if n > 0 {
			_, _ = data.Write(buffer[:n])
			remaining -= int64(n)
			if int64(data.Len()) > limit {
				return nil, errors.New("media payload exceeds limit")
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return data.Bytes(), nil
			}
			return nil, err
		}
		if n == 0 {
			return nil, errors.New("media reader returned no progress")
		}
	}
	return nil, errors.New("media payload exceeds limit")
}

func NewMediaManager(provider MediaProvider) (*MediaManager, error) {
	if err := provider.Validate(); err != nil {
		return nil, err
	}
	return &MediaManager{Provider: provider, Client: newMediaHTTPClient(3 * time.Minute)}, nil
}

type mediaLoopbackContextKey struct{}

var errMediaRedirectDisabled = errors.New("media redirects are disabled")

func newMediaHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = mediaDialContext
	return &http.Client{Timeout: timeout, Transport: transport, CheckRedirect: rejectMediaRedirect}
}

func rejectMediaRedirect(_ *http.Request, _ []*http.Request) error {
	return errMediaRedirectDisabled
}

func mediaDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return mediaDialContextWithResolver(ctx, network, address, net.DefaultResolver.LookupIPAddr)
}

func mediaDialContextWithResolver(ctx context.Context, network, address string, lookup func(context.Context, string) ([]net.IPAddr, error)) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" || port == "" {
		return nil, errors.New("media destination address is invalid")
	}
	var addresses []net.IPAddr
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		addresses = []net.IPAddr{{IP: ip}}
	} else {
		if lookup == nil {
			return nil, errors.New("media destination resolver is unavailable")
		}
		addresses, err = lookup(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("media destination lookup failed: %w", err)
		}
	}
	if len(addresses) == 0 {
		return nil, errors.New("media destination has no addresses")
	}
	allowLoopback := mediaLoopbackContext(ctx)
	for _, resolved := range addresses {
		if resolved.IP == nil || (allowLoopback && !resolved.IP.IsLoopback()) {
			return nil, errors.New("media loopback destination resolves outside loopback")
		}
		if !allowLoopback && mediaPrivateIP(resolved.IP) {
			return nil, errors.New("media destination resolves to a private address")
		}
	}
	var lastErr error
	for _, resolved := range addresses {
		if network == "tcp4" && resolved.IP.To4() == nil {
			continue
		}
		if network == "tcp6" && resolved.IP.To4() != nil {
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
		if splitErr != nil || remoteIP == nil || !remoteIP.Equal(resolved.IP) || (allowLoopback && !remoteIP.IsLoopback()) || (!allowLoopback && mediaPrivateIP(remoteIP)) {
			_ = conn.Close()
			lastErr = errors.New("media connected address was not approved")
			continue
		}
		return conn, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("media destination has no address for requested network")
}

func mediaPrivateIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	for _, cidr := range []string{"100.64.0.0/10", "198.18.0.0/15", "192.0.2.0/24", "198.51.100.0/24", "203.0.113.0/24", "2001:db8::/32"} {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func mediaLoopbackContext(ctx context.Context) bool {
	value, _ := ctx.Value(mediaLoopbackContextKey{}).(bool)
	return value
}

func rejectSensitiveMediaText(field, value string) error {
	if findings := ScanDLP(value); len(findings) > 0 {
		return fmt.Errorf("media %s contains sensitive data (%s)", field, findings[0].Kind)
	}
	return nil
}

func mediaRequestContext(ctx context.Context, rawURL string) (context.Context, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return nil, errors.New("media URL is invalid")
	}
	return context.WithValue(ctx, mediaLoopbackContextKey{}, isLoopbackHost(parsed.Hostname())), nil
}

func (m *MediaManager) GenerateImage(ctx context.Context, workspace, prompt, model string, pinnedRoots ...*os.Root) (MediaResult, error) {
	if strings.TrimSpace(prompt) == "" {
		return MediaResult{}, errors.New("image prompt is required")
	}
	if err := rejectSensitiveMediaText("prompt", prompt); err != nil {
		return MediaResult{}, err
	}
	if model == "" {
		model = m.Provider.ImageModel
	}
	if err := validateMediaOutputDirectory(workspace, ".agent-media", pinnedRoots...); err != nil {
		return MediaResult{}, err
	}
	payload, err := m.postJSON(ctx, "/images/generations", map[string]any{"model": model, "prompt": prompt, "n": 1, "response_format": "b64_json"})
	if err != nil {
		return MediaResult{}, err
	}
	entry, err := firstData(payload)
	if err != nil {
		return MediaResult{}, err
	}
	path, mediaType, err := m.materializeEntry(ctx, workspace, "generated-image", ".png", entry, pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	artifact, err := buildMediaArtifact(workspace, filepath.ToSlash(filepath.Join(".agent-media", filepath.Base(path))), pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	return MediaResult{Path: path, MediaType: mediaType, Artifact: artifact}, nil
}

func (m *MediaManager) GenerateVideo(ctx context.Context, workspace, prompt, model string, pinnedRoots ...*os.Root) (MediaResult, error) {
	if strings.TrimSpace(prompt) == "" {
		return MediaResult{}, errors.New("video prompt is required")
	}
	if err := rejectSensitiveMediaText("prompt", prompt); err != nil {
		return MediaResult{}, err
	}
	if model == "" {
		model = m.Provider.VideoModel
	}
	if err := validateMediaOutputDirectory(workspace, ".agent-media", pinnedRoots...); err != nil {
		return MediaResult{}, err
	}
	payload, err := m.postJSON(ctx, "/videos/generations", map[string]any{"model": model, "prompt": prompt})
	if err != nil {
		return MediaResult{}, err
	}
	entry, err := firstData(payload)
	if err != nil {
		return MediaResult{}, err
	}
	path, mediaType, err := m.materializeEntry(ctx, workspace, "generated-video", ".mp4", entry, pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	artifact, err := buildMediaArtifact(workspace, filepath.ToSlash(filepath.Join(".agent-media", filepath.Base(path))), pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	return MediaResult{Path: path, MediaType: mediaType, Artifact: artifact}, nil
}

func (m *MediaManager) GenerateSpeech(ctx context.Context, workspace, text, voice, model string, pinnedRoots ...*os.Root) (MediaResult, error) {
	if strings.TrimSpace(text) == "" {
		return MediaResult{}, errors.New("speech text is required")
	}
	if err := rejectSensitiveMediaText("speech text", text); err != nil {
		return MediaResult{}, err
	}
	if model == "" {
		model = m.Provider.SpeechModel
	}
	if err := validateMediaOutputDirectory(workspace, ".agent-media", pinnedRoots...); err != nil {
		return MediaResult{}, err
	}
	body, err := m.postBytes(ctx, "/audio/speech", map[string]any{"model": model, "input": text, "voice": voice, "response_format": "wav"})
	if err != nil {
		return MediaResult{}, err
	}
	relativePath := filepath.ToSlash(filepath.Join(".agent-media", "speech-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".wav"))
	path, err := writeMediaFile(workspace, relativePath, body, 100<<20, pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	artifact, err := buildMediaArtifact(workspace, filepath.ToSlash(filepath.Join(".agent-media", filepath.Base(path))), pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	return MediaResult{Path: path, MediaType: "audio/wav", Artifact: artifact}, nil
}

func (m *MediaManager) Transcribe(ctx context.Context, workspace, inputPath, model string, pinnedRoots ...*os.Root) (MediaResult, error) {
	if strings.TrimSpace(inputPath) == "" {
		return MediaResult{}, errors.New("audio input is required")
	}
	relativeInput, audio, err := readMediaInputBounded(ctx, workspace, inputPath, 100<<20, pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	return m.transcribeBytes(ctx, workspace, relativeInput, audio, model, pinnedRoots...)
}

func (m *MediaManager) transcribeBytes(ctx context.Context, workspace, relativeInput string, audio []byte, model string, pinnedRoots ...*os.Root) (MediaResult, error) {
	if int64(len(audio)) > 100<<20 {
		return MediaResult{}, errors.New("audio input exceeds 100 MiB")
	}
	if findings := ScanDLP(string(audio)); len(findings) > 0 {
		return MediaResult{}, fmt.Errorf("audio input contains sensitive data (%s)", findings[0].Kind)
	}
	transcriptRelative := filepath.ToSlash(filepath.Join(".agent-media", "transcript-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".txt"))
	if err := validateMediaOutputDirectory(workspace, ".agent-media", pinnedRoots...); err != nil {
		return MediaResult{}, err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(relativeInput))
	if err != nil {
		return MediaResult{}, err
	}
	if _, err := part.Write(audio); err != nil {
		return MediaResult{}, err
	}
	if model == "" {
		model = m.Provider.TranscriptionModel
	}
	_ = writer.WriteField("model", model)
	if err := writer.Close(); err != nil {
		return MediaResult{}, err
	}
	request, err := m.newRequest(ctx, http.MethodPost, m.endpoint("/audio/transcriptions"), &body)
	if err != nil {
		return MediaResult{}, err
	}
	m.headers(request)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := m.client().Do(request)
	if err != nil {
		return MediaResult{}, err
	}
	defer response.Body.Close()
	payload, err := decodeResponse(response)
	if err != nil {
		return MediaResult{}, err
	}
	text, _ := payload["text"].(string)
	text = sanitizeMediaOutputText(text)
	transcriptPath, err := writeMediaFile(workspace, transcriptRelative, []byte(text+"\n"), 4<<20, pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	artifact, err := buildMediaArtifact(workspace, filepath.ToSlash(filepath.Join(".agent-media", filepath.Base(transcriptPath))), pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	return MediaResult{Path: transcriptPath, MediaType: "text/plain", Text: text, Artifact: artifact}, nil
}

// AnalyzeImage sends an image to a vision-capable chat model. It supports both
// hosted OpenAI-compatible providers and a loopback Ollama /v1 endpoint.
func (m *MediaManager) AnalyzeImage(ctx context.Context, workspace, inputPath, prompt, model string, pinnedRoots ...*os.Root) (MediaResult, error) {
	if strings.TrimSpace(inputPath) == "" || strings.TrimSpace(prompt) == "" {
		return MediaResult{}, errors.New("image input and vision prompt are required")
	}
	_, data, err := readMediaInputBounded(ctx, workspace, inputPath, 25<<20, pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	return m.analyzeImageBytes(ctx, workspace, data, prompt, model, pinnedRoots...)
}

func (m *MediaManager) analyzeImageBytes(ctx context.Context, workspace string, data []byte, prompt, model string, pinnedRoots ...*os.Root) (MediaResult, error) {
	if strings.TrimSpace(prompt) == "" {
		return MediaResult{}, errors.New("vision prompt is required")
	}
	if int64(len(data)) > 25<<20 {
		return MediaResult{}, errors.New("image input exceeds 25 MiB")
	}
	if err := rejectSensitiveMediaText("prompt", prompt); err != nil {
		return MediaResult{}, err
	}
	if findings := ScanDLP(string(data)); len(findings) > 0 {
		return MediaResult{}, fmt.Errorf("image input contains sensitive data (%s)", findings[0].Kind)
	}
	visionRelative := filepath.ToSlash(filepath.Join(".agent-media", "vision-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".txt"))
	if err := validateMediaOutputDirectory(workspace, ".agent-media", pinnedRoots...); err != nil {
		return MediaResult{}, err
	}
	if model == "" {
		model = m.Provider.ImageModel
	}
	mimeType := http.DetectContentType(data)
	payload, err := m.postJSON(ctx, "/chat/completions", map[string]any{
		"model": model,
		"messages": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "text", "text": prompt},
			map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)}},
		}}},
	})
	if err != nil {
		return MediaResult{}, err
	}
	text := ""
	if choices, ok := payload["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			if message, ok := choice["message"].(map[string]any); ok {
				text, _ = message["content"].(string)
			}
		}
	}
	if strings.TrimSpace(text) == "" {
		return MediaResult{}, errors.New("vision provider returned no text")
	}
	text = sanitizeMediaOutputText(text)
	output, err := writeMediaFile(workspace, visionRelative, []byte(text+"\n"), 4<<20, pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	artifact, err := buildMediaArtifact(workspace, filepath.ToSlash(filepath.Join(".agent-media", filepath.Base(output))), pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	return MediaResult{Path: output, MediaType: "text/plain", Text: text, Artifact: artifact}, nil
}

func sanitizeMediaOutputText(value string) string {
	return RedactDLP(value)
}

func GenerateTone(workspace string, frequency float64, duration time.Duration, pinnedRoots ...*os.Root) (MediaResult, error) {
	if frequency <= 0 || frequency > 20000 {
		frequency = 440
	}
	if duration <= 0 || duration > 30*time.Second {
		duration = time.Second
	}
	const sampleRate = 8000
	samples := int(duration.Seconds() * sampleRate)
	data := make([]byte, samples*2)
	for i := range samples {
		sample := int16(12000 * sin(2*3.141592653589793*frequency*float64(i)/sampleRate))
		data[i*2] = byte(sample)
		data[i*2+1] = byte(sample >> 8)
	}
	var wav bytes.Buffer
	if err := writeWAVHeader(&wav, len(data), sampleRate); err != nil {
		return MediaResult{}, err
	}
	_, _ = wav.Write(data)
	path, err := writeMediaFile(workspace, filepath.ToSlash(filepath.Join(".agent-media", "tone-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".wav")), wav.Bytes(), 100<<20, pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	artifact, err := buildMediaArtifact(workspace, filepath.ToSlash(filepath.Join(".agent-media", filepath.Base(path))), pinnedRoots...)
	if err != nil {
		return MediaResult{}, err
	}
	return MediaResult{Path: path, MediaType: "audio/wav", Artifact: artifact}, nil
}

func (m *MediaManager) postJSON(ctx context.Context, path string, value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	request, err := m.newRequest(ctx, http.MethodPost, m.endpoint(path), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	m.headers(request)
	request.Header.Set("Content-Type", "application/json")
	response, err := m.client().Do(request)
	if err != nil {
		return nil, errors.New("media provider request failed")
	}
	defer response.Body.Close()
	return decodeResponse(response)
}

func (m *MediaManager) postBytes(ctx context.Context, path string, value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	request, err := m.newRequest(ctx, http.MethodPost, m.endpoint(path), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	m.headers(request)
	request.Header.Set("Content-Type", "application/json")
	response, err := m.client().Do(request)
	if err != nil {
		return nil, errors.New("media provider request failed")
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("media provider returned HTTP status %d", response.StatusCode)
	}
	data, err = readLimitedMediaBody(response.Body, 100<<20)
	if err != nil {
		return nil, err
	}
	if err := validateMediaContentType(".wav", response.Header.Get("Content-Type")); err != nil {
		return nil, err
	}
	if err := validateMediaMagic(".wav", data); err != nil {
		return nil, err
	}
	return data, nil
}

func (m *MediaManager) materializeEntry(ctx context.Context, workspace, prefix, extension string, entry map[string]any, pinnedRoots ...*os.Root) (string, string, error) {
	var err error
	if len(pinnedRoots) == 0 {
		workspace, err = safeMediaWorkspaceRoot(workspace)
		if err != nil {
			return "", "", err
		}
	} else if pinnedRoots[0] == nil {
		return "", "", errors.New("pinned media workspace is required")
	}
	var data []byte
	mediaType := "application/octet-stream"
	if encoded, ok := entry["b64_json"].(string); ok && encoded != "" {
		var err error
		data, err = base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return "", "", err
		}
		if declared, ok := entry["mime_type"].(string); ok && strings.TrimSpace(declared) != "" {
			mediaType = strings.TrimSpace(strings.Split(declared, ";")[0])
		}
	}
	if rawURL, ok := entry["url"].(string); ok && rawURL != "" {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()))) {
			return "", "", errors.New("media URL must use HTTPS or loopback HTTP")
		}
		request, err := m.newRequest(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return "", "", err
		}
		response, err := m.client().Do(request)
		if err != nil {
			if errors.Is(err, errMediaRedirectDisabled) {
				return "", "", errMediaRedirectDisabled
			}
			return "", "", errors.New("media download failed")
		}
		defer response.Body.Close()
		if response.StatusCode/100 != 2 {
			return "", "", fmt.Errorf("media download failed with status %d", response.StatusCode)
		}
		data, err = readLimitedMediaBody(response.Body, 100<<20)
		if err != nil {
			return "", "", err
		}
		mediaType = strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	}
	if len(data) == 0 {
		return "", "", errors.New("media provider returned no data")
	}
	if err := validateMediaContentType(extension, mediaType); err != nil {
		return "", "", err
	}
	if err := validateMediaMagic(extension, data); err != nil {
		return "", "", err
	}
	if mediaType == "" || mediaType == "application/octet-stream" {
		mediaType = mediaTypeForExtension(extension)
	}
	relativePath := filepath.ToSlash(filepath.Join(".agent-media", prefix+"-"+strconv.FormatInt(time.Now().UnixNano(), 10)+extension))
	path, err := writeMediaFile(workspace, relativePath, data, 100<<20, pinnedRoots...)
	if err != nil {
		return "", "", err
	}
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	return path, mediaType, nil
}

func (m *MediaManager) endpoint(path string) string {
	return strings.TrimRight(m.Provider.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (m *MediaManager) newRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	requestContext, err := mediaRequestContext(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(requestContext, method, rawURL, body)
}

func (m *MediaManager) client() *http.Client {
	base := m.Client
	if base == nil {
		return newMediaHTTPClient(3 * time.Minute)
	}
	client := *base
	client.CheckRedirect = rejectMediaRedirect
	switch transport := base.Transport.(type) {
	case nil:
		client.Transport = newMediaHTTPClient(client.Timeout).Transport
	case *http.Transport:
		safeTransport := transport.Clone()
		safeTransport.Proxy = nil
		safeTransport.DialContext = mediaDialContext
		client.Transport = safeTransport
	default:
		client.Transport = newMediaHTTPClient(client.Timeout).Transport
	}
	return &client
}

func (m *MediaManager) headers(request *http.Request) {
	if strings.TrimSpace(m.Provider.APIKey) != "" {
		request.Header.Set("Authorization", "Bearer "+m.Provider.APIKey)
	}
	request.Header.Set("User-Agent", "ollama-dz23-agentic-media/1")
}

func firstData(payload map[string]any) (map[string]any, error) {
	data, ok := payload["data"].([]any)
	if !ok || len(data) == 0 {
		return nil, errors.New("media response has no data")
	}
	entry, ok := data[0].(map[string]any)
	if !ok {
		return nil, errors.New("media response data is invalid")
	}
	return entry, nil
}

func decodeResponse(response *http.Response) (map[string]any, error) {
	body, err := readLimitedMediaBody(response.Body, 8<<20)
	if err != nil {
		return nil, errors.New("media provider response could not be read")
	}
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("media provider returned HTTP status %d", response.StatusCode)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, errors.New("media provider returned an invalid response")
	}
	return payload, nil
}

func readLimitedMediaBody(reader io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, errors.New("media payload limit is invalid")
	}
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("media payload exceeds limit")
	}
	return data, nil
}

func mediaTypeForExtension(extension string) string {
	switch strings.ToLower(filepath.Ext(extension)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".wav":
		return "audio/wav"
	default:
		return "application/octet-stream"
	}
}

func validateMediaContentType(extension, declared string) error {
	declared = strings.TrimSpace(strings.Split(declared, ";")[0])
	if declared == "" || declared == "application/octet-stream" {
		return nil
	}
	expected := mediaTypeForExtension(extension)
	if expected != "application/octet-stream" && declared != expected {
		return fmt.Errorf("media content type %q does not match %s", declared, expected)
	}
	return nil
}

func validateMediaMagic(extension string, data []byte) error {
	if len(data) == 0 {
		return errors.New("media provider returned no data")
	}
	switch strings.ToLower(filepath.Ext(extension)) {
	case ".png":
		if !bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
			return errors.New("media payload is not a PNG")
		}
	case ".jpg", ".jpeg":
		if len(data) < 3 || data[0] != 0xff || data[1] != 0xd8 || data[2] != 0xff {
			return errors.New("media payload is not a JPEG")
		}
	case ".webp":
		if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
			return errors.New("media payload is not a WebP")
		}
	case ".mp4":
		if len(data) < 12 || string(data[4:8]) != "ftyp" {
			return errors.New("media payload is not an MP4")
		}
	case ".wav":
		if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
			return errors.New("media payload is not a WAV")
		}
	}
	return nil
}

// OpenMediaWorkspaceRoot authorizes a media workspace once and returns the
// descriptor used by root-aware media operations.
func OpenMediaWorkspaceRoot(workspace string) (*os.Root, string, error) {
	return openMediaWorkspaceRoot(workspace)
}

func mediaRelativePathFromRoot(workspace string, root *os.Root, inputPath string) (string, error) {
	if root == nil || strings.TrimSpace(workspace) == "" || strings.TrimSpace(inputPath) == "" {
		return "", errors.New("pinned media workspace is required")
	}
	if filepath.IsAbs(inputPath) {
		workspaceAbs, err := filepath.Abs(workspace)
		if err != nil {
			return "", err
		}
		inputAbs, err := filepath.Abs(inputPath)
		if err != nil {
			return "", err
		}
		inputPath, err = filepath.Rel(workspaceAbs, inputAbs)
		if err != nil {
			return "", err
		}
	}
	return safeAnchoredWorkspaceRelative(root, inputPath, false)
}

func readMediaInputBoundedFromRoot(ctx context.Context, workspace string, root *os.Root, inputPath string, limit int64) (string, []byte, error) {
	relative, err := mediaRelativePathFromRoot(workspace, root, inputPath)
	if err != nil {
		return "", nil, err
	}
	file, err := root.Open(filepath.FromSlash(relative))
	if err != nil {
		return "", nil, err
	}
	defer file.Close()
	initial, err := file.Stat()
	if err != nil || !initial.Mode().IsRegular() {
		if err != nil {
			return "", nil, err
		}
		return "", nil, errors.New("media input must be a regular file")
	}
	if initial.Size() < 0 || initial.Size() > limit {
		if limit == 100<<20 {
			return "", nil, errors.New("audio input exceeds 100 MiB")
		}
		if limit == 25<<20 {
			return "", nil, errors.New("image input exceeds 25 MiB")
		}
		return "", nil, fmt.Errorf("media input exceeds %d byte limit", limit)
	}
	data, err := readMediaFile(ctx, file, limit)
	if err != nil {
		return "", nil, err
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(initial, after) || initial.Size() != after.Size() || !initial.ModTime().Equal(after.ModTime()) || int64(len(data)) != initial.Size() {
		return "", nil, ErrMediaInputChanged
	}
	return relative, data, nil
}

func writeMediaFileFromRoot(workspace string, root *os.Root, relativePath string, data []byte, limit int64) (string, error) {
	if root == nil || strings.TrimSpace(workspace) == "" {
		return "", errors.New("pinned media workspace is required")
	}
	if int64(len(data)) > limit {
		return "", errors.New("media payload exceeds limit")
	}
	relativePath = filepath.ToSlash(filepath.Clean(filepath.FromSlash(relativePath)))
	if !filepath.IsLocal(filepath.FromSlash(relativePath)) || relativePath == "." || strings.HasPrefix(relativePath, "../") {
		return "", errors.New("media output path escapes workspace")
	}
	if err := rejectMediaOutputSymlinks(root, relativePath); err != nil {
		return "", err
	}
	if err := root.MkdirAll(filepath.ToSlash(filepath.Dir(relativePath)), 0o700); err != nil {
		return "", err
	}
	file, err := root.OpenFile(relativePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	// A descriptor-relative path remains meaningful even if workspace's pathname
	// is renamed or replaced while this operation is in flight.
	return filepath.ToSlash(relativePath), nil
}

func validateMediaOutputDirectoryFromRoot(root *os.Root, relativePath string) error {
	if root == nil {
		return errors.New("pinned media workspace is required")
	}
	if err := rejectMediaOutputSymlinks(root, relativePath); err != nil {
		return err
	}
	info, err := root.Lstat(filepath.ToSlash(filepath.Clean(filepath.FromSlash(relativePath))))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("media output path is not a directory: %s", relativePath)
	}
	return nil
}

func buildMediaArtifact(workspace, relativePath string, pinnedRoots ...*os.Root) (ArtifactManifest, error) {
	if len(pinnedRoots) > 0 {
		return buildArtifactManifestFromRoot(pinnedRoots[0], relativePath, "", "", filepath.Base(filepath.FromSlash(relativePath)))
	}
	return BuildArtifactManifest(workspace, "", "", filepath.Base(filepath.FromSlash(relativePath)), relativePath)
}

func buildMediaArtifactFromRoot(root *os.Root, relativePath string) (ArtifactManifest, error) {
	return buildMediaArtifact("", relativePath, root)
}

func sin(value float64) float64 {
	x := value
	for x > 3.141592653589793 {
		x -= 6.283185307179586
	}
	for x < -3.141592653589793 {
		x += 6.283185307179586
	}
	term, sum := x, x
	for n := 1; n < 8; n++ {
		term *= -x * x / float64((2*n)*(2*n+1))
		sum += term
	}
	return sum
}

func writeWAVHeader(w io.Writer, dataSize, sampleRate int) error {
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	putLE32(header[4:8], uint32(36+dataSize))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	putLE32(header[16:20], 16)
	putLE16(header[20:22], 1)
	putLE16(header[22:24], 1)
	putLE32(header[24:28], uint32(sampleRate))
	putLE32(header[28:32], uint32(sampleRate*2))
	putLE16(header[32:34], 2)
	putLE16(header[34:36], 16)
	copy(header[36:40], "data")
	putLE32(header[40:44], uint32(dataSize))
	_, err := w.Write(header)
	return err
}
func putLE16(b []byte, v uint16) { b[0] = byte(v); b[1] = byte(v >> 8) }
func putLE32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}
