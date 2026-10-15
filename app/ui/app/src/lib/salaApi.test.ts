import { describe, expect, it } from "vitest";

import {
  SalaIndisponivel,
  buscarMembrosDaSala,
  urlDosMembros,
  type Fetcher,
} from "./salaApi";

/**
 * Complemento 48 / FASE 2 — o teste do cliente.
 *
 * *O que estes testes protegem é uma linha: **falha não é ausência.*** Se o cliente devolvesse `[]` num erro,
 * a tela mostraria «ainda não há membros» — e o teste do componente continuaria verde. *Este é o par que fecha
 * a porta.*
 */

function fetcherQueDevolve(corpo: unknown, ok = true, status = 200): Fetcher {
  return async () => ({ ok, status, json: async () => corpo });
}

function fetcherQueFalha(causa: unknown): Fetcher {
  return async () => {
    throw causa;
  };
}

const ana = { id: "ana", tipo: "humano" as const, nome: "Ana" };
const agente = { id: "agente-pesquisador", tipo: "agente" as const, nome: "Pesquisador" };

describe("buscarMembrosDaSala — falha nunca vira lista vazia", () => {
  it("rede em baixo LANÇA SalaIndisponivel (não devolve lista vazia)", async () => {
    await expect(
      buscarMembrosDaSala({ fetcher: fetcherQueFalha(new Error("offline")), projectID: "p", roomID: "r" }),
    ).rejects.toBeInstanceOf(SalaIndisponivel);
  });

  it("HTTP != 200 LANÇA, e diz o código", async () => {
    await expect(
      buscarMembrosDaSala({
        fetcher: fetcherQueDevolve({}, false, 500),
        projectID: "p",
        roomID: "r",
      }),
    ).rejects.toThrow(/500/);
  });

  it("corpo sem o campo `membros` LANÇA (resposta inválida não é sala vazia)", async () => {
    await expect(
      buscarMembrosDaSala({ fetcher: fetcherQueDevolve({ itens: [] }), projectID: "p", roomID: "r" }),
    ).rejects.toBeInstanceOf(SalaIndisponivel);
  });

  it("JSON inválido LANÇA", async () => {
    const fetcher: Fetcher = async () => ({
      ok: true,
      status: 200,
      json: async () => {
        throw new SyntaxError("bad json");
      },
    });
    await expect(buscarMembrosDaSala({ fetcher, projectID: "p", roomID: "r" })).rejects.toBeInstanceOf(
      SalaIndisponivel,
    );
  });

  it("membro com forma desconhecida LANÇA (não se inventa um tipo)", async () => {
    await expect(
      buscarMembrosDaSala({
        fetcher: fetcherQueDevolve({ membros: [{ id: "x", tipo: "robo" }] }),
        projectID: "p",
        roomID: "r",
      }),
    ).rejects.toBeInstanceOf(SalaIndisponivel);
  });
});

describe("buscarMembrosDaSala — o sucesso é o que o servidor disse", () => {
  it("lista vazia do SERVIDOR é lista vazia (e não erro)", async () => {
    const membros = await buscarMembrosDaSala({
      fetcher: fetcherQueDevolve({ membros: [] }),
      projectID: "p",
      roomID: "r",
    });
    expect(membros).toEqual([]);
  });

  it("traz humanos e agentes sem alterar o que veio", async () => {
    const membros = await buscarMembrosDaSala({
      fetcher: fetcherQueDevolve({ membros: [ana, agente] }),
      projectID: "p",
      roomID: "r",
    });
    expect(membros).toHaveLength(2);
    expect(membros[1].tipo).toBe("agente");
  });

  it("a URL é a do contrato, com os ids escapados", () => {
    expect(urlDosMembros("proj 1", "sala/2")).toBe(
      "/api/agent/v1/projects/proj%201/rooms/sala%2F2/members",
    );
  });
});
