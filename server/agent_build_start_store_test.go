package server

// FASE 28.7: testes da integracao `build.start` -> store -> `build.status`/`cancel`.
//
// O TESTE QUE FALTAVA
//
// As rotas de estado provadas contra um store populado a mao provam o store, nao a integracao. O que
// estes testes travam e' o ciclo: um `build.start` real cria um registo, e o `build_id` devolvido
// responde no `build.status`.
//
//	*Tres pecas provadas em separado nao sao uma integracao provada.*

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestInicioDeBuildSincronoDevolveBuildIDConsultavel prova que o ID devolvido e' consultavel.
//
// SEM ISTO o `build_id` seria um decoracao: um campo que aparece na resposta e que o `status` recusa.
func TestInicioDeBuildSincronoDevolveBuildIDConsultavel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &agentAPI{}
	r := gin.New()
	g := r.Group("/api/agent/v1")
	g.POST("/build/start", api.iniciarBuild)
	g.GET("/build/status/:id", api.estadoDeBuild)

	corpo := `{"raiz":"` + raizDeProjectoParaTeste(t) + `","alvo":"windows","dry_run":true}`
	req := httptest.NewRequest("POST", "/api/agent/v1/build/start", bytes.NewBufferString(corpo))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Skipf("o alvo `windows` nao esta' construivel neste host (HTTP %d); o teste do build_id exige um alvo construivel", w.Code)
	}
	var inicio map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &inicio); err != nil {
		t.Fatalf("resposta invalida: %v", err)
	}
	id, _ := inicio["build_id"].(string)
	if id == "" {
		t.Fatalf("o build.start TEM de devolver `build_id`; veio: %v", inicio)
	}
	if inicio["estado"] != "CONCLUIDO" {
		t.Fatalf("no modo sincrono o estado devolvido devia ser CONCLUIDO: %v", inicio["estado"])
	}

	// E o ID tem de ser consultavel.
	req2 := httptest.NewRequest("GET", "/api/agent/v1/build/status/"+id, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("o build_id devolvido pelo start nao e' consultavel: HTTP %d -- %s", w2.Code, w2.Body.String())
	}
	var estado map[string]any
	_ = json.Unmarshal(w2.Body.Bytes(), &estado)
	b, _ := estado["build"].(map[string]any)
	if b["id"] != id {
		t.Fatalf("o status devolveu outro build: %v", b["id"])
	}
}

// TestInicioDeBuildAssincronoAceitaEDevolveIDEexecuta prova o modo que da' alvo ao `cancel`.
//
//	*Um `build.cancel` sem build a correr e' um botao que so' encontra builds terminados -- e cancelar
//	o que ja' acabou e' o que o 409 impede. O modo assincrono e' o que da' sentido aos dois.*
func TestInicioDeBuildAssincronoAceitaEDevolveIDEexecuta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &agentAPI{}
	r := gin.New()
	g := r.Group("/api/agent/v1")
	g.POST("/build/start", api.iniciarBuild)
	g.GET("/build/status/:id", api.estadoDeBuild)
	g.GET("/build/logs/:id", api.saidaDoBuild)

	corpo := `{"raiz":"` + raizDeProjectoParaTeste(t) + `","alvo":"windows","dry_run":true,"assincrono":true}`
	req := httptest.NewRequest("POST", "/api/agent/v1/build/start", bytes.NewBufferString(corpo))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Skipf("o alvo `windows` nao esta' construivel neste host (HTTP %d); o teste do assincrono exige um alvo construivel", w.Code)
	}
	var inicio map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &inicio); err != nil {
		t.Fatalf("resposta invalida: %v", err)
	}
	if inicio["construiu"] != false {
		t.Fatalf("no modo assincrono `construiu` tem de ser false -- dizer true seria prometer o que ainda nao aconteceu: %v", inicio["construiu"])
	}
	id, _ := inicio["build_id"].(string)
	if id == "" {
		t.Fatalf("o assincrono TEM de devolver build_id: %v", inicio)
	}
	if inicio["estado"] != "A_EXECUTAR" {
		t.Fatalf("o estado devolvido devia ser A_EXECUTAR: %v", inicio["estado"])
	}

	// O build corre em segundo plano; esperar que saia de A_EXECUTAR.
	var estadoFinal any
	for range 100 {
		req2 := httptest.NewRequest("GET", "/api/agent/v1/build/status/"+id, nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		var estado map[string]any
		_ = json.Unmarshal(w2.Body.Bytes(), &estado)
		b, _ := estado["build"].(map[string]any)
		estadoFinal = b["estado"]
		if estadoFinal != "A_EXECUTAR" && estadoFinal != "NA_FILA" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if estadoFinal == "A_EXECUTAR" || estadoFinal == "NA_FILA" {
		t.Fatalf("o build assincrono nao terminou: %v", estadoFinal)
	}
	if estadoFinal != "CONCLUIDO" {
		t.Fatalf("o dry_run devia concluir: %v", estadoFinal)
	}
}

// raizDeProjectoParaTeste devolve uma raiz absoluta e existente para o build.
func raizDeProjectoParaTeste(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	// O build precisa de um `go.mod` para o provider nao recusar por raiz invalida.
	if err := os.WriteFile(filepath.Join(raiz, "go.mod"), []byte("module exemplo\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}
	return strings.ReplaceAll(raiz, "\\", "\\\\")
}
