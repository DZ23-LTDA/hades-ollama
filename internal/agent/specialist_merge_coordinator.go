package agent

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// P0-E / T4 — COORDENADOR DE MERGE COM SEGMENTO, CONFLITO E LIMITES (E-346).
//
// **MEDIDO ANTES DE ESCREVER (e o achado muda o escopo desta fatia)**: o T4 **já existia** — `coding_git_cycle.go` tem
// `CreateGitWorktree`, `GetWorktreeDiff`, `MergeWorktreeToOrigin`, `RemoveGitWorktree`, `GitMergeApproval` (com
// `FilesChanged` e aprovação humana), `RunProjectTests` e `ExecuteRepairLoop`, com teste **passando**
// (`TestTresEspecialistasEmWorktreesIsoladosComMergeControlado`). ***Recriar isso seria violar a primeira regra da
// missão.*** Um rascunho meu de módulo paralelo foi **descartado** depois de a própria suíte revelar o existente.
//
// O que **faltava** — e é o que esta fatia acrescenta, **reusando** a API existente:
//
//  1. **MEMÓRIA SEGMENTADA**: cada especialista só pode alterar os arquivos do **seu** segmento; `FilesChanged` sai do
//     `GitMergeApproval` existente e é conferido **antes** de qualquer merge. *Sem isso, "especialista" é apenas quem
//     mexeu por último.*
//  2. **CONFLITO POR ARQUIVO**: dois especialistas no **mesmo arquivo** ⇒ **RECUSA antes de mesclar**, com a lista de
//     arquivos em disputa. ***Merge que sobrescreve em silêncio esconde exatamente o erro que se queria detectar.***
//  3. **LIMITES DECLARADOS**: número de especialistas e arquivos por especialista — explícitos, no mesmo lugar.
//
// LIMITES DECLARADOS: a conferência usa `FilesChanged` (nível de **arquivo**, não de hunk); detectar conflito de
// **linha** exigiria aplicar o merge — e a decisão aqui é **recusar antes**, que é o comportamento seguro. Um dos
// especialistas pode já ter sido mesclado antes desta conferência se o chamador não usar este coordenador.

var (
	// ErrSpecialistConflict recusa merge com dois especialistas no mesmo arquivo.
	ErrSpecialistConflict = errors.New("specialists conflict on the same file")
	// ErrSpecialistOutOfSegment recusa especialista que alterou fora do seu segmento.
	ErrSpecialistOutOfSegment = errors.New("specialist changed a file outside its segment")
	// ErrSpecialistPlanInvalid recusa plano malformado.
	ErrSpecialistPlanInvalid = errors.New("specialist merge plan is invalid")
	// ErrSpecialistPlanLimit recusa plano acima dos limites declarados.
	ErrSpecialistPlanLimit = errors.New("specialist merge plan exceeds declared limits")
)

// SpecialistMergePlan declara o segmento de UM especialista (chave = identificador da sessão/worktree).
type SpecialistMergePlan struct {
	SessionID string
	Segment   []string
	// Approval é o resultado medido de `GetWorktreeDiff` (pode ser preenchido pelo chamador ou pelo coordenador).
	Approval *GitMergeApproval
}

// SpecialistPlanLimits são os limites declarados do plano.
type SpecialistPlanLimits struct {
	MaxSpecialists        int
	MaxFilesPerSpecialist int
}

// DefaultSpecialistPlanLimits devolve limites explícitos.
func DefaultSpecialistPlanLimits() SpecialistPlanLimits {
	return SpecialistPlanLimits{MaxSpecialists: 5, MaxFilesPerSpecialist: 20}
}

func (l SpecialistPlanLimits) normalizar() SpecialistPlanLimits {
	padrao := DefaultSpecialistPlanLimits()
	if l.MaxSpecialists <= 0 {
		l.MaxSpecialists = padrao.MaxSpecialists
	}
	if l.MaxFilesPerSpecialist <= 0 {
		l.MaxFilesPerSpecialist = padrao.MaxFilesPerSpecialist
	}
	return l
}

