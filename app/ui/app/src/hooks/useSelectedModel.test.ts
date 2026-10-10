import { describe, expect, it } from "vitest";

import { defaultModelRequest } from "./useSelectedModel";
import { Model } from "@/gotypes";

function model(name: string, overrides: Partial<Model> = {}): Model {
  return new Model({
    name,
    model: name,
    digest: `sha256:${name}`,
    size: 1024,
    modified_at: "2026-01-02T00:00:00Z",
    ...overrides,
  });
}

const base = {
  isLoading: false,
  modelCount: 1,
  selectedModel: "",
  current: null,
  defaultModel: model("llama3.2:3b"),
};

describe("defaultModelRequest", () => {
  it("pede o default quando nada está selecionado", () => {
    expect(defaultModelRequest(base)).toBe("llama3.2:3b");
  });

  it("mantém a seleção válida do chat", () => {
    const local = model("llama3.2:3b");
    expect(
      defaultModelRequest({
        ...base,
        selectedModel: "llama3.2:3b",
        current: local,
        defaultModel: model("gemma3:4b"),
      }),
    ).toBeNull();
  });

  it("preserva uma seleção desconhecida (nome de nuvem sintetizado)", () => {
    expect(
      defaultModelRequest({
        ...base,
        selectedModel: "gpt-oss:120b-cloud",
        current: null,
      }),
    ).toBeNull();
  });

  it("substitui um modelo de provider que o chat simples não usa", () => {
    const router = model("auto/coding", { kind: "router" });
    expect(
      defaultModelRequest({
        ...base,
        selectedModel: "auto/coding",
        current: router,
      }),
    ).toBe("llama3.2:3b");
  });

  it("não faz nada enquanto carrega ou sem modelos", () => {
    expect(defaultModelRequest({ ...base, isLoading: true })).toBeNull();
    expect(defaultModelRequest({ ...base, modelCount: 0 })).toBeNull();
  });

  it("não pede nada quando o default já é a seleção", () => {
    expect(
      defaultModelRequest({
        ...base,
        selectedModel: "llama3.2:3b",
        defaultModel: model("llama3.2:3b"),
      }),
    ).toBeNull();
  });

  it("permite uma nova tentativa para outro default", () => {
    expect(
      defaultModelRequest({
        ...base,
        defaultModel: model("gemma3:4b"),
      }),
    ).toBe("gemma3:4b");
  });
});
