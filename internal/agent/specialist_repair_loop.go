package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// P0-E / T4 — LIGAÇÃO AO LAÇO REAL DE REPARO (E-348).
//
// **MEDIDO**: `coding_git_cycle.go` já tem o laço de reparo de verdade —
// `ExecuteRepairLoop(ctx, r *Runtime, mission Mission, step *Step, testOpts TestRunOptions, repairFn func(attempt int,
// lastOutput string) error, maxTries int) (TestRunResult, error)` — que roda os testes do projeto e, **em falha, chama
// `repairFn`** até `maxTries`. O `r *Runtime` pode ser `nil` (o próprio código confere) e o laço usa
// `mission.Workspace`.
//
// O que faltava **não** era o laço: era **ligar** o laço a cada especialista **dentro da worktree dele** e depois
// entregar o resultado ao **coordenador** (segmento + conflito). Esta fatia faz exatamente essa ligação, **sem
// duplicar** nada:
//
//	especialista (fixer) → worktree isolada → ExecuteRepairLoop (testes de verdade) → GetWorktreeDiff
//	                     → ValidateSpecialistPlan (segmento/conflito) → CoordinateSpecialistMerges
//
// ***Um laço de reparo que nunca é ligado a um especialista isolado é um laço que conserta sempre o mesmo lugar.***

// SpecialistTask é o trabalho de UM especialista, limitado ao seu SEGMENTO.
type SpecialistTask struct {
	Name        string
	Segment     []string
	Instruction string
}

var (
	// ErrSpecialistInvalid recusa tarefa de especialista malformada.
	ErrSpecialistInvalid = errors.New("specialist task is invalid")
	// ErrSpecialistFixerInvalid recusa fixer ausente ou que não produz nada.
	ErrSpecialistFixerInvalid = errors.New("specialist fixer is invalid")
	// ErrSpecialistLoopFailed declara que o laço não ficou verde (resultado honesto, não exceção).
	ErrSpecialistLoopFailed = errors.New("specialist repair loop did not reach green")
)

// SpecialistFixer é quem escreve a correção DENTRO da worktree do especialista.
type SpecialistFixer interface {
	Fix(ctx context.Context, worktreeDir string, task SpecialistTask, attempt int, lastOutput string) error
}

// SpecialistLoopOptions são os limites declarados da execução ligada.
type SpecialistLoopOptions struct {
	MaxRepairTries int
	TestTimeout    time.Duration
	Limits         SpecialistPlanLimits
	// Runtime (opcional) faz o laço existente EMITIR eventos de observabilidade. Sem ele, os eventos NÃO são emitidos —
	// limite declarado desde E-348, agora fechável pelo chamador.
	Runtime *Runtime
	// MissionID/OrganizationID identificam a missão dos eventos.
	MissionID      string
	OrganizationID string
}

// DefaultSpecialistLoopOptions devolve limites explícitos.
func DefaultSpecialistLoopOptions() SpecialistLoopOptions {
	return SpecialistLoopOptions{MaxRepairTries: 2, TestTimeout: 10 * time.Minute, Limits: DefaultSpecialistPlanLimits()}
}

func (o SpecialistLoopOptions) normalizar() SpecialistLoopOptions {
	padrao := DefaultSpecialistLoopOptions()
	if o.MaxRepairTries <= 0 {
		o.MaxRepairTries = padrao.MaxRepairTries
	}
	if o.TestTimeout <= 0 {
		o.TestTimeout = padrao.TestTimeout
	}
	if o.Limits.MaxSpecialists <= 0 {
		o.Limits = padrao.Limits
	}
	return o
}

