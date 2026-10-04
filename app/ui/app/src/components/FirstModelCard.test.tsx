import {
  act,
  create,
  type ReactTestRenderer,
  type ReactTestInstance,
} from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { pullModel } from "@/api";
import { useModels, useRefetchModels } from "@/hooks/useModels";
import { useSettings } from "@/hooks/useSettings";
import {
  FIRST_MODEL_ERROR_MESSAGE,
  RECOMMENDED_FIRST_MODEL,
} from "@/lib/firstModel";
import { FirstModelCard } from "./FirstModelCard";

vi.mock("@/api", () => ({ pullModel: vi.fn() }));
vi.mock("@/hooks/useModels", () => ({
  useModels: vi.fn(),
  useRefetchModels: vi.fn(),
}));
vi.mock("@/hooks/useSettings", () => ({ useSettings: vi.fn() }));

const mockPull = vi.mocked(pullModel);
const mockUseModels = vi.mocked(useModels);
const mockUseRefetch = vi.mocked(useRefetchModels);
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
  // Default: no local models.
  mockUseModels.mockReturnValue({ data: [], isLoading: false } as never);
  mockUseRefetch.mockReturnValue(refetch as never);
  mockUseSettings.mockReturnValue({ settings: {}, setSettings } as never);
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
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
      data: [{ model: "qwen2.5:0.5b", digest: "abc" }],
      isLoading: false,
    } as never);
    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(<FirstModelCard />);
      await Promise.resolve();
    });
    expect(renderer.root.findAllByType("section")).toHaveLength(0);
  });

  it("downloads the recommended first model and selects it", async () => {
    mockPull.mockImplementation(async function* () {
      yield { status: "pulling", completed: 50, total: 100 };
      yield { status: "success", completed: 100, total: 100, done: true };
    });

    let renderer!: ReactTestRenderer;
    await act(async () => {
      renderer = create(<FirstModelCard />);
      await Promise.resolve();
    });

    // The field starts empty and shows the small, predictable recommended model
    // as a placeholder; leaving it empty downloads that recommended model.
    const input = renderer.root.findByType("input");
    expect(input.props.value).toBe("");
    expect(input.props.placeholder).toBe(RECOMMENDED_FIRST_MODEL);

    const button = findButton(renderer, "Baixar e começar");
    expect(button).toBeDefined();
    await act(async () => {
      button!.props.onClick();
      for (let i = 0; i < 6; i++) await Promise.resolve();
    });

    expect(mockPull).toHaveBeenCalledWith(
      RECOMMENDED_FIRST_MODEL,
      expect.anything(),
    );
    expect(setSettings).toHaveBeenCalledWith({
      SelectedModel: RECOMMENDED_FIRST_MODEL,
    });
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
