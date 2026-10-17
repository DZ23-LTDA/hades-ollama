package agent

import (
	"testing"
	"time"
)

// Os testes usam SEMPRE nomes do vocabulario fechado (`EventosVerificadosDeMissao`).
// *Antes usavam nomes inventados (`mission.started`, "a", "b") - e foi um defeito real
// que o teste de handler revelou: `CreateMission` emite `mission.planned`, que o nucleo
// recusa. O conversor passou a filtrar; estes testes passaram a falar a lingua do nucleo.*
func eventoDeTeste(id, missao, passo, tipo string, quando time.Time, payload any) Event {
	return Event{ID: id, MissionID: missao, Type: tipo, StepID: passo, CreatedAt: quando, Payload: payload}
}

// Os quatro campos que o Companion precisa sao transportados fielmente.
func TestConversorEventosDoCompanionMapeiaOsQuatroCampos(t *testing.T) {
	quando := time.Date(2026, 10, 15, 13, 0, 0, 0, time.UTC)
	eventos := []Event{eventoDeTeste("e1", "m1", "p1", "mission.step.started", quando, nil)}

	convertidos := ConversorEventosDoCompanion(eventos, nil)

	if len(convertidos) != 1 {
		t.Fatalf("esperava 1 evento convertido, veio %d", len(convertidos))
	}
	convertido := convertidos[0]
	if convertido.Tipo != "mission.step.started" || convertido.MissaoID != "m1" || convertido.PassoID != "p1" {
		t.Fatalf("campos nao transportados: %+v", convertido)
	}
	if !convertido.Quando.Equal(quando) {
		t.Fatalf("Quando devia ser o instante real do evento, veio %v", convertido.Quando)
	}
}

// ***O DEFEITO QUE ISTO IMPEDE.*** `CreateMission` emite `mission.planned`; o nucleo
// RECUSA esse evento. Se o conversor o encaminhasse, a derivacao falhava e a rota
// respondia 400 para toda a missao recem-criada. Aqui prova-se que ele e' DESCARTADO.
func TestConversorEventosDoCompanionDescartaEventoForaDoVocabulario(t *testing.T) {
	quando := time.Date(2026, 10, 15, 13, 0, 0, 0, time.UTC)
	eventos := []Event{
		eventoDeTeste("e0", "m1", "", "mission.planned", quando, nil), // emitido pelo runtime, fora do vocabulario
		eventoDeTeste("e1", "m1", "p1", "mission.created", quando, nil),
	}

	convertidos, descartados := ConversorEventosDoCompanionComRelatorio(eventos, nil)

	if descartados != 1 {
		t.Fatalf("esperava 1 descartado, veio %d (o descarte nao pode ser silencioso)", descartados)
	}
	if len(convertidos) != 1 || convertidos[0].Tipo != "mission.created" {
		t.Fatalf("so' o evento do vocabulario devia passar: %+v", convertidos)
	}
	if CompanionEventoEhVerificado("mission.planned") {
		t.Fatal("mission.planned NAO pertence ao vocabulario verificado")
	}
	if !CompanionEventoEhVerificado("mission.created") {
		t.Fatal("mission.created pertence ao vocabulario verificado")
	}
}

// ***A REGRA DE DLP.*** `Payload` nao tem sanitizacao garantida neste caminho: mesmo
// quando e' uma string legivel, nao pode virar `Detalhe`.
func TestConversorEventosDoCompanionNaoCopiaPayload(t *testing.T) {
	quando := time.Date(2026, 10, 15, 13, 0, 0, 0, time.UTC)
	segredo := "linha crua com dado que ainda NAO foi sanitizado"
	eventos := []Event{
		eventoDeTeste("e1", "m1", "", "mission.step.started", quando, segredo),
		eventoDeTeste("e2", "m1", "", "mission.step.finished", quando, map[string]any{"token": segredo}),
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
		eventoDeTeste("e1", "m1", "", "mission.step.started", quando, "cru"),
		eventoDeTeste("e2", "m1", "", "mission.step.finished", quando, "cru"),
	}

	convertidos := ConversorEventosDoCompanion(eventos, map[string]string{"e1": "sanitizado"})

	// *Nao se procura por posicao: o conversor ORDENA, logo a posicao nao e' a da entrada.
	// (Este teste falhou assim na primeira escrita - e a falha provou que a ordenacao actua.)*
	porTipo := map[string]CompanionEventoVerificado{}
	for _, convertido := range convertidos {
		porTipo[convertido.Tipo] = convertido
	}
	if porTipo["mission.step.started"].Detalhe != "sanitizado" {
		t.Fatalf("o detalhe sanitizado devia entrar: %q", porTipo["mission.step.started"].Detalhe)
	}
	if porTipo["mission.step.finished"].Detalhe != "" {
		t.Fatalf("evento sem detalhe declarado devia sair vazio: %q", porTipo["mission.step.finished"].Detalhe)
	}
}

// A ordem e' parte do contrato: por `Quando`, com desempate estavel e determinista.
// Desempate no mesmo instante: por `PassoID`, e depois por `Tipo` (a < b no vocabulario real).
func TestConversorEventosDoCompanionOrdenaPorQuandoEDesempata(t *testing.T) {
	maisCedo := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	maisTarde := time.Date(2026, 10, 15, 14, 0, 0, 0, time.UTC)
	eventos := []Event{
		eventoDeTeste("e4", "m1", "p2", "mission.step.started", maisCedo, nil),
		eventoDeTeste("e1", "m1", "p9", "mission.completed", maisTarde, nil),
		eventoDeTeste("e2", "m1", "p1", "mission.step.started", maisCedo, nil),
		eventoDeTeste("e3", "m1", "p1", "mission.approval.granted", maisCedo, nil),
	}

	convertidos := ConversorEventosDoCompanion(eventos, nil)

	// No mesmo instante: p1 com o Tipo que ordena primeiro, depois p1, depois p2.
	esperado := []string{"p1|mission.approval.granted", "p1|mission.step.started", "p2|mission.step.started", "p9|mission.completed"}
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
		eventoDeTeste("e1", "m1", "", "mission.created", quando, nil),
		eventoDeTeste("e2", "m2", "", "mission.created", quando, nil),
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
