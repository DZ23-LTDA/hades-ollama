import { test, expect, type Page, type Route } from "@playwright/test";
import { mkdirSync } from "node:fs";
import { join } from "node:path";

// Tela "Biblioteca" (library). Lista os artifacts produzidos pelas missoes a
// partir do contrato /api/agent/v1/missions, atendido aqui por um store em
// memoria no nivel de rede. Cada artifact expoe nome, tamanho e SHA-256.

const EVIDENCE_DIR = join(process.cwd(), "..", "..", "..", "docs", "evidencias");

async function captureEvidence(page: Page, name: string) {
  mkdirSync(EVIDENCE_DIR, { recursive: true });
  await page.screenshot({ path: join(EVIDENCE_DIR, `${name}.png`), fullPage: true });
}

type Artifact = {
  id: string;
  name: string;
  sha256: string;
  size: number;
  media_type?: string;
};
type Mission = {
  id: string;
  version: number;
  objective: string;
  state: string;
  artifacts?: Artifact[];
  created_at: string;
  updated_at: string;
};

async function installLibraryContract(
  page: Page,
  options: { failFirstList?: boolean; seed?: Mission[] } = {},
) {
  const missions: Mission[] = [...(options.seed ?? [])];
  let listCalls = 0;

  await page.route("**/api/agent/v1/missions", async (route: Route) => {
    if (route.request().method() === "GET") {
      listCalls += 1;
      if (options.failFirstList && listCalls === 1) {
        await route.fulfill({
          status: 503,
          json: { error: "missions store unavailable" },
        });
        return;
      }
      await route.fulfill({ json: { missions } });
      return;
    }
    await route.fallback();
  });

  return { missions, listCalls: () => listCalls };
}

const missionWithArtifact: Mission = {
  id: "mission-42",
  version: 1,
  objective: "Gerar relatório executivo",
  state: "DONE",
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  artifacts: [
    {
      id: "art-1",
      name: "relatorio-final.pdf",
      sha256:
        "abcd1234ef5678900000000000000000000000000000000000000000000000ff",
      size: 20_480,
      media_type: "application/pdf",
    },
  ],
};

test.describe("Tela Biblioteca (library)", () => {
  test("mostra o estado vazio quando nenhuma missão gerou artifacts", async ({
    page,
  }) => {
    await installLibraryContract(page);

    await page.goto("/library");
    await expect(
      page.getByRole("heading", { name: "Biblioteca", level: 2 }),
    ).toBeVisible();
    await expect(
      page.getByText("Os artifacts gerados aparecerão aqui", { exact: false }),
    ).toBeVisible();
    await captureEvidence(page, "biblioteca-vazio-desktop");
  });

  test("lista os artifacts com tamanho, hash e missão de origem", async ({
    page,
  }) => {
    await installLibraryContract(page, { seed: [missionWithArtifact] });

    await page.goto("/library");
    await expect(page.getByText("relatorio-final.pdf")).toBeVisible();
    await expect(
      page.getByText("20 KiB · SHA-256 abcd1234ef567890", { exact: false }),
    ).toBeVisible();
    await expect(
      page.getByText("Missão mission-42", { exact: false }),
    ).toBeVisible();
    await captureEvidence(page, "biblioteca-sucesso-desktop");
  });

  test("mostra erro de carregamento com nova tentativa", async ({ page }) => {
    await installLibraryContract(page, { failFirstList: true });

    await page.goto("/library");
    const alert = page.getByRole("alert");
    await expect(alert).toContainText("missions store unavailable");
    await captureEvidence(page, "biblioteca-erro-desktop");

    await page.getByRole("button", { name: "Tentar novamente" }).click();
    await expect(
      page.getByText("Os artifacts gerados aparecerão aqui", { exact: false }),
    ).toBeVisible();
  });

  test("funciona em 390x844 sem overflow horizontal", async ({ page }) => {
    await installLibraryContract(page, { seed: [missionWithArtifact] });

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/library");
    await expect(page.getByText("relatorio-final.pdf")).toBeVisible();

    const overflow = await page.evaluate(
      () =>
        document.documentElement.scrollWidth -
        document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
    await captureEvidence(page, "biblioteca-sucesso-mobile");
  });
});
