package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// E-536 (Estagio 46 - Hades Companion): **estados derivados de EVENTOS VERIFICADOS**.
//
// ***A REGRA DO COMPLEMENTO, E O QUE ELA PROIBE.***
//
// O documento e' explicito: *«Nao utilizar animacoes ou respostas falsas para simular trabalho»*, e o ADR-003 ja'
// decidira que o Companion vem **depois** da engine e **derivado de eventos verificados** - *«nunca de estado
// inventado»*.
//
// ⇒ ***Este ficheiro e' a fronteira entre um mascote e um leitor de factos.*** Ele **nao** desenha, **nao** anima e
// **nao** decide o que aconteceu: recebe eventos **que ja' aconteceram** e devolve um estado **com a razao**. *Se nao
// houver eventos, o estado e' `ausente` - e `ausente` e' a resposta, nao uma falha.*
//
// ***AS QUATRO REGRAS, E PORQUE CADA UMA EXISTE.***
//
//  1. **Sem eventos => `ausente`.** *E' o «inventar agente trabalhando» no seu caso mais puro: um mascote a rodar sem
//     nada a acontecer e' a mentira mais facil de escrever, e a mais dificil de detectar.*
//  2. **Evento DESCONHECIDO => RECUSA** (`ErrCompanionEventoDesconhecido`). *Nao se deriva estado do que nao se
//     conhece; e recusar e' melhor do que mapear por semelhanca - «parece um evento de sucesso» nao e' um evento de
//     sucesso.*
//  3. **Evento de OUTRA missao => ignorado.** *O Companion de uma missao nao fala de outra. E' o isolamento que os
//     `E-533`/`E-535` tambem respeitam, no seu eixo.*
//  4. **`concluido` e `falhou` SO' com os seus eventos.** *Nunca por omissao, nunca por tempo, nunca por ausencia de
//     sinal - *um silencio nao e' um sucesso.*
//
// ***O QUE ESTE FICHEIRO **NAO** FAZ.*** Nao le' ficheiros, nao consulta a missao, nao tem estado global: *a lista de
// eventos entra por parametro.* ⇒ **e' testavel sem infraestrutura**, e um Companion que nao consegue provar o que diz
// nao serve para nada.

var (
	// ErrCompanionEventoDesconhecido recusa derivar estado de um evento que nao pertence ao vocabulario verificado.
	ErrCompanionEventoDesconhecido = errors.New("companion event is not a verified event")
	// ErrCompanionMissaoAusente recusa derivar estado sem saber de que missao se fala.
	ErrCompanionMissaoAusente = errors.New("companion state requires a mission id")
)

// EventosVerificadosDeMissao e' o **vocabulario fechado**. *Um evento fora desta lista nao muda o estado: faz a
// derivacao FALHAR* - e' a regra 2, e e' o que impede o vocabulario de crescer por engano, no meio de um `switch`.
var EventosVerificadosDeMissao = []string{
	"mission.created",
	"mission.planning",
	"mission.step.started",
	"mission.step.finished",
	"mission.approval.requested",
	"mission.approval.granted",
	"mission.approval.denied",
	"mission.blocked",
	"mission.repair_exhausted",
	"mission.completed",
	"mission.failed",
	"mission.cancelled",
}

// CompanionEventoVerificado e' um evento que **aconteceu**, com o minimo para ser verificavel.
//
// *`Quando` existe para que o estado seja ordenavel no tempo - e para que um evento futuro nao seja tratado como
// passado.* `Detalhe` e' texto ja' sanitizado a montante: *este ficheiro nao inventa detalhe, so' o transporta.*
type CompanionEventoVerificado struct {
	Tipo     string    `json:"tipo"`
	MissaoID string    `json:"missao_id"`
	PassoID  string    `json:"passo_id,omitempty"`
	Quando   time.Time `json:"quando"`
	Detalhe  string    `json:"detalhe,omitempty"`
}

