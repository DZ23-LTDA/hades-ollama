package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// PERSISTÊNCIA DOS CONECTORES do Omnichannel (E-171).
//
// O limite foi DESCOBERTO por um teste na rodada anterior: o teste de "reinício" da janela de rate limit falhou
// com *"connector is not configured"*. Os conectores vivem em `r.connectors` (memória), e a consequência é pior
// que perder um número:
//
//   - **`Ingest` procura o conector por organização e canal**. Depois de um reinício, ele não existe mais, então
//     **todo webhook legítimo do canal é recusado** com "not configured" — o canal simplesmente para;
//   - e um `RegisterConnector` de novo gera **ID NOVO**, então qualquer assinatura referente ao ID antigo deixa de
//     valer: quem já estava integrado quebra sem entender por quê.
//
// O que é persistido: o REGISTRO do conector (ID, organização, canal, nome, **NOME da variável de ambiente do
// segredo** e validade). O **valor do segredo nunca é gravado** — ele continua só no ambiente, e há teste que
// confere isso no arquivo em disco.
//
// Decisões, seguindo o padrão da DLQ e da janela de rate limit:
//
//  1. **UM ARQUIVO POR ORGANIZAÇÃO** (`<dataRoot>/.agent-omnichannel-connectors/<org>.json`);
//  2. **ESCRITA ATÔMICA** por `writeJSONAtomic` (herda a política de durabilidade de E-168/E-169);
//  3. **SEM STORE, O ESTADO DIZ QUE NÃO HÁ PERSISTÊNCIA** (`ConnectorPersisted()==false`);
//  4. **FALHA DE LEITURA/GRAVAÇÃO É REPORTADA**, nunca engolida.

// OmnichannelConnectorStore guarda os conectores por organização.
type OmnichannelConnectorStore struct {
	root string
}

type omnichannelConnectorFile struct {
	Connectors []OmnichannelConnector `json:"connectors"`
}

// NewOmnichannelConnectorStore cria o store no diretório informado.
func NewOmnichannelConnectorStore(root string) *OmnichannelConnectorStore {
	return &OmnichannelConnectorStore{root: strings.TrimSpace(root)}
}

// Root devolve o diretório usado (vazio quando não há persistência configurada).
func (s *OmnichannelConnectorStore) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *OmnichannelConnectorStore) path(organizationID string) string {
	if s == nil || s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

// Load devolve os conectores persistidos da organização.
//
// Arquivo ausente é estado normal (nenhum conector cadastrado). Qualquer outro erro é DEVOLVIDO: um conector que
// não pôde ser lido é um canal que vai recusar mensagens legítimas.
func (s *OmnichannelConnectorStore) Load(organizationID string) ([]OmnichannelConnector, error) {
	caminho := s.path(organizationID)
	if caminho == "" {
		return nil, nil
	}
	var arquivo omnichannelConnectorFile
	err := readJSON(caminho, &arquivo)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return arquivo.Connectors, nil
}

// LoadAll devolve os conectores de TODAS as organizações.
//
// Existe porque nem toda busca é por organização e canal: `SignWebhook` procura **por ID** e não sabe a
// organização antes de carregar. Uma leitura completa (uma vez por processo) resolve isso sem inventar um índice.
func (s *OmnichannelConnectorStore) LoadAll() ([]OmnichannelConnector, error) {
	if s == nil || s.root == "" {
		return nil, nil
	}
	entradas, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var conectores []OmnichannelConnector
	for _, entrada := range entradas {
		if entrada.IsDir() || !strings.HasSuffix(entrada.Name(), ".json") {
			continue
		}
		var arquivo omnichannelConnectorFile
		if err := readJSON(filepath.Join(s.root, entrada.Name()), &arquivo); err != nil {
			return nil, err
		}
		conectores = append(conectores, arquivo.Connectors...)
	}
	return conectores, nil
}

// Save grava os conectores da organização.
func (s *OmnichannelConnectorStore) Save(organizationID string, connectors []OmnichannelConnector) error {
	caminho := s.path(organizationID)
	if caminho == "" {
		return nil
	}
	// Cópia explícita: o arquivo recebe o REGISTRO, e nunca o valor do segredo (que não está nesta struct).
	paraGravar := make([]OmnichannelConnector, 0, len(connectors))
	paraGravar = append(paraGravar, connectors...)
	return writeJSONAtomic(caminho, omnichannelConnectorFile{Connectors: paraGravar})
}

// omnichannelConnectorRoot devolve o diretório dos conectores dentro da raiz de dados.
func omnichannelConnectorRoot(dataRoot string) string {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return ""
	}
	return filepath.Join(dataRoot, ".agent-omnichannel-connectors")
}
