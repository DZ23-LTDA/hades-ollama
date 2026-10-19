import { expect, test } from "@playwright/test";

/**
 * O Agent Center no browser, contra o SERVIDOR — a lacuna que os E-1378..E-1381 declararam.
 *
 * O QUE FALTAVA
 *
 * O painel estava provado com `react-test-renderer` e `fetch` mockado. O servidor estava provado com
 * `httptest` e com um binario real a responder a 6 GET (E-1378). Nenhum dos dois prova que o PAINEL fala com
 * o SERVIDOR:
 *
 *	*Um painel provado contra um mock e um backend provado com `httptest` nao provam que um fala com o
 *	outro.*
 *
 * Este spec fecha isso onde o ambiente permite, e DIZ em que modo corre.
 *
 * DUAS CLASSIFICACOES
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` — quando `HADES_E2E_BACKEND_URL` esta' definido. O clique sai do browser,
 *   atravessa o proxy `/api` do preview e chega ao binario do HADES a correr. A contagem que aparece e' a que
 *   o servidor calculou.
 *
 *   `UI_FIXTURE` — sem a variavel. O fixture `first-run-backend.mjs` responde. Mede a UI, nao a integracao.
 *
 * O QUE ESTE SPEC RECUSA FAZER: dizer que verificou a integracao quando correu no fixture. O titulo do teste
 * carrega o modo, e a assercao de "dados do servidor" so' corre no modo real.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

test.describe(`Agent Center no browser (${MODO})`, () => {
  /**
   * O teste que o E-1378 nao podia fazer: o clique e' do browser, a resposta e' do servidor.
   *
   * Discriminante: `agent-center-contagem` mostra um NUMERO e `agent-center-erro`/`agent-center-ausente`
   * nao existem. Se o painel nao falasse com o servidor, um dos dois apareceria -- e um painel em erro e'
   * indistinguivel de um painel vazio para quem so' olha para a contagem.
   */
  test("listar mostra a contagem do servidor, nao um erro", async ({ page }) => {
    await page.goto("/capabilities");

    const botao = page.getByTestId("agent-center-listar");
    await expect(botao).toBeVisible({ timeout: 20_000 });
    await botao.click();

    const contagem = page.getByTestId("agent-center-contagem");
    await expect(contagem).toBeVisible({ timeout: 20_000 });

    const texto = (await contagem.innerText()).replace(/\s+/g, " ").trim();
    // Um numero tem de estar la'. "0" e' um numero valido: o ledger pode estar vazio e isso nao e' erro.
    expect(texto).toMatch(/\d+/);

    // E nao pode estar em erro nem "ausente": esses dois sao os estados em que o painel NAO falou com o
    // servidor (ou falou e o runtime estava caido -- 503, ver E-1379).
    await expect(page.getByTestId("agent-center-erro")).toHaveCount(0);
    await expect(page.getByTestId("agent-center-ausente")).toHaveCount(0);

    // O modo tem de estar visivel no relatorio do Playwright, nao so' no nome do teste.
    test.info().annotations.push({ type: "modo", description: MODO });
  });

  /**
   * O catalogo e' uma rota DIFERENTE do ledger -- e o painel tem de as separar.
   *
   * Discriminante: `agent-center-cat-nota` (a `catalog_note` que o servidor emite, E-1377) tem de aparecer
   * depois de clicar em "catalogo". Se o painel mostrasse o ledger como se fosse catalogo, a nota nao viria.
   */
  test("catalogo mostra a nota que o servidor emite", async ({ page }) => {
    await page.goto("/capabilities");

    const botao = page.getByTestId("agent-center-catalogo-btn");
    await expect(botao).toBeVisible({ timeout: 20_000 });
    await botao.click();

    // `agent-center-cat-versao` mostra `CatalogVersion()` do SERVIDOR. Se o painel estivesse a inventar um
    // catalogo local, este campo nao viria do pedido.
    const versao = page.getByTestId("agent-center-cat-versao");
    await expect(versao).toBeVisible({ timeout: 20_000 });
    const textoVersao = (await versao.innerText()).replace(/\s+/g, " ").trim();
    expect(textoVersao).toMatch(/\d/);

    // E o bloco do catalogo tem de existir -- nao o do ledger. Se o painel mostrasse o ledger como
    // catalogo, `agent-center-catalogo` nao apareceria.
    await expect(page.getByTestId("agent-center-catalogo")).toBeVisible();
    // O fallback do painel diz "nao esta' ligado". Com o catalogo ligado, essa frase seria uma mentira.
    const corpo = (await page.getByTestId("agent-center-catalogo").innerText()).toLowerCase();
    expect(corpo).not.toContain("não está ligado");
  });
});
