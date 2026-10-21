import { afterEach, describe, expect, it, vi } from "vitest";
import { criarMissaoDeMidia, executarMissaoDeMidia } from "./mediaStudioApi";

/**
 * A chamada do Multimodal Studio (Estagio 45). *Cada rota aqui foi lida no servidor antes de ser
 * escrita: `POST /missions` (853·3139), `POST /missions/:id/run` (875),
 * `POST /missions/:id/approvals/:aid` (878·1896).*
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

describe("criarMissaoDeMidia", () => {
  it("envia POST /missions com a capacidade de midia, o objetivo e o workspace", async () => {
    const fetchFalso = vi.fn().mockResolvedValue(respostaJSON({ id: "mis_1" }));
    vi.stubGlobal("fetch", fetchFalso);

    const missao = await criarMissaoDeMidia(
      "gerar uma imagem de teste",
      "/tmp/ws",
    );

    const url = String(fetchFalso.mock.calls[0][0]);
    expect(url.endsWith("/api/agent/v1/missions")).toBe(true);
    const init = fetchFalso.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("POST");
    const corpo = JSON.parse(String(init.body));
    expect(corpo.objective).toBe("gerar uma imagem de teste");
    expect(corpo.workspace).toBe("/tmp/ws");
    // ***Sem `media:execute` o runtime nao registra `media.process` (runtime.go:267).***
    expect(corpo.capabilities).toEqual(["media:execute"]);
    expect(missao.id).toBe("mis_1");
  });

  it("aceita o identificador em `mission_id` quando e' o que vem", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(respostaJSON({ mission_id: "mis_2" })),
    );

    const missao = await criarMissaoDeMidia("objectivo", "/tmp/ws");

    expect(missao.id).toBe("mis_2");
  });

  it("recusa identificar a missao quando o servidor nao devolve nenhum id", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(respostaJSON({ estado: "a_planear" })),
    );

    // *Nao se inventa o identificador: sem ele nao ha como executar nem aprovar.*
    await expect(criarMissaoDeMidia("objectivo", "/tmp/ws")).rejects.toThrow(
      /identificador/,
    );
  });

  it("nao chama o servidor com objetivo ou workspace vazios", async () => {
    const fetchFalso = vi.fn();
    vi.stubGlobal("fetch", fetchFalso);

    await expect(criarMissaoDeMidia("   ", "/tmp/ws")).rejects.toThrow(
      /objetivo/,
    );
    await expect(criarMissaoDeMidia("objectivo", "  ")).rejects.toThrow(
      /workspace/,
    );
    // *O servidor recusaria com «pinned media workspace is required»: dizemo-lo mais cedo.*
    expect(fetchFalso).not.toHaveBeenCalled();
  });

  it("so^ envia organization_id quando existe", async () => {
    // *Um `Response` so^ se le^ UMA vez: o mock tem de criar um novo por chamada* - reutilizar o
    // mesmo objecto fazia a 2.ª leitura vir vazia (foi o que reprovou este teste na 1.ª tentativa).
    const fetchFalso = vi
      .fn()
      .mockImplementation(() => Promise.resolve(respostaJSON({ id: "mis_3" })));
    vi.stubGlobal("fetch", fetchFalso);

    await criarMissaoDeMidia("objectivo", "/tmp/ws");
    expect(
      JSON.parse(String(fetchFalso.mock.calls[0][1].body)).organization_id,
    ).toBeUndefined();

    await criarMissaoDeMidia("objectivo", "/tmp/ws", "org-a");
    expect(
      JSON.parse(String(fetchFalso.mock.calls[1][1].body)).organization_id,
    ).toBe("org-a");
  });

  it("propaga a recusa do servidor em vez de fingir missao criada", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          respostaJSON({ error: "media:execute nao autorizado" }, 403),
        ),
    );

    await expect(criarMissaoDeMidia("objectivo", "/tmp/ws")).rejects.toThrow();
  });
});

describe("executarMissaoDeMidia", () => {
  it("envia POST para /run da missao certa", async () => {
    const fetchFalso = vi.fn().mockResolvedValue(respostaJSON({ ok: true }));
    vi.stubGlobal("fetch", fetchFalso);

    await executarMissaoDeMidia("mis_1");

    const url = String(fetchFalso.mock.calls[0][0]);
    expect(url.endsWith("/api/agent/v1/missions/mis_1/run")).toBe(true);
    expect((fetchFalso.mock.calls[0][1] as RequestInit).method).toBe("POST");
  });
});

describe("aprovacao de midia", () => {
  it("NAO existe cliente de aprovacao no estudio (um so caminho: a superficie de aprovacao)", async () => {
    // *Medido:* o painel cria e lanca a missao; aprovar pertence a' superficie que ja' existe
    // (`AgenticSplitShell`/`AgenticConsole`, com `MissionApproval.nonce`). Um cliente paralelo
    // sem chamador foi removido - este teste impede que ele volte por engano.
    const modulo = await import("./mediaStudioApi");
    expect("decidirAprovacaoDeMidia" in modulo).toBe(false);
  });
});
