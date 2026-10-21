import { afterEach, describe, expect, it, vi } from "vitest";
import { carregarEstadoDoCompanion, estadoIndisponivel } from "./companionApi";

/**
 * O cliente e' o ultimo elo do item 46: sem ele, a rota testada no servidor nao chega a
 * lado nenhum. *Estes testes existem para que o elo nao seja dado como feito sem prova.*
 */

function respostaJSON(corpo: unknown, status = 200): Response {
  return new Response(JSON.stringify(corpo), {
    status,
    headers: { "content-type": "application/json" },
  });
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("carregarEstadoDoCompanion", () => {
  it("pede a rota do Companion da missao certa", async () => {
    const fetchFalso = vi.fn().mockResolvedValue(
      respostaJSON({
        mission_id: "mis_1",
        estado: "a_trabalhar",
        razao: "passo em curso",
        scope: "missao do chamador",
      }),
    );
    vi.stubGlobal("fetch", fetchFalso);

    await carregarEstadoDoCompanion("mis_1");

    expect(fetchFalso).toHaveBeenCalledTimes(1);
    const url = String(fetchFalso.mock.calls[0][0]);
    expect(url.endsWith("/api/agent/v1/missions/mis_1/companion")).toBe(true);
  });

  it("transporta estado, razao e escopo como o servidor os deu", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        respostaJSON({
          mission_id: "mis_1",
          estado: "a_aprovar",
          razao: "ha' aprovacao pendente",
          scope: "missao do chamador",
        }),
      ),
    );

    const estado = await carregarEstadoDoCompanion("mis_1");

    expect(estado.estado).toBe("a_aprovar");
    expect(estado.razao).toBe("ha' aprovacao pendente");
    expect(estado.scope).toBe("missao do chamador");
  });

  it("escapa o id da missao no caminho", async () => {
    const fetchFalso = vi
      .fn()
      .mockResolvedValue(respostaJSON({ mission_id: "x", estado: "ausente" }));
    vi.stubGlobal("fetch", fetchFalso);

    await carregarEstadoDoCompanion("mis/../admin");

    const url = String(fetchFalso.mock.calls[0][0]);
    expect(url).not.toContain("/admin");
    expect(url).toContain("mis%2F..%2Fadmin");
  });

  it("NAO inventa estado quando a rota falha: o erro sobe", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(respostaJSON({ error: "sem permissao" }, 403)),
    );

    await expect(carregarEstadoDoCompanion("mis_alheia")).rejects.toThrow();
  });
});

describe("estadoIndisponivel", () => {
  it("declara a ausencia com a razao, em vez de um estado bonito", () => {
    const estado = estadoIndisponivel("404");

    expect(estado.estado).toBe("ausente");
    expect(estado.razao).toContain("404");
    expect(estado.scope).toContain("indisponivel");
  });
});
