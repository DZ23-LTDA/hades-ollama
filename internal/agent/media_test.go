package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMediaManagerGeneratesArtifactsThroughHTTPSProvider(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/images/generations":
			png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'p', 'n', 'g'}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(png)}}})
		case "/audio/transcriptions":
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "transcrição token=ghp_abcdefghijklmnopqrstuvwxyz123456"})
		case "/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "OCR token=ghp_abcdefghijklmnopqrstuvwxyz123456"}}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	manager, err := NewMediaManager(MediaProvider{Name: "test", BaseURL: server.URL, APIKey: "test-key", ImageModel: "image-test", TranscriptionModel: "stt-test"})
	if err != nil {
		t.Fatal(err)
	}
	manager.Client = server.Client()
	workspace := t.TempDir()
	image, err := manager.GenerateImage(context.Background(), workspace, "uma imagem de teste", "")
	if err != nil {
		t.Fatal(err)
	}
	if image.Artifact.SHA256 == "" || image.Artifact.Path == "" {
		t.Fatalf("invalid image artifact: %+v", image.Artifact)
	}
	input := filepath.Join(workspace, "input.wav")
	if err := os.WriteFile(input, []byte("wav"), 0o600); err != nil {
		t.Fatal(err)
	}
	transcript, err := manager.Transcribe(context.Background(), workspace, input, "")
	if err != nil || strings.Contains(transcript.Text, "ghp_abcdefghijklmnopqrstuvwxyz123456") || !strings.Contains(transcript.Text, "[REDACTED]") {
		t.Fatalf("transcript=%+v err=%v", transcript, err)
	}
	transcriptBytes, err := os.ReadFile(transcript.Path)
	if err != nil || strings.Contains(string(transcriptBytes), "ghp_abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatalf("transcript artifact retained provider credential: %q err=%v", transcriptBytes, err)
	}
	inputImage := filepath.Join(workspace, "input.png")
	if err := os.WriteFile(inputImage, []byte("fake-png"), 0o600); err != nil {
		t.Fatal(err)
	}
	vision, err := manager.AnalyzeImage(context.Background(), workspace, inputImage, "Extraia o texto da imagem", "vision-test")
	if err != nil || strings.Contains(vision.Text, "ghp_abcdefghijklmnopqrstuvwxyz123456") || !strings.Contains(vision.Text, "[REDACTED]") {
		t.Fatalf("vision=%+v err=%v", vision, err)
	}
	visionBytes, err := os.ReadFile(vision.Path)
	if err != nil || strings.Contains(string(visionBytes), "ghp_abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatalf("vision artifact retained provider credential: %q err=%v", visionBytes, err)
	}
}

func TestGenerateToneProducesWAVArtifact(t *testing.T) {
	result, err := GenerateTone(t.TempDir(), 440, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 44 || string(data[:4]) != "RIFF" || result.MediaType != "audio/wav" {
		t.Fatalf("invalid wav: len=%d type=%s", len(data), result.MediaType)
	}
}

func TestMediaMaterializeRejectsRedirectAndInvalidMagic(t *testing.T) {
	if err := validateMediaContentType(".png", "text/html"); err == nil {
		t.Fatal("expected MIME mismatch rejection")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("not-a-png"))
	}))
	defer server.Close()
	manager, err := NewMediaManager(MediaProvider{Name: "local", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.materializeEntry(context.Background(), t.TempDir(), "image", ".png", map[string]any{"url": server.URL + "/redirect"}); err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("expected redirect rejection, got %v", err)
	}
	if _, _, err := manager.materializeEntry(context.Background(), t.TempDir(), "image", ".png", map[string]any{"b64_json": base64.StdEncoding.EncodeToString([]byte("not-a-png"))}); err == nil || !strings.Contains(err.Error(), "PNG") {
		t.Fatalf("expected magic rejection, got %v", err)
	}
}

