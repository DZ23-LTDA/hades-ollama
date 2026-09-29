import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { connectorIcon } from "./connectorIcons";

describe("connectorIcon", () => {
  it("returns brand paths with brand color", () => {
    const icon = connectorIcon("pinterest");
    expect(icon.kind).toBe("svg");
    if (icon.kind === "svg") {
      expect(icon.path.length).toBeGreaterThan(20);
      expect(icon.color).toMatch(/^#[0-9A-F]{6}$/i);
    }
  });

  it("keeps near-black brands visible in dark mode", () => {
    const icon = connectorIcon("github");
    expect(icon.kind === "svg" && icon.color).toBe("currentColor");
  });

  it("uses a bundled image for brands outside simple-icons", () => {
    expect(connectorIcon("slack")).toEqual({
      kind: "image",
      src: "/connector-icons/slack.png",
    });
  });

  it("gives every catalog connector a logo", () => {
    const catalog = readFileSync(
      resolve(__dirname, "../../../../../internal/agent/connector_catalog.go"),
      "utf8",
    );
    const ids = [...catalog.matchAll(/ID: "([a-z0-9-]+)"/g)].map(
      (match) => match[1],
    );
    expect(ids.length).toBeGreaterThan(100);
    expect(ids.filter((id) => connectorIcon(id).kind === "generic")).toEqual([
      "fiscal-invoicing",
    ]);
    for (const id of ids) {
      const icon = connectorIcon(id);
      if (icon.kind === "image") {
        expect(
          existsSync(resolve(__dirname, "../../public", icon.src.slice(1))),
        ).toBe(true);
      }
    }
  });
});
