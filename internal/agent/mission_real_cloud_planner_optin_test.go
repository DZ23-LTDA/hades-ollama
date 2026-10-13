package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/internal/multillm"
)

// planejadorNuvemOpenAI é um adaptador de TESTE que fala OpenAI-compatible com o provedor real do
// preset. Existe porque o caminho de produção monta o planejador contra o Ollama local — este
// instrumento prova o que falta: um MODELO DE NUVEM planejando e executando dentro do runtime.
type planejadorNuvemOpenAI struct {
	endpoint    string
	chave       string
	modelo      string
	cliente     *http.Client
	limiteSaida int
}

func (p planejadorNuvemOpenAI) Chat(ctx context.Context, request *api.ChatRequest, callback api.ChatResponseFunc) error {
	mensagens := make([]map[string]string, 0, len(request.Messages))
	for _, m := range request.Messages {
		mensagens = append(mensagens, map[string]string{"role": m.Role, "content": m.Content})
	}
	corpo := map[string]any{
		"model":      p.modelo,
		"messages":   mensagens,
		"max_tokens": p.limiteSaida,
		"stream":     false,
	}
	bruto, err := json.Marshal(corpo)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(bruto))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.chave)
	resp, err := p.cliente.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	conteudo, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("provedor respondeu HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(conteudo)))
	}
	var decodificado struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(conteudo, &decodificado); err != nil {
		return fmt.Errorf("resposta nao decodificavel: %w", err)
	}
	if len(decodificado.Choices) == 0 || strings.TrimSpace(decodificado.Choices[0].Message.Content) == "" {
		return fmt.Errorf("provedor respondeu sem conteudo utilizavel")
	}
	if decodificado.Usage.PromptTokens == 0 && decodificado.Usage.CompletionTokens == 0 {
		return fmt.Errorf("provedor respondeu sem contagem de tokens: nao conta como consumo real")
	}
	// O consumo REAL do provedor entra pelo mesmo caminho do produto: api.Metrics alimenta
	// UsageFromMetrics, que o runtime grava em Mission.Usage.
	return callback(api.ChatResponse{
		Model:   p.modelo,
		Message: api.Message{Role: "assistant", Content: decodificado.Choices[0].Message.Content},
		Done:    true,
		Metrics: api.Metrics{
			PromptEvalCount: decodificado.Usage.PromptTokens,
			EvalCount:       decodificado.Usage.CompletionTokens,
		},
	})
}

// TestMissaoRealComPlanejadorDeNuvemEhOptIn roda uma MISSÃO DE VERDADE cujo planejamento vem de um
// MODELO DE NUVEM real. É opt-in: sem OLLAMA_AGENT_REAL_PROVIDER e OLLAMA_AGENT_REAL_MODEL o teste
// NÃO roda. Com as variáveis ele FALHA ALTO — nunca finge sucesso nem esconde o erro do provedor.
func TestMissaoRealComPlanejadorDeNuvemEhOptIn(t *testing.T) {
	provedor := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REAL_PROVIDER"))
	modelo := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REAL_MODEL"))
	if provedor == "" || modelo == "" {
		t.Skip("BLOCKED_BY_EXTERNAL_DEPENDENCY: defina OLLAMA_AGENT_REAL_PROVIDER e OLLAMA_AGENT_REAL_MODEL")
	}
	preset, ok := multillm.ProviderPresetByID(provedor)
	if !ok {
		t.Skipf("preset %q desconhecido", provedor)
	}
	caminho := ""
	for _, p := range preset.Paths {
		if strings.Contains(p, "chat") {
			caminho = p
			break
		}
	}
	if caminho == "" {
		t.Skipf("preset %q não declara caminho de chat", provedor)
	}
	chave := multillm.CredentialValue(preset.APIKeyEnv)
	if strings.TrimSpace(chave) == "" {
		t.Skipf("BLOCKED_BY_EXTERNAL_DEPENDENCY: %s ausente", preset.APIKeyEnv)
	}
	endpoint := multillm.JoinProviderURL(preset.BaseURL, caminho)
	t.Logf("missão com nuvem: provedor=%s modelo=%s endpoint=%s (chave não impressa)", provedor, modelo, endpoint)

	raiz := t.TempDir()
	store, err := NewJSONStore(filepath.Join(raiz, ".store"))
	if err != nil {
		t.Fatal(err)
	}
	planejador := &OllamaPlanner{
		Client: planejadorNuvemOpenAI{
			endpoint:    endpoint,
			chave:       chave,
			modelo:      modelo,
			cliente:     &http.Client{Timeout: 90 * time.Second},
			limiteSaida: 2048,
		},
		Model: modelo,
		// Tarifa DECLARADA para o teste (o catálogo local não traz preço destes modelos).
		Price: ModelPrice{InputCentsPer1K: 14, OutputCentsPer1K: 28, Known: true},
	}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:           store,
		Planner:         planejador,
		PlannerResolver: resolvedorFixo{provider: provedor, planner: planejador},
		WorkspaceRoot:   raiz,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	missao, err := runtime.CreateMission(ctx, CreateMissionRequest{
		Provider:  provedor,
		Model:     modelo,
		Objective: "listar os arquivos do workspace e informar quantos arquivos existem",
	})
	if err != nil {
		t.Fatalf("criar missão: %v", err)
	}

	// O modelo de nuvem decide o plano; se exigir aprovação, o instrumento aprova e continua.
	for tentativa := range 5 {
		if err := runtime.Run(ctx, missao.ID); err != nil {
			t.Fatalf("executar missão (tentativa %d): %v", tentativa+1, err)
		}
		atual, err := runtime.GetMission(missao.ID)
		if err != nil {
			t.Fatalf("ler missão: %v", err)
		}
		missao = atual
		if missao.CompletedAt != nil {
			break
		}
		aprovou := false
		for _, a := range missao.Approvals {
			if strings.EqualFold(string(a.Status), "pending") {
				if _, err := runtime.DecideApproval(missao.ID, a.ID, true, "aprovado pelo instrumento de nuvem"); err != nil {
					t.Fatalf("aprovar etapa %s: %v", a.StepID, err)
				}
				aprovou = true
				break
			}
		}
		if !aprovou {
			break
		}
	}

	if missao.CompletedAt == nil {
		t.Fatalf("missão com modelo de nuvem NÃO concluiu: estado=%s etapas=%d", missao.State, len(missao.Plan))
	}
	if missao.Usage == nil {
		t.Fatal("missão concluiu sem registrar consumo: tokens reais do provedor se perderam")
	}
	if missao.Usage.Provider != provedor || missao.Usage.Model != modelo {
		t.Fatalf("consumo atribuÃÂdo a provedor/modelo errado: %+v", *missao.Usage)
	}
	if missao.Usage.PromptTokens <= 0 || missao.Usage.CompletionTokens <= 0 {
		t.Fatalf("consumo sem tokens reais: %+v", *missao.Usage)
	}
	if !missao.Usage.CostKnown {
		t.Fatalf("tarifa declarada deveria produzir custo conhecido: %+v", *missao.Usage)
	}
	t.Logf("MISSÃO COM NUVEM: provedor=%s modelo=%s estado=%s etapas=%d tokens=%d/%d custo_centavos=%d",
		missao.Usage.Provider, missao.Usage.Model, missao.State, len(missao.Plan),
		missao.Usage.PromptTokens, missao.Usage.CompletionTokens, missao.Usage.CostCents)
}
