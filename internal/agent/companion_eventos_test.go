package agent

import (
	"testing"
	"time"
)

func eventoDeTeste(id, missao, passo, tipo string, quando time.Time, payload any) Event {
	return Event{ID: id, MissionID: missao, Type: tipo, StepID: passo, CreatedAt: quando, Payload: payload}
}

// Os quatro campos que o Companion precisa sao transportados fielmente.
func TestConversorEventosDoCompanionMapeiaOsQuatroCampos(t *testing.T) {
	quando := time.Date(2026, 10, 15, 13, 0, 0, 0, time.UTC)
	eventos := []Event{eventoDeTeste("e1", "m1", "p1", "mission.started", quando, nil)}

	convertidos := ConversorEventosDoCompanion(eventos, nil)

	if len(convertidos) != 1 {
		t.Fatalf("esperava 1 evento convertido, veio %d", len(convertidos))
	}
	convertido := convertidos[0]
	if convertido.Tipo != "mission.started" || convertido.MissaoID != "m1" || convertido.PassoID != "p1" {
		t.Fatalf("campos nao transportados: %+v", convertido)
	}
	if !convertido.Quando.Equal(quando) {
		t.Fatalf("Quando devia ser o instante real do evento, veio %v", convertido.Quando)
	}
}

// ***A REGRA DE DLP.*** `Payload` nao tem sanitizacao garantida neste caminho: mesmo
// quando e' uma string legivel, nao pode virar `Detalhe`. *Um detalhe vazado nao se
// desfaz depois.*
func TestConversorEventosDoCompanionNaoCopiaPayload(t *testing.T) {
	quando := time.Date(2026, 10, 15, 13, 0, 0, 0, time.UTC)
	segredo := "linha crua com dado que ainda NAO foi sanitizado"
	eventos := []Event{
		eventoDeTeste("e1", "m1", "", "mission.step", quando, segredo),
		eventoDeTeste("e2", "m1", "", "mission.step", quando, map[string]any{"token": segredo}),
	}

	convertidos := ConversorEventosDoCompanion(eventos, nil)

	for _, convertido := range convertidos {
		if convertido.Detalhe != "" {
			t.Fatalf("Payload vazou para Detalhe: %q", convertido.Detalhe)
		}
	}
}

// So' entra o detalhe que o chamador declarou explicitamente como sanitizado, e so' o
// do proprio evento.
func TestConversorEventosDoCompanionUsaSoDetalheSanitizadoExplicito(t *testing.T) {
	quando := time.Date(2026, 10, 15, 13, 0, 0, 0, time.UTC)
	eventos := []Event{
		eventoDeTeste("e1", "m1", "", "mission.step", quando, "cru"),
		eventoDeTeste("e2", "m1", "", "mission.step", quando, "cru"),
	}

	convertidos := ConversorEventosDoCompanion(eventos, map[string]string{"e1": "sanitizado"})

	if convertidos[0].Detalhe != "sanitizado" {
		t.Fatalf("o detalhe sanitizado devia entrar: %q", convertidos[0].Detalhe)
	}
	if convertidos[1].Detalhe != "" {
		t.Fatalf("evento sem detalhe declarado devia sair vazio: %q", convertidos[1].Detalhe)
	}
}

// A ordem e' parte do contrato: por `Quando`, com desempate estavel e determinista.
func TestConversorEventosDoCompanionOrdenaPorQuandoEDesempata(t *testing.T) {
	maisCedo := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	maisTarde := time.Date(2026, 10, 15, 14, 0, 0, 0, time.UTC)
	// Chegam fora de ordem; dois empatados no mesmo instante, com passo/tipo a desempatar.
	eventos := []Event{
		eventoDeTeste("e4", "m1", "p2", "b", maisCedo, nil),
		eventoDeTeste("e1", "m1", "p9", "z", maisTarde, nil),
		eventoDeTeste("e2", "m1", "p1", "b", maisCedo, nil),
		eventoDeTeste("e3", "m1", "p1", "a", maisCedo, nil),
	}

	convertidos := ConversorEventosDoCompanion(eventos, nil)

	// Primeiro os tres de `maisCedo`, ordenados por PassoID e depois Tipo: p1/a, p1/b, p2/b.
	esperado := []string{"p1|a", "p1|b", "p2|b", "p9|z"}
	for i, chave := range esperado {
		obtido := convertidos[i].PassoID + "|" + convertidos[i].Tipo
		if obtido != chave {
			t.Fatalf("ordem errada na posicao %d: esperava %s, veio %s (todos: %+v)", i, chave, obtido, convertidos)
		}
	}

	// Duas execucoes sobre a mesma entrada nao podem divergir.
	outra := ConversorEventosDoCompanion(eventos, nil)
	for i := range convertidos {
		if convertidos[i] != outra[i] {
			t.Fatalf("ordenacao nao determinista na posicao %d", i)
		}
	}
}

// O atalho da rota filtra por missao - e NAO substitui o controlo de acesso.
func TestEventosDoCompanionDaMissaoFiltraPorMissao(t *testing.T) {
	quando := time.Date(2026, 10, 15, 13, 0, 0, 0, time.UTC)
	eventos := []Event{
		eventoDeTeste("e1", "m1", "", "mission.started", quando, nil),
		eventoDeTeste("e2", "m2", "", "missao alheia", quando, nil),
		eventoDeTeste("e3", "m1", "", "mission.completed", quando, nil),
	}

	convertidos := EventosDoCompanionDaMissao("m1", eventos, nil)

	if len(convertidos) != 2 {
		t.Fatalf("esperava os 2 eventos de m1, veio %d", len(convertidos))
	}
	for _, convertido := range convertidos {
		if convertido.MissaoID != "m1" {
			t.Fatalf("evento de outra missao passou: %+v", convertido)
		}
	}
}

// Sem eventos nao ha' estado inventado: a lista sai vazia, nao nula nem preenchida.
func TestConversorEventosDoCompanionSemEventosNaoInventa(t *testing.T) {
	if convertidos := ConversorEventosDoCompanion(nil, nil); len(convertidos) != 0 {
		t.Fatalf("sem eventos devia sair vazio, veio %+v", convertidos)
	}
	if convertidos := EventosDoCompanionDaMissao("m1", nil, nil); len(convertidos) != 0 {
		t.Fatalf("missao sem eventos devia sair vazio, veio %+v", convertidos)
	}
}
