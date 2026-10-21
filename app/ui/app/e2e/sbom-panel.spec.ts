import { expect, test } from "@playwright/test";

/**
 * FASE 28.8 (E-1283): E2E do painel do SBOM no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1282 provou o painel com `react-test-renderer` e o `agentFetch` MOCKADO. A E-1281 provou a rota
 * com `httptest`. **Os dois nunca falaram um com o outro.**
 *
 * DUAS CLASSIFICACOES
 *
 *   `UI_FIXTURE`                  o fixture devolve um inventario com tipos REAIS (`MODULO_GO`,
 *                                 `PACOTE_NPM`) e `nao_produzido`
 *
 *   `BACKEND_REAL_AGENT_RUNTIME`  atravessa o proxy ate' ao router de producao. **Sem
 *                                 `HADES_RAIZ_SBOM`** o servidor responde **412**, e o spec exige que o
 *                                 painel mostre isso como erro accionavel nomeando a variavel.
 *
 * O QUE ESTE SPEC GARANTE NOS DOIS MODOS
 *
 *   1. o painel **diz que o SBOM e' minimo** antes de haver resultado;
 *   2. **NAO TEM CAMPO DE RAIZ** -- a raiz e' do servidor;
 *   3. **NAO EXISTE BOTAO DE ASSINAR NEM DE ATESTAR** -- o spec conta os botoes;
 *   4. **o corpo do pedido e' `{}`** -- inspeccionado com `postDataJSON`;
 *   5. **nenhum inventario e' mostrado sem um pedido**.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

const ROTA = "/capabilities";

test.describe(`painel do SBOM (${MODO})`, () => {
  test("diz que o SBOM e' minimo antes de haver resultado", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("sbom");
    await expect(painel).toBeVisible();
    await expect(painel.getByRole("heading", { name: /Inventario de software/ })).toBeVisible();
    await expect(painel).toContainText(/minimo/i);
    await expect(painel).toContainText(/Nao assina/i);
    await expect(painel).toContainText(/nunca deste painel/i);
  });

  // NAO TEM CAMPO DE RAIZ.
  test("nao tem campo para a raiz do projecto", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("sbom");
    await expect(painel).toBeVisible();
    // Nenhum input: o painel nao escolhe a raiz.
    await expect(painel.locator("input")).toHaveCount(0);
  });

  // NAO EXISTE BOTAO DE ASSINAR NEM DE ATESTAR.
  test("nao existe botao de assinar nem de atestar", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("sbom");
    await expect(painel).toBeVisible();
    const botoes = painel.getByRole("button");
    await expect(botoes).toHaveCount(1);
    const textos = await botoes.allInnerTexts();
    for (const texto of textos) {
      const minusculas = texto.toLowerCase();
      expect(minusculas).not.toContain("assinar");
      expect(minusculas).not.toContain("atestar");
      expect(minusculas).toContain("gerar");
    }
  });

  test("nenhum inventario e' mostrado sem um pedido", async ({ page }) => {
    await page.goto(ROTA);
    await expect(page.getByTestId("sbom")).toBeVisible();
    await expect(page.getByTestId("sbom-relatorio")).toHaveCount(0);
  });

  // O CORPO E' VAZIO -- A RAIZ NAO VAI NO PEDIDO.
  test("o corpo do pedido e' vazio -- sem raiz", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("sbom");

    const pedido = page.waitForRequest(
      (requisicao) =>
        requisicao.url().includes("/api/agent/v1/sbom/generate") &&
        requisicao.method() === "POST",
    );
    await painel.getByRole("button", { name: /gerar inventario/i }).click();
    const capturado = await pedido;

    const corpo = capturado.postDataJSON() as Record<string, unknown>;
    expect(Object.keys(corpo)).toEqual([]);
    // E nada que cheire a caminho.
    const texto = JSON.stringify(corpo).toLowerCase();
    for (const proibido of ["raiz", "caminho", "path", "directorio", "root"]) {
      expect(texto).not.toContain(proibido);
    }
  });

  test.describe("modo UI_FIXTURE", () => {
    test.skip(MODO !== "UI_FIXTURE", "corre so' com o fixture");

    test("mostra os componentes com os tipos REAIS do backend", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("sbom");
      await painel.getByRole("button", { name: /gerar inventario/i }).click();

      await expect(page.getByTestId("sbom-relatorio")).toBeVisible();
      await expect(page.getByTestId("sbom-total")).toHaveText("3");
      await expect(page.getByTestId("sbom-sem-licenca")).toHaveText("2");

      const componentes = page.getByTestId("sbom-componentes");
      await expect(componentes).toContainText("github.com/gin-gonic/gin");
      await expect(componentes).toContainText("left-pad");
      // Os tipos vem do BACKEND, nao de valores inventados.
      await expect(componentes).toContainText("modulo Go");
      await expect(componentes).toContainText("pacote npm");
    });

    test("distingue componentes sem licenca declarada", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("sbom");
      await painel.getByRole("button", { name: /gerar inventario/i }).click();

      const componentes = page.getByTestId("sbom-componentes");
      await expect(componentes).toContainText("sem licenca declarada");
      // E o que tem licenca mostra-a.
      await expect(componentes).toContainText("MIT");
    });

    test("mostra nao_produzido por extenso", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("sbom");
      await painel.getByRole("button", { name: /gerar inventario/i }).click();

      const lacunas = page.getByTestId("sbom-nao-produzido");
      await expect(lacunas).toBeVisible();
      await expect(lacunas).toContainText("dependencias transitivas");
      await expect(lacunas).toContainText("THIRD_PARTY_NOTICES");
      await expect(lacunas).toContainText("provenance e attestation");
    });

    test("assinado e provenance sao nao, e o aviso vem por inteiro", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("sbom");
      await painel.getByRole("button", { name: /gerar inventario/i }).click();

      await expect(page.getByTestId("sbom-assinado")).toHaveText("nao");
      await expect(page.getByTestId("sbom-provenance")).toHaveText("nao");
      const aviso = page.getByTestId("sbom-aviso-fronteira");
      await expect(aviso).toContainText(/SBOM MINIMO/);
      await expect(aviso).toContainText(/sempre false/);
      await expect(aviso).toContainText(/nunca do pedido/);
      // E a prova nao pode afirmar que e' completo.
      await expect(page.getByTestId("sbom-prova")).toContainText(/NAO inclui/);
    });
  });

  test.describe("modo BACKEND_REAL_AGENT_RUNTIME", () => {
    test.skip(MODO !== "BACKEND_REAL_AGENT_RUNTIME", "corre so' com o backend real");

    // Sem `HADES_RAIZ_SBOM` o servidor recusa com 412. O painel tem de mostrar isso como erro
    // accionavel -- nao como um inventario vazio.
    test("raiz nao configurada produz erro accionavel, nao um inventario", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("sbom");
      await expect(painel).toBeVisible();
      await painel.getByRole("button", { name: /gerar inventario/i }).click();

      const erro = page.getByTestId("sbom-erro");
      await expect(erro).toBeVisible();
      await expect(erro).toContainText(/HADES_RAIZ_SBOM/);
      await expect(page.getByTestId("sbom-relatorio")).toHaveCount(0);
    });
  });
});
