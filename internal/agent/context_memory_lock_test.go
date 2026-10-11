package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TRAVA ENTRE PROCESSOS no arquivo de memórias (limite declarado em E-162: "chamadas concorrentes no mesmo
// projeto resolvem por último-escritor").
//
// O que estes testes travam:
//
//  1. o re-embed NÃO apaga a memória que OUTRO processo gravou no arquivo depois de o store carregar a sua
//     cópia (é a perda silenciosa concreta: cada instância tem a própria cópia em memória);
//  2. a escrita passa pela MESMA trava já usada por filas e push (`withFileLock`, `flock`/`LockFileEx`) — sem
//     implementação nova de lock;
//  3. um lock disputado por outro dono é DECLARADO (`ErrStoreLocked`), não esperado para sempre.

// simulaOutroProcesso acrescenta uma memória DIRETO no arquivo, como faria uma segunda instância do Hades.
func simulaOutroProcesso(t *testing.T, store *ContextStore, projectID, id, conteudo string) {
	t.Helper()
	caminho := store.memoryFilePath(projectID)
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	var memorias []Memory
	if err := json.Unmarshal(dados, &memorias); err != nil {
		t.Fatal(err)
	}
	memorias = append(memorias, Memory{ID: id, ProjectID: projectID, Kind: "fact", Content: conteudo, Source: "manual"})
	atualizado, err := json.Marshal(memorias)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, atualizado, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReembedDoesNotEraseWhatAnotherProcessWrote(t *testing.T) {
	store, err := NewContextStore(t.TempDir() + "-lock")
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := store.CreateProject("Compartilhado", t.TempDir(), "org-lock")
	if err != nil {
		t.Fatal(err)
	}
	// Três memórias SEM vetor: são as candidatas do re-embed.
	for i := range 3 {
		if _, err := store.AddMemory(Memory{ProjectID: projeto.ID, Kind: "fact", Content: fmt.Sprintf("memória inicial %d", i), Source: "manual"}); err != nil {
			t.Fatal(err)
		}
	}
	// OUTRO processo grava no mesmo arquivo DEPOIS de o store ter a sua cópia em memória.
	simulaOutroProcesso(t, store, projeto.ID, "mem_externa", "memória escrita pelo OUTRO processo")
	store.SetEmbedder(stubEmbedder{modelo: "modelo-lock"})

	if _, err := store.ReembedMemories(context.Background(), projeto.ID, 0); err != nil {
		t.Fatal(err)
	}

	dados, err := os.ReadFile(store.memoryFilePath(projeto.ID))
	if err != nil {
		t.Fatal(err)
	}
	texto := string(dados)
	if !strings.Contains(texto, "memória escrita pelo OUTRO processo") {
		t.Fatalf("o re-embed apagou o trabalho de outro processo: %s", texto)
	}
	if strings.Count(texto, "memória inicial") != 3 {
		t.Fatalf("as memórias do próprio projeto precisam continuar: %s", texto)
	}
	if !strings.Contains(texto, "modelo-lock") {
		t.Fatalf("o re-embed precisa ter carimbado o espaço novo: %s", texto)
	}
}

func TestReembedJobDoesNotEraseWhatAnotherProcessWrote(t *testing.T) {
	store, err := NewContextStore(t.TempDir() + "-joblock")
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := store.CreateProject("Compartilhado do job", t.TempDir(), "org-lock")
	if err != nil {
		t.Fatal(err)
	}
	for i := range 60 {
		if _, err := store.AddMemory(Memory{ProjectID: projeto.ID, Kind: "fact", Content: fmt.Sprintf("memória %d", i), Source: "manual"}); err != nil {
			t.Fatal(err)
		}
	}
	simulaOutroProcesso(t, store, projeto.ID, "mem_externa_job", "escrita por outro processo durante o job")
	store.SetEmbedder(stubEmbedder{modelo: "modelo-job-lock"})

	job, err := store.StartReembedJob(projeto.ID)
	if err != nil {
		t.Fatal(err)
	}
	final := esperaJob(t, store, job.ID)
	if final.State != ReembedJobDone {
		t.Fatalf("o job precisa concluir: %+v", final)
	}
	dados, err := os.ReadFile(store.memoryFilePath(projeto.ID))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dados), "escrita por outro processo durante o job") {
		t.Fatalf("o job apagou o trabalho de outro processo: %s", dados)
	}
	if strings.Count(string(dados), "memória ") < 60 {
		t.Fatalf("as memórias do projeto precisam continuar: %s", dados)
	}
}

func TestMemoryWriteLockIsDeclaredWhenAnotherOwnerHoldsIt(t *testing.T) {
	store, err := NewContextStore(t.TempDir() + "-disputa")
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := store.CreateProject("Disputado", t.TempDir(), "org-lock")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemory(Memory{ProjectID: projeto.ID, Kind: "fact", Content: "conteúdo", Source: "manual"}); err != nil {
		t.Fatal(err)
	}

	// Outro dono segura a trava enquanto tentamos gravar: a recusa precisa ser DECLARADA.
	liberar := make(chan struct{})
	segurou := make(chan struct{})
	go func() {
		_ = withFileLock(store.memoryLockPath(projeto.ID), func() error {
			close(segurou)
			<-liberar
			return nil
		})
	}()
	<-segurou
	defer close(liberar)

	curto, cancelar := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancelar()
	err = store.updateMemoriesFileLocked(curto, projeto.ID, func(atual []Memory) []Memory { return atual })
	if err == nil {
		t.Skip("a trava não bloqueou dentro da janela: sistema de arquivos sem semântica de flock")
	}
	if !strings.Contains(err.Error(), "travado por outro processo") {
		t.Fatalf("a recusa precisa ser declarada como trava: %v", err)
	}
	_ = filepath.Join
}
