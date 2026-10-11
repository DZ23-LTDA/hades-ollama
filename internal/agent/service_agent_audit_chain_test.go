package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Integridade da trilha de auditoria (estágio 35): a CADEIA de hashes torna a adulteração DETECTÁVEL.
//
// O que estes testes travam:
//
//  1. cadeia intacta verifica;
//  2. CONTEÚDO alterado é detectado, apontando o registro;
//  3. registro REMOVIDO quebra o elo e é detectado;
//  4. registros REORDENADOS são detectados;
//  5. é RECUSADO estender uma trilha cuja cadeia não confere — gravar sobre adulteração produziria um
//     documento que parece íntegro a partir dali, escondendo a adulteração anterior;
//  6. registros ANTERIORES à cadeia não são declarados válidos (não verificáveis, e isso é dito).
func tamperAuditFile(t *testing.T, store *ServiceAgentAuditStore, organizationID string, mudar func(*serviceAgentAuditFile)) {
	t.Helper()
	entries, err := store.Load(organizationID)
	if err != nil {
		t.Fatal(err)
	}
	file := serviceAgentAuditFile{Entries: entries}
	mudar(&file)
	path := filepath.Join(store.Root(), sanitizeIDComponent(organizationID)+".json")
	if err := writeJSONAtomic(path, file); err != nil {
		t.Fatal(err)
	}
}

