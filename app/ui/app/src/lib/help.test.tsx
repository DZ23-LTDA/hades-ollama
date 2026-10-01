import { afterEach, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { HelpDialog } from "@/components/HelpDialog";
import { HELP_LINKS, newTaskShortcut } from "./help";

describe("help", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("labels the new-task shortcut per platform", () => {
    vi.stubGlobal("navigator", { platform: "MacIntel" });
    expect(newTaskShortcut()).toBe("⌘K");
    vi.stubGlobal("navigator", { platform: "Win32" });
    expect(newTaskShortcut()).toBe("Ctrl K");
  });

  it("points every help link at the project repository", () => {
    for (const link of HELP_LINKS) {
      expect(link.href).toMatch(
        /^https:\/\/github\.com\/DZ23-LTDA\/ollama-classe-a-plus/,
      );
    }
  });

  it("renders nothing when closed", () => {
    expect(
      renderToStaticMarkup(<HelpDialog open={false} onClose={vi.fn()} />),
    ).toBe("");
  });
});
