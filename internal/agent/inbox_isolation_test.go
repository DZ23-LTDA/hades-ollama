package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// TestInboxImpedeAcoesEntreOrganizacoes prova o lado NEGATIVO do isolamento que o teste de escopo não cobria:
// não basta a leitura cruzada falhar — ATRIBUIR, TRANSFERIR e FECHAR a conversa de outra organização também
// precisam ser recusados, e o estado da conversa alheia tem de permanecer INTACTO.
func TestInboxImpedeAcoesEntreOrganizacoes(t *testing.T) {
	raiz := t.TempDir()
	store, err := NewInboxStore(filepath.Join(raiz, "inbox"))
	if err != nil {
		t.Fatalf("abrir inbox: %v", err)
	}
	agora := time.Now().UTC()

	conversaA, _, err := store.AppendInbound("org-a", "whatsapp", "+5511933332222", "Hugo", "preciso de ajuda", "wamid.a1", agora)
	if err != nil {
		t.Fatal(err)
	}

	// AÇÕES CRUZADAS: todas precisam falhar.
	if _, err := store.Assign("org-b", conversaA.ID, "atendente-b", "human", agora); err == nil {
		t.Fatal("organização B NÃO pode atribuir a conversa da organização A")
	}
	if _, _, err := store.HandoffToHuman("org-b", conversaA.ID, "ator-b", "atendente-b", "quero pegar essa conversa", agora); err == nil {
		t.Fatal("organização B NÃO pode transferir a conversa da organização A")
	}
	if _, err := store.Close("org-b", conversaA.ID, "ator-b", "encerrando conversa alheia", agora); err == nil {
		t.Fatal("organização B NÃO pode fechar a conversa da organização A")
	}

	// ESTADO INTACTO: a conversa da organização A continua aberta, sem responsável e com 1 mensagem.
	intacta, mensagens, err := store.Conversation("org-a", conversaA.ID)
	if err != nil {
		t.Fatalf("conversa da organização A sumiu após tentativas cruzadas: %v", err)
	}
	if intacta.Status != conversaA.Status || intacta.AssignedTo != conversaA.AssignedTo {
		t.Fatalf("tentativa cruzada alterou o estado da conversa alheia: antes=%+v depois=%+v", conversaA, intacta)
	}
	if len(mensagens) != 1 || intacta.MessageCount != 1 {
		t.Fatalf("tentativa cruzada alterou as mensagens: %d mensagens, contagem=%d", len(mensagens), intacta.MessageCount)
	}
	t.Logf("ISOLAMENTO: status=%s responsavel=%q mensagens=%d", intacta.Status, intacta.AssignedTo, len(mensagens))
}

// TestInboxDeduplicacaoNaoCruzaOrganizacoes prova que o mesmo identificador externo em organizações DIFERENTES
// gera conversas distintas: deduplicar entre tenants faria uma mensagem LEGÍTIMA de outro cliente ser engolida
// como repetida — perda de mensagem disfarçada de idempotência.
func TestInboxDeduplicacaoNaoCruzaOrganizacoes(t *testing.T) {
	raiz := t.TempDir()
	store, err := NewInboxStore(filepath.Join(raiz, "inbox"))
	if err != nil {
		t.Fatalf("abrir inbox: %v", err)
	}
	agora := time.Now().UTC()

	conversaA, mensagemA, err := store.AppendInbound("org-a", "whatsapp", "+5511933332222", "Hugo", "mensagem da A", "wamid.mesmo-id", agora)
	if err != nil {
		t.Fatal(err)
	}
	conversaB, mensagemB, err := store.AppendInbound("org-b", "whatsapp", "+5511922221111", "Iara", "mensagem da B", "wamid.mesmo-id", agora)
	if err != nil {
		t.Fatal(err)
	}
	if conversaA.ID == conversaB.ID {
		t.Fatalf("organizações diferentes compartilharam a MESMA conversa: %s", conversaA.ID)
	}
	if mensagemB.Deduplicated {
		t.Fatal("mensagem LEGÍTIMA de outra organização foi engolida como repetida")
	}
	if mensagemA.Deduplicated {
		t.Fatal("a primeira mensagem não deveria ser marcada como repetida")
	}
	// A repetição DENTRO da mesma organização continua sendo deduplicada.
	if _, repetida, err := store.AppendInbound("org-a", "whatsapp", "+5511933332222", "Hugo", "mensagem da A", "wamid.mesmo-id", agora.Add(time.Minute)); err != nil {
		t.Fatal(err)
	} else if !repetida.Deduplicated {
		t.Fatal("a repetição dentro da MESMA organização precisa ser deduplicada")
	}
	t.Logf("DEDUPE POR TENANT: conversasA=%d conversasB=%d mesmaId=%v", len(store.ListConversations("org-a", "")), len(store.ListConversations("org-b", "")), conversaA.ID == conversaB.ID)
}

// TestMetricasNaoInferemEstadoDeOutraOrganizacao prova a parte "nem INFERE" do requisito: os indicadores de uma
// organização contam apenas as conversas dela.
func TestMetricasNaoInferemEstadoDeOutraOrganizacao(t *testing.T) {
	raiz := t.TempDir()
	store, err := NewInboxStore(filepath.Join(raiz, "inbox"))
	if err != nil {
		t.Fatalf("abrir inbox: %v", err)
	}
	agora := time.Now().UTC()

	for _, contato := range []string{"+5511911110001", "+5511911110002", "+5511911110003"} {
		if _, _, err := store.AppendInbound("org-a", "whatsapp", contato, "Contato A", "mensagem", "wamid.a-"+contato, agora); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := store.AppendInbound("org-b", "telegram", "@contato-b", "Contato B", "mensagem", "wamid.b-1", agora); err != nil {
		t.Fatal(err)
	}

	metricasA := store.Metrics("org-a", agora)
	metricasB := store.Metrics("org-b", agora)
	if metricasA.TotalConversations != 3 {
		t.Fatalf("métricas da org A deveriam contar 3 conversas, veio %+v", metricasA)
	}
	if metricasB.TotalConversations != 1 {
		t.Fatalf("métricas da org B deveriam contar 1 conversa (sem inferir a A), veio %+v", metricasB)
	}
	if metricasA.TotalConversations == metricasB.TotalConversations {
		t.Fatal("métricas iguais para organizações com volumes diferentes: indicador não está escopado")
	}
	t.Logf("NÃO INFERÊNCIA: orgA=%d orgB=%d", metricasA.TotalConversations, metricasB.TotalConversations)
}
