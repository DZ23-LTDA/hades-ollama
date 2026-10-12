package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------------------------------------------
// TURNO DE VOZ: LATENCIA MEDIDA E CONSENTIMENTO (E-271)
// ---------------------------------------------------------------------------------------------------------------
//
// O audio REAL exige provedor/credencial (BLOCKED_BY_EXTERNAL_DEPENDENCY). O que esta camada entrega e o CONTRATO
// do turno com a latencia MEDIDA por estagio: sem isso, "voz em producao" seria uma afirmacao sem numero.

// VoiceLiveStatus declara a natureza externa do audio real.
const VoiceLiveStatus = "BLOCKED_BY_EXTERNAL_DEPENDENCY"

// VoiceLiveReason explica o desbloqueio do audio real.
const VoiceLiveReason = "processar audio real (STT/TTS) exige provedor e credencial; em sandbox o turno registra a " +
	"latencia MEDIDA pelo adaptador, sem transcrever nem sintetizar audio"

// VoiceDefaultLatencyBudgetMS e' o orcamento de latencia por turno quando a organizacao nao define um.
const VoiceDefaultLatencyBudgetMS = int64(2000)

var (
	// ErrVoiceTurnInvalid cobre organizacao ausente, duracoes invalidas ou turno vazio.
	ErrVoiceTurnInvalid = errors.New("voice turn invalid")
	// ErrVoiceRecordingWithoutConsent recusa registrar gravacao sem consentimento do contato.
	ErrVoiceRecordingWithoutConsent = errors.New("voice recording requires consent")
)

// VoiceTurn e' um turno de atendimento por voz com a latencia MEDIDA de cada estagio.
type VoiceTurn struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Channel        string `json:"channel,omitempty"`
	ContactID      string `json:"contact_id,omitempty"`
	// Duracoes medidas pelo adaptador, em MILISSEGUNDOS inteiros.
	STTMS   int64 `json:"stt_ms"`
	AgentMS int64 `json:"agent_ms"`
	TTSMS   int64 `json:"tts_ms"`
	// TotalMS e' a soma dos estagios; o adaptador pode informar um total maior (fila/rede), que prevalece.
	TotalMS int64 `json:"total_ms"`
	// RecordingRequested + ConsentGranted: gravacao sem consentimento e RECUSADA, nao apenas anotada.
	RecordingRequested bool  `json:"recording_requested"`
	ConsentGranted     bool  `json:"consent_granted"`
	BudgetMS           int64 `json:"budget_ms"`
	WithinBudget       bool  `json:"within_budget"`
	// LiveStatus e' SEMPRE o bloqueio externo: esta camada nao processa audio real.
	LiveStatus string    `json:"live_status"`
	LiveReason string    `json:"live_reason"`
	CreatedAt  time.Time `json:"created_at"`
}

// VoiceLatencyStage resume um estagio.
type VoiceLatencyStage struct {
	Stage   string `json:"stage"`
	AvgMS   int64  `json:"avg_ms"`
	MaxMS   int64  `json:"max_ms"`
	Samples int    `json:"samples"`
}

// VoiceLatencyReport e' o painel de latencia da organizacao.
type VoiceLatencyReport struct {
	OrganizationID string              `json:"organization_id"`
	Turns          int                 `json:"turns"`
	WithinBudget   int                 `json:"within_budget"`
	OverBudget     int                 `json:"over_budget"`
	BudgetMS       int64               `json:"budget_ms"`
	Stages         []VoiceLatencyStage `json:"stages"`
	LiveStatus     string              `json:"live_status"`
	LiveReason     string              `json:"live_reason"`
}

type voiceTurnFile struct {
	Turns  []VoiceTurn `json:"turns"`
	Budget int64       `json:"budget_ms,omitempty"`
}

// VoiceTurnStore guarda os turnos e o orcamento de latencia por organizacao.
type VoiceTurnStore struct {
	root string
}

// NewVoiceTurnStore cria o armazenamento. `root` vazio mantem tudo em memoria (declarado).
func NewVoiceTurnStore(root string) *VoiceTurnStore {
	return &VoiceTurnStore{root: strings.TrimSpace(root)}
}

func (s *VoiceTurnStore) organizationPath(organizationID string) string {
	if s == nil || s.root == "" {
		return ""
	}
	seguro := strings.NewReplacer("/", "_", "\\", "_", "..", "_").Replace(consentScope(organizationID))
	if seguro == "" {
		return ""
	}
	return filepath.Join(s.root, "voice", seguro+".json")
}

func (s *VoiceTurnStore) load(organizationID string) (voiceTurnFile, error) {
	caminho := s.organizationPath(organizationID)
	if caminho == "" {
		return voiceTurnFile{}, nil
	}
	stored := voiceTurnFile{}
	if err := readJSON(caminho, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return voiceTurnFile{}, nil
		}
		return voiceTurnFile{}, fmt.Errorf("ler turnos de voz: %w", err)
	}
	return stored, nil
}

func (s *VoiceTurnStore) save(organizationID string, stored voiceTurnFile) error {
	caminho := s.organizationPath(organizationID)
	if caminho == "" {
		return nil
	}
	return writeJSONAtomic(caminho, stored)
}

