import { describe, expect, it } from "vitest";
import { getStoredInterfaceMode, persistInterfaceMode } from "./interfaceMode";

describe("interface mode", () => {
  it("defaults to simple and accepts advanced only when explicitly stored", () => {
    expect(getStoredInterfaceMode()).toBe("simple");
    expect(getStoredInterfaceMode({ getItem: () => "advanced" })).toBe("advanced");
    expect(getStoredInterfaceMode({ getItem: () => "invalid" })).toBe("simple");
  });

  it("does not break when storage is unavailable", () => {
    expect(getStoredInterfaceMode({ getItem: () => { throw new Error("blocked"); } })).toBe("simple");
    expect(() => persistInterfaceMode("advanced", { setItem: () => { throw new Error("blocked"); } })).not.toThrow();
  });
});
