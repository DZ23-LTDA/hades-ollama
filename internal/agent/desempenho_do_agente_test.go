package agent

import (
	"strings"
	"testing"
	"time"
)

func instante(hora int) time.Time {
	return time.Date(2026, 10, 14, hora, 0, 0, 0, time.UTC)
}

// A PROPRIEDADE QUE VALE DINHEIRO: custo DESCONHECIDO não pode ser somado como zero. Um painel
// que somasse `CostCents` de todas as missões mostraria «custo: 250» quando uma delas gastou o
// que ninguém sabe — e *número errado é pior do que número ausente*.
func TestDesempenhoDeclaraCustoDesconhecidoEmVezDeSomarZero(t *testing.T) {
	concluida := instante(2)
	relatorio := AvaliarDesempenho([]Mission{
		{
			ID: "mis_a", State: MissionCompleted, CreatedAt: instante(1), CompletedAt: &concluida,
			Usage: &ModelUsage{Calls: 1, PromptTokens: 100, CostCents: 250, CostKnown: true},
		},
		{
			ID: "mis_b", State: MissionCompleted, CreatedAt: instante(1), CompletedAt: &concluida,
			Usage: &ModelUsage{Calls: 1, PromptTokens: 200, CostCents: 0, CostKnown: false},
		},
	})

	if relatorio.CustoConhecidoCents != 250 {
		t.Fatalf("custo conhecido somado errado: %d", relatorio.CustoConhecidoCents)
	}
	if relatorio.MissoesComCustoConhecido != 1 || relatorio.MissoesComCustoDesconhecido != 1 {
		t.Fatalf("as missões de custo desconhecido têm de ser CONTADAS: conhecido=%d desconhecido=%d",
			relatorio.MissoesComCustoConhecido, relatorio.MissoesComCustoDesconhecido)
	}
	if relatorio.TokensDeEntrada != 300 {
		t.Fatalf("tokens somados errados: %d", relatorio.TokensDeEntrada)
	}
}

// O §10 PEDE «qualidade» E PROÍBE avaliação fictícia. A resposta possível é declarar que não se
// mede — o que se testa aqui é que a declaração EXISTE e que ela nomeia o que falta.
func TestDesempenhoDeclaraOQueNaoMede(t *testing.T) {
	relatorio := AvaliarDesempenho([]Mission{{ID: "mis_a", State: MissionCompleted, CreatedAt: instante(1)}})
	junto := strings.Join(relatorio.NaoMedido, " | ")
	for _, esperado := range []string{"qualidade", "satisfacao", "orcamento", "intervencoes"} {
		if !strings.Contains(junto, esperado) {
			t.Fatalf("o relatório tem de declarar que não mede %q: %s", esperado, junto)
		}
	}
	if len(relatorio.NaoMedido) == 0 {
		t.Fatal("um relatório sem a lista do que não mede é uma avaliação fictícia por omissão")
	}
}

// RETRABALHO É OBSERVÁVEL: conta tentativas dos passos, não opinião sobre o trabalho.
func TestDesempenhoContaRetrabalhoPorTentativas(t *testing.T) {
	relatorio := AvaliarDesempenho([]Mission{{
		ID: "mis_a", State: MissionCompleted, CreatedAt: instante(1),
		Plan: []Step{{ID: "s1", Attempts: 1}, {ID: "s2", Attempts: 3}, {ID: "s3", Attempts: 1}},
	}})
	if relatorio.Passos != 3 || relatorio.PassosComRetrabalho != 1 || relatorio.TentativasExtra != 2 {
		t.Fatalf("retrabalho errado: passos=%d com_retrabalho=%d extras=%d",
			relatorio.Passos, relatorio.PassosComRetrabalho, relatorio.TentativasExtra)
	}
	// 1 de 3 passos teve retrabalho.
	if relatorio.TaxaDeRetrabalho < 0.333 || relatorio.TaxaDeRetrabalho > 0.334 {
		t.Fatalf("taxa de retrabalho errada: %v", relatorio.TaxaDeRetrabalho)
	}
}

// INVARIANTE DAS CATEGORIAS: toda a missão cai em exactamente uma. Se um estado novo entrar no
// produto sem entrar aqui, este teste falha — foi para isso que o `default` existe.
func TestDesempenhoClassificaTodaMissaoEmUmaCategoria(t *testing.T) {
	concluida := instante(3)
	relatorio := AvaliarDesempenho([]Mission{
		{ID: "a", State: MissionCompleted, CreatedAt: instante(1), CompletedAt: &concluida},
		{ID: "b", State: MissionFailed, CreatedAt: instante(1)},
		{ID: "c", State: MissionCancelled, CreatedAt: instante(1)},
		{ID: "d", State: MissionRecovering, CreatedAt: instante(1)},
		{ID: "e", State: MissionRunning, CreatedAt: instante(1)},
		{ID: "f", State: MissionAwaitingApproval, CreatedAt: instante(1)},
		{ID: "g", State: MissionState("ESTADO_QUE_NAO_EXISTE"), CreatedAt: instante(1)},
	})
	if relatorio.Missoes != 7 {
		t.Fatalf("missões=%d", relatorio.Missoes)
	}
	soma := relatorio.Concluidas + relatorio.Falhadas + relatorio.Canceladas + relatorio.EmCurso + relatorio.Recuperando
	if soma != relatorio.Missoes {
		t.Fatalf("categorias somam %d e as missões são %d — uma missão ficou sem categoria", soma, relatorio.Missoes)
	}
	if relatorio.EmCurso != 3 {
		t.Fatalf("não-finais (RUNNING, AWAITING_APPROVAL e o estado desconhecido) deviam somar 3: %d", relatorio.EmCurso)
	}
}

// SEM MISSÕES NÃO HÁ DESEMPENHO A AFIRMAR — e não pode sair NaN da divisão.
func TestDesempenhoSemMissoesNaoAfirmaNada(t *testing.T) {
	relatorio := AvaliarDesempenho(nil)
	if relatorio.TaxaDeConclusao != 0 || relatorio.TaxaDeRetrabalho != 0 {
		t.Fatalf("sem missões as taxas têm de ser 0: conclusao=%v retrabalho=%v",
			relatorio.TaxaDeConclusao, relatorio.TaxaDeRetrabalho)
	}
	if relatorio.Missoes != 0 {
		t.Fatalf("missões=%d", relatorio.Missoes)
	}
}

// TEMPO: usa `CompletedAt` quando existe, e NUNCA soma negativo — uma marca de tempo invertida é
// dado estragado, não trabalho negativo.
func TestDesempenhoUsaCompletedAtENuncaSomaTempoNegativo(t *testing.T) {
	concluida := instante(5)
	relatorio := AvaliarDesempenho([]Mission{
		{ID: "a", State: MissionCompleted, CreatedAt: instante(1), UpdatedAt: instante(9), CompletedAt: &concluida},
		{ID: "b", State: MissionFailed, CreatedAt: instante(4), UpdatedAt: instante(6)},
		{ID: "c", State: MissionFailed, CreatedAt: instante(8), UpdatedAt: instante(2)},
	})
	// a = 4h (CompletedAt, não UpdatedAt), b = 2h, c = invertida ⇒ 0.
	if relatorio.TempoTotalDeExecucao != 6*time.Hour {
		t.Fatalf("tempo total errado: %v (esperado 6h)", relatorio.TempoTotalDeExecucao)
	}
}
