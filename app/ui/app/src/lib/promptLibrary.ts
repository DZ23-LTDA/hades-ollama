import { SLASH_COMMANDS } from "./slashCommands";

// Biblioteca de prompts: cria, salva e reutiliza prompts com variáveis,
// pastas, etiquetas, favoritos e contador de uso. A persistência é local
// (localStorage) e degrada para memória quando o armazenamento é negado
// (modo privado, webview restrito), seguindo o padrão de theme.ts.

export const PROMPT_LIBRARY_SCHEMA = "hades.prompt-library.v1";
export const PROMPT_STORAGE_KEY = "hades_prompt_library";
export const MAX_PROMPT_TITLE_CHARACTERS = 120;
export const MAX_PROMPT_BODY_CHARACTERS = 8000;
export const MAX_PROMPT_TAGS = 12;
export const MAX_PROMPT_TAG_CHARACTERS = 32;

export type PromptRecord = {
  id: string;
  title: string;
  body: string;
  folder: string;
  tags: string[];
  favorite: boolean;
  builtin: boolean;
  uses: number;
  createdAt: number;
  updatedAt: number;
  lastUsedAt: number | null;
};

export type PromptLibrary = {
  schema: string;
  prompts: PromptRecord[];
};

export type PromptInput = {
  title: string;
  body: string;
  folder?: string;
  tags?: string[];
};

export type PromptPatch = Partial<PromptInput> & { favorite?: boolean };

export type PromptFilter = {
  query?: string;
  folder?: string;
  tag?: string;
  favoritesOnly?: boolean;
};

export type PromptRenderResult = {
  text: string;
  missing: string[];
};

const VARIABLE_PATTERN = /\{\{\s*([^{}]+?)\s*\}\}/g;

function emptyLibrary(): PromptLibrary {
  return { schema: PROMPT_LIBRARY_SCHEMA, prompts: [] };
}

function trimmedText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function finiteNumber(value: unknown, fallback: number): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

/** Nomes de variáveis na ordem de aparição, sem repetição. */
export function extractPromptVariables(body: string): string[] {
  const found: string[] = [];
  for (const match of body.matchAll(VARIABLE_PATTERN)) {
    const name = match[1].trim();
    if (name && !found.includes(name)) found.push(name);
  }
  return found;
}

/**
 * Substitui cada `{{variavel}}` pelo valor informado. Variáveis sem valor
 * permanecem no texto e são listadas em `missing`, para a interface avisar
 * antes de enviar.
 */
export function renderPrompt(
  body: string,
  values: Record<string, string> = {},
): PromptRenderResult {
  const missing: string[] = [];
  const text = body.replace(VARIABLE_PATTERN, (raw, rawName: string) => {
    const name = rawName.trim();
    const value = values[name];
    if (typeof value === "string" && value.trim() !== "") return value;
    if (!missing.includes(name)) missing.push(name);
    return raw;
  });
  return { text, missing };
}

export function normalizePromptTags(tags: unknown): string[] {
  if (!Array.isArray(tags)) return [];
  const result: string[] = [];
  for (const tag of tags) {
    const clean = trimmedText(tag).slice(0, MAX_PROMPT_TAG_CHARACTERS);
    if (clean && !result.includes(clean)) result.push(clean);
    if (result.length >= MAX_PROMPT_TAGS) break;
  }
  return result;
}

/** Mensagem de erro de validação ou `null` quando o prompt é utilizável. */
export function promptValidationError(input: PromptInput): string | null {
  if (!trimmedText(input.title)) return "Informe um título para o prompt.";
  if (input.title.trim().length > MAX_PROMPT_TITLE_CHARACTERS) {
    return `O título deve ter no máximo ${MAX_PROMPT_TITLE_CHARACTERS} caracteres.`;
  }
  if (!trimmedText(input.body)) return "Informe o conteúdo do prompt.";
  if (input.body.trim().length > MAX_PROMPT_BODY_CHARACTERS) {
    return `O conteúdo deve ter no máximo ${MAX_PROMPT_BODY_CHARACTERS} caracteres.`;
  }
  return null;
}

