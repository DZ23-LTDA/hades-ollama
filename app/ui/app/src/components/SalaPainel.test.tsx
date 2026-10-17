import { describe, expect, it } from "vitest";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";

import { SalaMembros, TEXTOS_DA_SALA } from "./SalaMembros";
import {
  SalaPainel,
  estadoCarregando,
  estadoDeErro,
  estadoDoResultado,
  estadoPronto,
  type EstadoDaSala,
} from "./SalaPainel";

/**
 * Complemento 48 / FASE 2 — o teste do painel: **o erro não chega à tela como ausência.**
 *
 * *Estes testes são o último elo da cadeia* (cliente → estado → vista): se o erro virasse `pronta` com zero
 * membros, a tela diria «ainda não há membros» e **todos os testes anteriores continuariam verdes**.
 */

function html(estado: EstadoDaSala): string {
  return renderToStaticMarkup(createElement(SalaPainel, { estado }));
}

const ana = { id: "ana", tipo: "humano" as const, nome: "Ana" };

describe("estadoDoResultado — a decisão escrita em um só lugar", () => {
  it("falha vira `erro`, NUNCA `pronta` vazia", () => {
    const estado = estadoDoResultado({
      erro: new Error("sem ligação ao servidor"),
    });
    expect(estado.tipo).toBe("erro");
    if (estado.tipo === "erro") {
      expect(estado.mensagem).toContain("sem ligação ao servidor");
    }
  });

  it("sucesso com zero membros é `pronta` (sala vazia é um facto, não uma falha)", () => {
    const estado = estadoDoResultado({ membros: [] });
    expect(estado).toEqual({ tipo: "pronta", membros: [] });
  });

  it("erro não-Error também tem mensagem (nunca vazia)", () => {
    const estado = estadoDoResultado({ erro: "string solta" });
    expect(estado.tipo).toBe("erro");
    if (estado.tipo === "erro") {
      expect(estado.mensagem.trim()).not.toBe("");
    }
  });

  it("estadoDeErro sem motivo não produz erro invisível", () => {
    expect(estadoDeErro("   ")).toEqual({
      tipo: "erro",
      mensagem: "motivo não informado",
    });
  });
});

describe("SalaPainel — o que aparece na tela em cada estado", () => {
  it("erro: mostra o motivo e NÃO mostra «ainda não há membros»", () => {
    const saida = html(estadoDeErro("sem ligação ao servidor"));
    expect(saida).toContain("sem ligação ao servidor");
    expect(saida).not.toContain(TEXTOS_DA_SALA.vazia);
  });

  it("carregando: declara que está a carregar, não que está vazio", () => {
    const saida = html(estadoCarregando());
    expect(saida).toContain(TEXTOS_DA_SALA.carregando);
    expect(saida).not.toContain(TEXTOS_DA_SALA.vazia);
  });

  it("pronta e vazia: a ausência é declarada (o único caso em que essa frase é verdadeira)", () => {
    expect(html(estadoPronto([]))).toContain(TEXTOS_DA_SALA.vazia);
  });

  it("pronta com membros: lista humano e agente, com o tipo em palavras", () => {
    const saida = html(
      estadoPronto([ana, { id: "ag-1", tipo: "agente", nome: "Pesquisador" }]),
    );
    expect(saida).toContain("Ana (ana)");
    expect(saida).toContain(TEXTOS_DA_SALA.etiquetaAgente);
  });

  it("o painel não inventa botões em nenhum estado", () => {
    for (const estado of [
      estadoCarregando(),
      estadoDeErro("x"),
      estadoPronto([]),
      estadoPronto([ana]),
    ]) {
      expect(html(estado)).not.toContain("<button");
    }
  });

  it("o painel usa a MESMA vista do componente (uma decisão, um lugar)", () => {
    // Prova fraca em texto, forte em intenção: a classe do componente aparece no painel.
    expect(html(estadoPronto([ana]))).toContain("sala-membros");
    expect(
      renderToStaticMarkup(createElement(SalaMembros, { membros: [ana] })),
    ).toContain("sala-membros");
  });
});
