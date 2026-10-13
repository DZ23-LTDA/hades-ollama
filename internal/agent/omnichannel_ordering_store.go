package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ORDENACAO DURAVEL E COM TETO (E-299).
//
// Fecha o limite declarado em E-298: o estado de ordem vivia só em memória, então um reinício esquecia os últimos
// instantes e o PRIMEIRO evento de cada conversa voltava a parecer em ordem — exatamente o caso que a marcação
// existe para pegar. É a mesma classe que a dedupe tinha antes de E-172, e por isso segue o MESMO padrão:
//
//   - UM ARQUIVO POR ORGANIZAÇÃO (`<dataRoot>/.agent-omnichannel-ordering/<org>.json`);
//   - ESCRITA ATÔMICA por `writeJSONAtomic`;
//   - TETO por organização com poda dos mais antigos (sem teto, vira vazamento lento de disco);
//   - SEM STORE, O ESTADO DIZ QUE NÃO HÁ PERSISTÊNCIA;
//   - FALHA DE LEITURA/GRAVAÇÃO É REPORTADA, nunca engolida.
//
// OmnichannelMaxOrderHeadsPerOrg é o TETO de conversas com linha do tempo guardada por organização.
const OmnichannelMaxOrderHeadsPerOrg = 5000

// OmnichannelOrderingStore guarda o último instante aceito por chave de conversa, por organização.
type OmnichannelOrderingStore struct {
	root string
}

type omnichannelOrderingFile struct {
	Heads map[string]int64 `json:"heads"`
}

// NewOmnichannelOrderingStore cria o store no diretório informado.
func NewOmnichannelOrderingStore(root string) *OmnichannelOrderingStore {
	return &OmnichannelOrderingStore{root: strings.TrimSpace(root)}
}

// Root devolve o diretório do store.
func (s *OmnichannelOrderingStore) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *OmnichannelOrderingStore) path(organizationID string) string {
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

// Load lê as linhas do tempo da organização. Arquivo ausente não é erro: é organização nova.
func (s *OmnichannelOrderingStore) Load(organizationID string) (map[string]int64, error) {
	if err := requireOrganization(organizationID); err != nil {
		return nil, err
	}
	if s == nil || s.root == "" {
		return nil, nil
	}
	caminho := s.path(organizationID)
	dados, err := os.ReadFile(caminho)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(dados) == 0 {
		return nil, nil
	}
	var arquivo omnichannelOrderingFile
	if err := json.Unmarshal(dados, &arquivo); err != nil {
		return nil, err
	}
	if arquivo.Heads == nil {
		return nil, nil
	}
	return arquivo.Heads, nil
}

// Save grava as linhas do tempo aplicando o TETO: guarda as mais RECENTES, descarta as mais antigas. Sem teto, um
// processo longo acumularia uma chave por conversa para sempre.
func (s *OmnichannelOrderingStore) Save(organizationID string, heads map[string]int64) error {
	if err := requireOrganization(organizationID); err != nil {
		return err
	}
	if s == nil || s.root == "" {
		return nil
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	return writeJSONAtomic(s.path(organizationID), omnichannelOrderingFile{Heads: podarOrdering(heads, OmnichannelMaxOrderHeadsPerOrg)})
}

// podarOrdering mantém as `teto` chaves com instante mais RECENTE. Empate de instante é desempatado pela chave, para
// a poda ser DETERMINÍSTICA (duas execuções com a mesma entrada não podem escolher vítimas diferentes).
func podarOrdering(heads map[string]int64, teto int) map[string]int64 {
	if teto <= 0 || len(heads) <= teto {
		return heads
	}
	chaves := make([]string, 0, len(heads))
	for chave := range heads {
		chaves = append(chaves, chave)
	}
	sort.Slice(chaves, func(i, j int) bool {
		if heads[chaves[i]] != heads[chaves[j]] {
			return heads[chaves[i]] > heads[chaves[j]]
		}
		return chaves[i] < chaves[j]
	})
	podado := make(map[string]int64, teto)
	for _, chave := range chaves[:teto] {
		podado[chave] = heads[chave]
	}
	return podado
}

// omnichannelOrderingRoot resolve o diretório padrão a partir da raiz de dados.
func omnichannelOrderingRoot(dataRoot string) string {
	if strings.TrimSpace(dataRoot) == "" {
		return ""
	}
	return filepath.Join(dataRoot, ".agent-omnichannel-ordering")
}
