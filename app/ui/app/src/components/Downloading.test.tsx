import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import Downloading from "./Downloading";

describe("Downloading", () => {
  it("shows a 'preparing' state before the total size is known", () => {
    const html = renderToStaticMarkup(
      <Downloading completed={0} total={0} label="Baixando qwen2.5:0.5b" />,
    );
    expect(html).toContain("Baixando qwen2.5:0.5b");
    expect(html).toContain("Preparando download");
    expect(html).not.toContain("0 B / 0 B");
  });

  it("shows byte progress once the total size is known", () => {
    const html = renderToStaticMarkup(
      <Downloading completed={500_000_000} total={1_000_000_000} />,
    );
    expect(html).toContain("Downloading model");
    expect(html).toContain("50%");
    expect(html).not.toContain("Preparando download");
  });
});
