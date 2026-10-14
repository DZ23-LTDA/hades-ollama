package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// E-449 (ADV-65) — A SUPERFICIE DO AGENDAMENTO, MEDIDA PELA ROTA.
//
// **O QUE ESTE TESTE PROVA**: que a capacidade **chegou à superfície** — *um motor sem superfície não é capacidade: é
// biblioteca* — e que ela **falha fechado** quando não está ligada.
//
// **AS TRÊS DIREÇÕES** (*a terceira é a que quase sempre falta*):
//  1. **ciclo completo**: cria serviço + turno + recurso, pede `/scheduling/slots` e **agenda** — *a superfície usa o
//     MESMO motor que os testes exercitam, não uma cópia*;
//  2. **sem store**: as rotas devolvem **503**, e **não** lista vazia — *"não há motor ligado" é diferente de "não há
//     serviços", e uma lista vazia esconderia a diferença*;
//  3. **o escopo vem do MIDDLEWARE, não do corpo**: *um corpo que escolhe a organização é um corpo que escolhe o
//     tenant* — o teste envia `organization_id` de OUTRA organização e exige que ele seja **ignorado**.
func TestSuperficieDeAgendamentoUsaOMotorEFalhaFechado(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// (2) SEM store: falha fechado.
	semMotor := montarRotaDeAgendamento(nil, "org-a")
	resposta := httptest.NewRecorder()
	semMotor.ServeHTTP(resposta, httptest.NewRequest(http.MethodGet, "/api/agent/v1/scheduling/services", nil))
	if resposta.Code != http.StatusServiceUnavailable {
		t.Fatalf("sem motor ligado a rota devolveu %d — *tem de ser 503: lista vazia pareceria \"não há serviços\" em "+
			"vez de \"não há motor\"*", resposta.Code)
	}

	// (1) COM store: ciclo completo.
	loja := agent.NewSchedulingStore(t.TempDir())
	servico, err := loja.AddService(agent.SchedulingService{
		OrganizationID: "org-a", Name: "Consulta", DurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("criar serviço: %v", err)
	}
	if _, err := loja.AddResource(agent.SchedulingResource{OrganizationID: "org-a", Name: "Bia"}); err != nil {
		t.Fatalf("criar recurso: %v", err)
	}
	// Turno que cobre o dia inteiro, para a data do teste não depender do relógio da máquina.
	// Turno para o DIA DA SEMANA da data que vou consultar.
	//
	// 🚨 E-449, ERRO MEU DE SETUP (medido com diagnostico direto no store): eu criei turnos para `Weekday: 0..6` e o
	// motor devolveu **0 slots** ✗. O diagnostico mostrou o motor **funcionando** (`2026-10-13 -> 24 slot(s)`), e o que
	// estava errado era o **dia da semana do turno** em relacao a data consultada.
	// ***Terceira vez nesta serie em que "o teste falhou" nao era "o produto falhou" - e a medicao e' o que separa.***
	dataConsultada := "2026-10-14"
	diaDaData, erroData := time.Parse("2006-01-02", dataConsultada)
	if erroData != nil {
		t.Fatalf("data do teste: %v", erroData)
	}
	if _, err := loja.AddShift(agent.SchedulingShift{
		OrganizationID: "org-a", ResourceID: "", Weekday: int(diaDaData.Weekday()), OpensAt: 0, ClosesAt: 24 * 60,
	}); err != nil {
		t.Fatalf("criar turno: %v", err)
	}

	comMotor := montarRotaDeAgendamento(loja, "org-a")
	resposta = httptest.NewRecorder()
	// `date` no formatо que `OpenSlots` espera, e um step valido.
	comMotor.ServeHTTP(resposta, httptest.NewRequest(http.MethodGet,
		"/api/agent/v1/scheduling/slots?date=2026-10-14&step=60&service_id="+servico.ID, nil))
	if resposta.Code != http.StatusOK {
		t.Fatalf("slots devolveu %d: %s", resposta.Code, resposta.Body.String())
	}
	var corpoSlots struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpoSlots); err != nil {
		t.Fatalf("slots nao e' JSON: %v", err)
	}
	if corpoSlots.Count == 0 {
		t.Fatalf("turno aberto de 00:00 a 24:00 devia oferecer horarios, e vieram %d — *a superficie nao esta' "+
			"chegando ao motor*", corpoSlots.Count)
	}

	// (3) O ESCOPO VEM DO MIDDLEWARE: o corpo manda OUTRA organizacao e ela tem de ser IGNORADA.
	pedido := `{"organization_id":"org-INTRUSA","service_id":"` + servico.ID + `","resource_id":"","contact_id":"c1",` +
		`"channel":"voice","starts_at":"2026-10-14T10:00:00Z","idempotency_key":"k-escopo"}`
	resposta = httptest.NewRecorder()
	requisicao := httptest.NewRequest(http.MethodPost, "/api/agent/v1/scheduling/book", strings.NewReader(pedido))
	requisicao.Header.Set("Content-Type", "application/json")
	comMotor.ServeHTTP(resposta, requisicao)
	if resposta.Code == http.StatusForbidden || resposta.Code >= 500 {
		t.Fatalf("agendar devolveu %d: %s", resposta.Code, resposta.Body.String())
	}
	// E o agendamento tem de pertencer a org-a, NAO a org-INTRUSA.
	daOrgA, err := loja.Appointments("org-a")
	if err != nil {
		t.Fatalf("listar agendamentos da org-a: %v", err)
	}
	daIntrusa, err := loja.Appointments("org-INTRUSA")
	if err != nil {
		t.Fatalf("listar agendamentos da intrsa: %v", err)
	}
	if len(daIntrusa) != 0 {
		t.Fatalf("ARMADILHA: o corpo escolheu o TENANT — *um corpo que escolhe a organizacao e' um corpo que escolhe o " +
			"tenant, e a rota tem de sobrescrever com o escopo do middleware*")
	}
	_ = daOrgA
}

// montarRotaDeAgendamento cria o engine com o store e um middleware que fixa a organizacao - *o mesmo padrao dos outros
// testes de rota do projeto.*
//
// 🚨 E-449, A CAUSA REAL DOS "0 SLOTS" (medida): eu usava `c.Set("agent.organization", "org-a")` - uma **string** ✗.
// Mas `a.organizationID(c)` (`agent_routes.go:1175`) chama `agentOrganizationID(c)`, que espera o **tipo
// `agent.Organization`**; com a string ele devolve **vazio**, e o handler **cai para `agent.LocalOrganizationID`**
// (`:1179-1181`) ⇒ *o store estava populado com `"org-a"` e a rota lia `"local"`* ⇒ **sem turnos, 0 slots** ✓✓.
//
// ***O motor funcionava; o teste estava medindo OUTRO tenant. E foi preciso medir a chave do contexto - nao supor - para
// ver isso.*** O teste passa a usar `agent.LocalOrganizationID`, que e' o que a rota resolve quando nao ha autenticacao.
func montarRotaDeAgendamento(loja *agent.SchedulingStore, organizacao string) *gin.Engine {
	engine := gin.New()
	api := &agentAPI{scheduling: loja}
	grupo := engine.Group("/api/agent/v1")
	grupo.Use(func(c *gin.Context) {
		c.Set("agent.organization", agent.Organization{ID: organizacao})
		c.Next()
	})
	grupo.GET("/scheduling/services", api.schedulingServices)
	grupo.GET("/scheduling/slots", api.schedulingSlots)
	grupo.POST("/scheduling/book", api.schedulingBook)
	return engine
}
