import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SidebarLayout } from "./layout";

describe("SidebarLayout", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

	it("keeps the macOS title offset in step with the open sidebar", () => {
    vi.stubGlobal("window", { OLLAMA_PLATFORM: "darwin" });

    const html = renderToStaticMarkup(
      <SidebarLayout title="Connect your apps" sidebar={<nav />}>
        <div />
      </SidebarLayout>,
    );

	expect(html).toContain("pl-6");
    expect(html).toContain("transition-[padding-left]");
    expect(html).toContain("duration-300");
  });

  it("starts with the sidebar closed on narrow viewports", () => {
    vi.stubGlobal("window", {
      OLLAMA_PLATFORM: "darwin",
      matchMedia: (query: string) => ({ matches: query === "(max-width: 767px)" }),
    });

    const html = renderToStaticMarkup(
      <SidebarLayout title="Tarefas" sidebar={<nav data-testid="sidebar" />}>
        <div />
      </SidebarLayout>,
    );

    expect(html).toContain('aria-label="Show sidebar"');
    expect(html).not.toContain('data-testid="sidebar"');
  });
});
