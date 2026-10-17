import { describe, expect, it } from "vitest";
import { estadoParaMostrar, textoDoEstado } from "./ArtifactVersionsPanel";
import type { VersoesDoArtefacto } from "@/lib/artifactVersionsApi";

/**
 * As decisoes puras do painel de versoes (Estagio 47). *Testadas sem DOM, porque o `vitest` deste
 * repo nao tem jsdom - e a decisao importante nao e' visual: e' o que se mostra quando o servidor
 * recusou.*
 */

const vazio: VersoesDoArtefacto = {
  artefactoID: "art_1",
  versoes: [],
  scope: "missao do chamador",
};
const comDuas: VersoesDoArtefacto = {
  artefactoID: "art_1",
  versoes: [
    { id: "art_1", version: 2, sha256: "bbbb" },
    { id: "art_1", version: 1, sha256: "aaaa" },
  ],
  scope: "missao do chamador",
};

describe("estadoParaMostrar", () => {
  it("mostra o que o servidor devolveu", () => {
    const estado = estadoParaMostrar({ ok: true, dados: comDuas });

    expect(estado.tipo).toBe("pronta");
    if (estado.tipo === "pronta") {
      expect(estado.dados.versoes).toHaveLength(2);
    }
  });

  it("falha NAO vira lista vazia pronta: vira falha declarada", () => {
    const estado = estadoParaMostrar({
      ok: false,
      motivo: "artifact search requires a tenant",
    });

    // *Uma lista vazia diria «nao ha versoes»; a verdade e' «nao consegui saber».*
    expect(estado.tipo).toBe("falhou");
    if (estado.tipo === "falhou") {
      expect(estado.motivo).toContain("tenant");
    }
  });
});

describe("textoDoEstado", () => {
  it("diz que esta' a carregar", () => {
    expect(textoDoEstado({ tipo: "a_carregar" })).toContain("A carregar");
  });

  it("transporta o motivo REAL da falha para o ecra", () => {
    const texto = textoDoEstado({
      tipo: "falhou",
      motivo: "mission version conflict",
    });

    expect(texto).toContain("mission version conflict");
  });

  it("distingue «sem versoes» de «nao consegui ler»", () => {
    const semVersoes = textoDoEstado(
      estadoParaMostrar({ ok: true, dados: vazio }),
    );
    const falhou = textoDoEstado(
      estadoParaMostrar({ ok: false, motivo: "400" }),
    );

    expect(semVersoes).toContain("Sem versoes");
    expect(falhou).not.toContain("Sem versoes");
    expect(falhou).toContain("400");
  });

  it("mostra a contagem E o escopo (o historico e' por missao)", () => {
    const texto = textoDoEstado(
      estadoParaMostrar({ ok: true, dados: comDuas }),
    );

    expect(texto).toContain("2 versao");
    expect(texto).toContain("missao do chamador");
  });

  it("nunca devolve texto vazio em nenhum estado", () => {
    const estados = [
      { tipo: "a_carregar" } as const,
      { tipo: "falhou", motivo: "x" } as const,
      estadoParaMostrar({ ok: true, dados: vazio }),
      estadoParaMostrar({ ok: true, dados: comDuas }),
    ];
    for (const estado of estados) {
      expect(textoDoEstado(estado).trim().length).toBeGreaterThan(0);
    }
  });
});
