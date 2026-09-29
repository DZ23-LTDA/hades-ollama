import { test, expect, type Page, type Route } from "@playwright/test";
import { mkdirSync } from "node:fs";
import { join } from "node:path";

// Tela "Projetos" (projects). O carregamento consome /api/agent/v1/projects,
// atendido aqui por um store em memoria no nivel de rede. A criacao so aparece
// como sucesso quando o POST responde 201 e o projeto volta na listagem.

const EVIDENCE_DIR = join(process.cwd(), "..", "..", "..", "docs", "evidencias");

async function captureEvidence(page: Page, name: string) {
  mkdirSync(EVIDENCE_DIR, { recursive: true });
  await page.screenshot({ path: join(EVIDENCE_DIR, `${name}.png`), fullPage: true });
}

type Project = {
  id: string;
  name: string;
  root?: string;
  organization_id?: string;
  created_at: string;
  updated_at: string;
};

async function installProjectsContract(
  page: Page,
  options: { failFirstList?: boolean; seed?: Project[] } = {},
) {
  const projects: Project[] = [...(options.seed ?? [])];
  const createBodies: Array<Record<string, unknown>> = [];
  let listCalls = 0;

  await page.route("**/api/agent/v1/projects", async (route: Route) => {
    const request = route.request();
    if (request.method() === "GET") {
      listCalls += 1;
      if (options.failFirstList && listCalls === 1) {
        await route.fulfill({
          status: 503,
          json: { error: "projects store unavailable" },
        });
        return;
      }
      await route.fulfill({ json: { projects } });
      return;
    }
    if (request.method() === "POST") {
      const body = request.postDataJSON() as Record<string, unknown>;
      createBodies.push(body);
      const now = new Date().toISOString();
      const project: Project = {
        id: `proj-${projects.length + 1}`,
        name: String(body.name ?? ""),
        root: (body.root as string | undefined) || undefined,
        created_at: now,
        updated_at: now,
      };
      projects.unshift(project);
      await route.fulfill({ status: 201, json: project });
      return;
    }
    await route.fallback();
  });

  return { projects, createBodies, listCalls: () => listCalls };
}

test.describe("Tela Projetos (projects)", () => {
  test("carrega vazio, cria o projeto e confirma na listagem", async ({
    page,
  }) => {
    const contract = await installProjectsContract(page);

    await page.goto("/projects");
    await expect(
      page.getByRole("heading", { name: "Projetos", level: 2 }),
    ).toBeVisible();
    await expect(
      page.getByText("Crie um projeto para manter instruções", {
        exact: false,
      }),
    ).toBeVisible();
    await captureEvidence(page, "projetos-vazio-desktop");

    await page.getByLabel("Nome do novo projeto").fill("Plataforma Aurora");
    await page
      .getByLabel("Workspace do novo projeto")
      .fill("aurora-workspace");
    await page.getByRole("button", { name: "Criar", exact: true }).click();

    await expect(page.getByText("Plataforma Aurora")).toBeVisible();
    await expect(
      page.getByRole("status").filter({ hasText: "Projeto criado" }),
    ).toBeVisible();
    expect(contract.createBodies).toEqual([
      { name: "Plataforma Aurora", root: "aurora-workspace" },
    ]);
    await captureEvidence(page, "projetos-sucesso-desktop");
  });

  test("mostra erro de carregamento com nova tentativa", async ({ page }) => {
    await installProjectsContract(page, { failFirstList: true });

    await page.goto("/projects");
    const alert = page.getByRole("alert");
    await expect(alert).toContainText("projects store unavailable");
    await captureEvidence(page, "projetos-erro-desktop");

    await page.getByRole("button", { name: "Tentar novamente" }).click();
    await expect(
      page.getByText("Crie um projeto para manter instruções", {
        exact: false,
      }),
    ).toBeVisible();
  });

  test("funciona em 390x844 sem overflow horizontal", async ({ page }) => {
    await installProjectsContract(page, {
      seed: [
        {
          id: "proj-seed",
          name: "Projeto Órion",
          root: "orion",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    });

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto("/projects");
    await expect(page.getByText("Projeto Órion")).toBeVisible();

    const overflow = await page.evaluate(
      () =>
        document.documentElement.scrollWidth -
        document.documentElement.clientWidth,
    );
    expect(overflow).toBeLessThanOrEqual(0);
    await captureEvidence(page, "projetos-sucesso-mobile");
  });
});
