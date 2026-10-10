import { test, expect, type Page } from "@playwright/test";

// Regressão do loop de escrita de /api/v1/settings.
//
// Cenário: o endpoint aceita POST mas devolve um payload sem os campos
// gravados (payload parcial, proxy ou endpoint somente leitura). Antes da
// guarda em useSettings.ts, cada efeito "garanta X" reexecutava a cada
// invalidação da query e reenviava a escrita — a home disparava ~1.200 POST/s
// e ~1.180 GET/s, saturando o renderer e deixando a interface não interativa.
//
// Medido: 2.387 requisições a /api/v1/settings em 3s antes da correção; 16
// depois, e o mesmo valor em 6s (o fluxo converge, não é loop lento).

const MODELS = [
  {
    name: "llama3.2:3b",
    model: "llama3.2:3b",
    modified_at: "2026-01-02T00:00:00Z",
    size: 2019392431,
    digest: "sha256:llama3.2:3b",
    details: {
      format: "gguf",
      family: "llama",
      families: ["llama"],
      parameter_size: "3B",
      quantization_level: "Q4_K_M",
    },
  },
];

const FULL_SETTINGS = {
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

// Ordem importa: o catch-all entra primeiro e os stubs específicos depois.
async function mockBackend(page: Page, settingsBody: unknown) {
  await page.route("**/api/**", (route) => route.fulfill({ json: {} }));
  await page.route("**/api/version", (route) =>
    route.fulfill({ json: { version: "0.0.0-e2e" } }),
  );
  await page.route("**/api/tags", (route) => route.fulfill({ json: { models: MODELS } }));
  await page.route("**/api/v1/cloud", (route) =>
    route.fulfill({ json: { disabled: false, source: "none" } }),
  );
  await page.route("**/api/experimental/model-recommendations", (route) =>
    route.fulfill({ json: { recommendations: [] } }),
  );
  return page.route("**/api/v1/settings", (route) =>
    route.fulfill({ json: settingsBody }),
  );
}

// Um app saudável resolve os ajustes em poucas requisições. O limite é folgado
// de propósito: qualquer regressão para o comportamento anterior o estoura por
// três ordens de magnitude.
const MAX_SETTINGS_REQUESTS = 50;

test.describe("escrita de settings não entra em laço", () => {
  test.setTimeout(45_000);

  test("converge quando o backend não ecoa o valor gravado", async ({ page }) => {
    let settingsHits = 0;
    page.on("request", (request) => {
      if (request.url().includes("/api/v1/settings")) settingsHits += 1;
    });

    await mockBackend(page, {});
    await page.goto("/");
    await page.waitForTimeout(3000);
    const afterFirstWindow = settingsHits;
    await page.waitForTimeout(3000);

    expect(
      afterFirstWindow,
      `3s de carregamento geraram ${afterFirstWindow} requisições a /api/v1/settings`,
    ).toBeLessThan(MAX_SETTINGS_REQUESTS);
    expect(
      settingsHits - afterFirstWindow,
      "a segunda janela de 3s continuou disparando requisições: ainda há laço",
    ).toBe(0);
  });

  test("converge com payload completo", async ({ page }) => {
    let settingsHits = 0;
    page.on("request", (request) => {
      if (request.url().includes("/api/v1/settings")) settingsHits += 1;
    });

    await mockBackend(page, { settings: FULL_SETTINGS });
    await page.goto("/");
    await page.waitForTimeout(3000);

    expect(settingsHits).toBeLessThan(MAX_SETTINGS_REQUESTS);
  });
});
