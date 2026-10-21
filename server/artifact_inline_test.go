package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// I1a: o painel de construcao ao vivo precisa de EXECUTAR o artefacto (iframe), nao de o
// descarregar. FileAttachment manda "Content-Disposition: attachment"; com ?inline=1 a mesma
// rota passa a servir inline com content-type correcto. O comportamento por omissao nao muda.
func artigofalso(t *testing.T, nome, conteudo string) string {
	t.Helper()
	dir := t.TempDir()
	caminho := filepath.Join(dir, nome)
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("criar artefacto: %v", err)
	}
	return caminho
}

func servirInline(t *testing.T, caminho, nome, query string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Handler minimo que reproduz o ramo testado da rota real.
	r.GET("/artefacto", func(c *gin.Context) {
		c.Header("X-Artifact-SHA256", "abc123")
		if strings.EqualFold(c.Query("inline"), "1") {
			c.Header("Content-Disposition", "inline")
			c.Header("Content-Type", inlineContentType(nome))
			c.Header("Cache-Control", "no-store")
			c.File(caminho)
			return
		}
		c.FileAttachment(caminho, nome)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/artefacto"+query, nil))
	return w
}

// POSITIVO: inline serve HTML renderizavel, com SHA e sem cache.
func TestArtifactInlineServeHTMLRenderizavel(t *testing.T) {
	caminho := artigofalso(t, "index.html", "<h1>app gerada</h1>")
	w := servirInline(t, caminho, "index.html", "?inline=1")

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d", w.Code)
	}
	if got := w.Header().Get("Content-Disposition"); got != "inline" {
		t.Fatalf("Content-Disposition = %q, queria inline", got)
	}
	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("Content-Type = %q, queria text/html", got)
	}
	if got := w.Header().Get("X-Artifact-SHA256"); got != "abc123" {
		t.Fatalf("SHA256 perdido: %q", got)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, queria no-store", got)
	}
	if !strings.Contains(w.Body.String(), "app gerada") {
		t.Fatalf("corpo inesperado: %q", w.Body.String())
	}
}

// NEGATIVO: sem inline o comportamento antigo mantem-se (download).
func TestArtifactSemInlineContinuaDownload(t *testing.T) {
	caminho := artigofalso(t, "index.html", "<h1>x</h1>")
	w := servirInline(t, caminho, "index.html", "")

	if !strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("sem inline devia ser attachment, veio %q", w.Header().Get("Content-Disposition"))
	}
}

// POSITIVO: content-type por extensao (o preview tambem serve css/js/imagens).
func TestInlineContentTypePorExtensao(t *testing.T) {
	casos := map[string]string{
		"index.html": "text/html",
		"app.css":    "text/css",
		"app.js":     "text/javascript",
		"data.json":  "application/json",
		"logo.svg":   "image/svg+xml",
		"sem_ext":    "text/html",
	}
	for nome, esperado := range casos {
		if got := inlineContentType(nome); !strings.HasPrefix(got, esperado) {
			t.Fatalf("inlineContentType(%q) = %q, queria prefixo %q", nome, got, esperado)
		}
	}
}
