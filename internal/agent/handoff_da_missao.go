package agent

import (
	"errors"
	"strconv"
	"strings"
)

// HANDOFF DERIVADO DA MISSÃO (§4 + §6) — E-595.
//
// O `HandoffDoJob` (E-593) cobre a orquestração de tarefas. Este cobre a MISSÃO, que é a
// unidade que o Cowork e o Mission Control mostram — e que **já regista** o que um handoff
// precisa: objetivo, plano, capacidades concedidas, estado e aprovações
// (`types.go:75`). Não é preciso campo novo em `Mission`: derivar é mais seguro do que
// ANEXAR, porque anexar contexto vindo de outro agente é uma superfície de injecção que
// exigiria a sua própria governação.
//
// E CORRIGE UM LIMITE QUE EU PRÓPRIO DECLAREI NO E-593. Lá escrevi que «a orquestração não
// regista capacidades ⇒ a autorização sai vazia». Isso é verdade para o JOB e **falso para
// a MISSÃO**, que tem `Capabilities`. Logo o handoff de uma missão transporta a autorização
// REAL — e transporta SÓ a concedida: nunca uma capacidade que não esteja escrita no
// registo. Autorização que se deduz é autorização que se inventa.

// HandoffDaMissao deriva o contrato de handoff do registo da missão.
func HandoffDaMissao(mission Mission) (Handoff, error) {
	if strings.TrimSpace(mission.ID) == "" {
		return Handoff{}, errors.New("missão sem id: um handoff sem origem não é rastreável")
	}
	if strings.TrimSpace(mission.Objective) == "" {
		return Handoff{}, errors.New("missão sem objetivo: não há handoff possível")
	}

	// O ESTADO DO TRABALHO NUNCA É VAZIO — e não por conveniência: `Validar` (E-588) recusa
	// um handoff sem estado, e uma missão acabada de criar é exactamente o caso em que a
	// próxima IA mais precisa de saber que não há nada feito.
	estado := []string{
		"estado registado da missão: " + string(mission.State),
		"plano com " + strconv.Itoa(len(mission.Plan)) + " passo(s)",
	}

	// PENDENTES: cada passo do plano é trabalho por fazer. Um passo que exige aprovação vai
	// MARCADO — é o que impede a próxima IA de o executar por não saber que esperava decisão.
	pendentes := make([]string, 0, len(mission.Plan))
	for _, passo := range mission.Plan {
		titulo := strings.TrimSpace(passo.Title)
		if titulo == "" {
			titulo = strings.TrimSpace(passo.Kind)
		}
		if titulo == "" {
			titulo = "passo sem título"
		}
		linha := titulo
		if id := strings.TrimSpace(passo.ID); id != "" {
			linha = id + " — " + linha
		}
		if risco := strings.TrimSpace(string(passo.Risk)); risco != "" {
			linha += " (risco declarado: " + risco + ")"
		}
		if passo.RequiresApproval {
			linha += " ***EXIGE APROVAÇÃO***"
		}
		pendentes = append(pendentes, linha)
	}
	if n := len(mission.Approvals); n > 0 {
		pendentes = append(pendentes, strconv.Itoa(n)+" aprovação(ões) registada(s) na missão")
	}

	restricoes := []string{}
	if erro := strings.TrimSpace(mission.LastError); erro != "" {
		restricoes = append(restricoes, "último erro registado: "+erro)
	}
	if strings.TrimSpace(mission.Workspace) != "" {
		restricoes = append(restricoes, "workspace: "+strings.TrimSpace(mission.Workspace))
	}
	if mission.WorkspaceIsolated {
		restricoes = append(restricoes, "workspace isolado: não sair dele")
	}
	if branch := strings.TrimSpace(mission.GitBranch); branch != "" {
		restricoes = append(restricoes, "ramo git: "+branch)
	}

	handoff := Handoff{
		ProjetoID:        strings.TrimSpace(mission.ProjectID),
		MissaoID:         strings.TrimSpace(mission.ID),
		Objetivo:         strings.TrimSpace(mission.Objective),
		EstadoDoTrabalho: estado,
		Restricoes:       restricoes,
		Pendentes:        pendentes,
		Evidencias:       []string{"missão " + strings.TrimSpace(mission.ID) + " (versão " + strconv.FormatInt(mission.Version, 10) + ")"},
		Autorizacao: AutorizacaoDoHandoff{
			// SÓ A CONCEDIDA: o que está escrito no registo. Nunca uma capacidade inferida.
			Capacidades: append([]string(nil), mission.Capabilities...),
		},
	}
	if len(handoff.Autorizacao.Capacidades) == 0 {
		handoff.Autorizacao.AprovacoesPendentes = []string{"a missão não tem capacidades concedidas: a próxima IA precisa de autorização explícita"}
	}
	return handoff, nil
}
