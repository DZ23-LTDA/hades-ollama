package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// E-671 — A SONDA DE CAPACIDADE: ela NAO se deixa enganar por um endpoint que responde mas nao gera.
//
// **O DEFEITO MEDIDO (2026-10-15):** `OllamaDisponivel` devolveu `true` num host onde a GERACAO nunca respondia; a
// matriz real gastou **3 min por chamada** (timeout do cliente, ate 16 chamadas) e o pacote `internal/agent` morreu
// no limite de 10 min do `go test` — *a suite parecia "lenta" quando a sonda estava perguntando a coisa errada.*
//
// ***"O endpoint responde" nao e' "o modelo gera".*** Este teste codifica a diferenca, com um servidor que responde
// `/api/tags` com 200 e **trava** em `/api/generate` — exatamente o engodo que custou 10 minutos em silencio.
func TestSondaDeCapacidadeNaoSeDeixaEnganarPorEndpointQueResponde(t *testing.T) {
	// Orcamento curto: *a sonda existe para NAO gastar minutos descobrindo que nao da' para medir.*
	t.Setenv("HADES_T2_PROBE_TIMEOUT", "1")

	travado := make(chan struct{})
	servidor := httptest.NewServer(http.HandlerFunc(func(escritor http.ResponseWriter, requisicao *http.Request) {
		if requisicao.URL.Path == "/api/tags" {
			escritor.WriteHeader(http.StatusOK)
			_, _ = escritor.Write([]byte(`{"models":[]}`))
			return
		}
		<-travado // /api/generate NUNCA responde — e' o defeito medido, reproduzido de proposito
	}))
	// ATENCAO a ordem: `Close()` espera os handlers; o canal tem de ser fechado ANTES (defers sao LIFO).
	defer servidor.Close()
	defer close(travado)

	// (1) O engodo: a sonda de ENDPOINT diz que esta' tudo bem.
	if !OllamaDisponivel(context.Background(), servidor.URL) {
		t.Fatal("o servidor responde 200 a /api/tags: a sonda de DISPONIBILIDADE tem de devolver true — *e' esse o engano*")
	}

	// (2) A sonda de CAPACIDADE tem de RECUSAR, e rapido.
	inicio := time.Now()
	pronto := OllamaModeloPronto(context.Background(), servidor.URL, "qwen2.5:7b")
	decorrido := time.Since(inicio)
	if pronto {
		t.Fatal("a GERACAO travou: a sonda de capacidade NAO pode dizer 'pronto' — *era exactamente aqui que a suite " +
			"morria em silencio*")
	}
	if decorrido > 20*time.Second {
		t.Fatalf("a sonda gastou %s — *a sonda existe para NAO gastar minutos descobrindo que nao da' para medir*", decorrido)
	}
	t.Logf("medido: endpoint responde, geracao TRAVA, sonda de capacidade devolveu false em %s", decorrido)
}

// CONTRAPOSTA: quando a geracao COMPLETA, a sonda aprova — *senao ela nao seria uma sonda, seria uma recusa fixa.*
func TestSondaDeCapacidadeAprovaGeracaoQueCompleta(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(escritor http.ResponseWriter, requisicao *http.Request) {
		if requisicao.URL.Path == "/api/generate" {
			escritor.Header().Set("Content-Type", "application/json")
			_, _ = escritor.Write([]byte(`{"response":"ok","done":true}`))
			return
		}
		escritor.WriteHeader(http.StatusNotFound)
	}))
	defer servidor.Close()

	if !OllamaModeloPronto(context.Background(), servidor.URL, "qwen2.5:7b") {
		t.Fatal("a geracao completou com conteudo: a sonda de capacidade tem de APROVAR")
	}
}

// CONTRAPOSTA 2: `200` com corpo VAZIO NAO prova geracao — *era essa a brecha que a sonda de endpoint aceitava.*
func TestSondaDeCapacidadeRecusaRespostaVazia(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(escritor http.ResponseWriter, _ *http.Request) {
		escritor.WriteHeader(http.StatusOK)
	}))
	defer servidor.Close()

	if OllamaModeloPronto(context.Background(), servidor.URL, "qwen2.5:7b") {
		t.Fatal("resposta 200 VAZIA nao prova que o modelo gerou")
	}
}
