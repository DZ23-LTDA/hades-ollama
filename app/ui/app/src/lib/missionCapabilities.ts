// Catálogo de escopos concedíveis a uma missão.
//
// O servidor valida cada capability na política fail-closed
// (`internal/agent/capability_policy.go`): qualquer escopo desconhecido é
// rejeitado em `Runtime.CreateMission`, e as tools só executam quando o
// escopo declarado no descritor está concedido. A UI não deve inventar esse
// vocabulário: ela o deriva do catálogo real de ferramentas
// (`GET /api/agent/v1/tools`, que devolve `scopes`, `risk` e
// `requires_approval` por tool). Assim o seletor acompanha o runtime sem
// duplicar listas nem divergir com o tempo.

/** Toda missão nasce com leitura; o servidor não aceita missão sem escopo. */
export const BASELINE_CAPABILITY = "workspace:read";

/** Subconjunto do descritor de tool que importa para conceder permissões. */
export type ToolScopeSource = {
  name?: string;
  risk?: string;
  scopes?: string[];
  requires_approval?: boolean;
};

/** Escopo concedível, agregado a partir de todas as tools que o declaram. */
export type ScopeOption = {
  name: string;
  /** Escopos além do baseline dão poder real (disco, terminal, tela, rede). */
  elevated: boolean;
  /** Maior risco observado entre as tools que declaram o escopo. */
  risk: string;
  /** true quando alguma tool que declara o escopo exige aprovação humana. */
  requiresApproval: boolean;
  /** Tools do catálogo que dependem deste escopo. */
  tools: string[];
};

const RISK_ORDER = ["unknown", "low", "medium", "high", "critical"];

function normalizeRisk(risk?: string): string {
  const value = (risk ?? "").trim().toLowerCase();
  return RISK_ORDER.includes(value) ? value : "unknown";
}

function highestRisk(current: string, next: string): string {
  return RISK_ORDER.indexOf(next) > RISK_ORDER.indexOf(current)
    ? next
    : current;
}

/**
 * Agrega os escopos declarados pelas tools em opções únicas e ordenadas.
 * Escopos em branco são descartados: conceder algo que ninguém declarou só
 * produziria uma missão recusada pelo servidor.
 */
export function grantableScopes(
  tools: ToolScopeSource[] | null | undefined,
): ScopeOption[] {
  const byName = new Map<string, ScopeOption>();
  for (const tool of tools ?? []) {
    const toolName = (tool?.name ?? "").trim();
    for (const raw of tool?.scopes ?? []) {
      const name = (raw ?? "").trim();
      if (!name) continue;
      const option: ScopeOption = byName.get(name) ?? {
        name,
        elevated: name !== BASELINE_CAPABILITY,
        risk: "unknown",
        requiresApproval: false,
        tools: [],
      };
      option.risk = highestRisk(option.risk, normalizeRisk(tool.risk));
      option.requiresApproval =
        option.requiresApproval || tool.requires_approval === true;
      if (toolName && !option.tools.includes(toolName))
        option.tools.push(toolName);
      byName.set(name, option);
    }
  }
  return [...byName.values()]
    .sort((left, right) => {
      if (left.name === BASELINE_CAPABILITY)
        return right.name === BASELINE_CAPABILITY ? 0 : -1;
      if (right.name === BASELINE_CAPABILITY) return 1;
      return left.name.localeCompare(right.name);
    })
    .map((option) => ({ ...option, tools: [...option.tools].sort() }));
}

/** Um escopo só é concedível quando o catálogo o declara (fail-closed). */
export function isGrantableScope(
  scope: string,
  options: ScopeOption[],
): boolean {
  const name = (scope ?? "").trim();
  if (!name) return false;
  return options.some((option) => option.name === name);
}

/** Remove nomes desconhecidos ou vazios antes de enviar o payload. */
export function filterGrantable(
  selected: string[],
  options: ScopeOption[],
): string[] {
  return (selected ?? []).filter((scope) => isGrantableScope(scope, options));
}

/**
 * Payload final de `capabilities`: baseline sempre presente, sem duplicatas e
 * com ordem estável (baseline primeiro) para o diff do histórico não oscilar.
 */
export function buildMissionCapabilities(
  selected: string[] | null | undefined,
): string[] {
  const scopes = new Set<string>([BASELINE_CAPABILITY]);
  for (const raw of selected ?? []) {
    const name = (raw ?? "").trim();
    if (name) scopes.add(name);
  }
  const rest = [...scopes]
    .filter((scope) => scope !== BASELINE_CAPABILITY)
    .sort();
  return [BASELINE_CAPABILITY, ...rest];
}

const SCOPE_EFFECTS: Record<string, string> = {
  "workspace:read": "Lê arquivos do projeto selecionado.",
  "workspace:write": "Cria, altera e remove arquivos do projeto.",
  "repo:read": "Lê estado e histórico do repositório Git.",
  "terminal:allowlisted":
    "Executa comandos de terminal que estejam na allowlist.",
  "sandbox:execute": "Executa código em sandbox isolado.",
  "browser:navigate": "Navega na web em nome da missão.",
  "browser:files": "Baixa e lê arquivos durante a navegação.",
  "browser:takeover": "Assume uma sessão de navegador já autenticada.",
  "desktop:screen": "Captura a tela do computador.",
  "desktop:input": "Controla mouse e teclado do computador.",
  "desktop:clipboard": "Lê e escreve a área de transferência.",
  "desktop:process": "Inicia e controla processos locais.",
  "harness:cli":
    "Executa CLIs externas (claude, codex, gemini, qwen) sob aprovação e DLP.",
  "harness.cli":
    "Executa CLIs externas (claude, codex, gemini, qwen) sob aprovação e DLP.",
  "mcp:call": "Chama servidores MCP locais.",
  "mcp:remote:call": "Chama servidores MCP remotos.",
  "connector:external": "Envia e recebe dados por conectores externos.",
  "media:execute": "Processa e gera mídia (áudio, imagem, vídeo).",
};

/** Explicação curta em pt-BR do efeito do escopo, para decisão informada. */
export function describeScope(option: ScopeOption): string {
  const effect = SCOPE_EFFECTS[option.name];
  if (effect) return effect;
  if (!option.elevated) return "Escopo de leitura do projeto.";
  switch (option.risk) {
    case "critical":
      return "Escopo elevado de risco crítico; exige revisão antes de conceder.";
    case "high":
      return "Escopo elevado de risco alto; concede efeitos fora do projeto.";
    default:
      return "Escopo elevado declarado pelo runtime; revise antes de conceder.";
  }
}

/** Texto de aviso exibido quando a missão recebe escopos além da leitura. */
export function elevatedWarning(options: ScopeOption[]): string {
  const elevated = options.filter((option) => option.elevated);
  if (elevated.length === 0) return "";
  const list = elevated.map((option) => option.name).join(", ");
  const approvals = elevated
    .filter((option) => option.requiresApproval)
    .map((option) => option.name);
  const approvalNote =
    approvals.length > 0
      ? ` Exigem aprovação explícita: ${approvals.join(", ")}.`
      : " Cada tool que os usa ainda passa por DLP e egress.";
  return `Escopos elevados concedidos: ${list}.${approvalNote}`;
}
