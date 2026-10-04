import { describe, it, expect } from "vitest";
import { parseGeneratedComponents, buildGenerationPrompt } from "./studioAI";

describe("buildGenerationPrompt", () => {
  it("includes the allowed types and the user description", () => {
    const p = buildGenerationPrompt("uma landing de cafeteria");
    expect(p).toContain("pricing");
    expect(p).toContain("uma landing de cafeteria");
    expect(p).toContain('"components"');
  });
});

describe("parseGeneratedComponents", () => {
  it("parses a {components:[...]} object", () => {
    const out = parseGeneratedComponents(
      JSON.stringify({ components: [{ type: "hero", props: { title: "Oi" } }, { type: "footer", props: {} }] }),
    );
    expect(out).toHaveLength(2);
    expect(out[0]).toMatchObject({ type: "hero", props: { title: "Oi" } });
    expect(out[0].id).toMatch(/^ai_1_hero$/);
  });

  it("parses a bare array", () => {
    const out = parseGeneratedComponents('[{"type":"heading","props":{"text":"T"}}]');
    expect(out).toHaveLength(1);
    expect(out[0].type).toBe("heading");
  });

  it("strips a ```json fence", () => {
    const out = parseGeneratedComponents('```json\n{"components":[{"type":"paragraph","props":{"text":"x"}}]}\n```');
    expect(out).toHaveLength(1);
    expect(out[0].type).toBe("paragraph");
  });

  it("extracts JSON embedded in prose", () => {
    const out = parseGeneratedComponents('Claro! Aqui está: {"components":[{"type":"button","props":{"label":"Ir"}}]} pronto.');
    expect(out).toHaveLength(1);
    expect(out[0].props).toEqual({ label: "Ir" });
  });

  it("drops unknown component types", () => {
    const out = parseGeneratedComponents(
      JSON.stringify({ components: [{ type: "heading", props: {} }, { type: "iframe", props: {} }, { type: "script", props: {} }] }),
    );
    expect(out.map((c) => c.type)).toEqual(["heading"]);
  });

  it("coerces non-string prop values to strings and arrays to comma lists", () => {
    const out = parseGeneratedComponents(
      JSON.stringify({ components: [{ type: "metric", props: { value: 42, label: "KPI", items: ["a", "b"] } }] }),
    );
    expect(out[0].props).toEqual({ value: "42", label: "KPI", items: "a, b" });
  });

  it("returns empty on invalid JSON without throwing", () => {
    expect(parseGeneratedComponents("não é json")).toEqual([]);
    expect(parseGeneratedComponents("")).toEqual([]);
    expect(parseGeneratedComponents("{broken")).toEqual([]);
  });

  it("caps at 40 components", () => {
    const many = { components: Array.from({ length: 100 }, () => ({ type: "divider", props: {} })) };
    expect(parseGeneratedComponents(JSON.stringify(many))).toHaveLength(40);
  });

  it("truncates overly long prop values", () => {
    const long = "x".repeat(5000);
    const out = parseGeneratedComponents(JSON.stringify({ components: [{ type: "paragraph", props: { text: long } }] }));
    expect(out[0].props!.text.length).toBe(2000);
  });
});