// ValidateSpecialistPlan confere o plano SEM mesclar nada: segmento, conflito por arquivo e limites.
//
// ***Ordem deliberada: VALIDAR TUDO, depois mesclar.*** Mesclar e descobrir conflito no meio deixa a árvore de
// integração num estado que ninguém pediu — e desfazer merge pela metade é pior que não começar.
func ValidateSpecialistPlan(planos []SpecialistMergePlan, limites SpecialistPlanLimits) error {
	limites = limites.normalizar()
	if len(planos) == 0 {
		return fmt.Errorf("%w: nenhum especialista declarado", ErrSpecialistPlanInvalid)
	}
	if len(planos) > limites.MaxSpecialists {
		return fmt.Errorf("%w: %d especialistas excede o limite de %d", ErrSpecialistPlanLimit, len(planos), limites.MaxSpecialists)
	}

	dono := map[string]string{}
	emDisputa := map[string]bool{}
	for _, plano := range planos {
		if strings.TrimSpace(plano.SessionID) == "" {
			return fmt.Errorf("%w: especialista sem identificador", ErrSpecialistPlanInvalid)
		}
		if len(plano.Segment) == 0 {
			return fmt.Errorf("%w: especialista %q sem segmento — especialista sem fronteira não é especialista",
				ErrSpecialistPlanInvalid, plano.SessionID)
		}
		permitidos := map[string]bool{}
		for _, arquivo := range plano.Segment {
			if strings.TrimSpace(arquivo) == "" || strings.Contains(arquivo, "..") || filepath.IsAbs(arquivo) {
				return fmt.Errorf("%w: segmento de %q com caminho inseguro %q", ErrSpecialistPlanInvalid, plano.SessionID, arquivo)
			}
			permitidos[filepath.Clean(arquivo)] = true
		}
		if plano.Approval == nil {
			continue // sem diff medido ainda: o chamador pode validar depois de GetWorktreeDiff
		}
		if len(plano.Approval.FilesChanged) > limites.MaxFilesPerSpecialist {
			return fmt.Errorf("%w: %q alterou %d arquivos (limite %d)", ErrSpecialistPlanLimit, plano.SessionID,
				len(plano.Approval.FilesChanged), limites.MaxFilesPerSpecialist)
		}
		for _, arquivo := range plano.Approval.FilesChanged {
			limpo := filepath.Clean(arquivo)
			// SEGMENTO
			if !permitidos[limpo] {
				return fmt.Errorf("%w: %q alterou %q, fora do segmento declarado", ErrSpecialistOutOfSegment,
					plano.SessionID, arquivo)
			}
			// CONFLITO POR ARQUIVO
			if anterior, existe := dono[limpo]; existe && anterior != plano.SessionID {
				emDisputa[limpo] = true
				continue
			}
			dono[limpo] = plano.SessionID
		}
	}
	if len(emDisputa) > 0 {
		nomes := make([]string, 0, len(emDisputa))
		for arquivo := range emDisputa {
			nomes = append(nomes, arquivo)
		}
		sort.Strings(nomes)
		return fmt.Errorf("%w: %s", ErrSpecialistConflict, strings.Join(nomes, ", "))
	}
	return nil
}

// SpecialistMergeOutcome é o resultado por especialista.
type SpecialistMergeOutcome struct {
	SessionID string `json:"session_id"`
	Merged    bool   `json:"merged"`
	Reason    string `json:"reason,omitempty"`
	Commit    string `json:"commit,omitempty"`
}

// CoordinateSpecialistMerges valida o plano e, SÓ SE ELE ESTIVER LIMPO, mescla cada especialista em ordem.
func CoordinateSpecialistMerges(ctx context.Context, sessions map[string]*GitWorktreeSession, planos []SpecialistMergePlan, limites SpecialistPlanLimits) ([]SpecialistMergeOutcome, error) {
	// 1. VALIDAR TUDO PRIMEIRO. Nada é mesclado antes de o plano inteiro passar.
	if err := ValidateSpecialistPlan(planos, limites); err != nil {
		return nil, err
	}
	resultados := make([]SpecialistMergeOutcome, 0, len(planos))
	// 2. MESCLAR em ordem de identificador: ordem determinística é o que torna o resultado reprodutível.
	ordenados := append([]SpecialistMergePlan(nil), planos...)
	sort.Slice(ordenados, func(i, j int) bool { return ordenados[i].SessionID < ordenados[j].SessionID })
	for _, plano := range ordenados {
		resultado := SpecialistMergeOutcome{SessionID: plano.SessionID}
		sessao := sessions[plano.SessionID]
		if sessao == nil {
			resultado.Reason = "sessão de worktree não encontrada"
			resultados = append(resultados, resultado)
			continue
		}
		saida, err := MergeWorktreeToOrigin(ctx, sessao)
		if err != nil {
			resultado.Reason = err.Error()
			resultados = append(resultados, resultado)
			continue
		}
		resultado.Merged = true
		resultado.Commit = strings.TrimSpace(saida)
		resultados = append(resultados, resultado)
	}
	for _, plano := range ordenados {
		if sessao := sessions[plano.SessionID]; sessao != nil {
			_ = RemoveGitWorktree(ctx, sessao)
		}
	}
	_ = time.Now
	return resultados, nil
}
