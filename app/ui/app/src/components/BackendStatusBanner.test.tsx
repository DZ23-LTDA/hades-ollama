import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    isPending: false,
    data: false,
    refetch: vi.fn(),
  }),
}));

import { BackendStatusBanner } from "./BackendStatusBanner";

describe("BackendStatusBanner", () => {
  it("explica que o backend está offline e oferece retry", () => {
    const html = renderToStaticMarkup(<BackendStatusBanner />);
    expect(html).toContain("O backend local está offline");
    expect(html).toContain("Tentar novamente");
    expect(html).toContain('role="alert"');
    expect(html).toContain('aria-live="assertive"');
  });
});
