package multillm

import (
	"strings"
	"testing"
)

// TestJoinProviderURLNaoDuplicaSegmentoDeVersao trava a regra que o gateway jÃ¡ aplicava e que o
// instrumento opt-in havia divergido: BaseURL terminando em /v1 + rota comeÃ§ando com /v1/ deve
// produzir UM Ãºnico segmento de versÃ£o.
func TestJoinProviderURLNaoDuplicaSegmentoDeVersao(t *testing.T) {
	casos := []struct{ base, rota, quer string }{
		{"https://api.groq.com/openai/v1", "/v1/chat/completions", "https://api.groq.com/openai/v1/chat/completions"},
		{"https://api.deepseek.com/v1", "/v1/chat/completions", "https://api.deepseek.com/v1/chat/completions"},
		{"https://api.mistral.ai/v1", "/v1/chat/completions", "https://api.mistral.ai/v1/chat/completions"},
		{"https://openrouter.ai/api/v1", "/v1/chat/completions", "https://openrouter.ai/api/v1/chat/completions"},
		{"https://api.x.ai/v1", "/v1/chat/completions", "https://api.x.ai/v1/chat/completions"},
		{"https://api.openai.com/v1", "/v1/chat/completions", "https://api.openai.com/v1/chat/completions"},
		// Base sem versÃ£o: a rota traz o segmento e ele NÃƒO pode ser removido.
		{"https://api.anthropic.com", "/v1/chat/completions", "https://api.anthropic.com/v1/chat/completions"},
		// Base com a versÃ£o jÃ¡ embutida no fim do caminho (compatibilidade OpenAI do Google):
		// a rota Ã© relativa e nÃ£o repete a versÃ£o.
		{"https://generativelanguage.googleapis.com/v1beta/openai", "/v1/chat/completions", "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"},
		// Base com barra final nÃ£o pode gerar barra dupla.
		{"https://api.groq.com/openai/v1/", "/v1/chat/completions", "https://api.groq.com/openai/v1/chat/completions"},
	}
	for _, caso := range casos {
		obtido := JoinProviderURL(caso.base, caso.rota)
		if obtido != caso.quer {
			t.Fatalf("JoinProviderURL(%q, %q) = %q, quer %q", caso.base, caso.rota, obtido, caso.quer)
		}
		if strings.Contains(obtido, "/v1/v1") || strings.Contains(obtido, "/v1beta/v1beta") {
			t.Fatalf("segmento de versao duplicado em %q", obtido)
		}
	}
}

// TestPresetsOpenAICompatÃ­veisGeramURLFinalSemVersaoDuplicada percorre os PRESETS REAIS: a proteÃ§Ã£o
// Ã© da classe inteira, nÃ£o de um caso escolhido Ã  mÃ£o.
func TestPresetsOpenAICompativeisGeramURLFinalSemVersaoDuplicada(t *testing.T) {
	verificados := 0
	for _, preset := range BuiltInProviderPresets() {
		if preset.Type != ProviderTypeOpenAICompatible || len(preset.Paths) == 0 {
			continue
		}
		if strings.HasPrefix(preset.BaseURL, "http://127.0.0.1") || strings.Contains(preset.BaseURL, ".invalid") {
			continue // locais e o marcador genÃ©rico nÃ£o tÃªm endpoint real para comparar
		}
		for _, rota := range preset.Paths {
			final := JoinProviderURL(preset.BaseURL, rota)
			if !strings.HasPrefix(final, "https://") {
				t.Fatalf("preset %s: URL final nao e HTTPS: %s", preset.ID, final)
			}
			if strings.Contains(final, "/v1/v1") || strings.Contains(final, "//v1") || strings.Contains(final, "/v1beta/v1beta") {
				t.Fatalf("preset %s rota %s: versao duplicada na URL final %s", preset.ID, rota, final)
			}
			// A rota declarada tem de aparecer UMA vez na URL final (sem duplicar o segmento).
			if strings.Count(final, "/v1/") > 1 && strings.Contains(rota, "/v1/") {
				t.Fatalf("preset %s rota %s: /v1/ aparece mais de uma vez em %s", preset.ID, rota, final)
			}
			verificados++
		}
	}
	if verificados == 0 {
		t.Fatal("nenhum preset OpenAI-compatible verificado: o teste perdeu o proposito")
	}
	t.Logf("presets/rotas verificados: %d", verificados)
}
