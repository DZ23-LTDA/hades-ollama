package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// DEDUPE DURÁVEL E COM TETO (E-172). Dois defeitos no mesmo lugar, e os testes travam cada um:
//
//  1. o índice vivia só em memória ⇒ um reinício "esquecia" o que já entrou e a entrega repetida voltava a ser
//     processada — exatamente o que a dedupe existe para impedir;
//  2. o índice não tinha TETO nem poda ⇒ num processo de longa duração crescia sem limite.
//
// Propriedades travadas: a duplicata é reconhecida DEPOIS do reinício; o índice respeita o TETO; entrada vencida
// não protege; sem store o estado declara que não há persistência; erro de leitura é reportado.

const segredoDedupe = "segredo-de-teste-com-tamanho-suficiente"

func dedupeEnviar(t *testing.T, runtime *OmnichannelRuntime, connectorID, id string) (OmnichannelEvent, error) {
	t.Helper()
	agora := time.Now().UTC()
	corpo := []byte(`{"external_message_id":"` + id + `"}`)
	assinatura, err := runtime.SignWebhook(connectorID, agora.Unix(), corpo)
	if err != nil {
		t.Fatal(err)
	}
	evento := OmnichannelEvent{ExternalMessageID: id, ContactID: "+5511999998888", MessageType: "text", Preview: "olá"}
	return runtime.Ingest("org-dedupe", "whatsapp", assinatura, agora.Unix(), corpo, evento, agora)
}

func TestOmnichannelDedupeSurvivesRestart(t *testing.T) {
	store := NewOmnichannelDedupeStore(filepath.Join(t.TempDir(), "dedupe"))
	const secretEnv = "HADES_TEST_DEDUPE_RESTART"
	t.Setenv(secretEnv, segredoDedupe)

	primeiro := NewOmnichannelRuntime(nil)
	primeiro.SetDedupeStore(store)
	if !primeiro.DedupePersisted() {
		t.Fatal("com store configurado, o estado precisa declarar persistência")
	}
	agora := time.Now().UTC()
	connector, err := primeiro.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-dedupe", Channel: "whatsapp", Name: "WhatsApp", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dedupeEnviar(t, primeiro, connector.ID, "wamid-dedupe-1"); err != nil {
		t.Fatal(err)
	}

	// REINÍCIO: runtime novo, mesmo store. A MESMA entrega precisa ser reconhecida como duplicata.
	depois := NewOmnichannelRuntime(nil)
	depois.SetDedupeStore(store)
	if _, err := depois.RegisterConnector(OmnichannelConnector{
		ID: connector.ID, OrganizationID: "org-dedupe", Channel: "whatsapp", Name: "WhatsApp", SecretEnv: secretEnv, Enabled: true,
	}, agora); err != nil {
		t.Fatal(err)
	}
	repetido, err := dedupeEnviar(t, depois, connector.ID, "wamid-dedupe-1")
	if err != nil {
		t.Fatalf("entrega repetida não pode falhar: %v", err)
	}
	if !repetido.Deduplicated {
		t.Fatalf("a duplicata precisa ser reconhecida DEPOIS do reinício: %+v", repetido)
	}
	if repetido.ExternalMessageID != "wamid-dedupe-1" {
		t.Fatalf("o registro devolvido precisa identificar a mensagem: %+v", repetido)
	}
	// E uma mensagem NOVA continua entrando normalmente.
	novo, err := dedupeEnviar(t, depois, connector.ID, "wamid-dedupe-2")
	if err != nil {
		t.Fatal(err)
	}
	if novo.Deduplicated {
		t.Fatalf("mensagem nova não pode ser marcada como duplicata: %+v", novo)
	}
}

func TestOmnichannelDedupeRespectsTheCap(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "dedupe-teto")
	store := NewOmnichannelDedupeStore(raiz)
	agora := time.Now().UTC()

	// SEMEIA um índice acima do teto: o caminho testado é a PODA na carga (o anterior fazia 5025 ingestões,
	// cada uma reescrevendo o arquivo inteiro, e levava ~52 s — medir o caminho certo é mais rápido e mais claro).
	excedente := map[string]OmnichannelDedupeEntry{}
	for i := range OmnichannelMaxDedupePerOrg + 25 {
		excedente["org-dedupe|whatsapp|wamid-teto-"+itoa(i)] = OmnichannelDedupeEntry{
			ExternalMessageID: "wamid-teto-" + itoa(i), Channel: "whatsapp",
			IngestedAt: agora.Add(-time.Duration(i) * time.Second),
		}
	}
	if err := store.Save("org-dedupe", excedente); err != nil {
		t.Fatal(err)
	}

	const secretEnv = "HADES_TEST_DEDUPE_CAP"
	t.Setenv(secretEnv, segredoDedupe)
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetDedupeStore(store)
	connector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-dedupe", Channel: "whatsapp", Name: "WhatsApp", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	// Uma mensagem nova dispara a carga (e a poda) do índice da organização.
	if _, err := dedupeEnviar(t, runtime, connector.ID, "wamid-teto-novo"); err != nil {
		t.Fatal(err)
	}
	if total := runtime.DedupeCount("org-dedupe"); total > OmnichannelMaxDedupePerOrg {
		t.Fatalf("o índice de dedupe precisa respeitar o teto de %d: %d", OmnichannelMaxDedupePerOrg, total)
	}
	// E os mais ANTIGOS foram os descartados: o mais novo da semente continua protegido.
	repetido, err := dedupeEnviar(t, runtime, connector.ID, "wamid-teto-0")
	if err != nil {
		t.Fatal(err)
	}
	if !repetido.Deduplicated {
		t.Fatalf("o registro mais RECENTE da semente precisa continuar no índice: %+v", repetido)
	}
}

