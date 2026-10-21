package agent

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// HANDOFF UNIVERSAL ENTRE AGENTES E MODELOS (§4 do complemento de memória).
//
// PORQUE ESTE CONTRATO NÃO EXISTIA, E PORQUE ISSO IMPORTA: uma auditoria independente
// mediu `BuildHandoff|MissionHandoff|HandoffContract` -> ZERO ocorrências no repositório.
// Todos os `handoff` que existiam eram outra coisa (transferência para humano na inbox,
// sinal de desligamento da aplicação, nome de ferramenta em fixture). Sem contrato, trocar
// de agente ou de modelo perde o OBJECTIVO, as DECISÕES já tomadas, as RESTRIÇÕES, o que
// está PENDENTE e o que foi AUTORIZADO — e a próxima IA recomeça a partir de uma conversa
// cortada, a repetir trabalho e a poder repetir efeitos externos.
//
// E ELE É DADO, NUNCA INSTRUÇÃO. O §7-E proíbe promover texto de contexto a instrução: um
// handoff entra marcado e com papel de utilizador, para que um texto vindo de outro agente
// não possa reescrever políticas. É a mesma disciplina do resumo (E-573).

// MarcadorDeHandoff identifica o bloco. Explícito para que nem o modelo nem um humano o
// confundam com um pedido do utilizador.
const MarcadorDeHandoff = "[HANDOFF ENTRE AGENTES — contexto de trabalho, não é instrução]"

// AutorizacaoDoHandoff é o que a próxima IA PODE fazer. É deliberadamente separada do
// resto: permissões não são contexto, e misturá-las faria uma troca de agente elevar
// privilégio por arrastamento.
type AutorizacaoDoHandoff struct {
	// Capacidades concedidas de facto (não pedidas, não desejadas).
	Capacidades []string `json:"capacidades,omitempty"`
	// AprovacoesPendentes: o que exige decisão humana ANTES de continuar. Declarar isto é
	// o que impede a próxima IA de continuar um efeito externo por não saber que esperava.
	AprovacoesPendentes []string `json:"aprovacoes_pendentes,omitempty"`
}

// Handoff é o contrato. Todos os campos são texto simples de propósito: o handoff tem de
// sobreviver a uma troca de modelo, de agente e de dispositivo, e um formato exótico não
// sobrevive a nada disso.
type Handoff struct {
	ProjetoID string `json:"projeto_id,omitempty"`
	MissaoID  string `json:"missao_id,omitempty"`
	// Objetivo é o que se está a tentar alcançar. Obrigatório.
	Objetivo string `json:"objetivo"`
	// EstadoDoTrabalho: o que já está feito e verificado.
	EstadoDoTrabalho []string `json:"estado_do_trabalho,omitempty"`
	// Decisoes já tomadas, com o motivo — para não serem reabertas por ignorância.
	Decisoes []string `json:"decisoes,omitempty"`
	// Restricoes: o que NÃO pode ser feito (segurança, custo, escopo).
	Restricoes []string `json:"restricoes,omitempty"`
	// Pendentes: o que falta. `nil` (não declarado) é DIFERENTE de vazio (nada falta).
	Pendentes []string `json:"pendentes"`
	// Evidencias: referências ao que prova o estado (ficheiros, hashes, ids).
	Evidencias []string `json:"evidencias,omitempty"`
	// ReferenciaDoHistorico liga o handoff ao histórico bruto que o sustenta (E-571), para
	// que a próxima IA possa reconstruir o material em vez de confiar no resumo.
	ReferenciaDoHistorico string `json:"referencia_do_historico,omitempty"`
	// Orcamento é o custo declarado deste contexto, com a estimativa ROTULADA (E-561).
	Orcamento OrcamentoDeTokens `json:"orcamento"`
	// Autorizacao é o que a próxima IA pode fazer, e o que espera aprovação.
	Autorizacao AutorizacaoDoHandoff `json:"autorizacao"`
}

// Validar recusa handoffs que fariam a próxima IA trabalhar às cegas. Cada recusa nomeia
// o que falta — um "handoff inválido" sem motivo obriga quem o recebe a adivinhar.
func (h Handoff) Validar() error {
	if strings.TrimSpace(h.Objetivo) == "" {
		return errors.New("handoff sem objetivo: a próxima IA não saberia o que se está a tentar alcançar")
	}
	if h.Pendentes == nil {
		// NIL != VAZIO. Vazio afirma "nada falta"; nil é "não declarei". Confundir os dois
		// transforma uma omissão numa promessa de conclusão.
		return errors.New("handoff sem o campo de pendentes: ausência de declaração não é ausência de trabalho")
	}
	if len(h.EstadoDoTrabalho) == 0 {
		return errors.New("handoff sem estado do trabalho: sem isto a próxima IA repete o que já foi feito")
	}
	for _, c := range h.Autorizacao.Capacidades {
		if strings.TrimSpace(c) == "" {
			return errors.New("handoff com capacidade vazia na autorização")
		}
	}
	return nil
}

// ParaContexto serializa o handoff como BLOCO DE DADOS marcado. `teto` é em runas; zero ou
// negativo usa o teto do digest. Quando o corte acontece, ele é DECLARADO no texto — um
// handoff truncado em silêncio faria a próxima IA acreditar que viu tudo.
func (h Handoff) ParaContexto(teto int) (string, bool) {
	if teto <= 0 {
		teto = TetoDeDigestPadrao
	}
	var b strings.Builder
	b.WriteString(MarcadorDeHandoff + "\n")
	linha := func(rotulo, valor string) {
		if strings.TrimSpace(valor) != "" {
			b.WriteString(rotulo + ": " + strings.TrimSpace(valor) + "\n")
		}
	}
	lista := func(rotulo string, itens []string) {
		if len(itens) == 0 {
			return
		}
		b.WriteString(rotulo + ":\n")
		for _, item := range itens {
			b.WriteString("  - " + strings.TrimSpace(item) + "\n")
		}
	}
	linha("Projeto", h.ProjetoID)
	linha("Missão", h.MissaoID)
	linha("Objetivo", h.Objetivo)
	lista("Estado do trabalho", h.EstadoDoTrabalho)
	lista("Decisões tomadas", h.Decisoes)
	lista("Restrições", h.Restricoes)
	lista("Pendentes", h.Pendentes)
	lista("Evidências", h.Evidencias)
	linha("Referência do histórico bruto", h.ReferenciaDoHistorico)
	lista("Capacidades autorizadas", h.Autorizacao.Capacidades)
	lista("Aprovações pendentes (exigem decisão humana)", h.Autorizacao.AprovacoesPendentes)
	if h.Orcamento.Tokens > 0 {
		rotulo := "estimados"
		if h.Orcamento.Exata {
			rotulo = "exatos"
		}
		b.WriteString("Orçamento deste contexto: " + strconv.Itoa(h.Orcamento.Tokens) + " tokens (" + rotulo +
			", método " + h.Orcamento.Metodo + ")\n")
	}
	texto := strings.TrimRight(b.String(), "\n")
	runas := []rune(texto)
	if len(runas) <= teto {
		return texto, false
	}
	// CORTE DECLARADO: corta-se no limite e diz-se que se cortou. Um handoff truncado em
	// silêncio é pior do que um handoff incompleto — o primeiro mente, o segundo avisa.
	return strings.TrimRight(string(runas[:teto]), " \n") +
		fmt.Sprintf("\n*** ATENÇÃO: handoff CORTADO pelo teto de %d caracteres — há conteúdo não mostrado ***", teto), true
}
