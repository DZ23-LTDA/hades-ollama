package multillm

import (
	"sort"
	"strings"
)

// Presets guiados de provedor (estÃ¡gio 4 do Hades).
//
// O gateway Ã© orientado por TIPO de protocolo (`openai-compatible`,
// `anthropic`, `cli`) e os provedores vÃªm de configuraÃ§Ã£o JSON. Estes presets
// existem para o cadastro guiado em pt-BR: cada um entrega `base_url`, nome da
// variÃ¡vel de ambiente da credencial, estilo de autenticaÃ§Ã£o e caminhos
// suportados â€” nunca a credencial em si.
//
// Regra de honestidade embutida: um preset nÃ£o declara modelo nem capacidade.
// As capacidades de cada modelo continuam vindo de `ProbeProviderModel` /
// descoberta real, e a chave continua vindo do ambiente ou do cofre, nunca
// deste pacote.
//
// Cobertura: Ollama, vLLM, llama.cpp, LM Studio (servidores locais) e OpenAI,
// Anthropic, Gemini, DeepSeek, Groq, Mistral, OpenRouter e xAI (APIs). Outros
// servidores compatÃ­veis com OpenAI entram pelo preset genÃ©rico, cuja
// `base_url` precisa ser editada pelo operador.
type ProviderPreset struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	BaseURL     string   `json:"base_url"`
	APIKeyEnv   string   `json:"api_key_env,omitempty"`
	AuthStyle   string   `json:"auth_style,omitempty"`
	Paths       []string `json:"paths"`
	Local       bool     `json:"local"`
	RequiresKey bool     `json:"requires_key"`
	DocsURL     string   `json:"docs_url,omitempty"`
	Notes       string   `json:"notes,omitempty"`
}

// GenericPresetBaseURL Ã© um marcador reservado (TLD `.invalid`): o operador
// precisa substituÃ­-lo ao cadastrar um servidor compatÃ­vel com OpenAI que nÃ£o
// esteja na lista. Servir um host plausÃ­vel seria pior, porque pareceria
// funcional.
const GenericPresetBaseURL = "https://api.exemplo.invalid/v1"

const (
	// GenericPresetEnvName Ã© o nome da variÃ¡vel de ambiente esperada pelo
	// preset genÃ©rico; o operador pode trocÃ¡-lo na configuraÃ§Ã£o.
	GenericPresetEnvName = "PROVEDOR_COMPATIVEL_API_KEY"
)

