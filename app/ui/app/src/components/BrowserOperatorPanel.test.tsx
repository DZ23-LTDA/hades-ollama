import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { BrowserEnvironmentStatus, BrowserSetupResult } from "@/lib/agenticClient";

const getBrowserEnvironmentMock = vi.fn<[], Promise<BrowserEnvironmentStatus>>();
const setupBrowserEnvironmentMock = vi.fn<[], Promise<BrowserSetupResult>>();

vi.mock("@/lib/agenticClient", () => ({
  getBrowserEnvironment: () => getBrowserEnvironmentMock(),
  setupBrowserEnvironment: () => setupBrowserEnvironmentMock(),
}));

import { BrowserOperatorPanel } from "./BrowserOperatorPanel";

type TestNode = ReturnType<ReactTestRenderer["root"]["findAll"]>[number];
const fakeEvent = { stopPropagation() {}, preventDefault() {} };

function textOf(node: TestNode): string {
  const parts: string[] = [];
  const walk = (children: unknown): void => {
    if (typeof children === "string") parts.push(children);
    else if (Array.isArray(children)) children.forEach(walk);
    else if (children && typeof children === "object" && "children" in (children as TestNode).props)
      walk((children as TestNode).props.children);
  };
  walk(node.props.children);
  return parts.join(" ");
}
function hasTestId(r: ReactTestRenderer, id: string): boolean {
  return r.root.findAll((n) => n.props?.["data-testid"] === id).length > 0;
}
async function flush() {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}

const notReady: BrowserEnvironmentStatus = {
  python_ok: true,
  python_path: "/usr/bin/python3",
  playwright_ok: false,
  chromium_ok: false,
  ready: false,
  guidance: ["Instale o Playwright: python -m pip install playwright"],
  can_auto_setup: true,
};
const ready: BrowserEnvironmentStatus = {
  python_ok: true,
  playwright_ok: true,
  chromium_ok: true,
  ready: true,
  guidance: [],
  can_auto_setup: true,
};

describe("BrowserOperatorPanel", () => {
  beforeEach(() => {
    getBrowserEnvironmentMock.mockReset();
    setupBrowserEnvironmentMock.mockReset();
  });

  it("shows ready state", async () => {
    getBrowserEnvironmentMock.mockResolvedValue(ready);
    let r!: ReactTestRenderer;
    await act(async () => {
      r = create(<BrowserOperatorPanel />);
    });
    await flush();
    expect(hasTestId(r, "browser-ready")).toBe(true);
    act(() => r.unmount());
  });

  it("shows guidance and runs auto-setup when not ready", async () => {
    getBrowserEnvironmentMock.mockResolvedValue(notReady);
    setupBrowserEnvironmentMock.mockResolvedValue({
      steps: [{ description: "Instalando Playwright (pip)", ok: true }],
      status: ready,
    });
    let r!: ReactTestRenderer;
    await act(async () => {
      r = create(<BrowserOperatorPanel />);
    });
    await flush();
    expect(hasTestId(r, "browser-guidance")).toBe(true);

    const setupBtn = r.root
      .findAll((n) => n.type === "button" && textOf(n).includes("Preparar navegador"))
      .at(0);
    expect(setupBtn).toBeTruthy();
    await act(async () => {
      setupBtn!.props.onClick(fakeEvent);
    });
    await flush();
    expect(setupBrowserEnvironmentMock).toHaveBeenCalledTimes(1);
    expect(hasTestId(r, "browser-ready")).toBe(true);
    expect(hasTestId(r, "browser-setup-steps")).toBe(true);
    act(() => r.unmount());
  });
});
