package agent

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

// RE-EMBED ASSÍNCRONO (estágio 35): a rota síncrona de E-159 tem um teto de 200 memórias por chamada justamente
// porque uma requisição não pode virar uma operação que ninguém consegue terminar. Para bases grandes isso
// significava chamar várias vezes e adivinhar o progresso. Aqui a operação vira um JOB com estado observável.
//
// Decisões:
//
//  1. UM job por projeto. Dois jobs no mesmo projeto competiriam pelo mesmo arquivo e o último a gravar
//     apagaria o trabalho do outro — recusar o segundo é mais honesto que "deixar rodar e ver".
//  2. Progresso PERSISTIDO em LOTE: cada lote grava o arquivo. Um processo que morre no meio deixa o trabalho
//     já feito gravado, em vez de perder tudo no fim.
//  3. ESTADO EM MEMÓRIA é declarado: reiniciar o processo perde o registro do job (o que já foi gravado
//     permanece). Persistir o job seria outra feature, não um detalhe escondido.
//  4. Cancelamento é pedido, não promessa: o job verifica entre lotes e termina como `cancelled` com o que já
//     fez contabilizado.
const (
	ReembedJobRunning   = "running"
	ReembedJobDone      = "done"
	ReembedJobFailed    = "failed"
	ReembedJobCancelled = "cancelled"
)

// ErrReembedJobRunning diz que já existe job ativo para o projeto.
var ErrReembedJobRunning = errors.New("já existe um re-embed em andamento para este projeto")

// ErrReembedJobNotFound diz que o job não existe (ou se perdeu com o reinício do processo).
var ErrReembedJobNotFound = errors.New("job de re-embed não encontrado")

// reembedJobBatch é quantas memórias são processadas antes de gravar o arquivo.
//
// Pequeno o suficiente para o progresso ser visível e um crash não custar muito; grande o suficiente para não
// escrever o arquivo a cada memória.
const reembedJobBatch = 25

