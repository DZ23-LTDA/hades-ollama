package agent

import (
	"errors"
	"testing"
	"time"
)

// E-536 (Estagio 46) - o teste das QUATRO REGRAS. *Nao testa o `switch`: testa o que o complemento proibe.*

func ev(tipo, missao, passo string, minuto int) CompanionEventoVerificado {
	return CompanionEventoVerificado{
		Tipo: tipo, MissaoID: missao, PassoID: passo,
		Quando: time.Date(2026, 10, 14, 12, minuto, 0, 0, time.UTC),
	}
}

// ***REGRA 1 - E A MAIS IMPORTANTE DESTE FICHEIRO.*** Sem eventos, o estado e' `ausente`.
//
// *Um mascote a rodar sem nada a acontecer e' a mentira mais facil de escrever e a mais dificil de detectar. Este teste
// e' a unica coisa que a impede.*
func TestCompanionSemEventosNaoDizQueEstaATrabalhar(t *testing.T) {
	estado, razao, err := DerivarEstadoCompanion("m1", nil)
	if err != nil {
		t.Fatalf("sem eventos devia ser estado legitimo, veio erro: %v", err)
	}
	if estado != CompanionAusente {
		t.Fatalf("ARMADILHA: sem eventos o estado foi %q - *um Companion que afirma trabalho sem um unico evento e' "+
			"exactamente o «inventar agente trabalhando» que o complemento proibe*", estado)
	}
	if razao == "" {
		t.Fatal("ARMADILHA: o estado veio sem razao - *um estado sem procedencia e' indistinguivel de um inventado*")
	}

	// E a contraprova directa: com `ausente`, o Companion NAO pode afirmar trabalho.
	for _, frase := range []string{"Estou a trabalhar na sua missao", "processando o pedido", "estou a fazer isso"} {
		if CompanionPodeDizer(CompanionAusente, frase) {
			t.Fatalf("ARMADILHA: com estado `ausente`, o Companion pode dizer %q", frase)
		}
	}
}

// ***REGRA 2 -** evento desconhecido RECUSA.*** *«Parece um evento de sucesso» nao e' um evento de sucesso.*
func TestCompanionRecusaEventoDesconhecido(t *testing.T) {
	eventos := []CompanionEventoVerificado{
		ev("mission.created", "m1", "", 0),
		ev("mission.terminou_tudo_bem", "m1", "", 1), // *nao esta' no vocabulario fechado*
	}
	_, _, err := DerivarEstadoCompanion("m1", eventos)
	if !errors.Is(err, ErrCompanionEventoDesconhecido) {
		t.Fatalf("ARMADILHA: evento fora do vocabulario foi aceite (err=%v) - *o vocabulario e' fechado: crescer tem de "+
			"ser uma decisao, e nao um efeito lateral de escrever uma string*", err)
	}
}

// ***REGRA 3 -** evento de outra missao e' ignorado.**
func TestCompanionIsolaPorMissao(t *testing.T) {
	eventos := []CompanionEventoVerificado{
		ev("mission.completed", "OUTRA", "", 5), // *concluida - mas nao e' a nossa*
	}
	estado, _, err := DerivarEstadoCompanion("m1", eventos)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if estado != CompanionAusente {
		t.Fatalf("ARMADILHA: a conclusao de OUTRA missao produziu %q na nossa - *isolamento: o Companion de uma missao "+
			"nao fala de outra*", estado)
	}
}

// ***REGRA 4 -** `concluido` e `falhou` SO' com os seus eventos.**
func TestCompanionNaoConcluiSemEventoDeConclusao(t *testing.T) {
	// Uma missao que correu tudo e **nao** tem evento de conclusao: *o silencio nao e' sucesso.*
	eventos := []CompanionEventoVerificado{
		ev("mission.created", "m1", "", 0),
		ev("mission.planning", "m1", "", 1),
		ev("mission.step.started", "m1", "s1", 2),
		ev("mission.step.finished", "m1", "s1", 3),
	}
	estado, _, err := DerivarEstadoCompanion("m1", eventos)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if estado == CompanionConcluido {
		t.Fatal("ARMADILHA: o Companion concluiu SEM `mission.completed` - *um silencio nao e' um sucesso*")
	}
	if estado != CompanionAPlanejar {
		t.Fatalf("estado esperado a_planejar (ultimo nao-terminal), veio %q", estado)
	}

	// E com o evento, conclui - pelo evento, e nao por tempo.
	eventos = append(eventos, ev("mission.completed", "m1", "", 4))
	estado, razao, err := DerivarEstadoCompanion("m1", eventos)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if estado != CompanionConcluido || razao != "mission.completed" {
		t.Fatalf("esperado concluido por mission.completed, veio %q (%q)", estado, razao)
	}
}

