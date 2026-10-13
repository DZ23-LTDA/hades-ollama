package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Persistência da fila de mortos do Omnichannel Runtime (estágio 34).
//
// O limite declarado era: *"DLQ em memória (persistência é trabalho futuro)"*. Fila de mortos em
// memória tem um defeito específico e silencioso: **o evento que falhou desaparece no reinício** — e o
// que se perde não é um número, é a mensagem de um cliente que ninguém vai reprocessar porque ninguém
// sabe que ela existiu.
//
// Decisões:
//
//  1. **UM ARQUIVO POR ORGANIZAÇÃO** (`<dataRoot>/.agent-omnichannel-dlq/<org>.json`): o isolamento
//     entre tenants vale no disco, não só no filtro de leitura;
//  2. **TETO POR ORGANIZAÇÃO**: o teto global anterior permitia que o volume de um tenant **expulsasse**
//     as entradas de outro — um tenant apagando evidência alheia por volume;
//  3. **ESCRITA ATÔMICA** (`writeJSONAtomic`), que já cria o diretório pai (correção de raiz de E-127);
//  4. **FALHA DE PERSISTÊNCIA É VISÍVEL**: se o disco recusar, a entrada **não** é descartada da memória
//     e o estado passa a reportar o erro — perder em silêncio seria repetir o defeito que esta rodada
//     corrige.

// OmnichannelDLQStore guarda a fila de mortos por organização.
type OmnichannelDLQStore struct {
	root string
	// MaxPerOrganization limita a fila de CADA organização.
	MaxPerOrganization int
}

type omnichannelDLQFile struct {
	DeadLetters []OmnichannelDeadLetter `json:"dead_letters"`
}

// NewOmnichannelDLQStore cria o store com o teto por organização.
func NewOmnichannelDLQStore(root string) *OmnichannelDLQStore {
	return &OmnichannelDLQStore{
		root:               strings.TrimSpace(root),
		MaxPerOrganization: OmnichannelMaxDLQ,
	}
}

// Root devolve o diretório usado (vazio quando não há persistência configurada).
func (s *OmnichannelDLQStore) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *OmnichannelDLQStore) path(organizationID string) string {
	if s == nil || s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

// Load devolve as entradas persistidas da organização (mais antigas primeiro).
func (s *OmnichannelDLQStore) Load(organizationID string) ([]OmnichannelDeadLetter, error) {
	if err := requireOrganization(organizationID); err != nil {
		return nil, err
	}
	path := s.path(organizationID)
	if path == "" {
		return nil, nil
	}
	var file omnichannelDLQFile
	err := readJSON(path, &file)
	if err != nil {
		// ARQUIVO AUSENTE é estado normal: nada foi para a fila ainda.
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		// QUALQUER OUTRO erro (arquivo corrompido, permissão, disco) é REPORTADO. Tratar isso como
		// "fila vazia" faria uma fila de mortos corrompida parecer limpa — apagando a evidência de que
		// existiam mensagens não processadas.
		return nil, err
	}
	return file.DeadLetters, nil
}

// Save grava a fila da organização, respeitando o teto dela.
func (s *OmnichannelDLQStore) Save(organizationID string, entries []OmnichannelDeadLetter) error {
	if err := requireOrganization(organizationID); err != nil {
		return err
	}
	path := s.path(organizationID)
	if path == "" {
		return nil
	}
	limit := s.MaxPerOrganization
	if limit <= 0 {
		limit = OmnichannelMaxDLQ
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return writeJSONAtomic(path, omnichannelDLQFile{DeadLetters: entries})
}

// omnichannelDLQRoot devolve o diretório da fila de mortos dentro da raiz de dados.
func omnichannelDLQRoot(dataRoot string) string {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return ""
	}
	return filepath.Join(dataRoot, ".agent-omnichannel-dlq")
}
