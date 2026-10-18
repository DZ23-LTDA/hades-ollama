import { expect, test } from "@playwright/test";

/**
 * FASE 28.8 (E-1272): E2E do painel de CREDENCIAIS no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1271 provou o painel com `react-test-renderer` e o `agentFetch` MOCKADO. A E-1270 provou a rota
 * com `httptest`. **Os dois nunca falaram um com o outro.**
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * DUAS CLASSIFICACOES, E O SPEC DIZ QUAL CORRE
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` -- com `HADES_E2E_BACKEND_URL`. O pedido atravessa o proxy `/api` e
 *   chega ao router de producao. Nesse modo o kit **nao esta' configurado** no servidor de CI (a
 *   variavel `HADES_KIT_CREDENCIAIS` nao existe), por isso o painel mostra "nao configurado" -- e o
 *   spec exige exactamente essa mensagem, nao um erro.
 *
 *   `UI_FIXTURE` -- sem a variavel. O fixture serve um kit com um certificado publico e um segredo.
 *
 * O QUE ESTE SPEC GARANTE NOS DOIS MODOS
 *
 *   1. o painel aparece com titulo e nome acessivel;
 *   2. o conteudo veio do HTTP, nao de constantes do componente;
 *   3. **"nao configurado" nao e' erro** -- nao ha' alerta vermelho nesse caso;
 *   4. o aviso de fronteira esta' VISIVEL no DOM;
 *   5. **nenhum botao de accao existe** no painel: a rota e' GET e o painel e' somente leitura.
 *
 * O QUE ELE NAO FAZ: fingir que um modo e' o outro. Cada assercao tem a forma que o modo permite.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

const ROTA_DAS_CAPACIDADES = "/capabilities";

test.describe(`painel de credenciais (${MODO})`, () => {
  test("o painel aparece com titulo e nome acessivel", async ({ page }) => {
    await page.goto(ROTA_DAS_CAPACIDADES);
    const painel = page.getByTestId("credenciais");
    await expect(painel).toBeVisible();
    // Regiao com nome: o `<section>` tem `aria-labelledby` a apontar para o titulo.
    await expect(painel.getByRole("heading", { name: "Credenciais e certificados" })).toBeVisible();
  });

  test("o conteudo veio do HTTP, nao do componente", async ({ page }) => {
    // O pedido tem de sair. Se o painel inventasse o conteudo, a rota nunca seria chamada.
    const pedido = page.waitForRequest(
      (requisicao) =>
        requisicao.url().includes("/api/agent/v1/credentials/status") &&
        requisicao.method() === "GET",
    );
    await page.goto(ROTA_DAS_CAPACIDADES);
    await pedido;
    await expect(page.getByTestId("credenciais")).toBeVisible();
  });

  test("o aviso de fronteira esta' visivel no DOM", async ({ page }) => {
    await page.goto(ROTA_DAS_CAPACIDADES);
    await expect(page.getByTestId("credenciais")).toBeVisible();
    // Em qualquer dos modos o backend envia um `aviso_de_fronteira`; quando nao configurado, o painel
    // mostra o `aviso` no lugar. O que NAO pode acontecer e' nao haver mensagem nenhuma.
    const painel = page.getByTestId("credenciais");
    // Em qualquer dos modos ha' uma mensagem de fronteira: ou o `aviso_de_fronteira` (kit
    // configurado) ou o `aviso` (nao configurado). `toContainText` espera pelo texto, ao contrario
    // de um `innerText` lido uma vez.
    await expect(painel).toContainText(/nao sao abertos|nao configurado/i);
  });

  test("nenhum botao de accao existe no painel", async ({ page }) => {
    await page.goto(ROTA_DAS_CAPACIDADES);
    await expect(page.getByTestId("credenciais")).toBeVisible();
    // A rota e' GET e o painel e' somente leitura. Um botao aqui seria uma accao sem implementacao.
    //
    //	*Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
    await expect(page.getByTestId("credenciais").getByRole("button")).toHaveCount(0);
  });

  test.describe("modo UI_FIXTURE", () => {
    test.skip(MODO !== "UI_FIXTURE", "corre so' com o fixture");

    test("mostra o certificado publico e o segredo NAO lido", async ({ page }) => {
      await page.goto(ROTA_DAS_CAPACIDADES);
      const painel = page.getByTestId("credenciais");
      await expect(painel).toBeVisible();

      // O certificado publico aparece com sujeito e validade -- o fixture os declarou.
      await expect(painel).toContainText("apple/distribution.cer");
      await expect(painel).toContainText("Apple Distribution: leandro prado (K8G36CUP3S)");
      await expect(painel).toContainText("Certificado");

      // O segredo aparece, mas com a frase que diz que NAO foi lido.
      await expect(painel).toContainText("apple/hades-apple-distribution.p12");
      await expect(painel).toContainText("NAO lido");

      // E o painel NAO pode afirmar que o segredo foi verificado.
      const texto = await painel.innerText();
      expect(texto.toLowerCase()).not.toContain("verificado com sucesso");
    });

    test("mostra o resumo do inventario", async ({ page }) => {
      await page.goto(ROTA_DAS_CAPACIDADES);
      const resumo = page.getByTestId("credenciais-resumo");
      await expect(resumo).toBeVisible();
      await expect(resumo).toContainText("2 ficheiro(s)");
      await expect(resumo).toContainText("1 lido(s)");
      await expect(resumo).toContainText("1 nao lido(s)");
    });

    test("nao mostra alerta quando o kit esta' configurado", async ({ page }) => {
      await page.goto(ROTA_DAS_CAPACIDADES);
      await expect(page.getByTestId("credenciais")).toBeVisible();
      await expect(page.getByTestId("credenciais-erro")).toHaveCount(0);
    });
  });

  test.describe("modo BACKEND_REAL_AGENT_RUNTIME", () => {
    test.skip(MODO !== "BACKEND_REAL_AGENT_RUNTIME", "corre so' com o backend real");

    test("kit nao configurado nao e' erro", async ({ page }) => {
      // No servidor real de CI a variavel `HADES_KIT_CREDENCIAIS` nao existe. O painel tem de mostrar
      // a mensagem propria -- nao um alerta.
      //
      //	*"Nao configurado" e' um estado do produto, nao uma falha.*
      await page.goto(ROTA_DAS_CAPACIDADES);
      const painel = page.getByTestId("credenciais");
      await expect(painel).toBeVisible();
      await expect(page.getByTestId("credenciais-erro")).toHaveCount(0);
      await expect(page.getByTestId("credenciais-nao-configurado")).toBeVisible();
    });
  });
});
