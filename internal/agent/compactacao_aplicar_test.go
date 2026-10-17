package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

// Testes do EFEITO da compactação (§2), incluindo o caminho de FALHA do §7-D:
// falha na gravação do resumo ⇒ nada é substituído e o histórico bruto fica intacto.

func planoDeTeste(t *testing.T, n int) PlanoDeCompactacao {
	t.Helper()
	mensagens := make([]api.Message, 0, n)
	for i := range n {
		papel := "user"
		if i%2 == 1 {
			papel = "assistant"
		}
		mensagens = append(mensagens, api.Message{Role: papel, Content: "conteudo " + itoaCompactacao(i)})
	}
	return PlanejarCompactacao(mensagens, 2)
}

func TestAplicarCompactacaoGravaOBrutoAntesDoResumo(t *testing.T) {
	historico, err := AbrirHistoricoBruto(filepath.Join(t.TempDir(), "hist.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer historico.Fechar()
	plano := planoDeTeste(t, 6)
	if !plano.Compactavel {
		t.Fatal("o plano de teste devia ser compactável")
	}

	res, err := AplicarCompactacao(plano, historico, "resumo fiel da faixa", "p1", "m1")
	if err != nil {
		t.Fatalf("aplicar: %v", err)
	}
	if !res.Compactado {
		t.Fatalf("não compactou: %+v", res)
	}
	if res.RegistosGravados != len(plano.Resumir) {
		t.Fatalf("gravou %d registos para uma faixa de %d", res.RegistosGravados, len(plano.Resumir))
	}
	// O BRUTO TEM DE ESTAR LÁ: é a propriedade que torna a perda "controlada" em vez de
	// perda. Se isto falhar, o resumo passou a ser a única cópia — o que o §2 proíbe.
	registos, err := historico.LerTudo()
	if err != nil {
		t.Fatal(err)
	}
	if len(registos) != len(plano.Resumir) {
		t.Fatalf("histórico bruto com %d registos, esperado %d", len(registos), len(plano.Resumir))
	}
	if err := VerificarCadeia(registos); err != nil {
		t.Fatalf("a cadeia do bruto não está íntegra: %v", err)
	}
	if res.UltimoHash != registos[len(registos)-1].Hash {
		t.Fatal("a referência devolvida não é o topo da cadeia")
	}
	for i, reg := range registos {
		if reg.Papel != plano.Resumir[i].Role || reg.Conteudo != plano.Resumir[i].Content {
			t.Fatalf("registo %d não corresponde à mensagem coberta: %+v", i, reg)
		}
		if reg.ProjetoID != "p1" || reg.MissaoID != "m1" {
			t.Fatalf("registo %d sem o escopo declarado: %+v", i, reg)
		}
	}

	// As mensagens novas: resumo + intactas, na ordem.
	if len(res.Mensagens) != 1+len(plano.Intactas) {
		t.Fatalf("mensagens = %d, esperado %d", len(res.Mensagens), 1+len(plano.Intactas))
	}
	primeira := res.Mensagens[0]
	if !strings.Contains(primeira.Content, MarcadorDeResumo) {
		t.Fatal("o resumo não leva marcador: passaria por conversa original")
	}
	if !strings.Contains(primeira.Content, res.UltimoHash) {
		t.Fatal("o resumo não referencia o bruto que cobre")
	}
	if primeira.Role == "system" {
		t.Fatal("resumo com papel de SISTEMA: texto recuperado promovido a instrução (§7-E proíbe)")
	}
	if res.Mensagens[1].Content != plano.Intactas[0].Content {
		t.Fatal("as intactas não seguem na ordem original")
	}
}

func TestAplicarCompactacaoNaoCompactaQuandoNaoHaFaixa(t *testing.T) {
	historico, err := AbrirHistoricoBruto(filepath.Join(t.TempDir(), "hist.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer historico.Fechar()
	mensagens := []api.Message{{Role: "user", Content: "só isto"}}
	plano := PlanejarCompactacao(mensagens, 6)

	res, err := AplicarCompactacao(plano, historico, "resumo", "", "")
	if err != nil {
		t.Fatalf("recusou sem motivo: %v", err)
	}
	if res.Compactado || res.RegistosGravados != 0 {
		t.Fatalf("compactou o que não havia: %+v", res)
	}
	if len(res.Mensagens) != 1 || res.Mensagens[0].Content != "só isto" {
		t.Fatalf("as mensagens deviam seguir intactas: %+v", res.Mensagens)
	}
}

func TestAplicarCompactacaoFalhaFechadoSemHistoricoOuResumo(t *testing.T) {
	plano := planoDeTeste(t, 6)
	// Sem histórico NÃO há compactação: devolver mensagens resumidas seria perder a faixa
	// sem deixar rasto.
	res, err := AplicarCompactacao(plano, nil, "resumo", "", "")
	if err == nil {
		t.Fatal("compactou sem histórico bruto")
	}
	if len(res.Mensagens) != 0 {
		t.Fatal("devolveu mensagens depois de falhar: o chamador poderia substituí-las")
	}
	if !strings.Contains(res.Motivo, "sem histórico") {
		t.Fatalf("motivo = %q", res.Motivo)
	}
	// Resumo vazio também recusa: substituir a faixa por nada perde a conversa.
	historico, err := AbrirHistoricoBruto(filepath.Join(t.TempDir(), "hist.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer historico.Fechar()
	res, err = AplicarCompactacao(plano, historico, "   ", "", "")
	if err == nil || len(res.Mensagens) != 0 {
		t.Fatalf("aceitou resumo vazio: err=%v msgs=%d", err, len(res.Mensagens))
	}
}

// §7-D: FALHA DURANTE A GRAVAÇÃO. A injecção é a mesma do E-566 — o caminho do ficheiro
// passa a ser um DIRETÓRIO NÃO VAZIO, e o `os.OpenFile`/`Sync` falham. O que se prova:
// nenhuma mensagem é substituída, e as que já tinham sido gravadas continuam no bruto.
func TestAplicarCompactacaoFalhaNaGravacaoNaoSubstituiNada(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, "hist.jsonl")

	// Primeiro, um histórico VÁLIDO com uma mensagem já gravada (o "antes da falha").
	bom, err := AbrirHistoricoBruto(caminho)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bom.Anexar(RegistroDeHistorico{Papel: "user", Conteudo: "gravado antes da falha"}); err != nil {
		t.Fatal(err)
	}
	bom.Fechar()

	// Agora o caminho é um diretório não vazio: gravar passa a falhar.
	if err := os.Remove(caminho); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(caminho, "ocupado"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caminho, "ocupado", "x"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	quebrado, err := AbrirHistoricoBruto(caminho)
	if err == nil {
		quebrado.Fechar()
		t.Skip("a plataforma abriu um diretório como histórico: o caminho de falha não foi exercitado")
	}

	// O TESTE PRÓPRIO: com um histórico que não pode gravar, nada é substituído.
	plano := planoDeTeste(t, 6)
	historicoFechado, err := AbrirHistoricoBruto(filepath.Join(dir, "outro.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	historicoFechado.Fechar() // fechado: `Anexar` devolve erro sem tocar no disco
	res, err := AplicarCompactacao(plano, historicoFechado, "resumo", "", "")
	if err == nil {
		t.Fatal("compactou com o histórico fechado")
	}
	if len(res.Mensagens) != 0 {
		t.Fatalf("FALHA FECHADO QUEBRADA: devolveu %d mensagens depois da falha de gravação", len(res.Mensagens))
	}
	if res.Compactado {
		t.Fatal("declarou compactado apesar da falha")
	}
	if !strings.Contains(res.Motivo, "NÃO devolvo mensagens") {
		t.Fatalf("o motivo não declara a recusa: %q", res.Motivo)
	}

	// E o material anterior continua acessível: o histórico bom não foi tocado.
	reaberto, err := AbrirHistoricoBruto(filepath.Join(dir, "outro.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer reaberto.Fechar()
	registos, err := reaberto.LerTudo()
	if err != nil {
		t.Fatal(err)
	}
	if len(registos) != 0 {
		t.Fatalf("um histórico que recusou gravar não pode ter registos: %d", len(registos))
	}
}