// ReembedJob é o estado observável de um re-embed em andamento.
type ReembedJob struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	State     string `json:"state"`
	// Model é o espaço vetorial de destino — sem ele, "concluído" não diz concluído PARA QUÊ.
	Model       string     `json:"model"`
	Total       int        `json:"total"`
	Processed   int        `json:"processed"`
	Reembedded  int        `json:"reembedded"`
	Failed      int        `json:"failed"`
	Remaining   int        `json:"remaining"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
	Cancelled   bool       `json:"cancelled,omitempty"`
	Percentual  float64    `json:"percent,omitempty"`
	LastUpdated time.Time  `json:"last_updated"`
}

type reembedJobEntry struct {
	job    ReembedJob
	cancel context.CancelFunc
}

// EstimateReembed diz quantas memórias o re-embed teria de recalcular AGORA.
//
// Substitui a limitação declarada em E-159 ("sem estimativa prévia do volume"): quem opera sabe o tamanho do
// trabalho antes de começar, em vez de descobrir pelo `remaining`.
func (s *ContextStore) EstimateReembed(projectID string) int {
	return s.CountMemoriesOutsideVectorSpace(projectID)
}

// StartReembedJob cria e dispara o job assíncrono do projeto.
func (s *ContextStore) StartReembedJob(projectID string) (ReembedJob, error) {
	if s == nil {
		return ReembedJob{}, fmt.Errorf("%w: store não configurado", ErrEmbedderInvalid)
	}
	s.mu.RLock()
	embedder := s.embedder
	s.mu.RUnlock()
	if embedder == nil {
		return ReembedJob{}, fmt.Errorf("%w: não há embedder configurado para recalcular vetores", ErrEmbedderInvalid)
	}
	modelo := s.currentEmbeddingModel()

	s.reembedMu.Lock()
	defer s.reembedMu.Unlock()
	if s.reembedJobs == nil {
		s.reembedJobs = map[string]*reembedJobEntry{}
	}
	for _, entrada := range s.reembedJobs {
		if entrada.job.ProjectID == projectID && entrada.job.State == ReembedJobRunning {
			return entrada.job, ErrReembedJobRunning
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	agora := time.Now().UTC()
	job := ReembedJob{
		ID:          "rej_" + fmt.Sprintf("%d", agora.UnixNano()),
		ProjectID:   projectID,
		State:       ReembedJobRunning,
		Model:       modelo,
		Total:       s.EstimateReembed(projectID),
		StartedAt:   agora,
		LastUpdated: agora,
	}
	if job.Total == 0 {
		// Nada a fazer: o job nasce CONCLUÍDO em vez de fingir trabalho por um instante — e o contexto criado
		// para ele é cancelado aqui: nenhuma goroutine foi iniciada, então não há nada a esperar.
		cancel()
		fim := agora
		job.State = ReembedJobDone
		job.FinishedAt = &fim
		s.reembedJobs[job.ID] = &reembedJobEntry{job: job}
		return job, nil
	}
	entrada := &reembedJobEntry{job: job, cancel: cancel}
	s.reembedJobs[job.ID] = entrada
	go s.runReembedJob(ctx, job.ID, projectID, modelo)
	return job, nil
}

// ReembedJobStatus devolve o estado do job (cópia: quem lê não altera o andamento).
func (s *ContextStore) ReembedJobStatus(jobID string) (ReembedJob, error) {
	s.reembedMu.Lock()
	defer s.reembedMu.Unlock()
	entrada, ok := s.reembedJobs[jobID]
	if !ok {
		return ReembedJob{}, fmt.Errorf("%w: %s", ErrReembedJobNotFound, jobID)
	}
	return entrada.job, nil
}

// CancelReembedJob pede o cancelamento. Devolve falso quando o job não existe ou já terminou.
func (s *ContextStore) CancelReembedJob(jobID string) bool {
	s.reembedMu.Lock()
	defer s.reembedMu.Unlock()
	entrada, ok := s.reembedJobs[jobID]
	if !ok || entrada.job.State != ReembedJobRunning || entrada.cancel == nil {
		return false
	}
	entrada.cancel()
	return true
}

// ReembedJobs lista os jobs conhecidos, do mais recente para o mais antigo (visão de operação).
func (s *ContextStore) ReembedJobs() []ReembedJob {
	s.reembedMu.Lock()
	defer s.reembedMu.Unlock()
	jobs := make([]ReembedJob, 0, len(s.reembedJobs))
	for _, entrada := range s.reembedJobs {
		jobs = append(jobs, entrada.job)
	}
	sort.SliceStable(jobs, func(i, j int) bool { return jobs[i].StartedAt.After(jobs[j].StartedAt) })
	return jobs
}

// runReembedJob processa em LOTES, gravando o arquivo a cada lote.
//
// `tentadas` guarda o que já foi TENTADO — inclusive o que FALHOU. Sem isso, uma memória cujo embedder recusou
// continuaria "fora do espaço" e seria reprocessada para sempre: o job nunca terminaria e o laço viraria
// consumo infinito de CPU (o mesmo defeito de "loop sem teto" que a diretriz proíbe).
func (s *ContextStore) runReembedJob(ctx context.Context, jobID, projectID, modelo string) {
	tentadas := map[string]bool{}
	for {
		if ctx.Err() != nil {
			s.finishReembedJob(jobID, ReembedJobCancelled, "")
			return
		}
		lote, restantes := s.reembedCandidates(projectID, modelo, reembedJobBatch, tentadas)
		if len(lote) == 0 {
			s.finishReembedJob(jobID, ReembedJobDone, "")
			return
		}
		vetores := make(map[string][]float32, len(lote))
		falhas := 0
		for _, memoria := range lote {
			tentadas[memoria.ID] = true
			if ctx.Err() != nil {
				break
			}
			vetor, err := s.embedForJob(ctx, memoria.Content)
			if err != nil || len(vetor) == 0 {
				falhas++
				continue
			}
			vetores[memoria.ID] = vetor
		}
		aplicados := s.applyReembedVectors(projectID, modelo, vetores)
		s.advanceReembedJob(jobID, aplicados, falhas, restantes-len(lote))
		if ctx.Err() != nil {
			s.finishReembedJob(jobID, ReembedJobCancelled, "")
			return
		}
	}
}

// reembedCandidates devolve até `limite` memórias fora do espaço atual que ainda NÃO foram tentadas, e quantas
// candidatas restam além do lote.
func (s *ContextStore) reembedCandidates(projectID, modelo string, limite int, tentadas map[string]bool) ([]Memory, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	candidatas := make([]Memory, 0, limite)
	for _, memoria := range s.memories[projectID] {
		if len(memoria.Embedding) > 0 && memoria.EmbeddingModel == modelo {
			continue
		}
		if tentadas[memoria.ID] {
			continue
		}
		candidatas = append(candidatas, memoria)
	}
	total := len(candidatas)
	if total > limite {
		candidatas = candidatas[:limite]
	}
	return candidatas, total
}

// embedForJob chama o embedder fora do lock (gerar vetor é lento e não pode travar o store).
func (s *ContextStore) embedForJob(ctx context.Context, conteudo string) ([]float32, error) {
	s.mu.RLock()
	embedder := s.embedder
	s.mu.RUnlock()
	if embedder == nil {
		return nil, fmt.Errorf("%w: embedder removido durante o job", ErrEmbedderUnavailable)
	}
	return embedder.Embed(ctx, conteudo)
}

// applyReembedVectors grava os vetores calculados e persiste o lote.
func (s *ContextStore) applyReembedVectors(projectID, modelo string, vetores map[string][]float32) int {
	if len(vetores) == 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	memorias := s.memories[projectID]
	aplicados := 0
	for indice := range memorias {
		vetor, ok := vetores[memorias[indice].ID]
		if !ok {
			continue
		}
		memorias[indice].Embedding = vetor
		memorias[indice].EmbeddingModel = modelo
		aplicados++
	}
	s.memories[projectID] = memorias
	if s.root != "" && aplicados > 0 {
		// Gravar por lote: um processo morto no meio não perde o que já foi feito.
		_ = s.writeMemoriesFile(projectID, memorias)
	}
	return aplicados
}

func (s *ContextStore) advanceReembedJob(jobID string, aplicados, falhas, restantes int) {
	s.reembedMu.Lock()
	defer s.reembedMu.Unlock()
	entrada, ok := s.reembedJobs[jobID]
	if !ok {
		return
	}
	entrada.job.Reembedded += aplicados
	entrada.job.Failed += falhas
	entrada.job.Processed = entrada.job.Reembedded + entrada.job.Failed
	if restantes < 0 {
		restantes = 0
	}
	entrada.job.Remaining = restantes
	if entrada.job.Total > 0 {
		entrada.job.Percentual = float64(entrada.job.Processed) / float64(entrada.job.Total) * 100
	}
	entrada.job.LastUpdated = time.Now().UTC()
}

func (s *ContextStore) finishReembedJob(jobID, estado, motivo string) {
	s.reembedMu.Lock()
	defer s.reembedMu.Unlock()
	entrada, ok := s.reembedJobs[jobID]
	if !ok {
		return
	}
	fim := time.Now().UTC()
	entrada.job.State = estado
	entrada.job.FinishedAt = &fim
	entrada.job.Cancelled = estado == ReembedJobCancelled
	entrada.job.LastError = motivo
	// O que sobrou é recalculado pelo store: um job cancelado precisa dizer quanto NÃO foi feito.
	entrada.job.Remaining = s.CountMemoriesOutsideVectorSpace(entrada.job.ProjectID)
	if entrada.job.Total > 0 {
		entrada.job.Percentual = float64(entrada.job.Processed) / float64(entrada.job.Total) * 100
		if entrada.job.Percentual > 100 {
			entrada.job.Percentual = 100
		}
	}
	entrada.job.LastUpdated = fim
	entrada.cancel = nil
}
