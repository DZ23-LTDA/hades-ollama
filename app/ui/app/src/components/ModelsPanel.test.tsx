import {
  act,
  create,
  type ReactTestRenderer,
  type ReactTestInstance,
} from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useModels } from "@/hooks/useModels";
import { useModelPull } from "@/hooks/useModelPull";
import { useSettings } from "@/hooks/useSettings";
import { RECOMMENDED_FIRST_MODEL } from "@/lib/firstModel";
import { ModelsPanel } from "./ModelsPanel";

vi.mock("@/hooks/useModels", () => ({ useModels: vi.fn() }));
vi.mock("@/hooks/useSettings", () => ({ useSettings: vi.fn() }));
vi.mock("@/hooks/useModelPull", () => ({ useModelPull: vi.fn() }));

const mockUseModels = vi.mocked(useModels);
const mockUseSettings = vi.mocked(useSettings);
const mockUseModelPull = vi.mocked(useModelPull);

const setSettings = vi.fn();
const pull = vi.fn();

function model(name: string, digest?: string) {
  return { model: name, digest, isCloud: () => name.endsWith("cloud") };
}

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

function findButton(renderer: ReactTestRenderer, label: string) {
  return renderer.root
    .findAllByType("button")
    .find((candidate) => textContent(candidate).includes(label));
}

async function render(): Promise<ReactTestRenderer> {
  let renderer!: ReactTestRenderer;
  await act(async () => {
    renderer = create(<ModelsPanel />);
    await Promise.resolve();
  });
  return renderer;
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
  setSettings.mockReset();
  pull.mockReset().mockResolvedValue(true);
  mockUseModels.mockReturnValue({
    data: [model("llama3:8b", "d1"), model("qwen2.5:0.5b", "d2"), model("glm:cloud")],
  } as never);
  mockUseSettings.mockReturnValue({
    settings: { selectedModel: "llama3:8b" },
    setSettings,
  } as never);
  mockUseModelPull.mockReturnValue({
    pulling: false,
    progress: { completed: 0, total: 0 },
    status: null,
    error: null,
    pull,
    cancel: vi.fn(),
    reset: vi.fn(),
  } as never);
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("ModelsPanel", () => {
  it("lists downloaded local models and marks the active one", async () => {
    const renderer = await render();
    const text = textContent(renderer.root);
    expect(text).toContain("llama3:8b");
    expect(text).toContain("qwen2.5:0.5b");
    // Cloud models are not local installs.
    expect(text).not.toContain("glm:cloud");
    expect(text).toContain("Ativo");
    expect(findButton(renderer, "Usar")).toBeDefined();
  });

  it("makes a model active when 'Usar' is clicked", async () => {
    const renderer = await render();
    const use = findButton(renderer, "Usar");
    await act(async () => {
      use!.props.onClick();
      await Promise.resolve();
    });
    expect(setSettings).toHaveBeenCalledWith({ SelectedModel: "qwen2.5:0.5b" });
  });

  it("downloads a new model and selects it", async () => {
    const renderer = await render();
    const input = renderer.root.findByType("input");
    expect(input.props.value).toBe(RECOMMENDED_FIRST_MODEL);
    const button = findButton(renderer, "Baixar modelo");
    await act(async () => {
      button!.props.onClick();
      for (let i = 0; i < 6; i++) await Promise.resolve();
    });
    expect(pull).toHaveBeenCalledWith(RECOMMENDED_FIRST_MODEL);
    expect(setSettings).toHaveBeenCalledWith({
      SelectedModel: RECOMMENDED_FIRST_MODEL,
    });
  });

  it("shows an empty state when nothing is installed", async () => {
    mockUseModels.mockReturnValue({ data: [model("glm:cloud")] } as never);
    const renderer = await render();
    expect(textContent(renderer.root)).toContain("Nenhum modelo instalado ainda");
  });
});
