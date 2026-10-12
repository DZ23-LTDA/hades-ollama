package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// Estagio 28 — TURNO DE VOZ: LATENCIA MEDIDA E CONSENTIMENTO (E-271). Limite declarado: "completar STT/TTS e
// latencia em sandbox".
//
// O AUDIO REAL nao e' simulado: esta camada registra a latencia MEDIDA por estagio e DECLARA que processar audio
// exige provedor/credencial. Fingir transcricao seria apresentar um botao que nao funciona.
func TestTurnoDeVozMedeLatenciaRespeitaOrcamentoEExigeConsentimento(t *testing.T) {
	root := t.TempDir()
	loja := NewVoiceTurnStore(filepath.Join(root, "voice"))
	agora := time.Now().UTC()
	org := LocalOrganizationID

	// 1. Turno com latencia medida: o total e' a SOMA dos estagios.
	turno, err := loja.RecordTurn(VoiceTurn{OrganizationID: org, Channel: "telefone", ContactID: "ct_1", STTMS: 120, AgentMS: 480, TTSMS: 90}, agora)
	if err != nil {
		t.Fatalf("turno simples: %v", err)
	}
	if turno.TotalMS != 690 {
		t.Fatalf("total precisa ser a soma dos estagios: %d", turno.TotalMS)
	}
	if turno.LiveStatus != VoiceLiveStatus {
		t.Fatalf("o turno precisa DECLARAR que o audio real e externo: %+v", turno)
	}
	// Sem orcamento definido vale o padrao declarado; o turno esta dentro dele.
	if !turno.WithinBudget || turno.BudgetMS != VoiceDefaultLatencyBudgetMS {
		t.Fatalf("orcamento padrao nao aplicado: %+v", turno)
	}

	// 2. Orcamento ESTOURADO e' flagrado e CONTADO no relatorio.
	turnoLento, err := loja.RecordTurn(VoiceTurn{OrganizationID: org, STTMS: 900, AgentMS: 1500, TTSMS: 300}, agora)
	if err != nil {
		t.Fatal(err)
	}
	if turnoLento.WithinBudget {
		t.Fatalf("2700ms acima do padrao de %dms precisa estourar o orcamento: %+v", VoiceDefaultLatencyBudgetMS, turnoLento)
	}
	if err := loja.SetVoiceLatencyBudget(org, 3000); err != nil {
		t.Fatal(err)
	}
	depoisDoAumento, err := loja.RecordTurn(VoiceTurn{OrganizationID: org, STTMS: 900, AgentMS: 1500, TTSMS: 300}, agora)
	if err != nil {
		t.Fatal(err)
	}
	if !depoisDoAumento.WithinBudget {
		t.Fatalf("com orcamento de 3000ms o turno de 2700ms precisa caber: %+v", depoisDoAumento)
	}
	relatorio, err := loja.VoiceLatencyReport(org)
	if err != nil {
		t.Fatal(err)
	}
	if relatorio.Turns != 3 || relatorio.OverBudget != 1 {
		t.Fatalf("relatorio precisa contar o turno estourado: %+v", relatorio)
	}
	if relatorio.BudgetMS != 3000 {
		t.Fatalf("orcamento do relatorio: %d", relatorio.BudgetMS)
	}
	// Media por estagio: stt = (120+900+900)/3 = 640; total = (690+2700+2700)/3 = 2030.
	for _, estagio := range relatorio.Stages {
		if estagio.Stage == "stt" && estagio.AvgMS != 640 {
			t.Fatalf("media de stt: %d", estagio.AvgMS)
		}
		if estagio.Stage == "total" && estagio.AvgMS != 2030 {
			t.Fatalf("media de total: %d", estagio.AvgMS)
		}
	}

	// 3. GRAVACAO SEM CONSENTIMENTO e' RECUSADA e nada muda.
	antes := relatorio.Turns
	if _, err := loja.RecordTurn(VoiceTurn{OrganizationID: org, STTMS: 100, RecordingRequested: true}, agora); !errors.Is(err, ErrVoiceRecordingWithoutConsent) {
		t.Fatalf("gravacao sem consentimento precisa ser recusada: %v", err)
	}
	// Com consentimento, entra - e a gravacao consentida nao altera a latencia.
	if _, err := loja.RecordTurn(VoiceTurn{OrganizationID: org, STTMS: 100, RecordingRequested: true, ConsentGranted: true}, agora); err != nil {
		t.Fatalf("gravacao consentida: %v", err)
	}
	depois, err := loja.VoiceLatencyReport(org)
	if err != nil {
		t.Fatal(err)
	}
	if depois.Turns != antes+1 {
		t.Fatalf("a recusa nao pode registrar turno: %d -> %d", antes, depois.Turns)
	}

	// 4. Duracao invalida nao entra.
	if _, err := loja.RecordTurn(VoiceTurn{OrganizationID: org, STTMS: -1}, agora); !errors.Is(err, ErrVoiceTurnInvalid) {
		t.Fatalf("latencia negativa precisa ser recusada: %v", err)
	}
	if _, err := loja.RecordTurn(VoiceTurn{OrganizationID: org}, agora); !errors.Is(err, ErrVoiceTurnInvalid) {
		t.Fatalf("turno sem latencia medida precisa ser recusado: %v", err)
	}

	// 5. ISOLAMENTO: outra organizacao nao ve nem herda orcamento.
	outro, err := loja.VoiceLatencyReport("org-b")
	if err != nil {
		t.Fatal(err)
	}
	if outro.Turns != 0 || outro.BudgetMS != VoiceDefaultLatencyBudgetMS {
		t.Fatalf("metricas e orcamento nao podem vazar entre organizacoes: %+v", outro)
	}

	// 6. PERSISTENCIA: reinicio REAL sobre o mesmo diretorio.
	reaberta := NewVoiceTurnStore(filepath.Join(root, "voice"))
	recuperado, err := reaberta.VoiceLatencyReport(org)
	if err != nil {
		t.Fatal(err)
	}
	if recuperado.Turns != depois.Turns || recuperado.BudgetMS != 3000 {
		t.Fatalf("turnos e orcamento precisam sobreviver ao reinicio: %+v", recuperado)
	}
}
