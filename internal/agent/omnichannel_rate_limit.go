package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PERSISTÊNCIA DA JANELA DE RATE LIMIT do Omnichannel (E-170).
//
// O limite declarado era: *"rate-limit/replay em memória"*. Vale separar as duas coisas, porque elas não estavam
// no mesmo estado:
//
//   - a **proteção de replay** de webhook JÁ é durável (`WebhookReplayStore`, com trava de disco, aluguel e
//     retenção de 30 dias): a nota de continuidade estava ERRADA nesse ponto;
//   - o **rate limit por organização/canal** estava **só em memória** (`r.buckets`), e isso tem um defeito
//     específico: **reiniciar o processo ZERA a cota**. Quem estourou o limite e reinicia o serviço (ou explora
//     um reinício) volta a ter janela cheia — a proteção desaparece exatamente quando é atacada.
//
// Decisões, seguindo o padrão que o DLQ já usa neste pacote:
//
//  1. **UM ARQUIVO POR ORGANIZAÇÃO** (`<dataRoot>/.agent-omnichannel-rate/<org>.json`): o isolamento entre
//     tenants vale no disco, não só no filtro de leitura;
//  2. **ESCRITA ATÔMICA** por `writeJSONAtomic` — que já respeita a política de durabilidade de E-168/E-169;
//  3. **SEM STORE, O ESTADO DIZ QUE NÃO HÁ PERSISTÊNCIA** (`RateLimitPersisted()==false`), em vez de fingir;
//  4. **FALHA DE PERSISTÊNCIA É REPORTADA** (`LastRateLimitError()`), nunca engolida.

// OmnichannelRateLimitStore guarda as janelas de rate limit por organização.
type OmnichannelRateLimitStore struct {
	root string
}

type omnichannelRateLimitFile struct {
	// Buckets é indexado pelo CANAL; a organização está no nome do arquivo.
	Buckets map[string]omnichannelRateLimitBucket `json:"buckets"`
}

type omnichannelRateLimitBucket struct {
	WindowStart time.Time `json:"window_start"`
	Count       int       `json:"count"`
}

// NewOmnichannelRateLimitStore cria o store no diretório informado.
func NewOmnichannelRateLimitStore(root string) *OmnichannelRateLimitStore {
	return &OmnichannelRateLimitStore{root: strings.TrimSpace(root)}
}

// Root devolve o diretório usado (vazio quando não há persistência configurada).
func (s *OmnichannelRateLimitStore) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *OmnichannelRateLimitStore) path(organizationID string) string {
	if s == nil || s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

// Load devolve as janelas persistidas da organização.
//
// Arquivo ausente é estado normal (ainda não houve evento). Qualquer outro erro é DEVOLVIDO: uma janela que não
// pôde ser lida é uma proteção que não existe, e isso precisa ser dito.
func (s *OmnichannelRateLimitStore) Load(organizationID string) (map[string]omnichannelRateLimitBucket, error) {
	if err := requireOrganization(organizationID); err != nil {
		return nil, err
	}
	caminho := s.path(organizationID)
	if caminho == "" {
		return nil, nil
	}
	var arquivo omnichannelRateLimitFile
	err := readJSON(caminho, &arquivo)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if arquivo.Buckets == nil {
		return nil, nil
	}
	return arquivo.Buckets, nil
}

// Save grava as janelas da organização.
func (s *OmnichannelRateLimitStore) Save(organizationID string, buckets map[string]omnichannelRateLimitBucket) error {
	if err := requireOrganization(organizationID); err != nil {
		return err
	}
	caminho := s.path(organizationID)
	if caminho == "" {
		return nil
	}
	return writeJSONAtomic(caminho, omnichannelRateLimitFile{Buckets: buckets})
}

// janelaDeRate ainda vale? Uma janela fora do período não protege nada e não deve ser restaurada.
func janelaDeRateValida(inicio time.Time, agora time.Time) bool {
	if inicio.IsZero() {
		return false
	}
	decorrido := agora.Sub(inicio)
	return decorrido >= 0 && decorrido < OmnichannelRateWindow
}

// omnichannelRateLimitRoot devolve o diretório da janela de rate limit dentro da raiz de dados (mesmo padrão da
// DLQ: um diretório por finalidade, um arquivo por organização).
func omnichannelRateLimitRoot(dataRoot string) string {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return ""
	}
	return filepath.Join(dataRoot, ".agent-omnichannel-rate")
}
