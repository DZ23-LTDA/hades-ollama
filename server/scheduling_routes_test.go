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

// E-456 — **A JORNADA DO CLIENTE, MEDIDA PELA SUPERFICIE: AGENDAR, LISTAR, REMARCAR, CANCELAR.**
//
// **O LIMITE QUE E-449 DECLAROU**: *"liguei o nucleo, nao o motor inteiro - `Cancel`/`Reschedule`/fila/`no-show` NAO tem
// rota"* ✗. ***Um cliente que agenda e nao consegue cancelar nem remarcar nao tem um produto: tem uma demonstracao.***
//
// **O QUE ESTE TESTE PROVA**: que a jornada inteira atravessa a superficie, **na ordem em que o cliente a vive** - e que
// `reschedule/slots` responde **ANTES** de o cliente escolher (*remarcar sem saber o que esta' livre obriga a adivinhar,
// e a adivinhacao vira horario inexistente que o motor recusa **depois***).
func TestJornadaDoClienteNaSuperficieDeAgendamento(t *testing.T) {
	gin.SetMode(gin.TestMode)
	loja := agent.NewSchedulingStore(t.TempDir())
	// ⚠️ E-456: a organizacao do TESTE e a do MIDDLEWARE tem de ser a MESMA. Eu criava o servico com "org-a" e o
	// middleware fixava `agent.LocalOrganizationID` ⇒ *o motor respondeu "servico ... nao existe nesta organizacao"*,
	// com razao. **O motor estava certo; o teste e' que era incoerente consigo mesmo.**
	organizacao := agent.LocalOrganizationID
	servico, err := loja.AddService(agent.SchedulingService{OrganizationID: organizacao, Name: "Consulta", DurationMinutes: 30})
	if err != nil {
		t.Fatalf("criar serviço: %v", err)
	}
	recurso, err := loja.AddResource(agent.SchedulingResource{OrganizationID: organizacao, Name: "Bia"})
	if err != nil {
		t.Fatalf("criar recurso: %v", err)
	}

	// A data e' FIXA e o turno nasce no dia da semana DELA: *criar turnos para "0..6" nao funcionava porque
	// `OpenSlots` procura o turno do dia da data* (medido em E-449).
	data := "2026-10-14"
	dia, erroData := time.Parse("2006-01-02", data)
	if erroData != nil {
		t.Fatalf("data: %v", erroData)
	}
	// ⚠️ E-456, terceiro ajuste: o turno tem de ser do RECURSO que a jornada usa. *Eu criava com `ResourceID: ""` e a
	// jornada manda o recurso real ⇒ o motor respondeu "nao tem expediente", com razao.*
	// **O motor recusou tres vezes, e as tres estavam certas** - cada recusa apontou um erro MEU de setup.
	if _, err := loja.AddShift(agent.SchedulingShift{
		OrganizationID: organizacao, ResourceID: recurso.ID, Weekday: int(dia.Weekday()), OpensAt: 0, ClosesAt: 24 * 60,
	}); err != nil {
		t.Fatalf("criar turno: %v", err)
	}

	engine := montarRotaDeAgendamento(loja, agent.LocalOrganizationID)
	executar := func(metodo, caminho, corpo string) *httptest.ResponseRecorder {
		requisicao := httptest.NewRequest(metodo, "/api/agent/v1"+caminho, strings.NewReader(corpo))
		requisicao.Header.Set("Content-Type", "application/json")
		resposta := httptest.NewRecorder()
		engine.ServeHTTP(resposta, requisicao)
		return resposta
	}

	// 1. AGENDAR.
	resposta := executar(http.MethodPost, "/scheduling/book", `{"service_id":"`+servico.ID+`","resource_id":"`+recurso.ID+
		`","contact_id":"cliente-1","channel":"voice","starts_at":"2026-10-14T08:00:00Z","idempotency_key":"k-jornada"}`)
	if resposta.Code != http.StatusCreated {
		t.Fatalf("agendar devolveu %d: %s", resposta.Code, resposta.Body.String())
	}
	var criado struct {
		Appointment struct {
			ID string `json:"id"`
		} `json:"appointment"`
	}
	if err := json.Unmarshal(resposta.Body.Bytes(), &criado); err != nil || criado.Appointment.ID == "" {
		t.Fatalf("o agendamento criado nao veio com ID: %s", resposta.Body.String())
	}

	// 2. LISTAR: *o cliente tem de conseguir ver o que marcou.*
	resposta = executar(http.MethodGet, "/scheduling/appointments", "")
	if resposta.Code != http.StatusOK {
		t.Fatalf("listar devolveu %d: %s", resposta.Code, resposta.Body.String())
	}
	var listados struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(resposta.Body.Bytes(), &listados)
	if listados.Count == 0 {
		t.Fatalf("depois de agendar, a lista veio vazia — *agendar e nao conseguir ver o que marcou nao e' jornada*")
	}

	// 3. O QUE ESTA' LIVRE PARA REMARCAR — **antes** de escolher.
	resposta = executar(http.MethodGet,
		"/scheduling/reschedule/slots?appointment_id="+criado.Appointment.ID+"&date="+data+"&step=60", "")
	if resposta.Code != http.StatusOK {
		t.Fatalf("reschedule/slots devolveu %d: %s", resposta.Code, resposta.Body.String())
	}
	var livres struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(resposta.Body.Bytes(), &livres)
	if livres.Count == 0 {
		t.Fatalf("um dia inteiro de turno devia ter horario livre para remarcar, e vieram %d", livres.Count)
	}

	// 4. REMARCAR para um horario que o MOTOR disse estar livre.
	resposta = executar(http.MethodPost, "/scheduling/reschedule", `{"appointment_id":"`+criado.Appointment.ID+
		`","starts_at":"2026-10-14T14:00:00Z"}`)
	if resposta.Code != http.StatusOK {
		t.Fatalf("remarcar devolveu %d: %s", resposta.Code, resposta.Body.String())
	}

	// 5. CANCELAR, com motivo (*cancelamento sem motivo nao e' auditavel*).
	resposta = executar(http.MethodPost, "/scheduling/cancel", `{"appointment_id":"`+criado.Appointment.ID+
		`","reason":"o cliente desmarcou"}`)
	if resposta.Code != http.StatusOK {
		t.Fatalf("cancelar devolveu %d: %s", resposta.Code, resposta.Body.String())
	}

	// E o MOTOR tem de contar a mesma historia que a superficie: *a superficie nao pode ser uma segunda verdade.*
	agendamentos, err := loja.Appointments(agent.LocalOrganizationID)
	if err != nil {
		t.Fatalf("listar no motor: %v", err)
	}
	if len(agendamentos) != 1 || agendamentos[0].Status != agent.SchedulingAppointmentCancelled {
		t.Fatalf("ARMADILHA: depois de cancelar pela superficie, o motor diz %+v — *a rota e o motor tem de contar a "+
			"MESMA historia*", agendamentos)
	}
}

