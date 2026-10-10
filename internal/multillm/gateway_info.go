package multillm

import (
	"os"
	"strings"
)

// DefaultGatewayKeyEnv é a variável usada para autorizar clientes remotos
// quando o operador não escolhe outra em gateway_api_key_env.
const DefaultGatewayKeyEnv = "OLLAMA_DZ23_GATEWAY_KEY"

// LocalModelEnv é a variável que aponta o modelo local servido pelo alias
// local/private. Sem ela o alias responde 503 e o mesmo endereço base continua
// atendendo às APIs de terceiros.
const LocalModelEnv = "OLLAMA_DZ23_LOCAL_MODEL"

// GatewayRotation é a política efectiva da rotação automática já resolvida.
type GatewayRotation struct {
	Enabled       bool `json:"enabled"`
	MaxAttempts   int  `json:"max_attempts"`
	CrossProvider bool `json:"cross_provider"`
}

// Effective resolve a rotação declarada no arquivo de configuração. Um bloco
// rotation ausente equivale a ligado com três tentativas, o mesmo limite usado
// pelo gateway em tempo de execução.
func (c *RotationConfig) Effective() GatewayRotation {
	resolved := GatewayRotation{Enabled: true, MaxAttempts: defaultRotationAttempts}
	if c == nil {
		return resolved
	}
	if c.Enabled != nil {
		resolved.Enabled = *c.Enabled
	}
	if c.CrossProvider != nil {
		resolved.CrossProvider = *c.CrossProvider
	}
	if c.MaxAttempts > 0 {
		resolved.MaxAttempts = c.MaxAttempts
	}
	if resolved.MaxAttempts < 1 {
		resolved.MaxAttempts = 1
	}
	if resolved.MaxAttempts > maxRotationAttempts {
		resolved.MaxAttempts = maxRotationAttempts
	}
	return resolved
}

// RotationPolicy expõe a rotação já resolvida do registro.
func (r *Registry) RotationPolicy() GatewayRotation {
	if r == nil {
		return (*RotationConfig)(nil).Effective()
	}
	return r.rotation.Effective()
}

// GatewayKeyEnv devolve o nome da variável que autoriza clientes remotos, ou
// vazio quando o gateway aceita apenas este computador.
func (r *Registry) GatewayKeyEnv() string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.gatewayAPIKeyEnv)
}

// GatewayKeyPresent informa se a chave exigida por GatewayKeyEnv já existe.
func (r *Registry) GatewayKeyPresent() bool {
	return CredentialValue(r.GatewayKeyEnv()) != ""
}

// GatewayKey devolve a chave do gateway do operador. O valor nunca é escrito
// em log nem devolvido em erro; só a interface local pode exibi-lo.
func (r *Registry) GatewayKey() string {
	return CredentialValue(r.GatewayKeyEnv())
}

// GatewayProtocol é um caminho de protocolo aceito pelo gateway, com o cliente
// de terceiros que costuma falar esse protocolo.
type GatewayProtocol struct {
	Label  string `json:"label"`
	Path   string `json:"path"`
	Client string `json:"client"`
}

// GatewayProtocols é a lista estável dos protocolos atendidos. Todos terminam no
// mesmo ChatHandler, então a rotação vale para os três.
func GatewayProtocols() []GatewayProtocol {
	return []GatewayProtocol{
		{Label: "Compatible con OpenAI", Path: "/v1/chat/completions", Client: "openai-sdk"},
		{Label: "Anthropic Messages", Path: "/v1/messages", Client: "claude"},
		{Label: "Responses (Codex)", Path: "/v1/responses", Client: "codex"},
		{Label: "Lista de modelos", Path: "/v1/models", Client: "models"},
	}
}

// GatewayAliases são os nomes virtuais aceitos no campo model. O gateway
// escolhe o modelo de maior prioridade que atende a capacidade pedida, no
// mesmo endereço base que também serve a IA local.
func GatewayAliases() []string {
	return []string{"auto/coding", "auto/reasoning", "auto/vision", "local/private"}
}

// GatewayInfo é o bloco de conexão mostrado em "Configurar inferência de
// terceiros": endereço base, protocolos, rotação e chave do gateway.
type GatewayInfo struct {
	BaseURL    string            `json:"base_url"`
	ConfigPath string            `json:"config_path"`
	KeyEnv     string            `json:"gateway_key_env"`
	KeyPresent bool              `json:"gateway_key_present"`
	Key        string            `json:"gateway_key,omitempty"`
	Rotation   GatewayRotation   `json:"rotation"`
	Protocols  []GatewayProtocol `json:"protocols"`
	Aliases    []string          `json:"aliases"`
	LocalModel string            `json:"local_model,omitempty"`
	Loopback   bool              `json:"loopback_only"`
	Providers  int               `json:"providers"`
}

// DescribeGateway monta a conexão do gateway a partir da configuração já lida.
// revealKey só deve ser verdadeiro para o pedido da própria máquina: a chave é
// um segredo local e não pode sair do computador do operador.
func DescribeGateway(cfg Config, registry *Registry, baseURL, configPath string, revealKey bool) GatewayInfo {
	env := strings.TrimSpace(cfg.GatewayAPIKeyEnv)
	if env == "" {
		env = DefaultGatewayKeyEnv
	}
	info := GatewayInfo{
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		ConfigPath: configPath,
		KeyEnv:     env,
		KeyPresent: CredentialValue(env) != "",
		Rotation:   cfg.Rotation.Effective(),
		Protocols:  GatewayProtocols(),
		Aliases:    GatewayAliases(),
		LocalModel: strings.TrimSpace(os.Getenv(LocalModelEnv)),
		Loopback:   strings.TrimSpace(cfg.GatewayAPIKeyEnv) == "",
		Providers:  len(cfg.Providers),
	}
	if registry != nil {
		info.Rotation = registry.RotationPolicy()
	}
	if revealKey && info.KeyPresent {
		info.Key = CredentialValue(env)
	}
	return info
}
