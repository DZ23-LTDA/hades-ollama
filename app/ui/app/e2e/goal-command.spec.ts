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
});
