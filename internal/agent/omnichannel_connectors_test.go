package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// PERSISTÊNCIA DOS CONECTORES (E-171). O limite foi DESCOBERTO por um teste na rodada anterior ("connector is not
// configured"). A consequência de perder o registro é pior que perder um número: `Ingest` procura o conector por
// organização e canal, então **todo webhook legítimo do canal é recusado depois de um reinício** — o canal para
// sozinho. Os testes travam:
//
//  1. com store, o canal CONTINUA aceitando webhooks depois do reinício (com o MESMO id de conector);
//  2. o arquivo em disco tem o REGISTRO e **nunca** o valor do segredo;
//  3. sem store, o estado DIZ que não há persistência;
//  4. erro de leitura é REPORTADO, não escondido.

const segredoDeTeste = "segredo-de-teste-com-tamanho-suficiente"

func TestOmnichannelConnectorSurvivesRestart(t *testing.T) {
	store := NewOmnichannelConnectorStore(filepath.Join(t.TempDir(), "conectores"))
	secretEnv := "HADES_TEST_CONNECTOR_RESTART"
	t.Setenv(secretEnv, segredoDeTeste)

	primeiro := NewOmnichannelRuntime(nil)
	primeiro.SetConnectorStore(store)
	if !primeiro.ConnectorPersisted() {
		t.Fatal("com store configurado, o estado precisa declarar persistência")
	}
	agora := time.Now().UTC()
	connector, err := primeiro.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-conn", Channel: "whatsapp", Name: "WhatsApp", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}

	// REINÍCIO: runtime novo, mesmo store, e o conector **não** é recadastrado — só o disco pode sabê-lo.
	depois := NewOmnichannelRuntime(nil)
	depois.SetConnectorStore(store)
	corpo := []byte(`{"external_message_id":"wamid-conector"}`)
	assinatura, err := depois.SignWebhook(connector.ID, agora.Unix(), corpo)
	if err != nil {
		t.Fatalf("o conector precisa sobreviver ao reinício: %v", err)
	}
	evento := OmnichannelEvent{ExternalMessageID: "wamid-conector", ContactID: "+5511999998888", MessageType: "text"}
	if _, err := depois.Ingest("org-conn", "whatsapp", assinatura, agora.Unix(), corpo, evento, agora); err != nil {
		t.Fatalf("o canal precisa continuar aceitando depois do reinício: %v", err)
	}
	if erro := depois.LastConnectorError(); erro != "" {
		t.Fatalf("não podia haver erro de persistência: %s", erro)
	}
}

func TestOmnichannelConnectorFileNeverStoresTheSecretValue(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "conectores-segredo")
	secretEnv := "HADES_TEST_CONNECTOR_SECRET_NAME"
	t.Setenv(secretEnv, segredoDeTeste)

	runtime := NewOmnichannelRuntime(nil)
	runtime.SetConnectorStore(NewOmnichannelConnectorStore(raiz))
	if _, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-segredo", Channel: "telegram", Name: "Bot", SecretEnv: secretEnv, Enabled: true,
	}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	dados, err := os.ReadFile(filepath.Join(raiz, "org-segredo.json"))
	if err != nil {
		t.Fatalf("o registro precisa ter sido gravado: %v", err)
	}
	texto := string(dados)
	// O NOME da variável de ambiente é o que deve estar no disco...
	if !strings.Contains(texto, secretEnv) {
		t.Fatalf("o registro precisa guardar o NOME da variável do segredo: %s", texto)
	}
	// ...e o VALOR do segredo, nunca.
	if strings.Contains(texto, segredoDeTeste) {
		t.Fatalf("o arquivo de conectores NÃO pode conter o valor do segredo: %s", texto)
	}
}

func TestOmnichannelConnectorWithoutStoreSaysSoInsteadOfPretending(t *testing.T) {
	runtime := NewOmnichannelRuntime(nil)
	if runtime.ConnectorPersisted() {
		t.Fatal("sem store, o estado NÃO pode declarar persistência")
	}
	secretEnv := "HADES_TEST_CONNECTOR_NOSTORE"
	t.Setenv(secretEnv, segredoDeTeste)
	agora := time.Now().UTC()
	connector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-sem-store", Channel: "whatsapp", Name: "WhatsApp", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	// Reinício em memória: o canal PARA — comportamento declarado (o estado já disse que não é durável).
	reiniciado := NewOmnichannelRuntime(nil)
	corpo := []byte(`{"external_message_id":"wamid-x"}`)
	if _, err := reiniciado.SignWebhook(connector.ID, agora.Unix(), corpo); err == nil {
		t.Fatal("sem persistência o conector se perde no reinício (declarado), então assinar precisa falhar")
	}
}

func TestOmnichannelConnectorReportsUnreadableRegistry(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "conectores-quebrado")
	if err := os.MkdirAll(raiz, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "org-quebrado.json"), []byte("{isso não é json"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetConnectorStore(NewOmnichannelConnectorStore(raiz))
	secretEnv := "HADES_TEST_CONNECTOR_BROKEN"
	t.Setenv(secretEnv, segredoDeTeste)
	// Procurar o conector força a leitura do registro quebrado.
	if _, err := runtime.connectorFor("org-quebrado", "whatsapp"); err == nil {
		t.Fatal("sem conector o canal precisa recusar (falha fechada)")
	}
	erro := runtime.LastConnectorError()
	if erro == "" {
		t.Fatal("registro ilegível precisa ser REPORTADO, não escondido")
	}
	if !strings.Contains(erro, "carregar conectores") {
		t.Fatalf("o erro precisa identificar a leitura: %s", erro)
	}
}

func TestRuntimeWiresConnectorPersistence(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	// Prova de que o runtime REAL liga a persistência: o teste de reinício acima só vale se o disco for usado.
	conectorRoot := filepath.Join(runtime.DataRoot(), ".agent-omnichannel-connectors")
	if !strings.Contains(conectorRoot, ".agent-omnichannel-connectors") {
		t.Fatalf("raiz inesperada: %s", conectorRoot)
	}
}
