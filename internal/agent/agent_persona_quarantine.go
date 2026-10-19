// Store de quarentena: o que a triagem decidiu, guardado entre pedidos (FASE 30.2).
//
// # PORQUE ISTO FALTAVA
//
// A triagem da ronda anterior era stateless: devolvia a decisao e esquecia-a. Um estado que so' existe na
// resposta de um pedido nao e' um estado -- e' um aviso, e um aviso que ninguem volta a ver.
//
//	*Quarentena e' um estado, nao um log.*
//
// A quarentena guarda a persona retida, o relatorio que a reteve e QUEM e QUANDO a libertou.
//
// A LIBERTACAO NAO E' UMA PROMOCAO
//
// Sair da quarentena so' vai para `SECURITY_REVIEWED`: e' a afirmacao de que alguem olhou para os achados e
// os considerou explicaveis. NAO e' `PRODUCTION_APPROVED` -- a promocao continua a exigir as etapas de
// `TOOLS_RESOLVED`, avaliacao e aprovacao humana separada.
//
//	*Uma revisao de seguranca que aprova para producao nao e' uma revisao: e' um atalho.*
//
// A guarda de transicao e' a mesma da maquina de estados (`AllowedEvalTransition`), chamada aqui em vez de
// reimplementada: duas verdades sobre a mesma transicao um dia divergem.
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrQuarentenaNaoEncontrada cobre um id de quarentena que nao existe.
var ErrQuarentenaNaoEncontrada = errors.New("agent persona quarantine entry not found")

// ErrLibertacaoRecusada cobre uma libertacao sem revisor ou com transicao invalida.
var ErrLibertacaoRecusada = errors.New("agent persona quarantine release was rejected")

// QuarantineRecord e' uma persona retida.
type QuarantineRecord struct {
	ID string `json:"id"`
	// Scope e' o tenant. A quarentena e' por organizacao: uma persona retida numa nao aparece noutra.
	Scope string `json:"scope"`
	// PersonaID e' a identidade que veio no manifesto. Nao e' chave unica: a mesma persona pode ser
	// reimportada e o relatorio pode mudar.
	PersonaID string           `json:"persona_id"`
	State     EvalState        `json:"state"`
	Report    QuarantineReport `json:"report"`
	CreatedAt string           `json:"created_at"`
	// ReviewedBy/ReviewedAt ficam vazios ate' alguem libertar. O campo vazio e' a informacao: ninguem reviu.
	ReviewedBy string `json:"reviewed_by,omitempty"`
	ReviewedAt string `json:"reviewed_at,omitempty"`
}

// QuarantineStore guarda as entradas por escopo.
type QuarantineStore struct {
	mu      sync.Mutex
	root    string
	byScope map[string]map[string]QuarantineRecord
}

// NewQuarantineStore abre (ou cria) o store. Raiz vazia = so' em memoria, como no PersonaLedger.
func NewQuarantineStore(root string) (*QuarantineStore, error) {
	s := &QuarantineStore{root: strings.TrimSpace(root), byScope: map[string]map[string]QuarantineRecord{}}
	if s.root != "" {
		if err := os.MkdirAll(filepath.Join(s.root, "personas", "quarantine"), 0o755); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *QuarantineStore) scopePath(scope string) string {
	return filepath.Join(s.root, "personas", "quarantine", sanitizeIDComponent(scope)+".json")
}

// quarantineID deriva o id do CONTEUDO retido: persona + achados. Reimportar a MESMA persona com os MESMOS
// achados devolve o mesmo id.
func quarantineID(scope string, personaID string, r QuarantineReport) string {
	var b strings.Builder
	b.WriteString("v1|")
	b.WriteString(scope)
	b.WriteString("|")
	b.WriteString(personaID)
	b.WriteString("|")
	for _, f := range r.Findings {
		b.WriteString(f.Field)
		b.WriteString("#")
		b.WriteString(f.Pattern)
		b.WriteString("#")
		b.WriteString(string(f.Severity))
		b.WriteString(";")
	}
	return hashBytes([]byte(b.String()))[:24]
}

// loadLocked garante que o escopo esta' carregado do disco. Chamador segura o mutex.
func (s *QuarantineStore) loadLocked(scope string) error {
	if _, ok := s.byScope[scope]; ok {
		return nil
	}
	s.byScope[scope] = map[string]QuarantineRecord{}
	if s.root == "" {
		return nil
	}
	bruto, err := os.ReadFile(s.scopePath(scope))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var lista []QuarantineRecord
	if err := json.Unmarshal(bruto, &lista); err != nil {
		return fmt.Errorf("%w: %s", ErrPersonaLedgerCorrupt, err.Error())
	}
	for _, r := range lista {
		s.byScope[scope][r.ID] = r
	}
	return nil
}

// persistLocked grava o escopo ordenado. Chamador segura o mutex.
//
// Duas gravacoes do mesmo estado tem de produzir os mesmos bytes -- senao um teste de idempotencia compara
// o tempo, nao o conteudo.
func (s *QuarantineStore) persistLocked(scope string) error {
	if s.root == "" {
		return nil
	}
	escopo := s.byScope[scope]
	chaves := make([]string, 0, len(escopo))
	for chave := range escopo {
		chaves = append(chaves, chave)
	}
	sort.Strings(chaves)
	lista := make([]QuarantineRecord, 0, len(chaves))
	for _, k := range chaves {
		lista = append(lista, escopo[k])
	}
	b, err := json.MarshalIndent(lista, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.scopePath(scope)), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(s.scopePath(scope), b)
}

// Enter retem uma persona. Idempotente pelo id derivado do conteudo.
//
// Uma persona SEM achados nao entra em quarentena: nao ha' nada para reter. O chamador deve usar o estado
// devolvido por `ImportDecision` para decidir se chama `Enter`.
func (s *QuarantineStore) Enter(scope string, persona AgentPersona, relatorio QuarantineReport, agora time.Time) (QuarantineRecord, bool, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return QuarantineRecord{}, false, fmt.Errorf("%w: scope is required", ErrPersonaInvalid)
	}
	if err := requireOrganization(scope); err != nil {
		return QuarantineRecord{}, false, err
	}
	if !relatorio.Blocked && !relatorio.NeedsReview {
		return QuarantineRecord{}, false, fmt.Errorf("%w: nothing to quarantine (no findings)", ErrImportRecusado)
	}
	id := quarantineID(scope, persona.ID, relatorio)

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(scope); err != nil {
		return QuarantineRecord{}, false, err
	}
	if existente, ok := s.byScope[scope][id]; ok {
		// *Idempotencia de verdade e' devolver o MESMO RECURSO.*
		return existente, false, nil
	}
	reg := QuarantineRecord{
		ID:        id,
		Scope:     scope,
		PersonaID: persona.ID,
		State:     EvalQuarantined,
		Report:    relatorio,
		CreatedAt: agora.UTC().Format(time.RFC3339),
	}
	s.byScope[scope][id] = reg
	if err := s.persistLocked(scope); err != nil {
		return QuarantineRecord{}, false, err
	}
	return reg, true, nil
}

