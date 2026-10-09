import { expect, test, type Page } from "@playwright/test";

// E2E offline da biblioteca de prompts (P0-5). O backend é simulado no próprio
// navegador: nada de modelo baixado nem credencial externa. A persistência é
// provada com um reload de verdade, lendo o mesmo localStorage.
const MODELS = [
  {
    name: "llama3.2:3b",
    model: "llama3.2:3b",
    digest: "sha256:llama3.2:3b",
    size: 2_019_392_431,
    modified_at: "2026-01-02T00:00:00Z",
    details: { families: ["llama"], parameter_size: "3B" },
  },
];

// Preferências já migradas: o app lê /api/v1/settings no beforeLoad da home.
// Um corpo vazio deixa a home sem OnboardingVersion e presa em estado de
// carregamento (o teste expirava antes mesmo do painel montar). Por isso o
// retorno é completo e com OnboardingVersion igual ao CURRENT_ONBOARDING_VERSION.
const SETTINGS = {
  Expose: false,
  Browser: false,
  Survey: false,
  Models: "",
  Agent: false,
  Tools: false,
  WorkingDir: "",
  ContextLength: 4096,
  TurboEnabled: false,
  WebSearchEnabled: false,
  ThinkEnabled: false,
  ThinkLevel: "medium",
  SelectedModel: "llama3.2:3b",
  SidebarOpen: true,
  LastHomeView: "chat",
  OnboardingVersion: 1,
  AutoUpdateEnabled: false,
  ClaudeDesktopUsed: false,
};

// A ordem importa: o Playwright avalia as rotas registradas por último primeiro,
// então o coringa é registrado PRIMEIRO para não engolir as rotas específicas.
async function mockBackend(page: Page) {
  await page.route("**/api/**", (route) => route.fulfill({ json: {} }));
  await page.route("**/api/version", (route) =>
    route.fulfill({ json: { version: "0.0.0-e2e" } }),
  );
  await page.route("**/api/v1/settings", (route) =>
    route.fulfill({ json: { settings: SETTINGS } }),
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
}

async function openLibrary(page: Page) {
  await page.goto("/");
  await page.getByRole("button", { name: "Biblioteca de prompts" }).click();
  await expect(
    page.getByRole("region", { name: "Biblioteca de prompts" }),
  ).toBeVisible();
}

test.describe("biblioteca de prompts", () => {
  test("cria um prompt, sobrevive ao reload e reusa no campo de mensagem", async ({
    page,
  }) => {
    await mockBackend(page);
    await openLibrary(page);

    await expect(page.getByText("4 prompt(s) salvos neste navegador.")).toBeVisible();

    await page.getByLabel("Título do prompt").fill("Resumo semanal");
    await page
      .getByLabel("Conteúdo do prompt")
      .fill("Resuma os últimos sete dias em cinco pontos objetivos.");
    await page.getByLabel("Pasta do prompt").fill("Rotina");
    await page.getByLabel("Etiquetas do prompt").fill("resumo, rotina");
    await page.getByRole("button", { name: "Salvar prompt" }).click();

    await expect(page.getByText("Prompt salvo na biblioteca.")).toBeVisible();
    await expect(page.getByText("5 prompt(s) salvos neste navegador.")).toBeVisible();
    await expect(page.getByText("Resumo semanal")).toBeVisible();

    await page.reload();
    await page.getByRole("button", { name: "Biblioteca de prompts" }).click();
    await expect(page.getByText("Resumo semanal")).toBeVisible();
    await expect(page.getByText("5 prompt(s) salvos neste navegador.")).toBeVisible();

    // Usar fecha o painel e entrega o texto pronto ao campo de mensagem.
    await page
      .getByRole("button", { name: "Usar o prompt Resumo semanal" })
      .click();
    await expect(
      page.getByRole("region", { name: "Biblioteca de prompts" }),
    ).toBeHidden();
    await expect(page.getByLabel("Mensagem para o agente")).toHaveValue(
      "Resuma os últimos sete dias em cinco pontos objetivos.",
    );
  });

  test("bloqueia um prompt com variáveis até o preenchimento e depois entrega o texto", async ({
    page,
  }) => {
    await mockBackend(page);
    await openLibrary(page);

    await page.getByRole("button", { name: /^Usar o prompt \/goal/ }).click();

    const objetivo = page.getByLabel("Valor de objetivo");
    const contexto = page.getByLabel("Valor de contexto");
    await expect(objetivo).toBeVisible();
    await expect(contexto).toBeVisible();

    // Com uma variável em branco o painel recusa e diz exatamente qual falta.
    await objetivo.fill("reduzir o tempo de build");
    await page.getByRole("button", { name: "Gerar e usar" }).click();
    await expect(page.getByRole("alert")).toHaveText("Preencha: contexto");
    await expect(page.getByLabel("Mensagem para o agente")).toHaveValue("");

    await contexto.fill("monorepo Go e React");
    await page.getByRole("button", { name: "Gerar e usar" }).click();
    await expect(page.getByLabel("Mensagem para o agente")).toHaveValue(
      /reduzir o tempo de build[\s\S]*monorepo Go e React/,
    );
  });
});
