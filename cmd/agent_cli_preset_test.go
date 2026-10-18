package cmd

// FASE 8 do adendo V3: a CLI tem de ser operavel por humanos E por agentes.
//
// Duas propriedades provadas aqui:
//   1. `--json` produz UMA linha compacta (o que um script consome); sem a flag, indentado.
//   2. `agent preset` liga a CLI ao catalogo de presets de MISSAO (FASE 23).

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// capturarSaida troca o writer da CLI e devolve o buffer. Restaura no fim para nao
// contaminar os outros testes do pacote.
func capturarSaida(t *testing.T) *bytes.Buffer {
	t.Helper()
	anterior := agentCLIOut
	anteriorJSON := agentJSONOutput
	buffer := &bytes.Buffer{}
	agentCLIOut = buffer
	t.Cleanup(func() {
		agentCLIOut = anterior
		agentJSONOutput = anteriorJSON
	})
	return buffer
}

const catalogoDeTeste = `{"presets":[
 {"id":"briefing-diario","name":"Briefing diario","completion_criteria":["nomeia o que mudou"]},
 {"id":"monitor-agendado","name":"Monitor agendado","completion_criteria":["avisa so quando muda"]}
],"count":2}`

func servidorDePresets(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agent/v1/presets" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, catalogoDeTeste)
	}))
	t.Cleanup(server.Close)
	t.Setenv("OLLAMA_HOST", server.URL)
	return server
}

// TestAgentJSONEmiteUmaLinha e o discriminador da flag.
//
// *Um `--json` que continua indentado nao e' JSON para automacao: e' JSON bonito.*
func TestAgentJSONEmiteUmaLinha(t *testing.T) {
	servidorDePresets(t)

	// Sem a flag: indentado, varias linhas.
	agentJSONOutput = false
	humano := capturarSaida(t)
	if err := runAgentRequest(context.Background(), http.MethodGet, "/api/agent/v1/presets", nil); err != nil {
		t.Fatal(err)
	}
	if linhas := strings.Count(strings.TrimSpace(humano.String()), "\n"); linhas == 0 {
		t.Fatalf("sem --json a saida devia ser indentada, veio: %q", humano.String())
	}

	// Com a flag: UMA linha compacta.
	agentJSONOutput = true
	maquina := capturarSaida(t)
	if err := runAgentRequest(context.Background(), http.MethodGet, "/api/agent/v1/presets", nil); err != nil {
		t.Fatal(err)
	}
	saida := strings.TrimSpace(maquina.String())
	if strings.Contains(saida, "\n") {
		t.Fatalf("--json devia produzir UMA linha, veio %d linhas: %q", strings.Count(saida, "\n")+1, saida)
	}
	var decodificado map[string]any
	if err := json.Unmarshal([]byte(saida), &decodificado); err != nil {
		t.Fatalf("--json devia ser JSON valido numa linha: %v (%q)", err, saida)
	}
}

func TestAgentPresetListDevolveOCatalogo(t *testing.T) {
	servidorDePresets(t)
	agentJSONOutput = true
	buffer := capturarSaida(t)
	if err := runAgentRequest(context.Background(), http.MethodGet, "/api/agent/v1/presets", nil); err != nil {
		t.Fatal(err)
	}
	saida := buffer.String()
	if !strings.Contains(saida, "briefing-diario") || !strings.Contains(saida, "monitor-agendado") {
		t.Fatalf("o catalogo devia trazer os dois presets: %q", saida)
	}
}

func TestAgentPresetShowDevolveOPresetPedido(t *testing.T) {
	servidorDePresets(t)
	agentJSONOutput = true
	buffer := capturarSaida(t)
	if err := runAgentPresetShow(context.Background(), "monitor-agendado"); err != nil {
		t.Fatal(err)
	}
	saida := buffer.String()
	if !strings.Contains(saida, "monitor-agendado") {
		t.Fatalf("devia trazer o preset pedido: %q", saida)
	}
	if strings.Contains(saida, "briefing-diario") {
		t.Fatalf("devia trazer SO' o preset pedido: %q", saida)
	}
}

func TestAgentPresetShowRecusaIdInexistente(t *testing.T) {
	servidorDePresets(t)
	capturarSaida(t)
	err := runAgentPresetShow(context.Background(), "preset-que-nao-existe")
	if err == nil {
		t.Fatal("preset inexistente devia devolver erro (exit code != 0)")
	}
	if !strings.Contains(err.Error(), "preset-que-nao-existe") {
		t.Fatalf("o erro devia nomear o id pedido: %v", err)
	}
	// E deve dizer como descobrir os validos, em vez de deixar o utilizador a adivinhar.
	if !strings.Contains(err.Error(), "preset list") {
		t.Fatalf("o erro devia indicar `agent preset list`: %v", err)
	}
}

func TestAgentPresetShowRecusaIdVazio(t *testing.T) {
	capturarSaida(t)
	if err := runAgentPresetShow(context.Background(), "   "); err == nil {
		t.Fatal("id vazio devia ser recusado")
	}
}
