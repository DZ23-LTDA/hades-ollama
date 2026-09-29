import { test, expect, type Page, type Route } from "@playwright/test";

// Jornada Nova tarefa -> Tarefas. O runtime Go esta fora da faixa de UI, entao o
// contrato /api/agent/v1/missions e atendido por um store em memoria no nivel de
// rede: a UI faz os requests reais e so mostra sucesso quando o POST responde 201
// e a missao volta na listagem.
type Mission = {
  id: string;
  version: number;
  objective: string;
  state: string;
  plan: unknown[];
  artifacts: unknown[];
  created_at: string;
  updated_at: string;
};

async function installMissionContract(
  page: Page,
  options: { failNextCreate?: boolean } = {},
) {
  const missions: Mission[] = [];
  const createBodies: unknown[] = [];
  let failNextCreate = options.failNextCreate ?? false;

  await page.route("**/api/agent/v1/missions", async (route: Route) => {
    const request = route.request();
    if (request.method() === "GET") {
      await route.fulfill({ json: { missions } });
      return;
    }
    if (request.method() === "POST") {
      const body = request.postDataJSON() as { objective?: string };
      createBodies.push(body);
      if (failNextCreate) {
        failNextCreate = false;
        await route.fulfill({
          status: 503,
          json: { error: "agent runtime unavailable" },
        });
        return;
      }
      if (!body.objective?.trim()) {
        await route.fulfill({
          status: 400,
          json: { error: "objective is required" },
        });
        return;
      }
      const now = new Date().toISOString();
      const mission: Mission = {
        id: `mission-${missions.length + 1}`,
        version: 1,
        objective: body.objective,
        state: "PLANNED",
        plan: [],
        artifacts: [],
        created_at: now,
        updated_at: now,
      };
      missions.unshift(mission);
      await route.fulfill({ status: 201, json: mission });
      return;
    }
    await route.fallback();
  });

  return { missions, createBodies };
}

test.describe("Jornada Nova tarefa -> Tarefas", () => {
  test("cria a tarefa, trata erro do runtime e confirma na listagem", async ({
    page,
  }) => {
    const contract = await installMissionContract(page, {
      failNextCreate: true,
    });

    await page.goto("/tasks");
    await expect(
      page.getByRole("heading", { name: "Tarefas", level: 2 }),
    ).toBeVisible();
    await expect(page.getByText("Nenhuma tarefa criada ainda.")).toBeVisible();

    await page.getByRole("button", { name: "Nova tarefa" }).first().click();
    await expect(page).toHaveURL(/\/tasks\/new$/);
    const objective = page.getByLabel("Objetivo da tarefa");
    await expect(objective).toBeVisible();

    // Validacao local: nenhum request com objetivo vazio.
    await page.getByRole("button", { name: "Criar tarefa" }).click();
    await expect(
      page.getByText("Descreva o objetivo da tarefa antes de criar."),
    ).toBeVisible();
    await expect(objective).toHaveAttribute("aria-invalid", "true");
    expect(contract.createBodies).toHaveLength(0);

    // Erro real do contrato: permanece na tela, preserva o texto e anuncia o erro.
    await objective.fill("Revisar o README e listar pendências");
    await page.getByRole("button", { name: "Criar tarefa" }).click();
    const alert = page.getByRole("alert");
    await expect(alert).toContainText("A tarefa não foi criada.");
    await expect(alert).toContainText("agent runtime unavailable");
    await expect(alert).toBeFocused();
    await expect(page).toHaveURL(/\/tasks\/new$/);
    await expect(objective).toHaveValue("Revisar o README e listar pendências");

    // Nova tentativa pelo teclado (Ctrl+Enter no campo).
    await objective.focus();
    await objective.press("Control+Enter");
    await expect(page).toHaveURL(/\/tasks\?created=mission-1$/);
    const banner = page
      .getByRole("status")
      .filter({ hasText: "Tarefa criada" });
    await expect(banner).toContainText(
      "Revisar o README e listar pendências · estado PLANNED",
    );
    await expect(banner).toBeFocused();
    await expect(
      page.getByRole("list", { name: "Tarefas" }).getByRole("listitem"),
    ).toHaveCount(1);

    expect(contract.createBodies).toEqual([
      {
        objective: "Revisar o README e listar pendências",
        capabilities: ["workspace:read"],
        auto_run: false,
      },
      {
        objective: "Revisar o README e listar pendências",
        capabilities: ["workspace:read"],
        auto_run: false,
      },
    ]);
  });

  test("mostra erro de carregamento com nova tentativa", async ({ page }) => {
    let calls = 0;
    await page.route("**/api/agent/v1/missions", async (route) => {
      calls += 1;
      if (calls === 1)
        await route.fulfill({
          status: 500,
          json: { error: "missions store unavailable" },
        });
      else await route.fulfill({ json: { missions: [] } });
    });

    await page.goto("/tasks");
    const panel = page
      .locator("section")
      .filter({ hasText: "Dados persistidos" });
    await expect(panel.getByRole("alert")).toContainText(
      "missions store unavailable",
    );
    await panel.getByRole("button", { name: "Tentar novamente" }).click();
    await expect(page.getByText("Nenhuma tarefa criada ainda.")).toBeVisible();
    expect(calls).toBe(2);
  });

  test("funciona em 390x844 e pelo teclado", async ({ page }) => {
    await installMissionContract(page);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/tasks/new");

    const objective = page.getByLabel("Objetivo da tarefa");
    await objective.focus();
    await page.keyboard.type("Tarefa criada no celular");
    await page.keyboard.press("Tab");
    await expect(page.getByRole("link", { name: "Cancelar" })).toBeFocused();
    await page.keyboard.press("Tab");
    await expect(
      page.getByRole("button", { name: "Criar tarefa" }),
    ).toBeFocused();
    await page.keyboard.press("Enter");

    await expect(
      page.getByRole("status").filter({ hasText: "Tarefa criada" }),
    ).toBeVisible();
    const overflow = await page.evaluate(
      () =>
        document.documentElement.scrollWidth -
        document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
  });
});
