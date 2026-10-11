import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, describe, expect, it } from "vitest";

import { ScheduleRunReportCard } from "./ScheduleRunReportCard";
import type { AgentScheduleRunReport } from "@/lib/scheduleRun";

const parcial: AgentScheduleRunReport = {
  finished_at: "2026-10-10T20:00:00Z",
  missions: 0,
  steps: [
    { id: "gatilho", kind: "trigger.interval", status: "succeeded" },
    { id: "condicao", kind: "condition.if", status: "skipped", reason: "condição não satisfeita pelos dependentes" },
  ],
};

async function render(run: AgentScheduleRunReport | null, extra: { lastFailureCode?: string; lastRunAt?: string } = {}) {
  let renderer: ReactTestRenderer | null = null;
  await act(async () => {
    renderer = create(<ScheduleRunReportCard run={run} {...extra} />);
  });
  return renderer as unknown as ReactTestRenderer;
}

describe("ScheduleRunReportCard", () => {
  let renderer: ReactTestRenderer | null = null;

  afterEach(() => {
    renderer?.unmount();
    renderer = null;
  });

  it("mostra execução parcial como parcial, com o motivo do passo ignorado", async () => {
    renderer = await render(parcial);
    const text = JSON.stringify(renderer.toJSON());
    expect(text).toContain("Parcial — houve passo ignorado");
    expect(text).toContain("ignorado");
    expect(text).toContain("condição não satisfeita pelos dependentes");
    expect(text).toContain("0 missões criadas");
    expect(text).not.toContain("Sucesso");
  });

  it("mostra sucesso quando todos os passos executaram", async () => {
    renderer = await render({
      finished_at: "2026-10-10T20:00:00Z",
      missions: 1,
      steps: [
        { id: "gatilho", kind: "trigger.interval", status: "succeeded" },
        { id: "acao", kind: "action.mission", status: "succeeded" },
      ],
    });
    const text = JSON.stringify(renderer.toJSON());
    expect(text).toContain("Sucesso");
    expect(text).toContain("1 missão criada");
    expect(text).not.toContain("exige atenção");
  });

  it("mostra falha com código e passo culpado", async () => {
    renderer = await render(
      {
        finished_at: "2026-10-10T20:00:00Z",
        missions: 0,
        steps: [{ id: "acao", kind: "action.mission", status: "failed", reason: "planner indisponível" }],
      },
      { lastFailureCode: "mission_creation_failed:acao" },
    );
    const text = JSON.stringify(renderer.toJSON());
    expect(text).toContain("Falhou");
    expect(text).toContain("mission_creation_failed:acao");
    expect(text).toContain("planner indisponível");
  });

  it("explica quando não há execução registrada", async () => {
    renderer = await render(null);
    const text = JSON.stringify(renderer.toJSON());
    expect(text).toContain("Sem execução registrada");
    expect(text).toContain("Nenhuma execução registrada para esta rotina.");
  });
});
