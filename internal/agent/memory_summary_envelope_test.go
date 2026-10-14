package agent

import (
	"strings"
	"testing"
)

// Testes do envelope de resumo e do gatilho por orçamento (complemento de memória, §2).
// Cada teste afirma UMA propriedade; nenhum deles chama modelo, rede ou disco.

func TestEstimativaDeTokensEhSempreRotulada(t *testing.T) {
	// A PROPRIEDADE QUE IMPORTA: nunca sai um número sem dizer que é estimativa.
	vazio := EstimarTokens("")
	if vazio.Tokens != 0 || vazio.Exata || vazio.Metodo != EstimativaPorCaracteres {
		t.Fatalf("vazio: %+v", vazio)
	}
	um := EstimarTokens("a")
	quatro := EstimarTokens("abcd")
	cinco := EstimarTokens("abcde")
	if um.Tokens != 1 || quatro.Tokens != 1 {
		t.Fatalf("arredondamento para cima: 'a'=%d 'abcd'=%d", um.Tokens, quatro.Tokens)
	}
	if cinco.Tokens != 2 {
		t.Fatalf("'abcde' devia subir para 2 tokens (arredonda para cima), deu %d", cinco.Tokens)
	}
	if um.Exata || cinco.Exata {
		t.Fatal("estimativa marcada como exata: precisão falsa")
	}
	// Monotonicidade: texto maior nunca pode dar menos tokens.
	longo := EstimarTokens(strings.Repeat("x", 400))
	if longo.Tokens <= cinco.Tokens {
		t.Fatalf("monotonicidade quebrada: 400 chars=%d <= 5 chars=%d", longo.Tokens, cinco.Tokens)
	}
}

func TestSomarTokensPropagaORotulo(t *testing.T) {
	soma := SomarTokens(
		OrcamentoDeTokens{Tokens: 10, Exata: true, Metodo: "exato"},
		EstimarTokens("abcd"),
	)
	if soma.Exata {
		t.Fatal("soma com uma estimativa foi declarada EXATA")
	}
	if soma.Tokens != 11 {
		t.Fatalf("soma = %d, esperado 11", soma.Tokens)
	}
	todas := SomarTokens(
		OrcamentoDeTokens{Tokens: 3, Exata: true, Metodo: "exato"},
		OrcamentoDeTokens{Tokens: 4, Exata: true, Metodo: "exato"},
	)
	if !todas.Exata || todas.Tokens != 7 {
		t.Fatalf("soma de exatos = %+v", todas)
	}
}

func TestGatilhoNaoInventaOrcamento(t *testing.T) {
	// FALHA FECHADO: sem teto declarado, a resposta é "não sei", não "compacta".
	g := GatilhoDeCompactacao{}
	decidir, motivo := g.PrecisaCompactar(EstimarTokens(strings.Repeat("x", 100000)))
	if decidir || !strings.Contains(motivo, "não declarado") {
		t.Fatalf("sem orçamento devia recusar a decidir; veio decidir=%v motivo=%q", decidir, motivo)
	}
	// Configuração inválida (reserva >= teto) também não pode virar um "true" silencioso.
	invalido := GatilhoDeCompactacao{OrcamentoTokens: 1000, ReservaTokens: 1000}
	if decidir, motivo := invalido.PrecisaCompactar(EstimarTokens("qualquer")); decidir || !strings.Contains(motivo, "inválida") {
		t.Fatalf("config inválida: decidir=%v motivo=%q", decidir, motivo)
	}
}

func TestGatilhoDecidePorOrcamentoEExplica(t *testing.T) {
	g := GatilhoDeCompactacao{OrcamentoTokens: 1000, ReservaTokens: 200}
	// 801 tokens > 800 disponíveis ⇒ compacta, e o motivo traz os NÚMEROS.
	acima := OrcamentoDeTokens{Tokens: 801, Exata: false, Metodo: EstimativaPorCaracteres}
	decidir, motivo := g.PrecisaCompactar(acima)
	if !decidir || !strings.Contains(motivo, "801") || !strings.Contains(motivo, "800") {
		t.Fatalf("acima: decidir=%v motivo=%q", decidir, motivo)
	}
	if !strings.Contains(motivo, "estimado") {
		t.Fatalf("o motivo não declara que a contagem é estimada: %q", motivo)
	}
	// Exactamente no limite NÃO compacta (o teto é inclusivo) — e o motivo diz dentro.
	// NOTA: `if decidir, motivo := ...` ESCOPA as variáveis ao `if`. A primeira versão
	// deste teste afirmou sobre a variável de FORA (a da chamada anterior) e passou a
	// ler o motivo errado — o compilador não apanha isso, a asserção apanhou.
	dentro := OrcamentoDeTokens{Tokens: 800, Exata: true, Metodo: "exato"}
	decidirDentro, motivoDentro := g.PrecisaCompactar(dentro)
	if decidirDentro || !strings.Contains(motivoDentro, "dentro") {
		t.Fatalf("no limite: decidir=%v motivo=%q", decidirDentro, motivoDentro)
	}
	if !strings.Contains(motivoDentro, "exato") {
		t.Fatalf("uso exato devia ser rotulado como exato: %q", motivoDentro)
	}
}

