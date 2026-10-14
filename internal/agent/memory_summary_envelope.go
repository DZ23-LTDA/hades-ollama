package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// ENVELOPE VERSIONADO DE RESUMO E GATILHO POR ORÇAMENTO (complemento universal de
// contexto e memória, §2).
//
// O QUE ESTE FICHEIRO NÃO FAZ, DE PROPÓSITO: não substitui `MemorySummary` nem o
// `DigestSummarizer` do estágio 15, não gera texto, não chama modelo e não cria
// segundo caminho de resumo. Acrescenta por CIMA do resumo existente aquilo que o
// §2 exige e que não existia: versão, hash, fontes cobertas, orçamento com
// estimativa ROTULADA e validação explícita.
//
// A REGRA DE HONESTIDADE HERDADA: o estágio 15 já declara `Truncated` porque
// "silêncio aqui viraria 'resumo do que existe' quando é 'resumo do que coube'".
// Aqui a mesma regra vale para tokens: NÃO EXISTE tokenizer neste runtime, logo
// toda contagem sai com `Exata=false` e um método nomeado. Um número sem rótulo
// seria uma precisão falsa.

// OrcamentoDeTokens é uma contagem de tokens COM a sua origem declarada.
type OrcamentoDeTokens struct {
	Tokens int `json:"tokens"`
	// Exata é SEMPRE false nesta implementação: não há tokenizer compatível
	// disponível no runtime do agente. O campo existe para que uma contagem exata
	// futura possa dizer a verdade sem mudar o contrato.
	Exata  bool   `json:"exata"`
	Metodo string `json:"metodo"`
}

// EstimativaPorCaracteres é o rótulo do método conservador (≈4 caracteres/token,
// o limite prático observado para português e código; arredonda PARA CIMA).
const EstimativaPorCaracteres = "estimativa-por-caracteres-4:1"

// EstimarTokens devolve a contagem conservadora, sempre rotulada como estimativa.
func EstimarTokens(texto string) OrcamentoDeTokens {
	n := len([]rune(texto))
	tokens := (n + 3) / 4
	return OrcamentoDeTokens{Tokens: tokens, Exata: false, Metodo: EstimativaPorCaracteres}
}

// SomarTokens soma orçamentos e mantém o rótulo: a soma de estimativas continua a
// ser uma estimativa, e o método só pode ser declarado exato se TODAS o forem.
func SomarTokens(partes ...OrcamentoDeTokens) OrcamentoDeTokens {
	total := OrcamentoDeTokens{Exata: true, Metodo: EstimativaPorCaracteres}
	for _, p := range partes {
		total.Tokens += p.Tokens
		if !p.Exata {
			total.Exata = false
		}
		if p.Metodo != "" {
			total.Metodo = p.Metodo
		}
	}
	return total
}

// GatilhoDeCompactacao decide QUANDO compactar por ORÇAMENTO — e não apenas depois
// de o modelo recusar o contexto (que é o comportamento que existia).
type GatilhoDeCompactacao struct {
	// OrcamentoTokens é o teto de contexto disponível. Zero significa NÃO
	// DECLARADO, e sem orçamento não há gatilho: inventar um teto seria pior do
	// que declarar que não se sabe.
	OrcamentoTokens int
	// ReservaTokens é a margem para prompt de sistema, resultados de ferramentas e
	// resposta. É subtraída ao teto antes de comparar.
	ReservaTokens int
}

// PrecisaCompactar devolve a decisão E o motivo, sempre. Nunca devolve true sem
// dizer porquê, e nunca devolve false por não ter medido.
func (g GatilhoDeCompactacao) PrecisaCompactar(uso OrcamentoDeTokens) (bool, string) {
	if g.OrcamentoTokens <= 0 {
		return false, "orçamento não declarado: sem teto não há gatilho (não invento um limite)"
	}
	disponivel := g.OrcamentoTokens - g.ReservaTokens
	if disponivel <= 0 {
		return false, fmt.Sprintf("orçamento %d menor/igual à reserva %d: configuração inválida, nada a decidir",
			g.OrcamentoTokens, g.ReservaTokens)
	}
	rotulo := "estimado"
	if uso.Exata {
		rotulo = "exato"
	}
	if uso.Tokens > disponivel {
		return true, fmt.Sprintf("uso %d (%s) acima de %d disponíveis (teto %d − reserva %d)",
			uso.Tokens, rotulo, disponivel, g.OrcamentoTokens, g.ReservaTokens)
	}
	return false, fmt.Sprintf("uso %d (%s) dentro de %d disponíveis", uso.Tokens, rotulo, disponivel)
}

