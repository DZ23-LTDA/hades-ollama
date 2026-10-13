package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Auditoria DURÁVEL da execução de ferramentas dos agentes de atendimento (estágio 35).
//
// O limite declarado era: *"a trilha é POR TURNO — volta na resposta, mas não é auditoria durável"*. Trilha
// que desaparece no reinício não serve para auditoria: a pergunta que se faz depois de um incidente é
// **"o que aquele agente executou ontem?"** — e a resposta não pode depender de a resposta HTTP ter sido
// guardada por alguém.
//
// Decisões:
//
//  1. **UM ARQUIVO POR ORGANIZAÇÃO**: o isolamento vale no disco, não só no filtro de leitura;
//  2. **TETO POR ORGANIZAÇÃO**: um tenant com volume não pode expulsar a evidência de outro (defeito real
//     encontrado e corrigido na DLQ, em E-141);
//  3. **A TENTATIVA TAMBÉM É EVIDÊNCIA**: bloqueado, indisponível e falho entram na trilha — saber que uma
//     ferramenta foi BLOQUEADA é informação de auditoria, não ruído;
//  4. **O TEXTO DO CONTATO NÃO ENTRA**: o conteúdo de terceiro fica fora da trilha. Guardamos a ferramenta,
//     o efeito, o estado, o resumo proveniente do store e o motivo — nunca o que o cliente escreveu;
//  5. **FALHA DE PERSISTÊNCIA É REPORTADA**: nada de perder registro em silêncio.

// DefaultServiceAgentAuditRetention é o teto de registros por organização.
const DefaultServiceAgentAuditRetention = 500

