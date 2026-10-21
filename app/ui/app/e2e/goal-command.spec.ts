import { test, expect } from "@playwright/test";

// §FASE-3 item 8: *criar missao via `/goal` no composer*.
//
// Porque este spec existe: todas as outras provas do `/goal` sao de PARSER
// (`slashCommands.test.ts`, 13 testes) ou de SHELL (`shell.spec.ts`, que so'
// verifica que o composer existe). Nenhuma delas digita `/goal` num navegador.
// *Um teste que nao digita o comando nao prova o comando.*
test.describe("/goal no composer", () => {
  test("digitar /goal leva ao Mission Control com o objetivo preservado", async ({ page }) => {
    await page.goto("/");

    const composer = page.getByRole("textbox", { name: "Mensagem para o agente" });
    await expect(composer).toBeVisible();

    const objetivo = "Gerar o relatorio de abertura do workspace";
    await composer.fill(`/goal ${objetivo}`);

    // O menu de comandos deve reconhecer `/goal` como comando, nao como texto.
    await expect(page.getByText("/goal", { exact: false }).first()).toBeVisible();

    await composer.press("Enter");

    // O caminho e': parseSlashCommand -> /agentic?objective&slash&autorun
    await page.waitForURL(/\/agentic/, { timeout: 20_000 });

    const url = new URL(page.url());
    expect(url.pathname).toBe("/agentic");
    expect(url.searchParams.get("objective")).toBe(objetivo);
    expect(url.searchParams.get("slash")).toBe("goal");
    expect(url.searchParams.get("autorun")).toBe("true");
  });

  test("/goal sem objetivo NAO navega nem envia o comando cru", async ({ page }) => {
    await page.goto("/");

    const composer = page.getByRole("textbox", { name: "Mensagem para o agente" });
    await expect(composer).toBeVisible();

    await composer.fill("/goal");
    await composer.press("Enter");

    // A URL tem de continuar na home (sem `/agentic`) e o composer tem de
    // continuar la' -- o objetivo vazio pede complemento em vez de agir.
    await expect(page).not.toHaveURL(/\/agentic/);
    await expect(composer).toBeVisible();
  });

  test("digitar /goal CRIA a missao no backend (teste discriminante)", async ({
    page,
    request,
  }) => {
    // Este teste existe porque os anteriores provavam a NAVEGACAO e paravam ai'.
    // O teste anterior termina logo apos `waitForURL` -- antes de o `useEffect`
    // do AgenticConsole disparar o POST. Aqui esperamos pela criacao e lemos a API.
    //
    // Requer backend real: `HADES_E2E_BACKEND_URL` (o `webServer` do fixture nao
    // tem `/api/agent/v1`). Sem ele, fica SKIPPED com razao -- nunca PASS falso.
    test.skip(
      !process.env.HADES_E2E_BACKEND_URL,
      "precisa de backend real (HADES_E2E_BACKEND_URL) com agent runtime",
    );

    await page.goto("/");
    const composer = page.getByRole("textbox", { name: "Mensagem para o agente" });
    await expect(composer).toBeVisible();

    const objetivo = `missao-verificavel-${Date.now()}`;
    await composer.fill(`/goal ${objetivo}`);
    await composer.press("Enter");
    await page.waitForURL(/\/agentic/, { timeout: 20_000 });

    // O `useEffect` do AgenticConsole cria a missao de forma assincrona.
    // Esperamos ate' 25 s por ela aparecer na API.
    await expect
      .poll(
        async () => {
          const res = await request.get("/api/agent/v1/missions");
          if (!res.ok()) return -1;
          const body = (await res.json()) as { missions?: Array<{ objective?: string }> };
          return (body.missions ?? []).filter((m) => m.objective === objetivo).length;
        },
        { timeout: 25_000, intervals: [500, 1000, 2000] },
      )
      .toBeGreaterThan(0);
  });
});
