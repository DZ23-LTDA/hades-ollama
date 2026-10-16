package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Aprovação HUMANA para ferramentas de EFEITO (estágio 35).
//
// O limite declarado era: *"a escrita segue sem aprovação humana"*. Um agente que altera CRM, caixa de
// entrada ou atribuição por conta de uma frase do cliente é exatamente o risco que o documento de missão
// proíbe ("aprovação humana é necessária conforme risco"; `REQUER_APROVACAO` é um dos estados de permissão
// declarados no contexto).
//
// Decisões:
//
//  1. **LEITURA não precisa de aprovação; EFEITO precisa.** Bloquear leitura tornaria o agente inútil sem
//     ganho de segurança.
//  2. **A aprovação é AMARRADA aos PARÂMETROS da ação** (impressão digital de ferramenta + escopo + valores):
//     aprovar "criar oportunidade de 100" não autoriza criar de 10.000.
//  3. **USO ÚNICO**: ao executar, a aprovação é CONSUMIDA. Aprovação reutilizável seria autorização
//     permanente disfarçada de aprovação pontual.
//  4. **PERSISTE por organização**: aprovação que desaparece no reinício obriga o humano a decidir de novo —
//     e quem não entende por quê aprova no automático.
//  5. **NEGADO é decidido, não ignorado**: a tentativa seguinte é BLOQUEADA com o motivo e quem recusou.
//  6. **A aprovação VENCE**: passado o prazo (`ApprovalTTL`), ela deixa de valer e o turno exige decisão nova.
//     Sem prazo, uma autorização esquecida continua valendo indefinidamente — que é o oposto de "pontual".
//  7. **Quem aprova é conferido** quando existe lista de aprovadores da organização; sem lista, a decisão é
//     registrada com `RoleChecked = false`, para que a ausência de verificação seja DADO, não suposição.

// DefaultServiceAgentApprovalRetention é o teto de aprovações guardadas por organização.
const DefaultServiceAgentApprovalRetention = 500

// DefaultServiceAgentApprovalTTL é o prazo padrão de vigência de uma aprovação depois da decisão.
//
// Trinta minutos: tempo suficiente para o turno ser repetido por quem pediu, e curto o bastante para que uma
// autorização esquecida não continue valendo no dia seguinte. `ApprovalTTL = 0` desliga o prazo — e desligar
// é uma escolha EXPLÍCITA, registrada no estado (`TTLSeconds = 0`), não um esquecimento.
const DefaultServiceAgentApprovalTTL = 30 * time.Minute

// Estados e resultados de decisão.
const (
	ServiceAgentApprovalPending  = "pending"
	ServiceAgentApprovalApproved = "approved"
	ServiceAgentApprovalDenied   = "denied"
	ServiceAgentApprovalConsumed = "consumed"
	// ServiceAgentApprovalExpired: aprovada, mas o prazo passou antes de ser usada. É evidência de que a
	// autorização venceu — não um pedido que sumiu.
	ServiceAgentApprovalExpired = "expired"
)

