package agent

import (
	"sort"
	"time"
)

// DESPERTAR COALESCIDO (E-633).
//
// A LACUNA (Paperclip: «DB-backed wakeup queue with coalescing», ✅ lá / ausente cá). A auditoria
// do E-632 mostrou que a lacuna é de COMPOSIÇÃO: o Hades já tem fila durável com lease e *claim*
// idempotente, agendador de supervisor e heartbeat de dispositivo — o que falta é a SEMÂNTICA
// «vários sinais pendentes para o mesmo agente ⇒ UM despertar». *Não se cria uma fila nova ao
// lado da existente: seria o segundo sistema de tarefas que os complementos proíbem.*
//
// PORQUE ISTO É UMA FUNÇÃO PURA, E NÃO A LIGAÇÃO À FILA: é aqui que a regra pode ser PROVADA sem
// infraestrutura (a ligação à fila é a parte que pode falhar por razões que uma unidade pura não
// vê). A garantia vai para onde é testável — mesma decisão do E-610 e do E-627.

// JanelaDeCoalescencia é o intervalo dentro do qual sinais do MESMO agente viram um só despertar.
//
// Curta de propósito: o objectivo é não acordar o agente cinco vezes porque chegaram cinco
// mensagens, e não adiar trabalho. Um valor alto transformaria coalescência em LATÊNCIA.
const JanelaDeCoalescencia = 5 * time.Second

// SinalDeDespertar é UM motivo para acordar um agente. Os motivos vêm do que já existe: tarefa
// atribuída, mensagem de seguimento, rotina agendada.
type SinalDeDespertar struct {
	AgenteID string
	MissaoID string
	Motivo   string
	Quando   time.Time
}

// Despertar é UM acordar, com TODAS as razões que o provocaram.
//
// `Motivos` leva as razões DISTINTAS em ordem de chegada, e `Sinais` leva a contagem total: se a
// mesma razão chegou 20 vezes, o agente vê-a uma vez e vê que foram 20 — *perder a contagem seria
// esconder que houve repetição.*
type Despertar struct {
	AgenteID string
	Motivos  []string
	// Missoes leva as missões DISTINTAS que provocaram o despertar. Sem isto, o despertar diria
	// «acorda por causa de aprovação» sem dizer de QUÊ — e quem acorda não saberia o que abrir.
	Missoes  []string
	Sinais   int
	Primeiro time.Time
	Ultimo   time.Time
}

// DespertaresCoalescidos agrupa sinais por agente e, dentro de cada agente, por janela fixa
// contada a partir do PRIMEIRO sinal do grupo.
//
// A janela é FIXA (e não deslizante) de propósito: com janela deslizante, uma sequência contínua
// de sinais nunca fecharia o grupo, e o agente ficaria sem ser acordado enquanto o trabalho
// chegasse — *coalescer não pode virar adiamento infinito*.
//
// Nada é descartado: um sinal sem agente identificado produz um despertar com `AgenteID` vazio,
// que a superfície vê e pode recusar. *Ignorá-lo em silêncio seria perder trabalho sem o dizer.*
func DespertaresCoalescidos(sinais []SinalDeDespertar) []Despertar {
	if len(sinais) == 0 {
		return nil
	}
	porAgente := make(map[string][]SinalDeDespertar)
	ordem := make([]string, 0, len(sinais))
	for _, sinal := range sinais {
		if _, visto := porAgente[sinal.AgenteID]; !visto {
			ordem = append(ordem, sinal.AgenteID)
		}
		porAgente[sinal.AgenteID] = append(porAgente[sinal.AgenteID], sinal)
	}
	sort.Strings(ordem)

	despertares := make([]Despertar, 0, len(ordem))
	for _, agenteID := range ordem {
		doAgente := porAgente[agenteID]
		sort.SliceStable(doAgente, func(i, j int) bool { return doAgente[i].Quando.Before(doAgente[j].Quando) })

		actual := Despertar{AgenteID: agenteID, Primeiro: doAgente[0].Quando, Ultimo: doAgente[0].Quando}
		distintos := make(map[string]bool)
		missõesVistas := make(map[string]bool)
		acrescentar := func(sinal SinalDeDespertar) {
			actual.Sinais++
			if sinal.Quando.After(actual.Ultimo) {
				actual.Ultimo = sinal.Quando
			}
			if !distintos[sinal.Motivo] {
				distintos[sinal.Motivo] = true
				actual.Motivos = append(actual.Motivos, sinal.Motivo)
			}
			if sinal.MissaoID != "" && !missõesVistas[sinal.MissaoID] {
				missõesVistas[sinal.MissaoID] = true
				actual.Missoes = append(actual.Missoes, sinal.MissaoID)
			}
		}
		acrescentar(doAgente[0])
		for _, sinal := range doAgente[1:] {
			if sinal.Quando.Sub(actual.Primeiro) > JanelaDeCoalescencia {
				despertares = append(despertares, actual)
				actual = Despertar{AgenteID: agenteID, Primeiro: sinal.Quando, Ultimo: sinal.Quando}
				distintos = make(map[string]bool)
				missõesVistas = make(map[string]bool)
			}
			acrescentar(sinal)
		}
		despertares = append(despertares, actual)
	}
	return despertares
}
