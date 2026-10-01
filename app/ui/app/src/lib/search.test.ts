import { describe, expect, it } from "vitest";
import { SEARCH_PAGES, isTypingTarget, searchItems } from "./search";

describe("searchItems", () => {
  const chats = [
    { id: "c1", label: "Plano de marketing", hint: "Conversa", href: "/c/1" },
    {
      id: "c2",
      label: "Bug no instalador",
      hint: "erro de configuração",
      href: "/c/2",
    },
  ];
  const items = [...SEARCH_PAGES, ...chats];

  it("ignores accents and case and ranks prefix matches first", () => {
    const results = searchItems(items, "CONF");
    expect(results[0].label).toBe("Configurações");
    expect(results.map((item) => item.id)).toContain("c2");
  });

  it("finds pages such as Conectores", () => {
    expect(searchItems(items, "conect")[0].href).toBe("/connectors");
  });

  it("returns everything (capped) for an empty query and nothing for no match", () => {
    expect(searchItems(items, "   ", 5)).toHaveLength(5);
    expect(searchItems(items, "zzzz-nada")).toEqual([]);
  });
});

describe("isTypingTarget", () => {
  it("is false for non-elements", () => {
    expect(isTypingTarget(null)).toBe(false);
  });
});