func TestEnvelopeEhDeterministicoEDistingueConteudo(t *testing.T) {
	base := MemorySummary{
		ProjectID: "p1", Query: "estoque", Summarizer: "digest-deterministico",
		Text: "trecho relevante", Count: 2,
		Citations: []Citation{{Index: 1, Source: "docs/a.md#0"}, {Index: 2, Source: "docs/b.md#3"}},
	}
	a := ConstruirEnvelope(base)
	b := ConstruirEnvelope(base)
	if a.Hash != b.Hash {
		t.Fatalf("mesma entrada deu hashes diferentes: %s vs %s", a.Hash, b.Hash)
	}
	if a.Versao != VersaoDoEnvelopeDeResumo || len(a.Hash) != 64 {
		t.Fatalf("envelope malformado: versao=%d hash=%q", a.Versao, a.Hash)
	}
	if len(a.FontesCobertas) != 2 || a.FontesCobertas[0] != "docs/a.md#0" {
		t.Fatalf("fontes cobertas = %v", a.FontesCobertas)
	}
	alterado := base
	alterado.Text = "trecho DIFERENTE"
	if ConstruirEnvelope(alterado).Hash == a.Hash {
		t.Fatal("mudar o TEXTO não mudou o hash: o hash não cobre o conteúdo")
	}
	outraFonte := base
	outraFonte.Citations = []Citation{{Index: 1, Source: "docs/a.md#0"}, {Index: 2, Source: "docs/c.md#9"}}
	if ConstruirEnvelope(outraFonte).Hash == a.Hash {
		t.Fatal("mudar a FONTE não mudou o hash: o hash não cobre a proveniência")
	}
	// Fronteira: sem separadores, ("ab","c") e ("a","bc") colidiriam.
	f1 := base
	f1.Citations = []Citation{{Index: 1, Source: "ab"}, {Index: 2, Source: "c"}}
	f2 := base
	f2.Citations = []Citation{{Index: 1, Source: "a"}, {Index: 2, Source: "bc"}}
	if ConstruirEnvelope(f1).Hash == ConstruirEnvelope(f2).Hash {
		t.Fatal("colisão de fronteira entre fontes concatenadas")
	}
}

func TestEnvelopeDeclaraOQueNaoEstaBem(t *testing.T) {
	vazio := ConstruirEnvelope(MemorySummary{Summarizer: "digest-deterministico"})
	if len(vazio.Validacao) == 0 {
		t.Fatal("resumo vazio passou sem nenhuma validação")
	}
	juntas := strings.Join(vazio.Validacao, " | ")
	if !strings.Contains(juntas, "texto vazio") {
		t.Fatalf("validação não nomeia o texto vazio: %q", juntas)
	}
	if !strings.Contains(juntas, "estimativa") {
		t.Fatalf("validação não declara que os tokens são estimativa: %q", juntas)
	}

	// Truncated tem de aparecer: silêncio viraria "resumo do que existe".
	cortado := ConstruirEnvelope(MemorySummary{
		Summarizer: "digest-deterministico", Text: "x", Count: 1,
		Citations: []Citation{{Index: 1, Source: "docs/a.md#0"}}, Truncated: true,
	})
	if !strings.Contains(strings.Join(cortado.Validacao, " | "), "COUBE") {
		t.Fatalf("Truncated não foi declarado: %v", cortado.Validacao)
	}

	// Contagem divergente das citações é incoerência e tem de ser dita.
	divergente := ConstruirEnvelope(MemorySummary{
		Summarizer: "digest-deterministico", Text: "x", Count: 5,
		Citations: []Citation{{Index: 1, Source: "docs/a.md#0"}},
	})
	if !strings.Contains(strings.Join(divergente.Validacao, " | "), "difere das citações") {
		t.Fatalf("contagem divergente não foi declarada: %v", divergente.Validacao)
	}
}
