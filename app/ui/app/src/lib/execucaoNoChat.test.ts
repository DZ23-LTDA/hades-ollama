import { describe, expect, it } from "vitest";
import {
  citaComandoInventado,
  pedidoDeExecucao,
  respostaSemFerramentas,
} from "./execucaoNoChat";

// FATIA 4 / T4. O caso real que originou isto, palavra por palavra do dono:
const PEDIDO_REAL =
  "vamos criar uma estrutura básica de teste: mkdir teste-hades, com index.html, test.js e README.md";

describe("pedidoDeExecucao", () => {
  it("reconhece o pedido real do dono", () => {
    expect(pedidoDeExecucao(PEDIDO_REAL)).toBe(true);
  });

  it("reconhece variantes de execucao", () => {
    for (const t of ["cria uma pasta nova", "gera o projeto", "executa os testes", "apaga o ficheiro"]) {
      expect(pedidoDeExecucao(t)).toBe(true);
    }
  });

  it("NEGATIVO: conversa pura nao e' pedido de execucao", () => {
    for (const t of ["", "   ", "ola", "obrigado", "o que e' um repositorio git?"]) {
      expect(pedidoDeExecucao(t)).toBe(false);
    }
  });
});

describe("respostaSemFerramentas", () => {
  it("diz a verdade: esta superficie nao executa", () => {
    const r = respostaSemFerramentas().toLowerCase();
    expect(r).toContain("não consigo criar");
    expect(r).toContain("nova tarefa");
  });

  it("NEGATIVO: nao cita NENHUM dos comandos inventados", () => {
    expect(citaComandoInventado(respostaSemFerramentas())).toBe(false);
  });

  it("NEGATIVO: nao ensina a abrir terminal nem a correr mkdir", () => {
    const r = respostaSemFerramentas().toLowerCase();
    expect(r).not.toContain("mkdir");
    expect(r).not.toContain("abra o terminal");
    expect(r).not.toContain("hades create");
  });
});

describe("Comandos inventados", () => {
  it("os quatro que o modelo inventou sao todos detectados", () => {
    for (const c of ["create-dir", "create-file", "list-files", "open-browser"]) {
      expect(citaComandoInventado("hades " + c + " --path x")).toBe(true);
    }
  });
});
