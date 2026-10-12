package agent

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DEDUPE DURÁVEL E COM TETO do Omnichannel (E-172).
//
// Dois defeitos no mesmo lugar:
//
//  1. `eventsByKey` **vivia só em memória**: um reinício do processo apagava a memória de deduplicação, e uma
//     entrega repetida do mesmo `external_message_id` voltava a ser processada — exatamente o que a dedupe
//     existe para impedir;
//  2. `eventsByKey` **não tinha teto nem poda**: em processo de longa duração, o mapa crescia sem limite. Um
//     vazamento lento é mais difícil de perceber que uma falha, e por isso é pior.
//
// O índice persistido resolve os dois: entradas com a **chave de dedupe**, poda por **janela de retenção** e
// **teto por organização** (mesmo padrão da DLQ). Seguindo o que já existe no pacote:
//
//   - **UM ARQUIVO POR ORGANIZAÇÃO** (`<dataRoot>/.agent-omnichannel-dedupe/<org>.json`);
//   - **ESCRITA ATÔMICA** por `writeJSONAtomic` (herda a política de durabilidade de E-168/E-169);
//   - **SEM STORE, O ESTADO DIZ QUE NÃO HÁ PERSISTÊNCIA** (`DedupePersisted()==false`);
//   - **FALHA DE LEITURA/GRAVAÇÃO É REPORTADA**, nunca engolida.
const (
	// OmnichannelDedupeRetention é por quanto tempo uma chave de dedupe protege. A janela de replay de webhook é
	// muito menor; guardar por 7 dias dá margem para entregas atrasadas sem crescer para sempre.
	OmnichannelDedupeRetention = 7 * 24 * time.Hour
	// OmnichannelMaxDedupePerOrg é o TETO por organização. Sem teto, a dedupe vira vazamento de memória/disco.
	OmnichannelMaxDedupePerOrg = 5000
)

// OmnichannelDedupeEntry é o registro mínimo de uma mensagem já vista.
//
// É de propósito MENOR que o evento: a dedupe precisa saber "isto já entrou", e não recarregar o payload inteiro.
type OmnichannelDedupeEntry struct {
	ExternalMessageID string    `json:"external_message_id"`
	Channel           string    `json:"channel"`
	ContactID         string    `json:"contact_id,omitempty"`
	MessageType       string    `json:"message_type,omitempty"`
	IngestedAt        time.Time `json:"ingested_at"`
}

// OmnichannelDedupeStore guarda o índice de dedupe por organização.
type OmnichannelDedupeStore struct {
	root string
}

type omnichannelDedupeFile struct {
	Entries map[string]OmnichannelDedupeEntry `json:"entries"`
}

// NewOmnichannelDedupeStore cria o store no diretório informado.
func NewOmnichannelDedupeStore(root string) *OmnichannelDedupeStore {
	return &OmnichannelDedupeStore{root: strings.TrimSpace(root)}
}

// Root devolve o diretório usado (vazio quando não há persistência configurada).
func (s *OmnichannelDedupeStore) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *OmnichannelDedupeStore) path(organizationID string) string {
	if s == nil || s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

// Load devolve o índice persistido da organização. Arquivo ausente é estado normal.
func (s *OmnichannelDedupeStore) Load(organizationID string) (map[string]OmnichannelDedupeEntry, error) {
	caminho := s.path(organizationID)
	if caminho == "" {
		return nil, nil
	}
	var arquivo omnichannelDedupeFile
	err := readJSON(caminho, &arquivo)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if arquivo.Entries == nil {
		return nil, nil
	}
	return arquivo.Entries, nil
}

// Save grava o índice da organização já PODADO por quem chama.
func (s *OmnichannelDedupeStore) Save(organizationID string, entries map[string]OmnichannelDedupeEntry) error {
	caminho := s.path(organizationID)
	if caminho == "" {
		return nil
	}
	return writeJSONAtomic(caminho, omnichannelDedupeFile{Entries: entries})
}

// podarDedupe aplica RETENÇÃO e TETO e devolve o índice podado.
//
// A ordem importa: primeiro descarta o que venceu, depois — se ainda passar do teto — remove os mais ANTIGOS.
// Sem esta função, o índice (em memória e em disco) cresceria sem limite.
func podarDedupe(entries map[string]OmnichannelDedupeEntry, agora time.Time, teto int) map[string]OmnichannelDedupeEntry {
	if entries == nil {
		return nil
	}
	podado := make(map[string]OmnichannelDedupeEntry, len(entries))
	for chave, entrada := range entries {
		if entrada.IngestedAt.IsZero() || agora.Sub(entrada.IngestedAt) > OmnichannelDedupeRetention {
			continue
		}
		podado[chave] = entrada
	}
	if teto <= 0 || len(podado) <= teto {
		return podado
	}
	chaves := make([]string, 0, len(podado))
	for chave := range podado {
		chaves = append(chaves, chave)
	}
	sort.Slice(chaves, func(i, j int) bool {
		return podado[chaves[i]].IngestedAt.Before(podado[chaves[j]].IngestedAt)
	})
	for _, chave := range chaves[:len(chaves)-teto] {
		delete(podado, chave)
	}
	return podado
}

// omnichannelDedupeRoot devolve o diretório do índice de dedupe dentro da raiz de dados.
func omnichannelDedupeRoot(dataRoot string) string {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return ""
	}
	return filepath.Join(dataRoot, ".agent-omnichannel-dedupe")
}