// EnvelopeDeResumo é o resumo do estágio 15 com a rastreabilidade que faltava.
type EnvelopeDeResumo struct {
	Versao int `json:"versao"`
	// Hash é o digest canónico do resumo: cobre a versão, o resumidor, a consulta,
	// o TEXTO e as fontes cobertas. Muda se qualquer um mudar, logo é verificável.
	Hash string `json:"hash"`
	// FontesCobertas são as proveniências das citações, na ordem em que o resumo as
	// numerou. É o que o envelope PODE afirmar: não são "faixas de mensagens" (esse
	// dado não existe em `MemorySummary`) e chamá-las assim seria inventar.
	FontesCobertas []string          `json:"fontes_cobertas"`
	Orcamento      OrcamentoDeTokens `json:"orcamento"`
	// Validacao nomeia cada verificação feita. Lista vazia = nada a apontar.
	Validacao []string `json:"validacao,omitempty"`
}

// VersaoDoEnvelopeDeResumo sobe quando o formato do hash mudar — um hash sem
// versão não permite distinguir "formato diferente" de "conteúdo diferente".
const VersaoDoEnvelopeDeResumo = 1

// ConstruirEnvelope monta o envelope a partir do resumo JÁ produzido. Não gera
// texto: se o resumo estiver vazio, o envelope diz isso na validação em vez de
// esconder.
func ConstruirEnvelope(resumo MemorySummary) EnvelopeDeResumo {
	fontes := make([]string, 0, len(resumo.Citations))
	for _, c := range resumo.Citations {
		fontes = append(fontes, strings.TrimSpace(c.Source))
	}
	e := EnvelopeDeResumo{
		Versao:         VersaoDoEnvelopeDeResumo,
		FontesCobertas: fontes,
		Orcamento:      EstimarTokens(resumo.Text),
	}
	e.Hash = hashDoResumo(resumo, fontes, e.Versao)
	if strings.TrimSpace(resumo.Text) == "" {
		e.Validacao = append(e.Validacao, "texto vazio: o resumo não produziu conteúdo")
	}
	if resumo.Count != len(resumo.Citations) {
		e.Validacao = append(e.Validacao, fmt.Sprintf("contagem declarada %d difere das citações %d",
			resumo.Count, len(resumo.Citations)))
	}
	if resumo.Truncated {
		e.Validacao = append(e.Validacao, "fontes cortadas pelo teto: é resumo do que COUBE, não do que existe")
	}
	if !e.Orcamento.Exata {
		e.Validacao = append(e.Validacao, "tokens são "+e.Orcamento.Metodo+": estimativa, não contagem exata")
	}
	if len(fontes) == 0 && !resumo.Truncated {
		e.Validacao = append(e.Validacao, "sem fontes cobertas: nada a reconstruir a partir do original")
	}
	return e
}

// hashDoResumo usa separadores explícitos e ordem fixa: sem isso, dois resumos
// diferentes poderiam produzir a mesma string concatenada (colisão de fronteira).
func hashDoResumo(resumo MemorySummary, fontes []string, versao int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "v%d\x1f%s\x1f%s\x1f%s\x1f%t\x1f%d",
		versao, resumo.Summarizer, resumo.ProjectID, resumo.Query, resumo.Truncated, len(fontes))
	for _, f := range fontes {
		fmt.Fprintf(&b, "\x1f%s", f)
	}
	b.WriteString("\x1e")
	b.WriteString(resumo.Text)
	soma := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(soma[:])
}
