package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------------------------------------------
// REGISTRO DE CONTRATOS VERSIONADOS (E-276)
// ---------------------------------------------------------------------------------------------------------------
//
// O estágio 3 exige "contratos versionados, testes de contrato e registro de migrações". Motores reais existiam;
// o REGISTRO não. Este arquivo é o registro: cada componente canônico tem versão, dono, estado e histórico de
// migração. E subir versão sem registrar a migração é RECUSADO — uma versão que sobe sozinha mente sobre
// compatibilidade.

// ContractChange classificam o impacto de uma versão.
const (
	ContractChangeAditivo   = "aditivo"
	ContractChangeQuebra    = "quebra"
	ContractChangeCorrecao  = "correcao"
	ContractStatusEstavel   = "estavel"
	ContractStatusBeta      = "beta"
	ContractStatusPlanejado = "planejado"
)

var (
	// ErrContractInvalid recusa componente desconhecido, versão malformada, dono ausente ou duplicata.
	ErrContractInvalid = errors.New("component contract invalid")
	// ErrContractMigrationRequired recusa subir versão sem registrar a migração.
	ErrContractMigrationRequired = errors.New("component contract version bump requires a migration record")
	// ErrContractVersionRegression recusa voltar a versão: apaga a história de compatibilidade.
	ErrContractVersionRegression = errors.New("component contract version cannot go backwards")
	// ErrContractBreakingWithoutImpact recusa mudança que quebra sem declarar quem quebra.
	ErrContractBreakingWithoutImpact = errors.New("breaking contract change must declare affected consumers")
	// ErrContractUnknown recusa operar sobre componente não registrado.
	ErrContractUnknown = errors.New("component contract not registered")
)

// CanonicalComponentIDs são os 16 componentes canônicos do estágio 3. A lista é FECHADA: um componente fora dela
// não entra no registro, senão "canônico" deixaria de significar qualquer coisa.
func CanonicalComponentIDs() []string {
	return []string{
		"hades_core",
		"mission_control",
		"model_gateway",
		"harness_runtime",
		"agent_orchestrator",
		"tool_execution_runtime",
		"mcp_manager",
		"plugin_manager",
		"skill_manager",
		"memory_context_engine",
		"computer_use_runtime",
		"product_builder",
		"workflow_automation_engine",
		"security_policy_engine",
		"artifact_delivery_engine",
		"observability_evaluation_engine",
	}
}

// ContractMigration é o REGISTRO de uma mudança de versão: o que mudou, quem decidiu e quem é afetado.
type ContractMigration struct {
	FromVersion string    `json:"from_version"`
	ToVersion   string    `json:"to_version"`
	Change      string    `json:"change"`
	Reason      string    `json:"reason"`
	ActorID     string    `json:"actor_id"`
	Affected    []string  `json:"affected,omitempty"`
	RecordedAt  time.Time `json:"recorded_at"`
}

