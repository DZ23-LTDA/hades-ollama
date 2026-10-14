package agent

import (
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

// Testes do plano de compactação (§2 do complemento universal de memória).
// Função pura: nenhum modelo, rede, disco ou relógio.

func conversa(pares ...string) []api.Message {
	mensagens := make([]api.Message, 0, len(pares))
	for i, papel := range pares {
		mensagens = append(mensagens, api.Message{Role: papel, Content: "m" + string(rune('0'+i))})
	}
	return mensagens
}

func TestPlanoNaoCompactaQuandoNaoHaFaixaAntiga(t *testing.T) {
	// A PROPRIEDADE QUE EVITA A MENTIRA: sem faixa antiga, `Compactavel=false` e nada
	// muda. Um plano que dissesse "compactável" aqui levaria a substituir mensagens
	// recentes por um resumo delas mesmas.
	curta := conversa("user", "assistant", "user")
	p := PlanejarCompactacao(curta, JanelaRecentePadrao)
	if p.Compactavel {
		t.Fatalf("declarou compactável uma conversa que cabe na janela: %+v", p)
	}
	if len(p.Resumir) != 0 {
		t.Fatalf("reservou faixa a resumir sem haver faixa: %d", len(p.Resumir))
	}
	if len(p.Intactas) != len(curta) {
		t.Fatalf("não preservou tudo: intactas=%d de %d", len(p.Intactas), len(curta))
	}
	if !strings.Contains(p.Motivo, "nada a compactar") {
		t.Fatalf("motivo não declara a ausência de trabalho: %q", p.Motivo)
	}
}

func TestPlanoDeixaAsRecentesIntactasEDeclaraOsNumeros(t *testing.T) {
	longa := conversa("user", "assistant", "user", "assistant", "user", "assistant", "user", "assistant", "user")
	p := PlanejarCompactacao(longa, 4)
	if !p.Compactavel {
		t.Fatalf("não compactou uma conversa longa: %+v", p)
	}
	if len(p.Intactas) != 4 {
		t.Fatalf("intactas = %d, esperado 4", len(p.Intactas))
	}
	if len(p.Resumir)+len(p.Intactas) != len(longa) {
		t.Fatalf("a divisão perdeu mensagens: %d+%d != %d", len(p.Resumir), len(p.Intactas), len(longa))
	}
	if !strings.Contains(p.Motivo, "9") || !strings.Contains(p.Motivo, "4") {
		t.Fatalf("o motivo não traz os números da divisão: %q", p.Motivo)
	}
	// A ORDEM tem de ser preservada: o resumo é o prefixo, as intactas são o sufixo.
	if p.Resumir[0].Content != longa[0].Content {
		t.Fatal("a faixa a resumir não começa no início")
	}
	if p.Intactas[len(p.Intactas)-1].Content != longa[len(longa)-1].Content {
		t.Fatal("as intactas não terminam na última mensagem")
	}
}

func TestPlanoNuncaResumeAUltimaInstrucaoDoUtilizador(t *testing.T) {
	// O CASO PERIGOSO: o pedido corrente seguido de mensagens de ferramenta. Uma janela
	// pequena deixaria a INSTRUÇÃO fora das intactas, e o modelo responderia a uma versão
	// resumida do próprio pedido.
	comFerramentas := conversa("user", "assistant", "user", "assistant", "user", "tool", "tool", "tool")
	p := PlanejarCompactacao(comFerramentas, 2)
	if !p.Compactavel {
		t.Fatal("devia compactar a faixa antiga")
	}
	for _, m := range p.Intactas {
		_ = m
	}
	// A última mensagem de papel "user" tem de estar nas intactas.
	indiceUltimoUser := -1
	for i, m := range comFerramentas {
		if strings.EqualFold(m.Role, "user") {
			indiceUltimoUser = i
		}
	}
	if indiceUltimoUser < 0 {
		t.Fatal("conversa de teste sem mensagem de utilizador")
	}
	alvo := comFerramentas[indiceUltimoUser].Content
	achou := false
	for _, m := range p.Intactas {
		if m.Content == alvo {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("a última instrução do utilizador (%q) caiu na faixa a resumir: resumir=%d intactas=%d",
			alvo, len(p.Resumir), len(p.Intactas))
	}
}

func TestPlanoAceitaJanelaInvalidaSemExplodir(t *testing.T) {
	longa := conversa("user", "assistant", "user", "assistant", "user", "assistant", "user")
	for _, janela := range []int{0, -1, -100} {
		p := PlanejarCompactacao(longa, janela)
		if len(p.Resumir)+len(p.Intactas) != len(longa) {
			t.Fatalf("janela %d perdeu mensagens", janela)
		}
		if len(p.Intactas) < JanelaRecentePadrao {
			t.Fatalf("janela %d não caiu no padrão: intactas=%d", janela, len(p.Intactas))
		}
	}
	vazia := PlanejarCompactacao(nil, 3)
	if vazia.Compactavel || len(vazia.Intactas) != 0 {
		t.Fatalf("conversa vazia devia dar plano vazio: %+v", vazia)
	}
}
