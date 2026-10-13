package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Estágio 34 / E-299 — A ORDENAÇÃO SOBREVIVE AO REINÍCIO.
//
// Limite declarado em E-298: o estado de ordem vivia só em memória, então um reinício esquecia os instantes e o
// PRIMEIRO evento de cada conversa voltava a parecer em ordem — exatamente o caso que a marcação existe para pegar.
// Mesma classe que a dedupe tinha antes de E-172, e por isso o mesmo padrão de store.
//
// Propriedades travadas:
//  1. depois de um reinício REAL (runtime novo, mesmo store), um evento ATRASADO é marcado;
//  2. sem store, o runtime DIZ que não há persistência (não finge);
//  3. arquivo corrompido é REPORTADO, não engolido, e o runtime continua funcionando;
//  4. a poda por teto é determinística e mantém as linhas do tempo mais recentes.
func TestOrdenacaoDoOmnichannelSobreviveAoReinicio(t *testing.T) {
	raiz := t.TempDir()
	store := NewOmnichannelOrderingStore(omnichannelOrderingRoot(raiz))
	if store.Root() == "" {
		t.Fatal("o store precisa resolver um diretório")
	}
	const segredo = "segredo-de-teste-com-tamanho-suficiente"

	enviar := func(runtime *OmnichannelRuntime, id string, instante time.Time, agora time.Time) OmnichannelEvent {
		t.Helper()
		corpo := []byte(fmt.Sprintf(`{"external_message_id":%q}`, id))
		assinatura := omnichannelSignature(segredo, instante.Unix(), corpo)
		evento := omnichannelEvent(t)
		evento.ExternalMessageID = id
		evento.ConversationID = "conv-persistente"
		aceito, err := runtime.Ingest("org-a", "whatsapp", assinatura, instante.Unix(), corpo, evento, agora)
		if err != nil {
			t.Fatalf("ingest %s: %v", id, err)
		}
		return aceito
	}

	agora := time.Now().UTC()

	// Primeira vida do processo: registra um instante recente.
	primeiro, _, _ := omnichannelRuntime(t)
	primeiro.SetOrderingStore(store)
	if !primeiro.OrderingPersisted() {
		t.Fatal("com store ligado, OrderingPersisted precisa ser true")
	}
	if e := enviar(primeiro, "p1", agora.Add(-20*time.Second), agora); e.OutOfOrder {
		t.Fatalf("primeiro evento em ordem não pode ser marcado: %+v", e)
	}
	if erro := primeiro.LastOrderingError(); erro != "" {
		t.Fatalf("gravação normal não pode registrar erro: %q", erro)
	}

	// REINÍCIO REAL: runtime NOVO, mesmo store, mesma pasta de dados.
	segundo, _, _ := omnichannelRuntime(t)
	segundo.SetOrderingStore(NewOmnichannelOrderingStore(omnichannelOrderingRoot(raiz)))

	// 1. Um evento ATRASADO em relação ao que a vida anterior registrou precisa ser MARCADO.
	atrasado := enviar(segundo, "p2", agora.Add(-50*time.Second), agora)
	if !atrasado.OutOfOrder {
		t.Fatalf("após reinício, evento atrasado precisa ser marcado (a ordem precisa sobreviver): %+v", atrasado)
	}
	// E um evento em ordem depois do reinício NÃO é marcado.
	if e := enviar(segundo, "p3", agora.Add(-10*time.Second), agora); e.OutOfOrder {
		t.Fatalf("evento em ordem não pode ser marcado: %+v", e)
	}

	// 2. Sem store, o runtime DIZ que não há persistência.
	semStore, _, _ := omnichannelRuntime(t)
	if semStore.OrderingPersisted() {
		t.Fatal("sem store, OrderingPersisted precisa ser false — em memória é dito, não fingido")
	}

	// 3. Arquivo corrompido é REPORTADO e o runtime continua.
	corrompido := NewOmnichannelOrderingStore(omnichannelOrderingRoot(t.TempDir()))
	if err := os.MkdirAll(corrompido.Root(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corrompido.Root(), "org-a.json"), []byte("{isso nao e json"), 0o600); err != nil {
		t.Fatal(err)
	}
	terceiro, _, _ := omnichannelRuntime(t)
	terceiro.SetOrderingStore(corrompido)
	if e := enviar(terceiro, "p4", agora.Add(-5*time.Second), agora); e.OutOfOrder {
		t.Fatalf("com arquivo corrompido o runtime segue e não inventa ordem: %+v", e)
	}
	if erro := terceiro.LastOrderingError(); erro == "" {
		t.Fatal("arquivo corrompido precisa ser REPORTADO, não engolido")
	}

	// 4. Poda determinística: mantém as mais recentes, empate desempatado pela chave.
	heads := map[string]int64{"a": 10, "b": 30, "c": 20, "d": 30}
	podado := podarOrdering(heads, 2)
	if len(podado) != 2 {
		t.Fatalf("poda deveria manter 2, manteve %d", len(podado))
	}
	if _, temos := podado["d"]; !temos {
		t.Fatalf("empate de instante precisa ser desempatado pela chave de forma determinística: %+v", podado)
	}
	if _, temos := podado["b"]; !temos {
		t.Fatalf("as mais recentes precisam sobreviver: %+v", podado)
	}
	if novamente := podarOrdering(heads, 2); len(novamente) != 2 {
		t.Fatalf("poda precisa ser determinística: %+v", novamente)
	}
}
