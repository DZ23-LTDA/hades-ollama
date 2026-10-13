package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ErrMemoryChangedOutsideStore recusa sobrescrever uma alteração feita fora do store.
//
// A trava do arquivo de memórias é ADVISORY: quem escreve sem passar pelo store não é impedido. O que se pode fazer
// com honestidade é DETECTAR antes de apagar — e é isso que a guarda faz nos caminhos NÃO-DESTRUTIVOS.
var ErrMemoryChangedOutsideStore = errors.New("memory file changed outside the store")

// writeMemoriesFileGuardingAdditions grava um snapshot de caminho NÃO-DESTRUTIVO recusando apagar o que outro
// escritor acrescentou.
//
// POR QUE SÓ NOS NÃO-DESTRUTIVOS (E-288 → E-289): deleção legítima e sobrescrita por snapshot velho produzem o
// MESMO estado no arquivo. Sem intenção declarada, qualquer guarda recusa operação legítima — foi exatamente o que
// aconteceu quando a guarda cobria todos os caminhos e quebrou RemoveMemory, PruneMemories e trimMemoriesToLimit.
func (s *ContextStore) writeMemoriesFileGuardingAdditions(projectID string, snapshot []Memory) error {
	if s == nil || s.root == "" {
		return s.writeMemoriesFile(projectID, snapshot)
	}
	noSnapshot := make(map[string]bool, len(snapshot))
	for _, memoria := range snapshot {
		noSnapshot[memoria.ID] = true
	}
	if atuais, err := s.readMemoriesForGuard(projectID); err != nil {
		return err
	} else {
		acrescentadas := []string{}
		for _, memoria := range atuais {
			if !noSnapshot[memoria.ID] {
				acrescentadas = append(acrescentadas, memoria.ID)
			}
		}
		if len(acrescentadas) > 0 {
			sort.Strings(acrescentadas)
			return fmt.Errorf("%w: %s (memória(s) acrescentada(s) por outro escritor: %s)",
				ErrMemoryChangedOutsideStore, projectID, strings.Join(acrescentadas, ", "))
		}
	}
	return s.writeMemoriesFile(projectID, snapshot)
}

// readMemoriesForGuard lê o disco aplicando a INVERSA da transformação de disco: o arquivo pode estar cifrado, e
// comparar cifrado com claro acusaria divergência FALSA.
func (s *ContextStore) readMemoriesForGuard(projectID string) ([]Memory, error) {
	if s == nil || s.root == "" {
		return nil, nil
	}
	dados, err := os.ReadFile(s.memoryFilePath(projectID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(dados) == 0 {
		return nil, nil
	}
	var lidas []Memory
	if err := json.Unmarshal(dados, &lidas); err != nil {
		return nil, err
	}
	return memoriesFromDisk(lidas)
}