// CompanionEstado e' o que o Companion **pode** dizer. *O conjunto e' fechado de proposito: um estado novo tem de ser
// uma decisao, e nao uma string escrita num sitio qualquer.*
type CompanionEstado string

const (
	// CompanionAusente e' o estado SEM eventos. *Nao e' erro nem carregamento: e' ausencia de facto.*
	CompanionAusente CompanionEstado = "ausente"
	// CompanionAPlanejar: a missao foi criada e esta' a planear.
	CompanionAPlanejar CompanionEstado = "a_planear"
	// CompanionATrabalhar: **um passo comecou e ainda nao acabou** - o unico caso em que «a trabalhar» e' honesto.
	CompanionATrabalhar CompanionEstado = "a_trabalhar"
	// CompanionAAProvar: ha' uma aprovacao pedida e **nao resolvida**. *Aqui o Companion tem de parar de narrar e pedir
	// o humano - e' o unico estado em que a acao e' DELE.*
	CompanionAAProvar CompanionEstado = "a_aprovar"
	// CompanionBloqueado: bloqueio declarado. *Distinto de «a espera»: bloqueio tem nome.*
	CompanionBloqueado CompanionEstado = "bloqueado"
	// CompanionConcluido: SO' com `mission.completed`.
	CompanionConcluido CompanionEstado = "concluido"
	// CompanionFalhou: SO' com `mission.failed` ou `mission.repair_exhausted`.
	CompanionFalhou CompanionEstado = "falhou"
	// CompanionCancelado: SO' com `mission.cancelled`.
	CompanionCancelado CompanionEstado = "cancelado"
)

func eventoVerificado(tipo string) bool {
	for _, t := range EventosVerificadosDeMissao {
		if t == tipo {
			return true
		}
	}
	return false
}

