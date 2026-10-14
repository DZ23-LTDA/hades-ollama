package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// E-449 (ADV-65, P0 PARA A ENTREGA) — **A SUPERFÍCIE QUE FALTAVA AO MOTOR DE AGENDAMENTO.**
//
// **O QUE ESTAVA MEDIDO** (E-441): `SchedulingStore` — com `AddService`, `Resource`, `Book`, `Cancel`,
// `Reschedule`, fila de espera e no-show — **nunca era construído fora dos testes**. Busca ampla em todo o repositório:
// *em produção, as duas ocorrências de `NewSchedulingStore` eram o **comentário** e a **definição*** ✗ ⇒ ***o motor
// existia, era testado, e não estava ligado a nada*** ✓. O próprio projeto já nomeava essa classe no Estágio 26:
// *"a capacidade existia apenas no runtime: sem superfície, agendar e medir campanha não era…"*.
//
// **O QUE ESTAS ROTAS SÃO**: a porta de entrada. *Um motor sem superfície não é uma capacidade: é uma biblioteca.*
//
// **O QUE ELAS NÃO SÃO, e precisa ficar dito**: elas **não** implementam autenticação por papel, nem idempotência de
// booking, nem pagamento ✗ — *expõem o que o motor **já** faz, com o escopo de organização vindo do middleware*. E o
// **ADV-60** continua valendo: ***para OFERECER horário ao cliente, a rota usa `OpenSlots`*** (*que aplica turno,
// fechamento por data, intervalo, fuso e antecedência*), **não** `Availability` ✓.
//
// ***Expor a capacidade é o que a transforma em produto; expor sem teste é o que a transforma em promessa.***

// schedulingServices lista os serviços da organização. *Leitura pura: o escopo vem do middleware, não do corpo.*
func (a *agentAPI) schedulingServices(c *gin.Context) {
	if a.scheduling == nil {
		// FALHA FECHADO: *sem store não há superfície* — e dizer isso é melhor que devolver lista vazia, que pareceria
		// "não há serviços" em vez de "não há motor ligado".
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	servicos, err := a.scheduling.Services(a.organizationID(c))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"services": servicos})
}

// schedulingSlots devolve os horários REALMENTE oferecíveis numa data.
//
// ⚠️ ADV-60: usa **`OpenSlots`**, não `Availability`. *`Availability` decide apenas por agendamentos — não olha turno,
// fechamento por data, intervalo nem fuso; um slot marcado livre ali pode estar às 03:00 de um domingo com a empresa
// fechada.* ***"A função não mudou de comportamento — mudou de promessa."***
func (a *agentAPI) schedulingSlots(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	data := strings.TrimSpace(c.Query("date"))
	if data == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("o parâmetro `date` (AAAA-MM-DD) é obrigatório"))
		return
	}
	passo := 30
	// `step` é opcional e tem teto: *passo absurdo geraria centenas de milhares de slots.*
	if bruto := strings.TrimSpace(c.Query("step")); bruto != "" {
		convertido, err := parsePositiveInt(bruto)
		if err != nil || convertido <= 0 || convertido > 240 {
			writeAgentError(c, http.StatusBadRequest, errors.New("`step` precisa ser um inteiro entre 1 e 240 minutos"))
			return
		}
		passo = convertido
	}
	slots, err := a.scheduling.OpenSlots(a.organizationID(c), strings.TrimSpace(c.Query("resource_id")),
		data, strings.TrimSpace(c.Query("service_id")), passo)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"slots": slots, "count": len(slots)})
}

// schedulingBookRequest é o CONTRATO DE ENTRADA da superficie - **explicito e separado do motor.**
//
// 🚨 E-456 (medido): eu liguei `c.ShouldBindJSON` direto em `agent.BookRequest` ✗ e o agendamento devolveu **400 "todos
// obrigatorios"** - porque `BookRequest` **nao tem json tags** (`scheduling.go:259-268`): o `encoding/json` usa os
// **nomes dos campos** (`ResourceID`, `StartsAt`), o meu corpo mandava `resource_id`/`starts_at`, e **o corpo virou um
// zero silencioso** ✗.
//
// ***Expor a struct INTERNA como contrato de API acopla a API a implementacao - e o modo de falha e' silencioso: o
// pedido chega vazio, e o erro culpa os campos que o cliente mandou.*** A superficie passa a ter o seu proprio
// contrato, e a **traducao** para o motor e' explicita.
type schedulingBookRequest struct {
	ResourceID     string `json:"resource_id"`
	ServiceID      string `json:"service_id"`
	ContactID      string `json:"contact_id"`
	Channel        string `json:"channel"`
	StartsAt       string `json:"starts_at"`
	IdempotencyKey string `json:"idempotency_key"`
}

