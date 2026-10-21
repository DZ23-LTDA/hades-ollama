import { describe, expect, it } from "vitest";

import { descreverEvento, urlExecucao } from "./LiveBuildPage";
import type { AgentEvent } from "@/lib/agenticClient";

function evento(type: string, payload?: unknown): AgentEvent {
  return { id: "e-" + type, type, created_at: "2026-10-21T10:00:00Z", payload };
}

// O painel ao vivo existe para o utilizador ver o projeto a ser construido. Se o evento nao for
// traduzido, o painel mostra linhas cruas e nao conta historia nenhuma.
describe("descreverEvento", () => {
  it("traduz os tipos conhecidos para passos legiveis", () => {
    expect(descreverEvento(evento("plan.created")).titulo).toBe("Plano definido");
    expect(descreverEvento(evento("approval_requested")).titulo).toBe("Aprovação pedida");
    expect(descreverEvento(evento("artifact.created")).titulo).toBe("Artefacto gerado");
    expect(descreverEvento(evento("tool_call")).titulo).toBe("Ferramenta usada");
    expect(descreverEvento(evento("step_started")).titulo).toBe("Passo");
    expect(descreverEvento(evento("error")).titulo).toBe("Erro");
  });

  it("nao inventa rotulo para evento desconhecido: mostra o tipo", () => {
    expect(descreverEvento(evento("algo_que_ninguem_tratou")).titulo).toBe("algo_que_ninguem_tratou");
  });

  it("extrai o detalhe do payload quando existe", () => {
    const passo = descreverEvento(evento("tool_call", { name: "go test ./..." }));
    expect(passo.detalhe).toBe("go test ./...");
  });

  it("aguenta payload ausente ou estranho sem rebentar", () => {
    expect(() => descreverEvento({ id: "x", type: "step", created_at: "" })).not.toThrow();
    expect(descreverEvento(evento("step", null)).detalhe).toBeUndefined();
    expect(descreverEvento(evento("step", { name: 42 })).detalhe).toBeUndefined();
  });

  it("preserva o id e a hora do evento", () => {
    const passo = descreverEvento(evento("plan.created"));
    expect(passo.id).toBe("e-plan.created");
    expect(passo.quando).toBe("2026-10-21T10:00:00Z");
  });
});

// I1a: a URL de execucao tem de pedir o modo inline (executa) e escapar os ids.
describe("urlExecucao", () => {
  it("pede inline e aponta para a rota do artefacto", () => {
    const url = urlExecucao("m1", "a1");
    expect(url).toBe("/api/agent/v1/missions/m1/artifacts/a1?inline=1");
  });

  it("NEGATIVO: sem inline o servidor responde como download (nao e' o que queremos)", () => {
    expect(urlExecucao("m1", "a1")).toContain("inline=1");
  });

  it("escapa ids com caracteres especiais", () => {
    expect(urlExecucao("a/b", "c d")).toBe("/api/agent/v1/missions/a%2Fb/artifacts/c%20d?inline=1");
  });
});