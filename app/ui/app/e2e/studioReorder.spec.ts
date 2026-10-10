import {
  expect,
  test,
  type APIRequestContext,
  type Page,
} from "@playwright/test";

// Reordenação no Studio contra um backend HTTP real.
//
// Por que este spec existe: o canvas do Studio já tinha reordenação por arraste
// e por botão/teclado, com o helper puro `lib/studioReorder.ts` coberto por
// testes unitários — mas NENHUM teste provava que a nova ordem sobrevive ao
// recarregamento, ou seja, que ela é persistida no SERVIDOR
// (`updateBuilderVisual` -> POST /api/agent/v1/builders/{id}/visual).
//
// O que é verificado, ponta a ponta:
//
//   1. a ordem inicial do canvas vem da resposta de `GET
//      /api/agent/v1/builders/{id}` (fonte da verdade é o servidor);
//   2. arrastar um componente para outra posição grava a nova ordem via
//      `updateBuilderVisual` (`StudioCanvasPage.tsx:437-448`);
//   3. mover pelo botão acessível ("Mover para baixo", `:450-462`) grava a
//      mesma coisa pela via de teclado/leitor de tela;
//   4. depois de fechar a aba e abrir outra, a ordem lida do servidor é a nova
//      ordem — a persistência é real, não estado local de tela.
//
// Honestidade do escopo: o backend é o fixture HTTP real de
// `e2e/first-run-backend.mjs` (o mesmo processo que serve a jornada de primeira
// execução), com o contrato do builder em `e2e/studio-builder-fixture.mjs`. Nada
// de `page.route`: são requisições HTTP de verdade. O que é fixture é o ESTADO
// (um projeto com três componentes rotulados), não o caminho exercitado.
//
// Os eventos de arraste são `DragEvent` sintéticos com `DataTransfer` real,
// disparados em tarefas separadas porque `handleReorderDrop` lê o id arrastado
// do estado do React — precisa de uma renderização entre `dragstart` e `drop`.
// O caminho de botão usa clique real do Playwright.

const STUDIO_PATH = "/studio";
const RESET_PATH = "/api/__e2e/studio/reset";
const STATE_PATH = "/api/__e2e/studio/state";
const ROW_SELECTOR = 'div[title="Arraste para reordenar"]';

type ServerState = {
  version: number;
  order: string[];
  ids: string[];
  visual_writes: number;
  last_write_ids: string[];
};

async function resetStudio(request: APIRequestContext): Promise<void> {
  const response = await request.post(RESET_PATH);
  expect(response.ok()).toBeTruthy();
}

async function serverState(request: APIRequestContext): Promise<ServerState> {
  const response = await request.get(STATE_PATH);
  expect(response.ok()).toBeTruthy();
  return (await response.json()) as ServerState;
}

function rows(page: Page) {
  return page.locator(ROW_SELECTOR);
}

// As badges de cada componente têm o formato `#<posição> <tipo>`; ler o tipo em
// ordem de DOM é a forma direta de observar a ordem do canvas. O texto chega em
// caixa alta por causa do `uppercase` do Tailwind, então normalizamos.
async function uiOrder(page: Page): Promise<string[]> {
  const badges = await rows(page).locator("span.font-mono").allInnerTexts();
  return badges.map((text) =>
    text.trim().toLowerCase().split(/\s+/).slice(1).join(" "),
  );
}

async function expectUiOrder(page: Page, expected: string[]): Promise<void> {
  await expect
    .poll(async () => (await uiOrder(page)).join(","))
    .toBe(expected.join(","));
}