// schedulingBook cria um agendamento. *Escrita: passa pelo mesmo motor que os testes exercitam (`Book`), com as suas
// validacoes de sobreposicao, turno e organizacao.*
func (a *agentAPI) schedulingBook(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	var entrada schedulingBookRequest
	if err := c.ShouldBindJSON(&entrada); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	inicio, err := time.Parse(time.RFC3339, strings.TrimSpace(entrada.StartsAt))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest,
			errors.New("`starts_at` precisa ser RFC 3339 (ex.: 2026-10-14T10:00:00Z)"))
		return
	}
	// TRADUCAO EXPLICITA do contrato da superficie para o do motor.
	pedido := agent.BookRequest{
		// O escopo vem do MIDDLEWARE, não do corpo: *um corpo que escolhe a organização é um corpo que escolhe o tenant.*
		OrganizationID: a.organizationID(c),
		ResourceID:     strings.TrimSpace(entrada.ResourceID),
		ServiceID:      strings.TrimSpace(entrada.ServiceID),
		ContactID:      strings.TrimSpace(entrada.ContactID),
		Channel:        strings.TrimSpace(entrada.Channel),
		StartsAt:       inicio,
		IdempotencyKey: strings.TrimSpace(entrada.IdempotencyKey),
	}
	agendamento, err := a.scheduling.Book(pedido)
	if errors.Is(err, agent.ErrSchedulingDuplicate) {
		// 🚨 E-458: REENTREGA NÃO É ERRO DO CLIENTE. *O motor devolve, junto com `ErrSchedulingDuplicate`, o agendamento
		// que JA' EXISTE* - e a resposta honesta e' **409 com o mesmo recurso**, nao um 400 que sugere "pedido
		// malformado".
		//
		// ***Idempotência de verdade e' devolver o MESMO RECURSO.*** Um cliente que reentrega o pedido *- porque a rede
		// caiu, porque o agente repetiu -* quer o agendamento, nao uma licao sobre o formato do pedido.
		c.JSON(http.StatusConflict, gin.H{"appointment": agendamento, "idempotent_replay": true})
		return
	}
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"appointment": agendamento})
}

// E-456 — A JORNADA DO CLIENTE COMPLETA: LISTAR, CANCELAR, REMARCAR E FILA DE ESPERA.
//
// **O LIMITE QUE E-449 DECLAROU**: *"liguei o nucleo, nao o motor inteiro - `Cancel`/`Reschedule`/fila/`no-show` NAO tem
// rota"* ✗. *Um cliente que agenda e nao consegue cancelar nem remarcar nao tem um produto: tem uma demonstracao.*
//
// ⇒ estas rotas fecham a jornada. **E `FreeSlotsForReschedule` e' a peca que quase sempre falta**: *remarcar sem saber
// o que esta' livre obriga o cliente a adivinhar - e a adivinhacao vira horario inexistente, que o motor recusa depois.*

// schedulingAppointments lista os agendamentos da organizacao. *Leitura pura: o escopo vem do middleware.*
func (a *agentAPI) schedulingAppointments(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	agendamentos, err := a.scheduling.Appointments(a.organizationID(c))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"appointments": agendamentos, "count": len(agendamentos)})
}

// schedulingCancel cancela um agendamento. *Escrita com motivo obrigatorio - cancelamento sem motivo nao e' auditavel.*
func (a *agentAPI) schedulingCancel(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	var pedido struct {
		AppointmentID string `json:"appointment_id"`
		Reason        string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	agendamento, err := a.scheduling.Cancel(a.organizationID(c), strings.TrimSpace(pedido.AppointmentID),
		strings.TrimSpace(pedido.Reason))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"appointment": agendamento})
}

