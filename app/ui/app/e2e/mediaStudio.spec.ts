import { test, expect } from "@playwright/test";

// PROVA EM BROWSER DO ESTUDIO MULTIMODAL (item 45).
//
// Por que este teste existe: o painel so e' montado quando HA' uma missao
// (`AgenticSplitShell.tsx:715`), e o E2E de shell offline nao chega la'. Aqui a
// fronteira de REDE e' interceptada (`page.route`) e o que se prova e' o que o
// utilizador VE' no browser: o resumo do estudio e as 6 operacoes.
//
// *Nao ha' provider real:* `providerConfigurado` fica `undefined` de proposito,
// e o painel deve dizer "a confirmar" — nunca prometer disponibilidade que nao
// mediu. Isso e' o "fallback honesto" exigido pelo item 45.
//
// Ancoras (todas medidas no codigo, nao supostas):
//   AgenticConsole.tsx:160  -> selecao da missao por `?mission_id=`
//   AgenticConsole.tsx:165  -> GET /api/agent/v1/missions            { missions }
//   AgenticConsole.tsx:121  -> GET /api/agent/v1/missions/<id>
//   AgenticConsole.tsx:122  -> GET /api/agent/v1/missions/<id>/events
//   AgenticConsole.tsx:120  -> GET /api/agent/v1/metrics
//   AgenticConsole.tsx:143  -> GET /api/agent/v1/tools
//   AgenticSplitShell.tsx:643 -> botao da aba "Artefatos"
//   MediaStudioPanel.tsx:108  -> atributo `data-estudio-resumo`
//   MediaStudioPanel.tsx:116  -> atributo `data-estudio-recado`

const MISSAO = {
  id: "mis_e2e",
  objective: "Prova E2E do estudio multimodal",
  state: "PENDING",
  workspace: "/workspace/e2e",
  plan: { steps: [] },
  approvals: [],
  artifacts: [],
};

// As 6 operacoes declaradas pelo catalogo do estudio (`mediaOperacoes.ts`).
const OPERACOES = [
  "tone.generate",
  "audio.transcribe",
  "image.analyze",
  "image.generate",
  "video.generate",
  "speech.generate",
];

test.describe("Estudio multimodal (item 45)", () => {
  test.beforeEach(async ({ page }) => {
    // Intercepta a fronteira de rede. Ordem importa: o mais especifico primeiro.
    await page.route("**/api/agent/v1/missions/*/events", (route) =>
      route.fulfill({ json: { events: [] } }),
    );
    await page.route("**/api/agent/v1/missions/*", (route) =>
      route.fulfill({ json: MISSAO }),
    );
    await page.route("**/api/agent/v1/missions", (route) =>
      route.fulfill({ json: { missions: [MISSAO] } }),
    );
    await page.route("**/api/agent/v1/metrics", (route) =>
      route.fulfill({ json: {} }),
    );
    await page.route("**/api/agent/v1/tools", (route) =>
      route.fulfill({ json: { tools: [] } }),
    );
  });

  test("o painel mostra a operacao e o estado honesto do provider", async ({
    page,
  }) => {
    await page.goto("/agentic?mission_id=mis_e2e");

    // A aba de artefactos pode ou nao estar activa por omissao; clicamos se
    // existir (o shell tem as abas Browser / Editor / Artefatos).
    const abaArtefatos = page.getByRole("button", { name: /^Artefatos/ });
    if ((await abaArtefatos.count()) > 0) {
      await abaArtefatos.first().click();
    }

    const resumo = page.locator("[data-estudio-resumo]");
    await expect(resumo).toBeVisible();

    // Sem provider medido, o resumo NAO pode afirmar disponibilidade.
    await expect(resumo).toContainText(/confirmar/i);

    // As 6 operacoes do catalogo estao lancaveis na UI (o servidor decide ao
    // executar; a UI nao inventa disponibilidade).
    for (const operacao of OPERACOES) {
      await expect(
        page.getByText(operacao, { exact: false }).first(),
      ).toBeVisible();
    }
  });
});
