import {
  act,
  create,
  type ReactTestRenderer,
  type ReactTestInstance,
} from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { pullModel } from "@/api";
import { useModels, useRefetchModels } from "@/hooks/useModels";
import { useFeaturedModels } from "@/hooks/useFeaturedModels";
import { useSettings } from "@/hooks/useSettings";
import {
  FIRST_MODEL_ERROR_MESSAGE,
  FIRST_MODEL_FALLBACK,
  pickFirstModel,
} from "@/lib/firstModel";
import { FirstModelCard } from "./FirstModelCard";

vi.mock("@/api", () => ({ pullModel: vi.fn() }));
vi.mock("@/hooks/useModels", () => ({
  useModels: vi.fn(),
  useRefetchModels: vi.fn(),
}));
vi.mock("@/hooks/useFeaturedModels", () => ({ useFeaturedModels: vi.fn() }));
vi.mock("@/hooks/useSettings", () => ({ useSettings: vi.fn() }));

const mockPull = vi.mocked(pullModel);
const mockUseModels = vi.mocked(useModels);
const mockUseRefetch = vi.mocked(useRefetchModels);
const mockUseFeatured = vi.mocked(useFeaturedModels);
const mockUseSettings = vi.mocked(useSettings);

const setSettings = vi.fn();
const refetch = vi.fn().mockResolvedValue(undefined);

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

beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
  setSettings.mockReset();
  refetch.mockReset().mockResolvedValue(undefined);
  // Default: no local models, one small recommendation.
  mockUseModels.mockReturnValue({ data: [], isLoading: false } as never);
  mockUseRefetch.mockReturnValue(refetch as never);
  mockUseFeatured.mockReturnValue({
    data: [{ model: "gemma3:1b", description: "", vram_bytes: 1 }],
  } as never);
  mockUseSettings.mockReturnValue({ settings: {}, setSettings } as never);
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("pickFirstModel", () => {
  it("falls back to the small default when there is no recommendation", () => {
    expect(pickFirstModel(undefined)).toBe(FIRST_MODEL_FALLBACK);
    expect(pickFirstModel([])).toBe(FIRST_MODEL_FALLBACK);
  });

  it("ignores cloud models and picks the smallest by VRAM", () => {
    const picked = pickFirstModel([
      { model: "big:cloud", description: "", vram_bytes: 1 },
      { model: "qwen3.5", description: "", vram_bytes: 14 },
      { model: "gemma4", description: "", vram_bytes: 12 },
    ]);
    expect(picked).toBe("gemma4");
  });
});

describe("FirstModelCard", () => {
  it("renders nothing while models are loading", async () => {
    mockUseModels.mockReturnValue({ data: [], isLoading: true } as never);
    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(<FirstModelCard />);
      await Promise.resolve();
    });
    expect(renderer.root.findAllByType("section")).toHaveLength(0);
  });

  it("renders nothing when a local model already exists", async () => {
    mockUseModels.mockReturnValue({
      data: [{ model: "gemma3:1b", digest: "abc" }],
      isLoading: false,
    } as never);
    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(<FirstModelCard />);
      await Promise.resolve();
    });
    expect(renderer.root.findAllByType("section")).toHaveLength(0);
  });

  it("downloads the recommended model and selects it", async () => {
    mockPull.mockImplementation(async function* () {
      yield { status: "pulling", completed: 50, total: 100 };
      yield { status: "success", completed: 100, total: 100, done: true };
    });

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(<FirstModelCard />);
      await Promise.resolve();
    });

    // The recommended model is prefilled.
    const input = renderer.root.findByType("input");
    expect(input.props.value).toBe("gemma3:1b");

    const button = findButton(renderer, "Baixar e começar");
    expect(button).toBeDefined();
    await act(async () => {
      button!.props.onClick();
      for (let i = 0; i < 6; i++) await Promise.resolve();
    });

    expect(mockPull).toHaveBeenCalledWith("gemma3:1b", expect.anything());
    expect(setSettings).toHaveBeenCalledWith({ SelectedModel: "gemma3:1b" });
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("shows an error message when the pull fails", async () => {
    mockPull.mockImplementation(async function* () {
      yield { status: "pulling", completed: 0, total: 0 };
      throw new Error("boom");
    });

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(<FirstModelCard />);
      await Promise.resolve();
    });

    const button = findButton(renderer, "Baixar e começar");
    await act(async () => {
      button!.props.onClick();
      for (let i = 0; i < 6; i++) await Promise.resolve();
    });

    expect(textContent(renderer.root)).toContain(FIRST_MODEL_ERROR_MESSAGE);
    expect(setSettings).not.toHaveBeenCalled();
  });

  it("can be skipped", async () => {
    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(<FirstModelCard />);
      await Promise.resolve();
    });

    const skip = findButton(renderer, "Pular por enquanto");
    await act(async () => {
      skip!.props.onClick();
      await Promise.resolve();
    });

    expect(renderer.root.findAllByType("section")).toHaveLength(0);
  });
});
