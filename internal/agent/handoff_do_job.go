package agent

import (
	"errors"
	"strconv"
	"strings"
)

// INTEGRAÇÃO DO HANDOFF COM A ORQUESTRAÇÃO (§4 + §6) — E-593.
//
// O CONTRATO DO E-588 EXISTIA SEM CHAMADOR. Um contrato que ninguém alimenta é o defeito
// «apenas adaptador» que esta missão persegue: provado por teste, inerte em produção. Esta
// função é o primeiro produtor REAL de handoff: converte uma missão de orquestração no
// contrato portátil, para que outro agente, outro modelo ou outra máquina possa continuar —
// é o §4 («transferir contexto entre modelos e agentes») e o §6 («retomar missões após
// interrupções») no mesmo passo.
//
// COMO O ESTADO É DERIVADO, E PORQUE NÃO PELAS CONSTANTES: `FinishedAt`/`Error`/`Attempts`
// são DADOS já gravados; as constantes de estado são rótulos que podem mudar de nome sem
// que o significado mude. Derivar de dados mantém esta função correcta mesmo que os rótulos
// evoluam — e evita que eu escreva identificadores que não li.
//
// E ELE NÃO INVENTA AUTORIZAÇÃO: a orquestração não carrega capacidades, logo o handoff sai
// com a autorização VAZIA e com uma aprovação pendente declarada («não há autorização
// registada»). Um handoff que herdasse permissões por omissão seria a elevação de privilégio
// por arrastamento que o E-588 proíbe. O que não se sabe, declara-se.

// HandoffDoJob converte uma missão de orquestração no contrato de handoff (E-588).
func HandoffDoJob(job OrchestrationJob) (Handoff, error) {
	if strings.TrimSpace(job.ID) == "" {
		return Handoff{}, errors.New("missão sem id: um handoff sem origem não é rastreável")
	}
	if strings.TrimSpace(job.Objective) == "" {
		return Handoff{}, errors.New("missão sem objetivo: não há handoff possível")
	}

	feitas := make([]string, 0, len(job.Tasks))
	pendentes := make([]string, 0, len(job.Tasks))
	evidencias := make([]string, 0, len(job.Tasks))
	for _, t := range job.Tasks {
		rotulo := strings.TrimSpace(string(t.Role))
		if rotulo == "" {
			rotulo = "papel-nao-declarado"
		}
		descricao := rotulo + " — " + strings.TrimSpace(t.Objective)
		switch {
		case t.FinishedAt == nil:
			// NÃO TERMINOU: é trabalho pendente, e dizer isto é o que impede a próxima IA de
			// assumir que a missão acabou.
			pendentes = append(pendentes, descricao+" (tentativas: "+strconv.Itoa(t.Attempts)+")")
		case strings.TrimSpace(t.Error) != "":
			// Terminou COM erro: não é trabalho feito nem trabalho por fazer — é falha
			// declarada, e vai nomeada com o motivo para não ser repetida às cegas.
			pendentes = append(pendentes, descricao+" — FALHOU: "+strings.TrimSpace(t.Error))
		default:
			feitas = append(feitas, descricao+" (tentativas: "+strconv.Itoa(t.Attempts)+")")
		}
		// A EVIDÊNCIA É A IDENTIDADE DA TAREFA, não o texto que ela produziu: o texto pode
		// ser grande e o handoff tem teto; o id permite ir buscá-lo.
		if strings.TrimSpace(t.ID) != "" {
			evidencias = append(evidencias, "tarefa "+strings.TrimSpace(t.ID))
		}
	}
	if resumo := strings.TrimSpace(job.Summary); resumo != "" {
		feitas = append(feitas, "resumo da redução: "+resumo)
	}
	if len(feitas) == 0 {
		// UMA MISSÃO SEM NADA CONCLUÍDO AINDA É RETOMÁVEL, e `Validar` (E-588) exige estado
		// do trabalho não vazio para que a próxima IA não repita o que já foi feito. "Nada
		// concluído" é uma informação, não uma ausência: declará-la mantém o handoff válido
		// em vez de o recusar precisamente no caso mais urgente — o de uma missão que mal
		// começou.
		feitas = append(feitas, "nada concluído ainda")
	}

	// AS RESTRIÇÕES VÊM DO ORÇAMENTO: são limites REAIS que a próxima IA tem de respeitar,
	// e não conselhos. Declará-los evita que ela gaste o que já foi gasto.
	restricoes := []string{
		"no máximo " + strconv.Itoa(job.Budget.MaxAgents) + " agentes em paralelo",
		"até " + strconv.FormatInt(job.Budget.MaxSeconds, 10) + " segundos por missão",
		"até " + strconv.Itoa(job.Budget.MaxRetries) + " novas tentativas por tarefa",
		"até " + strconv.Itoa(job.Budget.MaxOutputBytes) + " bytes de saída por tarefa",
	}
	if len(job.Conflicts) > 0 {
		restricoes = append(restricoes, "conflitos já detetados: "+strings.Join(job.Conflicts, "; "))
	}

	handoff := Handoff{
		ProjetoID:        strings.TrimSpace(job.ProjectID),
		Objetivo:         strings.TrimSpace(job.Objective),
		EstadoDoTrabalho: feitas,
		Restricoes:       restricoes,
		// `Pendentes` NUNCA é nil: uma missão a ser entregue declara explicitamente o que
		// falta, mesmo que seja «nada». É a distinção do E-588 entre não declarado e vazio.
		Pendentes:  pendentes,
		Evidencias: evidencias,
		// A AUTORIZAÇÃO NÃO É HERDADA: sai vazia, com a falta declarada. Um handoff que
		// deduzisse permissões estaria a elevar privilégio por arrastamento.
		Autorizacao: AutorizacaoDoHandoff{
			AprovacoesPendentes: []string{"a orquestração não regista capacidades: a próxima IA precisa de autorização explícita"},
		},
	}
	if handoff.Pendentes == nil {
		handoff.Pendentes = []string{}
	}
	return handoff, nil
}
