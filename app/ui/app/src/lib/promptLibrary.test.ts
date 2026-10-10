import { describe, expect, it } from "vitest";

import type { PromptLibrary, PromptStorage } from "./promptLibrary";
import {
  MAX_PROMPT_TAGS,
  PROMPT_LIBRARY_SCHEMA,
  PROMPT_STORAGE_KEY,
  addPrompt,
  builtinPrompts,
  createPrompt,
  deletePrompt,
  duplicatePrompt,
  extractPromptVariables,
  filterPrompts,
  importLibrary,
  libraryFolders,
  libraryTags,
  loadPromptLibrary,
  parseLibrary,
  promptValidationError,
  recordPromptUse,
  renderPrompt,
  savePromptLibrary,
  serializeLibrary,
  togglePromptFavorite,
  updatePrompt,
  withBuiltins,
} from "./promptLibrary";

const NOW = 1_760_000_000_000;

function memoryStorage(initial: Record<string, string> = {}) {
  const data = new Map(Object.entries(initial));
  return {
    getItem: (key: string) => data.get(key) ?? null,
    setItem: (key: string, value: string) => {
      data.set(key, value);
    },
    snapshot: () => Object.fromEntries(data),
  };
}

const deniedStorage: PromptStorage = {
  getItem() {
    throw new Error("acesso negado");
  },
  setItem() {
    throw new Error("acesso negado");
  },
};

function sample(id = "p1", overrides: Partial<Parameters<typeof addPrompt>[1]> = {}) {
  return {
    ...createPrompt({ title: "Revisar PR", body: "Revise {{alvo}} com foco em {{foco}}" }, NOW, id),
    ...overrides,
  };
}

function libraryWith(...prompts: ReturnType<typeof sample>[]): PromptLibrary {
  return { schema: PROMPT_LIBRARY_SCHEMA, prompts };
}

describe("extractPromptVariables", () => {
  it("lista variáveis únicas na ordem de aparição", () => {
    expect(extractPromptVariables("{{a}} x {{ b }} y {{a}}")).toEqual(["a", "b"]);
  });

  it("devolve lista vazia sem variáveis", () => {
    expect(extractPromptVariables("texto simples {{}}")).toEqual([]);
  });
});

describe("renderPrompt", () => {
  it("substitui valores e mantém variáveis pendentes", () => {
    const result = renderPrompt("Olá {{nome}}, revise {{alvo}}", { nome: "Ana" });
    expect(result.text).toBe("Olá Ana, revise {{alvo}}");
    expect(result.missing).toEqual(["alvo"]);
  });

  it("trata espaço em branco e chaves com espaços", () => {
    const result = renderPrompt("{{ nome }}", { nome: "  " });
    expect(result.text).toBe("{{ nome }}");
    expect(result.missing).toEqual(["nome"]);
  });
});

describe("promptValidationError", () => {
  it("exige título e conteúdo", () => {
    expect(promptValidationError({ title: " ", body: "ok" })).toBe(
      "Informe um título para o prompt.",
    );
    expect(promptValidationError({ title: "ok", body: " " })).toBe(
      "Informe o conteúdo do prompt.",
    );
    expect(promptValidationError({ title: "ok", body: "texto" })).toBeNull();
  });
});

describe("createPrompt", () => {
  it("normaliza campos e limita etiquetas", () => {
    const prompt = createPrompt(
      {
        title: "  Título  ",
        body: "  corpo  ",
        folder: "  Trabalho ",
        tags: ["a", "a", " b "],
      },
      NOW,
      "p1",
    );
    expect(prompt).toMatchObject({
      id: "p1",
      title: "Título",
      body: "corpo",
      folder: "Trabalho",
      tags: ["a", "b"],
      favorite: false,
      builtin: false,
      uses: 0,
    });
  });

  it("limita a quantidade de etiquetas", () => {
    const tags = Array.from({ length: MAX_PROMPT_TAGS + 5 }, (_, index) => `t${index}`);
    expect(createPrompt({ title: "t", body: "b", tags }, NOW, "p1").tags).toHaveLength(
      MAX_PROMPT_TAGS,
    );
  });
});

describe("mutações da biblioteca", () => {
  it("atualiza, favorita, duplica, usa e exclui", () => {
    let library = libraryWith(sample("p1"));
    library = updatePrompt(library, "p1", { title: "Novo", tags: ["x"] }, NOW + 1);
    expect(library.prompts[0]).toMatchObject({
      title: "Novo",
      tags: ["x"],
      updatedAt: NOW + 1,
    });

    library = togglePromptFavorite(library, "p1");
    expect(library.prompts[0].favorite).toBe(true);

    library = duplicatePrompt(library, "p1", NOW + 2, "p2");
    expect(library.prompts.map((prompt) => prompt.id)).toEqual(["p1", "p2"]);
    expect(library.prompts[1].title).toBe("Novo (cópia)");
    expect(library.prompts[1].uses).toBe(0);

    library = recordPromptUse(library, "p1", NOW + 3);
    expect(library.prompts[0]).toMatchObject({ uses: 1, lastUsedAt: NOW + 3 });

    library = deletePrompt(library, "p1");
    expect(library.prompts.map((prompt) => prompt.id)).toEqual(["p2"]);
  });

  it("ignora ids inexistentes sem alterar a referência", () => {
    const library = libraryWith(sample("p1"));
    expect(updatePrompt(library, "nope", { title: "x" }, NOW)).toBe(library);
    expect(deletePrompt(library, "nope")).toBe(library);
    expect(recordPromptUse(library, "nope", NOW)).toBe(library);
    expect(duplicatePrompt(library, "nope", NOW)).toBe(library);
  });

  it("não adiciona prompts duplicados", () => {
    const library = libraryWith(sample("p1"));
    expect(addPrompt(library, sample("p1"))).toBe(library);
  });
});

