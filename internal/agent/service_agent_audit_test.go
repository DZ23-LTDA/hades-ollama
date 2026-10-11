package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Trilha DURÁVEL de execução de ferramentas (estágio 35). O que estes testes travam:
//
//  1. a trilha SOBREVIVE AO REINÍCIO — auditoria que desaparece não é auditoria;
//  2. a TENTATIVA também é evidência: bloqueado, indisponível e falho entram no registro;
//  3. o TETO é POR ORGANIZAÇÃO: o volume de um tenant não expulsa a evidência de outro;
//  4. ISOLAMENTO no disco, conferido lendo o ARQUIVO;
//  5. falha de gravação é REPORTADA, e o registro não é perdido em silêncio;
//  6. o TEXTO DO CONTATO não entra na trilha.
func auditFixture(t *testing.T) (*ServiceAgentRuntime, *ServiceAgentAuditStore, string, time.Time) {
	t.Helper()
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	store := NewServiceAgentAuditStore(serviceAgentAuditRoot(t.TempDir()))
	runtime.SetAuditStore(store)
	return runtime, store, conversationID, time.Now().UTC()
}

func TestServiceAgentAuditSurvivesRestart(t *testing.T) {
	runtime, store, conversationID, now := auditFixture(t)

	// Um turno com uma ferramenta EXECUTADA e uma BLOQUEADA (não declarada pelo recepcionista).
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi", ModelID: "modelo-local",
		RequestedTools: []string{"crm.contact.read", "inbox.assign"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if turn.AuditError != "" {
		t.Fatalf("a trilha precisa gravar sem erro: %s", turn.AuditError)
	}

	// REINÍCIO: runtime novo lendo o MESMO diretório.
	depois, _, _, _, _ := serviceAgentRuntime(t, true)
	depois.SetAuditStore(NewServiceAgentAuditStore(store.Root()))

	trail, err := depois.AuditTrail("org-a", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(trail) != 2 {
		t.Fatalf("a trilha precisa sobreviver ao reinício com as 2 tentativas: %d", len(trail))
	}
	// A TENTATIVA BLOQUEADA também é evidência de auditoria.
	var blocked, executed *ServiceAgentAuditEntry
	for index := range trail {
		switch trail[index].ToolName {
		case "inbox.assign":
			blocked = &trail[index]
		case "crm.contact.read":
			executed = &trail[index]
		}
	}
	if blocked == nil || blocked.Status != ServiceAgentToolBlocked || blocked.Reason == "" {
		t.Fatalf("a tentativa bloqueada precisa constar com motivo: %+v", blocked)
	}
	if executed == nil || executed.Status != ServiceAgentToolExecuted {
		t.Fatalf("a execução precisa constar: %+v", executed)
	}
	if executed.AgentID != ServiceAgentReceptionist || executed.ModelID != "modelo-local" {
		t.Fatalf("a trilha precisa identificar agente e modelo: %+v", executed)
	}
	if executed.Effect != ServiceAgentEffectRead {
		t.Fatalf("o efeito precisa constar: %+v", executed)
	}
}

func TestServiceAgentAuditRecordsUnavailableAndFailed(t *testing.T) {
	runtime, _, conversationID, now := auditFixture(t)

	// VENDAS: `commerce.stock.read` é do agente de COMÉRCIO — para vendas ela é BLOQUEADA (privilégio);
	// e `crm.opportunity.write` sem título FALHA no executor.
	sales, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSales, ConversationID: conversationID,
		Intent: "quero o plano", RequestedTools: []string{"commerce.stock.read", "crm.opportunity.write"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if sales.AuditError != "" {
		t.Fatalf("gravação da trilha falhou: %s", sales.AuditError)
	}

	// COMÉRCIO: declara `commerce.stock.read`, mas o store não está ligado — INDISPONÍVEL.
	commerce, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentCommerce, ConversationID: conversationID,
		Intent: "tem em estoque?", RequestedTools: []string{"commerce.stock.read"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if commerce.AuditError != "" {
		t.Fatalf("gravação da trilha falhou: %s", commerce.AuditError)
	}
	unavailable := callByName(commerce.ToolCalls, "commerce.stock.read")
	if unavailable == nil || unavailable.Status != ServiceAgentToolUnavailable {
		t.Fatalf("sem store ligado, o agente de comércio precisa ver INDISPONÍVEL: %+v", commerce.ToolCalls)
	}

	trail, err := runtime.AuditTrail("org-a", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(trail) != 3 {
		t.Fatalf("as três tentativas precisam estar na trilha: %d", len(trail))
	}
	statuses := map[string][]string{}
	for _, entry := range trail {
		statuses[entry.ToolName] = append(statuses[entry.ToolName], entry.Status)
	}
	// Bloqueado, indisponível e falho: os três estados são EVIDÊNCIA, não ruído.
	if !containsString(statuses["commerce.stock.read"], ServiceAgentToolBlocked) {
		t.Fatalf("a tentativa bloqueada precisa constar: %+v", statuses)
	}
	if !containsString(statuses["commerce.stock.read"], ServiceAgentToolUnavailable) {
		t.Fatalf("a indisponibilidade precisa constar: %+v", statuses)
	}
	if !containsString(statuses["crm.opportunity.write"], ServiceAgentToolFailed) {
		t.Fatalf("a falha precisa constar: %+v", statuses)
	}
}

func TestServiceAgentAuditCapIsPerOrganization(t *testing.T) {
	runtime, store, conversationID, now := auditFixture(t)
	store.MaxPerOrganization = 3

	// Organização A estoura o teto; B tem UM registro que não pode ser expulso pelo volume de A.
	for index := range 5 {
		if _, err := runtime.RunTurn(ServiceAgentTurnRequest{
			OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
			Intent: "oi", RequestedTools: []string{"crm.contact.read"},
		}, now.Add(time.Duration(index)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	// A trilha de B é gravada diretamente no store: o turno de B não tem conversa própria no fixture.
	if _, err := store.Append("org-b", []ServiceAgentAuditEntry{{
		ID: "aud_b", OrganizationID: "org-b", AgentID: ServiceAgentSales, ToolName: "catalog.read",
		Status: ServiceAgentToolExecuted, RecordedAt: now,
	}}); err != nil {
		t.Fatal(err)
	}

	a, err := runtime.AuditTrail("org-a", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != store.MaxPerOrganization {
		t.Fatalf("o teto da organização A precisa ser %d, veio %d", store.MaxPerOrganization, len(a))
	}
	b, err := runtime.AuditTrail("org-b", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 1 || b[0].ID != "aud_b" {
		t.Fatalf("o volume de A expulsou a evidência de B: %+v", b)
	}
}

func TestServiceAgentAuditIsolationHoldsOnDisk(t *testing.T) {
	runtime, store, conversationID, now := auditFixture(t)
	if _, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi", RequestedTools: []string{"crm.contact.read"},
	}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append("org-b", []ServiceAgentAuditEntry{{
		ID: "aud_org_b_unico", OrganizationID: "org-b", ToolName: "catalog.read", Status: ServiceAgentToolExecuted,
	}}); err != nil {
		t.Fatal(err)
	}

	// A conferência é ESTRUTURAL: lê o arquivo e examina as ENTRADAS.
	//
	// A primeira versão procurava a substring "aud_b" no texto do arquivo e falhava ~1 vez a cada 16
	// execuções, porque o ID gerado é `aud_` + os primeiros 12 caracteres de um UUID — quando o primeiro
	// caractere sorteado é `b`, o ID legítimo de org-a contém "aud_b". Procurar substring em identificador
	// aleatório é armadilha; comparar o CAMPO é o que a pergunta realmente faz.
	lerEntradas := func(organizacao string) []ServiceAgentAuditEntry {
		t.Helper()
		arquivo := filepath.Join(store.Root(), sanitizeIDComponent(organizacao)+".json")
		conteudo, err := os.ReadFile(arquivo)
		if err != nil {
			t.Fatalf("arquivo da organização %s ausente: %v", organizacao, err)
		}
		var file serviceAgentAuditFile
		if err := json.Unmarshal(conteudo, &file); err != nil {
			t.Fatalf("arquivo da organização %s ilegível: %v", organizacao, err)
		}
		return file.Entries
	}

	entradasA := lerEntradas("org-a")
	if len(entradasA) == 0 {
		t.Fatal("a trilha de org-a precisa conter o registro do turno")
	}
	for _, entry := range entradasA {
		if entry.OrganizationID != "org-a" {
			t.Fatalf("o arquivo de org-a contém registro de %q: %+v", entry.OrganizationID, entry)
		}
		if entry.ToolName == "catalog.read" {
			t.Fatalf("ferramenta de outra organização no arquivo de org-a: %+v", entry)
		}
	}
	if !containsString(auditToolNames(entradasA), "crm.contact.read") {
		t.Fatalf("o arquivo de org-a precisa conter a própria execução: %+v", entradasA)
	}

	entradasB := lerEntradas("org-b")
	if len(entradasB) != 1 || entradasB[0].OrganizationID != "org-b" {
		t.Fatalf("o arquivo de org-b precisa conter só o registro dela: %+v", entradasB)
	}
	if containsString(auditToolNames(entradasB), "crm.contact.read") {
		t.Fatalf("a execução de org-a vazou para o arquivo de org-b: %+v", entradasB)
	}
}

func auditToolNames(entries []ServiceAgentAuditEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.ToolName)
	}
	return names
}

func TestServiceAgentAuditReportsPersistenceFailure(t *testing.T) {
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	// Um ARQUIVO ocupa o lugar do diretório da trilha: a gravação falha de forma portátil.
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("arquivo, não diretório"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime.SetAuditStore(NewServiceAgentAuditStore(filepath.Join(blocker, "audit")))

	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi", RequestedTools: []string{"crm.contact.read"},
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	// A falha de auditoria precisa ser VISÍVEL: perder registro em silêncio é o pior desfecho.
	if turn.AuditError == "" {
		t.Fatal("falha de gravação da trilha precisa ser reportada no turno")
	}
	// A PROPRIEDADE testada é "a falha é reportada" — e o caminho para isso DIFERE por sistema: no Linux o
	// caminho bloqueado devolve ENOTDIR na leitura (erro), no Windows ele aparece como inexistente e o que
	// sobra é o erro de GRAVAÇÃO guardado no estado. Afirmar o mecanismo em vez da propriedade foi o que
	// fez este teste falhar na paridade Linux (mesma lição de E-115/E-127: valide a injeção no SO do gate).
	status, err := runtime.AuditStatus("org-a")
	if err != nil {
		// Erro declarado na leitura já é a falha reportada.
		return
	}
	if status.Error == "" {
		t.Fatal("o estado da trilha precisa reportar a falha")
	}
	if !status.Persisted {
		t.Fatal("o estado precisa declarar que a persistência está configurada")
	}
}

func TestServiceAgentAuditKeepsContactTextOut(t *testing.T) {
	runtime, store, conversationID, now := auditFixture(t)
	segredo := "meu cartao 4111 1111 1111 1111 e meu pedido"
	if _, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: segredo, RequestedTools: []string{"crm.contact.read"},
	}, now); err != nil {
		t.Fatal(err)
	}
	arquivo := filepath.Join(store.Root(), sanitizeIDComponent("org-a")+".json")
	conteudo, err := os.ReadFile(arquivo)
	if err != nil {
		t.Fatal(err)
	}
	// O que o cliente escreveu NÃO entra na trilha: conteúdo de terceiro fica fora da auditoria.
	if strings.Contains(string(conteudo), "4111") || strings.Contains(string(conteudo), segredo) {
		t.Fatal("o texto do contato não pode ser gravado na trilha de auditoria")
	}
	if !strings.Contains(string(conteudo), "crm.contact.read") {
		t.Fatal("a trilha precisa registrar a ferramenta executada")
	}
}

func TestServiceAgentAuditWithoutStoreDeclaresNoDurability(t *testing.T) {
	runtime, _, _, _, conversationID := serviceAgentRuntime(t, true)
	// Sem `SetAuditStore`, a trilha continua por turno e o estado NÃO presume durabilidade.
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi", RequestedTools: []string{"crm.contact.read"},
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if turn.AuditError != "" {
		t.Fatalf("sem store não há erro de gravação: %s", turn.AuditError)
	}
	if len(turn.ToolCalls) != 1 {
		t.Fatalf("a trilha por turno precisa continuar existindo: %+v", turn.ToolCalls)
	}
	status, err := runtime.AuditStatus("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if status.Persisted {
		t.Fatal("sem store, Persisted precisa ser false")
	}
	// E a leitura da trilha durável é erro declarado, não lista vazia.
	if _, err := runtime.AuditTrail("org-a", 0); err == nil {
		t.Fatal("sem trilha durável, a leitura precisa ser erro declarado")
	}
}