// SpecialistLoopOutcome é o resultado por especialista, com o que foi MEDIDO no laço.
type SpecialistLoopOutcome struct {
	Specialist string   `json:"specialist"`
	Worktree   string   `json:"worktree,omitempty"`
	Passed     bool     `json:"passed"`
	ExitCode   int      `json:"exit_code"`
	FailedTest []string `json:"failed_tests,omitempty"`
	FilesDiff  []string `json:"files_changed,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// SpecialistLoopReport é o resultado da execução completa (laços + coordenação).
type SpecialistLoopReport struct {
	Outcomes     []SpecialistLoopOutcome `json:"outcomes"`
	MergedFiles  []string                `json:"merged_files,omitempty"`
	Conflicts    []string                `json:"conflicts,omitempty"`
	Verdict      string                  `json:"verdict"`
	DurationSecs float64                 `json:"duration_seconds"`
}

// RunSpecialistRepairLoops cria uma worktree por especialista, roda o LAÇO DE REPARO REAL em cada uma e coordena o merge.
func RunSpecialistRepairLoops(ctx context.Context, repoDir, dataDir string, tarefas []SpecialistTask, fixer SpecialistFixer, opcoes SpecialistLoopOptions) (SpecialistLoopReport, error) {
	inicio := time.Now()
	opcoes = opcoes.normalizar()
	relatorio := SpecialistLoopReport{Verdict: "NAO_EXECUTADO"}
	if strings.TrimSpace(repoDir) == "" || fixer == nil {
		return relatorio, fmt.Errorf("%w: repositório e fixer são obrigatórios", ErrSpecialistFixerInvalid)
	}
	if len(tarefas) == 0 {
		return relatorio, fmt.Errorf("%w: nenhum especialista declarado", ErrSpecialistInvalid)
	}
	for _, tarefa := range tarefas {
		if strings.TrimSpace(tarefa.Name) == "" || len(tarefa.Segment) == 0 {
			return relatorio, fmt.Errorf("%w: especialista %q sem nome ou sem segmento", ErrSpecialistInvalid, tarefa.Name)
		}
	}

	sessoes := map[string]*GitWorktreeSession{}
	planos := make([]SpecialistMergePlan, 0, len(tarefas))
	defer func() {
		for _, sessao := range sessoes {
			_ = RemoveGitWorktree(ctx, sessao)
		}
	}()

	for _, tarefa := range tarefas {
		resultado := SpecialistLoopOutcome{Specialist: tarefa.Name}
		// 1. Worktree ISOLADA por especialista (mecanismo do Git, não pasta simulada).
		sessao, err := CreateGitWorktree(ctx, repoDir, dataDir, "spec-"+tarefa.Name, "hades/spec-"+tarefa.Name)
		if err != nil {
			resultado.Error = "worktree: " + err.Error()
			relatorio.Outcomes = append(relatorio.Outcomes, resultado)
			continue
		}
		resultado.Worktree = sessao.WorktreeDir
		sessoes[tarefa.Name] = sessao

		// 2. LAÇO DE REPARO REAL: os testes do PROJETO rodam dentro da worktree; em falha, o fixer é chamado.
		reparos := 0
		testOpts := TestRunOptions{Framework: "go", TimeoutSeconds: int(opcoes.TestTimeout.Seconds())}
		missao := Mission{Workspace: sessao.WorktreeDir, ID: opcoes.MissionID, OrganizationID: opcoes.OrganizationID}
		resultadoTeste, err := ExecuteRepairLoop(ctx, opcoes.Runtime,
			missao, &Step{ID: "spec-" + tarefa.Name}, testOpts,
			func(attempt int, lastOutput string) error {
				reparos++
				return fixer.Fix(ctx, sessao.WorktreeDir, tarefa, attempt, lastOutput)
			}, opcoes.MaxRepairTries)
		resultado.Passed = resultadoTeste.Passed
		resultado.ExitCode = resultadoTeste.ExitCode
		resultado.FailedTest = resultadoTeste.FailedTests
		if err != nil && resultadoTeste.ExitCode == 0 && !resultadoTeste.Passed {
			resultado.Error = err.Error()
		}
		// 3. O diff VEM DA MEDIÇÃO existente (não de suposição do que o fixer escreveu).
		aprovacao, errDiff := GetWorktreeDiff(ctx, sessao)
		if errDiff != nil {
			resultado.Error = "diff: " + errDiff.Error()
			relatorio.Outcomes = append(relatorio.Outcomes, resultado)
			continue
		}
		resultado.FilesDiff = aprovacao.FilesChanged
		planos = append(planos, SpecialistMergePlan{SessionID: tarefa.Name, Segment: tarefa.Segment, Approval: aprovacao})
		relatorio.Outcomes = append(relatorio.Outcomes, resultado)
	}

	// 4. COORDENAÇÃO: valida TUDO (segmento + conflito) antes de mesclar qualquer coisa.
	if err := ValidateSpecialistPlan(planos, opcoes.Limits); err != nil {
		relatorio.Verdict = "RECUSADO"
		relatorio.DurationSecs = time.Since(inicio).Seconds()
		if errors.Is(err, ErrSpecialistConflict) {
			relatorio.Conflicts = []string{err.Error()}
		}
		return relatorio, err
	}
	mesclados, err := CoordinateSpecialistMerges(ctx, sessoes, planos, opcoes.Limits)
	if err != nil {
		relatorio.Verdict = "RECUSADO"
		relatorio.DurationSecs = time.Since(inicio).Seconds()
		return relatorio, err
	}
	for _, item := range mesclados {
		if item.Merged {
			for _, plano := range planos {
				if plano.SessionID == item.SessionID && plano.Approval != nil {
					relatorio.MergedFiles = append(relatorio.MergedFiles, plano.Approval.FilesChanged...)
				}
			}
		}
	}
	sort.Strings(relatorio.MergedFiles)
	relatorio.DurationSecs = time.Since(inicio).Seconds()
	relatorio.Verdict = "MESCLADO"
	for _, resultado := range relatorio.Outcomes {
		if !resultado.Passed {
			relatorio.Verdict = "PARCIAL"
			break
		}
	}
	return relatorio, nil
}

// fixerDeArquivos escreve arquivos fixos dentro da worktree — fixer determinístico para teste do mecanismo.
type fixerDeArquivos struct {
	arquivos map[string]string
	chamadas int
}

func (f *fixerDeArquivos) Fix(_ context.Context, worktreeDir string, _ SpecialistTask, _ int, _ string) error {
	f.chamadas++
	nomes := make([]string, 0, len(f.arquivos))
	for nome := range f.arquivos {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)
	for _, nome := range nomes {
		caminho := filepath.Join(worktreeDir, nome)
		if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(caminho, []byte(f.arquivos[nome]), 0o600); err != nil {
			return err
		}
	}
	return nil
}