func TestOmnichannelDedupeDoesNotHonourExpiredEntries(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "dedupe-vencido")
	store := NewOmnichannelDedupeStore(raiz)
	// Entrada VENCIDA gravada no disco: não pode bloquear uma mensagem nova com o mesmo id.
	vencido := time.Now().UTC().Add(-2 * OmnichannelDedupeRetention)
	if err := store.Save("org-dedupe", map[string]OmnichannelDedupeEntry{
		"org-dedupe|whatsapp|wamid-vencido": {
			ExternalMessageID: "wamid-vencido", Channel: "whatsapp", IngestedAt: vencido,
		},
	}); err != nil {
		t.Fatal(err)
	}
	const secretEnv = "HADES_TEST_DEDUPE_EXPIRED"
	t.Setenv(secretEnv, segredoDedupe)
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetDedupeStore(store)
	agora := time.Now().UTC()
	connector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-dedupe", Channel: "whatsapp", Name: "WhatsApp", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	evento, err := dedupeEnviar(t, runtime, connector.ID, "wamid-vencido")
	if err != nil {
		t.Fatal(err)
	}
	if evento.Deduplicated {
		t.Fatalf("entrada vencida não pode proteger uma mensagem nova: %+v", evento)
	}
}

func TestOmnichannelDedupeWithoutStoreSaysSoInsteadOfPretending(t *testing.T) {
	runtime := NewOmnichannelRuntime(nil)
	if runtime.DedupePersisted() {
		t.Fatal("sem store, o estado NÃO pode declarar persistência")
	}
	if erro := runtime.LastDedupeError(); erro != "" {
		t.Fatalf("sem store não há erro a reportar: %s", erro)
	}
}

func TestOmnichannelDedupeReportsUnreadableIndex(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "dedupe-quebrado")
	if err := os.MkdirAll(raiz, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "org-dedupe.json"), []byte("{isso não é json"), 0o600); err != nil {
		t.Fatal(err)
	}
	const secretEnv = "HADES_TEST_DEDUPE_BROKEN"
	t.Setenv(secretEnv, segredoDedupe)
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetDedupeStore(NewOmnichannelDedupeStore(raiz))
	agora := time.Now().UTC()
	connector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-dedupe", Channel: "whatsapp", Name: "WhatsApp", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	// O evento continua sendo processado (a falha é do índice), mas o estado precisa ACUSAR.
	if _, err := dedupeEnviar(t, runtime, connector.ID, "wamid-quebrado"); err != nil {
		t.Fatalf("o evento continua sendo processado: %v", err)
	}
	if runtime.LastDedupeError() == "" {
		t.Fatal("índice ilegível precisa ser REPORTADO, não escondido")
	}
}

// itoa evita depender de fmt apenas para o teste.
func itoa(valor int) string {
	if valor == 0 {
		return "0"
	}
	var digitos []byte
	for valor > 0 {
		digitos = append([]byte{byte('0' + valor%10)}, digitos...)
		valor /= 10
	}
	return string(digitos)
}

// O arquivo em disco precisa ser JSON legível e conter a chave da mensagem (prova de que a durabilidade tem
// conteúdo real, e não só um arquivo vazio).
func TestOmnichannelDedupeFileHasTheKeys(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "dedupe-arquivo")
	const secretEnv = "HADES_TEST_DEDUPE_FILE"
	t.Setenv(secretEnv, segredoDedupe)
	runtime := NewOmnichannelRuntime(nil)
	runtime.SetDedupeStore(NewOmnichannelDedupeStore(raiz))
	agora := time.Now().UTC()
	connector, err := runtime.RegisterConnector(OmnichannelConnector{
		OrganizationID: "org-dedupe", Channel: "whatsapp", Name: "WhatsApp", SecretEnv: secretEnv, Enabled: true,
	}, agora)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dedupeEnviar(t, runtime, connector.ID, "wamid-arquivo"); err != nil {
		t.Fatal(err)
	}
	dados, err := os.ReadFile(filepath.Join(raiz, "org-dedupe.json"))
	if err != nil {
		t.Fatal(err)
	}
	var arquivo omnichannelDedupeFile
	if err := json.Unmarshal(dados, &arquivo); err != nil {
		t.Fatalf("o índice precisa ser JSON válido: %v", err)
	}
	if _, existe := arquivo.Entries["org-dedupe|whatsapp|wamid-arquivo"]; !existe {
		t.Fatalf("a chave da mensagem precisa estar no disco: %s", dados)
	}
}
