package agent

import (
	"fmt"
	"strconv"
	"strings"
)

// RENDERIZAÇÃO DO HANDOFF PARA O PREÂMBULO — REDIGIDA (E-597).
//
// PORQUE ESTA FUNÇÃO EXISTE, E O QUE ELA CONSERTA. O `ParaContexto` (E-588) foi desenhado
// como documento para humanos e agentes: leva o ID da missão, o ID do projeto, a referência
// do histórico bruto e o CAMINHO do workspace. Ligá-lo ao preâmbulo que vai ao modelo fez um
// teste existente falhar com a razão exacta:
//
//	preamble leaked "mis_96544b01-…"
//	preamble leaked "prj_411af0cd-…"
//	preamble leaked "C:\…\_tmp\gotmp\TestMissionC…"
//
// O TESTE ESTAVA CERTO E A LIGAÇÃO ESTAVA ERRADA. Identificador interno e caminho local não
// são conteúdo de trabalho: são topologia do servidor, e um modelo externo não precisa deles
// para continuar a missão. Pior: um caminho local revela a estrutura da máquina, e um ID
// interno pode ser usado para tentar aceder a outra coisa.
//
// A REGRA, EM UMA LINHA: ***o preâmbulo leva o que a próxima IA precisa para CONTINUAR e não
// leva o que só serve para LOCALIZAR.*** Por isso esta função é uma LISTA BRANCA — campos
// novos no contrato não entram aqui por omissão; entrar exige uma decisão explícita.
// É o oposto de uma lista negra, que envelhece mal: a lista negra esquece-se de um campo e
// vaza-o; a lista branca esquece-se de um campo e omite-o.

// ParaPreambulo devolve o bloco do handoff REDIGIDO para ir ao modelo. `teto` é em runas;
// zero ou negativo usa o teto do digest.
func (h Handoff) ParaPreambulo(teto int) (string, bool) {
	if teto <= 0 {
		teto = TetoDeDigestPadrao
	}
	var b strings.Builder
	b.WriteString(MarcadorDeHandoff + "\n")

	// LISTA BRANCA, e nada mais. Repare-se no que NÃO está aqui: `MissaoID`, `ProjetoID`,
	// `ReferenciaDoHistorico` e `Evidencias` (que contêm o ID da missão).
	if objetivo := strings.TrimSpace(h.Objetivo); objetivo != "" {
		b.WriteString("Objetivo: " + objetivo + "\n")
	}
	escreverLista := func(rotulo string, itens []string) {
		if len(itens) == 0 {
			return
		}
		b.WriteString(rotulo + ":\n")
		for _, item := range itens {
			b.WriteString("  - " + strings.TrimSpace(item) + "\n")
		}
	}
	escreverLista("Estado do trabalho", h.EstadoDoTrabalho)
	escreverLista("Pendentes", h.Pendentes)

	// RESTRIÇÕES: as operacionais viajam, as que carregam CAMINHO não. A informação útil
	// («não sair do workspace») mantém-se; o endereço da pasta não.
	if restricoes := restricoesSemCaminho(h.Restricoes); len(restricoes) > 0 {
		escreverLista("Restrições", restricoes)
	}

	// AUTORIZAÇÃO: capacidades e aprovações são o que a próxima IA PODE e o que ESPERA.
	escreverLista("Capacidades autorizadas", h.Autorizacao.Capacidades)
	escreverLista("Aprovações pendentes (exigem decisão humana)", h.Autorizacao.AprovacoesPendentes)

	if h.Orcamento.Tokens > 0 {
		rotulo := "estimados"
		if h.Orcamento.Exata {
			rotulo = "exatos"
		}
		b.WriteString("Orçamento deste contexto: " + strconv.Itoa(h.Orcamento.Tokens) + " tokens (" +
			rotulo + ", método " + h.Orcamento.Metodo + ")\n")
	}

	texto := strings.TrimRight(b.String(), "\n")
	runas := []rune(texto)
	if len(runas) <= teto {
		return texto, false
	}
	return strings.TrimRight(string(runas[:teto]), " \n") +
		fmt.Sprintf("\n*** ATENÇÃO: handoff CORTADO pelo teto de %d caracteres ***", teto), true
}

// restricoesSemCaminho remove as restrições que carregam caminho local. A decisão é por
// CONTEÚDO e deliberadamente conservadora: na dúvida, omite — e diz que omitiu, porque uma
// omissão silenciosa faria a próxima IA acreditar que viu todas as restrições.
func restricoesSemCaminho(restricoes []string) []string {
	seguras := make([]string, 0, len(restricoes))
	omitidas := 0
	for _, r := range restricoes {
		texto := strings.TrimSpace(r)
		if texto == "" {
			continue
		}
		// Um caminho local tem separador ou letra de unidade. As restrições operacionais
		// que interessam («não sair dele», «ramo git: X») não têm nenhum dos dois.
		if restricaoTemCaminho(texto) {
			omitidas++
			continue
		}
		seguras = append(seguras, texto)
	}
	if omitidas > 0 {
		seguras = append(seguras, fmt.Sprintf("(%d restrição(ões) com caminho local omitida(s) no preâmbulo)", omitidas))
	}
	return seguras
}

// restricaoTemCaminho decide se uma restrição carrega caminho local.
//
// A PRIMEIRA VERSÃO DESTA FUNÇÃO USAVA «tem separador ⇒ é caminho» E ESTAVA ERRADA: omitiu
// «ramo git: fix/checker-relogio», que é um nome de ramo legítimo com barras. *Um critério
// que confunde ramo com pasta remove informação útil em nome da segurança* — e a correcção
// não é afrouxar a segurança, é olhar para a FORMA do caminho em vez do separador solto:
//   - etiquetas que sabidamente carregam caminho (workspace, caminho, pasta, diretório);
//   - unidade do Windows (`X:\` ou `X:/`);
//   - início absoluto (`/…` ou `\…`);
//   - raízes conhecidas no meio (`/home/`, `/Users/`, `/tmp/`, `_tmp\`, `_tmp/`).
//
// Um ramo com barras não tem nenhuma destas formas.
func restricaoTemCaminho(texto string) bool {
	minusculo := strings.ToLower(texto)
	for _, etiqueta := range []string{"workspace:", "caminho:", "pasta:", "diretório:", "diretorio:"} {
		if strings.HasPrefix(minusculo, etiqueta) {
			return true
		}
	}
	if len(texto) >= 3 && texto[1] == ':' && (texto[2] == '\\' || texto[2] == '/') {
		return true
	}
	if strings.HasPrefix(texto, "/") || strings.HasPrefix(texto, "\\") {
		return true
	}
	for _, raiz := range []string{"/home/", "/Users/", "/tmp/", "_tmp\\", "_tmp/", "\\Users\\"} {
		if strings.Contains(texto, raiz) {
			return true
		}
	}
	return false
}
