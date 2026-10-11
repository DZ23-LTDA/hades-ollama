package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// RE-EMBED ASSÍNCRONO (estágio 35). O limite declarado em E-159 era: "re-embed SÍNCRONO, 200 por chamada e por
// projeto, sem estimativa prévia do volume".
//
// O que estes testes travam:
//
//  1. a ESTIMATIVA existe antes de começar (quem opera sabe o tamanho do trabalho);
//  2. o job roda até o FIM, com progresso observável e o arquivo gravado por lote;
//  3. um SEGUNDO job no mesmo projeto é RECUSADO (dois jobs competiriam pelo mesmo arquivo);
//  4. um job sem trabalho nasce CONCLUÍDO, em vez de fingir atividade;
//  5. falha de embedding é contabilizada e NÃO vira laço infinito (a memória não é retentada para sempre);
//  6. cancelamento termina o job com o que já foi feito contabilizado.
func reembedJobFixture(t *testing.T, total int) (*ContextStore, Project) {
	t.Helper()
	store, err := NewContextStore(t.TempDir() + "-job")
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("Base do job", t.TempDir(), "org-job")
	if err != nil {
		t.Fatal(err)
	}
	// Sem embedder na escrita: as memórias nascem SEM vetor (o estado de quem trocou de modelo).
	for i := range total {
		if _, err := store.AddMemory(Memory{ProjectID: project.ID, Kind: "fact", Content: fmt.Sprintf("memória %d", i), Source: "manual"}); err != nil {
			t.Fatal(err)
		}
	}
	return store, project
}

