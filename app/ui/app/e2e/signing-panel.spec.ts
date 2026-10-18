import { expect, test } from "@playwright/test";

/**
 * FASE 28.8 (E-1280): E2E do painel de ASSINATURA no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1279 provou o painel com `react-test-renderer` e o `agentFetch` MOCKADO. A rota `sign/plan` foi
 * provada com `httptest`. **Os dois nunca falaram um com o outro.**
 *
 * DUAS CLASSIFICACOES
 *
 *   `UI_FIXTURE`                      o fixture devolve uma avaliacao `APTA` ou `SEM_EKU_DE_ASSINATURA`
 *
 *   `BACKEND_REAL_AGENT_RUNTIME`      atravessa o proxy ate' ao router de producao. Um caminho de
 *                                    certificado inexistente produz **erro accionavel**, e nenhum
 *                                    plano aparece -- o painel nao inventa uma avaliacao.
 *
 * O QUE ESTE SPEC GARANTE NOS DOIS MODOS
 *
 *   1. o painel **diz que so' avalia** antes de haver resultado;
 *   2. **NAO EXISTE CAMPO DE SENHA** e **NAO EXISTE BOTAO DE ASSINAR** -- o spec conta os `input` e
 *      os `button` do painel;
 *   3. **o corpo do pedido nao tem chave privada nem senha** -- `postDataJSON` inspeccionado;
 *   4. **nenhuma avaliacao e' mostrada sem um pedido**.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

const ROTA = "/capabilities";

test.describe(`painel de assinatura (${MODO})`, () => {
  test("diz que so' avalia antes de haver resultado", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("assinatura");
    await expect(painel).toBeVisible();
    await expect(painel.getByRole("heading", { name: "Identidade de assinatura" })).toBeVisible();
    await expect(painel).toContainText(/apenas/);
    await expect(painel).toContainText(/Nao assina/);
  });

  // NAO EXISTE CAMPO DE SENHA.
  test("nao existe campo de senha no painel", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("assinatura");
    await expect(painel).toBeVisible();
    const campos = painel.locator("input");
    // Dois campos: certificado e rotulo. Nenhum outro.
    await expect(campos).toHaveCount(2);
    await expect(painel.locator('input[type="password"]')).toHaveCount(0);
    await expect(painel).not.toContainText(/senha/i);
    await expect(painel).not.toContainText(/chave privada:/i);
  });

  // NAO EXISTE BOTAO DE ASSINAR.
  test("nao existe botao de assinar", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("assinatura");
    await expect(painel).toBeVisible();
    const botoes = painel.getByRole("button");
    await expect(botoes).toHaveCount(1);
    const textos = await botoes.allInnerTexts();
    for (const texto of textos) {
      const minusculas = texto.toLowerCase();
      expect(minusculas).not.toContain("assinar");
      expect(minusculas).toContain("avaliar");
    }
  });

  test("o botao esta' desactivado sem certificado", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("assinatura");
    const botao = painel.getByRole("button", { name: /avaliar identidade/i });
    await expect(botao).toBeDisabled();
    await painel.locator("#assinatura-certificado").fill("C:/certificados/distribution.cer");
    await expect(botao).toBeEnabled();
  });

  test("nenhuma avaliacao e' mostrada sem um pedido", async ({ page }) => {
    await page.goto(ROTA);
    await expect(page.getByTestId("assinatura")).toBeVisible();
    await expect(page.getByTestId("assinatura-plano")).toHaveCount(0);
  });

  // O CORPO NAO TEM SEGREDO.
  test("o corpo do pedido nao tem chave privada nem senha", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("assinatura");

    const pedido = page.waitForRequest(
      (requisicao) =>
        requisicao.url().includes("/api/agent/v1/sign/plan") && requisicao.method() === "POST",
    );
    await painel.locator("#assinatura-certificado").fill("C:/certificados/distribution.cer");
    await painel.getByRole("button", { name: /avaliar identidade/i }).click();
    const capturado = await pedido;

    const corpo = capturado.postDataJSON() as {
      tipo: string;
      identidade: Record<string, unknown>;
    };
    expect(Object.keys(corpo).sort()).toEqual(["identidade", "tipo"]);
    expect(Object.keys(corpo.identidade).sort()).toEqual([
      "ambiente",
      "caminho_do_certificado",
      "rotulo",
    ]);
    // Nenhuma chave que cheire a segredo.
    const texto = JSON.stringify(corpo).toLowerCase();
    for (const proibido of ["senha", "password", "chave_privada", "private_key", "pfx", "p12"]) {
      expect(texto).not.toContain(proibido);
    }
  });

  test.describe("modo UI_FIXTURE", () => {
    test.skip(MODO !== "UI_FIXTURE", "corre so' com o fixture");

    test("uma identidade APTA mostra o sujeito, o emissor e ap ta=sim", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("assinatura");
      await painel.locator("#assinatura-certificado").fill("C:/certificados/distribution.cer");
      await painel.getByRole("button", { name: /avaliar identidade/i }).click();

      await expect(page.getByTestId("assinatura-plano")).toBeVisible();
      await expect(page.getByTestId("assinatura-apta")).toHaveText("sim");
      await expect(page.getByTestId("assinatura-aptidao")).toContainText(
        "a identidade serve para este tipo",
      );
      await expect(page.getByTestId("assinatura-plano")).toContainText("CN=DZ23 LTDA");
      await expect(page.getByTestId("assinatura-plano")).toContainText("CN=DZ23 Code Signing CA");
      // E diz que NAO assinou.
      await expect(page.getByTestId("assinatura-nada-assinado")).toHaveText("nao");
    });

    // UM CERTIFICADO SEM EKU NAO PODE SER TRATADO COMO APTO.
    test("sem EKU mostra o OID e aprovacao=nao", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("assinatura");
      await painel.locator("#assinatura-certificado").fill("C:/certificados/sem-eku.cer");
      await painel.getByRole("button", { name: /avaliar identidade/i }).click();

      await expect(page.getByTestId("assinatura-apta")).toHaveText("nao");
      await expect(page.getByTestId("assinatura-aptidao")).toContainText(
        "falta o EKU de assinatura de codigo",
      );
      await expect(page.getByTestId("assinatura-aptidao")).toContainText("1.3.6.1.5.5.7.3.3");
      await expect(page.getByTestId("assinatura-prova")).toContainText("nao assinavel");
    });

    test("mostra os alvos suportados e o aviso de fronteira", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("assinatura");
      await painel.locator("#assinatura-certificado").fill("C:/certificados/distribution.cer");
      await painel.getByRole("button", { name: /avaliar identidade/i }).click();

      await expect(page.getByTestId("assinatura-alvos")).toContainText("Authenticode (Windows)");
      await expect(page.getByTestId("assinatura-alvos")).toContainText(
        "Apple Developer ID (macOS)",
      );
      const aviso = page.getByTestId("assinatura-aviso-fronteira");
      await expect(aviso).toContainText(/nao le' a chave privada/);
      await expect(aviso).toContainText(/NAO sao validadas aqui/);
    });
  });

  test.describe("modo BACKEND_REAL_AGENT_RUNTIME", () => {
    test.skip(MODO !== "BACKEND_REAL_AGENT_RUNTIME", "corre so' com o backend real");

    // Um certificado que nao existe produz erro accionavel -- nunca uma avaliacao inventada.
    test("certificado inexistente produz erro, nao uma avaliacao", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("assinatura");
      await expect(painel).toBeVisible();
      await painel
        .locator("#assinatura-certificado")
        .fill("/caminho/que/nao/existe/nao-existe.cer");
      await painel.getByRole("button", { name: /avaliar identidade/i }).click();

      const erro = page.getByTestId("assinatura-erro");
      await expect(erro).toBeVisible();
      await expect(page.getByTestId("assinatura-plano")).toHaveCount(0);
    });
  });
});
