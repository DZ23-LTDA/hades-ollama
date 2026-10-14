package agent

import (
	"testing"
	"time"
)

func missaoDeDespertar(id, agente string, estado MissionState) Mission {
	return Mission{ID: id, AgentID: agente, State: estado, CreatedAt: instante(1), UpdatedAt: instante(1)}
}

// A MISSÃO SEM AGENTE NÃO PRODUZ SINAL — e é deliberado: não se inventa um agente para acordar.
// (O `DespertaresCoalescidos` sabe lidar com sinal sem dono, mas aqui a razão é CONHECIDA, e é
// melhor não o produzir do que empurrá-lo para quem consome.)
func TestMissaoSemAgenteNaoProduzSinal(t *testing.T) {
	if sinais := SinaisDeMissao(missaoDeDespertar("mis_1", "", MissionAwaitingApproval), instante(2)); len(sinais) != 0 {
		t.Fatalf("missão sem agente não devia produzir sinal: %+v", sinais)
	}
}

// Estados que NÃO acordam ninguém: concluída, falhada e cancelada são desfechos, não trabalho.
func TestMissaoTerminadaNaoAcorda(t *testing.T) {
	for _, estado := range []MissionState{MissionCompleted, MissionFailed, MissionCancelled} {
		if sinais := SinaisDeMissao(missaoDeDespertar("mis_1", "agente-a", estado), instante(2)); len(sinais) != 0 {
			t.Fatalf("estado %q não devia acordar: %+v", estado, sinais)
		}
	}
}

// O motivo mais caro de perder: uma decisão humana à espera.
func TestAprovacaoPendenteAcordaComEsseMotivo(t *testing.T) {
	missao := missaoDeDespertar("mis_1", "agente-a", MissionAwaitingApproval)
	sinais := SinaisDeMissao(missao, instante(2))
	if len(sinais) != 1 || sinais[0].Motivo != MotivoAprovacaoPendente {
		t.Fatalf("devia acordar com aprovação pendente: %+v", sinais)
	}
	if sinais[0].MissaoID != "mis_1" || sinais[0].AgenteID != "agente-a" {
		t.Fatalf("o sinal tem de dizer QUEM e POR CAUSA DE QUÊ: %+v", sinais[0])
	}
}

// Uma missão em recuperação acorda por esse motivo; e a aprovação registada conta mesmo que o
// estado não seja `AWAITING_APPROVAL` — o que manda é o REGISTO.
func TestRecuperacaoE_APROVACAORegistadaContam(t *testing.T) {
	recuperando := missaoDeDespertar("mis_2", "agente-a", MissionRecovering)
	sinais := SinaisDeMissao(recuperando, instante(2))
	if len(sinais) != 1 || sinais[0].Motivo != MotivoRecuperacao {
		t.Fatalf("recuperação devia acordar: %+v", sinais)
	}

	emCursoComAprovacao := missaoDeDespertar("mis_3", "agente-a", MissionRunning)
	emCursoComAprovacao.Approvals = []Approval{{Status: ApprovalStatus("PENDING")}}
	sinais = SinaisDeMissao(emCursoComAprovacao, instante(2))
	if len(sinais) != 1 || sinais[0].Motivo != MotivoAprovacaoPendente {
		t.Fatalf("aprovação PENDENTE registada devia acordar mesmo com estado RUNNING: %+v", sinais)
	}
}

