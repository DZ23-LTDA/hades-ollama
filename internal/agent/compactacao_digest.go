package agent

import (
	"strings"

	"github.com/ollama/ollama/api"
)

// RESUMO DETERMINÍSTICO DA FAIXA (§2) — sem modelo, e por isso sem invenção.
//
// PORQUE NÃO UM LLM AQUI: o §2 proíbe perder o objectivo e o §7-E proíbe promover texto
// recuperado a instrução. Um resumo gerado por modelo pode afirmar o que não está na
// conversa — e o Estágio 15 já resolveu o mesmo problema do lado da memória com um
// digest determinístico («se ele diz algo, aquilo está na memória; se não encontra, DIZ
// que não encontrou»). Este é o equivalente para mensagens: **copia o material coberto**,
// com papel e ordem, e **declara o corte** quando não cabe.
//
// O QUE ELE NÃO É: não é compactação inteligente e não finge ser. É redução fiel —
// "resumo do que COUBE, não do que existe", a mesma honestidade que o `Truncated` do
// Estágio 15 trava por teste.
//
// E O RESUMO POR MODELO CONTINUA POSSÍVEL: este ficheiro não fecha a porta; a interface
// `MemorySummarizer` do Estágio 15 já existe para quem quiser declarar um. Aqui o padrão
// é o determinístico, e a origem do texto vai declarada na própria mensagem (E-573).

// TetoDeDigestPadrao limita o resumo determinístico. Não é um valor sagrado: é um ponto
// de partida declarado, e o teste fixa o comportamento.
const TetoDeDigestPadrao = 4000

// ResumirPorDigest devolve o digest fiel da faixa e se houve corte. `teto` é em runas;
// zero ou negativo usa o padrão.
func ResumirPorDigest(mensagens []api.Message, teto int) (string, bool) {
	if teto <= 0 {
		teto = TetoDeDigestPadrao
	}
	var b strings.Builder
	cortado := false
	for i, m := range mensagens {
		papel := strings.TrimSpace(m.Role)
		if papel == "" {
			papel = "desconhecido"
		}
		linha := "- [" + papel + " " + itoaCompactacao(i+1) + "] " + strings.TrimSpace(m.Content) + "\n"
		// O CORTE É POR MENSAGEM INTEIRA, e não a meio de uma: um resumo que termina numa
		// frase truncada faz o modelo completar o que falta — que é inventar.
		if len([]rune(b.String()))+len([]rune(linha)) > teto {
			cortado = true
			break
		}
		b.WriteString(linha)
	}
	texto := strings.TrimRight(b.String(), "\n")
	if texto == "" && len(mensagens) > 0 {
		// Nada coube: é honesto dizer isso em vez de devolver vazio (o chamador recusa
		// resumo vazio e manteria a conversa inteira — que é a decisão certa).
		return "", true
	}
	return texto, cortado
}
