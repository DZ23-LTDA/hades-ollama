package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

// Teste PONTA A PONTA da compactação no nível do pacote: planejar -> digerir -> gravar
// bruto -> substituir, e depois RECUPERAR o material a partir do histórico bruto.
// É o mais próximo da secção 10 que se consegue sem o funil HTTP.

func conversaLonga(n int) []api.Message {
	mensagens := make([]api.Message, 0, n)
	for i := range n {
		papel := "user"
		if i%2 == 1 {
			papel = "assistant"
		}
		mensagens = append(mensagens, api.Message{Role: papel, Content: "mensagem numero " + itoaCompactacao(i)})
	}
	return mensagens
}

func TestFluxoCompactaGravaERecuperaOMaterial(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "hist.jsonl")
	historico, err := AbrirHistoricoBruto(caminho)
	if err != nil {
		t.Fatal(err)
	}
	originais := conversaLonga(10)

	res, mensagens, err := FluxoDaCompactacao(originais, 4, TetoDeDigestPadrao, historico, "proj-1", "missao-7")
	if err != nil {
		t.Fatalf("fluxo: %v", err)
	}
	if !res.Compactado {
		t.Fatalf("não compactou uma conversa de 10 com janela 4: %+v", res)
	}
	if len(mensagens) != 1+4 {
		t.Fatalf("mensagens devolvidas = %d, esperado 1 resumo + 4 intactas", len(mensagens))
	}
	if !strings.Contains(mensagens[0].Content, MarcadorDeResumo) {
		t.Fatal("a primeira mensagem não está marcada como resumo")
	}
	// AS ÚLTIMAS 4 SEGUEM INTACTAS, com o conteúdo original.
	for i := range 4 {
		esperado := originais[len(originais)-4+i].Content
		if mensagens[1+i].Content != esperado {
			t.Fatalf("mensagem intacta %d: %q != %q", i, mensagens[1+i].Content, esperado)
		}
	}

	// RECUPERAÇÃO (§2: "reancorar periodicamente nas fontes brutas e permitir
	// reconstrução"): o material coberto tem de poder ser lido do histórico.
	historico.Fechar()
	reaberto, err := AbrirHistoricoBruto(caminho)
	if err != nil {
		t.Fatalf("o histórico não sobreviveu: %v", err)
	}
	defer reaberto.Fechar()
	registos, err := reaberto.LerTudo()
	if err != nil {
		t.Fatal(err)
	}
	if len(registos) != 6 {
		t.Fatalf("o bruto tem %d registos; a faixa coberta era de 6", len(registos))
	}
	if err := VerificarCadeia(registos); err != nil {
		t.Fatalf("cadeia do bruto quebrada: %v", err)
	}
	// O CONTEÚDO COBERTO É RECUPERÁVEL, mensagem a mensagem, e com o escopo.
	for i, reg := range registos {
		if reg.Conteudo != originais[i].Content {
			t.Fatalf("registo %d: %q != %q (o material coberto não é recuperável)", i, reg.Conteudo, originais[i].Content)
		}
		if reg.ProjetoID != "proj-1" || reg.MissaoID != "missao-7" {
			t.Fatalf("registo %d sem escopo: %+v", i, reg)
		}
	}
	// E o resumo enviado ao modelo referencia o TOPO DA CADEIA do bruto.
	if !strings.Contains(mensagens[0].Content, res.UltimoHash) {
		t.Fatal("o resumo não referencia o bruto que cobre")
	}
}

func TestFluxoMantemAConversaOriginalQuandoNadaCabe(t *testing.T) {
	historico, err := AbrirHistoricoBruto(filepath.Join(t.TempDir(), "hist.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer historico.Fechar()
	originais := conversaLonga(10)
	// Teto minúsculo: nada do digest cabe.
	res, mensagens, err := FluxoDaCompactacao(originais, 4, 5, historico, "p", "m")
	if err != nil {
		t.Fatalf("não devia ser erro, é uma degradação declarada: %v", err)
	}
	if res.Compactado {
		t.Fatal("declarou compactado tendo mantido a conversa inteira")
	}
	if len(mensagens) != len(originais) {
		t.Fatalf("devolveu %d mensagens em vez das %d originais", len(mensagens), len(originais))
	}
	if !strings.Contains(res.Motivo, "nada do resumo coube") {
		t.Fatalf("o motivo não declara a degradação: %q", res.Motivo)
	}
	// E NADA foi gravado no bruto: não se guarda material para um resumo que não existiu.
	registos, err := historico.LerTudo()
	if err != nil {
		t.Fatal(err)
	}
	if len(registos) != 0 {
		t.Fatalf("gravou %d registos sem ter compactado", len(registos))
	}
}

func TestFluxoEntregaOriginaisQuandoAGravacaoFalha(t *testing.T) {
	historico, err := AbrirHistoricoBruto(filepath.Join(t.TempDir(), "hist.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	historico.Fechar() // fechado: gravar falha
	originais := conversaLonga(10)
	res, mensagens, err := FluxoDaCompactacao(originais, 4, TetoDeDigestPadrao, historico, "p", "m")
	if err == nil {
		t.Fatal("compactou com o histórico fechado")
	}
	if res.Compactado {
		t.Fatal("declarou compactado apesar da falha")
	}
	// A CONVERSÃO ORIGINAL SOBREVIVE: é isto que o §2 exige em erro/interrupção.
	if len(mensagens) != len(originais) {
		t.Fatalf("o chamador não ficou com as originais: %d != %d", len(mensagens), len(originais))
	}
	for i := range originais {
		if mensagens[i].Content != originais[i].Content {
			t.Fatalf("mensagem %d alterada apesar da falha", i)
		}
	}
}
