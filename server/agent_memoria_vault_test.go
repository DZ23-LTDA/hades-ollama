package server

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

func TestIdentidadeDaSessaoSoVemDoContexto(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. NEGATIVO: sem sessao nao ha' identidade -- e um corpo com identidade nao a cria.
	c, _ := gin.CreateTestContext(nil)
	c.Request = nil
	if id, ok := IdentidadeDaSessao(c); ok || id != "" {
		t.Fatalf("sem sessao devia devolver vazio, veio %q", id)
	}

	// 2. NEGATIVO: valor de tipo errado nao e' aceite por conveniencia.
	c2, _ := gin.CreateTestContext(nil)
	c2.Set("agent.user", "utilizador-forjado")
	if id, ok := IdentidadeDaSessao(c2); ok || id != "" {
		t.Fatalf("identidade de tipo errado devia ser recusada, veio %q", id)
	}

	// 3. NEGATIVO: ID vazio nao e' identidade.
	c3, _ := gin.CreateTestContext(nil)
	c3.Set("agent.user", agent.User{ID: "   "})
	if id, ok := IdentidadeDaSessao(c3); ok || id != "" {
		t.Fatalf("ID em branco devia ser recusado, veio %q", id)
	}

	// 4. POSITIVO: o que o middleware pos e' o que sai.
	c4, _ := gin.CreateTestContext(nil)
	c4.Set("agent.user", agent.User{ID: "user-123"})
	if id, ok := IdentidadeDaSessao(c4); !ok || id != "user-123" {
		t.Fatalf("esperava user-123, veio %q (ok=%v)", id, ok)
	}
}

func TestOrganizacaoDaSessaoSoVemDoContexto(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	if id, ok := OrganizacaoDaSessao(c); ok || id != "" {
		t.Fatalf("sem organizacao devia devolver vazio, veio %q", id)
	}
	c2, _ := gin.CreateTestContext(nil)
	c2.Set("agent.organization", agent.Organization{ID: "org-9"})
	if id, ok := OrganizacaoDaSessao(c2); !ok || id != "org-9" {
		t.Fatalf("esperava org-9, veio %q (ok=%v)", id, ok)
	}
}

func TestDecodificarCorpoEstritoRecusaIdentidadeNoCorpo(t *testing.T) {
	type pedido struct {
		Relativo string `json:"relativo"`
	}

	// NEGATIVO: um corpo que tenta impor a identidade e' RECUSADO, nao aceito por omissao.
	corpoComIdentidade := strings.NewReader(`{"relativo":"a/b.md","identidade":"outro-utilizador"}`)
	var p pedido
	if err := DecodificarCorpoEstrito(corpoComIdentidade, &p); err == nil {
		t.Fatal("corpo com campo identidade devia ser recusado")
	}

	// NEGATIVO: dois valores JSON no mesmo corpo (contrabando) tambem.
	var p2 pedido
	if err := DecodificarCorpoEstrito(strings.NewReader(`{"relativo":"a.md"}{"relativo":"b.md"}`), &p2); err == nil {
		t.Fatal("dois valores JSON deviam ser recusados")
	}

	// POSITIVO: o contrato declarado passa.
	var p3 pedido
	if err := DecodificarCorpoEstrito(strings.NewReader(`{"relativo":"a/b.md"}`), &p3); err != nil {
		t.Fatalf("corpo valido devia ser aceite: %v", err)
	}
	if p3.Relativo != "a/b.md" {
		t.Fatalf("relativo perdido: %q", p3.Relativo)
	}
}
