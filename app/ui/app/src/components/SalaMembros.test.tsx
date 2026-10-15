import { describe, expect, it } from "vitest";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";

import {
  SalaMembros,
  TEXTOS_DA_SALA,
  etiquetaDoTipo,
  rotuloDoMembro,
  type MembroDaSala,
} from "./SalaMembros";

/**
 * Complemento 48 / FASE 2 — o teste da superfície da sala.
 *
 * *Estes testes não medem o desenho: medem as três recusas que dão valor ao componente.*
 */

const ana: MembroDaSala = { id: "ana", tipo: "humano", nome: "Ana" };
const agente: MembroDaSala = { id: "agente-pesquisador", tipo: "agente", nome: "Pesquisador" };

function html(props: Partial<Parameters<typeof SalaMembros>[0]> = {}): string {
  return renderToStaticMarkup(createElement(SalaMembros, { membros: [], ...props }));
}

describe("SalaMembros — a falha NÃO se disfarça de ausência", () => {
  it("com erro, mostra o erro e NÃO mostra a mensagem de lista vazia", () => {
    const saida = html({ erro: "sem ligação ao servidor" });
    expect(saida).toContain("sem ligação ao servidor");
    // ***A REGRA CENTRAL:*** *«não consegui carregar» e «não há ninguém» são coisas diferentes.*
    expect(saida).not.toContain(TEXTOS_DA_SALA.vazia);
    expect(saida).toContain('role="alert"');
  });

  it("erro tem precedência sobre carregamento", () => {
    const saida = html({ erro: "falhou", carregando: true });
    expect(saida).toContain("falhou");
    expect(saida).not.toContain(TEXTOS_DA_SALA.carregando);
  });

  it("carregando NÃO é o mesmo que vazio", () => {
    const saida = html({ carregando: true, membros: [] });
    expect(saida).toContain(TEXTOS_DA_SALA.carregando);
    expect(saida).not.toContain(TEXTOS_DA_SALA.vazia);
  });

  it("sem membros e sem erro, a ausência é declarada", () => {
    expect(html({ membros: [] })).toContain(TEXTOS_DA_SALA.vazia);
  });
});

describe("SalaMembros — um agente não passa por pessoa", () => {
  it("lista humanos E agentes, com o tipo EM PALAVRAS", () => {
    const saida = html({ membros: [ana, agente] });
    expect(saida).toContain("Ana (ana)");
    expect(saida).toContain("Pesquisador (agente-pesquisador)");
    // O tipo vai no TEXTO: *um leitor de tela não recebe a cor.*
    expect(saida).toContain(TEXTOS_DA_SALA.etiquetaHumano);
    expect(saida).toContain(TEXTOS_DA_SALA.etiquetaAgente);
    expect(saida).toContain("sala-membro-agente");
  });

  it("não inventa botões (não há acção no backend, não há botão morto na tela)", () => {
    const saida = html({ membros: [ana, agente] });
    expect(saida).not.toContain("<button");
    expect(saida).not.toContain("<a ");
  });
});

describe("SalaMembros — os rótulos são o que dizem ser", () => {
  it("sem nome, o rótulo é o id — nunca vazio", () => {
    expect(rotuloDoMembro({ id: "x1", tipo: "agente" })).toBe("x1");
    expect(rotuloDoMembro({ id: "x1", tipo: "agente", nome: "   " })).toBe("x1");
  });

  it("etiquetaDoTipo distingue os dois tipos", () => {
    expect(etiquetaDoTipo("humano")).toBe(TEXTOS_DA_SALA.etiquetaHumano);
    expect(etiquetaDoTipo("agente")).toBe(TEXTOS_DA_SALA.etiquetaAgente);
  });
});
