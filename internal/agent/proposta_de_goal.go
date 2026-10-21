package agent

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrConversaVazia e' devolvido quando nao ha' conversa de onde propor.
var ErrConversaVazia = errors.New("conversa vazia: nao ha' de onde propor um goal")

// MensagemDaConversa e' o minimo que a proposta precisa de uma mensagem.
type MensagemDaConversa struct {
	Papel    string `json:"papel"` // "user" | "assistant"
	Conteudo string `json:"conteudo"`
}

// PropostaDeGoal e' um goal PROPOSTO -- nunca criado.
//
// A linha B8 do adendo pede "conversao conversa->/goal com motivo explicado". O que se pode fazer
// sem inventar e' isto: devolver o que a conversa DISSE, o que foi inferido e a razao, e **listar o
// que falta**. Criar o goal e' de quem le.
//
//	*Uma proposta que preenche o que faltava com um valor plausivel nao e' uma proposta: e' um goal
//	inventado com boas maneiras.*
type PropostaDeGoal struct {
	Goal      CompanyGoal `json:"goal"`
	Motivo    string      `json:"motivo"`
	Confianca string      `json:"confianca"` // "alta" | "media" | "baixa"
	// Faltando e' o que a conversa nao disse. Enquanto nao estiver vazio, `Metric` fica vazio de
	// proposito: `CompanyStore.AddGoal` recusa goal sem metrica, e essa recusa e' a proteccao.
	Faltando []string `json:"faltando"`
}

var (
	reNumeroComUnidade = regexp.MustCompile(`(\d+(?:[.,]\d+)?)\s*(%|ms|s|min|h|horas?|dias?|usuarios?|usuárias?|usuarios|pedidos?|req/s|rps|R\$|\$|euros?)`)
	rePrimeiraFrase    = regexp.MustCompile(`^(.{3,160}?)(?:[.!?\n]|$)`)
)

// metricasConhecidas mapeia palavra na conversa -> metrica do goal. Lista FECHADA: uma metrica que
// nao esteja aqui nao e' adivinhada -- fica em `Faltando`.
var metricasConhecidas = []struct {
	palavras []string
	metrica  string
}{
	{[]string{"latencia", "latência", "tempo de resposta", "resposta demora", "mais rapido", "mais rápido", "lento"}, "tempo de resposta"},
	{[]string{"receita", "faturamento", "vendas", "mrr"}, "receita"},
	{[]string{"custo", "gasto", "despesa"}, "custo"},
	{[]string{"usuarios", "usuários", "usuarios ativos", "base de usuarios"}, "usuários ativos"},
	{[]string{"erros", "falhas", "5xx", "disponibilidade", "uptime"}, "taxa de erro"},
	{[]string{"conversao", "conversão", "checkout", "carrinho"}, "taxa de conversão"},
	{[]string{"pedidos", "requisicoes", "requisições", "rps"}, "pedidos por segundo"},
}

// ProporGoalDaConversa devolve uma proposta, o motivo e o que falta.
//
// AS TRES REGRAS:
//  1. sem conversa nao ha' proposta (fail-closed);
//  2. nada e' inventado: metrica fora da lista fechada, alvo sem numero e periodo sem palavra ficam
//     em `Faltando`, e `Metric` so' e' preenchida quando a conversa a nomeia;
//  3. o motivo diz ONDE cada pedaco foi encontrado -- sem isso, a proposta e' indistinguivel de um
//     palpite bem formatado.
func ProporGoalDaConversa(conversa []MensagemDaConversa) (PropostaDeGoal, error) {
	texto := ""
	for _, m := range conversa {
		if strings.TrimSpace(m.Conteudo) == "" {
			continue
		}
		texto += strings.TrimSpace(m.Conteudo) + "\n"
	}
	if texto == "" {
		return PropostaDeGoal{}, ErrConversaVazia
	}
	minusculo := strings.ToLower(texto)

	proposta := PropostaDeGoal{}
	var achados []string
	var faltando []string

	// Titulo: a primeira frase da conversa, cortada em 160.
	if m := rePrimeiraFrase.FindStringSubmatch(strings.TrimSpace(texto)); m != nil {
		proposta.Goal.Title = strings.TrimSpace(m[1])
	}
	if proposta.Goal.Title == "" {
		faltando = append(faltando, "titulo")
	}

	// Metrica: lista fechada.
	for _, candidata := range metricasConhecidas {
		for _, palavra := range candidata.palavras {
			if strings.Contains(minusculo, palavra) {
				proposta.Goal.Metric = candidata.metrica
				achados = append(achados, fmt.Sprintf("metrica %q (a conversa diz %q)", candidata.metrica, palavra))
				break
			}
		}
		if proposta.Goal.Metric != "" {
			break
		}
	}
	if proposta.Goal.Metric == "" {
		faltando = append(faltando, "metrica")
	}

	// Alvo: primeiro numero com unidade.
	if m := reNumeroComUnidade.FindStringSubmatch(texto); m != nil {
		numero, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
		if err != nil {
			faltando = append(faltando, "alvo (numero ilegivel: "+m[1]+")")
		} else {
			proposta.Goal.Target = numero
			proposta.Goal.Unit = strings.TrimSpace(m[2])
			achados = append(achados, fmt.Sprintf("alvo %s %s (a conversa diz %q)", m[1], m[2], m[0]))
		}
	} else {
		faltando = append(faltando, "alvo")
	}

	// Periodo: lista fechada de palavras.
	switch {
	case strings.Contains(minusculo, "trimestre"):
		proposta.Goal.Period = "trimestre"
		achados = append(achados, "periodo trimestre")
	case strings.Contains(minusculo, "semana"):
		proposta.Goal.Period = "semana"
		achados = append(achados, "periodo semana")
	case strings.Contains(minusculo, "mes") || strings.Contains(minusculo, "mês"):
		proposta.Goal.Period = "mes"
		achados = append(achados, "periodo mes")
	default:
		faltando = append(faltando, "periodo")
	}

	proposta.Faltando = faltando
	switch len(faltando) {
	case 0:
		proposta.Confianca = "alta"
	case 1:
		proposta.Confianca = "media"
	default:
		proposta.Confianca = "baixa"
	}

	if len(achados) == 0 {
		proposta.Motivo = "a conversa nao nomeia metrica, alvo nem periodo: nada foi inferido"
	} else {
		proposta.Motivo = "encontrado na conversa: " + strings.Join(achados, "; ")
		if len(faltando) > 0 {
			proposta.Motivo += ". falta: " + strings.Join(faltando, ", ")
		}
	}
	return proposta, nil
}
