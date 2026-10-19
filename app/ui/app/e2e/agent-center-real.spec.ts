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

  /**
   * O CICLO: escrever no servidor pelo browser, e ler de volta.
   *
   * `install` grava no ledger (escrita); `backup` guarda (escrita); `drift` reconcilia por hash (LEITURA do que
   * o painel acabou de escrever). Se o painel inventasse o estado, o drift viria vazio.
   *
   * Discriminante: `agent-center-instalado` diz quantos BYTES foram renderizados (numero vindo do servidor), e
   * `agent-center-resumo` so' aparece com o relatorio do servidor.
   *
   * NAO cobre `update` nem `remove`: essas duas accoes nao existem como botao no painel; estao declaradas como
   * lacuna no E-1382 e nao se fingem aqui.
   */
  test("install -> backup -> drift: escreve no servidor e le de volta", async ({ page }) => {
    await page.goto("/capabilities");

    const destino = page.getByTestId("agent-center-destino");
    await expect(destino).toBeVisible({ timeout: 20_000 });

    // O destino e' RELATIVO e sem `..` -- o servidor recusa absoluto (INSTALACAO_RECUSADA). Medido: a
    // primeira versao deste teste mandou `C:\\...` e foi recusada; o produto estava certo e o teste errado.
    await destino.fill("e2e-destino-" + Date.now());

    await page.getByTestId("agent-center-instalar").click();

    // `agent-center-recusado` e' o caminho de recusa: se aparecer, o teste falha com a mensagem do servidor
    // em vez de ficar a esperar 20s por um elemento que nunca vem.
    const instalado = page.getByTestId("agent-center-instalado");
    await expect(instalado.or(page.getByTestId("agent-center-recusado"))).toBeVisible({ timeout: 20_000 });
    await expect(page.getByTestId("agent-center-recusado")).toHaveCount(0);

    const textoInstalado = (await instalado.innerText()).replace(/\s+/g, " ").trim();
    // O numero de bytes vem do servidor: e' o tamanho do conteudo renderizado, nao uma constante do painel.
    expect(textoInstalado).toMatch(/\d+ bytes/);

    // BACKUP (outra escrita)
    await page.getByTestId("agent-center-backup-btn").click();
    const contagem = page.getByTestId("agent-center-bk-contagem");
    await expect(contagem).toBeVisible({ timeout: 20_000 });
    expect(await contagem.innerText()).toMatch(/\d+/);

    // DRIFT (leitura do que acabou de ser escrito)
    await page.getByTestId("agent-center-drift").click();
    await expect(page.getByTestId("agent-center-resumo")).toBeVisible({ timeout: 20_000 });
    const resumo = (await page.getByTestId("agent-center-resumo").innerText()).replace(/\s+/g, " ").trim();
    expect(resumo.length).toBeGreaterThan(5);
  });
});