// schedulingRescheduleSlots devolve o que esta' livre para REMARCAR um agendamento especifico.
//
// *E' a peca que quase sempre falta na superficie de agendamento:* **remarcar sem saber o que esta' livre obriga o
// cliente a adivinhar**, e a adivinhacao vira horario inexistente - *que o motor recusa **depois**, quando o cliente ja'
// escolheu.* Aqui o motor responde **antes**.
func (a *agentAPI) schedulingRescheduleSlots(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	identificador := strings.TrimSpace(c.Query("appointment_id"))
	data := strings.TrimSpace(c.Query("date"))
	if identificador == "" || data == "" {
		writeAgentError(c, http.StatusBadRequest,
			errors.New("`appointment_id` e `date` (AAAA-MM-DD) são obrigatórios"))
		return
	}
	passo := 30
	if bruto := strings.TrimSpace(c.Query("step")); bruto != "" {
		convertido, err := parsePositiveInt(bruto)
		if err != nil || convertido <= 0 || convertido > 240 {
			writeAgentError(c, http.StatusBadRequest, errors.New("`step` precisa ser um inteiro entre 1 e 240 minutos"))
			return
		}
		passo = convertido
	}
	slots, err := a.scheduling.FreeSlotsForReschedule(a.organizationID(c), identificador, data, passo)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"slots": slots, "count": len(slots)})
}

// schedulingReschedule move um agendamento para outro instante. *A recusa do motor e' preservada: sobreposicao,
// fechamento e turno continuam sendo decididos por ele, nao por esta rota.*
func (a *agentAPI) schedulingReschedule(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	var pedido struct {
		AppointmentID string `json:"appointment_id"`
		StartsAt      string `json:"starts_at"`
	}
	if err := c.ShouldBindJSON(&pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	novoInicio, err := time.Parse(time.RFC3339, strings.TrimSpace(pedido.StartsAt))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, errors.New("`starts_at` precisa ser RFC 3339 (ex.: 2026-10-14T10:00:00Z)"))
		return
	}
	agendamento, err := a.scheduling.Reschedule(a.organizationID(c), strings.TrimSpace(pedido.AppointmentID), novoInicio)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"appointment": agendamento})
}

// E-457 — **A FILA DE ESPERA E O `NO-SHOW`, QUE FECHAVAM O LIMITE DE E-449.**
//
// **O QUE E-456 DEIXOU ABERTO**: *"fila de espera e `no-show` seguem sem rota"* ✗ — *e o motor tem `AddToWaitlist`,
// `OfferNextFromWaitlist`, `ConfirmWaitlistOffer` e `MarkNoShow`.* ***Capacidade sem superficie e' biblioteca - e a
// biblioteca ja' provou que existe.***
//
// **POR QUE O `E-456` NAO SE REPETE AQUI**: la' o contrato de entrada era uma struct **interna** sem `json` tags, e o
// corpo virava **zero silencioso** ✗. `SchedulingWaitlistEntry` **tem** tags ✓ e **ja' e' devolvido** ao cliente em
// `Waitlist` ⇒ *e' um **DTO publico**, e reusa-lo como entrada e' coerente* ✓. **A distincao importa**: o problema nunca
// foi "reusar struct" - foi reusar uma struct que **nao foi desenhada para ser lida de fora**.

// schedulingWaitlist lista a fila de espera da organizacao.
func (a *agentAPI) schedulingWaitlist(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	fila, err := a.scheduling.Waitlist(a.organizationID(c))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"waitlist": fila, "count": len(fila)})
}

// schedulingWaitlistAdd poe um contato na fila. *O escopo vem do middleware - o corpo nao escolhe o tenant.*
func (a *agentAPI) schedulingWaitlistAdd(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	var entrada agent.SchedulingWaitlistEntry
	if err := c.ShouldBindJSON(&entrada); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	entrada.OrganizationID = a.organizationID(c)
	// `ID`, `CreatedAt` e `Status` sao do MOTOR: *aceitar do corpo seria deixar o cliente escolher a propria identidade
	// e o proprio estado na fila.*
	entrada.ID = ""
	entrada.CreatedAt = time.Time{}
	entrada.Status = ""
	criada, err := a.scheduling.AddToWaitlist(entrada)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"entry": criada})
}

// E-479 — **OS LEMBRETES, A ÚLTIMA PEÇA DECLARADA EM FALTA NA SUPERFÍCIE.**
//
// **O LIMITE QUE E-457 DECLAROU**: *"`DueReminders`/`MarkReminderSent` seguem **sem rota**"* ✗ — *e o motor já os tinha,
// com uma regra que a superfície não podia deixar de preservar: **registrar o mesmo lembrete duas vezes é recusado*** ✓.
//
// **POR QUE IMPORTA**: um lembrete enviado duas vezes é um incómodo; um lembrete **não registado** é um cliente que
// **não recebe aviso**. ⇒ *a superfície tem de expor as DUAS metades: quem deve ser avisado, e o registo de que foi.*

