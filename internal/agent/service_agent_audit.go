package agent

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
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
}

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

// Append acrescenta entradas à trilha da organização e devolve o total gravado.
func (s *ServiceAgentAuditStore) Append(organizationID string, entries []ServiceAgentAuditEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	existing, err := s.Load(organizationID)
	if err != nil {
		return 0, err
	}
	existing = append(existing, entries...)
	if err := s.Save(organizationID, existing); err != nil {
		return 0, err
	}
	sort.SliceStable(existing, func(i, j int) bool { return existing[i].RecordedAt.Before(existing[j].RecordedAt) })
	if len(existing) > s.MaxPerOrganization {
		return s.MaxPerOrganization, nil
	}
	return len(existing), nil
}

// serviceAgentAuditRoot devolve o diretório da trilha dentro da raiz de dados.
func serviceAgentAuditRoot(dataRoot string) string {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return ""
	}
	return filepath.Join(dataRoot, ".agent-service-audit")
}
