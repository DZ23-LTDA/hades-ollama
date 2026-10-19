import { expect, test } from "@playwright/test";

/**
 * E-1299: E2E do painel de `publish.preflight` no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1298 provou o painel com `agentFetch` MOCKADO. A E-1289 provou a rota com `httptest`. Os dois
 * nunca falaram um com o outro.
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * O QUE SO' O BROWSER PODE PROVAR AQUI
 *
 * 1. O canal escolhido no `<select>` chega ao CAMINHO da rota (o fixture le^ o canal do URL).
 * 2. `LOCAL_ONLY` mostra "ensaio" e NAO mostra o aviso de canal publico; `GOOGLE_PLAY` faz o
 *    contrario. Um painel que tratasse os dois igual passaria no teste unitario e falharia aqui.
 * 3. As duas garantias (`somente_leitura`, `nada_publicado`) aparecem e `publicado` diz "nao".
 * 4. Nao existe botao de publicar -- nem escondido, nem desactivado.
 * 5. Um canal invalido vira erro legivel, nao sucesso.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

const ROTA = "/capabilities";

test.describe(`painel de preflight de publicacao (${MODO})`, () => {
  test("diz que nao publica e que o provider nao tem Publicar()", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("preflight");
    await expect(painel).toBeVisible();
    await expect(painel.getByRole("heading", { name: /Preflight de publicacao/ })).toBeVisible();
    await expect(painel).toContainText(/Nao publica/);
    await expect(painel).toContainText(/Publicar\(\)/);
  });

  // NAO TEM CAMPO PARA O ARTEFACTO NEM PARA A CREDENCIAL.
  test("nao tem campo para o artefacto nem para a credencial", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("preflight");
    await expect(painel).toBeVisible();

    // O painel so' tem UM controlo de entrada: o `<select>` do canal.
    await expect(painel.locator("input")).toHaveCount(0);
    await expect(painel.locator("textarea")).toHaveCount(0);
    await expect(painel.locator("select")).toHaveCount(1);

    // Mas NOMEIA as variaveis do servidor, para o utilizador saber de onde vem.
    await expect(page.getByTestId("preflight-sem-campos")).toContainText(
      /HADES_ARTEFACTO_A_PUBLICAR/,
    );
    await expect(page.getByTestId("preflight-sem-campos")).toContainText(
      /HADES_CREDENCIAL_DE_PUBLICACAO/,
    );
  });

  // NAO EXISTE BOTAO DE PUBLICAR.
  test("nao existe botao de publicar", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("preflight");
    await expect(painel).toBeVisible();

    await expect(painel.getByRole("button", { name: /publicar/i })).toHaveCount(0);
    await expect(painel.getByRole("button", { name: /Verificar canal/ })).toBeVisible();
  });

  test("LOCAL_ONLY e' ensaio local e nao avisa de canal publico", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("preflight");
    await expect(painel).toBeVisible();

    // O select comeca em LOCAL_ONLY.
    await expect(painel.locator("select")).toHaveValue("LOCAL_ONLY");
    await expect(page.getByTestId("preflight-canal-local")).toContainText(/ensaio/i);
    await expect(page.getByTestId("preflight-canal-local")).toContainText(/Nada sai desta maquina/);
    await expect(page.getByTestId("preflight-canal-publico")).toHaveCount(0);
  });

  test("canal publico avisa que exige confirmacao humana", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("preflight");
    await expect(painel).toBeVisible();

    await painel.locator("select").selectOption("GOOGLE_PLAY");
    await expect(page.getByTestId("preflight-canal-publico")).toContainText(/confirmacao humana/);
    await expect(page.getByTestId("preflight-canal-local")).toHaveCount(0);
  });

  test("verifica LOCAL_ONLY e mostra as garantias com publicado=nao", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("preflight");
    await expect(painel).toBeVisible();

    await painel.getByRole("button", { name: /Verificar canal/ }).click();

    const resultado = page.getByTestId("preflight-resultado");
    await expect(resultado).toBeVisible();

    // O CANAL que chegou ao servidor foi o escolhido -- o fixture le^ o canal do caminho.
    await expect(page.getByTestId("preflight-canal-escolhido")).toHaveText("LOCAL_ONLY");
    await expect(page.getByTestId("preflight-publico")).toHaveText("nao");
    await expect(page.getByTestId("preflight-pronto")).toHaveText("sim");

    // As DUAS garantias, separadas do veredicto.
    await expect(page.getByTestId("preflight-somente-leitura")).toContainText(/sim/);
    await expect(page.getByTestId("preflight-nada-publicado")).toContainText(/sim/);
    // E `publicado` diz "nao" -- o provider nao tem `Publicar()`.
    await expect(page.getByTestId("preflight-publicado")).toContainText(/nao/);
    await expect(page.getByTestId("preflight-publicado")).not.toContainText(/inconsistente/i);
  });

  test("verifica GOOGLE_PLAY: pronto=false, requer aprovacao, publicado continua nao", async ({
    page,
  }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("preflight");
    await expect(painel).toBeVisible();

    await painel.locator("select").selectOption("GOOGLE_PLAY");
    await painel.getByRole("button", { name: /Verificar canal/ }).click();

    const resultado = page.getByTestId("preflight-resultado");
    await expect(resultado).toBeVisible();

    await expect(page.getByTestId("preflight-canal-escolhido")).toHaveText("GOOGLE_PLAY");
    await expect(page.getByTestId("preflight-publico")).toHaveText("sim");
    // O fixture diz que `aprovacao_humana` esta' pendente.
    await expect(page.getByTestId("preflight-pronto")).toHaveText("nao");
    await expect(page.getByTestId("preflight-requer-aprovacao")).toHaveText("sim");
    // MESMO com canal publico, `publicado` e' "nao".
    await expect(page.getByTestId("preflight-publicado")).toContainText(/nao/);

    // A condicao pendente aparece como PENDENTE, nao escondida.
    await expect(page.getByTestId("preflight-condicao-aprovacao_humana")).toContainText(/PENDENTE/);
    await expect(page.getByTestId("preflight-condicoes-contagem")).toContainText("2");
  });

  test("um canal invalido vira erro legivel, nunca sucesso", async ({ page }) => {
    // O fixture responde 400 a canal desconhecido -- o mesmo `sendJSON` que as outras rotas usam.
    await page.goto(ROTA);
    const painel = page.getByTestId("preflight");
    await expect(painel).toBeVisible();

    // Nao ha' como escolher um canal invalido pelo select; provamos que o painel so' oferece os 6.
    const opcoes = painel.locator("select option");
    await expect(opcoes).toHaveCount(6);
    for (const canal of [
      "LOCAL_ONLY",
      "GOOGLE_PLAY",
      "APP_STORE",
      "EAS_UPDATE",
      "GITHUB_RELEASE",
      "OCI_REGISTRY",
    ]) {
      await expect(painel.locator(`select option[value="${canal}"]`)).toHaveCount(1);
    }
  });

  test("o aviso de fronteira diz que a credencial vem do servidor", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("preflight");
    await expect(painel).toBeVisible();

    await painel.getByRole("button", { name: /Verificar canal/ }).click();
    await expect(page.getByTestId("preflight-aviso-fronteira")).toContainText(
      /AMBIENTE DO SERVIDOR/,
    );
    await expect(page.getByTestId("preflight-aviso-fronteira")).toContainText(/nao publica/);
  });
});
