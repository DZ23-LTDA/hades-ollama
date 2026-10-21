package agent

import (
	"testing"
	"time"
)

func sinal(agente, motivo string, segundo int) SinalDeDespertar {
	return SinalDeDespertar{
		AgenteID: agente,
		Motivo:   motivo,
		Quando:   time.Date(2026, 10, 14, 12, 0, segundo, 0, time.UTC),
	}
}

// A GARANTIA CENTRAL: vários sinais do MESMO agente dentro da janela viram UM despertar — e esse
// despertar carrega TODAS as razões e a contagem. *Acordar cinco vezes porque chegaram cinco
// mensagens é desperdício; perder a contagem seria esconder que houve repetição.*
func TestSinaisDoMesmoAgenteNaJanelaViramUmDespertar(t *testing.T) {
	despertares := DespertaresCoalescidos([]SinalDeDespertar{
		sinal("agente-a", "tarefa atribuida", 0),
		sinal("agente-a", "mensagem de seguimento", 2),
		sinal("agente-a", "tarefa atribuida", 4),
	})
	if len(despertares) != 1 {
		t.Fatalf("três sinais na janela deviam dar UM despertar; resultado: %d", len(despertares))
	}
	unico := despertares[0]
	if unico.Sinais != 3 {
		t.Fatalf("a contagem de sinais tem de ser preservada: %d", unico.Sinais)
	}
	if len(unico.Motivos) != 2 {
		t.Fatalf("as razões deviam ser DISTINTAS (2), não repetidas: %v", unico.Motivos)
	}
	if unico.Motivos[0] != "tarefa atribuida" || unico.Motivos[1] != "mensagem de seguimento" {
		t.Fatalf("as razões deviam vir em ordem de chegada: %v", unico.Motivos)
	}
	if !unico.Primeiro.Equal(sinal("x", "y", 0).Quando) || !unico.Ultimo.Equal(sinal("x", "y", 4).Quando) {
		t.Fatalf("a janela devia ir do primeiro ao último sinal: %v..%v", unico.Primeiro, unico.Ultimo)
	}
}

// FORA DA JANELA NÃO SE FUNDE: coalescer não pode virar adiamento.
func TestSinaisForaDaJanelaNaoSeFundem(t *testing.T) {
	despertares := DespertaresCoalescidos([]SinalDeDespertar{
		sinal("agente-a", "tarefa", 0),
		sinal("agente-a", "tarefa", int(JanelaDeCoalescencia/time.Second)+1),
	})
	if len(despertares) != 2 {
		t.Fatalf("um sinal fora da janela devia dar DOIS despertares, deu %d", len(despertares))
	}
}

// AGENTES DIFERENTES NUNCA SE MISTURAM — o despertar é por agente, não por lote.
func TestAgentesDiferentesNaoSeMisturam(t *testing.T) {
	despertares := DespertaresCoalescidos([]SinalDeDespertar{
		sinal("agente-b", "tarefa", 0),
		sinal("agente-a", "mensagem", 0),
	})
	if len(despertares) != 2 {
		t.Fatalf("dois agentes deviam dar dois despertares: %d", len(despertares))
	}
	// Saída determinística e ordenada por agente.
	if despertares[0].AgenteID != "agente-a" || despertares[1].AgenteID != "agente-b" {
		t.Fatalf("a saída devia estar ordenada por agente: %v, %v", despertares[0].AgenteID, despertares[1].AgenteID)
	}
}

// NADA É DESCARTADO EM SILÊNCIO: um sinal sem agente produz um despertar visível com agente
// vazio, que a superfície pode recusar. Ignorá-lo seria perder trabalho sem o dizer.
func TestSinalSemAgenteNaoEhDescartado(t *testing.T) {
	despertares := DespertaresCoalescidos([]SinalDeDespertar{sinal("", "tarefa orfa", 0)})
	if len(despertares) != 1 || despertares[0].AgenteID != "" || despertares[0].Sinais != 1 {
		t.Fatalf("um sinal sem agente tem de aparecer, não desaparecer: %+v", despertares)
	}
}

// Sem sinais não há despertares — e não se inventa um.
func TestSemSinaisNaoHaDespertares(t *testing.T) {
	if despertares := DespertaresCoalescidos(nil); len(despertares) != 0 {
		t.Fatalf("sem sinais deviam sair zero despertares: %+v", despertares)
	}
}

// A JANELA É FIXA A PARTIR DO PRIMEIRO SINAL, E NÃO ESTICA — e é isto que impede a coalescência de
// virar adiamento: uma sequência contínua de sinais com intervalos curtos NÃO mantém o agente à
// espera indefinidamente; passada a janela, ele é acordado com o que houver.
//
// (Primeira versão deste teste exigia que três sinais espaçados por menos que a janela, mas
// somando mais que ela, ficassem num só grupo. Isso seria uma janela DESLIZANTE — o oposto do que
// está implementado e documentado. O teste é que estava errado, e ficou registado.)
func TestJanelaFixaNaoEstica(t *testing.T) {
	passo := int(JanelaDeCoalescencia/time.Second) - 2
	if passo < 1 {
		t.Fatal("a janela é demasiado curta para este teste")
	}
	despertares := DespertaresCoalescidos([]SinalDeDespertar{
		sinal("agente-a", "tarefa", 0),
		sinal("agente-a", "tarefa", passo),
		sinal("agente-a", "tarefa", passo*2),
	})
	if len(despertares) != 2 {
		t.Fatalf("o terceiro sinal cai fora da janela do primeiro e devia abrir um grupo novo: %d", len(despertares))
	}
	if despertares[0].Sinais != 2 || despertares[1].Sinais != 1 {
		t.Fatalf("contagens erradas: %d e %d", despertares[0].Sinais, despertares[1].Sinais)
	}
}

// E O LIMITE É INCLUSIVO: um sinal EXACTAMENTE na janela ainda pertence ao grupo.
func TestSinalNoLimiteDaJanelaPertenceAoGrupo(t *testing.T) {
	limite := int(JanelaDeCoalescencia / time.Second)
	despertares := DespertaresCoalescidos([]SinalDeDespertar{
		sinal("agente-a", "tarefa", 0),
		sinal("agente-a", "mensagem", limite),
	})
	if len(despertares) != 1 || despertares[0].Sinais != 2 {
		t.Fatalf("um sinal no limite da janela pertence ao grupo: %+v", despertares)
	}
}
