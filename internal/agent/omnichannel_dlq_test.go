package agent

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Persistência da fila de mortos (estágio 34). O que estes testes travam:
//
//  1. o evento que falhou SOBREVIVE ao reinício — fila em memória perdia a mensagem de um cliente;
//  2. a tentativa e o motivo novo também sobrevivem (histórico do incidente);
//  3. o teto é POR ORGANIZAÇÃO: o volume de um tenant não expulsa a evidência de outro;
//  4. isolamento vale no DISCO: o arquivo de uma organização não contém entrada de outra;
//  5. persistir que falha NÃO apaga da memória, e o estado REPORTA o erro.
func omnicanalFixture(t *testing.T) (*OmnichannelRuntime, *OmnichannelDLQStore, time.Time) {
	t.Helper()
	root := t.TempDir()
	store := NewOmnichannelDLQStore(omnichannelDLQRoot(root))
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetDLQStore(store)
	return runtime, store, time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
}

func deadLetterEvent(organizationID, externalID string) OmnichannelEvent {
	return OmnichannelEvent{
		OrganizationID:    organizationID,
		ChannelID:         "whatsapp",
		ConnectorID:       "cn_" + organizationID,
		ExternalMessageID: externalID,
		ContactID:         "5511999999999",
		MessageType:       "text",
		Preview:           "mensagem que falhou",
	}
}

func TestOmnichannelDeadLetterSurvivesRestart(t *testing.T) {
	runtime, store, now := omnicanalFixture(t)
	runtime.deadLetter(deadLetterEvent("org_a", "wamid.1"), "whatsapp", "inbox: falha simulada", now)

	// REINÍCIO: runtime novo, mesmo diretório de persistência.
	depois := NewOmnichannelRuntime(nil)
	depois.SetDLQStore(NewOmnichannelDLQStore(store.Root()))

	entries := depois.DeadLetters("org_a")
	if len(entries) != 1 {
		t.Fatalf("a fila de mortos precisa sobreviver ao reinício: %d entradas", len(entries))
	}
	if entries[0].Event.ExternalMessageID != "wamid.1" || entries[0].Reason == "" {
		t.Fatalf("entrada recuperada incompleta: %+v", entries[0])
	}
	if entries[0].Attempts != 1 {
		t.Fatalf("tentativas recuperadas erradas: %d", entries[0].Attempts)
	}
}

func TestOmnichannelDeadLetterPersistsAttemptsAndReason(t *testing.T) {
	runtime, store, now := omnicanalFixture(t)
	runtime.deadLetter(deadLetterEvent("org_a", "wamid.2"), "whatsapp", "primeiro motivo", now)
	entries := runtime.DeadLetters("org_a")
	if len(entries) != 1 {
		t.Fatalf("esperava 1 entrada, veio %d", len(entries))
	}
	// Sem inbox, a retentativa falha e ATUALIZA o motivo, incrementando a tentativa.
	if _, err := runtime.RetryDeadLetter("org_a", entries[0].ID, now.Add(time.Minute)); err == nil {
		t.Fatal("sem inbox a retentativa precisa falhar")
	}

	depois := NewOmnichannelRuntime(nil)
	depois.SetDLQStore(NewOmnichannelDLQStore(store.Root()))
	recuperadas := depois.DeadLetters("org_a")
	if len(recuperadas) != 1 {
		t.Fatalf("a entrada precisa continuar na fila: %d", len(recuperadas))
	}
	if recuperadas[0].Attempts != 2 {
		t.Fatalf("a tentativa precisa sobreviver ao reinício: %d", recuperadas[0].Attempts)
	}
	if !strings.Contains(recuperadas[0].Reason, "inbox") {
		t.Fatalf("o motivo novo precisa sobreviver ao reinício: %q", recuperadas[0].Reason)
	}
}

func TestOmnichannelDeadLetterDiscardPersistsRemoval(t *testing.T) {
	runtime, store, now := omnicanalFixture(t)
	runtime.deadLetter(deadLetterEvent("org_a", "wamid.3"), "whatsapp", "motivo", now)
	entries := runtime.DeadLetters("org_a")
	if _, err := runtime.DiscardDeadLetter("org_a", entries[0].ID); err != nil {
		t.Fatal(err)
	}
	// Descartar precisa chegar ao disco: senão o reinício ressuscita o que foi abandonado.
	depois := NewOmnichannelRuntime(nil)
	depois.SetDLQStore(NewOmnichannelDLQStore(store.Root()))
	if restantes := depois.DeadLetters("org_a"); len(restantes) != 0 {
		t.Fatalf("descarte precisa ser persistido: sobraram %d", len(restantes))
	}
	// Descartar o que não existe é erro declarado, não silêncio.
	if _, err := runtime.DiscardDeadLetter("org_a", "dlq_inexistente"); err == nil {
		t.Fatal("descartar entrada inexistente precisa ser erro")
	}
}

