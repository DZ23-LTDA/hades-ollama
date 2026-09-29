import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { RouteErrorFallback } from "./RouteErrorFallback";

describe("RouteErrorFallback", () => {
  it("always offers a way out of a failed page", () => {
    const html = renderToStaticMarkup(
      <RouteErrorFallback
        error={new Error("boom")}
        reset={vi.fn()}
        info={{ componentStack: "" }}
      />,
    );

    expect(html).toContain('role="alert"');
    expect(html).toContain("Voltar");
    expect(html).toContain("Ir para o início");
    expect(html).toContain("Recarregar");
    // Details stay collapsed until requested.
    expect(html).not.toContain("boom");
  });
});
