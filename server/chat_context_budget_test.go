package server

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/api"
)

// Testes do orçamento de contexto no funil do chat (§2 do complemento de memória).
// Nenhum deles chama modelo, rede ou disco.

func TestOrcamentoDoChatNaoInventaTeto(t *testing.T) {
	// A PROPRIEDADE QUE MAIS IMPORTA: sem o operador declarar o teto, NADA muda.
	t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_TOKENS", "")
	t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_RESERVE_TOKENS", "")
	g := OrcamentoDeContextoDoChat()
	if g.OrcamentoTokens != 0 {
		t.Fatalf("teto inventado sem configuração: %d", g.OrcamentoTokens)
	}
	d := DecidirOrcamentoDoChat("preambulo", []api.Message{{Role: "user", Content: strings.Repeat("x", 4000)}}, g)
	if d.Compactar {
		t.Fatal("decidiu compactar sem orçamento declarado")
	}
	if !strings.Contains(d.Motivo, "não declarado") {
		t.Fatalf("motivo não declara a ausência de orçamento: %q", d.Motivo)
	}
	// E o valor inválido (texto, zero, negativo) também NÃO vira um teto.
	for _, bruto := range []string{"abc", "0", "-5", "  "} {
		t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_TOKENS", bruto)
		if g := OrcamentoDeContextoDoChat(); g.OrcamentoTokens != 0 {
			t.Fatalf("valor %q virou teto %d", bruto, g.OrcamentoTokens)
		}
	}
}

func TestOrcamentoDoChatDecideEDeclaraEstimativa(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_TOKENS", "1000")
	t.Setenv("OLLAMA_AGENT_CHAT_CONTEXT_RESERVE_TOKENS", "200")
	g := OrcamentoDeContextoDoChat()
	if g.OrcamentoTokens != 1000 || g.ReservaTokens != 200 {
		t.Fatalf("gatilho lido = %+v", g)
	}
	// 801 tokens estimados > 800 disponíveis ⇒ compacta, e o motivo traz os números.
	grande := []api.Message{{Role: "user", Content: strings.Repeat("x", 3204)}}
	d := DecidirOrcamentoDoChat("", grande, g)
	if !d.Compactar {
		t.Fatalf("não decidiu compactar acima do orçamento: %+v", d)
	}
	if !strings.Contains(d.Motivo, "801") || !strings.Contains(d.Motivo, "800") {
		t.Fatalf("motivo sem os números: %q", d.Motivo)
	}
	if d.Uso.Exata {
		t.Fatal("uso marcado como exato: não há tokenizer neste runtime")
	}
	if !strings.Contains(d.Motivo, "estimado") {
		t.Fatalf("motivo não rotula a estimativa: %q", d.Motivo)
	}
	// Abaixo do orçamento: NÃO compacta — e o motivo diz porquê.
	d = DecidirOrcamentoDoChat("curto", []api.Message{{Role: "user", Content: "oi"}}, g)
	if d.Compactar || !strings.Contains(d.Motivo, "dentro") {
		t.Fatalf("abaixo do orçamento: %+v", d)
	}
}

func TestMedirUsoSomaPreambuloEMensagens(t *testing.T) {
	// O preâmbulo NÃO pode ser esquecido: ele é contexto que ocupa janela.
	sem := MedirUsoDoContexto("", []api.Message{{Role: "user", Content: strings.Repeat("x", 400)}})
	com := MedirUsoDoContexto(strings.Repeat("p", 400), []api.Message{{Role: "user", Content: strings.Repeat("x", 400)}})
	if com.Tokens <= sem.Tokens {
		t.Fatalf("preâmbulo não contou: sem=%d com=%d", sem.Tokens, com.Tokens)
	}
	vazio := MedirUsoDoContexto("", nil)
	if vazio.Tokens != 0 || vazio.Exata {
		t.Fatalf("vazio = %+v", vazio)
	}
}

func TestDecisaoRegistadaEDeclaradaQuandoAusente(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	// AUSÊNCIA não pode ser lida como "dentro do orçamento".
	if _, ok := DecisaoDeOrcamentoRegistada(c); ok {
		t.Fatal("devolveu decisão sem nada registado")
	}
	// E não pode explodir sem contexto.
	if _, ok := DecisaoDeOrcamentoRegistada(nil); ok {
		t.Fatal("devolveu decisão com contexto nil")
	}
	RegistarDecisaoDeOrcamento(nil, DecisaoDeOrcamento{}) // não pode entrar em pânico
	d := DecisaoDeOrcamento{Compactar: true, Motivo: "uso 900 (estimado) acima de 800 disponíveis"}
	RegistarDecisaoDeOrcamento(c, d)
	volta, ok := DecisaoDeOrcamentoRegistada(c)
	if !ok || !volta.Compactar || volta.Motivo != d.Motivo {
		t.Fatalf("decisão registada não voltou: %+v ok=%v", volta, ok)
	}
}