describe("filtros e coleções", () => {
  const library = libraryWith(
    sample("p1", { title: "Revisar PR", folder: "Engenharia", tags: ["revisão"] }),
    sample("p2", { title: "Resumo semanal", folder: "Gestão", favorite: true }),
  );

  it("filtra por texto, pasta, etiqueta e favoritos", () => {
    expect(filterPrompts(library.prompts, { query: "revisar" })).toHaveLength(1);
    expect(filterPrompts(library.prompts, { folder: "Gestão" })).toHaveLength(1);
    expect(filterPrompts(library.prompts, { tag: "revisão" })).toHaveLength(1);
    expect(filterPrompts(library.prompts, { favoritesOnly: true })).toHaveLength(1);
    expect(filterPrompts(library.prompts)).toHaveLength(2);
  });

  it("busca também no corpo e nas etiquetas", () => {
    expect(filterPrompts(library.prompts, { query: "alvo" })).toHaveLength(2);
  });

  it("lista pastas e etiquetas ordenadas", () => {
    expect(libraryFolders(library)).toEqual(["Engenharia", "Gestão"]);
    expect(libraryTags(library)).toEqual(["revisão"]);
  });
});

describe("prompts iniciais", () => {
  it("deriva um prompt por comando de barra", () => {
    const builtins = builtinPrompts(NOW);
    expect(builtins).toHaveLength(4);
    expect(builtins.map((prompt) => prompt.id)).toEqual([
      "builtin:goal",
      "builtin:plan",
      "builtin:test",
      "builtin:review",
    ]);
    expect(builtins.every((prompt) => prompt.builtin)).toBe(true);
    expect(builtins[0].tags).toContain("builtin");
  });

  it("acrescenta apenas os ausentes e preserva edições", () => {
    const edited = { ...builtinPrompts(NOW)[0], title: "Meu objetivo" };
    const library = libraryWith(edited);
    const merged = withBuiltins(library, NOW);
    expect(merged.prompts).toHaveLength(4);
    const goal = merged.prompts.find((prompt) => prompt.id === "builtin:goal");
    expect(goal?.title).toBe("Meu objetivo");
    expect(withBuiltins(merged, NOW)).toBe(merged);
  });
});

describe("persistência", () => {
  it("serializa e relê mantendo o esquema", () => {
    const library = withBuiltins(libraryWith(sample("p1")), NOW);
    const reloaded = parseLibrary(serializeLibrary(library), NOW);
    expect(reloaded.schema).toBe(PROMPT_LIBRARY_SCHEMA);
    expect(reloaded.prompts).toHaveLength(library.prompts.length);
  });

  it("tolera JSON inválido, esquema desconhecido e registros inválidos", () => {
    expect(parseLibrary("{não é json", NOW).prompts).toEqual([]);
    expect(parseLibrary(JSON.stringify({ schema: "outro", prompts: [] }), NOW).prompts).toEqual(
      [],
    );
    const raw = JSON.stringify({
      schema: PROMPT_LIBRARY_SCHEMA,
      prompts: [
        { id: "ok", title: "Válido", body: "corpo" },
        { id: "", title: "Sem id", body: "corpo" },
        { id: "sem-corpo", title: "Sem corpo" },
        { id: "ok", title: "Duplicado", body: "corpo" },
      ],
    });
    const parsed = parseLibrary(raw, NOW);
    expect(parsed.prompts.map((prompt) => prompt.title)).toEqual(["Válido"]);
    expect(parsed.prompts[0]).toMatchObject({ uses: 0, lastUsedAt: null, favorite: false });
  });

  it("grava e carrega pelo armazenamento injetado", () => {
    const storage = memoryStorage();
    expect(savePromptLibrary(libraryWith(sample("p1")), storage)).toBe(true);
    expect(storage.snapshot()[PROMPT_STORAGE_KEY]).toContain("p1");
    const loaded = loadPromptLibrary(storage, NOW);
    expect(loaded.prompts).toHaveLength(5);
    expect(loaded.prompts.map((prompt) => prompt.id)).toContain("p1");
  });

  it("degrada para os prompts iniciais quando o armazenamento é negado", () => {
    expect(loadPromptLibrary(deniedStorage, NOW).prompts).toHaveLength(4);
    expect(savePromptLibrary(libraryWith(sample("p1")), deniedStorage)).toBe(false);
  });

  it("funciona sem armazenamento disponível", () => {
    expect(loadPromptLibrary(undefined, NOW).prompts).toHaveLength(4);
    expect(savePromptLibrary(libraryWith(sample("p1")))).toBe(false);
  });
});

describe("importLibrary", () => {
  it("importa prompts novos e ignora repetidos", () => {
    const incoming = serializeLibrary(libraryWith(sample("p9")));
    const result = importLibrary(libraryWith(sample("p1")), incoming, NOW);
    expect(result.error).toBeNull();
    expect(result.imported).toBe(1);
    expect(result.library.prompts.map((prompt) => prompt.id)).toEqual(["p1", "p9"]);

    const again = importLibrary(result.library, incoming, NOW);
    expect(again.imported).toBe(0);
    expect(again.error).toBe("Todos os prompts já existem.");

    const invalid = importLibrary(result.library, "nada", NOW);
    expect(invalid.error).toBe("Nenhum prompt válido encontrado.");
    expect(invalid.library).toBe(result.library);
  });
});
