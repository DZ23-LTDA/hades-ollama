import { describe, expect, it } from "vitest";
import { Model } from "@/gotypes";
import { isModelSelectable, modelGroup, sortModelsClean } from "./modelPickerUtils";

describe("modelGroup", () => {
  it("separates local, router, and remote provider models", () => {
    expect(modelGroup(new Model({ model: "qwen3:8b", kind: "local" }))).toBe(
      "Modelos locais",
    );
    expect(
      modelGroup(new Model({ model: "auto/coding", kind: "router" })),
    ).toBe("Roteamento inteligente");
    expect(
      modelGroup(
        new Model({
          model: "groq/llama-3.3-70b-versatile",
          kind: "remote",
          provider: "groq",
        }),
      ),
    ).toBe("groq");
  });

  it("assigns unavailable models to the Indisponíveis group", () => {
    expect(
      modelGroup(
        new Model({
          model: "openai/gpt-4o",
          kind: "remote",
          provider: "openai",
          available: false,
          status: "NOT_CONFIGURED",
          reason: "chave ausente",
        }),
      ),
    ).toBe("Indisponíveis");
  });
});

describe("isModelSelectable", () => {
  it("allows only models with status PASS or available true", () => {
    const localPass = new Model({ model: "llama3:8b", kind: "local", available: true, status: "PASS" });
    const remotePass = new Model({ model: "groq/llama-3", kind: "remote", available: true, status: "PASS" });
    const unconfigured = new Model({
      model: "anthropic/claude-3-5",
      kind: "remote",
      available: false,
      status: "NOT_CONFIGURED",
      reason: "sem chave",
    });
    const failing = new Model({
      model: "mistral/large",
      kind: "remote",
      available: false,
      status: "FAIL",
      reason: "500 Internal Server Error",
    });

    expect(isModelSelectable(localPass)).toBe(true);
    expect(isModelSelectable(remotePass)).toBe(true);
    expect(isModelSelectable(unconfigured)).toBe(false);
    expect(isModelSelectable(failing)).toBe(false);
  });
});

describe("sortModelsClean", () => {
  it("places all PASS models first and separates all unavailable models at the end", () => {
    const models = [
      new Model({ model: "openai/gpt-4o", available: false, status: "NOT_CONFIGURED" }),
      new Model({ model: "llama3:8b", available: true, status: "PASS" }),
      new Model({ model: "anthropic/claude", available: false, status: "FAIL" }),
      new Model({ model: "auto/coding", kind: "router", available: true, status: "PASS" }),
    ];

    const sorted = sortModelsClean(models);

    // The first 2 must be the selectable ones
    expect(sorted[0].model).toBe("llama3:8b");
    expect(sorted[1].model).toBe("auto/coding");
    // The last 2 must be the unavailable ones
    expect(sorted[2].model).toBe("openai/gpt-4o");
    expect(sorted[3].model).toBe("anthropic/claude");
    expect(isModelSelectable(sorted[2])).toBe(false);
    expect(isModelSelectable(sorted[3])).toBe(false);
  });
});
