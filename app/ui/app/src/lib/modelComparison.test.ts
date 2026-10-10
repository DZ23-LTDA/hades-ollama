import { describe, expect, it } from "vitest";

import {
  MAX_COMPARISON_MODELS,
  abortColumn,
  applyStreamEvent,
  comparisonCandidates,
  createColumn,
  elapsedMs,
  failColumn,
  formatMilliseconds,
  promptIsRunnable,
  selectionIsRunnable,
  summarize,
  toggleCandidate,
  type ComparisonColumn,
} from "./modelComparison";

const models = [
  {
    model: "qwen2.5:7b",
    digest: "sha256:aaa",
    available: true,
    capabilities: ["chat", "tools"],
  },
  {
    model: "llama3.2:3b",
    digest: "sha256:bbb",
    available: true,
    capabilities: ["chat"],
  },
  {
    model: "nomic-embed-text",
    digest: "sha256:ccc",
    available: true,
    capabilities: ["embedding"],
  },
  {
    model: "ausente:1b",
    digest: "sha256:ddd",
    available: false,
    capabilities: ["chat"],
  },
  { model: "qwen2.5:7b", digest: "sha256:aaa", available: true },
  { model: "", available: true },
  // Sugestao de download: sem digest, sem provider e sem nuvem.
  { model: "gpt-oss:120b", available: true, capabilities: ["chat"] },
  // Modelo servido por provider configurado (router multi-provider).
  {
    model: "anthropic/claude",
    provider: "anthropic",
    available: true,
    capabilities: ["chat"],
  },
  // Modelo Ollama Cloud.
  { model: "gpt-oss:120b-cloud", available: true, capabilities: ["chat"] },
];

describe("comparisonCandidates", () => {
  it("mantem apenas modelos executaveis de chat e sem duplicatas", () => {
    expect(comparisonCandidates(models).map((item) => item.id)).toEqual([
      "anthropic/claude",
      "gpt-oss:120b-cloud",
      "llama3.2:3b",
      "qwen2.5:7b",
    ]);
  });

  it("aceita lista ausente", () => {
    expect(comparisonCandidates(undefined)).toEqual([]);
  });

  it("aceita modelo instalado sem lista de capacidades", () => {
    expect(
      comparisonCandidates([
        { model: "modelo-legado", digest: "sha256:eee", available: true },
      ]).map((item) => item.id),
    ).toEqual(["modelo-legado"]);
  });

  it("exclui sugestao de download e modelo sem capacidades de chat", () => {
    expect(comparisonCandidates(models).map((item) => item.id)).not.toContain(
      "gpt-oss:120b",
    );
    expect(comparisonCandidates(models).map((item) => item.id)).not.toContain(
      "nomic-embed-text",
    );
  });
});

describe("toggleCandidate", () => {
  it("adiciona e remove preservando a ordem", () => {
    expect(toggleCandidate([], "a")).toEqual(["a"]);
    expect(toggleCandidate(["a", "b"], "a")).toEqual(["b"]);
  });

  it("recusa selecao acima do limite em vez de descartar escolha", () => {
    const full = ["a", "b", "c", "d"];
    expect(full).toHaveLength(MAX_COMPARISON_MODELS);
    expect(toggleCandidate(full, "e")).toEqual(full);
    expect(toggleCandidate(full, "a")).toEqual(["b", "c", "d"]);
  });

  it("exige pelo menos dois modelos", () => {
    expect(selectionIsRunnable([])).toBe(false);
    expect(selectionIsRunnable(["a"])).toBe(false);
    expect(selectionIsRunnable(["a", "b"])).toBe(true);
  });
});

describe("promptIsRunnable", () => {
  it("exige prompt util e dois modelos", () => {
    expect(promptIsRunnable("   ", ["a", "b"])).toBe(false);
    expect(promptIsRunnable("oi", ["a"])).toBe(false);
    expect(promptIsRunnable("oi", ["a", "b"])).toBe(true);
  });

  it("recusa prompt acima do teto da superficie", () => {
    expect(promptIsRunnable("x".repeat(8001), ["a", "b"])).toBe(false);
  });
});

