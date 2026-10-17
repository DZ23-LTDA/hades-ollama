import { afterEach, describe, expect, it, vi } from "vitest";
import { carregarEstadoDoCompanion, ESTADOS_CONHECIDOS } from "./companionApi";

/**
 * A VALIDACAO NA FRONTEIRA (dívida declarada no round 102).
 *
 * Nasceu de um defeito MEDIDO no teste do host: um estado fora do conjunto conhecido
 * atravessava a API, o componente procurava-o no seu mapa, recebia `undefined` e
 * rebentava - `Cannot read properties of undefined (reading 'titulo')` - derrubando o
 * painel inteiro da missao, nao apenas o mascote.
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

describe("carregarEstadoDoCompanion — validacao do estado", () => {
  it("estado DESCONHECIDO vira ausencia declarada, nunca um valor que o componente nao sabe desenhar", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        respostaJSON({
          mission_id: "mis_1",
          estado: "estado_que_nao_existe",
        }),
      ),
    );

    const estado = await carregarEstadoDoCompanion("mis_1");

    expect(estado.estado).toBe("ausente");
    // A razao tem de dizer QUAL foi o valor recebido - senao ninguem consegue diagnosticar.
    expect(estado.razao).toContain("estado_que_nao_existe");
  });

  it("estado ausente da resposta (undefined) tambem vira ausencia declarada", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(respostaJSON({ mission_id: "mis_1" })),
    );

    const estado = await carregarEstadoDoCompanion("mis_1");

    expect(estado.estado).toBe("ausente");
  });

  it("estado CONHECIDO passa intacto (guarda de regressao)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        respostaJSON({
          mission_id: "mis_1",
          estado: "a_trabalhar",
          razao: "passo em curso",
        }),
      ),
    );

    const estado = await carregarEstadoDoCompanion("mis_1");

    expect(estado.estado).toBe("a_trabalhar");
    expect(estado.razao).toBe("passo em curso");
  });

  it("a lista de estados validos tem exactamente os oito do servidor (sem duplicados)", () => {
    expect(ESTADOS_CONHECIDOS).toHaveLength(8);
    expect(new Set(ESTADOS_CONHECIDOS).size).toBe(8);
    expect(ESTADOS_CONHECIDOS).toContain("ausente");
    expect(ESTADOS_CONHECIDOS).toContain("cancelado");
  });
});
