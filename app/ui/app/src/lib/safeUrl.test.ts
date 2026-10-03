import { describe, expect, it } from "vitest";

import { safeHttpUrl, safeImageSrc } from "./safeUrl";

describe("safeHttpUrl", () => {
  it("allows http(s) and same-origin relative paths", () => {
    expect(safeHttpUrl("https://example.test/x")).toBe("https://example.test/x");
    expect(safeHttpUrl("http://example.test/x")).toBe("http://example.test/x");
    expect(safeHttpUrl("/api/agent/v1/x")).toBe("/api/agent/v1/x");
  });

  it("blocks script-executing and other dangerous schemes", () => {
    expect(safeHttpUrl("javascript:alert(1)")).toBe("#");
    expect(safeHttpUrl("JavaScript:alert(1)")).toBe("#");
    expect(safeHttpUrl("data:text/html,<script>alert(1)</script>")).toBe("#");
    expect(safeHttpUrl("vbscript:msgbox(1)")).toBe("#");
    expect(safeHttpUrl("//evil.example")).toBe("#");
    expect(safeHttpUrl("")).toBe("#");
    expect(safeHttpUrl(undefined)).toBe("#");
  });
});

describe("safeImageSrc", () => {
  it("allows data:image, http(s) and relative paths", () => {
    expect(safeImageSrc("data:image/png;base64,AAAA")).toBe(
      "data:image/png;base64,AAAA",
    );
    expect(safeImageSrc("https://example.test/a.png")).toBe(
      "https://example.test/a.png",
    );
    expect(safeImageSrc("/frames/a.png")).toBe("/frames/a.png");
  });

  it("blocks non-image data URLs and script schemes", () => {
    expect(safeImageSrc("data:text/html,<script>alert(1)</script>")).toBe("");
    expect(safeImageSrc("javascript:alert(1)")).toBe("");
    expect(safeImageSrc("//evil.example/a.png")).toBe("");
    expect(safeImageSrc(undefined)).toBe("");
  });
});
