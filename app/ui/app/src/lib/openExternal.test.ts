import { describe, it, expect, vi, afterEach } from "vitest";
import { openExternal } from "./openExternal";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("openExternal", () => {
  it("uses the native binding when present (desktop app webview)", () => {
    const native = vi.fn();
    const open = vi.fn();
    vi.stubGlobal("window", { openExternal: native, open });
    openExternal("https://ollama.com/");
    expect(native).toHaveBeenCalledWith("https://ollama.com/");
    expect(open).not.toHaveBeenCalled();
  });

  it("falls back to window.open when the native binding is absent (browser)", () => {
    const open = vi.fn();
    vi.stubGlobal("window", { open });
    openExternal("https://example.com/auth");
    expect(open).toHaveBeenCalledWith("https://example.com/auth", "_blank", "noopener,noreferrer");
  });

  it("is a no-op when window is undefined (SSR/node)", () => {
    vi.stubGlobal("window", undefined);
    expect(() => openExternal("https://x")).not.toThrow();
  });
});