export function createPrompt(
  input: PromptInput,
  now: number,
  id?: string,
): PromptRecord {
  return {
    id: id ?? `prompt-${now.toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
    title: input.title.trim().slice(0, MAX_PROMPT_TITLE_CHARACTERS),
    body: input.body.trim().slice(0, MAX_PROMPT_BODY_CHARACTERS),
    folder: trimmedText(input.folder),
    tags: normalizePromptTags(input.tags),
    favorite: false,
    builtin: false,
    uses: 0,
    createdAt: now,
    updatedAt: now,
    lastUsedAt: null,
  };
}

/** Aplica um patch em um prompt; devolve a biblioteca inalterada se o id não existe. */
export function updatePrompt(
  library: PromptLibrary,
  id: string,
  patch: PromptPatch,
  now: number,
): PromptLibrary {
  let changed = false;
  const prompts = library.prompts.map((prompt) => {
    if (prompt.id !== id) return prompt;
    changed = true;
    return {
      ...prompt,
      title:
        patch.title === undefined
          ? prompt.title
          : patch.title.trim().slice(0, MAX_PROMPT_TITLE_CHARACTERS),
      body:
        patch.body === undefined
          ? prompt.body
          : patch.body.trim().slice(0, MAX_PROMPT_BODY_CHARACTERS),
      folder: patch.folder === undefined ? prompt.folder : trimmedText(patch.folder),
      tags: patch.tags === undefined ? prompt.tags : normalizePromptTags(patch.tags),
      favorite: patch.favorite === undefined ? prompt.favorite : patch.favorite,
      updatedAt: now,
    };
  });
  return changed ? { ...library, prompts } : library;
}

export function deletePrompt(library: PromptLibrary, id: string): PromptLibrary {
  const prompts = library.prompts.filter((prompt) => prompt.id !== id);
  return prompts.length === library.prompts.length ? library : { ...library, prompts };
}

export function duplicatePrompt(
  library: PromptLibrary,
  id: string,
  now: number,
  newId?: string,
): PromptLibrary {
  const source = library.prompts.find((prompt) => prompt.id === id);
  if (!source) return library;
  const copy: PromptRecord = {
    ...source,
    id: newId ?? `prompt-${now.toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
    title: `${source.title} (cópia)`.slice(0, MAX_PROMPT_TITLE_CHARACTERS),
    builtin: false,
    uses: 0,
    createdAt: now,
    updatedAt: now,
    lastUsedAt: null,
  };
  return { ...library, prompts: [...library.prompts, copy] };
}

export function togglePromptFavorite(library: PromptLibrary, id: string): PromptLibrary {
  return {
    ...library,
    prompts: library.prompts.map((prompt) =>
      prompt.id === id ? { ...prompt, favorite: !prompt.favorite } : prompt,
    ),
  };
}

export function addPrompt(library: PromptLibrary, prompt: PromptRecord): PromptLibrary {
  if (library.prompts.some((item) => item.id === prompt.id)) return library;
  return { ...library, prompts: [...library.prompts, prompt] };
}

/** Registra o uso: incrementa o contador e marca a data do último uso. */
export function recordPromptUse(
  library: PromptLibrary,
  id: string,
  now: number,
): PromptLibrary {
  let changed = false;
  const prompts = library.prompts.map((prompt) => {
    if (prompt.id !== id) return prompt;
    changed = true;
    return { ...prompt, uses: prompt.uses + 1, lastUsedAt: now };
  });
  return changed ? { ...library, prompts } : library;
}

export function libraryFolders(library: PromptLibrary): string[] {
  const folders = new Set<string>();
  for (const prompt of library.prompts) if (prompt.folder) folders.add(prompt.folder);
  return [...folders].sort((a, b) => a.localeCompare(b));
}

export function libraryTags(library: PromptLibrary): string[] {
  const tags = new Set<string>();
  for (const prompt of library.prompts) for (const tag of prompt.tags) tags.add(tag);
  return [...tags].sort((a, b) => a.localeCompare(b));
}

export function filterPrompts(
  prompts: readonly PromptRecord[],
  filter: PromptFilter = {},
): PromptRecord[] {
  const query = trimmedText(filter.query).toLowerCase();
  return prompts.filter((prompt) => {
    if (filter.favoritesOnly && !prompt.favorite) return false;
    if (filter.folder && prompt.folder !== filter.folder) return false;
    if (filter.tag && !prompt.tags.includes(filter.tag)) return false;
    if (!query) return true;
    return (
      prompt.title.toLowerCase().includes(query) ||
      prompt.body.toLowerCase().includes(query) ||
      prompt.folder.toLowerCase().includes(query) ||
      prompt.tags.some((tag) => tag.toLowerCase().includes(query))
    );
  });
}

/**
 * Prompts iniciais derivados dos comandos de barra já existentes, para que a
 * biblioteca não nasça vazia. `builtin: true` identifica a origem.
 */
export function builtinPrompts(now: number): PromptRecord[] {
  const bodies: Record<string, string> = {
    goal: "Crie uma missão com o objetivo: {{objetivo}}\n\nContexto: {{contexto}}",
    plan: "Monte o plano da missão \"{{objetivo}}\" sem executar nada.",
    test: "Crie uma missão para rodar os testes do projeto {{projeto}} e reportar as falhas.",
    review: "Revise o artefato {{artefato}} e liste riscos com correções testáveis.",
  };
  return SLASH_COMMANDS.map((command, index) => ({
    id: `builtin:${command.id}`,
    title: `${command.label} — ${command.description}`,
    body: bodies[command.id] ?? command.label,
    folder: "Comandos",
    tags: ["builtin", command.mode],
    favorite: index === 0,
    builtin: true,
    uses: 0,
    createdAt: now,
    updatedAt: now,
    lastUsedAt: null,
  }));
}

/**
 * Acrescenta os prompts iniciais que ainda não existem na biblioteca. Não
 * sobrescreve nada: um prompt inicial editado pelo usuário é preservado.
 */
export function withBuiltins(library: PromptLibrary, now: number): PromptLibrary {
  const existing = new Set(library.prompts.map((prompt) => prompt.id));
  const missing = builtinPrompts(now).filter((prompt) => !existing.has(prompt.id));
  if (missing.length === 0) return library;
  return { ...library, prompts: [...missing, ...library.prompts] };
}

function normalizeRecord(raw: unknown, now: number): PromptRecord | null {
  if (typeof raw !== "object" || raw === null) return null;
  const record = raw as Record<string, unknown>;
  const id = trimmedText(record.id);
  const title = trimmedText(record.title).slice(0, MAX_PROMPT_TITLE_CHARACTERS);
  const body = trimmedText(record.body).slice(0, MAX_PROMPT_BODY_CHARACTERS);
  if (!id || !title || !body) return null;
  return {
    id,
    title,
    body,
    folder: trimmedText(record.folder),
    tags: normalizePromptTags(record.tags),
    favorite: record.favorite === true,
    builtin: record.builtin === true,
    uses: Math.max(0, Math.floor(finiteNumber(record.uses, 0))),
    createdAt: finiteNumber(record.createdAt, now),
    updatedAt: finiteNumber(record.updatedAt, now),
    lastUsedAt:
      typeof record.lastUsedAt === "number" && Number.isFinite(record.lastUsedAt)
        ? record.lastUsedAt
        : null,
  };
}

export function serializeLibrary(library: PromptLibrary): string {
  return JSON.stringify({ schema: PROMPT_LIBRARY_SCHEMA, prompts: library.prompts }, null, 2);
}

/**
 * Lê uma biblioteca serializada. Entradas inválidas são descartadas e um
 * esquema desconhecido resulta em biblioteca vazia, em vez de erro.
 */
export function parseLibrary(raw: string | null | undefined, now: number): PromptLibrary {
  if (!raw) return emptyLibrary();
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return emptyLibrary();
  }
  if (typeof parsed !== "object" || parsed === null) return emptyLibrary();
  const candidate = parsed as { schema?: unknown; prompts?: unknown };
  if (trimmedText(candidate.schema) !== PROMPT_LIBRARY_SCHEMA) return emptyLibrary();
  if (!Array.isArray(candidate.prompts)) return emptyLibrary();
  const prompts: PromptRecord[] = [];
  for (const item of candidate.prompts) {
    const record = normalizeRecord(item, now);
    if (record && !prompts.some((existing) => existing.id === record.id)) {
      prompts.push(record);
    }
  }
  return { schema: PROMPT_LIBRARY_SCHEMA, prompts };
}

