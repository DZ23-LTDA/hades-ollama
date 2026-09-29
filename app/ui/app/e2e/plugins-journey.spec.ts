import { test, expect, type Page, type Route } from "@playwright/test";
import { mkdirSync } from "node:fs";
import { join } from "node:path";

// Tela "Plugins" (marketplace de conectores + MCP). O catalogo interno rico
// (connectorCatalog.ts) e sempre exibido, mesclado com o que o backend homologou.
// Os contratos /connectors, /connector-catalog, /mcp e /api/dz23/cli-catalog sao
// atendidos aqui por stores em memoria no nivel de rede. A superficie e resiliente:
// mostra o catalogo mesmo com o backend indisponivel.

const EVIDENCE_DIR = join(process.cwd(), "..", "..", "..", "docs", "evidencias");

async function captureEvidence(page: Page, name: string) {
  mkdirSync(EVIDENCE_DIR, { recursive: true });
  await page.screenshot({ path: join(EVIDENCE_DIR, `${name}.png`), fullPage: true });
}

async function installBackend(
  page: Page,
  options: { failAll?: boolean } = {},
) {
  const fail = options.failAll ?? false;
  const ok = (route: Route, json: unknown) =>
    fail
      ? route.fulfill({ status: 503, json: { error: "backend unavailable" } })
      : route.fulfill({ json });

  await page.route("**/api/agent/v1/connectors", (route) =>
    ok(route, { connectors: [] }),
  );
  await page.route("**/api/agent/v1/connector-catalog", (route) =>
    ok(route, { connectors: [] }),
  );
  await page.route("**/api/agent/v1/mcp", (route) =>
    ok(route, { servers: [], remote_servers: [] }),
  );
  await page.route("**/api/dz23/cli-catalog", (route) => ok(route, { tools: [] }));
}

test.describe("Tela Plugins — marketplace de conectores", () => {
  test("renderiza o catálogo rico com categorias, em claro e escuro", async ({
    page,
  }) => {
    await installBackend(page);

    await page.goto("/plugins");
    await expect(
      page.getByRole("heading", { name: "Plugins", level: 2 }),
    ).toBeVisible();

    // Conectores de varias categorias vindos do catalogo interno.
    for (const name of ["Gmail", "Google Workspace", "Notion", "GitHub", "Instagram"]) {
      await expect(
        page.getByRole("heading", { name, level: 4 }),
      ).toBeVisible();
    }
    // Fonte de dados tambem presente.
    await expect(
      page.getByRole("heading", { name: "Similarweb", level: 4 }),
    ).toBeVisible();
    // Chips de categoria do Manus.
    await expect(
      page.getByRole("tab", { name: "Fontes de dados" }),
    ).toBeVisible();
    await expect(
      page.getByText("disponíveis no catálogo", { exact: false }),
    ).toBeVisible();
    await captureEvidence(page, "plugins-marketplace-desktop");

    // Mesma tela no tema escuro (paridade visual com o Manus).
    await page.emulateMedia({ colorScheme: "dark" });
    await captureEvidence(page, "plugins-marketplace-dark");
    await page.emulateMedia({ colorScheme: "light" });
  });

  test("a busca filtra o catálogo por texto", async ({ page }) => {
    await installBackend(page);

    await page.goto("/plugins");
    await page.getByLabel("Pesquisar conectores").fill("notion");
    await expect(
      page.getByRole("heading", { name: "Notion", level: 4 }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Gmail", level: 4 }),
    ).toHaveCount(0);
  });

  test("resiliente: mostra o catálogo mesmo com o backend indisponível", async ({
    page,
  }) => {
    await installBackend(page, { failAll: true });

    await page.goto("/plugins");
    // Sem alerta de erro: o marketplace continua utilizável offline.
    await expect(
      page.getByRole("heading", { name: "Gmail", level: 4 }),
    ).toBeVisible();
    await expect(
      page.getByText("Nenhum connector configurado", { exact: false }),
    ).toBeVisible();
    await captureEvidence(page, "plugins-offline-desktop");
  });

  test("funciona em 390x844 sem overflow horizontal", async ({ page }) => {
    await installBackend(page);

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/plugins");
    await expect(
      page.getByRole("heading", { name: "Gmail", level: 4 }),
    ).toBeVisible();

    const overflow = await page.evaluate(
      () =>
        document.documentElement.scrollWidth -
        document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
    await captureEvidence(page, "plugins-marketplace-mobile");
  });
});
