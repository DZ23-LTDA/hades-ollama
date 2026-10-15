package agent

import (
	"errors"
	"fmt"
	"strings"
)

// Complemento 48 / FASE 4 — MEMORIA VIVA COM ROTULOS.
//
// O QUE FOI MEDIDO ANTES DE ESCREVER (2026-10-15, commit 93f12ac19): este vocabulario tinha **ZERO**
// ocorrencias no codigo. A memoria do Hades existia, mas **sem dizer de que NATUREZA era cada afirmacao** - e
// e' isso que faz uma proposta ser lida como facto depois de tres conversas.
//
// REGRA DE OURO DESTE ARQUIVO (fail-closed): **um rotulo so' sobe com a prova que ele exige.**
// *FACT_VERIFIED sem fonte e' opiniao com carimbo; DECISION_APPROVED sem aprovador e' decisao de ninguem.*

// RotuloDeMemoria diz que NATUREZA tem a afirmacao.
type RotuloDeMemoria string

const (
	// MemoriaFatoVerificado: medido, com FONTE. Nao se promove a isto sem `Fonte`.
	MemoriaFatoVerificado RotuloDeMemoria = "FACT_VERIFIED"
	// MemoriaDecisaoAprovada: decidido por alguem identificado. Nao se promove a isto sem `Aprovador`.
	MemoriaDecisaoAprovada RotuloDeMemoria = "DECISION_APPROVED"
	// MemoriaProposta: sugerida, ainda nao decidida.
	MemoriaProposta RotuloDeMemoria = "PROPOSAL"
	// MemoriaHipotese: plausivel e nao verificada.
	MemoriaHipotese RotuloDeMemoria = "HYPOTHESIS"
	// MemoriaConteudoNaoConfiavel: veio de fora (pagina, arquivo, ferramenta) e **nao** e' instrucao nem facto.
	MemoriaConteudoNaoConfiavel RotuloDeMemoria = "UNTRUSTED_CONTENT"
)

// ErrRotuloDesconhecido: rotulo fora do vocabulario. *Vocabulario aberto transforma "nao medido" em qualquer
// coisa que o autor quiser chamar-lhe.*
var ErrRotuloDesconhecido = errors.New("unknown memory label")

// ErrPromocaoSemProva: a promocao pedida exigiria uma prova que a memoria nao tem.
var ErrPromocaoSemProva = errors.New("promotion requires evidence the memory does not have")

// ErrTransicaoDeMemoriaInvalida: a passagem de rotulo nao e' permitida (ver `TransicaoPermitida`).
var ErrTransicaoDeMemoriaInvalida = errors.New("memory label transition is not allowed")

// Memoria e' uma afirmacao com natureza declarada.
type Memoria struct {
	ID       string          `json:"id"`
	Rotulo   RotuloDeMemoria `json:"rotulo"`
	Conteudo string          `json:"conteudo"`
	// Fonte e' obrigatoria para FACT_VERIFIED (de onde veio a medicao).
	Fonte string `json:"fonte,omitempty"`
	// Aprovador e' obrigatorio para DECISION_APPROVED (quem decidiu).
	Aprovador string `json:"aprovador,omitempty"`
}

// Validar recusa memorias que nao sustentam o proprio rotulo. Cada recusa diz O QUE FALTA - um "invalido" sem
// motivo obriga quem recebe a adivinhar.
func (m Memoria) Validar() error {
	if strings.TrimSpace(m.Conteudo) == "" {
		return errors.New("memory without content")
	}
	switch m.Rotulo {
	case MemoriaFatoVerificado:
		if strings.TrimSpace(m.Fonte) == "" {
			return fmt.Errorf("%w: FACT_VERIFIED needs a source (was the measurement, and where)", ErrPromocaoSemProva)
		}
	case MemoriaDecisaoAprovada:
		if strings.TrimSpace(m.Aprovador) == "" {
			return fmt.Errorf("%w: DECISION_APPROVED needs an approver (a decision nobody signed is nobody's decision)", ErrPromocaoSemProva)
		}
	case MemoriaProposta, MemoriaHipotese, MemoriaConteudoNaoConfiavel:
		// Estes rotulos NAO exigem prova: e' exactamente por isso que eles existem.
	default:
		return fmt.Errorf("%w: %q", ErrRotuloDesconhecido, string(m.Rotulo))
	}
	return nil
}

// TransicaoPermitida diz se uma memoria PODE mudar de rotulo. A tabela e' deliberadamente estreita.
//
//	UNTRUSTED_CONTENT -> HYPOTHESIS/PROPOSAL     (pode ser considerada; nunca afirmada)
//	UNTRUSTED_CONTENT -> FACT_VERIFIED/DECISION  **PROIBIDO** - conteudo de fora nao vira facto por repeticao
//	HYPOTHESIS        -> PROPOSAL/FACT_VERIFIED  (FACT exige Fonte)
//	PROPOSAL          -> DECISION_APPROVED/HYPOTHESIS (DECISION exige Aprovador)
//	FACT_VERIFIED     -> HYPOTHESIS              (retratacao: quando a prova cai, o rotulo cai com ela)
//	DECISION_APPROVED -> PROPOSAL                (uma decisao pode ser substituida por nova proposta)
func TransicaoPermitida(de, para RotuloDeMemoria) bool {
	if de == para {
		return true // idempotente: reafirmar o mesmo rotulo nao e' promocao
	}
	switch de {
	case MemoriaConteudoNaoConfiavel:
		return para == MemoriaHipotese || para == MemoriaProposta
	case MemoriaHipotese:
		return para == MemoriaProposta || para == MemoriaFatoVerificado
	case MemoriaProposta:
		return para == MemoriaDecisaoAprovada || para == MemoriaHipotese
	case MemoriaFatoVerificado:
		return para == MemoriaHipotese
	case MemoriaDecisaoAprovada:
		return para == MemoriaProposta
	default:
		return false
	}
}

// Promover muda o rotulo **e exige a prova do rotulo novo**: o resultado passa por `Validar()` antes de ser
// devolvido. Duas travas em serie - a transicao e a prova - *porque so' a primeira deixaria passar um
// FACT_VERIFIED sem fonte, que e' o pior dos mundos: com aparencia de medicao e sem medicao nenhuma.*
func (m Memoria) Promover(para RotuloDeMemoria) (Memoria, error) {
	if !TransicaoPermitida(m.Rotulo, para) {
		return Memoria{}, fmt.Errorf("%w: %s -> %s", ErrTransicaoDeMemoriaInvalida, m.Rotulo, para)
	}
	promovida := m
	promovida.Rotulo = para
	if err := promovida.Validar(); err != nil {
		return Memoria{}, err
	}
	return promovida, nil
}
