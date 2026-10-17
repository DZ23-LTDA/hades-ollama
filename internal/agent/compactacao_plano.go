package agent

import (
	"strconv"
	"strings"

	"github.com/ollama/ollama/api"
)

// PLANO DE COMPACTAÇÃO INCREMENTAL HIERÁRQUICA (§2 do complemento universal de memória).
//
// O QUE ESTE FICHEIRO DECIDE, E O QUE ELE DELIBERADAMENTE NÃO FAZ: decide QUAIS
// mensagens ficam intactas e QUAL faixa vai ser resumida. Não resume, não grava, não
// chama modelo, não altera nada. É a decisão, separada do efeito — para poder ser
// testada sem provider e para que o efeito (gravar o bruto, resumir, substituir) seja
// um passo explícito e auditável.
//
// AS DUAS REGRAS QUE NÃO SÃO NEGOCIÁVEIS, e ambas vêm do §2:
//  1. AS MENSAGENS RECENTES FICAM INTACTAS. Compactar tudo destrói a conversa em curso.
//  2. A ÚLTIMA MENSAGEM DO UTILIZADOR FICA SEMPRE INTACTA. É a instrução CORRENTE; se o
//     resumo a engolir, o modelo passa a responder a uma versão resumida do pedido — e
//     "resumir a instrução actual" é exactamente o defeito que o §2 chama de perder o
//     objectivo. Por isso a janela recente é ALARGADA quando necessário para a incluir.

// JanelaRecentePadrao é quantas mensagens finais ficam sempre intactas quando o
// chamador não declara outra coisa. Não é um número sagrado: é um ponto de partida
// declarado, e o teste fixa o comportamento, não a virtude do valor.
const JanelaRecentePadrao = 6

// PlanoDeCompactacao é a decisão, com a contagem que a justifica. Nada aqui é um efeito.
type PlanoDeCompactacao struct {
	// Resumir é a faixa ANTIGA, que pode ser substituída por um resumo versionado.
	Resumir []api.Message
	// Intactas são as mensagens que seguem tal e qual para o modelo.
	Intactas []api.Message
	// Motivo declara a decisão em números, e é vazio apenas quando não há nada a decidir.
	Motivo string
	// Compactavel é false quando não há faixa antiga suficiente: nesse caso o plano não
	// muda NADA, e dizer "compactei" seria falso.
	Compactavel bool
}

// PlanejarCompactacao divide as mensagens. `janela` é quantas finais ficam intactas;
// zero ou negativo usa o padrão.
func PlanejarCompactacao(mensagens []api.Message, janela int) PlanoDeCompactacao {
	if janela <= 0 {
		janela = JanelaRecentePadrao
	}
	total := len(mensagens)

	// A JANELA ALARGA-SE PARA INCLUIR A ÚLTIMA MENSAGEM DO UTILIZADOR. Sem isto, uma
	// janela pequena com mensagens de ferramenta a seguir ao pedido deixaria a instrução
	// corrente fora das intactas.
	corte := total - janela
	for i := total - 1; i >= 0; i-- {
		if strings.EqualFold(strings.TrimSpace(mensagens[i].Role), "user") {
			if i < corte {
				corte = i
			}
			break
		}
	}
	if corte < 0 {
		corte = 0
	}
	if corte == 0 {
		return PlanoDeCompactacao{
			Intactas:    append([]api.Message(nil), mensagens...),
			Motivo:      "nada a compactar: o histórico cabe inteiro na janela recente",
			Compactavel: false,
		}
	}
	return PlanoDeCompactacao{
		Resumir:  append([]api.Message(nil), mensagens[:corte]...),
		Intactas: append([]api.Message(nil), mensagens[corte:]...),
		// NOTA: `itoa` NÃO pode ser usado aqui — o pacote já tem um `itoa` declarado num
		// ficheiro de teste, e um símbolo de teste não existe no build de produção. O
		// compilador apanhou-o ("itoa redeclared in this block"); `strconv.Itoa` é a
		// resposta certa e não depende de ordem de ficheiros.
		Motivo: "compactar " + strconv.Itoa(corte) + " de " + strconv.Itoa(total) +
			" mensagens; " + strconv.Itoa(total-corte) + " ficam intactas",
		Compactavel: true,
	}
}
