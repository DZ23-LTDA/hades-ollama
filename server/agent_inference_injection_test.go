package server

import (
	"testing"
)

// E-667 — A INJECAO EM PRODUCAO (o que faltava no E-666).
//
// O executor `model.infer` existia e estava provado **dentro** do pacote `agent` — onde o cliente de inferencia e'
// injetado a mao pelo proprio teste. Isso **nao** prova que o SERVIDOR injeta: *um executor provado so' em teste de
// pacote e' um executor que ninguem alcanca em producao.*
//
// Este teste passa pelo caminho REAL de construcao (`newDefaultAgentRuntime`) e mede o que ficou registrado, nas
// DUAS direcoes — porque a direcao que costuma faltar e' a segunda:
//  1. com `OLLAMA_AGENT_MODEL` configurado, o kind `model.infer` **existe**;
//  2. **sem** modelo configurado, o kind **nao existe** — *fail-closed: e' melhor recusar no planejamento do que
//     aceitar o passo e falhar no meio da execucao.*
func TestRuntimeDeProducaoRegistraInferenciaSomenteComModeloConfigurado(t *testing.T) {
	casos := []struct {
		nome           string
		modelo         string
		querRegistrado bool
	}{
		{nome: "com modelo configurado", modelo: "qwen2.5:7b", querRegistrado: true},
		{nome: "sem modelo configurado", modelo: "", querRegistrado: false},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			t.Setenv("OLLAMA_AGENT_ROOT", t.TempDir())
			t.Setenv("OLLAMA_AGENT_STORE", t.TempDir())
			t.Setenv("OLLAMA_AGENT_DATABASE_URL", "")
			t.Setenv("OLLAMA_AGENT_REDIS_URL", "")
			t.Setenv("OLLAMA_AGENT_MODEL", caso.modelo)

			runtime, err := newDefaultAgentRuntime()
			if err != nil {
				t.Fatalf("construir o runtime de producao: %v", err)
			}
			kinds := runtime.RegisteredToolKinds()
			registrado := false
			for _, kind := range kinds {
				if kind == "model.infer" {
					registrado = true
				}
			}
			if registrado != caso.querRegistrado {
				t.Fatalf("`model.infer` registrado=%v, esperado %v — *a injecao de dependencia tem de decidir o "+
					"registro do kind*; kinds=%v", registrado, caso.querRegistrado, kinds)
			}
			if len(kinds) == 0 {
				t.Fatal("o runtime de producao tem de registrar as ferramentas base — *um registry vazio esvaziaria a assercao*")
			}
		})
	}
}
