// Connection details shown on the "Endpoint da API" page. The Ollama server
// speaks its native API plus OpenAI- and Anthropic-compatible APIs on the
// same port, so every tool points at the same base address.
export const LOCAL_BASE_URL = "http://localhost:11434";

export type EndpointInfo = { label: string; url: string; hint: string };

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