func TestMediaDownloadLimitsAndRejectsPrivateActualAddress(t *testing.T) {
	if _, err := readLimitedMediaBody(strings.NewReader("1234"), 3); err == nil {
		t.Fatal("expected media body limit rejection")
	}
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
		close(accepted)
	}()
	_, err = mediaDialContext(context.Background(), "tcp", listener.Addr().String())
	if err == nil || !strings.Contains(err.Error(), "private") {
		t.Fatalf("expected private media dial rejection, got %v", err)
	}
	select {
	case <-accepted:
		t.Fatal("private media destination received TCP before refusal")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestMediaDialRejectsPrivateResolvedAddressBeforeTCP(t *testing.T) {
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
		close(accepted)
	}()
	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}, {IP: net.ParseIP("203.0.113.8")}}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = mediaDialContextWithResolver(ctx, "tcp", "media.example:"+portOf(listener.Addr().String()), lookup)
	if err == nil || !strings.Contains(err.Error(), "private") {
		t.Fatalf("expected pre-resolution private rejection, got %v", err)
	}
	select {
	case <-accepted:
		t.Fatal("resolved private media destination received TCP")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestMediaTranscriptionRestrictsWorkspaceAndSize(t *testing.T) {
	manager, err := NewMediaManager(MediaProvider{Name: "local", BaseURL: "http://127.0.0.1:43123"})
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	outside := t.TempDir()
	outsideInput := filepath.Join(outside, "outside.wav")
	if err := os.WriteFile(outsideInput, []byte("wav"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Transcribe(context.Background(), workspace, outsideInput, ""); err == nil || !strings.Contains(err.Error(), "escapes workspace") {
		t.Fatalf("expected workspace containment rejection, got %v", err)
	}
	linkedInput := filepath.Join(workspace, "linked.wav")
	if err := os.Symlink(outsideInput, linkedInput); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := manager.Transcribe(context.Background(), workspace, linkedInput, ""); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
	largeInput := filepath.Join(workspace, "large.wav")
	file, err := os.OpenFile(largeInput, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(100<<20 + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Transcribe(context.Background(), workspace, largeInput, ""); err == nil || !strings.Contains(err.Error(), "exceeds 100 MiB") {
		t.Fatalf("expected audio size rejection, got %v", err)
	}
}

func TestMediaOutputsRejectInternalSymlinkBeforeProvider(t *testing.T) {
	providerRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerRequests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"iVBORw0KGgo="}],"choices":[{"message":{"content":"unused"}}]}`))
	}))
	defer server.Close()
	manager, err := NewMediaManager(MediaProvider{Name: "fixture", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	manager.Client = server.Client()
	workspace, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(workspace, ".agent-media")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	inputAudio := filepath.Join(workspace, "input.wav")
	if err := os.WriteFile(inputAudio, []byte("wav"), 0o600); err != nil {
		t.Fatal(err)
	}
	inputImage := filepath.Join(workspace, "input.png")
	if err := os.WriteFile(inputImage, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name string
		call func() error
	}{
		{name: "image", call: func() error {
			_, err := manager.GenerateImage(context.Background(), workspace, "fixture", "")
			return err
		}},
		{name: "video", call: func() error {
			_, err := manager.GenerateVideo(context.Background(), workspace, "fixture", "")
			return err
		}},
		{name: "speech", call: func() error {
			_, err := manager.GenerateSpeech(context.Background(), workspace, "fixture", "alloy", "")
			return err
		}},
		{name: "transcription", call: func() error {
			_, err := manager.Transcribe(context.Background(), workspace, inputAudio, "")
			return err
		}},
		{name: "vision", call: func() error {
			_, err := manager.AnalyzeImage(context.Background(), workspace, inputImage, "describe", "")
			return err
		}},
		{name: "tone", call: func() error { _, err := GenerateTone(workspace, 440, time.Millisecond); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			err := check.call()
			if err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("expected symlink rejection, got %v", err)
			}
		})
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("media output escaped workspace: %d files", len(entries))
	}
	if providerRequests != 0 {
		t.Fatalf("provider was contacted before unsafe output destination was rejected: %d requests", providerRequests)
	}
}

func TestWriteMediaFileAllowsSafeOutputAndRejectsPrivateComponents(t *testing.T) {
	workspace := t.TempDir()
	path, err := writeMediaFile(workspace, filepath.ToSlash(filepath.Join(".agent-media", "safe.txt")), []byte("fixture"), 64)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "fixture" {
		t.Fatalf("safe output=%q err=%v", data, err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(workspace, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := writeMediaFile(workspace, "linked/private.txt", []byte("blocked"), 64); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "private.txt")); !os.IsNotExist(err) {
		t.Fatalf("private output was created outside workspace: %v", err)
	}
}

func TestAnalyzeImageRejectsLargeFileBeforeReadOrProvider(t *testing.T) {
	providerRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerRequests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"unused"}}]}`))
	}))
	defer server.Close()
	manager, err := NewMediaManager(MediaProvider{Name: "fixture", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	manager.Client = server.Client()
	workspace := t.TempDir()
	input := filepath.Join(workspace, "large.png")
	file, err := os.OpenFile(input, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(25<<20 + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = manager.AnalyzeImage(context.Background(), workspace, input, "describe", "vision")
	if err == nil || !strings.Contains(err.Error(), "exceeds 25 MiB") {
		t.Fatalf("expected pre-read size rejection, got %v", err)
	}
	if providerRequests != 0 {
		t.Fatalf("vision provider was contacted for oversized input: %d requests", providerRequests)
	}
}

func TestReadMediaFileHonorsCancellationAndBound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reads := 0
	reader := strings.NewReader("should not be read")
	wrapped := readerWithReadCounter{reader: reader, reads: &reads}
	if _, err := readMediaFile(ctx, wrapped, 1024); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if reads != 0 {
		t.Fatalf("reader was consumed after cancellation: %d reads", reads)
	}
	if _, err := readMediaFile(context.Background(), strings.NewReader("12345"), 4); err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("expected bounded read rejection, got %v", err)
	}
}

type readerWithReadCounter struct {
	reader interface{ Read([]byte) (int, error) }
	reads  *int
}

func (r readerWithReadCounter) Read(p []byte) (int, error) {
	*r.reads++
	return r.reader.Read(p)
}

func TestMediaProviderErrorsDoNotEchoResponseBodies(t *testing.T) {
	const sentinel = "media-opaque-provider-diagnostic"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(sentinel))
	}))
	defer server.Close()
	manager, err := NewMediaManager(MediaProvider{Name: "local", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	manager.Client = server.Client()
	if _, err := manager.postJSON(context.Background(), "/json", map[string]any{"input": "safe"}); err == nil || !strings.Contains(err.Error(), "502") || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("media JSON error=%v; expected status only and no provider body", err)
	}
	if _, err := manager.postBytes(context.Background(), "/bytes", map[string]any{"input": "safe"}); err == nil || !strings.Contains(err.Error(), "502") || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("media binary error=%v; expected status only and no provider body", err)
	}
}