// ServiceAgentApproval é um pedido de autorização para uma ferramenta de efeito.
type ServiceAgentApproval struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	AgentID        string `json:"agent_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	ToolName       string `json:"tool_name"`
	// Fingerprint amarra a aprovação a PARÂMETROS e ESCOPO: aprovar um valor não autoriza outro.
	Fingerprint string     `json:"fingerprint"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	RequestedBy string     `json:"requested_by,omitempty"`
	DecidedAt   *time.Time `json:"decided_at,omitempty"`
	DecidedBy   string     `json:"decided_by,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	ConsumedAt  *time.Time `json:"consumed_at,omitempty"`
	// ExpiresAt é o instante em que a aprovação deixa de valer (DecidedAt + TTL). Ausente quando não há prazo.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// RoleChecked diz se a decisão passou por LISTA DE APROVADORES. `false` é declarado: sem lista
	// configurada o sistema exige ator identificado, mas NÃO verifica se ele pode aprovar — e isso precisa
	// aparecer no registro, em vez de virar confiança implícita.
	RoleChecked bool `json:"role_checked"`
	// ValueCents é o valor da ação aprovada (0 quando a ferramenta não lida com valor). Ele existe para o
	// QUÓRUM: acima do limite configurado, uma decisão não basta.
	ValueCents int64 `json:"value_cents,omitempty"`
	// RequiredApprovals é quantas decisões FAVORÁVEIS de atores DISTINTOS são exigidas. Fica gravado no
	// pedido: mudar a configuração depois não altera o que já foi exigido de quem pediu.
	RequiredApprovals int `json:"required_approvals"`
	// Decisions guarda CADA decisão tomada — com autor e instante. É o que permite exigir atores distintos
	// e provar quem aprovou o quê.
	Decisions []ServiceAgentApprovalDecision `json:"decisions,omitempty"`
}

// ServiceAgentApprovalDecision é uma decisão individual sobre o pedido.
type ServiceAgentApprovalDecision struct {
	ActorID   string    `json:"actor_id"`
	Approved  bool      `json:"approved"`
	Reason    string    `json:"reason,omitempty"`
	DecidedAt time.Time `json:"decided_at"`
}

// Approvers é a lista de atores DISTINTOS que aprovaram.
func (a ServiceAgentApproval) Approvers() []string {
	vistos := map[string]bool{}
	out := make([]string, 0, len(a.Decisions))
	for _, decisao := range a.Decisions {
		if !decisao.Approved || vistos[decisao.ActorID] {
			continue
		}
		vistos[decisao.ActorID] = true
		out = append(out, decisao.ActorID)
	}
	return out
}

// QuorumMet diz se o pedido já tem aprovações suficientes.
func (a ServiceAgentApproval) QuorumMet() bool {
	required := a.RequiredApprovals
	if required <= 0 {
		required = 1
	}
	return len(a.Approvers()) >= required
}

// ServiceAgentApprovalStore persiste as aprovações por organização.
type ServiceAgentApprovalStore struct {
	root string
	// MaxPerOrganization limita quantas aprovações cada organização guarda.
	MaxPerOrganization int
	// ApprovalTTL é o prazo de vigência depois da decisão; zero significa SEM prazo.
	ApprovalTTL time.Duration
	// PendingTTL é o prazo máximo de um pedido PENDENTE (mesmo com aprovação parcial). Zero desliga.
	//
	// Sem isto, um pedido com uma assinatura fica pendente para sempre — e a segunda assinatura, semanas
	// depois, se somaria a um primeiro olhar que já não corresponde ao mesmo contexto. Quórum com assinatura
	// velha não é quórum: é carimbo.
	PendingTTL time.Duration
	// QuorumAboveCents ativa o QUÓRUM acima deste valor: uma decisão deixa de bastar. Zero desliga.
	QuorumAboveCents int64
	// QuorumApprovals é quantas decisões de atores DISTINTOS o quórum exige.
	QuorumApprovals int
	// Notifier avisa que existe pedido esperando decisão. Nil significa SEM aviso (padrão).
	Notifier ServiceAgentNotifier

	notifyMu          sync.Mutex
	notificationsSent int
	lastNotifyError   string
}

// NotificationsSent diz quantos avisos foram ENTREGUES (prova em teste e visão de operação).
func (s *ServiceAgentApprovalStore) NotificationsSent() int {
	s.notifyMu.Lock()
	defer s.notifyMu.Unlock()
	return s.notificationsSent
}

// LastNotificationError devolve a falha do último aviso — sem URL e sem segredo.
func (s *ServiceAgentApprovalStore) LastNotificationError() string {
	s.notifyMu.Lock()
	defer s.notifyMu.Unlock()
	return s.lastNotifyError
}

// notifyPending avisa que um pedido NOVO está esperando — e NUNCA falha o pedido por causa do aviso.
//
// O controle é a aprovação humana; o aviso é conveniência. Um destino fora do ar que travasse a escrita
// seria trocar um problema de operação por um incidente.
func (s *ServiceAgentApprovalStore) notifyPending(approval ServiceAgentApproval) {
	if s == nil || s.Notifier == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), DefaultServiceAgentNotifyTimeout)
	defer cancel()
	err := s.Notifier.NotifyPendingApproval(ctx, ServiceAgentApprovalNotification{
		Event:             "approval.pending",
		ApprovalID:        approval.ID,
		OrganizationID:    approval.OrganizationID,
		ToolName:          approval.ToolName,
		ValueCents:        approval.ValueCents,
		RequiredApprovals: approval.RequiredApprovals,
		RequestedAt:       approval.RequestedAt,
	})
	s.notifyMu.Lock()
	defer s.notifyMu.Unlock()
	if err != nil {
		s.lastNotifyError = err.Error()
		return
	}
	s.notificationsSent++
	s.lastNotifyError = ""
}

// DefaultServiceAgentQuorumAboveCents é o limite a partir do qual UMA decisão não basta.
//
// R$ 1.000 (em centavos): alto o bastante para merecer dois olhares e baixo o bastante para o controle
// existir de fato na operação. Configurável, e `0` desliga — declarado, não implícito.
const DefaultServiceAgentQuorumAboveCents int64 = 100000

// DefaultServiceAgentQuorumApprovals é quantas decisões de atores distintos o quórum exige.
const DefaultServiceAgentQuorumApprovals = 2

// DefaultServiceAgentPendingTTL é quanto um pedido pode esperar por decisão (24 horas).
//
// Um dia: tempo de sobra para uma pessoa decidir em horário comercial, e curto o bastante para uma
// autorização parcialmente assinada não atravessar a semana acumulando contexto que ninguém revisou.
const DefaultServiceAgentPendingTTL = 24 * time.Hour

// OverduePendingThreshold é a partir de quanto tempo um pedido pendente é marcado como ATRASADO na fila.
//
// A marcação não decide nada: ela diz a quem olha a fila que aquele pedido está esperando demais — sem isso,
// "pendente" e "esquecido" ficam indistinguíveis na tela.
const OverduePendingThreshold = 4 * time.Hour

type serviceAgentApprovalFile struct {
	Approvals []ServiceAgentApproval `json:"approvals"`
	// Approvers é a lista de quem PODE aprovar nesta organização. Vazia significa "qualquer ator
	// identificado", e as decisões nesse modo saem com RoleChecked = false.
	Approvers []string `json:"approvers,omitempty"`
}

// NewServiceAgentApprovalStore cria o store de aprovações.
func NewServiceAgentApprovalStore(root string) *ServiceAgentApprovalStore {
	return &ServiceAgentApprovalStore{
		root:               strings.TrimSpace(root),
		MaxPerOrganization: DefaultServiceAgentApprovalRetention,
		ApprovalTTL:        DefaultServiceAgentApprovalTTL,
		PendingTTL:         DefaultServiceAgentPendingTTL,
		// O quórum vem LIGADO por padrão para valores altos: proteção que precisa ser lembrada de ligar não
		// protege. Desligar é uma escolha explícita (`QuorumAboveCents = 0`).
		QuorumAboveCents: DefaultServiceAgentQuorumAboveCents,
		QuorumApprovals:  DefaultServiceAgentQuorumApprovals,
	}
}

// RequiredApprovalsFor diz quantas decisões distintas o valor exige.
func (s *ServiceAgentApprovalStore) RequiredApprovalsFor(valueCents int64) int {
	if s == nil || s.QuorumAboveCents <= 0 || valueCents <= s.QuorumAboveCents {
		return 1
	}
	if s.QuorumApprovals <= 1 {
		return 1
	}
	return s.QuorumApprovals
}

// Root devolve o diretório usado (vazio quando não há persistência).
func (s *ServiceAgentApprovalStore) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *ServiceAgentApprovalStore) path(organizationID string) string {
	if s == nil || s.root == "" {
		return ""
	}
	return filepath.Join(s.root, sanitizeIDComponent(organizationID)+".json")
}

// Load devolve as aprovações da organização.
func (s *ServiceAgentApprovalStore) Load(organizationID string) ([]ServiceAgentApproval, error) {
	if err := requireOrganization(organizationID); err != nil {
		return nil, err
	}
	file, err := s.loadFile(organizationID)
	if err != nil {
		return nil, err
	}
	return file.Approvals, nil
}

// loadFile devolve o arquivo inteiro (aprovações + lista de aprovadores).
func (s *ServiceAgentApprovalStore) loadFile(organizationID string) (serviceAgentApprovalFile, error) {
	path := s.path(organizationID)
	if path == "" {
		return serviceAgentApprovalFile{}, nil
	}
	var file serviceAgentApprovalFile
	err := readJSON(path, &file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return serviceAgentApprovalFile{}, nil
		}
		// Arquivo ilegível é REPORTADO: tratar como "sem aprovações" esconderia pedidos pendentes.
		return serviceAgentApprovalFile{}, err
	}
	return file, nil
}

// ttl devolve o prazo efetivo (zero significa SEM prazo, escolha explícita).
func (s *ServiceAgentApprovalStore) ttl() time.Duration {
	if s == nil || s.ApprovalTTL <= 0 {
		return 0
	}
	return s.ApprovalTTL
}

// SetApprovers define QUEM pode aprovar nesta organização. Lista vazia desliga a checagem de papel — e as
// decisões passam a ser registradas com `RoleChecked = false`, para a ausência de verificação ser visível.
// E-907 / ALTO-4: `SetApprovers` fazia leitura-modificacao-escrita SEM trava, como o `Consume` (E-424) e o
// `Decide` (E-426) faziam. *Fechar a janela no `Consume` e no `Decide` e deixa-la aberta nos tres irmaos
// (`SetApprovers`, `Save`, `expireStale`) e' exactamente o que o `E-426` avisou por escrito:*
// ***«Fechar a janela num sitio e deixa-la aberta no irmao e' o comeco do proximo incidente.»***
//
// O dano medido desta vez: um `ExpireStale`/`Save`/`SetApprovers` que LEU antes de um `Consume` gravar
// reescreve o ficheiro inteiro com o instantaneo ANTIGO - e a aprovacao CONSUMIDA **ressuscita**.
// `loadFile`/`saveFile` NAO travam por dentro, logo o corpo `*Locked` pode ser chamado sob a trava sem
// deadlock (o mesmo argumento que o `E-426` registou).
func (s *ServiceAgentApprovalStore) SetApprovers(organizationID string, approvers []string) error {
	path := s.path(organizationID)
	if path == "" {
		return s.setApproversLocked(organizationID, approvers)
	}
	// E-430: a pasta tem de existir ANTES de a trava existir.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return withFileLock(path+".lock", func() error {
		return s.setApproversLocked(organizationID, approvers)
	})
}

// setApproversLocked e' o corpo do `SetApprovers`, executado SOB a trava da organizacao.
func (s *ServiceAgentApprovalStore) setApproversLocked(organizationID string, approvers []string) error {
	file, err := s.loadFile(organizationID)
	if err != nil {
		return err
	}
	limpos := make([]string, 0, len(approvers))
	for _, approver := range approvers {
		if nome := strings.TrimSpace(approver); nome != "" {
			limpos = append(limpos, nome)
		}
	}
	file.Approvers = limpos
	return s.saveFile(organizationID, file)
}

// Approvers devolve a lista de aprovadores da organização (vazia significa sem checagem de papel).
func (s *ServiceAgentApprovalStore) Approvers(organizationID string) ([]string, error) {
	file, err := s.loadFile(organizationID)
	if err != nil {
		return nil, err
	}
	return file.Approvers, nil
}

// Save grava as aprovacoes respeitando o teto e PRESERVANDO a lista de aprovadores da organizacao.
//
// E-907 / ALTO-4: era leitura-modificacao-escrita SEM trava. *Agora e' o wrapper publico; o corpo virou
// `saveLocked` para que os chamadores que JA tem a trava (`requestLocked`, `expireStaleLocked`) o usem SEM
// trava aninhada - um `withFileLock` dentro de outro trava para sempre.*
func (s *ServiceAgentApprovalStore) Save(organizationID string, approvals []ServiceAgentApproval) error {
	if err := requireOrganization(organizationID); err != nil {
		return err
	}
	path := s.path(organizationID)
	if path == "" {
		return s.saveLocked(organizationID, approvals)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return withFileLock(path+".lock", func() error {
		return s.saveLocked(organizationID, approvals)
	})
}

// saveLocked e' o corpo do `Save`, executado SOB a trava da organizacao (ou sem persistencia).
func (s *ServiceAgentApprovalStore) saveLocked(organizationID string, approvals []ServiceAgentApproval) error {
	file, err := s.loadFile(organizationID)
	if err != nil {
		return err
	}
	// UMA leitura serve as duas coisas: aplicar a invariante precisa do estado atual, e o estado atual e' o
	// que acabamos de ler.
	file.Approvals = preservarConsumidas(file.Approvals, approvals)
	return s.saveFile(organizationID, file)
}

// preservarConsumidas aplica a invariante **O CONSUMO E' TERMINAL** antes de uma gravacao que substitua a
// lista inteira de aprovacoes.
//
// E-907 / ALTO-4: `Save` grava o FICHEIRO INTEIRO. Um chamador que passe um instantaneo lido ANTES de um
// `Consume` rebaixaria a aprovacao consumida de volta a "aprovada" - *e uma ferramenta ja' executada poderia
// executar DE NOVO*. A trava sozinha NAO impede isto: a trava serializa os escritores, mas nao corrige o
// conteudo que o chamador entrega. *Foi o que o teste discriminador apanhou.*
//
// A regra: uma aprovacao que esta' CONSUMIDA no disco NAO PODE sair de consumida por uma gravacao que a
// substitua. Quem quiser reabrir cria um pedido NOVO - *que e' o que o pedido de aprovacao existe para tornar
// visivel e auditavel.*
//
// Nota: `setApproversLocked` NAO chama esta funcao porque NAO aceita lista de aprovacoes de fora - ele lê sob
// a trava e so' substitui `file.Approvers`. E `expireStaleLocked` idem: lê sob a trava e so' promove para
// `expired`.
func preservarConsumidas(atual, proposta []ServiceAgentApproval) []ServiceAgentApproval {
	consumidas := make(map[string]ServiceAgentApproval, len(atual))
	for _, a := range atual {
		if a.Status == ServiceAgentApprovalConsumed {
			consumidas[a.ID] = a
		}
	}
	if len(consumidas) == 0 {
		return proposta
	}
	for i := range proposta {
		if doDisco, ok := consumidas[proposta[i].ID]; ok && proposta[i].Status != ServiceAgentApprovalConsumed {
			proposta[i] = doDisco
		}
	}
	return proposta
}

// saveFile grava o arquivo aplicando o teto por organização.
func (s *ServiceAgentApprovalStore) saveFile(organizationID string, file serviceAgentApprovalFile) error {
	path := s.path(organizationID)
	if path == "" {
		return nil
	}
	approvals := file.Approvals
	limit := s.MaxPerOrganization
	if limit <= 0 {
		limit = DefaultServiceAgentApprovalRetention
	}
	if len(approvals) > limit {
		// As PENDENTES ficam: descartar um pedido pendente por causa de volume silenciaria uma decisão
		// que alguém precisa tomar. O corte atinge primeiro as já decididas.
		pendentes := make([]ServiceAgentApproval, 0, len(approvals))
		decididas := make([]ServiceAgentApproval, 0, len(approvals))
		for _, approval := range approvals {
			if approval.Status == ServiceAgentApprovalPending {
				pendentes = append(pendentes, approval)
			} else {
				decididas = append(decididas, approval)
			}
		}
		sobra := limit - len(pendentes)
		if sobra < 0 {
			sobra = 0
		}
		if len(decididas) > sobra {
			decididas = decididas[len(decididas)-sobra:]
		}
		approvals = append(pendentes, decididas...)
		sort.SliceStable(approvals, func(i, j int) bool { return approvals[i].RequestedAt.Before(approvals[j].RequestedAt) })
	}
	file.Approvals = approvals
	return writeJSONAtomic(path, file)
}

// expireStale marca como EXPIRADAS as aprovações cujo prazo passou — depois da decisão (`ApprovalTTL`) ou
// ainda PENDENTES (`PendingTTL`, com ou sem assinatura parcial).
//
// Marcar (em vez de apenas ignorar) é o que transforma "autorização vencida" em EVIDÊNCIA: quem audita
// precisa ver que houve pedido/aprovação e que venceu, não um registro que simplesmente deixou de valer.
//
// A expiração do PENDENTE é o que impede o desfecho pior do quórum: uma primeira assinatura esquecida que,
// semanas depois, se somaria a um segundo olhar sobre um contexto que ninguém revisou junto.
// E-907 / ALTO-4: era leitura-modificacao-escrita SEM trava - e a sua gravacao e' a mais perigosa das tres,
// porque reescreve o FICHEIRO INTEIRO: um `expireStale` que leu antes de um `Consume` gravar marca expiradas
// sobre um instantaneo antigo e **desfaz o consumo**. Agora e' o wrapper; o corpo virou `expireStaleLocked`.
func (s *ServiceAgentApprovalStore) expireStale(organizationID string, now time.Time) error {
	path := s.path(organizationID)
	if path == "" {
		return s.expireStaleLocked(organizationID, now)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return withFileLock(path+".lock", func() error {
		return s.expireStaleLocked(organizationID, now)
	})
}

// expireStaleLocked e' o corpo do `expireStale`, executado SOB a trava da organizacao.
func (s *ServiceAgentApprovalStore) expireStaleLocked(organizationID string, now time.Time) error {
	ttl := s.ttl()
	pendingTTL := s.pendingTTL()
	if ttl <= 0 && pendingTTL <= 0 {
		return nil
	}
	// Releitura SOB a trava: a lista que o chamador viu antes pode ja' ter sido alterada por um `Consume`.
	// Sem esta releitura, a gravacao seguinte ressuscitaria o que o `Consume` marcou como consumido.
	file, err := s.loadFile(organizationID)
	if err != nil {
		return err
	}
	mudou := false
	for index := range file.Approvals {
		approval := &file.Approvals[index]
		switch approval.Status {
		case ServiceAgentApprovalApproved:
			if approval.DecidedAt == nil || ttl <= 0 {
				continue
			}
			if now.UTC().After(approval.DecidedAt.UTC().Add(ttl)) {
				approval.Status = ServiceAgentApprovalExpired
				mudou = true
			}
		case ServiceAgentApprovalPending:
			if pendingTTL <= 0 {
				continue
			}
			if now.UTC().After(approval.RequestedAt.UTC().Add(pendingTTL)) {
				approval.Status = ServiceAgentApprovalExpired
				mudou = true
			}
		}
	}
	if !mudou {
		return nil
	}
	return s.saveFile(organizationID, file)
}

// pendingTTL devolve o prazo efetivo do pedido pendente (zero significa SEM prazo).
func (s *ServiceAgentApprovalStore) pendingTTL() time.Duration {
	if s == nil || s.PendingTTL <= 0 {
		return 0
	}
	return s.PendingTTL
}

// ServiceAgentApprovalQueueItem é o que quem espera precisa ver: o que falta e há quanto tempo.
type ServiceAgentApprovalQueueItem struct {
	ID              string `json:"id"`
	ToolName        string `json:"tool_name"`
	ConversationID  string `json:"conversation_id,omitempty"`
	ApprovalsGiven  int    `json:"approvals_given"`
	ApprovalsNeeded int    `json:"approvals_needed"`
	AgeSeconds      int64  `json:"age_seconds"`
	// Overdue diz que o pedido passou do limiar de atenção — sem isto, "pendente" e "esquecido" ficam
	// indistinguíveis para quem olha a fila.
	Overdue bool `json:"overdue"`
}

// ServiceAgentApprovalQueue é o resumo da fila de aprovações de uma organização.
type ServiceAgentApprovalQueue struct {
	OrganizationID string                          `json:"organization_id"`
	Pending        int                             `json:"pending"`
	Overdue        int                             `json:"overdue"`
	OldestSeconds  int64                           `json:"oldest_seconds"`
	PendingTTL     int64                           `json:"pending_ttl_seconds"`
	ExpiresCount   int                             `json:"expired"`
	Items          []ServiceAgentApprovalQueueItem `json:"items"`
}

// ApprovalQueue monta a fila de aprovações com idade e atraso de cada pedido.
//
// É a forma HONESTA de "notificar": não há push, e-mail nem webhook — o que existe é uma fila que diz o que
// falta, há quanto tempo e o que já passou do limiar de atenção.
func (s *ServiceAgentApprovalStore) ApprovalQueue(organizationID string, now time.Time) (ServiceAgentApprovalQueue, error) {
	if err := s.expireStale(organizationID, now); err != nil {
		return ServiceAgentApprovalQueue{}, err
	}
	approvals, err := s.Load(organizationID)
	if err != nil {
		return ServiceAgentApprovalQueue{}, err
	}
	queue := ServiceAgentApprovalQueue{
		OrganizationID: organizationID,
		PendingTTL:     int64(s.pendingTTL().Seconds()),
		Items:          make([]ServiceAgentApprovalQueueItem, 0, 4),
	}
	for _, approval := range approvals {
		if approval.OrganizationID != organizationID {
			continue
		}
		if approval.Status == ServiceAgentApprovalExpired {
			queue.ExpiresCount++
			continue
		}
		if approval.Status != ServiceAgentApprovalPending {
			continue
		}
		idade := now.UTC().Sub(approval.RequestedAt.UTC())
		if idade < 0 {
			idade = 0
		}
		needed := approval.RequiredApprovals
		if needed <= 0 {
			needed = 1
		}
		item := ServiceAgentApprovalQueueItem{
			ID:              approval.ID,
			ToolName:        approval.ToolName,
			ConversationID:  approval.ConversationID,
			ApprovalsGiven:  len(approval.Approvers()),
			ApprovalsNeeded: needed,
			AgeSeconds:      int64(idade.Seconds()),
			Overdue:         idade >= OverduePendingThreshold,
		}
		if item.Overdue {
			queue.Overdue++
		}
		if item.AgeSeconds > queue.OldestSeconds {
			queue.OldestSeconds = item.AgeSeconds
		}
		queue.Items = append(queue.Items, item)
	}
	queue.Pending = len(queue.Items)
	// Mais antigos primeiro: a fila existe para atacar o que está esperando há mais tempo.
	sort.SliceStable(queue.Items, func(i, j int) bool { return queue.Items[i].AgeSeconds > queue.Items[j].AgeSeconds })
	return queue, nil
}

// E-907 / ALTO-4: era a QUARTA leitura-modificacao-escrita sem trava. Era a mais subtil das quatro porque
// chama `Save` por dentro - *e o `Save` passou a ser um wrapper com trava, logo chama-lo daqui daria
// **trava aninhada**, que neste helper segura para sempre.* Por isso o corpo virou `requestLocked` e usa
// `saveFile` diretamente, com a **releitura e a deduplicacao DENTRO da trava**.
//
// A deduplicacao dentro da trava importa: dois `Request` simultâneos para a mesma impressao digital liam a
// lista antes de qualquer gravacao, nao se viam, e criavam **dois pedidos pendentes** - *exactamente o que
// esta funcao existe para impedir, e o numero de decisoes que faltam apareceria inflado.*
func (s *ServiceAgentApprovalStore) Request(organizationID string, request ServiceAgentApproval) (ServiceAgentApproval, error) {
	path := s.path(organizationID)
	if path == "" {
		resultado, criou, err := s.requestLocked(organizationID, request)
		if err != nil {
			return ServiceAgentApproval{}, err
		}
		if criou {
			s.notifyPending(resultado)
		}
		return resultado, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return ServiceAgentApproval{}, err
	}
	var resultado ServiceAgentApproval
	criou := false
	err := withFileLock(path+".lock", func() error {
		var errInterno error
		resultado, criou, errInterno = s.requestLocked(organizationID, request)
		return errInterno
	})
	if err != nil {
		return ServiceAgentApproval{}, err
	}
	// Aviso DEPOIS de persistir E FORA da trava: avisar sobre pedido que nao ficou gravado seria alarme falso,
	// e avisar com a trava na mao aumentaria o tempo de posse sem necessidade.
	//
	// E SO' quando um pedido NOVO foi gravado: *repetir o turno enquanto o humano decide **nao pode** reenviar
	// o aviso* - e' o defeito que `TestServiceAgentApprovalNotificationSendsOnceWithMinimalPayload` discrimina.
	// Os dois caminhos de REUSO (`pendente` e `negado`) saem de `requestLocked` com `criou=false`.
	if criou {
		s.notifyPending(resultado)
	}
	return resultado, nil
}

// requestLocked e' o corpo do `Request`, executado SOB a trava da organizacao.
//
// Devolve `criou=true` apenas quando um pedido NOVO foi acrescentado ao ficheiro. O reuso de um pedido ja
// existente devolve `false`: quem decide se avisa e' o wrapper, e o aviso so' sai para pedido novo.
func (s *ServiceAgentApprovalStore) requestLocked(organizationID string, request ServiceAgentApproval) (ServiceAgentApproval, bool, error) {
	file, err := s.loadFile(organizationID)
	if err != nil {
		return ServiceAgentApproval{}, false, err
	}
	approvals := file.Approvals
	for _, existing := range approvals {
		if existing.Status == ServiceAgentApprovalPending && existing.Fingerprint == request.Fingerprint {
			return existing, false, nil
		}
		// Pedido NEGADO para a mesma impressao digital nao vira novo pedido: a decisao foi tomada.
		if existing.Status == ServiceAgentApprovalDenied && existing.Fingerprint == request.Fingerprint {
			return existing, false, nil
		}
	}
	if strings.TrimSpace(request.ID) == "" {
		request.ID = "apr_" + uuid.NewString()[:12]
	}
	request.Status = ServiceAgentApprovalPending
	if request.RequiredApprovals <= 0 {
		// O quorum fica GRAVADO no pedido: mudar a configuracao depois nao altera o que ja foi exigido.
		request.RequiredApprovals = s.RequiredApprovalsFor(request.ValueCents)
	}
	if request.RequestedAt.IsZero() {
		request.RequestedAt = time.Now().UTC()
	}
	file.Approvals = append(approvals, request)
	if err := s.saveFile(organizationID, file); err != nil {
		return ServiceAgentApproval{}, false, err
	}
	return request, true, nil
}

// Decide registra a decisão humana. Sem ator identificado não há aprovação: não se decide no anonimato.
//
// Com LISTA DE APROVADORES configurada para a organização, quem não está nela é RECUSADO — e a recusa é
// erro declarado, não um registro silencioso. Sem lista, a decisão é aceita e gravada com
// `RoleChecked = false`: a ausência de verificação vira DADO em vez de confiança implícita.
// 🚨 E-426: o `Decide` tinha a MESMA leitura-modificação-escrita sem trava do `Consume` (E-424). Aqui o dano é
// menor — *duas decisões simultâneas que leem a lista antes da gravação fazem uma **perder**, e o quórum fica
// **sub-contado**, que falha fechado* — mas é a mesma classe, e uma delas já custou um CRÍTICO.
// ***Fechar a janela num sítio e deixá-la aberta no irmão é o começo do próximo incidente.***
//
// A decisão e a gravação passam a ser o mesmo passo indivisível, sob a trava da organização. O corpo virou
// `decideLocked`; `loadFile`/`saveFile` NÃO travam por dentro (medido), então não há deadlock.
func (s *ServiceAgentApprovalStore) Decide(organizationID, approvalID string, approved bool, actorID, reason string, now time.Time) (ServiceAgentApproval, error) {
	path := s.path(organizationID)
	if path == "" {
		// Sem persistência não existe estado compartilhado a proteger (raiz vazia = store sem arquivo).
		return s.decideLocked(organizationID, approvalID, approved, actorID, reason, now)
	}
	// E-430 / ALTO-3: mesma correção do `Consume` — *a pasta tem de existir antes de a trava existir.*
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return ServiceAgentApproval{}, err
	}
	var resultado ServiceAgentApproval
	err := withFileLock(path+".lock", func() error {
		var errInterno error
		resultado, errInterno = s.decideLocked(organizationID, approvalID, approved, actorID, reason, now)
		return errInterno
	})
	return resultado, err
}

// decideLocked é o corpo do `Decide`, executado SOB a trava da organização.
func (s *ServiceAgentApprovalStore) decideLocked(organizationID, approvalID string, approved bool, actorID, reason string, now time.Time) (ServiceAgentApproval, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return ServiceAgentApproval{}, fmt.Errorf("%w: decidir aprovação exige o ator humano", ErrServiceAgentApprovalInvalid)
	}
	file, err := s.loadFile(organizationID)
	if err != nil {
		return ServiceAgentApproval{}, err
	}
	roleChecked := false
	if len(file.Approvers) > 0 {
		permitido := false
		for _, approver := range file.Approvers {
			if approver == actorID {
				permitido = true
				break
			}
		}
		if !permitido {
			return ServiceAgentApproval{}, fmt.Errorf("%w: %q não está na lista de aprovadores desta organização", ErrServiceAgentApprovalNotAllowed, actorID)
		}
		roleChecked = true
	}
	approvals := file.Approvals
	for index := range approvals {
		if approvals[index].ID != approvalID {
			continue
		}
		if approvals[index].Status == ServiceAgentApprovalConsumed {
			// Decidir sobre aprovação já usada reabriria uma autorização que o sistema consumiu.
			return ServiceAgentApproval{}, fmt.Errorf("%w: aprovação já consumida", ErrServiceAgentApprovalConflict)
		}
		if approvals[index].Status == ServiceAgentApprovalExpired {
			// Pedido VENCIDO não aceita decisão: aceitar reabriria um contexto que ninguém revisou a tempo —
			// e é justamente isso que o prazo do pendente existe para impedir.
			return ServiceAgentApproval{}, fmt.Errorf("%w: pedido vencido; é preciso um pedido novo", ErrServiceAgentApprovalExpired)
		}
		decidedAt := now.UTC()
		approvals[index].DecidedBy = actorID
		approvals[index].Reason = strings.TrimSpace(reason)
		approvals[index].RoleChecked = roleChecked
		approvals[index].ExpiresAt = nil
		// CADA decisão é registrada. É o que permite exigir atores DISTINTOS e provar quem aprovou o quê —
		// sobrescrever a decisão anterior faria o segundo "sim" apagar o primeiro.
		approvals[index].Decisions = append(approvals[index].Decisions, ServiceAgentApprovalDecision{
			ActorID: actorID, Approved: approved, Reason: strings.TrimSpace(reason), DecidedAt: decidedAt,
		})
		if !approved {
			// Uma recusa encerra o pedido: quórum existe para somar cuidados, não para superar objeção.
			approvals[index].DecidedAt = &decidedAt
			approvals[index].Status = ServiceAgentApprovalDenied
		} else if approvals[index].QuorumMet() {
			approvals[index].DecidedAt = &decidedAt
			approvals[index].Status = ServiceAgentApprovalApproved
			if ttl := s.ttl(); ttl > 0 {
				expiresAt := decidedAt.Add(ttl)
				approvals[index].ExpiresAt = &expiresAt
			}
		} else {
			// Quórum incompleto: segue PENDENTE, e o pedido carrega o progresso (n de m).
			approvals[index].Status = ServiceAgentApprovalPending
		}
		file.Approvals = approvals
		if err := s.saveFile(organizationID, file); err != nil {
			return ServiceAgentApproval{}, err
		}
		return approvals[index], nil
	}
	return ServiceAgentApproval{}, fmt.Errorf("%w: aprovação %q não existe nesta organização", ErrServiceAgentApprovalNotFound, approvalID)
}

// FindApproved devolve a aprovação APROVADA e VÁLIDA mais antiga para a impressão digital, se houver.
func (s *ServiceAgentApprovalStore) FindApproved(organizationID, fingerprint string, now time.Time) (ServiceAgentApproval, bool, error) {
	// Aprovação vencida é marcada antes da busca: assim o estado registrado conta a verdade, e o turno pede
	// decisão nova em vez de executar com autorização que já não valia.
	if err := s.expireStale(organizationID, now); err != nil {
		return ServiceAgentApproval{}, false, err
	}
	approvals, err := s.Load(organizationID)
	if err != nil {
		return ServiceAgentApproval{}, false, err
	}
	for _, approval := range approvals {
		// 🚨 E-426 (MEDIDO por auditoria independente): faltava aqui o mesmo controle que `ListApprovals` e
		// `AuditTrail` já fazem — *uma assimetria que denuncia descuido*. Hoje a separação por ARQUIVO protege,
		// mas defesa em profundidade que só existe num dos lados não é defesa: é sorte. O `Fingerprint` já contém
		// `org=`, então esta checagem é o **segundo** cinto — e o teste que alega isolamento entre tenants
		// (`service_agent_approvals_test.go:194`) era satisfeito só pela separação de arquivo.
		if approval.Status == ServiceAgentApprovalApproved &&
			approval.OrganizationID == consentScope(organizationID) &&
			approval.Fingerprint == fingerprint {
			return approval, true, nil
		}
	}
	return ServiceAgentApproval{}, false, nil
}

// FindDenied diz se a impressão digital já foi NEGADA por um humano.
func (s *ServiceAgentApprovalStore) FindDenied(organizationID, fingerprint string) (ServiceAgentApproval, bool, error) {
	approvals, err := s.Load(organizationID)
	if err != nil {
		return ServiceAgentApproval{}, false, err
	}
	for _, approval := range approvals {
		// E-426: mesmo controle do `FindApproved` — ver o comentário lá.
		if approval.Status == ServiceAgentApprovalDenied &&
			approval.OrganizationID == consentScope(organizationID) &&
			approval.Fingerprint == fingerprint {
			return approval, true, nil
		}
	}
	return ServiceAgentApproval{}, false, nil
}

// Consume marca a aprovação como USADA e devolve o registro. É o que impede reutilização.
//
// A checagem de PRAZO acontece AQUI também, além da busca: entre a decisão e o consumo pode ter passado
// tempo suficiente para a autorização vencer, e um caminho que consome sem conferir anularia o prazo.
// 🚨 E-424 / ADV-66 (CRÍTICO, medido por auditoria independente): **a janela que este comentário dizia ter fechado
// continuava ABERTA.** `Consume` era leitura-modificação-escrita **sem trava**: duas execuções simultâneas liam
// `approved` antes de qualquer escrita, ambas gravavam `consumed` e **ambas executavam**. Sonda do auditor, com o
// MESMO pedido e UMA decisão humana:
//
//	8 turnos simultâneos → executadas=3 escritas_no_CRM=3, todas com o MESMO ApprovalID
//	8 Consume simultâneos → 20 de 20 rodadas com MAIS DE UM consumo bem-sucedido
//
// ⇒ **Agora a leitura-modificação-escrita acontece SOB TRAVA DE ARQUIVO** (`withFileLock`, o mesmo primitivo que o
// outbox de push e o agendamento usam): *serializa entre goroutines e entre processos*. O corpo virou
// `consumeLocked` — *e `loadFile`/`saveFile` NÃO travam por dentro (medido), então não há deadlock.*
// ***"Uso único" só é verdade se a decisão de consumir e a gravação forem o mesmo passo indivisível.***
func (s *ServiceAgentApprovalStore) Consume(organizationID, approvalID string, now time.Time) (ServiceAgentApproval, error) {
	path := s.path(organizationID)
	if path == "" {
		// Sem persistência não existe estado compartilhado a proteger (raiz vazia = store sem arquivo).
		return s.consumeLocked(organizationID, approvalID, now)
	}
	// 🚨 E-430 / ALTO-3 (medido por auditoria independente): **a trava que eu acrescentei em E-424 exigia um
	// diretório que ninguém cria.** `withFileLock` usa `CreateFile(..., OPEN_ALWAYS)` no Windows e `open(..., O_CREAT)`
	// no Unix — **nenhum dos dois cria diretórios-PAI** ⇒ *numa organização que nunca teve aprovação, o erro era
	// `open file lock: O sistema não pode encontrar o caminho especificado.`* ✗, e o `switch` da rota mandava isso
	// para **400** — *um erro de INFRAESTRUTURA respondendo como erro de CLIENTE*, onde antes era **404**.
	//
	// E é literalmente a **terceira repetição** do defeito que `store.go:968-973` documenta ter corrigido na raiz:
	//   *"dois stores já falharam na PRIMEIRA gravação com 'caminho não encontrado' … Corrigir na raiz evita a
	//   terceira repetição do mesmo defeito."*
	//
	// ***Uma trava adquirida antes de a pasta existir não é proteção: é um novo jeito de falhar.***
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return ServiceAgentApproval{}, err
	}
	var resultado ServiceAgentApproval
	err := withFileLock(path+".lock", func() error {
		var errInterno error
		resultado, errInterno = s.consumeLocked(organizationID, approvalID, now)
		return errInterno
	})
	return resultado, err
}

// consumeLocked é o corpo do `Consume`, executado SOB a trava da organização.
//
// A checagem de PRAZO acontece AQUI também, além da busca: entre a decisão e o consumo pode ter passado
// tempo suficiente para a autorização vencer, e um caminho que consome sem conferir anularia o prazo.
func (s *ServiceAgentApprovalStore) consumeLocked(organizationID, approvalID string, now time.Time) (ServiceAgentApproval, error) {
	file, err := s.loadFile(organizationID)
	if err != nil {
		return ServiceAgentApproval{}, err
	}
	approvals := file.Approvals
	for index := range approvals {
		if approvals[index].ID != approvalID {
			continue
		}
		if approvals[index].Status != ServiceAgentApprovalApproved {
			// Sem esta checagem, duas execuções simultâneas consumiriam a mesma aprovação.
			return ServiceAgentApproval{}, fmt.Errorf("%w: aprovação não está aprovada", ErrServiceAgentApprovalConflict)
		}
		if ttl := s.ttl(); ttl > 0 && approvals[index].DecidedAt != nil &&
			now.UTC().After(approvals[index].DecidedAt.UTC().Add(ttl)) {
			// Vencida: marca e RECUSA. Aprovação fora do prazo não autoriza nada.
			approvals[index].Status = ServiceAgentApprovalExpired
			file.Approvals = approvals
			if err := s.saveFile(organizationID, file); err != nil {
				return ServiceAgentApproval{}, err
			}
			return ServiceAgentApproval{}, fmt.Errorf("%w: aprovação venceu antes de ser usada", ErrServiceAgentApprovalExpired)
		}
		consumedAt := now.UTC()
		approvals[index].Status = ServiceAgentApprovalConsumed
		approvals[index].ConsumedAt = &consumedAt
		file.Approvals = approvals
		if err := s.saveFile(organizationID, file); err != nil {
			return ServiceAgentApproval{}, err
		}
		return approvals[index], nil
	}
	return ServiceAgentApproval{}, fmt.Errorf("%w: aprovação %q não existe nesta organização", ErrServiceAgentApprovalNotFound, approvalID)
}

// ListApprovals lista as aprovações da organização por estado (vazio lista todas), mais recentes primeiro.
func (s *ServiceAgentApprovalStore) ListApprovals(organizationID, status string, now time.Time) ([]ServiceAgentApproval, error) {
	// Marca vencidas antes de listar: a listagem é o que um humano olha para decidir, e mostrar como
	// "aprovada" o que já venceu seria mentir para quem confia na tela.
	if err := s.expireStale(organizationID, now); err != nil {
		return nil, err
	}
	approvals, err := s.Load(organizationID)
	if err != nil {
		return nil, err
	}
	status = strings.TrimSpace(status)
	out := make([]ServiceAgentApproval, 0, len(approvals))
	for _, approval := range approvals {
		if approval.OrganizationID != organizationID {
			continue
		}
		if status != "" && approval.Status != status {
			continue
		}
		out = append(out, approval)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].RequestedAt.After(out[j].RequestedAt) })
	return out, nil
}

// Erros dedicados das aprovações.
var (
	// ErrServiceAgentToolScope: o pedido não identifica (ou identifica algo fora) o escopo da sessão.
	ErrServiceAgentToolScope        = errors.New("service agent tool scope is invalid")
	ErrServiceAgentApprovalInvalid  = errors.New("service agent approval request is invalid")
	ErrServiceAgentApprovalNotFound = errors.New("service agent approval not found")
	ErrServiceAgentApprovalConflict = errors.New("service agent approval state conflict")
	// ErrServiceAgentApprovalNotAllowed: o ator não está na lista de aprovadores da organização.
	ErrServiceAgentApprovalNotAllowed = errors.New("service agent approval actor is not allowed")
	// ErrServiceAgentApprovalExpired: a aprovação venceu antes de ser usada.
	ErrServiceAgentApprovalExpired = errors.New("service agent approval expired")
)

// serviceAgentApprovalFingerprint amarra a aprovação a FERRAMENTA + ESCOPO + PARÂMETROS.
//
// Só a ferramenta não basta: "criar oportunidade" com 100 centavos e com 10.000 centavos são decisões
// diferentes, e aprovar uma não pode liberar a outra. Só o escopo também não basta — o humano aprova uma
// AÇÃO, não um agente.
func serviceAgentApprovalFingerprint(request ServiceAgentToolRequest, toolName string) string {
	partes := []string{
		strings.TrimSpace(toolName),
		"org=" + consentScope(request.OrganizationID),
		"conversation=" + strings.TrimSpace(request.ConversationID),
		"contact=" + strings.TrimSpace(request.ContactID),
		"title=" + strings.TrimSpace(request.Title),
		fmt.Sprintf("value=%d", request.ValueCents),
		"owner=" + strings.TrimSpace(request.OwnerID),
		"assignee=" + strings.TrimSpace(request.Assignee),
		"company=" + strings.TrimSpace(request.CompanyID),
		"session=" + strings.TrimSpace(request.SessionID),
		"provider=" + strings.TrimSpace(request.Provider),
		"body=" + strings.TrimSpace(request.Body),
		// 🚨 E-425 (ALTO, medido por DUAS auditorias independentes):
		//   (a) `reason` NÃO estava aqui, e ele **é o efeito**: `inbox.handoff` grava
		//       `"transferido para atendimento humano: " + reason` na conversa. Sonda: aprovação pedida com
		//       reason="MOTIVO-APROVADO" e turno repetido com "MOTIVO-NAO-APROVADO" ⇒ `status=executed`, e a
		//       conversa gravou literalmente o motivo **não aprovado**. Controle: trocar `assignee` ⇒
		//       `approval_required` (*o probe distingue — não é artefato*);
		//   (b) `agent` NÃO estava aqui: pedido feito por `support`, aprovado, e um turno idêntico com
		//       `AgentID=receptionist` **executou** — *a aprovação de um agente consumida por outro*, que é
		//       literalmente o item 1 da missão ("aceite aprovação de… outro agente"). Agrava: `inbox.handoff`
		//       é declarado por **6 dos 7** agentes.
		// ***A impressão digital dizia "AMARRADA aos PARÂMETROS" e media doze deles; o efeito de dois deles
		// ficava de fora.***
		"reason=" + strings.TrimSpace(request.Reason),
		"agent=" + strings.TrimSpace(request.AgentID),
	}
	soma := sha256.Sum256([]byte(strings.Join(partes, "\x1f")))
	return hex.EncodeToString(soma[:])
}

// serviceAgentApprovalRoot devolve o diretório das aprovações dentro da raiz de dados.
func serviceAgentApprovalRoot(dataRoot string) string {
	dataRoot = strings.TrimSpace(dataRoot)
	if dataRoot == "" {
		return ""
	}
	return filepath.Join(dataRoot, ".agent-service-approvals")
}