// List devolve as entradas de um escopo, ordenadas por id.
func (s *QuarantineStore) List(scope string) ([]QuarantineRecord, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return nil, fmt.Errorf("%w: scope is required", ErrPersonaInvalid)
	}
	if err := requireOrganization(scope); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(scope); err != nil {
		return nil, err
	}
	escopo := s.byScope[scope]
	chaves := make([]string, 0, len(escopo))
	for chave := range escopo {
		chaves = append(chaves, chave)
	}
	sort.Strings(chaves)
	out := make([]QuarantineRecord, 0, len(chaves))
	for _, k := range chaves {
		out = append(out, escopo[k])
	}
	return out, nil
}

// Get devolve uma entrada pelo id.
func (s *QuarantineStore) Get(scope, id string) (QuarantineRecord, error) {
	scope = strings.TrimSpace(scope)
	id = strings.TrimSpace(id)
	if scope == "" || id == "" {
		return QuarantineRecord{}, fmt.Errorf("%w: scope and id are required", ErrPersonaInvalid)
	}
	if err := requireOrganization(scope); err != nil {
		return QuarantineRecord{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(scope); err != nil {
		return QuarantineRecord{}, err
	}
	reg, ok := s.byScope[scope][id]
	if !ok {
		return QuarantineRecord{}, fmt.Errorf("%w: %s", ErrQuarentenaNaoEncontrada, id)
	}
	return reg, nil
}

// Release liberta uma entrada para `SECURITY_REVIEWED`.
//
// Exige revisor NAO VAZIO: uma libertacao sem autor nao e' uma revisao, e' um campo em branco. E exige que a
// transicao seja permitida pela maquina de estados -- a mesma funcao que a governa.
func (s *QuarantineStore) Release(scope, id, revisor string, agora time.Time) (QuarantineRecord, error) {
	scope = strings.TrimSpace(scope)
	id = strings.TrimSpace(id)
	revisor = strings.TrimSpace(revisor)
	if scope == "" || id == "" {
		return QuarantineRecord{}, fmt.Errorf("%w: scope and id are required", ErrPersonaInvalid)
	}
	if revisor == "" {
		return QuarantineRecord{}, fmt.Errorf("%w: a review needs a reviewer", ErrLibertacaoRecusada)
	}
	if err := requireOrganization(scope); err != nil {
		return QuarantineRecord{}, err
	}
	// A transicao e' a mesma da maquina de estados. *Duas verdades sobre a mesma transicao um dia divergem.*
	if !AllowedEvalTransition(EvalQuarantined, EvalSecurityReviewed) {
		return QuarantineRecord{}, fmt.Errorf("%w: QUARANTINE -> SECURITY_REVIEWED is not allowed", ErrLibertacaoRecusada)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(scope); err != nil {
		return QuarantineRecord{}, err
	}
	reg, ok := s.byScope[scope][id]
	if !ok {
		return QuarantineRecord{}, fmt.Errorf("%w: %s", ErrQuarentenaNaoEncontrada, id)
	}
	// Uma entrada ja' libertada nao volta a ser libertada: *uma transicao que nao muda nada nao e' uma
	// transicao.*
	if reg.State != EvalQuarantined {
		return QuarantineRecord{}, fmt.Errorf("%w: %s is already %s", ErrLibertacaoRecusada, id, reg.State)
	}
	reg.State = EvalSecurityReviewed
	reg.ReviewedBy = revisor
	reg.ReviewedAt = agora.UTC().Format(time.RFC3339)
	s.byScope[scope][id] = reg
	if err := s.persistLocked(scope); err != nil {
		return QuarantineRecord{}, err
	}
	return reg, nil
}
