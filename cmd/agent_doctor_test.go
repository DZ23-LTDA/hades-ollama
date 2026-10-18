package cmd

// FASE 8 do adendo V3: "codigos de saida documentados".
//
// O discriminador que importa: um doctor que devolve sempre o mesmo codigo nao e' um
// diagnostico -- e' uma constante com nome bonito.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// exitCoderDe devolve o codigo declarado pelo erro, ou -1 se ele nao declarar nenhum.
func exitCoderDe(err error) int {
	var exit interface{ ExitCode() int }
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return -1
}

// servidorDoctor levanta um endpoint cujas rotas respondem conforme o mapa.
// Uma rota ausente do mapa responde 404.
func servidorDoctor(t *testing.T, rotas map[string]int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, existe := rotas[r.URL.Path]
		if !existe {
			http.NotFound(w, r)
			return
		}
		if status == 200 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{}`)
			return
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	t.Setenv("OLLAMA_HOST", server.URL)
	return server
}

func TestDoctorSaudavelSaiComZero(t *testing.T) {
	servidorDoctor(t, map[string]int{
		"/api/agent/v1/health":  200,
		"/api/agent/v1/tools":   200,
		"/api/agent/v1/presets": 200,
	})
	buffer := capturarSaida(t)
	if err := runAgentDoctor(context.Background()); err != nil {
		t.Fatalf("endpoint saudavel devia sair com 0, veio: %v", err)
	}
	var relatorio doctorReport
	if err := json.Unmarshal(buffer.Bytes(), &relatorio); err != nil {
		t.Fatalf("relatorio devia ser JSON: %v", err)
	}
	if relatorio.Veredicto != "saudavel" || relatorio.ExitCode != 0 {
		t.Fatalf("veredicto = %q exit = %d, queria saudavel/0", relatorio.Veredicto, relatorio.ExitCode)
	}
}

// TestDoctorSemSaudeSaiComUm e o discriminador do exit 1.
func TestDoctorSemSaudeSaiComUm(t *testing.T) {
	servidorDoctor(t, map[string]int{
		"/api/agent/v1/health":  500,
		"/api/agent/v1/tools":   200,
		"/api/agent/v1/presets": 200,
	})
	capturarSaida(t)
	err := runAgentDoctor(context.Background())
	if err == nil {
		t.Fatal("sem saude o doctor devia devolver erro")
	}
	if codigo := exitCoderDe(err); codigo != 1 {
		t.Fatalf("exit code = %d, queria 1 (inutilizavel)", codigo)
	}
}

// TestDoctorDegradadoSaiComDois e o discriminador do exit 2 -- e o que impede o
// doctor de tratar "o essencial vive" e "tudo vive" como a mesma coisa.
func TestDoctorDegradadoSaiComDois(t *testing.T) {
	servidorDoctor(t, map[string]int{
		"/api/agent/v1/health":  200,
		"/api/agent/v1/tools":   500, // nao-essencial a falhar
		"/api/agent/v1/presets": 200,
	})
	capturarSaida(t)
	err := runAgentDoctor(context.Background())
	if err == nil {
		t.Fatal("com um check nao-essencial em falha o doctor devia devolver erro")
	}
	if codigo := exitCoderDe(err); codigo != 2 {
		t.Fatalf("exit code = %d, queria 2 (degradado)", codigo)
	}
}

// TestDoctorRotaAusenteNaoContaComoFalha: um servidor mais antigo sem /presets nao
// esta' "degradado" -- apenas nao tem a rota. *Inventar uma falha para uma sonda que
// nao existe seria um diagnostico que mente.*
func TestDoctorRotaAusenteNaoContaComoFalha(t *testing.T) {
	servidorDoctor(t, map[string]int{
		"/api/agent/v1/health": 200,
		"/api/agent/v1/tools":  200,
		// /api/agent/v1/presets ausente -> 404 -> indisponivel
	})
	buffer := capturarSaida(t)
	if err := runAgentDoctor(context.Background()); err != nil {
		t.Fatalf("rota ausente nao devia degradar: %v", err)
	}
	var relatorio doctorReport
	if err := json.Unmarshal(buffer.Bytes(), &relatorio); err != nil {
		t.Fatalf("relatorio devia ser JSON: %v", err)
	}
	var presets doctorCheck
	for _, c := range relatorio.Checks {
		if strings.Contains(c.Rota, "presets") {
			presets = c
		}
	}
	if presets.Estado != doctorOmitido {
		t.Fatalf("presets devia ser %q, veio %q", doctorOmitido, presets.Estado)
	}
	if relatorio.Veredicto != "saudavel" {
		t.Fatalf("veredicto devia ser saudavel, veio %q", relatorio.Veredicto)
	}
}

// TestDoctorNuncaExpoeOToken: o relatorio diz SE existe, nunca o valor.
func TestDoctorNuncaExpoeOToken(t *testing.T) {
	const segredo = "token-que-nao-pode-aparecer-no-relatorio"
	t.Setenv("OLLAMA_AGENT_TOKEN", segredo)
	servidorDoctor(t, map[string]int{
		"/api/agent/v1/health":  200,
		"/api/agent/v1/tools":   200,
		"/api/agent/v1/presets": 200,
	})
	buffer := capturarSaida(t)
	if err := runAgentDoctor(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buffer.String(), segredo) {
		t.Fatalf("o relatorio vazou o token: %q", buffer.String())
	}
}

// TestDoctorJSONEmiteUmaLinha: o doctor respeita o `--json` como toda a CLI.
func TestDoctorJSONEmiteUmaLinha(t *testing.T) {
	servidorDoctor(t, map[string]int{
		"/api/agent/v1/health":  200,
		"/api/agent/v1/tools":   200,
		"/api/agent/v1/presets": 200,
	})
	agentJSONOutput = true
	buffer := capturarSaida(t)
	if err := runAgentDoctor(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.TrimSpace(buffer.String()), "\n") {
		t.Fatalf("--json devia ser uma linha: %q", buffer.String())
	}
}

// TestDoctorErroDeclaraExitCode garante o contrato com o main.go.
func TestDoctorErroDeclaraExitCode(t *testing.T) {
	err := doctorErro{Codigo: 2}
	if exitCoderDe(err) != 2 {
		t.Fatalf("doctorErro devia declarar o codigo 2, veio %d", exitCoderDe(err))
	}
	if !strings.Contains(err.Error(), "degradado") {
		t.Fatalf("a mensagem devia nomear o veredicto: %v", err)
	}
}

var _ = bytes.MinRead
