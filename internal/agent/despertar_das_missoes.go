package agent

import (
	"strings"
	"time"
)

// DAS MISSOES AO DESPERTAR (E-637). Liga o REGISTO da missão à regra de coalescência (E-633).
//
// A pergunta que isto responde é a que o dispatcher precisa de fazer: *quais agentes têm de ser
// acordados, por que razão, e por causa de que missões.* Sem esta ponte, `DespertaresCoalescidos`
// seria uma regra bonita sem nada para agrupar.
//
// SÓ O QUE ESTÁ NO REGISTO: nenhum motivo é inferido. Uma missão concluída ou falhada não acorda
// ninguém; uma missão sem agente atribuído não produz sinal — *não se inventa um agente para
// acordar*.

const (
	// MotivoAprovacaoPendente: há uma decisão humana à espera. É o motivo mais caro de perder.
	MotivoAprovacaoPendente = "aprovacao pendente"
	// MotivoRecuperacao: a missão ficou interrompida e precisa de ser retomada.
	MotivoRecuperacao = "missao interrompida a recuperar"
	// MotivoTrabalhoPendente: a missão está pronta e ninguém a executou.
	MotivoTrabalhoPendente = "missao pronta a executar"
)

// SinaisDeMissao traduz o estado registado de UMA missão em sinais de despertar.
func SinaisDeMissao(missao Mission, quando time.Time) []SinalDeDespertar {
	agenteID := strings.TrimSpace(missao.AgentID)
	if agenteID == "" {
		// Sem agente atribuído não há a quem acordar. Um sinal com agente vazio seria tratado
		// adiante como «despertar sem dono» — e é melhor NÃO o produzir aqui, onde a razão é
		// conhecida, do que o empurrar para quem consome.
		return nil
	}
	motivos := make([]string, 0, 2)
	acrescentarMotivo := func(motivo string) {
		for _, existente := range motivos {
			if existente == motivo {
				return
			}
		}
		motivos = append(motivos, motivo)
	}
	switch missao.State {
	case MissionAwaitingApproval:
		acrescentarMotivo(MotivoAprovacaoPendente)
	case MissionRecovering:
		acrescentarMotivo(MotivoRecuperacao)
	case MissionReady, MissionPlanning, MissionCreated:
		acrescentarMotivo(MotivoTrabalhoPendente)
	}
	// Uma missão pode ter aprovação pendente SEM estar em `AWAITING_APPROVAL` (o estado é derivado
	// do plano e pode não ter sido actualizado): o que manda é a aprovação registada.
	for _, aprovacao := range missao.Approvals {
		if aprovacao.Status == ApprovalStatus("PENDING") {
			acrescentarMotivo(MotivoAprovacaoPendente)
			break
		}
	}
	sinais := make([]SinalDeDespertar, 0, len(motivos))
	for _, motivo := range motivos {
		sinais = append(sinais, SinalDeDespertar{
			AgenteID: agenteID,
			MissaoID: missao.ID,
			Motivo:   motivo,
			Quando:   quando,
		})
	}
	return sinais
}

// DespertaresDasMissoes reúne os sinais de VÁRIAS missões e devolve os despertares COALESCIDOS.
//
// O dispatcher chama isto com as missões visíveis e obtém, por agente, UM despertar que leva todas
// as razões e todas as missões — em vez de N despertares para N missões.
func DespertaresDasMissoes(missoes []Mission, quando time.Time) []Despertar {
	var sinais []SinalDeDespertar
	for _, missao := range missoes {
		sinais = append(sinais, SinaisDeMissao(missao, quando)...)
	}
	return DespertaresCoalescidos(sinais)
}