func TestServiceAgentAuditChainVerifiesWhenIntact(t *testing.T) {
	runtime, store, conversationID, now := auditFixture(t)
	// Três turnos: a cadeia precisa encadear registros de turnos diferentes.
	for index := range 3 {
		if _, err := runtime.RunTurn(ServiceAgentTurnRequest{
			OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
			Intent: "oi", RequestedTools: []string{"crm.contact.read"},
		}, now.Add(time.Duration(index)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	verification, err := store.VerifyTrail("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Valid || verification.Entries != 3 || verification.Legacy != 0 {
		t.Fatalf("cadeia intacta precisa verificar: %+v", verification)
	}
	// E o estado exposto carrega a verificação.
	status, err := runtime.AuditStatus("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if status.Verification == nil || !status.Verification.Valid {
		t.Fatalf("o estado precisa trazer a verificação: %+v", status.Verification)
	}
}

func TestServiceAgentAuditDetectsTamperedContent(t *testing.T) {
	runtime, store, conversationID, now := auditFixture(t)
	if _, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi", RequestedTools: []string{"crm.contact.read"},
	}, now); err != nil {
		t.Fatal(err)
	}
	// Adultera o STATUS de um registro já gravado: o hash deixa de conferir.
	tamperAuditFile(t, store, "org-a", func(file *serviceAgentAuditFile) {
		file.Entries[0].Status = ServiceAgentToolBlocked
	})
	verification, err := store.VerifyTrail("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Valid {
		t.Fatal("conteúdo adulterado precisa ser DETECTADO")
	}
	if verification.FirstDivergence == "" || !strings.Contains(verification.Reason, "adulterado") {
		t.Fatalf("a verificação precisa apontar o registro adulterado: %+v", verification)
	}
	// E o estado da trilha passa a denunciar isso.
	status, err := runtime.AuditStatus("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if status.Verification == nil || status.Verification.Valid {
		t.Fatalf("o estado precisa declarar a trilha inválida: %+v", status.Verification)
	}
}

func TestServiceAgentAuditDetectsRemovedEntry(t *testing.T) {
	runtime, store, conversationID, now := auditFixture(t)
	for index := range 2 {
		if _, err := runtime.RunTurn(ServiceAgentTurnRequest{
			OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
			Intent: "oi", RequestedTools: []string{"crm.contact.read"},
		}, now.Add(time.Duration(index)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	// Remove o PRIMEIRO registro: o elo do segundo aponta para um anterior que não existe mais.
	tamperAuditFile(t, store, "org-a", func(file *serviceAgentAuditFile) {
		file.Entries = file.Entries[1:]
	})
	verification, err := store.VerifyTrail("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Valid {
		t.Fatal("registro removido precisa quebrar a cadeia")
	}
}

func TestServiceAgentAuditDetectsReordering(t *testing.T) {
	runtime, store, conversationID, now := auditFixture(t)
	for index := range 2 {
		if _, err := runtime.RunTurn(ServiceAgentTurnRequest{
			OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
			Intent: "oi", RequestedTools: []string{"crm.contact.read"},
		}, now.Add(time.Duration(index)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	tamperAuditFile(t, store, "org-a", func(file *serviceAgentAuditFile) {
		file.Entries[0], file.Entries[1] = file.Entries[1], file.Entries[0]
	})
	verification, err := store.VerifyTrail("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Valid {
		t.Fatal("registros reordenados precisam ser detectados")
	}
}

func TestServiceAgentAuditRefusesToExtendBrokenChain(t *testing.T) {
	runtime, store, conversationID, now := auditFixture(t)
	if _, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi", RequestedTools: []string{"crm.contact.read"},
	}, now); err != nil {
		t.Fatal(err)
	}
	tamperAuditFile(t, store, "org-a", func(file *serviceAgentAuditFile) {
		file.Entries[0].Reason = "motivo reescrito"
	})

	// Estender uma trilha já adulterada esconderia a adulteração anterior: a gravação é RECUSADA, e o
	// turno reporta o motivo.
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentReceptionist, ConversationID: conversationID,
		Intent: "oi", RequestedTools: []string{"crm.contact.read"},
	}, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if turn.AuditError == "" {
		t.Fatal("a recusa de estender cadeia quebrada precisa aparecer no turno")
	}
	if !strings.Contains(turn.AuditError, "chain") {
		t.Fatalf("o motivo precisa identificar a cadeia quebrada: %s", turn.AuditError)
	}
	// E o erro dedicado existe para quem quiser tratar o caso.
	if !errors.Is(errors.New(turn.AuditError), ErrServiceAgentAuditChainBroken) &&
		!strings.Contains(turn.AuditError, ErrServiceAgentAuditChainBroken.Error()) {
		t.Fatalf("o erro precisa ser reconhecível: %v", ErrServiceAgentAuditChainBroken)
	}
}

func TestServiceAgentAuditLegacyEntriesAreNotDeclaredValid(t *testing.T) {
	store := NewServiceAgentAuditStore(serviceAgentAuditRoot(t.TempDir()))
	// Trilha escrita ANTES da cadeia: sem hash. Não pode ser declarada válida.
	legacy := serviceAgentAuditFile{Entries: []ServiceAgentAuditEntry{{
		ID: "aud_legado", OrganizationID: "org-a", AgentID: ServiceAgentSales, ToolName: "catalog.read",
		Status: ServiceAgentToolExecuted, RecordedAt: time.Now().UTC(),
	}}}
	path := filepath.Join(store.Root(), sanitizeIDComponent("org-a")+".json")
	if err := os.MkdirAll(store.Root(), 0o700); err != nil {
		t.Fatal(err)
	}
	dados, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, dados, 0o600); err != nil {
		t.Fatal(err)
	}
	verification, err := store.VerifyTrail("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Valid {
		t.Fatal("registro anterior à cadeia NÃO pode ser declarado válido")
	}
	// ...e também não pode ser chamado de adulterado: aqui a limitação é de IDADE, não há evidência de edição.
	if verification.Broken {
		t.Fatalf("trilha antiga não é trilha adulterada: %+v", verification)
	}
	if verification.Legacy != 1 || !strings.Contains(verification.Reason, "não verificável") {
		t.Fatalf("a limitação precisa ser declarada: %+v", verification)
	}
	// E a trilha continua utilizável: congelar a auditoria por causa de registros antigos seria pior que o
	// problema. O registro novo entra encadeado a partir do gênesis.
	total, err := store.Append("org-a", []ServiceAgentAuditEntry{{
		ID: "aud_novo", OrganizationID: "org-a", ToolName: "crm.contact.read", Status: ServiceAgentToolExecuted,
	}})
	if err != nil {
		t.Fatalf("trilha legada não pode impedir nova gravação: %v", err)
	}
	if total != 2 {
		t.Fatalf("esperava 2 registros após o append: %d", total)
	}
}
