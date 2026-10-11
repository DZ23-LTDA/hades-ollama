import { describe, expect, it } from "vitest";

import {
  scheduleRunConcerns,
  scheduleRunSummary,
  scheduleRunVerdict,
  scheduleRunVerdictLabel,
  scheduleStepKindLabel,
  scheduleStepAttemptsLabel,
  scheduleDeadLetterLabel,
  scheduleDeadLetters,
  scheduleFlowVersionLabel,
  schedulePendingApprovalLabel,
  schedulePendingApprovals,
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

  it("só mostra tentativas quando houve repetição", () => {
    expect(scheduleStepAttemptsLabel({ id: "a", kind: "action.mission", status: "succeeded", attempts: 1 })).toBe("");
    expect(scheduleStepAttemptsLabel({ id: "a", kind: "action.mission", status: "succeeded" })).toBe("");
    expect(scheduleStepAttemptsLabel({ id: "a", kind: "action.mission", status: "succeeded", attempts: 3 })).toBe(
      "3 tentativas",
    );
    expect(scheduleStepAttemptsLabel({ id: "a", kind: "action.mission", status: "failed", attempts: 5 })).toBe(
      "5 tentativas",
    );
  });

  it("expõe a fila morta sem inventar entrada", () => {
    expect(scheduleDeadLetters(null)).toEqual([]);
    expect(scheduleDeadLetters({ finished_at: "2026-10-11T06:00:00Z", missions: 0 })).toEqual([]);
    const entry = {
      step_id: "acao",
      kind: "action.mission",
      reason: "projeto ausente",
      attempts: 3,
      failed_at: "2026-10-11T06:00:00Z",
    };
    expect(scheduleDeadLetters({ finished_at: "2026-10-11T06:00:00Z", missions: 0, dead_letters: [entry] })).toEqual([
      entry,
    ]);
    // O rótulo carrega o motivo REAL do servidor e as tentativas quando houve retry.
    expect(scheduleDeadLetterLabel(entry)).toBe("acao após 3 tentativas — projeto ausente");
    expect(scheduleDeadLetterLabel({ ...entry, attempts: 1 })).toBe("acao — projeto ausente");
  });

  it("mostra a versão do fluxo só quando o servidor informa", () => {
    expect(scheduleFlowVersionLabel(null)).toBe("");
    expect(scheduleFlowVersionLabel({ finished_at: "2026-10-11T07:00:00Z", missions: 0 })).toBe("");
    expect(scheduleFlowVersionLabel({ finished_at: "2026-10-11T07:00:00Z", missions: 0, flow_version: 3 })).toBe(
      "fluxo v3",
    );
  });

  it("nomeia os nós novos do vocabulário", () => {
    expect(scheduleStepKindLabel("action.transform")).toBe("transformação (variável)");
    expect(scheduleStepKindLabel("logic.loop")).toBe("laço limitado");
    expect(scheduleStepKindLabel("approval.request")).toBe("aprovação humana");
    expect(scheduleStepKindLabel("action.connector")).toBe("ação: conector");
    // Tipo desconhecido não vira texto vazio: o operador precisa ver o valor real.
    expect(scheduleStepKindLabel("kind.desconhecido")).toBe("kind.desconhecido");
  });

  it("trata aprovação pendente como fluxo parado, nunca como sucesso", () => {
    const run = {
      finished_at: "2026-10-11T12:00:00Z",
      missions: 0,
      steps: [
        { id: "gatilho", kind: "trigger.interval", status: "succeeded" as const },
        { id: "liberar", kind: "approval.request", status: "pending" as const },
        { id: "enviar", kind: "action.mission", status: "skipped" as const },
      ],
      pending_approvals: [{ step_id: "liberar", run_key: "sched@run", objective: "liberar o envio" }],
    };
    expect(scheduleRunVerdict(run)).toBe("partial");
    expect(scheduleStepStatusLabel("pending")).toBe("aguardando aprovação");
    // O painel mostra o que decidir e para qual execução.
    expect(schedulePendingApprovals(run)).toHaveLength(1);
    expect(schedulePendingApprovalLabel({ step_id: "liberar", objective: "liberar o envio" })).toBe(
      "liberar: liberar o envio",
    );
    // Sem pendência o painel não aparece.
    expect(schedulePendingApprovals(null)).toEqual([]);
    expect(schedulePendingApprovals({ finished_at: "2026-10-11T12:00:00Z", missions: 0 })).toEqual([]);
  });
});