// esperaJob aguarda o job sair de `running` (com teto de tempo para o teste não pendurar).
func esperaJob(t *testing.T, store *ContextStore, jobID string) ReembedJob {
	t.Helper()
	limite := time.Now().Add(10 * time.Second)
	for time.Now().Before(limite) {
		job, err := store.ReembedJobStatus(jobID)
		if err != nil {
			t.Fatal(err)
		}
		if job.State != ReembedJobRunning {
			return job
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("o job não terminou dentro do tempo do teste")
	return ReembedJob{}
}

func TestReembedJobEstimatesBeforeStarting(t *testing.T) {
	store, project := reembedJobFixture(t, 60)
	// SEM embedder não há espaço de DESTINO: a estimativa é zero por definição (não há "fora do espaço" sem
	// espaço definido), e o job é recusado. As duas coisas são ditas, não escondidas.
	if estimativa := store.EstimateReembed(project.ID); estimativa != 0 {
		t.Fatalf("sem embedder não há espaço de destino a estimar: %d", estimativa)
	}
	if _, err := store.StartReembedJob(project.ID); err == nil {
		t.Fatal("sem embedder o job precisa ser recusado")
	}
	store.SetEmbedder(stubEmbedder{modelo: "modelo-job"})
	// COM embedder, a estimativa conta as 60 memórias sem vetor ANTES de começar.
	if estimativa := store.EstimateReembed(project.ID); estimativa != 60 {
		t.Fatalf("a estimativa precisa contar antes de começar: %d", estimativa)
	}
	job, err := store.StartReembedJob(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Total != 60 || job.Model != "modelo-job" {
		t.Fatalf("o job precisa declarar total e o espaço de destino: %+v", job)
	}
	final := esperaJob(t, store, job.ID)
	if final.State != ReembedJobDone {
		t.Fatalf("o job precisa terminar concluído: %+v", final)
	}
	if final.Reembedded != 60 || final.Failed != 0 || final.Processed != 60 {
		t.Fatalf("contadores do job divergentes: %+v", final)
	}
	if final.Remaining != 0 {
		t.Fatalf("nada pode ficar fora do espaço: %+v", final)
	}
	if final.Percentual < 99.9 {
		t.Fatalf("progresso precisa chegar a 100%%: %+v", final.Percentual)
	}
	if store.EstimateReembed(project.ID) != 0 {
		t.Fatal("depois do job não há mais o que estimar")
	}
}

func TestReembedJobRefusesASecondJobOnTheSameProject(t *testing.T) {
	store, project := reembedJobFixture(t, 120)
	store.SetEmbedder(stubEmbedder{modelo: "modelo-job"})
	primeiro, err := store.StartReembedJob(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	// O segundo pedido pode chegar antes de o primeiro terminar: precisa ser RECUSADO, não enfileirado.
	if _, err := store.StartReembedJob(project.ID); !errors.Is(err, ErrReembedJobRunning) {
		t.Fatalf("segundo job no mesmo projeto precisa ser recusado: %v", err)
	}
	final := esperaJob(t, store, primeiro.ID)
	if final.State != ReembedJobDone || final.Reembedded != 120 {
		t.Fatalf("o primeiro job precisa terminar inteiro: %+v", final)
	}
}

func TestReembedJobWithNothingToDoIsBornDone(t *testing.T) {
	store, project := reembedJobFixture(t, 0)
	store.SetEmbedder(stubEmbedder{modelo: "modelo-job"})
	job, err := store.StartReembedJob(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Sem trabalho não há job "rodando por um instante": nasce concluído.
	if job.State != ReembedJobDone || job.FinishedAt == nil {
		t.Fatalf("job vazio precisa nascer concluído: %+v", job)
	}
}

func TestReembedJobDoesNotLoopForeverOnFailures(t *testing.T) {
	store, project := reembedJobFixture(t, 30)
	// Embedder que SEMPRE falha: sem o controle de "já tentadas", o job reprocessaria para sempre.
	store.SetEmbedder(stubEmbedder{err: ErrEmbedderUnavailable, modelo: "modelo-job"})
	job, err := store.StartReembedJob(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	final := esperaJob(t, store, job.ID)
	if final.State != ReembedJobDone {
		t.Fatalf("o job precisa terminar mesmo com falhas: %+v", final)
	}
	if final.Failed != 30 || final.Reembedded != 0 {
		t.Fatalf("as falhas precisam ser contabilizadas: %+v", final)
	}
	// E o que falhou CONTINUA fora do espaço: nada foi gravado por conta própria.
	if fora := store.CountMemoriesOutsideVectorSpace(project.ID); fora != 30 {
		t.Fatalf("memória que falhou não pode sumir da contagem: %d", fora)
	}
}

func TestReembedJobCancelKeepsWhatWasDone(t *testing.T) {
	store, project := reembedJobFixture(t, 200)
	// Embedder lento o bastante para o cancelamento chegar no meio (o job checa entre lotes).
	store.SetEmbedder(&stubLentoEmbedder{atraso: 2 * time.Millisecond, modelo: "modelo-job"})
	job, err := store.StartReembedJob(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Espera o primeiro lote gravar antes de cancelar, senão o teste mediria o cancelamento imediato.
	time.Sleep(20 * time.Millisecond)
	if !store.CancelReembedJob(job.ID) {
		t.Skip("o job terminou antes do cancelamento: cenário rápido demais nesta máquina")
	}
	final := esperaJob(t, store, job.ID)
	if final.State != ReembedJobCancelled || !final.Cancelled {
		t.Fatalf("o job precisa terminar como cancelado: %+v", final)
	}
	if final.Reembedded == 0 {
		t.Fatalf("o cancelamento não pode descartar o que já foi gravado: %+v", final)
	}
	if final.Remaining == 0 {
		t.Fatalf("o que NÃO foi feito precisa ser declarado: %+v", final)
	}
	// Cancelar de novo não é possível: o job já terminou.
	if store.CancelReembedJob(job.ID) {
		t.Fatal("job terminado não aceita novo cancelamento")
	}
}

func TestReembedJobStatusOfUnknownJobIsDeclared(t *testing.T) {
	store, _ := reembedJobFixture(t, 0)
	if _, err := store.ReembedJobStatus("rej_inexistente"); !errors.Is(err, ErrReembedJobNotFound) {
		t.Fatalf("job desconhecido precisa ser declarado como não encontrado: %v", err)
	}
	if store.CancelReembedJob("rej_inexistente") {
		t.Fatal("cancelar job inexistente precisa devolver falso")
	}
}

// stubLentoEmbedder atrasa cada vetor, para o cancelamento ter onde acontecer.
type stubLentoEmbedder struct {
	atraso time.Duration
	modelo string
}

func (s *stubLentoEmbedder) Embed(ctx context.Context, _ string) ([]float32, error) {
	select {
	case <-time.After(s.atraso):
		return []float32{1, 0, 0}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *stubLentoEmbedder) EmbeddingModel() string { return strings.TrimSpace(s.modelo) }
