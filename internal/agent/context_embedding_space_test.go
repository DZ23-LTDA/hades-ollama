package agent

import (
	"context"
	"strings"
	"testing"
)

// ESPAÇO VETORIAL: trocar o modelo de embedding não pode virar ruído plausível (estágio 35, estágio 15).
//
// O limite declarado era: "trocar o modelo de embedding não re-embeda o histórico, sem backfill". O risco
// concreto não é a ausência de recurso — é a busca comparar vetores de espaços DIFERENTES e devolver um
// resultado que parece bom. Estes testes travam:
//
//  1. vetor carimbado com OUTRO modelo não é comparado semanticamente (e a resposta DECLARA quantos ficaram);
//  2. `ReembedMemories` traz esse conhecimento de volta ao espaço atual, e a busca semântica volta a achá-lo;
//  3. vetor SEM carimbo (embedder que não se identifica) também não é comparado — não há como saber o espaço;
//  4. falha de embedding em uma memória não impede o re-embed das outras.
func TestServiceAgentKnowledgeSearchDeclaresMemoriesOutsideVectorSpace(t *testing.T) {
	runtime, store, conversationID, now := knowledgeFixture(t)
	// Memória gravada no espaço ANTIGO.
	antigo := stubEmbedder{modelo: "modelo-antigo", vetores: map[string][]float32{"troca": {1, 0, 0}}}
	store.SetEmbedder(antigo)
	project := addKnowledge(t, store, "Base antiga", "org-a", "aceitamos troca em 30 dias")

	// O modelo muda. A consulta agora é no espaço NOVO, e o vetor antigo NÃO é comparável.
	novo := stubEmbedder{modelo: "modelo-novo", vetores: map[string][]float32{
		"troca": {1, 0, 0},
		"prazo": {1, 0, 0}, // mesma direção, mas o CARIMBO é que decide a comparabilidade
	}}
	store.SetEmbedder(novo)
	runtime.SetKnowledge(store)

	if fora := store.CountMemoriesOutsideVectorSpace(project.ID); fora != 1 {
		t.Fatalf("a memória está em outro espaço e precisa ser contada: %d", fora)
	}
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "qual o prazo de troca?", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("a busca precisa executar: %+v", turn.ToolCalls)
	}
	// "prazo de troca" casa literalmente com o conteúdo, mas o essencial é a DECLARAÇÃO do espaço.
	if !strings.Contains(call.Summary, "fora do espaço vetorial atual") {
		t.Fatalf("a resposta precisa declarar o conhecimento fora do espaço: %q", call.Summary)
	}
}

func TestServiceAgentKnowledgeReembedBringsMemoriesBack(t *testing.T) {
	runtime, store, conversationID, now := knowledgeFixture(t)
	antigo := stubEmbedder{modelo: "modelo-antigo", vetores: map[string][]float32{"troca": {1, 0, 0}}}
	store.SetEmbedder(antigo)
	project := addKnowledge(t, store, "Base antiga", "org-a", "aceitamos troca em 30 dias")

	// Depois do re-embed, o vetor é recalculado no espaço atual e o carimbo acompanha.
	novo := stubEmbedder{modelo: "modelo-novo", vetores: map[string][]float32{
		"troca":    {1, 0, 0},
		"prazo":    {1, 0, 0},
		"devolver": {1, 0, 0},
	}}
	store.SetEmbedder(novo)
	resultado, err := store.ReembedMemories(context.Background(), project.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Reembedded != 1 || resultado.Failed != 0 {
		t.Fatalf("o re-embed precisa recalcular a memória antiga: %+v", resultado)
	}
	if fora := store.CountMemoriesOutsideVectorSpace(project.ID); fora != 0 {
		t.Fatalf("depois do re-embed nada pode ficar fora do espaço: %d", fora)
	}
	// E a busca volta a ser SEMÂNTICA: consulta sem palavra em comum encontra o conteúdo.
	runtime.SetKnowledge(store)
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "prazo para devolver produto", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || !strings.Contains(call.Summary, "aceitamos troca") {
		t.Fatalf("depois do re-embed a busca semântica precisa achar: %+v", turn.ToolCalls)
	}
	if strings.Contains(call.Summary, "fora do espaço vetorial") {
		t.Fatalf("sem memória fora do espaço não há o que declarar: %q", call.Summary)
	}
}

func TestServiceAgentKnowledgeUnidentifiedEmbedderIsStillComparable(t *testing.T) {
	runtime, store, conversationID, now := knowledgeFixture(t)
	// Embedder SEM identidade: não há como distinguir espaços, então a comparação CONTINUA — e isso é
	// declarado. Recusar aqui quebraria em SILÊNCIO uma busca semântica que já funcionava (foi o que a
	// suíte completa pegou: `TestSemanticMemorySearchRanksByCosineSimilarity`).
	semNome := stubEmbedder{vetores: map[string][]float32{"troca": {1, 0, 0}, "prazo": {1, 0, 0}}}
	store.SetEmbedder(semNome)
	project := addKnowledge(t, store, "Base sem nome", "org-a", "aceitamos troca em 30 dias")

	if carimbadas := store.CountMemoriesOutsideVectorSpace(project.ID); carimbadas != 0 {
		t.Fatalf("sem identidade não há espaço a declarar: %d", carimbadas)
	}
	memorias, err := store.SearchMemoriesContext(context.Background(), project.ID, "prazo para devolver produto", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(memorias) != 1 {
		t.Fatalf("sem identidade a comparação semântica segue valendo: %+v", memorias)
	}
	// E a resposta não inventa aviso de espaço: não há espaço identificado para comparar.
	runtime.SetKnowledge(store)
	turn, err := runtime.RunTurn(ServiceAgentTurnRequest{
		OrganizationID: "org-a", AgentID: ServiceAgentSupport, ConversationID: conversationID,
		Intent: "prazo para devolver produto", RequestedTools: []string{"knowledge.search"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	call := callByName(turn.ToolCalls, "knowledge.search")
	if call == nil || !strings.Contains(call.Summary, "aceitamos troca") {
		t.Fatalf("a busca precisa achar: %+v", turn.ToolCalls)
	}
	if strings.Contains(call.Summary, "fora do espaço vetorial") {
		t.Fatalf("sem carimbo não há espaço a declarar: %q", call.Summary)
	}
}

func TestServiceAgentReembedKeepsGoingWhenOneMemoryFails(t *testing.T) {
	store, err := NewContextStore(t.TempDir() + "-reembed")
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("Base", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	// Duas memórias gravadas sem embedder (sem vetor).
	for _, conteudo := range []string{"texto que falha", "texto que funciona"} {
		if _, err := store.AddMemory(Memory{ProjectID: project.ID, Kind: "fact", Content: conteudo, Source: "manual"}); err != nil {
			t.Fatal(err)
		}
	}
	// Embedder que falha só no texto problemático.
	parcial := stubParcialEmbedder{}
	store.SetEmbedder(parcial)
	resultado, err := store.ReembedMemories(context.Background(), project.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Failed != 1 || resultado.Reembedded != 1 {
		t.Fatalf("uma falha não pode impedir as outras: %+v", resultado)
	}
}

// stubParcialEmbedder falha em um texto específico e funciona nos demais.
type stubParcialEmbedder struct{}

func (stubParcialEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if strings.Contains(strings.ToLower(text), "falha") {
		return nil, ErrEmbedderInvalid
	}
	return []float32{1, 0, 0}, nil
}

func (stubParcialEmbedder) EmbeddingModel() string { return "modelo-parcial" }
