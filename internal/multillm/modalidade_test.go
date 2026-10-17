package multillm

import "testing"

// ***E-431 - TESTE DE CONTRATO DA MATRIZ DE MODALIDADE x ENGINE.***
//
// A matriz so' vale se o **contrato** que ela promete for verificavel. Este ficheiro testa o **contrato**, e **nao** a
// tabela: *um teste que repetisse a tabela so' provaria que copiei duas vezes.*
//
// ⇒ Os quatro invariantes que interessam:
//  1. **toda** modalidade declarada tem capacidade **nao vazia** - senao o `Registry` compara com nada;
//  2. **toda** modalidade tem **pelo menos um** motor - *uma modalidade que ninguem produz e' um botao que nao faz nada*;
//  3. **nao ha** par (modalidade, capacidade) **duplicado** - *duplicado e' duas fontes de verdade na mesma tabela*;
//  4. `ModalidadeCompativel` pergunta **OU**, e recusa o que nao tem a capacidade - *e' o E-430*.

func TestMatrizDeModalidadesRespeitaOsInvariantes(t *testing.T) {
	vistos := map[string]bool{}
	porModalidade := map[Modalidade]int{}

	for _, e := range MatrizDeModalidades() {
		if e.Modalidade == "" {
			t.Fatalf("entrada com modalidade vazia: %+v", e)
		}
		if e.Capacidade == "" {
			t.Fatalf("ARMADILHA: a modalidade %q tem capacidade VAZIA - *o Registry compararia com nada, e a "+
				"promessa falharia em silencio*", e.Modalidade)
		}
		if len(e.Motores) == 0 {
			t.Fatalf("ARMADILHA: a modalidade %q nao tem NENHUM motor - *uma modalidade que ninguem produz e' um "+
				"botao que nao faz nada*", e.Modalidade)
		}
		if e.Observacao == "" {
			t.Fatalf("ARMADILHA: a modalidade %q nao declara limite - *uma matriz sem limites e' publicidade*",
				e.Modalidade)
		}
		chave := string(e.Modalidade) + "|" + e.Capacidade
		if vistos[chave] {
			t.Fatalf("ARMADILHA: par (modalidade, capacidade) DUPLICADO: %q - *duplicado e' duas fontes de verdade "+
				"na mesma tabela*", chave)
		}
		vistos[chave] = true
		porModalidade[e.Modalidade]++
	}

	// O ponto 2, medido por modalidade (e nao por linha): garante que **cada** modalidade tem motor.
	for _, m := range []Modalidade{ModalidadeTexto, ModalidadeImagem, ModalidadeVideo, ModalidadeAudio, ModalidadeDocumento} {
		if porModalidade[m] == 0 {
			t.Fatalf("ARMADILHA: a modalidade %q existe como constante mas NAO esta' na matriz - *a constante "+
				"promete, a matriz nao cumpre*", m)
		}
		capacidades, ok := CapacidadesParaModalidade(m)
		if !ok || len(capacidades) == 0 {
			t.Fatalf("modalidade %q sem capacidades", m)
		}
	}
}

func TestCapacidadesParaModalidadeTraduzOsDoisSentidosDaImagem(t *testing.T) {
	// `image` e' ENTRADA (`vision`) e SAIDA (`image`). *Confundir os dois e' o defeito que o E-430 impede.*
	capacidades, ok := CapacidadesParaModalidade(ModalidadeImagem)
	if !ok {
		t.Fatal("modalidade image nao encontrada")
	}
	temVisao, temGeracao := false, false
	for _, c := range capacidades {
		switch c {
		case "vision":
			temVisao = true
		case "image":
			temGeracao = true
		}
	}
	if !temVisao || !temGeracao {
		t.Fatalf("ARMADILHA: `image` tinha de cobrir os DOIS sentidos (vision e image); veio %v", capacidades)
	}

	if _, ok := CapacidadesParaModalidade(Modalidade("modalidade-que-nao-existe")); ok {
		t.Fatal("ARMADILHA: modalidade inexistente devolveu capacidades - *um instrumento que responde a tudo nao " +
			"distingue nada*")
	}
}

func TestModalidadeCompativelPerguntaOuENaoE(t *testing.T) {
	// (a) CONTROL: um modelo que so' ve' NAO gera imagem.
	soVisao := Model{ID: "local/so-visao", Capabilities: []string{"vision", "text"}}
	if !ModalidadeCompativel(ModalidadeImagem, soVisao) {
		t.Fatal("um modelo com `vision` tinha de satisfazer a modalidade `image` (entrada) - *isto e' o OU*")
	}

	// (b) O E-430: **video para modelo textual** tem de ser RECUSADO, e nao atendido com sucesso falso.
	textual := Model{ID: "local/textual", Capabilities: []string{"text"}}
	if ModalidadeCompativel(ModalidadeVideo, textual) {
		t.Fatal("ARMADILHA (E-430): um modelo TEXTUAL foi dado como compativel com VIDEO - *e' o sucesso falso que o " +
			"complemento proibe: 'nao mandar pedido de video para modelo textual e afirmar sucesso'*")
	}
	if ModalidadeCompativel(ModalidadeAudio, textual) {
		t.Fatal("ARMADILHA (E-430): um modelo textual foi dado como compativel com AUDIO")
	}
	if ModalidadeCompativel(ModalidadeDocumento, textual) {
		t.Fatal("ARMADILHA (E-430): um modelo textual foi dado como compativel com DOCUMENTO")
	}

	// (c) modalidade inexistente nunca e' compativel.
	if ModalidadeCompativel(Modalidade("nao-existe"), textual) {
		t.Fatal("modalidade inexistente devolveu compativel")
	}
}

// TestAMatrizNaoEPartilhadaPorReferencia e' a contraprova do `MatrizDeModalidades`: *uma fonte de verdade que o
// consumidor edita nao e' fonte de verdade.*
func TestAMatrizNaoEPartilhadaPorReferencia(t *testing.T) {
	primeira := MatrizDeModalidades()
	if len(primeira) == 0 {
		t.Fatal("matriz vazia")
	}
	originalCap := primeira[0].Capacidade
	originalMotor := primeira[0].Motores[0]

	primeira[0].Capacidade = "ADULTERADO"
	primeira[0].Motores[0] = "ADULTERADO"

	segunda := MatrizDeModalidades()
	if segunda[0].Capacidade != originalCap || segunda[0].Motores[0] != originalMotor {
		t.Fatalf("ARMADILHA: editar a copia mudou a matriz (%q/%q) - *a fonte de verdade esta' a ser partilhada por "+
			"referencia*", segunda[0].Capacidade, segunda[0].Motores[0])
	}
}