describe("applyStreamEvent", () => {
  const base = (): ComparisonColumn =>
    createColumn({ id: "qwen2.5:7b", label: "qwen2.5:7b" }, 1000);

  it("acumula a resposta e mede o primeiro trecho", () => {
    const first = applyStreamEvent(
      base(),
      { eventName: "chat", content: "Olá" },
      1200,
    );
    const second = applyStreamEvent(
      first,
      { eventName: "chat", content: " mundo" },
      1400,
    );
    expect(second.status).toBe("streaming");
    expect(second.answer).toBe("Olá mundo");
    expect(second.characters).toBe(9);
    expect(second.firstOutputMs).toBe(200);
    expect(second.startedAt).toBe(1000);
  });

  it("registra o chat temporario criado pelo backend", () => {
    const column = applyStreamEvent(
      base(),
      { eventName: "chat_created", chatId: "tmp-1" },
      1100,
    );
    expect(column.chatId).toBe("tmp-1");
    expect(column.status).toBe("streaming");
  });

  it("mede o primeiro trecho tambem quando a saida comeca em thinking", () => {
    const column = applyStreamEvent(
      base(),
      { eventName: "thinking", thinking: "hmm" },
      1300,
    );
    expect(column.thinking).toBe("hmm");
    expect(column.firstOutputMs).toBe(300);
    expect(column.characters).toBe(0);
  });

  it("fecha a coluna em done e ignora eventos posteriores", () => {
    const done = applyStreamEvent(base(), { eventName: "done" }, 1800);
    expect(done.status).toBe("done");
    expect(done.finishedAt).toBe(1800);
    const late = applyStreamEvent(done, { eventName: "chat", content: "x" }, 1900);
    expect(late).toBe(done);
  });

  it("marca erro com a mensagem recebida", () => {
    const failed = applyStreamEvent(
      base(),
      { eventName: "error", error: "modelo indisponível" },
      1700,
    );
    expect(failed.status).toBe("error");
    expect(failed.error).toBe("modelo indisponível");
    expect(failed.finishedAt).toBe(1700);
  });

  it("erra com mensagem propria quando o evento nao traz detalhe", () => {
    const failed = applyStreamEvent(base(), { eventName: "error" }, 1600);
    expect(failed.status).toBe("error");
    expect(failed.error).toContain("Falha desconhecida");
  });

  it("nao muta a coluna de entrada", () => {
    const column = base();
    applyStreamEvent(column, { eventName: "chat", content: "a" }, 1500);
    expect(column.answer).toBe("");
    expect(column.characters).toBe(0);
  });
});

describe("abortColumn e failColumn", () => {
  it("preserva a resposta parcial ao interromper", () => {
    const streaming = applyStreamEvent(
      createColumn({ id: "a", label: "a" }, 0),
      { eventName: "chat", content: "parcial" },
      100,
    );
    const aborted = abortColumn(streaming, 200);
    expect(aborted.status).toBe("aborted");
    expect(aborted.answer).toBe("parcial");
    expect(aborted.finishedAt).toBe(200);
  });

  it("nao reabre nem sobrescreve coluna concluida", () => {
    const done = applyStreamEvent(
      createColumn({ id: "a", label: "a" }, 0),
      { eventName: "done" },
      50,
    );
    expect(abortColumn(done, 60)).toBe(done);
    expect(failColumn(done, "boom", 70)).toBe(done);
  });

  it("registra falha de transporte em coluna em andamento", () => {
    const failed = failColumn(
      createColumn({ id: "a", label: "a" }, 0),
      "rede caiu",
      90,
    );
    expect(failed.status).toBe("error");
    expect(failed.error).toBe("rede caiu");
  });
});

describe("summarize e formatMilliseconds", () => {
  it("aponta o modelo mais rapido entre os concluidos", () => {
    const fast = applyStreamEvent(
      createColumn({ id: "rapido", label: "rapido" }, 0),
      { eventName: "done" },
      800,
    );
    const slow = applyStreamEvent(
      createColumn({ id: "lento", label: "lento" }, 0),
      { eventName: "done" },
      2400,
    );
    const failed = failColumn(
      createColumn({ id: "quebrado", label: "quebrado" }, 0),
      "erro",
      300,
    );
    const summary = summarize([fast, slow, failed], 3000);
    expect(summary.answered).toBe(2);
    expect(summary.failed).toBe(1);
    expect(summary.fastestModelId).toBe("rapido");
    expect(summary.fastestMs).toBe(800);
    expect(elapsedMs(slow, 9999)).toBe(2400);
  });

  it("nao elege vencedor quando nada concluiu", () => {
    const summary = summarize(
      [createColumn({ id: "a", label: "a" }, 0)],
      5000,
    );
    expect(summary.fastestModelId).toBeUndefined();
    expect(summary.fastestMs).toBeUndefined();
    expect(summary.totalCharacters).toBe(0);
  });

  it("formata latencia sem inventar numero", () => {
    expect(formatMilliseconds(undefined)).toBe("—");
    expect(formatMilliseconds(Number.NaN)).toBe("—");
    expect(formatMilliseconds(-5)).toBe("—");
    expect(formatMilliseconds(820)).toBe("820 ms");
    expect(formatMilliseconds(1500)).toBe("1,5 s");
    expect(formatMilliseconds(24000)).toBe("24 s");
  });
});
