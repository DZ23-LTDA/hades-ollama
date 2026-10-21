import { describe, expect, it } from "vitest";

import { SalaIndisponivel } from "../lib/salaApi";
import { carregarEstado } from "./salaEstado";
import type { Fetcher } from "../lib/salaApi";

/**
 * Complemento 48 / FASE 2 — a prova da lógica do efeito **sem DOM**.
 *
 * *É este par que fecha a cadeia:* o cliente lança em falha, o painel sabe distinguir os estados, e **esta função
 * decide** — de forma testável — qual dos dois sai. Se ela devolvesse `pronta` num erro, tudo o resto continuaria
 * verde e a tela mentiria.
 */

const ana = { id: "ana", tipo: "humano" as const, nome: "Ana" };

function fetcherOk(corpo: unknown): Fetcher {
  return async () => ({ ok: true, status: 200, json: async () => corpo });
}

function fetcherErro(causa: unknown): Fetcher {
  return async () => {
    throw causa;
  };
}

describe("carregarEstado — nunca lança, nunca confunde falha com ausência", () => {
  it("sucesso com membros ⇒ `pronta`", async () => {
    const estado = await carregarEstado({
      fetcher: fetcherOk({ membros: [ana] }),
      projectID: "p",
      roomID: "r",
    });
    expect(estado.tipo).toBe("pronta");
    if (estado.tipo === "pronta") expect(estado.membros).toHaveLength(1);
  });

  it("sucesso com ZERO membros ⇒ `pronta` vazia (sala vazia é um facto)", async () => {
    const estado = await carregarEstado({
      fetcher: fetcherOk({ membros: [] }),
      projectID: "p",
      roomID: "r",
    });
    expect(estado).toEqual({ tipo: "pronta", membros: [] });
  });

  it("falha de rede ⇒ `erro` (e NÃO `pronta` vazia)", async () => {
    const estado = await carregarEstado({
      fetcher: fetcherErro(
        new SalaIndisponivel("a leitura da sala falhou (rede)"),
      ),
      projectID: "p",
      roomID: "r",
    });
    expect(estado.tipo).toBe("erro");
    if (estado.tipo === "erro") expect(estado.mensagem).toContain("rede");
  });

  it("resposta inválida ⇒ `erro` (o cliente lança e aqui vira estado)", async () => {
    const estado = await carregarEstado({
      fetcher: fetcherOk({ semMembros: true }),
      projectID: "p",
      roomID: "r",
    });
    expect(estado.tipo).toBe("erro");
  });

  it("a função é total: não lança nem com um erro não-Error", async () => {
    const estado = await carregarEstado({
      fetcher: fetcherErro("texto solto"),
      projectID: "p",
      roomID: "r",
    });
    expect(estado.tipo).toBe("erro");
    if (estado.tipo === "erro") expect(estado.mensagem.trim()).not.toBe("");
  });
});