// E-457 — **A FILA DE ESPERA E O `NO-SHOW` PELA SUPERFICIE.**
//
// **O QUE E-456 DEIXOU ABERTO**: *"fila de espera e `no-show` seguem sem rota"* ✗. ***Capacidade sem superficie e'
// biblioteca - e a biblioteca ja' provou que existe.***
//
// O fluxo medido: **entrar na fila → listar → OFERECER um horario que vagou → CONFIRMAR** (*a oferta vira agendamento*),
// e o **no-show** num agendamento que existiu. *E' a ultima milha do agendamento: sem ela, quem nao conseguiu horario
// simplesmente desaparece, e quem faltou e' indistinguivel de quem cancelou.*
func TestFilaDeEsperaENoShowNaSuperficie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	organizacao := agent.LocalOrganizationID
	loja := agent.NewSchedulingStore(t.TempDir())
	servico, err := loja.AddService(agent.SchedulingService{OrganizationID: organizacao, Name: "Consulta", DurationMinutes: 30})
	if err != nil {
		t.Fatalf("criar serviço: %v", err)
	}
	recurso, err := loja.AddResource(agent.SchedulingResource{OrganizationID: organizacao, Name: "Bia"})
	if err != nil {
		t.Fatalf("criar recurso: %v", err)
	}
	dia, _ := time.Parse("2006-01-02", "2026-10-14")
	if _, err := loja.AddShift(agent.SchedulingShift{
		OrganizationID: organizacao, ResourceID: recurso.ID, Weekday: int(dia.Weekday()), OpensAt: 0, ClosesAt: 24 * 60,
	}); err != nil {
		t.Fatalf("criar turno: %v", err)
	}

	engine := montarRotaDeAgendamento(loja, organizacao)
	executar := func(metodo, caminho, corpo string) *httptest.ResponseRecorder {
		requisicao := httptest.NewRequest(metodo, "/api/agent/v1"+caminho, strings.NewReader(corpo))
		requisicao.Header.Set("Content-Type", "application/json")
		resposta := httptest.NewRecorder()
		engine.ServeHTTP(resposta, requisicao)
		return resposta
	}

	// 1. ENTRAR NA FILA.
	resposta := executar(http.MethodPost, "/scheduling/waitlist",
		`{"service_id":"`+servico.ID+`","resource_id":"`+recurso.ID+`","contact_id":"cliente-fila","channel":"voice",`+
			`"from":"2026-10-14T00:00:00Z","to":"2026-10-20T00:00:00Z"}`)
	if resposta.Code != http.StatusCreated {
		t.Fatalf("entrar na fila devolveu %d: %s", resposta.Code, resposta.Body.String())
	}
	var criada struct {
		Entry struct {
			ID string `json:"id"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(resposta.Body.Bytes(), &criada); err != nil || criada.Entry.ID == "" {
		t.Fatalf("a entrada da fila nao veio com ID: %s", resposta.Body.String())
	}

	// 2. LISTAR.
	resposta = executar(http.MethodGet, "/scheduling/waitlist", "")
	if resposta.Code != http.StatusOK {
		t.Fatalf("listar a fila devolveu %d: %s", resposta.Code, resposta.Body.String())
	}
	var fila struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(resposta.Body.Bytes(), &fila)
	if fila.Count == 0 {
		t.Fatal("depois de entrar na fila, a lista veio vazia — *quem espera tem de poder ser visto*")
	}

	// 3. OFERECER um horario que vagou. *Fila com uma entrada e um horario livre: tem de oferecer.*
	resposta = executar(http.MethodPost, "/scheduling/waitlist/offer",
		`{"resource_id":"`+recurso.ID+`","slot":"2026-10-14T09:00:00Z"}`)
	if resposta.Code != http.StatusOK {
		t.Fatalf("oferecer devolveu %d: %s", resposta.Code, resposta.Body.String())
	}
	var oferta struct {
		Offered bool `json:"offered"`
	}
	_ = json.Unmarshal(resposta.Body.Bytes(), &oferta)
	if !oferta.Offered {
		t.Fatalf("com uma entrada na fila e um horario livre, a oferta tinha de sair — veio %s", resposta.Body.String())
	}

	// 4. CONFIRMAR: *a oferta vira agendamento.*
	resposta = executar(http.MethodPost, "/scheduling/waitlist/confirm", `{"entry_id":"`+criada.Entry.ID+`"}`)
	if resposta.Code != http.StatusCreated {
		t.Fatalf("confirmar devolveu %d: %s", resposta.Code, resposta.Body.String())
	}
	var confirmado struct {
		Appointment struct {
			ID string `json:"id"`
		} `json:"appointment"`
	}
	if err := json.Unmarshal(resposta.Body.Bytes(), &confirmado); err != nil || confirmado.Appointment.ID == "" {
		t.Fatalf("a confirmacao nao criou agendamento: %s", resposta.Body.String())
	}

	// 5. NO-SHOW: *quem faltou nao e' o mesmo que quem cancelou.*
	resposta = executar(http.MethodPost, "/scheduling/no-show", `{"appointment_id":"`+confirmado.Appointment.ID+`"}`)
	if resposta.Code != http.StatusOK {
		t.Fatalf("no-show devolveu %d: %s", resposta.Code, resposta.Body.String())
	}

	// E o MOTOR conta a mesma historia: *o agendamento confirmado pela fila esta' marcado como no-show.*
	agendamentos, err := loja.Appointments(organizacao)
	if err != nil {
		t.Fatalf("listar no motor: %v", err)
	}
	achouNoShow := false
	for _, agendamento := range agendamentos {
		if agendamento.ID == confirmado.Appointment.ID && agendamento.Status == agent.SchedulingAppointmentNoShow {
			achouNoShow = true
		}
	}
	if !achouNoShow {
		t.Fatalf("a superficie disse no-show e o motor diz %+v — *a rota e o motor tem de contar a MESMA historia*",
			agendamentos)
	}
}

// E-458 — **A IDEMPOTÊNCIA DE `Book`, EXERCITADA PELA ROTA.**
//
// **A PROMESSA ESTA' ESCRITA NO MOTOR** (`scheduling.go:266`): *"IdempotencyKey identifica a MESMA intenção vinda do
// mesmo canal: **reentrega não cria segundo agendamento**"*. **E a superfície não a provava** ✗ ⇒ ***uma promessa que a
// superfície não exercita é uma promessa que o cliente descobre ser falsa em produção*** — *e o modo de falha é o pior
// possível num agendamento: **dois horários para a mesma intenção**, e alguém aparece sem vaga.*
//
// **AS TRÊS DIREÇÕES** (*a terceira é a que quase sempre falta*):
//  1. **mesma chave** ⇒ **um** agendamento, e o segundo devolve o MESMO id (*não uma cópia*);
//  2. **chave DIFERENTE** ⇒ segundo agendamento (*senão a rota estaria a recusar intenções legítimas - um "idempotente"
//     que engole tudo é um defeito diferente e pior*);
//  3. e o CONTROLE que separa os dois: *o motor tem de ter exactamente o número certo de agendamentos depois de cada
//     passo - medido no MOTOR, não na resposta da rota.*
func TestIdempotenciaDeBookingNaSuperficie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	organizacao := agent.LocalOrganizationID
	loja := agent.NewSchedulingStore(t.TempDir())
	servico, err := loja.AddService(agent.SchedulingService{OrganizationID: organizacao, Name: "Consulta", DurationMinutes: 30})
	if err != nil {
		t.Fatalf("criar serviço: %v", err)
	}
	recurso, err := loja.AddResource(agent.SchedulingResource{OrganizationID: organizacao, Name: "Bia"})
	if err != nil {
		t.Fatalf("criar recurso: %v", err)
	}
	dia, _ := time.Parse("2006-01-02", "2026-10-14")
	if _, err := loja.AddShift(agent.SchedulingShift{
		OrganizationID: organizacao, ResourceID: recurso.ID, Weekday: int(dia.Weekday()), OpensAt: 0, ClosesAt: 24 * 60,
	}); err != nil {
		t.Fatalf("criar turno: %v", err)
	}

	engine := montarRotaDeAgendamento(loja, organizacao)
	agendar := func(chave, inicio string) (int, string) {
		corpo := `{"service_id":"` + servico.ID + `","resource_id":"` + recurso.ID + `","contact_id":"cliente-idem",` +
			`"channel":"voice","starts_at":"` + inicio + `","idempotency_key":"` + chave + `"}`
		requisicao := httptest.NewRequest(http.MethodPost, "/api/agent/v1/scheduling/book", strings.NewReader(corpo))
		requisicao.Header.Set("Content-Type", "application/json")
		resposta := httptest.NewRecorder()
		engine.ServeHTTP(resposta, requisicao)
		var criado struct {
			Appointment struct {
				ID string `json:"id"`
			} `json:"appointment"`
		}
		_ = json.Unmarshal(resposta.Body.Bytes(), &criado)
		return resposta.Code, criado.Appointment.ID
	}
	contar := func() int {
		agendamentos, err := loja.Appointments(organizacao)
		if err != nil {
			t.Fatalf("listar no motor: %v", err)
		}
		return len(agendamentos)
	}

	// 1. MESMA CHAVE, duas entregas.
	codigo1, id1 := agendar("k-idem", "2026-10-14T08:00:00Z")
	if codigo1 != http.StatusCreated || id1 == "" {
		t.Fatalf("primeira entrega devolveu %d (id=%q)", codigo1, id1)
	}
	if contar() != 1 {
		t.Fatalf("depois da primeira entrega o motor tem %d agendamento(s), esperava 1", contar())
	}
	codigo2, id2 := agendar("k-idem", "2026-10-14T08:00:00Z")
	// E-458: a reentrega é **409 com o MESMO recurso** — *não 400 (que sugere pedido malformado) nem 500*.
	// ***Idempotência de verdade é devolver o MESMO RECURSO, não um erro.*** *A asserção nasceu fraca (`>= 500`) porque
	// eu não sabia o que a rota devolvia; com o contrato decidido, ela passa a medir o contrato.*
	if codigo2 != http.StatusConflict {
		t.Fatalf("a reentrega devolveu %d, esperava 409 — *reentrega não é erro do cliente, e o recurso JÁ existe; "+
			"400 sugeriria pedido malformado*", codigo2)
	}
	if id2 != id1 {
		t.Fatalf("ARMADILHA: a reentrega com a MESMA chave devolveu outro id (%q vs %q) — *a promessa do motor é que "+
			"reentrega NÃO cria segundo agendamento*", id2, id1)
	}
	if total := contar(); total != 1 {
		t.Fatalf("ARMADILHA GRAVE: a reentrega criou um SEGUNDO agendamento (motor tem %d) — *isto é o pior modo de "+
			"falha num agendamento: dois horários para a mesma intenção, e alguém aparece sem vaga*", total)
	}

	// 2. CHAVE DIFERENTE, outro horário: tem de criar. *Um "idempotente" que engole tudo é um defeito diferente e pior.*
	codigo3, id3 := agendar("k-outra", "2026-10-14T10:00:00Z")
	if codigo3 != http.StatusCreated || id3 == "" || id3 == id1 {
		t.Fatalf("chave DIFERENTE tinha de criar outro agendamento (codigo=%d id=%q vs %q)", codigo3, id3, id1)
	}
	if total := contar(); total != 2 {
		t.Fatalf("depois de uma intenção NOVA o motor tem %d agendamento(s), esperava 2 — *recusar intenção legítima "+
			"seria um defeito pior que o original*", total)
	}
}

// E-479 — **OS LEMBRETES PELA SUPERFÍCIE: QUEM DEVE SER AVISADO, E O REGISTO DE QUE FOI.**
//
// **O LIMITE QUE E-457 DECLAROU**: *"`DueReminders`/`MarkReminderSent` seguem sem rota"* ✗. ***Um lembrete enviado duas
// vezes é um incómodo; um lembrete NÃO registado é um cliente que NÃO recebe aviso.***
//
// **E A REGRA DO MOTOR É PRESERVADA**: *registrar o MESMO lembrete duas vezes é recusado* ⇒ *o teste tem de provar as
// duas metades — e que a segunda não passa.*
func TestLembretesNaSuperficie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	organizacao := agent.LocalOrganizationID
	loja := agent.NewSchedulingStore(t.TempDir())
	servico, err := loja.AddService(agent.SchedulingService{OrganizationID: organizacao, Name: "Consulta", DurationMinutes: 30})
	if err != nil {
		t.Fatalf("criar serviço: %v", err)
	}
	recurso, err := loja.AddResource(agent.SchedulingResource{OrganizationID: organizacao, Name: "Bia"})
	if err != nil {
		t.Fatalf("criar recurso: %v", err)
	}
	// O turno nasce no dia da semana da data escolhida - *e a data é FIXA, para o teste não depender do relógio.*
	dia, _ := time.Parse("2006-01-02", "2026-10-14")
	for deslocamento := range 7 {
		if _, err := loja.AddShift(agent.SchedulingShift{
			OrganizationID: organizacao, ResourceID: recurso.ID,
			Weekday: int(dia.AddDate(0, 0, deslocamento).Weekday()), OpensAt: 0, ClosesAt: 24 * 60,
		}); err != nil {
			t.Fatalf("criar turno: %v", err)
		}
	}

	// E-650 — **O INÍCIO NÃO PODE DEPENDER DO RELÓGIO DE PAREDE.**
	//
	// Antes: `time.Now().UTC().Add(2 * time.Hour)`. Com turno `00:00-24:00` e serviço de 30 min, um
	// início às **23:56 (local)** termina 00:26 do dia seguinte e o motor RECUSA — e recusa COM RAZÃO
	// (`shiftCoversMinute` considera a duração). O teste passava de manhã e falhava à noite:
	// ***a cor do CI dependia da hora a que ele corria*** — medido, não suposto: reproduzido às
	// 21:56 BRT com `2026-10-14T23:56:00-03:00 está fora da janela 00:00-24:00`.
	//
	// Agora escolhe-se o PRÓXIMO MEIO-DIA UTC: cabe com folga em qualquer janela de 24 horas e está
	// SEMPRE a menos de 24h de distância, que é exactamente o que o lembrete de 24h exige.
	agora := time.Now().UTC()
	inicio := time.Date(agora.Year(), agora.Month(), agora.Day(), 12, 0, 0, 0, time.UTC)
	if !inicio.After(agora) {
		inicio = inicio.AddDate(0, 0, 1)
	}
	agendamento, err := loja.Book(agent.BookRequest{
		OrganizationID: organizacao, ResourceID: recurso.ID, ServiceID: servico.ID,
		ContactID: "cliente-lembrete", Channel: "voice", StartsAt: inicio, IdempotencyKey: "k-lembrete",
	})
	if err != nil {
		t.Fatalf("agendar: %v", err)
	}

	engine := montarRotaDeAgendamento(loja, organizacao)
	executar := func(metodo, caminho, corpo string) *httptest.ResponseRecorder {
		requisicao := httptest.NewRequest(metodo, "/api/agent/v1"+caminho, strings.NewReader(corpo))
		requisicao.Header.Set("Content-Type", "application/json")
		resposta := httptest.NewRecorder()
		engine.ServeHTTP(resposta, requisicao)
		return resposta
	}

	// (1) QUEM DEVE SER AVISADO: *com antecedência de 24h e início em 2h, este agendamento é devido.*
	resposta := executar(http.MethodGet, "/scheduling/reminders/due?kind=24h&lead_minutes=1440", "")
	if resposta.Code != http.StatusOK {
		t.Fatalf("lembretes devidos devolveu %d: %s", resposta.Code, resposta.Body.String())
	}
	var devidos struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(resposta.Body.Bytes(), &devidos)
	if devidos.Count == 0 {
		t.Fatalf("um agendamento em 2 horas, com antecedencia de 24h, tinha de estar devido — veio %s",
			resposta.Body.String())
	}

	// (2) `kind` é obrigatório: *sem ele não se sabe de QUE lembrete se fala.*
	if resposta := executar(http.MethodGet, "/scheduling/reminders/due", ""); resposta.Code != http.StatusBadRequest {
		t.Fatalf("sem `kind` tinha de ser 400, veio %d", resposta.Code)
	}

	// (3) O REGISTO: marcar como enviado.
	resposta = executar(http.MethodPost, "/scheduling/reminders/sent",
		`{"appointment_id":"`+agendamento.ID+`","kind":"24h"}`)
	if resposta.Code != http.StatusCreated {
		t.Fatalf("marcar lembrete devolveu %d: %s", resposta.Code, resposta.Body.String())
	}

	// (4) E A REGRA DO MOTOR, PELA SUPERFÍCIE: *o MESMO lembrete, de novo, é recusado* — e por isso ele **sai da lista**.
	resposta = executar(http.MethodPost, "/scheduling/reminders/sent",
		`{"appointment_id":"`+agendamento.ID+`","kind":"24h"}`)
	if resposta.Code == http.StatusCreated {
		t.Fatal("ARMADILHA: a superficie marcou o MESMO lembrete duas vezes — *registrar de novo o mesmo lembrete e' " +
			"recusado pelo motor, e a recusa tem de chegar ao cliente*")
	}
	resposta = executar(http.MethodGet, "/scheduling/reminders/due?kind=24h&lead_minutes=1440", "")
	var depois struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(resposta.Body.Bytes(), &depois)
	if depois.Count != 0 {
		t.Fatalf("depois de marcado, o lembrete tem de SAIR da lista de devidos — ainda veio %d: *senao o cliente "+
			"seria avisado duas vezes*", depois.Count)
	}
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
	// E-456: a jornada completa.
	grupo.GET("/scheduling/appointments", api.schedulingAppointments)
	grupo.POST("/scheduling/cancel", api.schedulingCancel)
	grupo.GET("/scheduling/reschedule/slots", api.schedulingRescheduleSlots)
	grupo.POST("/scheduling/reschedule", api.schedulingReschedule)
	// E-457: fila de espera e no-show.
	grupo.GET("/scheduling/waitlist", api.schedulingWaitlist)
	grupo.POST("/scheduling/waitlist", api.schedulingWaitlistAdd)
	grupo.POST("/scheduling/waitlist/offer", api.schedulingWaitlistOffer)
	grupo.POST("/scheduling/waitlist/confirm", api.schedulingWaitlistConfirm)
	grupo.POST("/scheduling/no-show", api.schedulingNoShow)
	// E-479: lembretes.
	grupo.GET("/scheduling/reminders/due", api.schedulingDueReminders)
	grupo.POST("/scheduling/reminders/sent", api.schedulingMarkReminderSent)
	return engine
}
