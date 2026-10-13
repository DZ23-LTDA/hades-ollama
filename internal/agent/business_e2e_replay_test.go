package agent

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

// ESTÁGIO 38 — REEXECUTAR A JORNADA: RECUSA EXPLÍCITA, SEM INFLAR ESTOQUE (E-305, corrige a leitura de E-304).
//
// E-304 afirmou "a reserva de estoque não sobrevive ao reinício". MEDIDO com diagnóstico de movimentos, isso estava
// ERRADO: antes e depois do reinício o saldo é IDÊNTICO (`OnHand: 8, Reserved: 0`) com TODOS os movimentos no lugar
// (`entrada`, `reserva`, `liberacao`, `saida`) — e o teste que já existia no repositório
// (`TestBusinessE2ESurvivesRealRestartWithoutDuplicating`) PASSA afirmando exatamente isso. A persistência sempre
// funcionou; a leitura estava errada.
//
// O que é REAL: a jornada é uma cerimônia de empresa NOVA e NÃO SUPORTA reexecução — ela somava a `carga inicial` de
// novo (estoque 10 -> 18) e depois falhava com o erro interno "reserva inesperada", confuso para quem opera.
//
// Propriedades travadas:
//  1. reexecutar na mesma empresa RECUSA com erro próprio (`ErrBusinessJourneyAlreadyCertified`);
//  2. a recusa NÃO deixa efeito: o estoque fica exatamente como estava (nada de inflar antes de falhar);
//  3. o teste fica ARMADO: a primeira jornada precisa ter mudado o estado;
//  4. o tenant vizinho não é tocado.
func TestReexecutarJornadaRecusaSemInflarEstoque(t *testing.T) {
	primeiro, _ := businessRuntime(t)
	agora := time.Now().UTC()
	entrada := DefaultBusinessJourneyInput(LocalOrganizationID, "org-vizinhanca")

	vizinhoAntes, err := primeiro.BusinessStatus("org-vizinhanca", agora)
	if err != nil {
		t.Fatal(err)
	}
	inicial, err := primeiro.BusinessStatus(LocalOrganizationID, agora)
	if err != nil {
		t.Fatal(err)
	}

	relatorio, err := primeiro.CertifyBusinessJourney(entrada, agora)
	if err != nil {
		t.Fatalf("primeira certificação: %v", err)
	}
	if relatorio.Verdict != BusinessVerdictOK {
		t.Fatalf("a primeira jornada precisa certificar: %+v", relatorio)
	}
	depoisDaPrimeira, err := primeiro.BusinessStatus(LocalOrganizationID, agora)
	if err != nil {
		t.Fatal(err)
	}
	if mesmoEstado(inicial, depoisDaPrimeira) {
		t.Fatal("a jornada não alterou o estado: a comparação seria vazia e não provaria nada")
	}
	saldo := primeiro.Commerce().Stock(LocalOrganizationID, entrada.SKU)

	// 1. REEXECUÇÃO: recusa EXPLÍCITA, não erro interno obscuro.
	if _, err := primeiro.CertifyBusinessJourney(entrada, agora); !errors.Is(err, ErrBusinessJourneyAlreadyCertified) {
		t.Fatalf("reexecutar precisa recusar com ErrBusinessJourneyAlreadyCertified, veio: %v", err)
	}
	// 2. A recusa não pode ter deixado efeito: estoque idêntico.
	if depois := primeiro.Commerce().Stock(LocalOrganizationID, entrada.SKU); depois != saldo {
		t.Fatalf("a recusa INFLou o estoque:\nantes:  %+v\ndepois: %+v", saldo, depois)
	}
	// 4. O vizinho continua como estava.
	vizinhoDepois, err := primeiro.BusinessStatus("org-vizinhanca", agora)
	if err != nil {
		t.Fatal(err)
	}
	if !mesmoEstado(vizinhoAntes, vizinhoDepois) {
		t.Fatalf("o tenant vizinho foi tocado:\nantes:  %+v\ndepois: %+v", vizinhoAntes, vizinhoDepois)
	}
}

// mesmoEstado compara dois relatórios de status de negócio ignorando o instante de geração.
func mesmoEstado(a, b BusinessStatusReport) bool {
	a.GeneratedAt, b.GeneratedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(a, b)
}
