import { describe, expect, it } from "vitest";
import { resolveViteHost } from "./viteHostPolicy";

describe("Vite host policy", () => {
  it("binds to loopback by default", () => {
    expect(resolveViteHost({})).toBe("127.0.0.1");
  });

  it("rejects LAN exposure without explicit opt-in", () => {
    expect(() => resolveViteHost({ HADES_UI_HOST: "0.0.0.0" })).toThrow(
      /HADES_UI_ALLOW_LAN=true/,
    );
  });

  it("rejects LAN exposure unless backend auth is explicitly required", () => {
    expect(() =>
      resolveViteHost({
        HADES_UI_HOST: "0.0.0.0",
        HADES_UI_ALLOW_LAN: "true",
      }),
    ).toThrow(/HADES_UI_AUTH_REQUIRED=true/);
  });

  it("allows LAN only with both explicit controls", () => {
    expect(
      resolveViteHost({
        HADES_UI_HOST: "192.168.1.20",
        HADES_UI_ALLOW_LAN: "true",
        HADES_UI_AUTH_REQUIRED: "true",
      }),
    ).toBe("192.168.1.20");
  });
});
