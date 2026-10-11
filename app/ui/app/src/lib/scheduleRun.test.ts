import { describe, expect, it } from "vitest";

import {
  scheduleRunConcerns,
  scheduleRunSummary,
  scheduleRunVerdict,
  scheduleRunVerdictLabel,
  scheduleStepKindLabel,
  scheduleStepStatusLabel,
  type AgentScheduleRunReport,
} from "./scheduleRun";

function run(overrides: Partial<AgentScheduleRunReport> = {}): AgentScheduleRunReport {
  return {
    finished_at: "2026-10-10T20:00:00Z",
    missions: 1,
    steps: [
      { id: "gatilho", kind: "trigger.interval", status: "succeeded" },
      { id: "acao", kind: "action.mission", status: "succeeded" },
    ],
    ...overrides,
  };
}

describe("relatório de execução de automação", () => {
  it("não trata execução com passo ignorado como sucesso", () => {
    const parcial = run({
      missions: 0,
      steps: [
        { id: "gatilho", kind: "trigger.interval", status: "succeeded" },
        { id: "condicao", kind: "condition.if", status: "skipped", reason: "condição não satisfeita" },
      ],
    });
    expect(scheduleRunVerdict(parcial)).toBe("partial");
    expect(scheduleRunVerdictLabel("partial")).toContain("ignorado");
    expect(scheduleRunVerdict(run())).toBe("success");
    expect(
      scheduleRunVerdict(
        run({ steps: [{ id: "acao", kind: "action.mission", status: "failed", reason: "planner indisponível" }] }),
      ),
    ).toBe("failed");
    expect(scheduleRunVerdict(null)).toBe("unknown");
    expect(scheduleRunVerdictLabel("unknown")).toContain("Sem execução");
  });

  it("resume contando o que rodou, o que foi ignorado e as missões criadas", () => {
    expect(scheduleRunSummary(run())).toBe("2 executado(s) · 1 missão criada");
    const parcial = run({
      missions: 0,
      steps: [
        { id: "gatilho", kind: "trigger.interval", status: "succeeded" },
        { id: "condicao", kind: "condition.if", status: "skipped" },
        { id: "acao", kind: "action.mission", status: "skipped" },
      ],
    });
    expect(scheduleRunSummary(parcial)).toBe("1 executado(s) · 2 ignorado(s) · 0 missões criadas");
    expect(scheduleRunSummary(null)).toContain("Nenhuma execução registrada");
  });

  it("expõe apenas os passos que exigem atenção, com motivo", () => {
    const report = run({
      steps: [
        { id: "gatilho", kind: "trigger.interval", status: "succeeded" },
        { id: "condicao", kind: "condition.if", status: "skipped", reason: "condição não satisfeita" },
        { id: "acao", kind: "action.mission", status: "failed", reason: "planner indisponível" },
      ],
    });
    const concerns = scheduleRunConcerns(report);
    expect(concerns.map((step) => step.id)).toEqual(["condicao", "acao"]);
    expect(concerns[0].reason).toBe("condição não satisfeita");
    expect(scheduleRunConcerns(null)).toEqual([]);
  });

  it("traduz status e tipo de passo para pt-BR", () => {
    expect(scheduleStepStatusLabel("succeeded")).toBe("executado");
    expect(scheduleStepStatusLabel("skipped")).toBe("ignorado");
    expect(scheduleStepStatusLabel("failed")).toBe("falhou");
    expect(scheduleStepStatusLabel("outro")).toBe("desconhecido");
    expect(scheduleStepKindLabel("trigger.interval")).toBe("gatilho por tempo");
    expect(scheduleStepKindLabel("action.mission")).toBe("ação: criar missão");
    expect(scheduleStepKindLabel("condition.if")).toBe("condição");
    expect(scheduleStepKindLabel("")).toBe("passo");
  });

  it("tolera relatório sem passos (rotina legada)", () => {
    expect(scheduleRunVerdict({ finished_at: "2026-10-10T20:00:00Z", missions: 1 })).toBe("success");
    expect(scheduleRunSummary({ finished_at: "2026-10-10T20:00:00Z", missions: 1 })).toBe(
      "0 executado(s) · 1 missão criada",
    );
  });
});
