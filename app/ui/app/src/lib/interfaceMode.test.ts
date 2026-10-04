import { describe, expect, it } from "vitest";
import { getStoredInterfaceMode, persistInterfaceMode } from "./interfaceMode";

describe("interface mode", () => {
  it("defaults to advanced (full menu) and selects simple only when explicitly stored", () => {
    expect(getStoredInterfaceMode()).toBe("advanced");
    expect(getStoredInterfaceMode({ getItem: () => "simple" })).toBe("simple");
    expect(getStoredInterfaceMode({ getItem: () => "advanced" })).toBe("advanced");
    expect(getStoredInterfaceMode({ getItem: () => "invalid" })).toBe("advanced");
  });

  it("does not break when storage is unavailable", () => {
    expect(getStoredInterfaceMode({ getItem: () => { throw new Error("blocked"); } })).toBe("advanced");
    expect(() => persistInterfaceMode("advanced", { setItem: () => { throw new Error("blocked"); } })).not.toThrow();
  });
});
