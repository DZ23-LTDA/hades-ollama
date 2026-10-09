import { test, expect } from "@playwright/test";

// Smoke E2E do shell local-first (sem provider externo). Valida que o app
// carrega, mostra a home real e navega pelas rotas principais offline.
// Seletores por texto/role (robustos a mudanca de estrutura).
test.describe("Hades shell", () => {
  test("home renders the unified chat composer and sidebar", async ({ page }) => {
    await page.goto("/");

    // Home unificada: `/` redireciona para a tela de chat ("O que posso fazer
    // por voce?") que o botao "Nova tarefa" tambem abre. O lancador de missoes
    // avancado vive no Agentic Console, acessivel pelo atalho da home.
    await expect(
      page.getByRole("heading", {
        name: "O que posso fazer por você?",
        level: 1,
      }),
    ).toBeVisible();
    await expect(
      page.getByRole("textbox", { name: "Mensagem para o agente" }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Importar projeto" }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Modo missão (avançado)" }),
    ).toBeVisible();
    await expect(page.getByText("Ollama Classe A+", { exact: true })).toHaveCount(0);
    await expect(page.getByText("Ollama Full", { exact: true })).toHaveCount(0);

    // Indicador de modo local-first na barra lateral: o landmark de navegacao
    // nomeado ("Navegação principal") e a superficie estavel que declara o modo
    // local-first. Na home o landmark e o <nav> do ChatSidebar, que embute o
    // AppNavigation; o AppSidebar (mesmo rotulo) cobre as rotas de workspace.
    await expect(
      page.getByRole("navigation", { name: "Navegação principal" }),
    ).toBeVisible();

    // Itens da navegacao lateral principal, com os rotulos reais em pt-BR.
    for (const item of ["Tarefas", "Empresa"]) {
      await expect(page.getByText(item, { exact: false }).first()).toBeVisible();
    }
  });

  test("agentic mission console renders on mobile", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/agentic");

    // Smoke offline: o Console de Missões (/agentic) mostra o cabecalho, o
    // composer de missao e o botao "Criar missão" sem depender de uma missao
    // ativa (coberta por go test no runtime).
    await expect(
      page.getByRole("heading", { name: "Console de Missões" }),
    ).toBeVisible();
    await expect(
      page.getByPlaceholder(/Descreva o que você quer construir/i),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Criar missão" }),
    ).toBeVisible();
  });
  // Navegacao para sub-rotas (/tasks, /agentic) depende do backend e nao e
  // deterministica offline; a jornada profunda e coberta por go test no runtime.
  // O smoke offline foca no shell da home, que renderiza local-first de forma estavel.
});
