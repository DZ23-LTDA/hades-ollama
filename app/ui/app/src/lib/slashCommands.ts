export type SlashCommandId = "goal" | "plan" | "test" | "review" | "spark" | "cowork" | "reject";

export type SlashCommand = {
  id: SlashCommandId;
  label: string;
  description: string;
  mode: "goal" | "plan";
  autoRun: boolean;
};

export const SLASH_COMMANDS: readonly SlashCommand[] = [
  {
    id: "goal",
    label: "/goal",
    description: "cria uma missão e delega aos papéis do swarm",
    mode: "goal",
    autoRun: true,
  },
  {
    id: "plan",
    label: "/plan",
    description: "gera o plano da missão sem executar",
    mode: "plan",
    autoRun: false,
  },
  {
    id: "test",
    label: "/test",
    description: "cria uma missão para rodar os testes do projeto",
    mode: "goal",
    autoRun: true,
  },
  {
    id: "review",
    label: "/review",
    description: "cria uma missão para revisar o código ou artefato atual",
    mode: "goal",
    autoRun: true,
  },
  // E-975: `spark` e `cowork` estavam NO TIPO (`SlashCommandId`, L1) e NAO no catalogo.
  //
  // *Um tipo que promete mais do que o catalogo entrega*: `filterSlashCommands` nunca os
  // podia devolver, e o menu nunca os mostrava. E' o `E-964` literalmente - *o tipo dizia
  // que existia, o catalogo dizia que nao*.
  //
  // **E sao MODOS do mesmo mecanismo, nao motores separados** (§FASE-3 item 3): ambos
  // entram por `parseSlashCommand` -> `/agentic?objective&slash&autorun` -> `AgenticConsole`,
  // que cria a missao pela **mesma** `POST /api/agent/v1/missions`.
  //
  // *E o que NAO se faz aqui*: mudar o `AgenticConsole:230` de `slashCommand === "goal"`
  // para `mode === "goal"`. Isso faria `/test` e `/review` **passarem a delegar**, e eles
  // nao delegavam - *alterar comportamento para o codigo "ficar mais coerente" e' o que o
  // §ANTILOOP item 10 proibe.*
  {
    id: "spark",
    label: "/spark",
    description: "cria uma missão curta de ideação e rascunho, sem execução pesada",
    mode: "goal",
    autoRun: false,
  },
  {
    id: "cowork",
    label: "/cowork",
    description: "cria uma missão colaborativa com os papéis do swarm sobre o mesmo runtime",
    mode: "goal",
    autoRun: true,
  },
];

export function filterSlashCommands(query: string): SlashCommand[] {
  const normalized = query.trim().toLowerCase();
  if (!normalized) return [...SLASH_COMMANDS];
  return SLASH_COMMANDS.filter(
    (command) =>
      command.id.startsWith(normalized) ||
      command.description.toLowerCase().includes(normalized),
  );
}

export function parseSlashCommand(input: string): {
  command: SlashCommand;
  objective: string;
} | null {
  const match = input.trim().match(/^\/([a-z0-9_-]+)(?:\s+([\s\S]*))?$/i);
  if (!match) return null;
  const command = SLASH_COMMANDS.find((item) => item.id === match[1].toLowerCase());
  if (!command) return null;
  return { command, objective: (match[2] ?? "").trim() };
}

export function slashCommandQuery(input: string): string | null {
  const trimmed = input.trimStart();
  if (!trimmed.startsWith("/") || /\s/.test(trimmed.slice(1))) return null;
  return trimmed.slice(1);
}

export function moveSlashCommandIndex(
  current: number,
  direction: "next" | "previous",
  count: number,
): number {
  if (count <= 0) return 0;
  return direction === "next"
    ? (current + 1) % count
    : (current - 1 + count) % count;
}

export function slashCommandURL(input: string): string | null {
  const parsed = parseSlashCommand(input);
  if (!parsed || !parsed.objective) return null;
  const params = new URLSearchParams({
    objective: parsed.objective,
    slash: parsed.command.id,
    autorun: String(parsed.command.autoRun),
  });
  return `/agentic?${params.toString()}`;
}
