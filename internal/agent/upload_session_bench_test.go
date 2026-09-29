package agent

import (
	"bytes"
	"crypto/rand"
	"os"
	"strconv"
	"testing"
)

// runUpload envia sizeBytes em chunks de chunkBytes e finaliza com verificacao
// de hash. Retorna a sessao finalizada.
func runUpload(tb testing.TB, m *UploadManager, sizeBytes, chunkBytes int64) UploadSession {
	tb.Helper()
	payload := make([]byte, sizeBytes)
	if _, err := rand.Read(payload); err != nil {
		tb.Fatal(err)
	}
	sum := sha256Hex(payload)
	s, err := m.StartUpload("org-a", "prj", "big.bin", sizeBytes, chunkBytes, sum)
	if err != nil {
		tb.Fatal(err)
	}
	for off := int64(0); off < sizeBytes; off += chunkBytes {
		end := off + chunkBytes
		if end > sizeBytes {
			end = sizeBytes
		}
		if _, err := m.AppendChunk("org-a", s.ID, off, payload[off:end]); err != nil {
			tb.Fatalf("chunk off=%d: %v", off, err)
		}
	}
	done, err := m.FinalizeUpload("org-a", s.ID)
	if err != nil || done.State != UploadCompleted {
		tb.Fatalf("finalize: %+v err=%v", done, err)
	}
	got, err := os.ReadFile(done.FinalPath)
	if err != nil || !bytes.Equal(got, payload) {
		tb.Fatalf("conteudo final divergente: err=%v", err)
	}
	return done
}

// Teste de tamanho medio que roda no CI (16 MiB): prova multi-chunk + hash em
// escala nao-trivial de forma rapida.
func TestUploadSessionHandlesMediumFile(t *testing.T) {
	const size = 16 << 20 // 16 MiB
	m, err := NewUploadManager(t.TempDir(), 1<<30, size)
	if err != nil {
		t.Fatal(err)
	}
	done := runUpload(t, m, size, 4<<20) // chunks de 4 MiB
	if done.ReceivedBytes != size {
		t.Fatalf("received=%d", done.ReceivedBytes)
	}
}

// Benchmark de vazao. Tamanho via UPLOAD_BENCH_MB (default 100). Rode com:
//
//	UPLOAD_BENCH_MB=100 go test ./internal/agent -run x -bench BenchmarkUploadThroughput -benchtime=1x
func BenchmarkUploadThroughput(b *testing.B) {
	mb := int64(100)
	if v := os.Getenv("UPLOAD_BENCH_MB"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			mb = parsed
		}
	}
	size := mb << 20
	b.SetBytes(size)
	for range b.N {
		m, err := NewUploadManager(b.TempDir(), size*4, size)
		if err != nil {
			b.Fatal(err)
		}
		runUpload(b, m, size, 8<<20) // chunks de 8 MiB
	}
}
