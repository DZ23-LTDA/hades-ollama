// Testes do executor de avaliacao.
//
// Discriminantes: sem motor, a rota RECUSA ANTES de inferir (nao gasta tempo do utilizador para lhe dizer o
// que ja' se sabia); o endpoint respeita `OLLAMA_HOST` sem inventar esquema; um modelo ausente e' recusado
// com os disponiveis listados.
package server

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// *Aprovar uma acao sem ver o custo e aprovar um numero que nao se leu.*
func TestSemMotorNaoInferimos(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "http://127.0.0.1:1") // porta fechada: nada escuta
	ctx := context.Background()

	m := detectarMotor(ctx, "")
	if m.Disponivel {
		t.Fatalf("motor declarado disponivel numa porta fechada: %+v", m)
	}
	if m.Motivo == "" {
		t.Fatal("indisponivel sem motivo: *um estado que mente e' pior do que nenhum*")
	}

	// A recusa tem de acontecer ANTES de qualquer inferencia.
	_, _, err := executarAvaliacao(ctx, "hades-executor", "engenharia", "")
	if !errors.Is(err, errMotorIndisponivel) {
		t.Fatalf("esperava errMotorIndisponivel, veio %v", err)
	}
}

func TestOllamaEndpointRespeitaOHost(t *testing.T) {
	casos := map[string]string{
		"":                       "http://127.0.0.1:11434",
		"http://127.0.0.1:12345": "http://127.0.0.1:12345",
		"https://exemplo.test/":  "https://exemplo.test",
		"127.0.0.1:9999":         "http://127.0.0.1:9999",
		"  http://host:1  ":      "http://host:1",
	}
	for entrada, esperado := range casos {
		t.Setenv("OLLAMA_HOST", entrada)
		if obtido := ollamaEndpoint(); obtido != esperado {
			t.Fatalf("OLLAMA_HOST=%q: esperava %q, veio %q", entrada, esperado, obtido)
		}
	}
}

// Um pedido invalido com o motor em baixo tem de recusar pelo MOTOR, nao pela persona: a ordem importa,
// porque a mensagem ao utilizador e' diferente.
func TestMotorEmBaixoVemAntesDaPersona(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "http://127.0.0.1:1")
	_, _, err := executarAvaliacao(context.Background(), "persona-que-nao-existe", "divisao-que-nao-existe", "")
	if !errors.Is(err, errMotorIndisponivel) {
		t.Fatalf("a persona foi validada antes do motor: %v", err)
	}
}

// Nenhuma execucao pode devolver PRODUCTION_APPROVED -- nem quando o motor responde.
func TestExecutorNuncaAprovaProducao(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "http://127.0.0.1:1")
	rel, _, _ := executarAvaliacao(context.Background(), "hades-executor", "engenharia", "")
	// Sem motor o relatorio vem vazio; o que se afirma e' que o campo zero nao e' uma aprovacao.
	if strings.Contains(string(rel.State), "PRODUCTION_APPROVED") {
		t.Fatalf("estado de aprovacao sem execucao: %q", rel.State)
	}
}

func TestLimiteDeCasosExiste(t *testing.T) {
	if evalRunnerMaxCasos <= 0 {
		t.Fatal("sem teto de casos: uma avaliacao sem fim e' um timeout")
	}
	if evalRunnerTimeout <= 0 {
		t.Fatal("sem timeout por chamada: um caso pendurado pendura a suite")
	}
}
