package agent

import (
	"strings"
	"testing"
)

// Testes do contrato de handoff (§4). Nenhum deles chama modelo, rede ou disco.

func handoffCompleto() Handoff {
	return Handoff{
		ProjetoID:        "proj-1",
		MissaoID:         "missao-7",
		Objetivo:         "entregar a compactacao com recuperacao provada",
		EstadoDoTrabalho: []string{"bruto append-only implementado e testado"},
		Decisoes:         []string{"resumo e' digest deterministico, nao gerado por modelo"},
		Restricoes:       []string{"nao desativar o -race"},
		Pendentes:        []string{"ligar o handoff ao Mission Control"},
		Evidencias:       []string{"E-571..E-585"},
		Orcamento:        OrcamentoDeTokens{Tokens: 1234, Exata: false, Metodo: EstimativaPorCaracteres},
		Autorizacao: AutorizacaoDoHandoff{
			Capacidades:         []string{"workspace:read"},
			AprovacoesPendentes: []string{"deploy em producao"},
		},
	}
}

func TestHandoffCompletoEhValido(t *testing.T) {
	if err := handoffCompleto().Validar(); err != nil {
		t.Fatalf("recusou um handoff completo: %v", err)
	}
}

func TestHandoffRecusaAsCegueirasQueImportam(t *testing.T) {
	casos := []struct {
		nome   string
		mexer  func(*Handoff)
		contem string
	}{
		{"sem objetivo", func(h *Handoff) { h.Objetivo = "   " }, "sem objetivo"},
		{"sem estado do trabalho", func(h *Handoff) { h.EstadoDoTrabalho = nil }, "sem estado do trabalho"},
		{"capacidade vazia", func(h *Handoff) { h.Autorizacao.Capacidades = []string{"  "} }, "capacidade vazia"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			h := handoffCompleto()
			caso.mexer(&h)
			err := h.Validar()
			if err == nil {
				t.Fatal("aceitou um handoff que deixaria a próxima IA às cegas")
			}
			if !strings.Contains(err.Error(), caso.contem) {
				t.Fatalf("recusou por outro motivo: %v", err)
			}
		})
	}
}

// A DISTINÇÃO CENTRAL DESTE CONTRATO: `Pendentes` NIL não é `Pendentes` VAZIO.
// Vazio AFIRMA "nada falta"; nil diz "não declarei". Confundir os dois transforma uma
// omissão numa promessa de conclusão — e é assim que uma missão é declarada terminada
// sem estar.
func TestHandoffDistinguePendentesNaoDeclaradoDeVazio(t *testing.T) {
	naoDeclarado := handoffCompleto()
	naoDeclarado.Pendentes = nil
	err := naoDeclarado.Validar()
	if err == nil {
		t.Fatal("aceitou pendentes NÃO DECLARADOS como se nada faltasse")
	}
	if !strings.Contains(err.Error(), "ausência de declaração") {
		t.Fatalf("motivo inesperado: %v", err)
	}

	// Vazio EXPLÍCITO é válido: afirma que nada falta, e quem o afirma assume-o.
	vazio := handoffCompleto()
	vazio.Pendentes = []string{}
	if err := vazio.Validar(); err != nil {
		t.Fatalf("recusou pendentes vazios EXPLÍCITOS: %v", err)
	}
}

func TestHandoffParaContextoEhDadoMarcadoENaoInstrucao(t *testing.T) {
	texto, cortado := handoffCompleto().ParaContexto(TetoDeDigestPadrao)
	if cortado {
		t.Fatal("cortou um handoff pequeno")
	}
	if !strings.HasPrefix(texto, MarcadorDeHandoff) {
		t.Fatalf("o bloco não começa com o marcador de dado: %q", texto[:60])
	}
	for _, esperado := range []string{
		"entregar a compactacao com recuperacao provada",
		"resumo e' digest deterministico",
		"nao desativar o -race",
		"ligar o handoff ao Mission Control",
		"E-571..E-585",
		"workspace:read",
		"deploy em producao",
	} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("o bloco perdeu %q:\n%s", esperado, texto)
		}
	}
	// O ORÇAMENTO VAI ROTULADO: um número sem rótulo seria precisão falsa.
	if !strings.Contains(texto, "estimados") || !strings.Contains(texto, EstimativaPorCaracteres) {
		t.Fatalf("o orçamento não está rotulado como estimativa:\n%s", texto)
	}
	// E a autorização aparece SEPARADA do contexto, com as aprovações pendentes à vista.
	if !strings.Contains(texto, "Aprovações pendentes") {
		t.Fatal("as aprovações pendentes não estão declaradas no bloco")
	}
}

func TestHandoffDeclaraOCorteEmVezDeTruncarEmSilencio(t *testing.T) {
	h := handoffCompleto()
	// Muitos pendentes para forçar o corte.
	h.Pendentes = nil
	for i := range 200 {
		h.Pendentes = append(h.Pendentes, "pendencia numero "+strconvItoa(i))
	}
	texto, cortado := h.ParaContexto(500)
	if !cortado {
		t.Fatal("não declarou o corte")
	}
	if len([]rune(texto)) < 500 {
		t.Fatalf("o texto devolvido é mais curto do que o teto: %d", len([]rune(texto)))
	}
	// O AVISO TEM DE ESTAR LÁ: um handoff truncado em silêncio faria a próxima IA acreditar
	// que viu tudo.
	if !strings.Contains(texto, "CORTADO") || !strings.Contains(texto, "não mostrado") {
		t.Fatalf("o corte não foi declarado no texto:\n%s", texto)
	}
}

// strconvItoa evita importar strconv só para o teste e não colide com o `itoa` do pacote
// (que vive num ficheiro de teste e por isso NÃO existe no build de produção — a lição do E-572).
func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}
