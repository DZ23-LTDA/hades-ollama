// Connection details shown on the "Endpoint da API" page. The Ollama server
// speaks its native API plus OpenAI- and Anthropic-compatible APIs on the
// same port, so every tool points at the same base address.
export const LOCAL_BASE_URL = "http://localhost:11434";

export type EndpointInfo = { label: string; url: string; hint: string };

export type EndpointHealthStatus = "checking" | "online" | "offline";

export function endpointHealthLabel(status: EndpointHealthStatus): string {
  if (status === "online") return "Online";
  if (status === "offline") return "Offline";
  return "Verificando…";
}

export function endpoints(base: string = LOCAL_BASE_URL): EndpointInfo[] {
  return [
    {
      label: "Ollama (nativo)",
      url: base,
      hint: "/api/chat, /api/generate, /api/tags",
    },
    {
      label: "Compatível com OpenAI",
      url: `${base}/v1`,
      hint: "chat/completions, responses, models",
    },
    {
      label: "Compatível com Anthropic",
      url: base,
      hint: "/v1/messages (Claude Code, SDK Anthropic)",
    },
  ];
}

export type ToolSetup = {
  id: string;
  title: string;
  description: string;
  code: string;
};

// quote wraps a model name for the shell only when it needs it.
function quote(value: string): string {
  return /^[\w.:/-]+$/.test(value) ? value : `"${value.replace(/"/g, '\\"')}"`;
}

// Gateway de inferência de terceiros: o mesmo servidor roteia para os modelos
// locais e para as APIs de terceiros, com rotação automática quando um provedor
// falha. Estes tipos espelham multillm.GatewayInfo devolvido por
// GET /api/v1/gateway/connection.
export type GatewayRotation = {
  enabled: boolean;
  max_attempts: number;
  cross_provider: boolean;
};

export type GatewayProtocol = { label: string; path: string; client: string };

export type GatewayConnection = {
  base_url: string;
  config_path: string;
  gateway_key_env: string;
  gateway_key_present: boolean;
  gateway_key?: string;
  rotation: GatewayRotation;
  protocols: GatewayProtocol[];
  loopback_only: boolean;
  providers: number;
  restart_required?: boolean;
};

// fetchGatewayConnection lê a conexão do gateway. Ausência do endpoint (build
// antigo, servidor local puro) não é erro: a página degrada e continua útil.
export async function fetchGatewayConnection(): Promise<GatewayConnection | null> {
  try {
    const response = await fetch("/api/v1/gateway/connection", {
      headers: { Accept: "application/json" },
    });
    if (!response.ok) return null;
    return (await response.json()) as GatewayConnection;
  } catch {
    return null;
  }
}

// rotateGatewayKey gera e guarda uma nova chave do gateway. Um erro aqui precisa
// chegar à tela, então a exceção sobe com a mensagem do servidor.
export async function rotateGatewayKey(): Promise<GatewayConnection> {
  const response = await fetch("/api/v1/gateway/key", { method: "POST" });
  const body = (await response.json().catch(() => ({}))) as { error?: string };
  if (!response.ok) {
    throw new Error(body.error ?? "Não foi possível gerar a chave do gateway");
  }
  return body as GatewayConnection;
}

export function gatewayRotationLabel(rotation: GatewayRotation): string {
  if (!rotation.enabled) return "Rotação desligada";
  const attempts = `${rotation.max_attempts} tentativa${rotation.max_attempts === 1 ? "" : "s"}`;
  return rotation.cross_provider
    ? `Rotação ligada · ${attempts} · entre provedores diferentes`
    : `Rotação ligada · ${attempts} · provedores reservas`;
}

// gatewaySetups monta os comandos que apontam Claude Code e Codex para o
// gateway. A chave do gateway substitui o token fixo usado no modo local.
export function gatewaySetups(
  connection: Pick<
    GatewayConnection,
    "base_url" | "gateway_key" | "gateway_key_env"
  >,
  model: string,
): ToolSetup[] {
  const base = connection.base_url.replace(/\/+$/, "");
  const key = connection.gateway_key ?? `<${connection.gateway_key_env}>`;
  const m = quote(model || "auto");
  return [
    {
      id: "gateway-claude",
      title: "Claude Code pelo gateway",
      description:
        "Aponta o Claude Code para o gateway; a rotação escolhe o provedor sozinha.",
      code: [
        `$env:ANTHROPIC_BASE_URL = "${base}"`,
        `$env:ANTHROPIC_AUTH_TOKEN = "${key}"`,
        `$env:ANTHROPIC_API_KEY = ""`,
        `claude --model ${m}`,
      ].join("\n"),
    },
    {
      id: "gateway-codex",
      title: "Codex CLI pelo gateway",
      description:
        "O Codex fala /v1/responses no mesmo endereço, com a mesma chave.",
      code: [
        `$env:OPENAI_BASE_URL = "${base}/v1"`,
        `$env:OPENAI_API_KEY = "${key}"`,
        `codex --model ${m}`,
      ].join("\n"),
    },
    {
      id: "gateway-curl",
      title: "Teste rápido do gateway",
      description: "Confere a lista de modelos com a chave do gateway.",
      code: `curl -H "Authorization: Bearer ${key}" ${base}/v1/models`,
    },
  ];
}

export function toolSetups(
  model: string,
  base: string = LOCAL_BASE_URL,
): ToolSetup[] {
  const m = quote(model || "qwen2.5-coder:7b");
  return [
    {
      id: "claude",
      title: "Claude Code",
      description:
        "O jeito mais simples: o Ollama abre o Claude Code já apontado para ele.",
      code: `ollama launch claude --model ${m}`,
    },
    {
      id: "claude-env",
      title: "Claude Code (manual, PowerShell)",
      description:
        "Para abrir o Claude Code você mesmo, defina estas variáveis antes.",
      code: [
        `$env:ANTHROPIC_BASE_URL = "${base}"`,
        `$env:ANTHROPIC_AUTH_TOKEN = "ollama"`,
        `$env:ANTHROPIC_API_KEY = ""`,
        `claude --model ${m}`,
      ].join("\n"),
    },
    {
      id: "codex",
      title: "Codex CLI",
      description: "O Ollama configura e abre o Codex com o modelo escolhido.",
      code: `ollama launch codex --model ${m}`,
    },
    {
      id: "openai-sdk",
      title: "SDK OpenAI (Python)",
      description:
        "Qualquer app compatível com OpenAI funciona trocando a base_url.",
      code: [
        "from openai import OpenAI",
        "",
        `client = OpenAI(base_url="${base}/v1", api_key="ollama")`,
        `resp = client.chat.completions.create(model="${model || "qwen2.5-coder:7b"}", messages=[{"role": "user", "content": "Olá"}])`,
        "print(resp.choices[0].message.content)",
      ].join("\n"),
    },
    {
      id: "curl",
      title: "Teste rápido (curl)",
      description: "Confere se o servidor responde.",
      code: `curl ${base}/v1/models`,
    },
  ];
}