async function dragRow(
  page: Page,
  fromIndex: number,
  toIndex: number,
): Promise<void> {
  await page.evaluate(
    async ({ fromIndex, toIndex }) => {
      const elements = Array.from(
        document.querySelectorAll('div[title="Arraste para reordenar"]'),
      );
      const source = elements[fromIndex];
      const target = elements[toIndex];
      if (!source || !target) {
        throw new Error(
          `linhas do canvas não encontradas (${elements.length})`,
        );
      }
      const dataTransfer = new DataTransfer();
      const fire = (element: Element, type: string) => {
        element.dispatchEvent(
          new DragEvent(type, {
            bubbles: true,
            cancelable: true,
            dataTransfer,
          }),
        );
      };
      // Tarefas separadas de propósito: o handler de `drop` lê o id arrastado do
      // estado do React, que só é atualizado depois de uma renderização.
      const tick = () => new Promise((resolve) => setTimeout(resolve, 50));
      fire(source, "dragstart");
      await tick();
      fire(target, "dragover");
      await tick();
      fire(target, "drop");
      await tick();
      fire(source, "dragend");
    },
    { fromIndex, toIndex },
  );
}

// Reinício de verdade: fecha a aba e abre outra. Recarregar a mesma aba poderia
// reaproveitar o documento em memória; uma aba nova obriga a reler a ordem do
// servidor e é o que dá sentido à palavra "persistiu".
async function restart(page: Page): Promise<Page> {
  const context = page.context();
  await page.close();
  const fresh = await context.newPage();
  await fresh.goto(STUDIO_PATH);
  await expect(rows(fresh).first()).toBeVisible();
  return fresh;
}

test.describe("Studio — reordenação persistida no servidor", () => {
  test.beforeEach(async ({ request }) => {
    await resetStudio(request);
  });

  test("a ordem inicial do canvas e a ausência de escrita vêm do servidor", async ({
    page,
    request,
  }) => {
    await page.goto(STUDIO_PATH);

    await expectUiOrder(page, ["heading", "paragraph", "button"]);

    const state = await serverState(request);
    expect(state.order).toEqual(["heading", "paragraph", "button"]);
    // Abrir o canvas não pode gravar nada: se gravasse, a ordem exibida poderia
    // ser fruto de um POST e não da leitura do servidor.
    expect(state.visual_writes).toBe(0);
  });

  test("arrastar o último componente para a primeira posição persiste no servidor", async ({
    page,
    request,
  }) => {
    await page.goto(STUDIO_PATH);
    await expectUiOrder(page, ["heading", "paragraph", "button"]);

    await dragRow(page, 2, 0);
    await expectUiOrder(page, ["button", "heading", "paragraph"]);

    const state = await serverState(request);
    expect(state.order).toEqual(["button", "heading", "paragraph"]);
    expect(state.visual_writes).toBeGreaterThanOrEqual(1);
    // A ordem gravada é a do componente arrastado (`comp_cta`), não uma cópia
    // local: o servidor recebeu os ids na nova posição.
    expect(state.last_write_ids).toEqual([
      "comp_cta",
      "comp_hero",
      "comp_lead",
    ]);

    const fresh = await restart(page);
    await expectUiOrder(fresh, ["button", "heading", "paragraph"]);
  });

  test("mover pelo botão acessível persiste a mesma ordem no servidor", async ({
    page,
    request,
  }) => {
    await page.goto(STUDIO_PATH);
    await expectUiOrder(page, ["heading", "paragraph", "button"]);

    // Paridade de acessibilidade: a reordenação não pode depender do mouse.
    const firstRow = rows(page).first();
    await firstRow.hover();
    await firstRow.getByTitle("Mover para baixo").click();
    await expectUiOrder(page, ["paragraph", "heading", "button"]);

    const state = await serverState(request);
    expect(state.order).toEqual(["paragraph", "heading", "button"]);
    expect(state.visual_writes).toBeGreaterThanOrEqual(1);
    expect(state.last_write_ids).toEqual([
      "comp_lead",
      "comp_hero",
      "comp_cta",
    ]);

    const fresh = await restart(page);
    await expectUiOrder(fresh, ["paragraph", "heading", "button"]);
  });
});