export type PromptStorage = Pick<Storage, "getItem" | "setItem">;

/** Carrega a biblioteca do armazenamento local; sem armazenamento, só os iniciais. */
export function loadPromptLibrary(
  storage?: PromptStorage,
  now: number = Date.now(),
): PromptLibrary {
  const target =
    storage ?? (typeof window !== "undefined" ? window.localStorage : undefined);
  let raw: string | null = null;
  try {
    raw = target?.getItem(PROMPT_STORAGE_KEY) ?? null;
  } catch {
    raw = null;
  }
  return withBuiltins(parseLibrary(raw, now), now);
}

/** Grava a biblioteca; falha de armazenamento não interrompe a interface. */
export function savePromptLibrary(
  library: PromptLibrary,
  storage?: PromptStorage,
): boolean {
  const target =
    storage ?? (typeof window !== "undefined" ? window.localStorage : undefined);
  try {
    if (!target) return false;
    target.setItem(PROMPT_STORAGE_KEY, serializeLibrary(library));
    return true;
  } catch {
    return false;
  }
}

/** Importa uma biblioteca de texto JSON, descartando o que não for válido. */
export function importLibrary(
  current: PromptLibrary,
  raw: string,
  now: number,
): { library: PromptLibrary; imported: number; error: string | null } {
  const incoming = parseLibrary(raw, now);
  if (incoming.prompts.length === 0) {
    return { library: current, imported: 0, error: "Nenhum prompt válido encontrado." };
  }
  let library = current;
  let imported = 0;
  for (const prompt of incoming.prompts) {
    if (library.prompts.some((existing) => existing.id === prompt.id)) continue;
    library = addPrompt(library, prompt);
    imported += 1;
  }
  if (imported === 0) {
    return { library: current, imported: 0, error: "Todos os prompts já existem." };
  }
  return { library, imported, error: null };
}
