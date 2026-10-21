package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// A ROTA DE BUILD CONSULTA O PARE DO SERVIDOR.
//
// Antes desta fatia, o executor sabia consultar o travao mas **nenhum handler lho passava** -- em
// producao a verificacao nunca acontecia. O teste aciona o pare do servidor, chama a rota, e exige:
// recusa, motivo no corpo, e NENHUM directorio criado.
func TestRotaDeBuildRecusaComOPareDoServidorAcionado(t *testing.T) {
	directorio := t.TempDir()
	destino := filepath.Join(directorio, "dist-que-nao-deve-existir")
	if err := os.WriteFile(filepath.Join(directorio, "go.mod"),
		[]byte("module exemplo.pt/hades\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directorio, "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatalf("escrever main.go: %v", err)
	}

	// Acionar o pare DO SERVIDOR e garantir que e' desacionado no fim, para nao contaminar outros
	// testes que correm no mesmo pacote.
	if err := pareDoServidor.Acionar("vazamento entre tenants em investigacao", "teste"); err != nil {
		t.Fatalf("acionar o pare do servidor: %v", err)
	}
	t.Cleanup(func() {
		_ = pareDoServidor.Rearmar("teste", "limpeza do teste")
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(sessaoDeBuild("org-teste"))
	router.POST(rotaDoInicioDeBuild, (&agentAPI{}).iniciarBuild)

	codigo, corpo := pedirInicioDeBuild(t, router, map[string]any{
		"raiz":    directorio,
		"alvo":    "linux",
		"destino": destino,
		"dry_run": true,
	})

	if codigo == http.StatusOK || codigo == http.StatusCreated {
		t.Fatalf("com o pare acionado a rota NAO pode aceitar o build (veio %d): %v", codigo, corpo)
	}
	texto := ""
	for _, chave := range []string{"erro", "detalhe"} {
		if v, ok := corpo[chave].(string); ok {
			texto += v + " "
		}
	}
	if !strings.Contains(texto, "vazamento entre tenants") {
		t.Fatalf("a recusa tem de citar o motivo do pare; corpo: %v", corpo)
	}
	if _, errStat := os.Stat(destino); !os.IsNotExist(errStat) {
		t.Fatalf("o destino NAO pode ter sido criado com o pare acionado (stat: %v)", errStat)
	}
}