// BuiltInProviderPresets devolve a lista canÃ´nica de presets, ordenada por ID.
func BuiltInProviderPresets() []ProviderPreset {
	presets := []ProviderPreset{
		{
			ID:          "ollama",
			Name:        "Ollama (local)",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "http://127.0.0.1:11434",
			Paths:       []string{"/api/chat", "/api/generate", "/v1/chat/completions", "/v1/embeddings"},
			Local:       true,
			RequiresKey: false,
			DocsURL:     "https://docs.ollama.com/api",
			Notes:       "Motor local padrÃ£o do Hades; nÃ£o exige credencial.",
		},
		{
			ID:          "vllm",
			Name:        "vLLM (local)",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "http://127.0.0.1:8000/v1",
			Paths:       []string{"/v1/chat/completions", "/v1/completions", "/v1/embeddings"},
			Local:       true,
			RequiresKey: false,
			Notes:       "Servidor OpenAI-compatible; ajuste a porta se o operador tiver mudado.",
		},
		{
			ID:          "llamacpp",
			Name:        "llama.cpp (llama-server local)",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "http://127.0.0.1:8080/v1",
			Paths:       []string{"/v1/chat/completions", "/v1/completions", "/v1/embeddings"},
			Local:       true,
			RequiresKey: false,
			Notes:       "Porta padrÃ£o do llama-server.",
		},
		{
			ID:          "lmstudio",
			Name:        "LM Studio (local)",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "http://127.0.0.1:1234/v1",
			Paths:       []string{"/v1/chat/completions", "/v1/completions", "/v1/embeddings"},
			Local:       true,
			RequiresKey: false,
			Notes:       "Porta padrÃ£o do servidor local do LM Studio.",
		},
		{
			ID:          "openai",
			Name:        "OpenAI",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "https://api.openai.com/v1",
			APIKeyEnv:   "OPENAI_API_KEY",
			AuthStyle:   AuthStyleBearer,
			Paths:       []string{"/v1/chat/completions", "/v1/responses", "/v1/embeddings"},
			RequiresKey: true,
		},
		{
			ID:          "anthropic",
			Name:        "Anthropic",
			Type:        ProviderTypeAnthropic,
			BaseURL:     "https://api.anthropic.com",
			APIKeyEnv:   "ANTHROPIC_API_KEY",
			AuthStyle:   AuthStyleAnthropic,
			Paths:       []string{"/v1/messages"},
			RequiresKey: true,
		},
		{
			ID:        "gemini",
			Name:      "Google Gemini (endpoint compatÃ­vel com OpenAI)",
			Type:      ProviderTypeOpenAICompatible,
			BaseURL:   "https://generativelanguage.googleapis.com/v1beta/openai",
			APIKeyEnv: "GEMINI_API_KEY",
			AuthStyle: AuthStyleBearer,
			// A camada de compatibilidade do Google jÃ¡ inclui a versÃ£o no base (.../v1beta/openai),
			// entÃ£o a rota Ã© relativa: /chat/completions, nÃ£o /v1/chat/completions.
			Paths:       []string{"/v1/chat/completions", "/v1/embeddings"},
			RequiresKey: true,
			Notes:       "Usa a camada de compatibilidade oficial; o protocolo nativo nÃ£o Ã© usado aqui.",
		},
		{
			ID:          "deepseek",
			Name:        "DeepSeek",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "https://api.deepseek.com/v1",
			APIKeyEnv:   "DEEPSEEK_API_KEY",
			AuthStyle:   AuthStyleBearer,
			Paths:       []string{"/v1/chat/completions", "/v1/completions"},
			RequiresKey: true,
		},
		{
			ID:          "groq",
			Name:        "Groq",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "https://api.groq.com/openai/v1",
			APIKeyEnv:   "GROQ_API_KEY",
			AuthStyle:   AuthStyleBearer,
			Paths:       []string{"/v1/chat/completions", "/v1/completions"},
			RequiresKey: true,
		},
		{
			ID:          "mistral",
			Name:        "Mistral",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "https://api.mistral.ai/v1",
			APIKeyEnv:   "MISTRAL_API_KEY",
			AuthStyle:   AuthStyleBearer,
			Paths:       []string{"/v1/chat/completions", "/v1/embeddings"},
			RequiresKey: true,
		},
		{
			ID:          "openrouter",
			Name:        "OpenRouter",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "https://openrouter.ai/api/v1",
			APIKeyEnv:   "OPENROUTER_API_KEY",
			AuthStyle:   AuthStyleBearer,
			Paths:       []string{"/v1/chat/completions", "/v1/completions"},
			RequiresKey: true,
			Notes:       "Agregador; o roteamento por capacidade continua sendo decidido pelo Hades.",
		},
		{
			ID:          "xai",
			Name:        "xAI (Grok)",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "https://api.x.ai/v1",
			APIKeyEnv:   "XAI_API_KEY",
			AuthStyle:   AuthStyleBearer,
			Paths:       []string{"/v1/chat/completions", "/v1/completions"},
			RequiresKey: true,
		},
		{
			ID:          "openai-compatible",
			Name:        "Servidor compatÃ­vel com OpenAI (genÃ©rico)",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     GenericPresetBaseURL,
			APIKeyEnv:   GenericPresetEnvName,
			AuthStyle:   AuthStyleBearer,
			Paths:       []string{"/v1/chat/completions", "/v1/completions", "/v1/embeddings"},
			RequiresKey: false,
			Notes:       "Substitua a base_url antes de habilitar; Lemonade e outros servidores entram por aqui atÃ© terem preset verificado.",
		},
	}
	sort.Slice(presets, func(i, j int) bool { return presets[i].ID < presets[j].ID })
	return presets
}

// ProviderPresetByID busca um preset pelo identificador (sem diferenciar
// maiÃºsculas de minÃºsculas).
func ProviderPresetByID(id string) (ProviderPreset, bool) {
	for _, preset := range BuiltInProviderPresets() {
		if strings.EqualFold(preset.ID, strings.TrimSpace(id)) {
			return preset, true
		}
	}
	return ProviderPreset{}, false
}

// Provider monta um `Provider` vÃ¡lido a partir do preset. A credencial NÃƒO Ã©
// copiada: fica apenas o nome da variÃ¡vel de ambiente. Servidores locais sÃ£o
// marcados com AllowPrivate + AllowInsecureLoopback porque falam HTTP em
// loopback; isso Ã© explÃ­cito e nÃ£o vale para os presets remotos.
func (p ProviderPreset) Provider(models []ModelConfig) (Provider, error) {
	provider := Provider{
		Name:         p.ID,
		Type:         p.Type,
		BaseURL:      p.BaseURL,
		APIKeyEnv:    p.APIKeyEnv,
		AuthStyle:    p.AuthStyle,
		Paths:        append([]string(nil), p.Paths...),
		Models:       append([]ModelConfig(nil), models...),
		AllowPrivate: p.Local,
	}
	if p.Local {
		provider.AllowInsecureLoopback = true
	}
	if err := validateProvider(provider); err != nil {
		return Provider{}, err
	}
	return provider, nil
}
