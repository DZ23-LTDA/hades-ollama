// O transporte deixa de perder o motivo (E-1380).
//
// O QUE ESTE TESTE DISCRIMINA
//
// `vi.mock` do transporte seria MAIS UM mock a provar o mock. Aqui o `fetch` global e' que e' mockado: a
// cadeia `fetch -> agentFetch -> traduzir -> erro tipado` corre INTEIRA.
//
//	*Um painel provado contra um mock e um backend provado contra httptest nao provam que um fala com o outro.*
//
// O que este ficheiro NAO prova: que o painel usa o status. Tentei-o e regrediu 2 testes do painel; a causa
// nao ficou isolada e o bloco foi revertido. Fica registado como aberto, nao como feito.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ErroDaApi } from "@/lib/agenticClient";

const fetchOriginal = globalThis.fetch;

function responder(status: number, corpo: unknown, texto?: string): void {
  globalThis.fetch = vi.fn(async () => {
    return new Response(JSON.stringify(corpo), {
      status,
      statusText: texto ?? "erro",
      headers: { "content-type": "application/json" },
    });
  }) as unknown as typeof fetch;
}

beforeEach(() => {
  globalThis.fetch = fetchOriginal;
});

afterEach(() => {
  globalThis.fetch = fetchOriginal;
  vi.restoreAllMocks();
});

describe("ErroDaApi carrega o que o servidor disse", () => {
  it("guarda status e corpo, e continua a ser um Error", () => {
    const e = new ErroDaApi("falhou", 503, { error: "agent_api_unavailable" });
    expect(e).toBeInstanceOf(Error);
    expect(e.status).toBe(503);
    expect(e.message).toBe("falhou");
    expect(e.campo("error")).toBe("agent_api_unavailable");
  });

  it("campo() nao inventa quando o corpo nao tem a chave", () => {
    expect(new ErroDaApi("x", 500, { outra: 1 }).campo("reason")).toBeUndefined();
  });

  it("campo() tolera corpo que nao e' objecto", () => {
    expect(new ErroDaApi("x", 500, "texto").campo("reason")).toBeUndefined();
  });
});
