package agent

import (
	"strings"
	"testing"
)

// Testes do preâmbulo REDIGIDO (E-597). O objecto é a AUSÊNCIA: identificadores internos e
// caminhos locais não podem chegar ao modelo. Nasceu de um vazamento medido:
//   preamble leaked "mis_96544b01-…" · "prj_411af0cd-…" · "C:\…\_tmp\gotmp\TestMissionC…"

func handoffComSegredosDeTopologia() Handoff {
	return Handoff{
		ProjetoID:             "prj_411af0cd-f38c-445d-bbb8-051744fff4ae",
		MissaoID:              "mis_96544b01-04f6-4703-b8a2-19cd3e59fad5",
		Objetivo:              "entregar a compactação com recuperação provada",
		EstadoDoTrabalho:      []string{"bruto append-only implementado"},
		Pendentes:             []string{"ligar o funil", "publicar ***EXIGE APROVAÇÃO***"},
		Restricoes:            []string{"workspace: C:\\Users\\DZ23\\_tmp\\ws", "ramo git: fix/checker-relogio", "workspace isolado: não sair dele"},
		ReferenciaDoHistorico: "C:\\Users\\DZ23\\_tmp\\historico\\proj-1.jsonl",
		Evidencias:            []string{"missão mis_96544b01-04f6-4703-b8a2-19cd3e59fad5 (versão 7)"},
		Orcamento:             OrcamentoDeTokens{Tokens: 1234, Metodo: EstimativaPorCaracteres},
		Autorizacao: AutorizacaoDoHandoff{
			Capacidades:         []string{"workspace:read"},
			AprovacoesPendentes: []string{"deploy em produção"},
		},
	}
}

func TestParaPreambuloNaoLevaIdentificadoresNemCaminhos(t *testing.T) {
	h := handoffComSegredosDeTopologia()
	texto, _ := h.ParaPreambulo(TetoDeDigestPadrao)

	// (1) IDENTIFICADORES INTERNOS: nenhum, em NENHUMA forma.
	for _, proibido := range []string{
		h.MissaoID, h.ProjetoID,
		"mis_96544b01", "prj_411af0cd",
	} {
		if strings.Contains(texto, proibido) {
			t.Fatalf("o preâmbulo vazou o identificador %q:\n%s", proibido, texto)
		}
	}
	// (2) CAMINHOS LOCAIS: nenhum. E o rótulo "Projeto"/"Missão" nem aparece.
	for _, proibido := range []string{`C:\`, "_tmp", "historico", "Projeto:", "Missão:"} {
		if strings.Contains(texto, proibido) {
			t.Fatalf("o preâmbulo vazou topologia (%q):\n%s", proibido, texto)
		}
	}
	// (3) E DIZ QUE OMITIU: uma omissão silenciosa faria a próxima IA acreditar que viu
	// todas as restrições.
	if !strings.Contains(texto, "omitida") {
		t.Fatalf("não declarou as restrições omitidas:\n%s", texto)
	}
}

func TestParaPreambuloMantemOQueServeParaContinuar(t *testing.T) {
	texto, cortado := handoffComSegredosDeTopologia().ParaPreambulo(TetoDeDigestPadrao)
	if cortado {
		t.Fatal("cortou um handoff pequeno")
	}
	if !strings.HasPrefix(texto, MarcadorDeHandoff) {
		t.Fatal("não começa com o marcador de dado (§7-E)")
	}
	// O CONTEÚDO ÚTIL TEM DE ESTAR LÁ: sem isto a «correcção» teria removido o handoff.
	for _, esperado := range []string{
		"entregar a compactação com recuperação provada",
		"bruto append-only implementado",
		"ligar o funil",
		"EXIGE APROVAÇÃO",
		"ramo git: fix/checker-relogio", // restrição sem caminho: viaja
		"workspace:read",
		"deploy em produção",
		"estimados", // orçamento rotulado
	} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("o preâmbulo perdeu conteúdo útil %q:\n%s", esperado, texto)
		}
	}
}

func TestParaPreambuloDeclaraOCorte(t *testing.T) {
	h := handoffComSegredosDeTopologia()
	h.Pendentes = nil
	for i := range 300 {
		h.Pendentes = append(h.Pendentes, "pendência número "+strconvItoa(i))
	}
	texto, cortado := h.ParaPreambulo(400)
	if !cortado || !strings.Contains(texto, "CORTADO") {
		t.Fatalf("o corte não foi declarado (cortado=%v):\n%s", cortado, texto)
	}
	// E o corte não reintroduz o que a redacção tirou.
	if strings.Contains(texto, h.MissaoID) {
		t.Fatal("o corte reintroduziu o identificador")
	}
}
