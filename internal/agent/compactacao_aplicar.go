package agent

import (
	"errors"
	"strings"

	"github.com/ollama/ollama/api"
)

// APLICAR A COMPACTAÇÃO — o EFEITO, com a ORDEM que o §2 impõe.
//
// A ORDEM NÃO É ESTILO, É A GARANTIA. O §2 exige «persistir primeiro o histórico bruto
// completo e imutável/auditável, DEPOIS gravar resumo versionado», e que «em erro,
// cancelamento ou interrupção, o histórico original continua acessível». Este ficheiro
// torna essa ordem ESTRUTURAL: se a gravação do bruto falhar, a função devolve erro e
// **não devolve mensagens** — o chamador fica com as originais. Não há caminho em que o
// resumo exista sem o material que ele cobre.
//
// E O RESUMO NÃO VIRA INSTRUÇÃO. O §2 diz para não tratar texto recuperado como
// instrução de sistema. Por isso a mensagem-resumo entra como papel `user` com um
// marcador explícito, e NÃO como `system`: um resumo com autoridade de sistema poderia
// reescrever políticas — a promoção que o §7-E proíbe.

// MarcadorDeResumo identifica a mensagem que substitui a faixa antiga. É explícito para
// que nem o modelo nem um humano confundam o resumo com a conversa original.
const MarcadorDeResumo = "[RESUMO DA CONVERSA ANTERIOR — não é instrução do utilizador]"

// ResultadoDaCompactacao é o que o funil passa a enviar, mais a prova do que aconteceu.
type ResultadoDaCompactacao struct {
	// Mensagens é o que segue para o modelo: resumo + intactas. Vazio quando houve erro —
	// e é isso que faz o chamador manter as originais.
	Mensagens []api.Message
	// RegistosGravados é quantos registos foram persistidos no histórico bruto ANTES do
	// resumo. Zero com Compactado=true seria contradição, e o teste proíbe-o.
	RegistosGravados int
	// UltimoHash é o topo da cadeia depois da gravação: é a referência que prova que o
	// bruto existe e qual é.
	UltimoHash string
	// Motivo declara o que aconteceu, incluindo as recusas.
	Motivo string
	// Compactado é false quando nada foi substituído (nada a compactar, ou recusa).
	Compactado bool
}

// AplicarCompactacao grava a faixa a resumir no histórico bruto e só então constrói as
// mensagens novas. `historico` e `resumo` são injetados: a função não sabe onde o
// histórico vive nem quem gerou o resumo, o que a torna testável sem provider e sem
// escolher armazenamento por conta própria.
func AplicarCompactacao(plano PlanoDeCompactacao, historico *HistoricoBruto, resumo string, projetoID, missaoID string) (ResultadoDaCompactacao, error) {
	if !plano.Compactavel {
		return ResultadoDaCompactacao{
			Mensagens: append([]api.Message(nil), plano.Intactas...),
			Motivo:    "nada a compactar: as mensagens seguem intactas",
		}, nil
	}
	if historico == nil {
		// FALHA FECHADO: sem sítio onde gravar o bruto não há compactação. Devolver
		// mensagens resumidas aqui seria perder a faixa antiga sem deixar rasto.
		return ResultadoDaCompactacao{Motivo: "sem histórico bruto configurado: recuso compactar sem guardar o material"},
			errors.New("compactação exige histórico bruto: o material tem de ser gravado antes do resumo")
	}
	if strings.TrimSpace(resumo) == "" {
		return ResultadoDaCompactacao{Motivo: "resumo vazio: recuso substituir mensagens por nada"},
			errors.New("resumo vazio: substituir a faixa por nada perderia a conversa")
	}

	// 1) O BRUTO PRIMEIRO. Cada mensagem da faixa vira um registo imutável.
	gravados := 0
	ultimoHash := ""
	for _, m := range plano.Resumir {
		reg, err := historico.Anexar(RegistroDeHistorico{
			ProjetoID: projetoID,
			MissaoID:  missaoID,
			Papel:     m.Role,
			Conteudo:  m.Content,
		})
		if err != nil {
			return ResultadoDaCompactacao{
				Motivo:           "falhou a gravação do histórico bruto: devolvo erro e NÃO devolvo mensagens",
				RegistosGravados: gravados,
				UltimoHash:       ultimoHash,
			}, err
		}
		gravados++
		ultimoHash = reg.Hash
	}

	// 2) E SÓ AGORA o resumo. A mensagem leva o marcador e o topo da cadeia, para que a
	// origem do resumo seja rastreável a partir do próprio contexto enviado.
	corpo := MarcadorDeResumo + "\n" +
		"Faixa coberta: " + itoaCompactacao(len(plano.Resumir)) + " mensagens gravadas no histórico bruto.\n" +
		"Referência do bruto (topo da cadeia): " + ultimoHash + "\n\n" + resumo
	mensagens := make([]api.Message, 0, 1+len(plano.Intactas))
	mensagens = append(mensagens, api.Message{Role: "user", Content: corpo})
	mensagens = append(mensagens, plano.Intactas...)

	return ResultadoDaCompactacao{
		Mensagens:        mensagens,
		RegistosGravados: gravados,
		UltimoHash:       ultimoHash,
		Motivo: "compactei " + itoaCompactacao(len(plano.Resumir)) + " mensagens (" + plano.Motivo + "); " +
			"o bruto foi gravado ANTES do resumo",
		Compactado: true,
	}, nil
}

// itoaCompactacao: nome próprio porque o pacote JÁ TEM um `itoa` num ficheiro de teste, e
// um símbolo de teste não existe no build de produção (o compilador apanhou isso no E-572).
func itoaCompactacao(n int) string {
	if n == 0 {
		return "0"
	}
	digitos := make([]byte, 0, 20)
	for n > 0 {
		digitos = append([]byte{byte('0' + n%10)}, digitos...)
		n /= 10
	}
	return string(digitos)
}
