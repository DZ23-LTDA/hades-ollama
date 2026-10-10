import { test, expect } from "@playwright/test";

/**
 * E2E do editor visual de automações (P0-2): monta um fluxo de 3 nós
 * (gatilho -> ação -> ação), conecta as etapas na tela, publica e prova que o
 * grafo de passos persistido sai da UI exatamente no formato que o runtime
 * executa em ordem topológica (`internal/agent/schedule_flow.go`).
 *
 * O job de E2E sobe apenas `vite preview` (sem backend Go), então o tráfego de
 * `/api/agent/v1/schedules` é interceptado por rota: a prova aqui é do contrato
 * cliente -> HTTP na UI real. A execução real do grafo, com o mesmo modelo de
 * permissão do runtime, é provada pelos testes Go:
 * `internal/agent/schedule_flow_test.go` (ordem, bloqueio de dependentes) e
 * `server/agent_schedule_flow_test.go` (persiste e relê
 * gatilho -> ação -> condição passando pelo `decodeJSON` estrito).
 */

type ScheduleBody = {
  objective?: string;
  interval_seconds?: number;
  steps?: Array<{
    id: string;
    kind: string;
    depends_on?: string[];
    objective?: string;
  }>;
};

test.describe("Editor visual de automações", () => {
  test("publica um fluxo de 3 nós com o grafo de passos persistido", async ({
    page,
  }) => {
    const posted: ScheduleBody[] = [];

    await page.route("**/api/agent/v1/schedules**", async (route) => {
      const request = route.request();
      if (request.method() === "POST") {
        const body = request.postDataJSON() as ScheduleBody;
        posted.push(body);
        await route.fulfill({
          status: 201,
          contentType: "application/json",
          body: JSON.stringify({ id: "sch_e2e", ...body }),
        });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ schedules: [] }),
      });
    });

    await page.goto("/scheduled");
    await page.getByRole("button", { name: "Editor visual (beta)" }).click();
    await expect(page.getByTestId("flow-editor")).toBeVisible();

    // n1_trigger: gatilho por intervalo (os ids do editor são determinísticos).
    await page.getByRole("button", { name: "Intervalo", exact: true }).click();

    const inputs = page.getByTestId("flow-editor").locator("input");

    // n2_action: primeira ação de missão (inspetor: [0] rótulo, [1] objetivo).
    await page
      .getByRole("button", { name: "Rodar missão", exact: true })
      .click();
    await page.getByTestId("node-n2_action").click();
    await inputs.nth(0).fill("Coletar métricas");
    await inputs.nth(1).fill("Coletar métricas do repositório");

    // n3_action: segunda ação de missão, dependente da primeira.
    await page
      .getByRole("button", { name: "Rodar missão", exact: true })
      .click();
    await page.getByTestId("node-n3_action").click();
    await inputs.nth(0).fill("Publicar resumo");
    await inputs.nth(1).fill("Publicar resumo do build");

    // Conexões persistidas: gatilho -> coleta -> resumo.
    await page.getByLabel("Conectar saída de Intervalo").click();
    await page.getByTestId("node-n2_action").click();
    await page.getByLabel("Conectar saída de Coletar métricas").click();
    await page.getByTestId("node-n3_action").click();

    await expect(page.getByTestId("flow-valid")).toBeVisible();
    await page.getByRole("button", { name: "Publicar fluxo" }).click();

    // O servidor recebe o grafo de passos já na ordem de execução.
    await expect
      .poll(() => posted.length, {
        message: "publicar deveria enviar o agendamento para a API",
      })
      .toBe(1);

    const body = posted[0];
    expect(body.interval_seconds).toBe(3600);
    expect(body.steps).toEqual([
      { id: "n1_trigger", kind: "trigger.interval" },
      {
        id: "n2_action",
        kind: "action.mission",
        depends_on: ["n1_trigger"],
        objective: "Coletar métricas do repositório",
      },
      {
        id: "n3_action",
        kind: "action.mission",
        depends_on: ["n2_action"],
        objective: "Publicar resumo do build",
      },
    ]);
    // Fluxo com mais de uma ação vira um plano ordenado (o grafo de passos é a
    // execução real; o objetivo textual continua legível no histórico).
    expect(body.objective).toContain("Execute este fluxo de automação");
    expect(body.objective).toContain("Coletar métricas do repositório");
    expect(body.objective).toContain("Publicar resumo do build");

    // A UI confirma a publicação e volta para a lista de rotinas ativas.
    await expect(
      page.getByText("Fluxo publicado como automação. Veja em Rotinas Ativas."),
    ).toBeVisible();
  });
});
