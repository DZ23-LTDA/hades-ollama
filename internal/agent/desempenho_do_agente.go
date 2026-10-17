package agent

import (
	"time"
)

// AVALIAÇÃO DE DESEMPENHO POR AGENTE (complemento Paperclip §10), DERIVADA SÓ DE DADOS REGISTADOS.
//
// O QUE O §10 PEDE, E O QUE ELE PROÍBE: pede taxa de conclusão, qualidade, erros e retrabalho,
// tempo, tokens, custos, intervenções humanas e cumprimento de orçamento — «baseada em métricas
// observáveis». E proíbe o que é mais fácil: ***«evite avaliações fictícias geradas por LLM sem
// verificação»***.
//
// POR ISSO ESTE FICHEIRO NÃO CHAMA MODELO NENHUM. Ele soma o que a missão JÁ registou
// (`State`, `Plan[].Attempts`, `Usage`, `UsageOrigin`, marcas de tempo e aprovações) e ***declara
// o que não consegue medir*** em `NaoMedido`, em vez de preencher com uma nota inventada. Uma
// "nota de qualidade" saída de um modelo seria a avaliação fictícia que o complemento proíbe.
//
// E HÁ UM CUIDADO DE CONTABILIDADE QUE VEM DO §8: `ModelUsage.CostKnown` distingue custo
// CONHECIDO de custo DESCONHECIDO. Somar zeros em silêncio faria um painel mostrar «custo: 0» para
// missões que gastaram dinheiro e não sabem quanto — *número errado é pior do que número ausente*.
// Aqui o custo conhecido é somado e as missões de custo desconhecido são CONTADAS e declaradas.

// RelatorioDeDesempenho é a avaliação de um conjunto de missões. Todos os campos vêm de dados
// registados; `NaoMedido` diz o que a avaliação NÃO cobre.
type RelatorioDeDesempenho struct {
	Missoes     int `json:"missions"`
	Concluidas  int `json:"completed"`
	Falhadas    int `json:"failed"`
	Canceladas  int `json:"cancelled"`
	EmCurso     int `json:"running"`
	Recuperando int `json:"recovering"`

	// TaxaDeConclusao é concluídas / total. Zero missões dá 0, e não uma divisão por zero
	// disfarçada: um relatório sem missões não tem desempenho a afirmar.
	TaxaDeConclusao float64 `json:"completion_rate"`

	Passos              int     `json:"steps"`
	PassosComRetrabalho int     `json:"steps_with_rework"`
	TentativasExtra     int     `json:"extra_attempts"`
	TaxaDeRetrabalho    float64 `json:"rework_rate"`

	AprovacoesPendentes int `json:"pending_approvals"`

	TokensDeEntrada             int   `json:"prompt_tokens"`
	TokensDeSaida               int   `json:"completion_tokens"`
	ChamadasDeModelo            int   `json:"model_calls"`
	CustoConhecidoCents         int64 `json:"known_cost_cents"`
	MissoesComCustoConhecido    int   `json:"missions_with_known_cost"`
	MissoesComCustoDesconhecido int   `json:"missions_with_unknown_cost"`

	// TempoTotalDeExecucao soma, por missão, de `CreatedAt` até `CompletedAt` (ou `UpdatedAt`
	// quando ainda não concluiu). Nunca é negativo.
	TempoTotalDeExecucao time.Duration `json:"total_execution_time"`

	// NaoMedido é a parte honesta: o que o §10 pede e estes dados NÃO permitem afirmar.
	NaoMedido []string `json:"not_measured"`
}

// O que o §10 pede e o registo da missão não permite medir. Declarado como constante para o
// relatório e o relatório dizerem exactamente a mesma coisa.
var oQueNaoSeMedeAqui = []string{
	"qualidade do resultado (exige verificação humana ou teste, não sai do estado da missão)",
	"satisfacao do utilizador (so' quando informada, e nao e' registada hoje)",
	"cumprimento de orcamento (exige o limite definido; o registo da missao nao guarda o tecto)",
	"intervencoes humanas por tipo (conta-se a aprovacao PENDENTE, nao a decisao tomada)",
}

// AvaliarDesempenho agrega missões em um relatório. Função PURA: não lê disco, não chama modelo,
// não escreve nada. É isso que a torna testável e que a impede de «avaliar» o que não mediu.
func AvaliarDesempenho(missoes []Mission) RelatorioDeDesempenho {
	relatorio := RelatorioDeDesempenho{Missoes: len(missoes)}

	for _, missao := range missoes {
		// Estados REAIS da missão (types.go:12-21). Os não-finais contam como «em curso», e o
		// `default` é DELIBERADO: acrescentar um estado novo ao produto não pode fazer o
		// relatório deixar de contar as missões que o usam. (Foi o compilador que me corrigiu
		// aqui: eu tinha escrito `MissionPending`/`MissionPlanned`, que não existem.)
		switch missao.State {
		case MissionCompleted:
			relatorio.Concluidas++
		case MissionFailed:
			relatorio.Falhadas++
		case MissionCancelled:
			relatorio.Canceladas++
		case MissionRecovering:
			relatorio.Recuperando++
		default:
			relatorio.EmCurso++
		}

		for _, passo := range missao.Plan {
			relatorio.Passos++
			if passo.Attempts > 1 {
				relatorio.PassosComRetrabalho++
				relatorio.TentativasExtra += passo.Attempts - 1
			}
		}

		for _, aprovacao := range missao.Approvals {
			if aprovacao.Status == ApprovalStatus("PENDING") {
				relatorio.AprovacoesPendentes++
			}
		}

		if missao.Usage != nil {
			relatorio.TokensDeEntrada += missao.Usage.PromptTokens
			relatorio.TokensDeSaida += missao.Usage.CompletionTokens
			relatorio.ChamadasDeModelo += missao.Usage.Calls
			if missao.Usage.CostKnown {
				relatorio.CustoConhecidoCents += missao.Usage.CostCents
				relatorio.MissoesComCustoConhecido++
			} else {
				// Custo desconhecido NÃO entra na soma e é CONTADO: um painel que somasse zero
				// estaria a afirmar «custou nada» sobre o que ninguém sabe quanto custou.
				relatorio.MissoesComCustoDesconhecido++
			}
		}

		fim := missao.UpdatedAt
		if missao.CompletedAt != nil {
			fim = *missao.CompletedAt
		}
		if duracao := fim.Sub(missao.CreatedAt); duracao > 0 {
			relatorio.TempoTotalDeExecucao += duracao
		}
	}

	if relatorio.Missoes > 0 {
		relatorio.TaxaDeConclusao = float64(relatorio.Concluidas) / float64(relatorio.Missoes)
	}
	if relatorio.Passos > 0 {
		relatorio.TaxaDeRetrabalho = float64(relatorio.PassosComRetrabalho) / float64(relatorio.Passos)
	}
	relatorio.NaoMedido = append([]string(nil), oQueNaoSeMedeAqui...)
	return relatorio
}