// DerivarEstadoCompanion devolve o estado **e a razao**, a partir dos eventos verificados de UMA missao.
//
// *Devolve a razao junto do estado de proposito*: um estado sem procedencia e' indistinguivel de um estado inventado, e
// o complemento proibe exactamente isso. A razao nomeia o **evento** que a sustenta.
func DerivarEstadoCompanion(missaoID string, eventos []CompanionEventoVerificado) (CompanionEstado, string, error) {
	missaoID = strings.TrimSpace(missaoID)
	if missaoID == "" {
		return "", "", ErrCompanionMissaoAusente
	}

	// **(2) O vocabulario e' fechado, e um evento desconhecido RECUSA** - *antes de qualquer filtragem, para que um
	// evento invalido de outra missao tambem falhe, e nao seja silenciosamente ignorado.*
	for _, e := range eventos {
		if !eventoVerificado(e.Tipo) {
			return "", "", fmt.Errorf("%w: %q (missao %q)", ErrCompanionEventoDesconhecido, e.Tipo, e.MissaoID)
		}
	}

	// **(3) Isolamento por missao.**
	meus := make([]CompanionEventoVerificado, 0, len(eventos))
	for _, e := range eventos {
		if strings.TrimSpace(e.MissaoID) == missaoID {
			meus = append(meus, e)
		}
	}

	// **(1) Sem eventos => `ausente`.** *A regra mais importante deste ficheiro.*
	if len(meus) == 0 {
		return CompanionAusente, "sem eventos verificados desta missao", nil
	}

	// Ordem determinista: por tempo e, em empate, por tipo - *para que dois eventos no mesmo instante nao produzam
	// estados diferentes em execucoes diferentes.*
	sort.SliceStable(meus, func(i, j int) bool {
		if meus[i].Quando.Equal(meus[j].Quando) {
			return meus[i].Tipo < meus[j].Tipo
		}
		return meus[i].Quando.Before(meus[j].Quando)
	})

	// **(4) Estados terminais so' pelos seus eventos** - e o ULTIMO terminal visto decide, para que uma falha seguida
	// de conclusao (ou o inverso) nao fique ambigua.
	terminal := CompanionEstado("")
	razaoTerminal := ""
	passoAberto := ""
	aprovacaoPendente := ""
	bloqueado := ""

	for _, e := range meus {
		switch e.Tipo {
		case "mission.created":
			if terminal == "" {
				terminal, razaoTerminal = CompanionAPlanejar, e.Tipo
			}
		case "mission.planning":
			if terminal == "" {
				terminal, razaoTerminal = CompanionAPlanejar, e.Tipo
			}
		case "mission.step.started":
			passoAberto = e.PassoID
		case "mission.step.finished":
			if passoAberto == e.PassoID {
				passoAberto = ""
			}
		case "mission.approval.requested":
			aprovacaoPendente = e.PassoID
		case "mission.approval.granted", "mission.approval.denied":
			aprovacaoPendente = ""
		case "mission.blocked":
			bloqueado = e.Detalhe
			if bloqueado == "" {
				bloqueado = e.Tipo
			}
			terminal, razaoTerminal = CompanionBloqueado, e.Tipo
		case "mission.repair_exhausted", "mission.failed":
			passoAberto = ""
			aprovacaoPendente = ""
			terminal, razaoTerminal = CompanionFalhou, e.Tipo
		case "mission.completed":
			passoAberto = ""
			aprovacaoPendente = ""
			terminal, razaoTerminal = CompanionConcluido, e.Tipo
		case "mission.cancelled":
			passoAberto = ""
			aprovacaoPendente = ""
			terminal, razaoTerminal = CompanionCancelado, e.Tipo
		}
	}

	// Um estado terminal **fecha** a narrativa: *o Companion nao volta a dizer «a trabalhar» depois de saber o fim.*
	if terminal == CompanionConcluido || terminal == CompanionFalhou || terminal == CompanionCancelado {
		return terminal, razaoTerminal, nil
	}
	// Bloqueio e' terminal enquanto nao houver evento posterior que o levante - e *nenhum dos verificados o levanta
	// explicitamente*, logo ele mantem-se: **nao se inventa uma desbloqueio.**
	if terminal == CompanionBloqueado {
		return CompanionBloqueado, razaoTerminal, nil
	}
	// Aprovacao pendente **vence** «a trabalhar»: *enquanto o humano nao decidir, o sistema nao esta' a avancar - esta'
	// a espera dele. Dizer «a trabalhar» aqui seria a versao mais discreta da mentira.*
	if aprovacaoPendente != "" {
		return CompanionAAProvar, "mission.approval.requested (passo " + aprovacaoPendente + ")", nil
	}
	if passoAberto != "" {
		return CompanionATrabalhar, "mission.step.started (passo " + passoAberto + ")", nil
	}
	if terminal != "" {
		return terminal, razaoTerminal, nil
	}
	// *Inalcancavel com o vocabulario actual - e escrito para que, se deixar de ser, falhe visivelmente em vez de
	// devolver um estado vazio.*
	return CompanionAusente, "eventos verificados presentes mas sem estado aplicavel", nil
}

// CompanionPodeDizer diz se uma frase e' ADMISSIVEL para um estado. *E' a versao minima de «nao inventar»: o Companion
// so' pode afirmar trabalho quando ha' um passo aberto.*
//
// *Nao e' um filtro de texto sofisticado - e' uma guarda: se alguem ligar o texto a um estado fixo, isto recusa.*
func CompanionPodeDizer(estado CompanionEstado, frase string) bool {
	f := strings.ToLower(strings.TrimSpace(frase))
	if f == "" {
		return false
	}
	afirmaTrabalho := strings.Contains(f, "a trabalhar") || strings.Contains(f, "processando") ||
		strings.Contains(f, "a executar") || strings.Contains(f, "estou a fazer")
	if afirmaTrabalho && estado != CompanionATrabalhar {
		return false
	}
	afirmaSucesso := strings.Contains(f, "conclu") || strings.Contains(f, "terminado") || strings.Contains(f, "sucesso")
	if afirmaSucesso && estado != CompanionConcluido {
		return false
	}
	return true
}
