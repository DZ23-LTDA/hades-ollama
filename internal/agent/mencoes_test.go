package agent

import "testing"

// Complemento 48 / FASE 2 — o PARSER de mencoes. *Nao executa nada: diz quem foi mencionado.*
// Quem decide se executa e' a politica (autorizacao + capacidade + orcamento), e essa separacao e' o que
// impede uma mencao de virar um comando.

func nomesDasMencoes(mencoes []Mencao) []string {
	nomes := make([]string, 0, len(mencoes))
	for _, mencao := range mencoes {
		nomes = append(nomes, mencao.Nome)
	}
	return nomes
}

func TestMencaoIndividualEhExtraida(t *testing.T) {
	mencoes := MencoesNoTexto("Prompts: @ana, pode revisar o handoff?")
	if len(mencoes) != 1 || mencoes[0].Nome != "ana" {
		t.Fatalf("esperava uma mencao 'ana', veio %#v", mencoes)
	}
	if mencoes[0].Coletiva {
		t.Fatal("'ana' NAO e' mencao coletiva")
	}
}

func TestMencoesColetivasSaoMarcadas(t *testing.T) {
	mencoes := MencoesNoTexto("@todos e @equipe: a janela fecha hoje")
	if len(mencoes) != 2 {
		t.Fatalf("esperava 2 mencoes, veio %#v", mencoes)
	}
	for _, mencao := range mencoes {
		if !mencao.Coletiva {
			t.Fatalf("%q devia ser marcada como COLETIVA (acorda o time inteiro)", mencao.Nome)
		}
	}
}

func TestEmailNaoEhMencao(t *testing.T) {
	// *Sem esta regra, todo endereco de e-mail acordaria um agente imaginario.*
	if mencoes := MencoesNoTexto("manda para ana@exemplo.com, por favor"); len(mencoes) != 0 {
		t.Fatalf("e-mail nao pode virar mencao, veio %#v", mencoes)
	}
	// E o caso misto: mencao de verdade ANTES do e-mail - so' a mencao conta.
	mencoes := MencoesNoTexto("@ana, confirma o envio para bob@exemplo.com")
	if len(mencoes) != 1 || mencoes[0].Nome != "ana" {
		t.Fatalf("esperava so' 'ana', veio %#v", mencoes)
	}
}

func TestPontuacaoNaoEngordaONome(t *testing.T) {
	mencoes := MencoesNoTexto("(@ana), e depois @bob.")
	nomes := nomesDasMencoes(mencoes)
	if len(nomes) != 2 || nomes[0] != "ana" || nomes[1] != "bob" {
		t.Fatalf("a pontuacao tem de ficar de fora do nome, veio %#v", nomes)
	}
}

func TestMencaoRepetidaContaUmaVezEAMaisculaNaoDuplica(t *testing.T) {
	// A primeira ocorrencia vence (ordem reproduzivel) e a comparacao nao depende de maiusculas.
	mencoes := MencoesNoTexto("@Ana revisa e @ana aprova e @ANA entrega")
	if len(mencoes) != 1 || mencoes[0].Nome != "ana" {
		t.Fatalf("@Ana/@ana/@ANA sao a MESMA mencao, veio %#v", nomesDasMencoes(mencoes))
	}
}

func TestArrobaSoltaNaoEhMencao(t *testing.T) {
	for _, corpo := range []string{"@", "custo @ 100 reais", "   ", ""} {
		if mencoes := MencoesNoTexto(corpo); len(mencoes) != 0 {
			t.Fatalf("%q nao pode produzir mencao, veio %#v", corpo, mencoes)
		}
	}
}

func TestNomeComAcentoEhNome(t *testing.T) {
	mencoes := MencoesNoTexto("@joão e @maria_clara e @dev-1")
	nomes := nomesDasMencoes(mencoes)
	if len(nomes) != 3 || nomes[0] != "joão" || nomes[1] != "maria_clara" || nomes[2] != "dev-1" {
		t.Fatalf("acento, sublinhado e hifen fazem parte do nome, veio %#v", nomes)
	}
}

func TestMencoesDeAgentesSeparamOColetivo(t *testing.T) {
	mencoes := MencoesNoTexto("@todos, o @ana e o @bob seguem; @equipe tambem")
	agentes := nomesDasMencoes(MencoesDeAgentes(mencoes))
	if len(agentes) != 2 || agentes[0] != "ana" || agentes[1] != "bob" {
		t.Fatalf("as coletivas nao podem entrar nas mencoes de agentes, veio %#v", agentes)
	}
}
