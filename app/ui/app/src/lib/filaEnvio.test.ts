import { describe, expect, it } from "vitest";

import { enfileirar, tirarProximo } from "./filaEnvio";

// Defeito D: submeter durante o stream descartava a mensagem em silêncio
// (`if (... || isStreaming) return;` em ChatForm). A fila garante que nada se perde.
describe("fila de envio", () => {
  it("POSITIVO: enfileira por ordem", () => {
    let fila: string[] = [];
    fila = enfileirar(fila, "primeira");
    fila = enfileirar(fila, "segunda");
    expect(fila).toEqual(["primeira", "segunda"]);
  });

  it("NEGATIVO: texto vazio ou só espaços não entra na fila", () => {
    expect(enfileirar([], "")).toEqual([]);
    expect(enfileirar(["a"], "   ")).toEqual(["a"]);
  });

  it("POSITIVO: retira o primeiro e mantém o resto na ordem", () => {
    const { proximo, resto } = tirarProximo(["a", "b", "c"]);
    expect(proximo).toBe("a");
    expect(resto).toEqual(["b", "c"]);
  });

  it("NEGATIVO: fila vazia devolve null e não rebenta", () => {
    const { proximo, resto } = tirarProximo([]);
    expect(proximo).toBeNull();
    expect(resto).toEqual([]);
  });

  it("o texto é normalizado (trim) ao entrar", () => {
    expect(enfileirar([], "  com espaços  ")).toEqual(["com espaços"]);
  });

  it("não muta a fila recebida", () => {
    const original = ["a"];
    enfileirar(original, "b");
    tirarProximo(original);
    expect(original).toEqual(["a"]);
  });
});
