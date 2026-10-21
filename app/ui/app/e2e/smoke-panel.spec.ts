import { expect, test } from "@playwright/test";

/**
 * FASE 28.8 (E-1275): E2E do painel de SMOKE no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1274 provou o painel com `react-test-renderer` e o `agentFetch` MOCKADO. A E-1273 provou a rota
 * com `httptest`. **Os dois nunca falaram um com o outro.**
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * DUAS CLASSIFICACOES, E O SPEC DIZ QUAL CORRE
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` -- com `HADES_E2E_BACKEND_URL`. A rota existe no router de producao,
 *   mas **`HADES_ARTEFACTOS` nao esta' definido em CI**, por isso o servidor responde **412** -- e o
 *   spec exige que o painel mostre isso como erro accionavel, nao como um resultado.
 *
 *   `UI_FIXTURE` -- sem a variavel. O fixture devolve um smoke aprovado com aviso de sandbox.
 *
 * O QUE ESTE SPEC GARANTE NOS DOIS MODOS
 *
 *   1. o painel aparece e **diz que executa** antes de haver resultado;
 *   2. o botao esta' desactivado sem caminho -- *nao se executa o que nao foi indicado*;
 *   3. o pedido HTTP sai de verdade quando se carrega no botao;
 *   4. **nenhum resultado e' mostrado sem um pedido** -- o painel nao inventa um veredicto.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

const ROTA_DAS_CAPACIDADES = "/capabilities";

test.describe(`painel de smoke (${MODO})`, () => {
  test("o painel aparece e diz que EXECUTA", async ({ page }) => {
    await page.goto(ROTA_DAS_CAPACIDADES);
    const painel = page.getByTestId("smoke");
    await expect(painel).toBeVisible();
    await expect(painel.getByRole("heading", { name: "Executar um smoke" })).toBeVisible();
    // O aviso de que executa tem de estar ANTES do botao, nao depois do resultado.
    await expect(painel).toContainText(/executa/i);
    await expect(painel).toContainText(/nao escolhe a raiz/i);
  });

  test("o botao esta' desactivado sem caminho", async ({ page }) => {
    await page.goto(ROTA_DAS_CAPACIDADES);
    const botao = page.getByTestId("smoke").getByRole("button", { name: /executar smoke/i });
    await expect(botao).toBeDisabled();
    // Escrever um caminho activa-o.
    await page.getByTestId("smoke").locator("#smoke-caminho").fill("C:/artefactos/hades.exe");
    await expect(botao).toBeEnabled();
  });

  test("nenhum resultado e' mostrado sem um pedido", async ({ page }) => {
    await page.goto(ROTA_DAS_CAPACIDADES);
    await expect(page.getByTestId("smoke")).toBeVisible();
    // O painel nao pode inventar um veredicto antes de se carregar no botao.
    await expect(page.getByTestId("smoke-resultado")).toHaveCount(0);
  });

  test("o pedido HTTP sai de verdade quando se carrega no botao", async ({ page }) => {
    await page.goto(ROTA_DAS_CAPACIDADES);
    const painel = page.getByTestId("smoke");
    await expect(painel).toBeVisible();
    await painel.locator("#smoke-caminho").fill("C:/artefactos/hades.exe");
    await painel.locator("#smoke-argumentos").fill("--version");

    // O pedido tem de sair. Se o painel inventasse o resultado, a rota nunca seria chamada.
    const pedido = page.waitForRequest(
      (requisicao) =>
        requisicao.url().includes("/api/agent/v1/build/smoke") &&
        requisicao.method() === "POST",
    );
    await painel.getByRole("button", { name: /executar smoke/i }).click();
    await pedido;
  });

  test.describe("modo UI_FIXTURE", () => {
    test.skip(MODO !== "UI_FIXTURE", "corre so' com o fixture");

    test("mostra exit code, duracao e saida do resultado", async ({ page }) => {
      await page.goto(ROTA_DAS_CAPACIDADES);
      const painel = page.getByTestId("smoke");
      await expect(painel).toBeVisible();
      await painel.locator("#smoke-caminho").fill("C:/artefactos/hades.exe");
      await painel.locator("#smoke-argumentos").fill("--version");
      await painel.getByRole("button", { name: /executar smoke/i }).click();

      const resultado = page.getByTestId("smoke-resultado");
      await expect(resultado).toBeVisible();
      await expect(resultado).toContainText("0.40.1");
      await expect(resultado).toContainText("42 ms");
      await expect(page.getByTestId("smoke-codigo")).toHaveText("0");
    });

    test("mostra o aviso de sandbox do backend", async ({ page }) => {
      await page.goto(ROTA_DAS_CAPACIDADES);
      const painel = page.getByTestId("smoke");
      await painel.locator("#smoke-caminho").fill("C:/artefactos/hades.exe");
      await painel.locator("#smoke-argumentos").fill("--version");
      await painel.getByRole("button", { name: /executar smoke/i }).click();

      const aviso = page.getByTestId("smoke-aviso-sandbox");
      await expect(aviso).toBeVisible();
      await expect(aviso).toContainText(/NAO ha' sandbox/i);
    });

    // O PAINEL NAO PODE DIZER QUE O ARTEFACTO FUNCIONA.
    test("nao afirma que o artefacto funciona", async ({ page }) => {
      await page.goto(ROTA_DAS_CAPACIDADES);
      const painel = page.getByTestId("smoke");
      await painel.locator("#smoke-caminho").fill("C:/artefactos/hades.exe");
      await painel.locator("#smoke-argumentos").fill("--version");
      await painel.getByRole("button", { name: /executar smoke/i }).click();

      await expect(page.getByTestId("smoke-resultado")).toBeVisible();
      const texto = await painel.innerText();
      expect(texto).toContain("NAO provado que faz o que promete");
      expect(texto.toLowerCase()).not.toContain("binario seguro");
    });
  });

  test.describe("modo BACKEND_REAL_AGENT_RUNTIME", () => {
    test.skip(MODO !== "BACKEND_REAL_AGENT_RUNTIME", "corre so' com o backend real");

    // Sem `HADES_ARTEFACTOS` o servidor recusa com 412. O painel tem de mostrar isso como erro
    // accionavel -- nao como um resultado inventado.
    test("raiz nao configurada produz erro accionavel, nao um resultado", async ({ page }) => {
      await page.goto(ROTA_DAS_CAPACIDADES);
      const painel = page.getByTestId("smoke");
      await expect(painel).toBeVisible();
      await painel.locator("#smoke-caminho").fill("/artefactos/hades");
      await painel.locator("#smoke-argumentos").fill("--version");
      await painel.getByRole("button", { name: /executar smoke/i }).click();

      const erro = page.getByTestId("smoke-erro");
      await expect(erro).toBeVisible();
      await expect(erro).toContainText(/HADES_ARTEFACTOS/);
      // E nenhum resultado pode ter aparecido.
      await expect(page.getByTestId("smoke-resultado")).toHaveCount(0);
    });
  });
});