func TestOmnichannelDeadLetterCapIsPerOrganization(t *testing.T) {
	runtime, _, now := omnicanalFixture(t)
	// A organização A estoura o teto; B tem uma única entrada que NÃO pode ser expulsa pelo volume de A.
	for i := range OmnichannelMaxDLQ + 20 {
		runtime.deadLetter(deadLetterEvent("org_a", "a-"+strconv.Itoa(i)), "whatsapp", "volume", now.Add(time.Duration(i)*time.Second))
	}
	runtime.deadLetter(deadLetterEvent("org_b", "b-1"), "whatsapp", "unica", now)

	a := runtime.DeadLetters("org_a")
	if len(a) != OmnichannelMaxDLQ {
		t.Fatalf("o teto da organização A precisa ser %d, veio %d", OmnichannelMaxDLQ, len(a))
	}
	b := runtime.DeadLetters("org_b")
	if len(b) != 1 {
		t.Fatalf("o volume de A expulsou a evidência de B: %d entradas em B", len(b))
	}
	// As MAIS ANTIGAS de A saem primeiro; a última entrada continua sendo a mais recente.
	if a[len(a)-1].Event.ExternalMessageID == "" {
		t.Fatal("a entrada mais recente de A precisa ser preservada")
	}
}

func TestOmnichannelDeadLetterIsolationHoldsOnDisk(t *testing.T) {
	runtime, store, now := omnicanalFixture(t)
	runtime.deadLetter(deadLetterEvent("org_a", "segredo-de-a"), "whatsapp", "motivo", now)
	runtime.deadLetter(deadLetterEvent("org_b", "segredo-de-b"), "whatsapp", "motivo", now)

	arquivoA := filepath.Join(store.Root(), sanitizeIDComponent("org_a")+".json")
	conteudo, err := os.ReadFile(arquivoA)
	if err != nil {
		t.Fatalf("arquivo da organização A ausente: %v", err)
	}
	if strings.Contains(string(conteudo), "segredo-de-b") {
		t.Fatal("o arquivo da organização A contém entrada da organização B")
	}
	if !strings.Contains(string(conteudo), "segredo-de-a") {
		t.Fatal("o arquivo da organização A precisa conter a entrada dela")
	}
	// E a leitura escopada confirma que ninguém enxerga o do outro.
	if ids := runtime.DeadLetters("org_b"); len(ids) != 1 || ids[0].Event.ExternalMessageID != "segredo-de-b" {
		t.Fatalf("leitura escopada de B errada: %+v", ids)
	}
}

func TestOmnichannelDeadLetterReportsPersistenceFailure(t *testing.T) {
	root := t.TempDir()
	// Um ARQUIVO ocupa o lugar do diretório de persistência: a escrita falha de forma portátil
	// (mesma técnica de injeção usada no resto do pacote).
	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("arquivo, não diretório"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewOmnichannelDLQStore(filepath.Join(blocker, "dlq"))
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetDLQStore(store)
	now := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)

	runtime.deadLetter(deadLetterEvent("org_a", "wamid.falha"), "whatsapp", "motivo", now)

	// A entrada NÃO é descartada: perder em silêncio é o defeito que estamos corrigindo.
	if entries := runtime.DeadLetters("org_a"); len(entries) != 1 {
		t.Fatalf("a entrada precisa continuar em memória quando a persistência falha: %d", len(entries))
	}
	status, err := runtime.OmnichannelDLQStatus("org_a")
	if err != nil {
		t.Fatal(err)
	}
	if status.Error == "" {
		t.Fatal("a falha de persistência precisa ser REPORTADA no estado")
	}
	if !status.Persisted || status.Storage == "" {
		t.Fatalf("o estado precisa declarar onde tentou persistir: %+v", status)
	}
	if status.Entries != 1 || status.PerOrganization != OmnichannelMaxDLQ {
		t.Fatalf("estado incompleto: %+v", status)
	}
}

func TestOmnichannelDeadLetterWithoutStoreDeclaresMemoryOnly(t *testing.T) {
	runtime := NewOmnichannelRuntime(nil)
	now := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	runtime.deadLetter(deadLetterEvent("org_a", "wamid.mem"), "whatsapp", "motivo", now)

	status, err := runtime.OmnichannelDLQStatus("org_a")
	if err != nil {
		t.Fatal(err)
	}
	// Sem store, a durabilidade NÃO é presumida: o estado diz que é só memória.
	if status.Persisted {
		t.Fatalf("sem store, Persisted precisa ser false: %+v", status)
	}
	if status.Entries != 1 {
		t.Fatalf("a fila em memória precisa continuar funcionando: %+v", status)
	}
}

func TestOmnichannelDeadLetterCorruptedFileIsReportedNotHidden(t *testing.T) {
	root := t.TempDir()
	store := NewOmnichannelDLQStore(omnichannelDLQRoot(root))
	if err := os.MkdirAll(store.Root(), 0o700); err != nil {
		t.Fatal(err)
	}
	// Arquivo CORROMPIDO no lugar da fila: tratar isso como "fila vazia" faria uma fila de mortos
	// corrompida parecer limpa, apagando a evidência de mensagens não processadas.
	arquivo := filepath.Join(store.Root(), sanitizeIDComponent("org_a")+".json")
	if err := os.WriteFile(arquivo, []byte("{ isto não é json }"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetDLQStore(store)

	// A leitura não explode e a fila em memória segue utilizável...
	if entries := runtime.DeadLetters("org_a"); len(entries) != 0 {
		t.Fatalf("arquivo corrompido não pode inventar entradas: %d", len(entries))
	}
	// ...mas o PROBLEMA aparece no estado, em vez de virar silêncio.
	status, err := runtime.OmnichannelDLQStatus("org_a")
	if err != nil {
		t.Fatal(err)
	}
	if status.Error == "" {
		t.Fatal("arquivo corrompido precisa ser reportado no estado da fila")
	}
}
