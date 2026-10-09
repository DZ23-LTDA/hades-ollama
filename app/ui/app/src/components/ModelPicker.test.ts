import { describe, expect, it } from "vitest";
import { Model } from "@/gotypes";
import { getModelCostTag, isModelSelectable, modelGroup, sortModelsClean } from "./modelPickerUtils";

describe("modelGroup", () => {
  it("separates local, router, cli_subscription, and remote provider models", () => {
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
    expect(
      modelGroup(
        new Model({
          model: "claude-3-7-sonnet",
          kind: "cli_subscription",
          available: true,
          status: "PASS",
        }),
      ),
    ).toBe("Assinatura (CLI)");
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
    expect(
      modelGroup(
        new Model({
          model: "gemini-2.5-pro",
          kind: "cli_subscription",
          available: false,
          status: "NOT_PRESENT",
          reason: "CLI ausente",
        }),
      ),
    ).toBe("Indisponíveis");
    expect(
      modelGroup(
        new Model({
          model: "o3-mini",
          kind: "cli_subscription",
          available: false,
          status: "NOT_CONFIGURED",
          reason: "fazer login",
        }),
      ),
    ).toBe("Indisponíveis");
  });
});

describe("getModelCostTag", () => {
  it("returns the human pt-BR cost labels and only tags cli_subscription when PASS", () => {
    const activeSub = new Model({
      model: "claude-3-7-sonnet",
      kind: "cli_subscription",
      available: true,
      status: "PASS",
    });
    const loggedOutSub = new Model({
      model: "gpt-4o",
      kind: "cli_subscription",
      available: false,
      status: "NOT_CONFIGURED",
    });
    const missingSub = new Model({
      model: "gemini-2.5-pro",
      kind: "cli_subscription",
      available: false,
      status: "NOT_PRESENT",
    });
    const localFree = new Model({
      model: "llama3:8b",
      kind: "local",
      available: true,
      status: "PASS",
      cost_tag: "0-local",
    });
    const standardModel = new Model({
      model: "qwen3:8b",
      kind: "local",
      available: true,
      status: "PASS",
    });

    // cli_subscription (PASS) is included in the subscription.
    expect(getModelCostTag(activeSub)).toBe("Incluído na assinatura");
    // A logged-out / missing subscription has no cost label.
    expect(getModelCostTag(loggedOutSub)).toBeUndefined();
    expect(getModelCostTag(missingSub)).toBeUndefined();
    // Local models carrying the raw "0-local" tag read as "Grátis".
    expect(getModelCostTag(localFree)).toBe("Grátis");
    // A model without a cost_tag has no label.
    expect(getModelCostTag(standardModel)).toBeUndefined();
  });
});

describe("isModelSelectable", () => {
  it("allows only models with status PASS or available true", () => {
    const localPass = new Model({ model: "llama3:8b", kind: "local", available: true, status: "PASS" });
    const remotePass = new Model({ model: "groq/llama-3", kind: "remote", available: true, status: "PASS" });
    const subPass = new Model({ model: "copilot/gpt-4o", kind: "cli_subscription", available: true, status: "PASS" });
    const unconfigured = new Model({
      model: "anthropic/claude-3-5",
      kind: "remote",
      available: false,
      status: "NOT_CONFIGURED",
      reason: "sem chave",
    });
    const subUnconfigured = new Model({
      model: "o3-mini",
      kind: "cli_subscription",
      available: false,
      status: "NOT_CONFIGURED",
      reason: "fazer login",
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
    expect(isModelSelectable(subPass)).toBe(true);
    expect(isModelSelectable(unconfigured)).toBe(false);
    expect(isModelSelectable(subUnconfigured)).toBe(false);
    expect(isModelSelectable(failing)).toBe(false);
  });
});

describe("sortModelsClean", () => {
  it("places all PASS models first and separates all unavailable models at the end", () => {
    const models = [
      new Model({ model: "openai/gpt-4o", available: false, status: "NOT_CONFIGURED" }),
      new Model({ model: "llama3:8b", available: true, status: "PASS" }),
      new Model({ model: "claude-3-7-sonnet", kind: "cli_subscription", available: true, status: "PASS" }),
      new Model({ model: "anthropic/claude", available: false, status: "FAIL" }),
      new Model({ model: "gemini-2.5-pro", kind: "cli_subscription", available: false, status: "NOT_PRESENT" }),
      new Model({ model: "auto/coding", kind: "router", available: true, status: "PASS" }),
    ];

    const sorted = sortModelsClean(models);

    // The first 3 must be the selectable ones (PASS)
    expect(sorted[0].model).toBe("llama3:8b");
    expect(sorted[1].model).toBe("claude-3-7-sonnet");
    expect(sorted[2].model).toBe("auto/coding");
    expect(isModelSelectable(sorted[0])).toBe(true);
    expect(isModelSelectable(sorted[1])).toBe(true);
    expect(isModelSelectable(sorted[2])).toBe(true);

    // The last 3 must be the unavailable ones
    expect(sorted[3].model).toBe("openai/gpt-4o");
    expect(sorted[4].model).toBe("anthropic/claude");
    expect(sorted[5].model).toBe("gemini-2.5-pro");
    expect(isModelSelectable(sorted[3])).toBe(false);
    expect(isModelSelectable(sorted[4])).toBe(false);
    expect(isModelSelectable(sorted[5])).toBe(false);
  });
});
