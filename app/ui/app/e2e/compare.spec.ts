import { expect, test, type Page } from "@playwright/test";

// E2E offline da comparação lado a lado (P0-4). Nenhum provedor externo: o
// backend é simulado no próprio navegador, então o teste é determinístico e
// roda no CI sem modelo baixado nem credencial.
const MODELS = [
  {
    name: "llama3.2:3b",
    model: "llama3.2:3b",
    digest: "sha256:llama3.2:3b",
    size: 2_019_392_431,
    modified_at: "2026-01-02T00:00:00Z",
    details: { families: ["llama"], parameter_size: "3B" },
  },
  {
    name: "qwen2.5:7b",
    model: "qwen2.5:7b",
    digest: "sha256:qwen2.5:7b",
    size: 4_683_088_672,
    modified_at: "2026-01-03T00:00:00Z",
    details: { families: ["qwen2"], parameter_size: "7B" },
  },
];

function jsonl(events: Array<Record<string, unknown>>): string {
  return `${events.map((event) => JSON.stringify(event)).join("\n")}\n`;
}

// A ordem importa: o Playwright avalia as rotas registradas por último primeiro,
// então o coringa é registrado PRIMEIRO para não engolir as rotas específicas.
async function mockBackend(page: Page, options: { failModel?: string } = {}) {
  await page.route("**/api/**", (route) => route.fulfill({ json: {} }));
  await page.route("**/api/version", (route) =>
    route.fulfill({ json: { version: "0.0.0-e2e" } }),
  );
  await page.route("**/api/tags", (route) =>
    route.fulfill({ json: { models: MODELS } }),
  );
  await page.route("**/api/v1/cloud", (route) =>
    route.fulfill({ json: { disabled: false, source: "none" } }),
  );
  await page.route("**/api/experimental/model-recommendations", (route) =>
    route.fulfill({ json: { recommendations: [] } }),
  );
  await page.route("**/api/v1/chat/**", async (route) => {
    const body = route.request().postDataJSON() as { model?: string; temporary?: boolean };
    const model = body.model ?? "desconhecido";
    if (options.failModel === model) {
      await route.fulfill({ status: 500, json: { error: "modelo indisponível" } });
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: "application/x-ndjson",
      body: jsonl([
        { eventName: "chat_created", chatId: `temp-${model}` },
        { eventName: "chat", content: `Resposta simulada de ${model}` },
        { eventName: "done" },
      ]),
    });
  });
}

async function openComparison(page: Page) {
  await page.goto("/compare");
  await expect(
    page.getByRole("heading", { name: "Comparar modelos", level: 2 }),
  ).toBeVisible();
}

async function selectModel(page: Page, model: string) {
  await page.getByRole("checkbox", { name: `Comparar com ${model}` }).check();
}

test.describe("comparação de modelos", () => {
  test("roda a mesma pergunta em dois modelos e mostra as respostas lado a lado", async ({
    page,
  }) => {
    await mockBackend(page);
    await openComparison(page);

    const runButton = page.getByRole("button", { name: "Comparar modelos" });
    await expect(runButton).toBeDisabled();

    await selectModel(page, "llama3.2:3b");
    await selectModel(page, "qwen2.5:7b");
    await page
      .getByRole("textbox", { name: "Pergunta para comparar" })
      .fill("explique o que é uma fila de mensagens mortas");
    await expect(runButton).toBeEnabled();

    await runButton.click();

    await expect(
      page.getByRole("article", { name: "Resposta de llama3.2:3b" }),
    ).toContainText("Resposta simulada de llama3.2:3b");
    await expect(
      page.getByRole("article", { name: "Resposta de qwen2.5:7b" }),
    ).toContainText("Resposta simulada de qwen2.5:7b");
    await expect(page.getByText("2 de 2 responderam.")).toBeVisible();
  });

  test("um modelo que falha não derruba a resposta do outro", async ({ page }) => {
    await mockBackend(page, { failModel: "qwen2.5:7b" });
    await openComparison(page);

    await selectModel(page, "llama3.2:3b");
    await selectModel(page, "qwen2.5:7b");
    await page
      .getByRole("textbox", { name: "Pergunta para comparar" })
      .fill("pergunta curta");
    await page.getByRole("button", { name: "Comparar modelos" }).click();

    await expect(
      page.getByRole("article", { name: "Resposta de llama3.2:3b" }),
    ).toContainText("Resposta simulada de llama3.2:3b");
    await expect(
      page.getByRole("article", { name: "Resposta de qwen2.5:7b" }),
    ).toContainText("HTTP 500");
    await expect(page.getByText("1 de 2 responderam.")).toBeVisible();
  });
});
