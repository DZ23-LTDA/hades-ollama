import { describe, expect, it } from "vitest";
import { estadoDaOperacao, objetivoDaOperacao } from "./MediaStudioPanel";
import { OPERACOES_DE_MIDIA } from "@/lib/mediaOperacoes";

/**
 * As decisoes puras do Multimodal Studio (Estagio 45). *Sem DOM: o `vitest` deste repo nao tem
 * jsdom, e o que importa nao e' visual - e' o que a UI DECLARA antes de lancar a missao.*
 */

const gerarImagem = OPERACOES_DE_MIDIA.find(
  (d) => d.operacao === "image.generate",
)!;
const tomLocal = OPERACOES_DE_MIDIA.find(
  (d) => d.operacao === "tone.generate",
)!;

describe("objetivoDaOperacao", () => {
  it("declara a operacao E os campos preenchidos", () => {
    const objetivo = objetivoDaOperacao(gerarImagem, {
      prompt: "um gato",
      model: "sdxl",
    });

    // *A operacao vai declarada no objetivo: nao se espera que um modelo a adivinhe.*
    expect(objetivo).toContain("image.generate");
    expect(objetivo).toContain("prompt=um gato");
    expect(objetivo).toContain("model=sdxl");
  });

  it("omite campos vazios em vez de os enviar em branco", () => {
    const objetivo = objetivoDaOperacao(gerarImagem, {
      prompt: "um gato",
      model: "   ",
    });

    expect(objetivo).toContain("prompt=um gato");
    expect(objetivo).not.toContain("model=");
  });

  it("funciona sem entradas nenhumas (tom local)", () => {
    expect(objetivoDaOperacao(tomLocal, {})).toBe("Midia: tone.generate");
  });
});

describe("estadoDaOperacao", () => {
  it("marca o que corre localmente sem provider", () => {
    expect(estadoDaOperacao(tomLocal, false)).toBe("Disponivel (local)");
  });

  it("marca o que passa a estar disponivel com provider", () => {
    expect(estadoDaOperacao(gerarImagem, true)).toBe(
      "Disponivel (provider configurado)",
    );
  });

  it("explica a indisponibilidade com a frase LITERAL do servidor", () => {
    const estado = estadoDaOperacao(gerarImagem, false);

    expect(estado).toContain("Indisponivel");
    // *A meta proibe fingir: a razao mostrada e' a do servidor, nao uma invencao da UI.*
    expect(estado).toContain("media provider is not configured");
  });

  it("nunca devolve estado vazio, para nenhuma operacao e em nenhum dos dois mundos", () => {
    for (const descritor of OPERACOES_DE_MIDIA) {
      expect(estadoDaOperacao(descritor, false).trim().length).toBeGreaterThan(
        0,
      );
      expect(estadoDaOperacao(descritor, true).trim().length).toBeGreaterThan(
        0,
      );
    }
  });
});
