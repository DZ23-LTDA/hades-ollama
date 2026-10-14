package agent

import (
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

// Testes do resumo determinístico da faixa (§2): fiel, ordenado e declarando o corte.

func TestDigestCopiaOMaterialComPapelEOrdem(t *testing.T) {
	mensagens := []api.Message{
		{Role: "user", Content: "preciso do relatório de vendas"},
		{Role: "assistant", Content: "vou consultar o estoque"},
		{Role: "tool", Content: "42 unidades"},
	}
	texto, cortado := ResumirPorDigest(mensagens, TetoDeDigestPadrao)
	if cortado {
		t.Fatal("não devia cortar uma faixa pequena")
	}
	// FIDELIDADE: cada conteúdo coberto tem de estar no resumo. Se algum desaparecer, o
	// resumo passou a ser uma versão da conversa que mente por omissão.
	for _, m := range mensagens {
		if !strings.Contains(texto, m.Content) {
			t.Fatalf("o resumo perdeu a mensagem %q:\n%s", m.Content, texto)
		}
	}
	// ORDEM preservada, e o papel declarado em cada linha.
	iUser := strings.Index(texto, "preciso do relatório")
	iAssist := strings.Index(texto, "vou consultar o estoque")
	iTool := strings.Index(texto, "42 unidades")
	if !(iUser < iAssist && iAssist < iTool) {
		t.Fatalf("a ordem foi alterada: %d %d %d", iUser, iAssist, iTool)
	}
	for _, papel := range []string{"[user 1]", "[assistant 2]", "[tool 3]"} {
		if !strings.Contains(texto, papel) {
			t.Fatalf("o resumo não declara %s: %s", papel, texto)
		}
	}
}

func TestDigestDeclaraOCorteENaoCortaAMeioDeUmaMensagem(t *testing.T) {
	longa := strings.Repeat("a", 200)
	mensagens := []api.Message{
		{Role: "user", Content: "curta"},
		{Role: "assistant", Content: longa},
		{Role: "user", Content: "outra curta"},
	}
	// Teto que cabe a primeira e não a segunda: o corte tem de ser por mensagem INTEIRA.
	texto, cortado := ResumirPorDigest(mensagens, 60)
	if !cortado {
		t.Fatal("não declarou o corte")
	}
	if !strings.Contains(texto, "curta") {
		t.Fatalf("perdeu a mensagem que cabia: %q", texto)
	}
	if strings.Contains(texto, longa) || strings.Contains(texto, "aaaa") {
		t.Fatal("cortou a mensagem a meio: um resumo truncado a meio faz o modelo completar o que falta")
	}
	if strings.Contains(texto, "outra curta") {
		t.Fatal("incluiu uma mensagem posterior ao corte")
	}
}

func TestDigestVazioQuandoNadaCabeESemMensagens(t *testing.T) {
	// Nada cabe: devolve vazio E cortado=true. O chamador (E-573) recusa resumo vazio e
	// mantém a conversa inteira — que é a decisão certa.
	texto, cortado := ResumirPorDigest([]api.Message{{Role: "user", Content: strings.Repeat("x", 500)}}, 10)
	if texto != "" || !cortado {
		t.Fatalf("esperava vazio+cortado, veio %q cortado=%v", texto, cortado)
	}
	vazio, cortadoVazio := ResumirPorDigest(nil, 100)
	if vazio != "" || cortadoVazio {
		t.Fatalf("sem mensagens não há corte a declarar: %q %v", vazio, cortadoVazio)
	}
	// Teto inválido cai no padrão, e não em "corta tudo".
	texto, cortado = ResumirPorDigest([]api.Message{{Role: "user", Content: "ok"}}, 0)
	if cortado || !strings.Contains(texto, "ok") {
		t.Fatalf("teto 0 não caiu no padrão: %q cortado=%v", texto, cortado)
	}
	// Papel vazio não vira linha sem rótulo.
	texto, _ = ResumirPorDigest([]api.Message{{Content: "sem papel"}}, 100)
	if !strings.Contains(texto, "[desconhecido 1]") {
		t.Fatalf("papel vazio não foi declarado: %q", texto)
	}
}
