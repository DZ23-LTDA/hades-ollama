import { describe, expect, it } from "vitest";
import { triggerDownload } from "./download";

// O ambiente de testes roda SEM DOM (vitest `environment: "node"`). Um `document.createElement` solto
// ali viraria `ReferenceError` no meio de um fluxo, uma falha confusa que parece defeito do produto.
// Este teste trava a GUARDA: sem navegador, o erro é explícito e explicado.
describe("triggerDownload", () => {
  it("falha alto e explicado quando não há navegador (sem document/URL de objeto)", () => {
    expect(typeof document).toBe("undefined");
    expect(() => triggerDownload(new Blob(["conteudo"], { type: "text/csv" }), "arquivo.csv")).toThrow(
      /download requer um navegador/,
    );
  });
});