// A PONTE: três missões do mesmo agente ⇒ UM despertar, com TODAS as missões e as razões distintas.
func TestDespertaresDasMissoesCoalescemPorAgente(t *testing.T) {
	missoes := []Mission{
		missaoDeDespertar("mis_1", "agente-a", MissionAwaitingApproval),
		missaoDeDespertar("mis_2", "agente-a", MissionRecovering),
		missaoDeDespertar("mis_3", "agente-a", MissionReady),
		missaoDeDespertar("mis_4", "agente-b", MissionAwaitingApproval),
		missaoDeDespertar("mis_5", "", MissionReady), // sem agente: não conta
	}
	despertares := DespertaresDasMissoes(missoes, instante(3))
	if len(despertares) != 2 {
		t.Fatalf("dois agentes deviam dar dois despertares: %+v", despertares)
	}
	primeiro := despertares[0]
	if primeiro.AgenteID != "agente-a" {
		t.Fatalf("ordem por agente devia pôr agente-a primeiro: %+v", primeiro)
	}
	if len(primeiro.Motivos) != 3 {
		t.Fatalf("o despertar devia levar as 3 razões distintas: %+v", primeiro.Motivos)
	}
	if len(primeiro.Missoes) != 3 {
		t.Fatalf("o despertar devia dizer as 3 missões: %+v", primeiro.Missoes)
	}
	if primeiro.Sinais != 3 {
		t.Fatalf("a contagem devia ser 3: %d", primeiro.Sinais)
	}
	// E nada de missões de OUTRO agente no despertar de agente-a.
	for _, missaoID := range primeiro.Missoes {
		if missaoID == "mis_4" {
			t.Fatal("*** o despertar de agente-a levou uma missão de agente-b ***")
		}
	}
}

// Sem missões não há despertares — e nada é inventado.
func TestDespertaresDasMissoesSemMissoes(t *testing.T) {
	if despertares := DespertaresDasMissoes(nil, time.Now().UTC()); len(despertares) != 0 {
		t.Fatalf("sem missões não há despertares: %+v", despertares)
	}
}

// O QUE VAI À FILA, E O QUE NÃO VAI (E-638). Esta é a decisão que separa «informar» de «executar».
func TestSoTrabalhoPendenteVaiParaAFila(t *testing.T) {
	pronta := missaoDeDespertar("mis_pronta", "agente-a", MissionReady)
	aprovacao := missaoDeDespertar("mis_aprovar", "agente-a", MissionAwaitingApproval)
	recuperando := missaoDeDespertar("mis_recuperar", "agente-a", MissionRecovering)
	porID := map[string]Mission{pronta.ID: pronta, aprovacao.ID: aprovacao, recuperando.ID: recuperando}

	// Um despertar com trabalho pendente ENTRE os motivos leva à fila só a missão pronta.
	misto := Despertar{AgenteID: "agente-a", Motivos: []string{MotivoAprovacaoPendente, MotivoTrabalhoPendente}, Missoes: []string{pronta.ID, aprovacao.ID, recuperando.ID}}
	fila := MissoesAExecutar(misto, porID)
	if len(fila) != 1 || fila[0] != pronta.ID {
		t.Fatalf("só a missão pronta devia ir à fila: %v", fila)
	}

	// E um despertar SÓ informativo não leva NADA — aprovar é de uma pessoa.
	informativo := Despertar{AgenteID: "agente-a", Motivos: []string{MotivoAprovacaoPendente, MotivoRecuperacao}, Missoes: []string{aprovacao.ID, recuperando.ID}}
	if fila := MissoesAExecutar(informativo, porID); len(fila) != 0 {
		t.Fatalf("*** despertar informativo não pode enfileirar ***: %v", fila)
	}
}

// A FOTOGRAFIA NÃO MANDA NA FILA: se a missão deixou de estar pronta, não entra.
func TestMissaoQueDeixouDeEstarProntaNaoEntra(t *testing.T) {
	despertar := Despertar{AgenteID: "agente-a", Motivos: []string{MotivoTrabalhoPendente}, Missoes: []string{"mis_1"}}
	corrente := map[string]Mission{"mis_1": missaoDeDespertar("mis_1", "agente-a", MissionRunning)}
	if fila := MissoesAExecutar(despertar, corrente); len(fila) != 0 {
		t.Fatalf("missão que já corre não pode ser enfileirada: %v", fila)
	}
	if fila := MissoesAExecutar(despertar, map[string]Mission{}); len(fila) != 0 {
		t.Fatalf("missão ausente da leitura não pode ser enfileirada: %v", fila)
	}
}
