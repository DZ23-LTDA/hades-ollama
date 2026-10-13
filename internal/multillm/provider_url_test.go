package multillm

import (
	"strings"
	"testing"
)

// TestJoinProviderURLNaoDuplicaSegmentoDeVersao trava a regra que o gateway já aplicava e que o
// instrumento opt-in havia divergido: BaseURL terminando em /v1 + rota começando com /v1/ deve
// produzir UM único segmento de versão.
func TestJoinProviderURLNaoDuplicaSegmentoDeVersao(t *testing.T) {
	casos := []struct{ base, rota, quer string }{
		{"https://api.groq.com/openai/v1", "/v1/chat/completions", "https://api.groq.com/openai/v1/chat/completions"},
		{"https://api.deepseek.com/v1", "/v1/chat/completions", "https://api.deepseek.com/v1/chat/completions"},
		{"https://api.mistral.ai/v1", "/v1/chat/completions", "https://api.mistral.ai/v1/chat/completions"},
		{"https://openrouter.ai/api/v1", "/v1/chat/completions", "https://openrouter.ai/api/v1/chat/completions"},
		{"https://api.x.ai/v1", "/v1/chat/completions", "https://api.x.ai/v1/chat/completions"},
		{"https://api.openai.com/v1", "/v1/chat/completions", "https://api.openai.com/v1/chat/completions"},
		// Base sem versão: a rota traz o segmento e ele NÃO pode ser removido.
		{"https://api.anthropic.com", "/v1/chat/completions", "https://api.anthropic.com/v1/chat/completions"},
		// Base com a versão já embutida no fim do caminho (compatibilidade OpenAI do Google):
		// a rota é relativa e não repete a versão.
		{"https://generativelanguage.googleapis.com/v1beta/openai", "/v1/chat/completions", "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"},
		// Base com barra final não pode gerar barra dupla.
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

// TestPresetsOpenAICompatíveisGeramURLFinalSemVersaoDuplicada percorre os PRESETS REAIS: a proteção
// é da classe inteira, não de um caso escolhido à mão.
func TestPresetsOpenAICompativeisGeramURLFinalSemVersaoDuplicada(t *testing.T) {
	verificados := 0
	for _, preset := range BuiltInProviderPresets() {
		if preset.Type != ProviderTypeOpenAICompatible || len(preset.Paths) == 0 {
			continue
		}
		if strings.HasPrefix(preset.BaseURL, "http://127.0.0.1") || strings.Contains(preset.BaseURL, ".invalid") {
			continue // locais e o marcador genérico não têm endpoint real para comparar
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