// ***E A REGRA QUE O COMPLEMENTO NAO NOMEIA, MAS DECORRE DA HONESTIDADE:*** enquanto ha' aprovacao pendente, o
// Companion **nao** esta' a trabalhar - esta' a esperar o humano.
func TestCompanionComAprovacaoPendenteNaoDizQueEstaATrabalhar(t *testing.T) {
	eventos := []CompanionEventoVerificado{
		ev("mission.created", "m1", "", 0),
		ev("mission.step.started", "m1", "s1", 1),
		ev("mission.approval.requested", "m1", "s1", 2),
	}
	estado, _, err := DerivarEstadoCompanion("m1", eventos)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if estado != CompanionAAProvar {
		t.Fatalf("ARMADILHA: com aprovacao pendente o estado foi %q - *dizer «a trabalhar» enquanto o sistema espera "+
			"pelo humano e' a versao mais discreta da mentira*", estado)
	}
	if !CompanionPodeDizer(estado, "Preciso da sua decisão para continuar") {
		t.Fatal("o Companion devia poder pedir a decisão")
	}
	if CompanionPodeDizer(estado, "Estou a trabalhar nisso") {
		t.Fatal("ARMADILHA: com aprovacao pendente, o Companion afirmou trabalho")
	}

	// E quando o humano decide, a aprovacao deixa de estar pendente.
	comDecisao := append(eventos, ev("mission.approval.granted", "m1", "s1", 3))
	estado2, _, err := DerivarEstadoCompanion("m1", comDecisao)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if estado2 != CompanionATrabalhar {
		t.Fatalf("depois da aprovacao o estado devia voltar a a_trabalhar (o passo continua aberto), veio %q", estado2)
	}
}

// O bloqueio mantem-se: *nao se inventa um desbloqueio.*
func TestCompanionBloqueioNaoSeLevantaSozinho(t *testing.T) {
	eventos := []CompanionEventoVerificado{
		ev("mission.created", "m1", "", 0),
		ev("mission.blocked", "m1", "", 1),
		ev("mission.step.finished", "m1", "s1", 2), // *um passo a acabar nao desbloqueia nada*
	}
	estado, razao, err := DerivarEstadoCompanion("m1", eventos)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if estado != CompanionBloqueado || razao != "mission.blocked" {
		t.Fatalf("ARMADILHA: o bloqueio foi levantado sem evento que o levante (estado=%q razao=%q)", estado, razao)
	}
}

// E `CompanionPodeDizer` e' a guarda minima: *afirmar sucesso exige o estado `concluido`.*
func TestCompanionNaoAfirmaSucessoSemConclusao(t *testing.T) {
	for _, estado := range []CompanionEstado{CompanionAusente, CompanionATrabalhar, CompanionAAProvar, CompanionBloqueado, CompanionFalhou} {
		if CompanionPodeDizer(estado, "Tudo concluído com sucesso") {
			t.Fatalf("ARMADILHA: com estado %q o Companion afirmou sucesso", estado)
		}
	}
	if !CompanionPodeDizer(CompanionConcluido, "Tudo concluído") {
		t.Fatal("com estado concluido, afirmar conclusão tem de ser permitido")
	}
}

// `DerivarEstadoCompanion` exige missao: *sem saber de quem se fala, nao ha' estado honesto possivel.*
func TestCompanionExigeMissao(t *testing.T) {
	if _, _, err := DerivarEstadoCompanion("   ", nil); !errors.Is(err, ErrCompanionMissaoAusente) {
		t.Fatalf("missao vazia devia recusar, veio %v", err)
	}
}

// E o vocabulario fechado tem de conter os eventos que os outros ficheiros JA' produzem.
func TestCompanionVocabularioCobreOsEventosQueJaExistem(t *testing.T) {
	// *`mission.repair_exhausted` e' emitido por `coding_git_cycle.go` - medido. Se ele sai do vocabulario, o
	// Companion deixa de conseguir falar daquela falha.*
	for _, tipo := range []string{"mission.repair_exhausted", "mission.created", "mission.completed"} {
		if !eventoVerificado(tipo) {
			t.Fatalf("ARMADILHA: %q nao esta' no vocabulario verificado", tipo)
		}
	}
}
