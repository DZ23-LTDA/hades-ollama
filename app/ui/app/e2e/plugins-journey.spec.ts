import { test, expect, type Page, type Route } from "@playwright/test";
import { mkdirSync } from "node:fs";
import { join } from "node:path";

// Tela "Plugins" (connectors + MCP). Consome quatro contratos em paralelo:
// /connectors, /connector-catalog, /mcp e /api/dz23/cli-catalog. Todos sao
// atendidos aqui por stores em memoria no nivel de rede. O catalogo real e
// pesquisavel; conectores e MCP configurados tem estados vazios proprios.

const EVIDENCE_DIR = join(process.cwd(), "..", "..", "..", "docs", "evidencias");

async function captureEvidence(page: Page, name: string) {
  mkdirSync(EVIDENCE_DIR, { recursive: true });
  await page.screenshot({ path: join(EVIDENCE_DIR, `${name}.png`), fullPage: true });
}

type CatalogEntry = {
  id: string;
  name: string;
  category: string;
  kind: string;
  description: string;
  auth: string;
  source: string;
  status: string;
};

const catalogSeed: CatalogEntry[] = [
  {
    id: "github",
    name: "GitHub",
    category: "Desenvolvimento",
    kind: "oauth",
    description: "Repositórios, issues e pull requests.",
    auth: "oauth",
    source: "builtin",
    status: "available",
  },
  {
    id: "slack",
    name: "Slack",
    category: "Comunicação",
    kind: "oauth",
    description: "Mensagens e canais.",
    auth: "oauth",
    source: "builtin",
    status: "available",
  },
];

async function installPluginsContract(
  page: Page,
  options: { failCatalog?: boolean; catalog?: CatalogEntry[] } = {},
) {
  const catalog = options.catalog ?? catalogSeed;
  let catalogCalls = 0;

  await page.route("**/api/agent/v1/connectors", async (route: Route) => {
    if (route.request().method() === "GET") {
      await route.fulfill({ json: { connectors: [] } });
      return;
    }
    await route.fallback();
  });
  await page.route("**/api/agent/v1/connector-catalog", async (route: Route) => {
    catalogCalls += 1;
    if (options.failCatalog && catalogCalls === 1) {
      await route.fulfill({
        status: 503,
        json: { error: "connector catalog unavailable" },
      });
      return;
    }
    await route.fulfill({ json: { connectors: catalog } });
  });
  await page.route("**/api/agent/v1/mcp", async (route: Route) => {
    if (route.request().method() === "GET") {
      await route.fulfill({ json: { servers: [], remote_servers: [] } });
      return;
    }
    await route.fallback();
  });
  await page.route("**/api/dz23/cli-catalog", async (route: Route) => {
    await route.fulfill({ json: { tools: [] } });
  });

  return { catalogCalls: () => catalogCalls };
}

test.describe("Tela Plugins (connectors + MCP)", () => {
  test("mostra o catálogo real pesquisável e os estados vazios de configuração", async ({
    page,
  }) => {
    await installPluginsContract(page);

    await page.goto("/plugins");
    await expect(
      page.getByRole("heading", { name: "Plugins", level: 2 }),
    ).toBeVisible();

    // Catalogo real renderizado a partir do contrato.
    await expect(
      page.getByRole("heading", { name: "GitHub", level: 4 }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Slack", level: 4 }),
    ).toBeVisible();
    await expect(
      page.getByText("Disponível para configuração").first(),
    ).toBeVisible();

    // Estados vazios das secoes configuradas.
    await expect(
      page.getByText("Nenhum connector configurado", { exact: false }),
    ).toBeVisible();
    await expect(
      page.getByText("Nenhum servidor MCP carregado", { exact: false }),
    ).toBeVisible();
    await captureEvidence(page, "plugins-sucesso-desktop");

    // Pesquisa filtra o catalogo.
    await page.getByLabel("Pesquisar conectores").fill("slack");
    await expect(
      page.getByRole("heading", { name: "Slack", level: 4 }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "GitHub", level: 4 }),
    ).toHaveCount(0);
    await captureEvidence(page, "plugins-busca-desktop");
  });

  test("mostra erro de carregamento com nova tentativa", async ({ page }) => {
    await installPluginsContract(page, { failCatalog: true });

    await page.goto("/plugins");
    const alert = page.getByRole("alert");
    await expect(alert).toContainText("connector catalog unavailable");
    await captureEvidence(page, "plugins-erro-desktop");

    await page.getByRole("button", { name: "Tentar novamente" }).click();
    await expect(
      page.getByRole("heading", { name: "GitHub", level: 4 }),
    ).toBeVisible();
  });

  test("funciona em 390x844 sem overflow horizontal", async ({ page }) => {
    await installPluginsContract(page);

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/plugins");
    await expect(
      page.getByRole("heading", { name: "GitHub", level: 4 }),
    ).toBeVisible();

    const overflow = await page.evaluate(
      () =>
        document.documentElement.scrollWidth -
        document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
    await captureEvidence(page, "plugins-sucesso-mobile");
  });
});