// SetVoiceLatencyBudget define o orcamento por turno. Zero volta ao padrao DECLARADO.
func (s *VoiceTurnStore) SetVoiceLatencyBudget(organizationID string, budgetMS int64) error {
	organizationID = consentScope(organizationID)
	if organizationID == "" {
		return fmt.Errorf("%w: organizacao e obrigatoria", ErrVoiceTurnInvalid)
	}
	if budgetMS < 0 {
		return fmt.Errorf("%w: orcamento negativo (%d)", ErrVoiceTurnInvalid, budgetMS)
	}
	stored, err := s.load(organizationID)
	if err != nil {
		return err
	}
	stored.Budget = budgetMS
	return s.save(organizationID, stored)
}

func (s *VoiceTurnStore) budgetMS(organizationID string, stored voiceTurnFile) int64 {
	if stored.Budget > 0 {
		return stored.Budget
	}
	return VoiceDefaultLatencyBudgetMS
}

// RecordTurn registra um turno com a latencia MEDIDA. Gravacao sem consentimento e RECUSADA.
func (s *VoiceTurnStore) RecordTurn(turn VoiceTurn, now time.Time) (VoiceTurn, error) {
	organizationID := consentScope(turn.OrganizationID)
	if organizationID == "" {
		return VoiceTurn{}, fmt.Errorf("%w: organizacao e obrigatoria", ErrVoiceTurnInvalid)
	}
	if turn.STTMS < 0 || turn.AgentMS < 0 || turn.TTSMS < 0 || turn.TotalMS < 0 {
		return VoiceTurn{}, fmt.Errorf("%w: latencia negativa (stt=%d agente=%d tts=%d total=%d)",
			ErrVoiceTurnInvalid, turn.STTMS, turn.AgentMS, turn.TTSMS, turn.TotalMS)
	}
	soma := turn.STTMS + turn.AgentMS + turn.TTSMS
	if soma == 0 {
		return VoiceTurn{}, fmt.Errorf("%w: turno sem latencia medida nao e' turno", ErrVoiceTurnInvalid)
	}
	if turn.RecordingRequested && !turn.ConsentGranted {
		// LGPD: gravar conversa exige consentimento. Recusar e' a unica resposta honesta.
		return VoiceTurn{}, fmt.Errorf("%w: contato %q", ErrVoiceRecordingWithoutConsent, turn.ContactID)
	}
	stored, err := s.load(organizationID)
	if err != nil {
		return VoiceTurn{}, err
	}
	gravado := turn
	gravado.OrganizationID = organizationID
	if gravado.ID == "" {
		gravado.ID = "vt_" + uuid.NewString()[:12]
	}
	// O total informado pelo adaptador PREVALECE quando for maior: fila e rede tambem sao latencia do cliente.
	if gravado.TotalMS < soma {
		gravado.TotalMS = soma
	}
	gravado.BudgetMS = s.budgetMS(organizationID, stored)
	gravado.WithinBudget = gravado.TotalMS <= gravado.BudgetMS
	gravado.LiveStatus = VoiceLiveStatus
	gravado.LiveReason = VoiceLiveReason
	gravado.CreatedAt = now.UTC()
	stored.Turns = append(stored.Turns, gravado)
	if err := s.save(organizationID, stored); err != nil {
		return VoiceTurn{}, err
	}
	return gravado, nil
}

// VoiceTurns lista os turnos da organizacao (mais recentes primeiro).
func (s *VoiceTurnStore) VoiceTurns(organizationID string, limit int) ([]VoiceTurn, error) {
	stored, err := s.load(organizationID)
	if err != nil {
		return nil, err
	}
	turnos := append([]VoiceTurn(nil), stored.Turns...)
	sort.Slice(turnos, func(i, j int) bool { return turnos[i].CreatedAt.After(turnos[j].CreatedAt) })
	if limit > 0 && len(turnos) > limit {
		turnos = turnos[:limit]
	}
	return turnos, nil
}

// VoiceLatencyReport calcula media e pico por estagio e conta os turnos acima do orcamento.
func (s *VoiceTurnStore) VoiceLatencyReport(organizationID string) (VoiceLatencyReport, error) {
	organizationID = consentScope(organizationID)
	stored, err := s.load(organizationID)
	if err != nil {
		return VoiceLatencyReport{}, err
	}
	report := VoiceLatencyReport{
		OrganizationID: organizationID,
		BudgetMS:       s.budgetMS(organizationID, stored),
		LiveStatus:     VoiceLiveStatus,
		LiveReason:     VoiceLiveReason,
	}
	agregados := map[string]*VoiceLatencyStage{}
	ordem := []string{}
	for _, turno := range stored.Turns {
		report.Turns++
		if turno.WithinBudget {
			report.WithinBudget++
		} else {
			report.OverBudget++
		}
		for _, amostra := range []struct {
			nome string
			ms   int64
		}{{"stt", turno.STTMS}, {"agente", turno.AgentMS}, {"tts", turno.TTSMS}, {"total", turno.TotalMS}} {
			atual, visto := agregados[amostra.nome]
			if !visto {
				atual = &VoiceLatencyStage{Stage: amostra.nome}
				agregados[amostra.nome] = atual
				ordem = append(ordem, amostra.nome)
			}
			atual.Samples++
			atual.AvgMS += amostra.ms
			if amostra.ms > atual.MaxMS {
				atual.MaxMS = amostra.ms
			}
		}
	}
	for _, nome := range ordem {
		estagio := agregados[nome]
		if estagio.Samples > 0 {
			estagio.AvgMS = estagio.AvgMS / int64(estagio.Samples)
		}
		report.Stages = append(report.Stages, *estagio)
	}
	return report, nil
}
