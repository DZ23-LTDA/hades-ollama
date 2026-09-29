import { test, expect, type Page, type Route } from "@playwright/test";
import { mkdirSync } from "node:fs";
import { join } from "node:path";

// Tela "Habilidades" (skills). O carregamento consome apenas o contrato
// /api/agent/v1/skills, atendido aqui por um store em memoria no nivel de rede.
// A tela distingue skills confiaveis das que exigem revisao e trata vazio/erro.

const EVIDENCE_DIR = join(process.cwd(), "..", "..", "..", "docs", "evidencias");

async function captureEvidence(page: Page, name: string) {
  mkdirSync(EVIDENCE_DIR, { recursive: true });
  await page.screenshot({ path: join(EVIDENCE_DIR, `${name}.png`), fullPage: true });
}

type Skill = {
  id: string;
  version: string;
  description: string;
  scopes?: string[];
  tools?: string[];
  trusted: boolean;
  enabled: boolean;
};

async function installSkillsContract(
  page: Page,
  options: { failFirstList?: boolean; seed?: Skill[] } = {},
) {
  const skills: Skill[] = [...(options.seed ?? [])];
  let listCalls = 0;

  await page.route("**/api/agent/v1/skills", async (route: Route) => {
    if (route.request().method() === "GET") {
      listCalls += 1;
      if (options.failFirstList && listCalls === 1) {
        await route.fulfill({
          status: 503,
          json: { error: "skills store unavailable" },
        });
        return;
      }
      await route.fulfill({ json: { skills } });
      return;
    }
    await route.fallback();
  });

  return { skills, listCalls: () => listCalls };
}

test.describe("Tela Habilidades (skills)", () => {
  test("mostra o estado vazio quando o runtime não carregou skills", async ({
    page,
  }) => {
    await installSkillsContract(page);

    await page.goto("/skills");
    await expect(
      page.getByRole("heading", { name: "Habilidades", level: 2 }),
    ).toBeVisible();
    await expect(
      page.getByText("Nenhuma skill foi carregada pelo runtime", {
        exact: false,
      }),
    ).toBeVisible();
    await captureEvidence(page, "habilidades-vazio-desktop");
  });

  test("lista as skills carregadas distinguindo confiança", async ({ page }) => {
    await installSkillsContract(page, {
      seed: [
        {
          id: "pdf-fill",
          version: "1.2.0",
          description: "Preenche formulários PDF com validação.",
          tools: ["pdf.read", "pdf.write"],
          trusted: true,
          enabled: true,
        },
        {
          id: "web-scan",
          version: "0.4.1",
          description: "Coleta estruturada de páginas.",
          tools: ["web.fetch"],
          trusted: false,
          enabled: false,
        },
      ],
    });

    await page.goto("/skills");
    await expect(page.getByText("pdf-fill")).toBeVisible();
    await expect(
      page.getByText("Preenche formulários PDF com validação."),
    ).toBeVisible();
    await expect(
      page.getByText("2 tools · trusted", { exact: false }),
    ).toBeVisible();
    await expect(
      page.getByText("1 tools · requer revisão", { exact: false }),
    ).toBeVisible();
    await captureEvidence(page, "habilidades-sucesso-desktop");
  });

  test("mostra erro de carregamento com nova tentativa", async ({ page }) => {
    await installSkillsContract(page, { failFirstList: true });

    await page.goto("/skills");
    const alert = page.getByRole("alert");
    await expect(alert).toContainText("skills store unavailable");
    await captureEvidence(page, "habilidades-erro-desktop");

    await page.getByRole("button", { name: "Tentar novamente" }).click();
    await expect(
      page.getByText("Nenhuma skill foi carregada pelo runtime", {
        exact: false,
      }),
    ).toBeVisible();
  });

  test("funciona em 390x844 sem overflow horizontal", async ({ page }) => {
    await installSkillsContract(page, {
      seed: [
        {
          id: "pdf-fill",
          version: "1.2.0",
          description: "Preenche formulários PDF.",
          tools: ["pdf.read"],
          trusted: true,
          enabled: true,
        },
      ],
    });

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/skills");
    await expect(page.getByText("pdf-fill")).toBeVisible();

    const overflow = await page.evaluate(
      () =>
        document.documentElement.scrollWidth -
        document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
    await captureEvidence(page, "habilidades-sucesso-mobile");
  });
});
