import { afterEach, describe, expect, it, vi } from "vitest";
import {
  carregarVersoesDoArtefacto,
  restaurarVersaoDoArtefacto,
  versoesIndisponiveis,
} from "./artifactVersionsApi";

/**
 * O cliente do versionamento (Estagio 47 / item 47). *A superficie HTTP esta' testada no servidor
 * (404/409/sucesso/invariante de tenant); aqui prova-se que o cliente NAO inventa nada em cima
 * dela: le^ o que veio, descarta o que nao sabe desenhar e declara a ausencia.*
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

describe("carregarVersoesDoArtefacto", () => {
  it("pede a rota de versoes do artefacto certo", async () => {
    const fetchFalso = vi
      .fn()
      .mockResolvedValue(
        respostaJSON({
          mission_id: "mis_1",
          artifact_id: "art_1",
          versions: [],
          scope: "missao do chamador",
        }),
      );
    vi.stubGlobal("fetch", fetchFalso);

    await carregarVersoesDoArtefacto("mis_1", "art_1");

    const url = String(fetchFalso.mock.calls[0][0]);
    expect(
      url.endsWith("/api/agent/v1/missions/mis_1/artifacts/art_1/versions"),
    ).toBe(true);
  });

  it("descarta entradas que nao sabe desenhar em vez de as mostrar", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        respostaJSON({
          mission_id: "mis_1",
          artifact_id: "art_1",
          versions: [
            { id: "art_1", version: 2, sha256: "sha2" },
            { id: "art_1" }, // sem version: nao e' uma versao desenhavel
            null,
            "lixo",
            { id: "art_1", version: 1, sha256: "sha1" },
          ],
          scope: "missao do chamador",
        }),
      ),
    );

    const resultado = await carregarVersoesDoArtefacto("mis_1", "art_1");

    expect(resultado.versoes).toHaveLength(2);
    // A versao e' o NUMERO do servidor, nao a posicao na lista.
    expect(resultado.versoes.map((v) => v.version)).toEqual([2, 1]);
  });

  it("transporta o escopo declarado pelo servidor", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          respostaJSON({
            mission_id: "mis_1",
            artifact_id: "art_1",
            versions: [],
            scope: "missao do chamador",
          }),
        ),
    );

    const resultado = await carregarVersoesDoArtefacto("mis_1", "art_1");

    expect(resultado.scope).toBe("missao do chamador");
  });

  it("se o servidor NAO declarar o escopo, di-lo em vez de assumir global", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(respostaJSON({ mission_id: "mis_1", versions: [] })),
    );

    const resultado = await carregarVersoesDoArtefacto("mis_1", "art_1");

    expect(resultado.scope).toContain("indisponivel");
  });

  it("NAO inventa lista quando a rota falha: o erro sobe (400 sem tenant / 404 de outra missao)", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          respostaJSON({ error: "artifact search requires a tenant" }, 400),
        ),
    );

    await expect(
      carregarVersoesDoArtefacto("mis_1", "art_1"),
    ).rejects.toThrow();
  });
});

describe("restaurarVersaoDoArtefacto", () => {
  it("envia POST com a versao e o expected_version (o CAS do artefacto)", async () => {
    const fetchFalso = vi
      .fn()
      .mockResolvedValue(
        respostaJSON({
          mission_id: "mis_1",
          restored: { id: "art_1", version: 3 },
          scope: "missao do chamador",
        }),
      );
    vi.stubGlobal("fetch", fetchFalso);

    const restaurado = await restaurarVersaoDoArtefacto("mis_1", "art_1", 1, 2);

    const url = String(fetchFalso.mock.calls[0][0]);
    expect(url.endsWith("/artifacts/art_1/restore")).toBe(true);
    const init = fetchFalso.mock.calls[0][1] as RequestInit;
    expect(init.method).toBe("POST");
    const corpo = JSON.parse(String(init.body));
    expect(corpo.version).toBe(1);
    // *Sem o expected_version o servidor recusa com 409 - e' o que impede restaurar sobre uma
    // versao corrente que o cliente nao viu.*
    expect(corpo.expected_version).toBe(2);
    expect(restaurado?.version).toBe(3);
  });

  it("devolve null quando o servidor nao devolve um artefacto restaurado desenhavel", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(respostaJSON({ mission_id: "mis_1" })),
    );

    const restaurado = await restaurarVersaoDoArtefacto("mis_1", "art_1", 1, 2);

    expect(restaurado).toBeNull();
  });

  it("propaga o 409 de conflito em vez de fingir sucesso", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          respostaJSON({ error: "mission version conflict" }, 409),
        ),
    );

    await expect(
      restaurarVersaoDoArtefacto("mis_1", "art_1", 1, 99),
    ).rejects.toThrow();
  });
});

describe("versoesIndisponiveis", () => {
  it("declara a ausencia com o motivo, sem lista e sem promessa", () => {
    const indisponivel = versoesIndisponiveis("500");

    expect(indisponivel.versoes).toEqual([]);
    expect(indisponivel.scope).toContain("500");
    expect(indisponivel.scope).toContain("indisponivel");
  });
});
