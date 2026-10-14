package server

import (
	"errors"
	"net/http"
	"strings"

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

// schedulingBook cria um agendamento. *Escrita: passa pelo mesmo motor que os testes exercitam (`Book`), com as suas
// validações de sobreposição, turno e organização.*
func (a *agentAPI) schedulingBook(c *gin.Context) {
	if a.scheduling == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("motor de agendamento não está ligado neste runtime"))
		return
	}
	var pedido agent.BookRequest
	if err := c.ShouldBindJSON(&pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	// O escopo vem do MIDDLEWARE, não do corpo: *um corpo que escolhe a organização é um corpo que escolhe o tenant.*
	pedido.OrganizationID = a.organizationID(c)
	agendamento, err := a.scheduling.Book(pedido)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"appointment": agendamento})
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
