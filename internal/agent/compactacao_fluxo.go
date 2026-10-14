package agent

import (
	"strings"

	"github.com/ollama/ollama/api"
)

// FLUXO COMPLETO DA COMPACTAÇÃO — as quatro peças numa chamada.
//
// O QUE ELE JUNTA, E EM QUE ORDEM:
//  1. `PlanejarCompactacao` (E-572) decide o que fica intacto e o que é resumido;
//  2. `ResumirPorDigest` (E-574) produz o texto, fiel ao material e declarando o corte;
//  3. `AplicarCompactacao` (E-573) GRAVA O BRUTO PRIMEIRO e só depois devolve as
//     mensagens novas — ou falha fechado, sem devolver nenhuma.
//
// PORQUE ISTO EXISTE SEPARADO DO FUNIL: é aqui que a recuperação pode ser provada ponta
// a ponta sem servidor, sem provider e sem HTTP. A ligação ao `chat_context.go` passa a
// ser uma chamada, e não quatro decisões espalhadas — e o teste de recuperação mede o
// comportamento real das quatro juntas em vez de medir cada uma isolada e supor o resto.
//
// DEGRADAR É SEGURO E DECLARADO: se o digest não couber em nada, `ResumirPorDigest`
// devolve vazio e `AplicarCompactacao` recusa — o resultado é a conversa ORIGINAL
// intacta, com o motivo a dizê-lo. Preferir a conversa completa a um resumo que não diz
// nada é a escolha certa, e não um caso de erro.

// FluxoDaCompactacao devolve o resultado e, quando não compactou, as mensagens originais.
func FluxoDaCompactacao(mensagens []api.Message, janela, tetoDigest int, historico *HistoricoBruto, projetoID, missaoID string) (ResultadoDaCompactacao, []api.Message, error) {
	plano := PlanejarCompactacao(mensagens, janela)
	if !plano.Compactavel {
		res, err := AplicarCompactacao(plano, historico, "", projetoID, missaoID)
		return res, append([]api.Message(nil), mensagens...), err
	}
	resumo, cortado := ResumirPorDigest(plano.Resumir, tetoDigest)
	_ = cortado // o corte já vai declarado no próprio texto pelo digest? não: ver abaixo.
	if strings.TrimSpace(resumo) == "" {
		// NADA COUBE. Não se substitui a conversa por nada: devolve-se a original e diz-se.
		return ResultadoDaCompactacao{
				Motivo: "nada do resumo coube no teto: mantenho a conversa original em vez de a substituir por nada",
			},
			append([]api.Message(nil), mensagens...), nil
	}
	res, err := AplicarCompactacao(plano, historico, resumo, projetoID, missaoID)
	if err != nil {
		// O `AplicarCompactacao` já devolveu ZERO mensagens: aqui garante-se que o
		// chamador fica explicitamente com as originais.
		return res, append([]api.Message(nil), mensagens...), err
	}
	if cortado {
		res.Motivo += " · ATENÇÃO: o digest foi CORTADO pelo teto (resumo do que coube, não do que existe)"
	}
	return res, res.Mensagens, nil
}
