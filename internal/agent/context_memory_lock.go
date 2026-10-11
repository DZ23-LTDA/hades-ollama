package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TRAVA ENTRE PROCESSOS no arquivo de memórias (E-162 declarou o limite: "chamadas concorrentes no mesmo projeto
// resolvem por último-escritor").
//
// O re-embed em lote reescreve o arquivo do projeto. Se dois processos (ou duas instâncias de store no mesmo
// diretório) carregaram o arquivo antes, o segundo a gravar impõe a sua cópia — e o trabalho do primeiro SOME
// EM SILÊNCIO. Serializar a escrita resolve a corrida; **reler antes de gravar** resolve a perda.
//
// A trava é a MESMA infraestrutura já usada por filas e push (`withFileLock`/`withFileLockContext`, com
// `flock` no Unix e `LockFileEx` no Windows): não há implementação nova de lock aqui, só uso no caminho que
// estava desprotegido.
var (
	// ErrStoreLocked diz que outro processo está gravando este projeto e o prazo de espera acabou.
	//
	// É erro DECLARADO, não espera infinita: um dono que travou não pode pendurar quem só quer gravar.
	ErrStoreLocked = errors.New("arquivo do projeto está travado por outro processo")
)

// MemoryFileLockTimeout é quanto se espera pela trava antes de recusar.
const MemoryFileLockTimeout = 10 * time.Second

// memoryLockPath é o lock do arquivo de memórias do projeto (o arquivo em si nunca é removido).
func (s *ContextStore) memoryLockPath(projectID string) string {
	return filepath.Join(s.root, "memories", projectID+".json.lock")
}

// memoryFilePath é o arquivo de memórias do projeto.
func (s *ContextStore) memoryFilePath(projectID string) string {
	return filepath.Join(s.root, "memories", projectID+".json")
}

// updateMemoriesFileLocked aplica uma alteração ao conteúdo ATUAL do arquivo: lê de novo, muda e grava, tudo
// dentro da trava.
//
// É o que corrige a perda de trabalho entre processos: sem o "ler de novo", a cópia em memória de quem grava
// por último apaga o que o outro processo escreveu no meio do caminho.
func (s *ContextStore) updateMemoriesFileLocked(ctx context.Context, projectID string, alterar func([]Memory) []Memory) error {
	if s == nil || s.root == "" || alterar == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	travaCtx, cancelar := context.WithTimeout(ctx, MemoryFileLockTimeout)
	defer cancelar()
	err := withFileLockContext(travaCtx, s.memoryLockPath(projectID), func() error {
		// O arquivo pode não existir ainda (projeto recém-criado): nesse caso a alteração parte do vazio.
		atual := []Memory{}
		if dados, err := os.ReadFile(s.memoryFilePath(projectID)); err == nil && len(dados) > 0 {
			if err := json.Unmarshal(dados, &atual); err != nil {
				return fmt.Errorf("ler memórias do projeto para atualizar: %w", err)
			}
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("ler memórias do projeto para atualizar: %w", err)
		}
		return s.writeMemoriesFile(projectID, alterar(atual))
	})
	if err != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
		return fmt.Errorf("%w: %s", ErrStoreLocked, projectID)
	}
	return err
}