func TestMediaProviderRejectsUserinfo(t *testing.T) {
	const sentinel = "media-userinfo-sentinel"
	if err := (MediaProvider{BaseURL: "https://" + sentinel + ":password@example.com", APIKey: "test-key"}).Validate(); err == nil || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("userinfo provider validation error=%v; expected rejection without userinfo", err)
	}
}

func TestMediaDownloadTransportErrorDoesNotEchoSignedQuery(t *testing.T) {
	const sentinel = "media-signed-query-sentinel"
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	serverURL := server.URL
	server.Close()
	manager, err := NewMediaManager(MediaProvider{Name: "local", BaseURL: "http://127.0.0.1:43123"})
	if err != nil {
		t.Fatal(err)
	}
	manager.Client = server.Client()
	_, _, err = manager.materializeEntry(context.Background(), t.TempDir(), "image", ".png", map[string]any{
		"url": serverURL + "/download?X-Amz-Signature=" + sentinel,
	})
	if err == nil || !strings.Contains(err.Error(), "media download failed") || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("media download error=%v; expected sanitized transport failure", err)
	}
}

func TestMediaOutputTextRedactsCredentialAssignments(t *testing.T) {
	const secret = "ghp_abcdefghijklmnopqrstuvwxyz123456"
	output := sanitizeMediaOutputText("recognized text: token=" + secret)
	if strings.Contains(output, secret) {
		t.Fatalf("media output retained a credential: %q", output)
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Fatalf("media output was not visibly redacted: %q", output)
	}
}

func TestMediaLoopbackHostnameCannotResolveToPublicAddress(t *testing.T) {
	ctx := context.WithValue(context.Background(), mediaLoopbackContextKey{}, true)
	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}, {IP: net.ParseIP("203.0.113.8")}}, nil
	}
	if _, err := mediaDialContextWithResolver(ctx, "tcp", "localhost:443", lookup); err == nil || !strings.Contains(err.Error(), "outside loopback") {
		t.Fatalf("media loopback accepted a public DNS answer: %v", err)
	}
}
