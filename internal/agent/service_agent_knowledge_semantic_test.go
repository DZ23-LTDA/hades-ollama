package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// BUSCA SEMÂNTICA e RESERVA DECLARADA (estágio 35).
//
// O limite declarado era: "a busca é LEXICAL por padrão; a semântica exige embedder, e não a declaramos
// ativa". O embedder JÁ é configurável (`OLLAMA_AGENT_EMBED_MODEL`, ligado em `server/agent_routes.go`), e o
// defeito real que estes testes travam é outro, mais grave que a ausência: **degradação silenciosa**.
//
//  1. com embedder, a busca encontra memória SEM NENHUMA palavra em comum com a consulta (é semântica de
//     verdade, não substring disfarçada);
//  2. com embedder CONFIGURADO mas indisponível, a busca cai para literal e a resposta DIZ que caiu —
//     antes disso, `SearchMemories` engolia o erro e a resposta era "nenhuma memória encontrada";
//  3. resposta/configuração INVÁLIDA não vira reserva silenciosa: a ferramenta FALHA e declara o motivo.

// stubEmbedder mapeia texto a vetor por palavra-chave: é um modelo de brinquedo determinístico, e é o
// bastante para provar QUAL caminho de busca rodou.
type stubEmbedder struct {
	vetores map[string][]float32
	err     error
	// modelo identifica o ESPAÇO VETORIAL. Vazio significa "não se identifica": os vetores saem SEM carimbo
	// e a busca recusa compará-los, porque não há como saber de que espaço são.
	modelo string
}

// EmbeddingModel diz o espaço vetorial deste embedder.
func (s stubEmbedder) EmbeddingModel() string { return s.modelo }

func (s stubEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	minusculo := strings.ToLower(text)
	for chave, vetor := range s.vetores {
		if strings.Contains(minusculo, chave) {
			return vetor, nil
		}
	}
	// Vetor ortogonal: não se parece com nada do conjunto.
	return []float32{0, 0, 1}, nil
}

// semanticKnowledgeFixture liga o embedder ANTES de gravar a memória: é na escrita que o vetor nasce.
func semanticKnowledgeFixture(t *testing.T, embedder Embedder) (*ServiceAgentRuntime, *ContextStore, string, time.Time) {
	t.Helper()
	runtime, context, conversationID, now := knowledgeFixture(t)
	if embedder != nil {
		context.SetEmbedder(embedder)
	}
	return runtime, context, conversationID, now
}

func TestServiceAgentKnowledgeSearchIsSemanticWhenEmbedderIsConfigured(t *testing.T) {
	// A memória fala de "troca"; a consulta fala de "prazo"/"devolver". NENHUMA palavra em comum: se a
	// resposta trouxer a memória, foi o vetor — busca literal não teria como encontrar.
	embedder := stubEmbedder{modelo: "modelo-troca-v1", vetores: map[string][]float32{
		"troca":    {1, 0, 0},
		"prazo":    {1, 0, 0},
		"devolver": {1, 0, 0},
	}}
	runtime, context, conversationID, now := semanticKnowledgeFixture(t, embedder)
	addKnowledge(t, context, "Base semântica", "org-a", "aceitamos troca em até 30 dias")
	runtime.SetKnowledge(context)

	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "prazo para devolver produto", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("a busca precisa executar: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Summary, "aceitamos troca") {
		t.Fatalf("a busca SEMÂNTICA precisa achar memória sem palavra em comum: %q", call.Summary)
	}
	if strings.Contains(call.Summary, "busca LITERAL") {
		t.Fatalf("com embedder funcionando não pode haver reserva: %q", call.Summary)
	}
}

func TestServiceAgentKnowledgeSearchDeclaresLiteralFallbackWhenEmbedderIsDown(t *testing.T) {
	// Embedder configurado e CAINDO na hora da CONSULTA: a memória é gravada antes, porque com embedder
	// quebrado na ESCRITA o próprio AddMemory falha (comportamento correto do motor).
	runtime, context, conversationID, now := knowledgeFixture(t)
	addKnowledge(t, context, "Base literal", "org-a", "política de troca em até 30 dias")
	context.SetEmbedder(stubEmbedder{err: ErrEmbedderUnavailable})
	runtime.SetKnowledge(context)

	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "política de troca", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("a reserva literal precisa manter a busca funcionando: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Summary, "política de troca") {
		t.Fatalf("a reserva literal encontra por correspondência: %q", call.Summary)
	}
	// O que NÃO pode acontecer é a reserva ser invisível: quem lê precisa saber que o caminho semântico
	// não rodou nesta resposta.
	if !strings.Contains(call.Summary, "busca LITERAL") || !strings.Contains(call.Summary, "embedder indisponível") {
		t.Fatalf("a reserva precisa ser DECLARADA: %q", call.Summary)
	}
}

func TestServiceAgentKnowledgeSearchFailsOnInvalidEmbedderResponse(t *testing.T) {
	// Resposta inválida é DEFEITO, não indisponibilidade: não pode virar reserva silenciosa.
	runtime, context, conversationID, now := knowledgeFixture(t)
	addKnowledge(t, context, "Base quebrada", "org-a", "política de troca em até 30 dias")
	context.SetEmbedder(stubEmbedder{err: ErrEmbedderInvalid})
	runtime.SetKnowledge(context)

	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "política de troca", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || call.Status != ServiceAgentToolFailed {
		t.Fatalf("embedder inválido precisa FALHAR a busca: %+v", turn.ToolCalls)
	}
	if !strings.Contains(call.Reason, "resposta inesperada") {
		t.Fatalf("a falha precisa dizer a classe: %q", call.Reason)
	}
	if strings.Contains(call.Summary, "troca") {
		t.Fatalf("falha não pode devolver conteúdo como se tivesse buscado: %q", call.Summary)
	}
}

func TestServiceAgentKnowledgeSearchWithoutEmbedderHasNoFallbackMarker(t *testing.T) {
	// Sem embedder NÃO há reserva: a busca literal é o comportamento normal, e a resposta não deve sugerir
	// que algo caiu.
	runtime, context, conversationID, now := semanticKnowledgeFixture(t, nil)
	addKnowledge(t, context, "Base padrão", "org-a", "política de troca em até 30 dias")
	runtime.SetKnowledge(context)

	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "política de troca", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("sem embedder a busca literal precisa funcionar: %+v", turn.ToolCalls)
	}
	if strings.Contains(call.Summary, "embedder indisponível") {
		t.Fatalf("sem embedder configurado não há reserva a declarar: %q", call.Summary)
	}
}
