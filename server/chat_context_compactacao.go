package server

import (
	"strings"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/internal/agent"
)

// COMPACTAÇÃO NO FUNIL DO CHAT — a unidade que o middleware chama (E-583).
//
// PORQUE ESTA FUNÇÃO EXISTE EM VEZ DE O CÓDIGO VIVER DENTRO DO MIDDLEWARE: abrir
// ficheiros, derivar caminhos e tratar erros dentro de um `gin.HandlerFunc` torna a
// ligação impossível de testar sem HTTP e mistura duas responsabilidades (decidir o
// contexto e servir o pedido). Aqui a raiz é INJECTADA e o resultado é explícito, para
// que a ligação final sejam três linhas no middleware e o comportamento seja medido
// directamente.
//
// A REGRA DE OURO DESTA FUNÇÃO: NUNCA piora o que o utilizador tinha. Se a raiz não
// estiver declarada, se a decisão não mandar compactar, ou se QUALQUER passo falhar, ela
// devolve as mensagens ORIGINAIS tal e qual. Uma compactação que falha não pode custar a
// conversa — é o §2 («em erro, cancelamento ou interrupção, o histórico original continua
// acessível») aplicado ao ponto onde o erro acontece.
//
// E ela NÃO decide se deve compactar: recebe a decisão do E-568. Decide e executa ficam
// separados, para que cada um possa ser medido sozinho.

// ResultadoDaCompactacaoNoChat diz o que aconteceu, para registo e para teste.
type ResultadoDaCompactacaoNoChat struct {
	// Aplicada é true SÓ quando as mensagens devolvidas são diferentes das originais.
	Aplicada bool `json:"aplicada"`
	// Motivo declara o que aconteceu, incluindo todas as recusas.
	Motivo string `json:"motivo"`
	// Caminho é o ficheiro do histórico bruto usado (vazio quando não se compactou).
	Caminho string `json:"caminho,omitempty"`
	// ReferenciaDoBruto é o topo da cadeia gravada — a prova de que o material existe.
	ReferenciaDoBruto string `json:"referencia_do_bruto,omitempty"`
}

// CompactarMensagensDoChat aplica o fluxo de compactação quando a decisão o manda,
// devolvendo SEMPRE um conjunto de mensagens utilizável.
func CompactarMensagensDoChat(raiz, organizacao, projeto string, mensagens []api.Message, decisao DecisaoDeOrcamento) ([]api.Message, ResultadoDaCompactacaoNoChat) {
	if !decisao.Compactar {
		return mensagens, ResultadoDaCompactacaoNoChat{Motivo: "decisão não manda compactar: " + decisao.Motivo}
	}
	if strings.TrimSpace(raiz) == "" {
		// FALHA FECHADO: sem raiz não há onde gravar o bruto, e compactar sem guardar o
		// material perderia a faixa antiga sem rasto. Mantém-se a conversa inteira.
		return mensagens, ResultadoDaCompactacaoNoChat{
			Motivo: "raiz do histórico não declarada: mantive a conversa original em vez de compactar sem guardar o material",
		}
	}
	// NADA A COMPACTAR ⇒ NENHUM EFEITO. Esta verificação vem ANTES de derivar ou abrir
	// qualquer ficheiro: a primeira versão abria o histórico primeiro e, numa conversa
	// curta, deixava `<raiz>/historico/.../<projeto>.jsonl` criado e vazio em disco — um
	// efeito colateral para uma compactação que não aconteceu. O teste apanhou-o.
	// *Um efeito que precede a decisão é um efeito que ninguém autorizou.*
	if plano := agent.PlanejarCompactacao(mensagens, 0); !plano.Compactavel {
		return mensagens, ResultadoDaCompactacaoNoChat{Motivo: "nada a compactar: " + plano.Motivo}
	}
	caminho, err := agent.CaminhoDoHistorico(raiz, organizacao, projeto)
	if err != nil {
		return mensagens, ResultadoDaCompactacaoNoChat{
			Motivo: "caminho do histórico recusado: " + err.Error(),
		}
	}
	historico, err := agent.AbrirHistoricoBruto(caminho)
	if err != nil {
		return mensagens, ResultadoDaCompactacaoNoChat{
			Caminho: caminho,
			Motivo:  "não consegui abrir o histórico bruto: " + err.Error(),
		}
	}
	defer historico.Fechar()

	// As mensagens do chat não trazem missão; o projeto é o escopo que existe aqui.
	resultado, novas, err := agent.FluxoDaCompactacao(mensagens, 0, 0, historico, projeto, "")
	if err != nil {
		return mensagens, ResultadoDaCompactacaoNoChat{
			Caminho: caminho,
			Motivo:  "compactação recusada, conversa original mantida: " + err.Error(),
		}
	}
	if !resultado.Compactado {
		return mensagens, ResultadoDaCompactacaoNoChat{
			Caminho: caminho,
			Motivo:  "compactação não aplicada: " + resultado.Motivo,
		}
	}
	if len(novas) == 0 {
		// Salvaguarda: um resultado "compactado" sem mensagens é contradição; devolver
		// vazio apagaria a conversa. Mantém-se a original.
		return mensagens, ResultadoDaCompactacaoNoChat{
			Caminho: caminho,
			Motivo:  "compactação devolveu zero mensagens: contradição, mantive a conversa original",
		}
	}
	return novas, ResultadoDaCompactacaoNoChat{
		Aplicada:          true,
		Motivo:            resultado.Motivo,
		Caminho:           caminho,
		ReferenciaDoBruto: resultado.UltimoHash,
	}
}
