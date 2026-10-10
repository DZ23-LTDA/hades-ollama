package multillm

import (
	"sort"
	"strings"
)

// Presets guiados de provedor (estágio 4 do Hades).
//
// O gateway é orientado por TIPO de protocolo (`openai-compatible`,
// `anthropic`, `cli`) e os provedores vêm de configuração JSON. Estes presets
// existem para o cadastro guiado em pt-BR: cada um entrega `base_url`, nome da
// variável de ambiente da credencial, estilo de autenticação e caminhos
// suportados — nunca a credencial em si.
//
// Regra de honestidade embutida: um preset não declara modelo nem capacidade.
// As capacidades de cada modelo continuam vindo de `ProbeProviderModel` /
// descoberta real, e a chave continua vindo do ambiente ou do cofre, nunca
// deste pacote.
//
// Cobertura: Ollama, vLLM, llama.cpp, LM Studio (servidores locais) e OpenAI,
// Anthropic, Gemini, DeepSeek, Groq, Mistral, OpenRouter e xAI (APIs). Outros
// servidores compatíveis com OpenAI entram pelo preset genérico, cuja
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

// GenericPresetBaseURL é um marcador reservado (TLD `.invalid`): o operador
// precisa substituí-lo ao cadastrar um servidor compatível com OpenAI que não
// esteja na lista. Servir um host plausível seria pior, porque pareceria
// funcional.
const GenericPresetBaseURL = "https://api.exemplo.invalid/v1"

const (
	// GenericPresetEnvName é o nome da variável de ambiente esperada pelo
	// preset genérico; o operador pode trocá-lo na configuração.
	GenericPresetEnvName = "PROVEDOR_COMPATIVEL_API_KEY"
)

// BuiltInProviderPresets devolve a lista canônica de presets, ordenada por ID.
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
			Notes:       "Motor local padrão do Hades; não exige credencial.",
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
			Notes:       "Porta padrão do llama-server.",
		},
		{
			ID:          "lmstudio",
			Name:        "LM Studio (local)",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "http://127.0.0.1:1234/v1",
			Paths:       []string{"/v1/chat/completions", "/v1/completions", "/v1/embeddings"},
			Local:       true,
			RequiresKey: false,
			Notes:       "Porta padrão do servidor local do LM Studio.",
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
			ID:          "gemini",
			Name:        "Google Gemini (endpoint compatível com OpenAI)",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     "https://generativelanguage.googleapis.com/v1beta/openai",
			APIKeyEnv:   "GEMINI_API_KEY",
			AuthStyle:   AuthStyleBearer,
			Paths:       []string{"/v1/chat/completions", "/v1/embeddings"},
			RequiresKey: true,
			Notes:       "Usa a camada de compatibilidade oficial; o protocolo nativo não é usado aqui.",
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
			Name:        "Servidor compatível com OpenAI (genérico)",
			Type:        ProviderTypeOpenAICompatible,
			BaseURL:     GenericPresetBaseURL,
			APIKeyEnv:   GenericPresetEnvName,
			AuthStyle:   AuthStyleBearer,
			Paths:       []string{"/v1/chat/completions", "/v1/completions", "/v1/embeddings"},
			RequiresKey: false,
			Notes:       "Substitua a base_url antes de habilitar; Lemonade e outros servidores entram por aqui até terem preset verificado.",
		},
	}
	sort.Slice(presets, func(i, j int) bool { return presets[i].ID < presets[j].ID })
	return presets
}

// ProviderPresetByID busca um preset pelo identificador (sem diferenciar
// maiúsculas de minúsculas).
func ProviderPresetByID(id string) (ProviderPreset, bool) {
	for _, preset := range BuiltInProviderPresets() {
		if strings.EqualFold(preset.ID, strings.TrimSpace(id)) {
			return preset, true
		}
	}
	return ProviderPreset{}, false
}

// Provider monta um `Provider` válido a partir do preset. A credencial NÃO é
// copiada: fica apenas o nome da variável de ambiente. Servidores locais são
// marcados com AllowPrivate + AllowInsecureLoopback porque falam HTTP em
// loopback; isso é explícito e não vale para os presets remotos.
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
