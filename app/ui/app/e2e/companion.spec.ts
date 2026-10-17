import { expect, test } from "@playwright/test";

/**
 * Prova E2E do Companion (item 46) — a UI mostra o estado que o SERVIDOR declarou.
 *
 * A rota existe no produto (`GET /api/agent/v1/missions/:id/companion`) e tem testes de
 * servidor verdes (6/6). O fixture `companion-fixture.mjs` serve o contrato
 * `{mission_id, estado, razao, scope}` — o mesmo que `src/lib/companionApi.ts:46-51` exige —
 * e so' responde quando o `Referer` traz `mission_id=mis_companion_e2e` (isolamento medido:
 * servir rotas a todos os testes reprovou a `shell.spec.ts` por strict mode violation).
 *
 * CLASSIFICACAO: `LOCAL_BACKEND_REAL` — HTTP de verdade contra o stub do repositorio.
 * ***Nao prova o backend Go em producao***; prova a fiacao UI -> HTTP -> contrato.
 */

const MISSAO = "mis_companion_e2e";

/**
 * O Companion so' e' renderizado quando a aba de canvas e' `"artifacts"`:
 *   `AgenticSplitShell.tsx:706`  `) : activeCanvasTab === "artifacts" ? (`
 *   `:710`                       `{mission?.id ? <CompanionLigado missionID={mission.id} /> : null}`
 * A aba activa e' estado local (`:140`) e muda por clique nos botoes `:619` (browser),
 * `:635` (artifacts) e `:648` (code). *Medido: sem este clique, a rota `/companion`
 * NUNCA e' pedida ao servidor* — foi assim que a causa foi isolada.
 */
async function abrirAbaDeArtefatos(page: import("@playwright/test").Page) {
  await page.goto(`/agentic?mission_id=${MISSAO}`);
  const aba = page.getByRole("button", { name: /Artefatos/i }).first();
  await aba.waitFor({ state: "visible", timeout: 20000 });
  await aba.click();
}

test.describe("Companion (item 46)", () => {
  test("mostra o estado REAL que o servidor declarou, com o titulo do mapa da UI", async ({
    page,
  }) => {
    await abrirAbaDeArtefatos(page);

    // O fixture declara `estado: "a_trabalhar"`. O mapa `TEXTO_DO_ESTADO`
    // (`CompanionMascote.tsx:45-77`) traduz esse estado para o titulo "A trabalhar".
    // Se a UI inventasse um estado proprio, este texto nao apareceria.
    await expect(page.getByText("A trabalhar").first()).toBeVisible({
      timeout: 20000,
    });
  });

  test("expoe o mascote com nome acessivel (nao e' so' decoracao)", async ({ page }) => {
    await abrirAbaDeArtefatos(page);

    // `CompanionMascote.tsx:112` -> aria-label={`Companion ${nome}`}.
    // Prova que o mascote tem contrato de acessibilidade, nao apenas pixels.
    await expect(page.getByLabel(/^Companion /).first()).toBeVisible({
      timeout: 20000,
    });
  });
});