// ComponentContract é o contrato versionado de UM componente canônico.
type ComponentContract struct {
	Component  string              `json:"component"`
	Version    string              `json:"version"`
	Owner      string              `json:"owner"`
	Status     string              `json:"status"`
	Summary    string              `json:"summary,omitempty"`
	Migrations []ContractMigration `json:"migrations,omitempty"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

type componentContractFile struct {
	Contracts []ComponentContract `json:"contracts"`
}

// ContractRegistry guarda os contratos. É PLATAFORMA (global), não por organização: os contratos descrevem o
// produto que TODOS usam. Dado de organização nunca entra aqui.
type ContractRegistry struct {
	root      string
	contracts map[string]ComponentContract
}

// NewContractRegistry abre (ou cria) o registro em `root`. `root` vazio mantém tudo em memória (declarado).
func NewContractRegistry(root string) (*ContractRegistry, error) {
	registro := &ContractRegistry{root: strings.TrimSpace(root), contracts: map[string]ComponentContract{}}
	if err := registro.load(); err != nil {
		return nil, err
	}
	return registro, nil
}

func (r *ContractRegistry) path() string {
	if r.root == "" {
		return ""
	}
	return filepath.Join(r.root, "component_contracts.json")
}

func (r *ContractRegistry) load() error {
	caminho := r.path()
	if caminho == "" {
		return nil
	}
	stored := componentContractFile{}
	if err := readJSON(caminho, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("ler contratos: %w", err)
	}
	for _, contrato := range stored.Contracts {
		r.contracts[contrato.Component] = contrato
	}
	return nil
}

func (r *ContractRegistry) save() error {
	caminho := r.path()
	if caminho == "" {
		return nil
	}
	ids := make([]string, 0, len(r.contracts))
	for id := range r.contracts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	stored := componentContractFile{}
	for _, id := range ids {
		stored.Contracts = append(stored.Contracts, r.contracts[id])
	}
	return writeJSONAtomic(caminho, stored)
}

func canonicalComponent(id string) bool {
	for _, canonico := range CanonicalComponentIDs() {
		if canonico == id {
			return true
		}
	}
	return false
}

// parseVersion aceita APENAS `maior.menor` com inteiros não negativos. Versão "v1", "1.2.3" ou "beta" não entram:
// formato frouxo impede comparar e é justamente a comparação que sustenta a promessa de compatibilidade.
func parseVersion(version string) (int, int, error) {
	partes := strings.Split(strings.TrimSpace(version), ".")
	if len(partes) != 2 {
		return 0, 0, fmt.Errorf("%w: versão %q precisa ser maior.menor", ErrContractInvalid, version)
	}
	maior, err := strconv.Atoi(partes[0])
	if err != nil || maior < 0 {
		return 0, 0, fmt.Errorf("%w: maior inválido em %q", ErrContractInvalid, version)
	}
	menor, err := strconv.Atoi(partes[1])
	if err != nil || menor < 0 {
		return 0, 0, fmt.Errorf("%w: menor inválido em %q", ErrContractInvalid, version)
	}
	return maior, menor, nil
}

// Register registra um contrato NOVO. Componente já registrado não é sobrescrito: versão se move por Bump.
func (r *ContractRegistry) Register(contract ComponentContract, now time.Time) (ComponentContract, error) {
	component := strings.TrimSpace(contract.Component)
	if component == "" {
		return ComponentContract{}, fmt.Errorf("%w: componente é obrigatório", ErrContractInvalid)
	}
	if !canonicalComponent(component) {
		return ComponentContract{}, fmt.Errorf("%w: %q não é componente canônico", ErrContractInvalid, component)
	}
	if _, _, err := parseVersion(contract.Version); err != nil {
		return ComponentContract{}, err
	}
	if strings.TrimSpace(contract.Owner) == "" {
		return ComponentContract{}, fmt.Errorf("%w: %s sem dono", ErrContractInvalid, component)
	}
	if _, existe := r.contracts[component]; existe {
		return ComponentContract{}, fmt.Errorf("%w: %s já registrado (use Bump)", ErrContractInvalid, component)
	}
	status := strings.TrimSpace(contract.Status)
	if status == "" {
		status = ContractStatusEstavel
	}
	gravado := contract
	gravado.Component = component
	gravado.Status = status
	gravado.UpdatedAt = now.UTC()
	r.contracts[component] = gravado
	if err := r.save(); err != nil {
		delete(r.contracts, component)
		return ComponentContract{}, err
	}
	return gravado, nil
}

// Bump sobe a versão de um contrato EXIGINDO o registro da migração. É a propriedade central do estágio 3.
func (r *ContractRegistry) Bump(component, newVersion, change, reason, actorID string, affected []string, now time.Time) (ComponentContract, error) {
	component = strings.TrimSpace(component)
	atual, existe := r.contracts[component]
	if !existe {
		return ComponentContract{}, fmt.Errorf("%w: %s", ErrContractUnknown, component)
	}
	if _, _, err := parseVersion(newVersion); err != nil {
		return ComponentContract{}, err
	}
	// SUBIR VERSÃO SEM MIGRAÇÃO É RECUSADO. Um número maior sem registro diz "nada mudou para você" enquanto algo
	// mudou - e quem já integrou só descobre em produção.
	if strings.TrimSpace(reason) == "" || strings.TrimSpace(actorID) == "" {
		return ComponentContract{}, fmt.Errorf("%w: %s -> %s", ErrContractMigrationRequired, atual.Version, newVersion)
	}
	mudanca := strings.TrimSpace(change)
	if mudanca == "" {
		mudanca = ContractChangeAditivo
	}
	switch mudanca {
	case ContractChangeAditivo, ContractChangeQuebra, ContractChangeCorrecao:
	default:
		return ComponentContract{}, fmt.Errorf("%w: mudança %q desconhecida", ErrContractInvalid, mudanca)
	}
	if mudanca == ContractChangeQuebra && len(affected) == 0 {
		// QUEBRA SEM DIZER QUEM QUEBRA é a mudança que derruba produção no sábado.
		return ComponentContract{}, fmt.Errorf("%w: %s", ErrContractBreakingWithoutImpact, component)
	}
	maiorAtual, menorAtual, _ := parseVersion(atual.Version)
	maiorNovo, menorNovo, _ := parseVersion(newVersion)
	if maiorNovo < maiorAtual || (maiorNovo == maiorAtual && menorNovo < menorAtual) {
		return ComponentContract{}, fmt.Errorf("%w: %s (%s -> %s)", ErrContractVersionRegression, component, atual.Version, newVersion)
	}
	if maiorNovo == maiorAtual && menorNovo == menorAtual {
		return ComponentContract{}, fmt.Errorf("%w: %s já está em %s", ErrContractInvalid, component, newVersion)
	}
	anterior := atual
	// HISTÓRICO APPEND-ONLY: a migração anterior é PRESERVADA. Reescrever apagaria a trilha de quem já migrou.
	atual.Migrations = append(append([]ContractMigration(nil), anterior.Migrations...), ContractMigration{
		FromVersion: anterior.Version, ToVersion: newVersion, Change: mudanca, Reason: strings.TrimSpace(reason),
		ActorID: strings.TrimSpace(actorID), Affected: append([]string(nil), affected...), RecordedAt: now.UTC(),
	})
	atual.Version = newVersion
	atual.UpdatedAt = now.UTC()
	r.contracts[component] = atual
	if err := r.save(); err != nil {
		r.contracts[component] = anterior
		return ComponentContract{}, err
	}
	return atual, nil
}

// Contracts devolve os contratos ordenados por componente.
func (r *ContractRegistry) Contracts() []ComponentContract {
	ids := make([]string, 0, len(r.contracts))
	for id := range r.contracts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	contratos := make([]ComponentContract, 0, len(ids))
	for _, id := range ids {
		contratos = append(contratos, r.contracts[id])
	}
	return contratos
}

// Contract devolve UM contrato.
func (r *ContractRegistry) Contract(component string) (ComponentContract, bool) {
	contrato, ok := r.contracts[strings.TrimSpace(component)]
	return contrato, ok
}

// MissingCanonicalComponents lista os componentes canônicos AINDA sem contrato. Um registro parcial que se diz
// completo é pior do que nenhum: a lista precisa poder ser CONFERIDA.
func (r *ContractRegistry) MissingCanonicalComponents() []string {
	faltando := []string{}
	for _, canonico := range CanonicalComponentIDs() {
		if _, existe := r.contracts[canonico]; !existe {
			faltando = append(faltando, canonico)
		}
	}
	return faltando
}
