import { afterEach, describe, expect, it } from "vitest";
import { enMessages } from "./messages.en";
import { ptBRMessages } from "./messages.pt-BR";
import {
  createTranslator,
  getLanguage,
  setLanguage,
  t,
  translate,
} from "./index";

afterEach(() => {
  setLanguage("pt-BR");
});

describe("i18n", () => {
  it("devolve o texto esperado da chave em pt-BR e em inglês", () => {
    expect(translate("pt-BR", "productWorkspace.projects.title")).toBe(
      "Projetos",
    );
    expect(translate("en", "productWorkspace.projects.title")).toBe("Projects");
  });

  it("cai para o pt-BR quando a chave existe apenas no dicionário base", () => {
    const translateWithFallback = createTranslator({
      "pt-BR": { "so.pt": "Só em português" },
      en: {},
    });

    expect(translateWithFallback("en", "so.pt")).toBe("Só em português");
  });

  it("devolve a própria chave quando ela não existe em nenhum dicionário", () => {
    expect(translate("en", "chave.inexistente")).toBe("chave.inexistente");
    expect(translate("pt-BR", "chave.inexistente")).toBe("chave.inexistente");
  });

  it("interpola parâmetros nomeados sem concatenar", () => {
    expect(
      translate("pt-BR", "productWorkspace.stats.projects", { count: 3 }),
    ).toBe("3 projetos");
    expect(
      translate("en", "productWorkspace.stats.projects", { count: 3 }),
    ).toBe("3 projects");
    expect(
      translate("pt-BR", "productWorkspace.tasks.meta", {
        id: "m-1",
        steps: 2,
        artifacts: 1,
        model: "qwen2.5:7b",
      }),
    ).toBe("m-1 · 2 passos · 1 artefatos · qwen2.5:7b");
  });

  it("o conjunto de chaves de en é superconjunto do de pt-BR", () => {
    const missing = Object.keys(ptBRMessages).filter(
      (key) => !(key in enMessages),
    );

    expect(
      missing,
      `Chaves ausentes em messages.en.ts: ${missing.join(", ")}`,
    ).toEqual([]);
  });

  it("t lê o idioma activo", () => {
    expect(getLanguage()).toBe("pt-BR");
    expect(t("productWorkspace.projects.title")).toBe("Projetos");

    setLanguage("en");
    expect(getLanguage()).toBe("en");
    expect(t("productWorkspace.projects.title")).toBe("Projects");
  });
});