// schedulingDueReminders devolve os agendamentos que precisam de lembrete AGORA.
//
// ⚠️ `now` **não** vem do corpo: *um corpo que escolhe «agora» escolhe quem deve ser avisado.*
func (a *agentAPI) schedulingDueReminders(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	tipo := strings.TrimSpace(c.Query("kind"))
	if tipo == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("`kind` é obrigatório (ex.: `24h`)"))
		return
	}
	antecedencia := 24 * 60
	if bruto := strings.TrimSpace(c.Query("lead_minutes")); bruto != "" {
		convertido, err := parsePositiveInt(bruto)
		if err != nil || convertido <= 0 || convertido > 100000 {
			writeAgentError(c, http.StatusBadRequest,
				errors.New("`lead_minutes` precisa ser um inteiro entre 1 e 100000"))
			return
		}
		antecedencia = convertido
	}
	devidos, err := a.scheduling.DueReminders(a.organizationID(c), tipo,
		time.Duration(antecedencia)*time.Minute, time.Now().UTC())
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"appointments": devidos, "count": len(devidos), "kind": tipo,
		"lead_minutes": antecedencia,
	})
}

// schedulingMarkReminderSent registra que o lembrete FOI enviado. *A recusa de duplicado é do MOTOR, e é preservada.*
func (a *agentAPI) schedulingMarkReminderSent(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	var pedido struct {
		AppointmentID string `json:"appointment_id"`
		Kind          string `json:"kind"`
	}
	if err := c.ShouldBindJSON(&pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if err := a.scheduling.MarkReminderSent(a.organizationID(c), strings.TrimSpace(pedido.AppointmentID),
		strings.TrimSpace(pedido.Kind), time.Now().UTC()); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"marked": true})
}

// schedulingWaitlistOffer oferece o proximo da fila para um horario que vagou.
//
// ⚠️ `now` NAO vem do corpo: *um corpo que escolhe "agora" escolhe o proprio prazo de expiracao da oferta.*
func (a *agentAPI) schedulingWaitlistOffer(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	var pedido struct {
		ResourceID string `json:"resource_id"`
		Slot       string `json:"slot"`
	}
	if err := c.ShouldBindJSON(&pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	horario, err := time.Parse(time.RFC3339, strings.TrimSpace(pedido.Slot))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, errors.New("`slot` precisa ser RFC 3339"))
		return
	}
	oferta, ofereceu, err := a.scheduling.OfferNextFromWaitlist(a.organizationID(c),
		strings.TrimSpace(pedido.ResourceID), horario, time.Now().UTC())
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if !ofereceu {
		// *Fila vazia nao e' erro: e' ausencia de oferta - e o status tem de dizer isso, nao 500.*
		c.JSON(http.StatusOK, gin.H{"offered": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"offered": true, "entry": oferta})
}

// schedulingWaitlistConfirm confirma a oferta, criando o agendamento. *A oferta EXPIRA: o motor decide, nao a rota.*
func (a *agentAPI) schedulingWaitlistConfirm(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	var pedido struct {
		EntryID string `json:"entry_id"`
	}
	if err := c.ShouldBindJSON(&pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	agendamento, err := a.scheduling.ConfirmWaitlistOffer(a.organizationID(c),
		strings.TrimSpace(pedido.EntryID), time.Now().UTC())
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"appointment": agendamento})
}

// schedulingNoShow marca o no-show. *Quem faltou nao e' o mesmo que quem cancelou, e a agenda precisa distinguir.*
func (a *agentAPI) schedulingNoShow(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	var pedido struct {
		AppointmentID string `json:"appointment_id"`
	}
	if err := c.ShouldBindJSON(&pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	agendamento, err := a.scheduling.MarkNoShow(a.organizationID(c), strings.TrimSpace(pedido.AppointmentID),
		time.Now().UTC())
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"appointment": agendamento})
}

// parsePositiveInt converte sem `strconv.Atoi` cru: *`Atoi` aceita sinal, e "-5" passaria por "inteiro".*
func parsePositiveInt(bruto string) (int, error) {
	valor := 0
	for _, caractere := range bruto {
		if caractere < '0' || caractere > '9' {
			return 0, errors.New("não é um inteiro positivo")
		}
		valor = valor*10 + int(caractere-'0')
		if valor > 1_000_000 {
			return 0, errors.New("valor grande demais")
		}
	}
	if bruto == "" {
		return 0, errors.New("vazio")
	}
	return valor, nil
}