// ServiceAgentAuditEntry é uma linha da trilha: o que foi executado, quando e com que resultado.
type ServiceAgentAuditEntry struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	AgentID        string    `json:"agent_id"`
	ConversationID string    `json:"conversation_id,omitempty"`
	ToolName       string    `json:"tool_name"`
	Effect         string    `json:"effect,omitempty"`
	Status         string    `json:"status"`
	Summary        string    `json:"summary,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	ModelID        string    `json:"model_id,omitempty"`
	DecidedAt      time.Time `json:"decided_at"`
	RecordedAt     time.Time `json:"recorded_at"`
	// PreviousHash e Hash formam a CADEIA: cada registro carrega o hash do anterior.
	//
	// Isto dá **detecção** de adulteração, não prevenção: quem tem acesso ao disco pode reescrever a trilha
	// inteira e recalcular a cadeia. Detectar já é a diferença entre "a trilha pode ter sido editada sem
	// deixar rastro" e "a edição aparece na verificação" — e a assinatura com chave é o passo seguinte.
	PreviousHash string `json:"previous_hash,omitempty"`
	Hash         string `json:"hash,omitempty"`
}

// ServiceAgentAuditGenesis é o elo anterior do primeiro registro da cadeia.
const ServiceAgentAuditGenesis = "genesis"

// ServiceAgentAuditStore persiste a trilha de execução por organização.
type ServiceAgentAuditStore struct {
	root string
	// MaxPerOrganization limita a trilha de CADA organização.
	MaxPerOrganization int
}

type serviceAgentAuditFile struct {
	Entries []ServiceAgentAuditEntry `json:"entries"`
}

// NewServiceAgentAuditStore cria o store da trilha.
func NewServiceAgentAuditStore(root string) *ServiceAgentAuditStore {
	return &ServiceAgentAuditStore{
		root:               strings.TrimSpace(root),
		MaxPerOrganization: DefaultServiceAgentAuditRetention,
	}
}

// Root devolve o diretório usado (vazio quando não há persistência).
func (s *ServiceAgentAuditStore) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *ServiceAgentAuditStore) path(organizationID string) string {
	if s == nil || s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

// Load devolve a trilha persistida da organização (mais antigas primeiro).
func (s *ServiceAgentAuditStore) Load(organizationID string) ([]ServiceAgentAuditEntry, error) {
	if err := requireOrganization(organizationID); err != nil {
		return nil, err
	}
	path := s.path(organizationID)
	if path == "" {
		return nil, nil
	}
	var file serviceAgentAuditFile
	err := readJSON(path, &file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		// Arquivo ilegível é REPORTADO: tratá-lo como "sem registros" faria uma trilha corrompida parecer
		// limpa — exatamente o engano que uma auditoria não pode cometer.
		return nil, err
	}
	return file.Entries, nil
}

// Save grava a trilha da organização respeitando o teto dela.
func (s *ServiceAgentAuditStore) Save(organizationID string, entries []ServiceAgentAuditEntry) error {
	if err := requireOrganization(organizationID); err != nil {
		return err
	}
	path := s.path(organizationID)
	if path == "" {
		return nil
	}
	limit := s.MaxPerOrganization
	if limit <= 0 {
		limit = DefaultServiceAgentAuditRetention
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return writeJSONAtomic(path, serviceAgentAuditFile{Entries: entries})
}

// auditEntryHash calcula o hash de um registro, encadeado ao anterior.
//
// Os campos entram em ordem FIXA e separados por um delimitador que não aparece nos valores usuais: hash
// que depende de ordem de mapa ou de formatação instável acusaria adulteração onde não houve.
func auditEntryHash(entry ServiceAgentAuditEntry) string {
	partes := []string{
		entry.PreviousHash,
		entry.ID,
		entry.OrganizationID,
		entry.AgentID,
		entry.ConversationID,
		entry.ToolName,
		entry.Effect,
		entry.Status,
		entry.Summary,
		entry.Reason,
		entry.ModelID,
		entry.DecidedAt.UTC().Format(time.RFC3339Nano),
		entry.RecordedAt.UTC().Format(time.RFC3339Nano),
	}
	soma := sha256.Sum256([]byte(strings.Join(partes, "\x1f")))
	return hex.EncodeToString(soma[:])
}

// ServiceAgentAuditVerification é o resultado da verificação da cadeia.
type ServiceAgentAuditVerification struct {
	OrganizationID string `json:"organization_id"`
	// Valid diz se a cadeia inteira confere.
	Valid bool `json:"valid"`
	// Broken diz se há EVIDÊNCIA DE ADULTERAÇÃO (hash que não confere ou elo quebrado). É diferente de uma
	// trilha com registros antigos não verificáveis: ali não há evidência de edição, só de idade.
	Broken bool `json:"broken"`
	// Entries é quantos registros foram percorridos.
	Entries int `json:"entries"`
	// Legacy é quantos registros existem ANTES da cadeia (sem hash), que não podem ser verificados.
	Legacy int `json:"legacy,omitempty"`
	// FirstDivergence é o ID do primeiro registro que não confere.
	FirstDivergence string `json:"first_divergence,omitempty"`
	// Reason explica a divergência em pt-BR.
	Reason string `json:"reason,omitempty"`
}

// VerifyTrail percorre a cadeia e diz onde ela quebra, se quebrar.
func (s *ServiceAgentAuditStore) VerifyTrail(organizationID string) (ServiceAgentAuditVerification, error) {
	result := ServiceAgentAuditVerification{OrganizationID: organizationID, Valid: true}
	entries, err := s.Load(organizationID)
	if err != nil {
		return result, err
	}
	result.Entries = len(entries)
	expectedPrevious := ServiceAgentAuditGenesis
	for index, entry := range entries {
		if entry.Hash == "" {
			// Registro anterior à cadeia: não é verificável, e isso é DITO — sem acusar adulteração que não
			// há, e sem declarar válido o que não pode ser conferido.
			result.Legacy++
			result.Valid = false
			if result.FirstDivergence == "" {
				result.FirstDivergence = entry.ID
				result.Reason = "registro anterior à cadeia de hashes (não verificável); não há evidência de adulteração"
			}
			continue
		}
		if entry.PreviousHash != expectedPrevious {
			result.Valid = false
			result.Broken = true
			result.FirstDivergence = entry.ID
			result.Reason = fmt.Sprintf("elo quebrado na posição %d: o registro aponta para outro anterior", index)
			return result, nil
		}
		if auditEntryHash(entry) != entry.Hash {
			result.Valid = false
			result.Broken = true
			result.FirstDivergence = entry.ID
			result.Reason = fmt.Sprintf("conteúdo adulterado na posição %d: o hash não confere", index)
			return result, nil
		}
		expectedPrevious = entry.Hash
	}
	return result, nil
}

// Append acrescenta entradas à trilha da organização, ENCADEANDO cada uma ao hash anterior.
//
// A cadeia existente é verificada ANTES de estender: gravar sobre uma trilha já adulterada produziria um
// documento que parece íntegro a partir dali, escondendo a adulteração anterior.
func (s *ServiceAgentAuditStore) Append(organizationID string, entries []ServiceAgentAuditEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	existing, err := s.Load(organizationID)
	if err != nil {
		return 0, err
	}
	verification, err := s.VerifyTrail(organizationID)
	if err != nil {
		return 0, err
	}
	// Recusa só diante de EVIDÊNCIA DE ADULTERAÇÃO. Trilha com registros antigos não verificáveis (idade)
	// não pode congelar a auditoria para sempre — a limitação é declarada, não transformada em bloqueio.
	if verification.Broken {
		return 0, fmt.Errorf("%w: %s", ErrServiceAgentAuditChainBroken, verification.Reason)
	}
	previous := ServiceAgentAuditGenesis
	if len(existing) > 0 {
		previous = existing[len(existing)-1].Hash
		if previous == "" {
			// A trilha anterior não tem cadeia: começar uma nova é aceitável, mas a quebra fica declarada
			// pela verificação (contagem `Legacy`), em vez de silenciada.
			previous = ServiceAgentAuditGenesis
		}
	}
	for index := range entries {
		entries[index].PreviousHash = previous
		entries[index].Hash = auditEntryHash(entries[index])
		previous = entries[index].Hash
	}
	existing = append(existing, entries...)
	if err := s.Save(organizationID, existing); err != nil {
		return 0, err
	}
	if len(existing) > s.MaxPerOrganization {
		return s.MaxPerOrganization, nil
	}
	return len(existing), nil
}

// ErrServiceAgentAuditChainBroken marca tentativa de estender trilha cuja cadeia não confere.
var ErrServiceAgentAuditChainBroken = errors.New("service agent audit chain is broken")

// ErrServiceAgentAuditUnavailable: a trilha durável não está configurada, então um ato de política NÃO pode
// ser registrado — e isso precisa ser dito, não silenciado.
var ErrServiceAgentAuditUnavailable = errors.New("service agent audit trail is unavailable")

// serviceAgentAuditRoot devolve o diretório da trilha dentro da raiz de dados.
func serviceAgentAuditRoot(dataRoot string) string {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return ""
	}
	return filepath.Join(dataRoot, ".agent-service-audit")
}
