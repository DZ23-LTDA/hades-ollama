export type SlashCommandId = "goal" | "plan" | "test" | "review";

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
