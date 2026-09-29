import { test, expect } from "@playwright/test";

// Smoke E2E do shell local-first (sem provider externo). Valida que o app
// carrega, mostra a home real e navega pelas rotas principais offline.
// Seletores por texto (robustos a mudanca de role/estrutura).
test.describe("Ollama Full shell", () => {
  test("home renders local-first composer and sidebar", async ({ page }) => {
    await page.goto("/");

    // Home real (dashboard local-first): saudação, composer de tarefa e cards.
    await expect(
      page.getByRole("heading", { name: "Ollama Full", level: 1 }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Missão agentic" }),
    ).toBeVisible();
    await expect(page.getByText("Ollama Classe A+", { exact: true })).toHaveCount(0);

    // Indicador de modo local-first (approvals/secrets protegidos).
    await expect(page.getByText(/Modo local-first/i)).toBeVisible();

    // Navegacao lateral principal presente.
    for (const item of ["Agente", "Tarefas", "Empresa"]) {
      await expect(page.getByText(item, { exact: false }).first()).toBeVisible();
    }
  });
  // Navegacao para sub-rotas (/tasks, /agentic) depende do backend e nao e
  // deterministica offline; a jornada profunda e coberta por go test no runtime.
  // O smoke offline foca no shell da home, que renderiza local-first de forma estavel.
});
