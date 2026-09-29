import { test, expect, type Page, type Route } from "@playwright/test";
import { mkdirSync } from "node:fs";
import { join } from "node:path";

// Jornada da tela "Agendado" (scheduled). O runtime Go esta fora da faixa de UI,
// entao os contratos /api/agent/v1/schedules e /api/agent/v1/projects sao
// atendidos por um store em memoria no nivel de rede. A UI faz os requests reais
// e so mostra sucesso quando o POST responde 201 e o schedule volta na listagem.

// docs/evidencias fica na raiz do projeto; o Playwright roda a partir de app/ui/app.
const EVIDENCE_DIR = join(process.cwd(), "..", "..", "..", "docs", "evidencias");

async function captureEvidence(page: Page, name: string) {
  mkdirSync(EVIDENCE_DIR, { recursive: true });
  await page.screenshot({ path: join(EVIDENCE_DIR, `${name}.png`), fullPage: true });
}

type Schedule = {
  id: string;
  objective: string;
  interval_seconds: number;
  enabled: boolean;
  next_run_at: string;
  project_id?: string;
  organization_id?: string;
};

async function installScheduleContract(
  page: Page,
  options: { failFirstList?: boolean; seed?: Schedule[] } = {},
) {
  const schedules: Schedule[] = [...(options.seed ?? [])];
  const createBodies: Array<Record<string, unknown>> = [];
  let listCalls = 0;

  // Projetos alimentam o <select> do formulario de agendamento.
  await page.route("**/api/agent/v1/projects", async (route: Route) => {
    if (route.request().method() === "GET") {
      await route.fulfill({
        json: { projects: [{ id: "proj-1", name: "Projeto Alpha" }] },
      });
      return;
    }
    await route.fallback();
  });

  await page.route("**/api/agent/v1/schedules", async (route: Route) => {
    const request = route.request();
    if (request.method() === "GET") {
      listCalls += 1;
      if (options.failFirstList && listCalls === 1) {
        await route.fulfill({
          status: 503,
          json: { error: "schedules store unavailable" },
        });
        return;
      }
      await route.fulfill({ json: { schedules } });
      return;
    }
    if (request.method() === "POST") {
      const body = request.postDataJSON() as Record<string, unknown>;
      createBodies.push(body);
      const nextRun = new Date(Date.now() + 3_600_000).toISOString();
      const schedule: Schedule = {
        id: `sched-${schedules.length + 1}`,
        objective: String(body.objective ?? ""),
        interval_seconds: Number(body.interval_seconds ?? 0),
        enabled: true,
        next_run_at: nextRun,
        project_id: (body.project_id as string | undefined) ?? undefined,
        organization_id: "org-local",
      };
      schedules.unshift(schedule);
      await route.fulfill({ status: 201, json: schedule });
      return;
    }
    await route.fallback();
  });

  return { schedules, createBodies, listCalls: () => listCalls };
}

test.describe("Tela Agendado (scheduled)", () => {
  test("carrega vazio, cria o agendamento e confirma na listagem", async ({
    page,
  }) => {
    const contract = await installScheduleContract(page);

    await page.goto("/scheduled");
    await expect(
      page.getByRole("heading", { name: "Agendado", level: 2 }),
    ).toBeVisible();
    await expect(
      page.getByText("Nenhuma automação está agendada", { exact: false }),
    ).toBeVisible();
    await captureEvidence(page, "agendado-vazio-desktop");

    // Cria um agendamento real pelo contrato de rede.
    await page
      .getByLabel("Objetivo do novo schedule")
      .fill("Rodar auditoria noturna");
    await page.getByLabel("Intervalo em segundos").fill("3600");
    await page
      .getByLabel("Projeto do schedule")
      .selectOption({ label: "Projeto Alpha" });
    await page.getByRole("button", { name: "Agendar", exact: true }).click();

    // Sucesso confirmado pela listagem persistida e pelo aviso ao vivo.
    await expect(page.getByText("Rodar auditoria noturna")).toBeVisible();
    await expect(page.getByText("a cada 3600s", { exact: false })).toBeVisible();
    await expect(
      page.getByRole("status").filter({ hasText: "Schedule criado" }),
    ).toBeVisible();
    expect(contract.createBodies).toEqual([
      {
        objective: "Rodar auditoria noturna",
        interval_seconds: 3600,
        project_id: "proj-1",
        enabled: true,
      },
    ]);
    await captureEvidence(page, "agendado-sucesso-desktop");
  });

  test("mostra erro de carregamento com nova tentativa", async ({ page }) => {
    await installScheduleContract(page, { failFirstList: true });

    await page.goto("/scheduled");
    const alert = page.getByRole("alert");
    await expect(alert).toContainText("schedules store unavailable");
    await captureEvidence(page, "agendado-erro-desktop");

    await page.getByRole("button", { name: "Tentar novamente" }).click();
    await expect(
      page.getByText("Nenhuma automação está agendada", { exact: false }),
    ).toBeVisible();
  });

  test("funciona em 390x844 sem overflow horizontal", async ({ page }) => {
    await installScheduleContract(page, {
      seed: [
        {
          id: "sched-seed",
          objective: "Sincronização diária",
          interval_seconds: 86_400,
          enabled: true,
          next_run_at: new Date().toISOString(),
        },
      ],
    });

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/scheduled");
    await expect(page.getByText("Sincronização diária")).toBeVisible();

    const overflow = await page.evaluate(
      () =>
        document.documentElement.scrollWidth -
        document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
    await captureEvidence(page, "agendado-sucesso-mobile");
  });
});
