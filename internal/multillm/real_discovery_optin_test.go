package multillm

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestDescobertaRealDeModelosEhOptIn exercita a DESCOBERTA REAL de modelos pelo mesmo caminho do
// produto (`ListUpstreamModels`, que monta `BaseURL + /models` e aplica a autenticação do preset).
//
// É OPT-IN: sem OLLAMA_AGENT_REAL_PROVIDER o teste NÃO roda, para a suíte seguir hermética. Com a
// variável definida ele FALHA ALTO com o erro real do provedor — nunca finge sucesso. O log traz
// provedor, quantidade e alguns nomes de modelo; NUNCA a chave.
func TestDescobertaRealDeModelosEhOptIn(t *testing.T) {
	id := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REAL_PROVIDER"))
	if id == "" {
		t.Skip("sem OLLAMA_AGENT_REAL_PROVIDER: a descoberta real usa conta do operador")
	}
	preset, ok := ProviderPresetByID(id)
	if !ok {
		t.Skipf("preset %q desconhecido", id)
	}
	if strings.HasPrefix(preset.BaseURL, "http://127.0.0.1") || strings.HasPrefix(preset.BaseURL, "http://localhost") {
		t.Skipf("preset %q é local: não é chamada externa", id)
	}
	if CredentialValue(preset.APIKeyEnv) == "" {
		t.Skipf("BLOCKED_BY_EXTERNAL_DEPENDENCY: %s ausente para o preset %q", preset.APIKeyEnv, id)
	}
	t.Logf("preset: id=%s base=%s chave_em=%q (valor não impresso)", preset.ID, preset.BaseURL, preset.APIKeyEnv)

	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelar()
	provedor, err := preset.Provider(nil)
	if err != nil {
		t.Fatalf("preset %q nao pode virar provider: %v", id, err)
	}
	modelos, err := ListUpstreamModels(ctx, provedor, &http.Client{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("descoberta real do provedor %q falhou: %v", id, err)
	}
	if len(modelos) == 0 {
		t.Fatalf("provedor %q respondeu sem modelos", id)
	}
	amostra := modelos
	if len(amostra) > 8 {
		amostra = amostra[:8]
	}
	t.Logf("DESCOBERTA REAL: provedor=%s modelos=%d amostra=%v", id, len(modelos), amostra)
}
