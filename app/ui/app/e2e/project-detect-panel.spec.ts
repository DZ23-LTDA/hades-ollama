import { expect, test } from "@playwright/test";

/**
 * FASE 28.8 (E-1288): E2E do painel de deteccao de projecto no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1287 provou o painel com `agentFetch` MOCKADO. A E-1286 provou a rota com `httptest`. **Os dois
 * nunca falaram um com o outro.**
 *
 * O QUE SO' O BROWSER PODE PROVAR AQUI
 *
 * O fixture devolve as stacks `go` e `node` e os gestores `go-modules` e `npm` -- os valores REAIS do
 * backend. O spec exige os rotulos traduzidos: "Go", "Node", "Go modules", "npm".
 *
 * Se o painel inventasse `golang` ou `nodejs` no `switch`, o teste unitario passaria (usaria o mesmo
 * valor inventado) e o E2E falharia.
 *
 *	*Um teste que inventa o valor que o backend devolve nunca falha por causa do backend.*
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

const ROTA = "/capabilities";

test.describe(`painel de deteccao de projecto (${MODO})`, () => {
  test("diz que a rota so' le' e que os alvos sao sugestao", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("deteccao");
    await expect(painel).toBeVisible();
    await expect(painel.getByRole("heading", { name: /Deteccao de projecto/ })).toBeVisible();
    await expect(painel).toContainText(/le' os nomes dos ficheiros/);
    await expect(painel).toContainText(/Nao escreve, nao instala e nao constroi/);
    await expect(page.getByTestId("deteccao-alvos-sugestao")).toContainText(/sugestao/i);
    await expect(page.getByTestId("deteccao-alvos-sugestao")).toContainText(
      /Nao significam que a toolchain esteja instalada/i,
    );
  });

  // NAO TEM CAMPO PARA A RAIZ.
  test("nao tem campo para a raiz do projecto", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("deteccao");
    await expect(painel).toBeVisible();
    await expect(painel.locator("input")).toHaveCount(0);
    await expect(painel).toContainText(/nunca deste painel/);
  });

  // NAO EXISTE BOTAO DE INSTALAR, CONSTRUIR OU ASSINAR.
  test("nao existe botao de instalar, construir ou assinar", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("deteccao");
    await expect(painel).toBeVisible();
    const botoes = painel.getByRole("button");
    await expect(botoes).toHaveCount(1);
    const textos = await botoes.allInnerTexts();
    for (const texto of textos) {
      const minusculas = texto.toLowerCase();
      for (const proibido of ["instalar", "construir", "assinar", "publicar", "build"]) {
        expect(minusculas).not.toContain(proibido);
      }
      expect(minusculas).toContain("detectar");
    }
  });

  test("nenhum resultado e' mostrado sem um pedido", async ({ page }) => {
    await page.goto(ROTA);
    await expect(page.getByTestId("deteccao")).toBeVisible();
    await expect(page.getByTestId("deteccao-relatorio")).toHaveCount(0);
  });

  // O CORPO E' VAZIO -- A RAIZ NAO VAI NO PEDIDO.
  test("o corpo do pedido e' vazio -- sem raiz", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("deteccao");

    const pedido = page.waitForRequest(
      (requisicao) =>
        requisicao.url().includes("/api/agent/v1/project/detect") &&
        requisicao.method() === "POST",
    );
    await painel.getByRole("button", { name: /detectar projecto/i }).click();
    const capturado = await pedido;

    const corpo = capturado.postDataJSON() as Record<string, unknown>;
    expect(Object.keys(corpo)).toEqual([]);
    const texto = JSON.stringify(corpo).toLowerCase();
    for (const proibido of ["raiz", "caminho", "path", "directorio", "root"]) {
      expect(texto).not.toContain(proibido);
    }
  });

  test.describe("modo UI_FIXTURE", () => {
    test.skip(MODO !== "UI_FIXTURE", "corre so' com o fixture");

    // O TESTE QUE SO' O BROWSER FAZ: os valores vem do backend.
    test("traduz as stacks e gestores com os valores REAIS do backend", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("deteccao");
      await painel.getByRole("button", { name: /detectar projecto/i }).click();

      await expect(page.getByTestId("deteccao-relatorio")).toBeVisible();
      // `go` -> "Go", `node` -> "Node"
      const stacks = page.getByTestId("deteccao-lista-stacks");
      await expect(stacks).toContainText("Go");
      await expect(stacks).toContainText("Node");
      // `go-modules` -> "Go modules", `npm` -> "npm"
      const gestores = page.getByTestId("deteccao-lista-gestores");
      await expect(gestores).toContainText("Go modules");
      await expect(gestores).toContainText("npm");

      await expect(page.getByTestId("deteccao-stacks")).toHaveText("2");
      await expect(page.getByTestId("deteccao-gestores")).toHaveText("2");
      await expect(page.getByTestId("deteccao-alvos")).toHaveText("2");
    });

    // A PROVA NAO PROMETE TOOLCHAIN.
    test("a prova diz que toolchain.doctor e' que decide", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("deteccao");
      await painel.getByRole("button", { name: /detectar projecto/i }).click();

      const prova = page.getByTestId("deteccao-prova");
      await expect(prova).toBeVisible();
      await expect(prova).toContainText(/NAO prova que a toolchain esteja instalada/);
      await expect(prova).toContainText(/toolchain.doctor/);
    });

    test("mostra a raiz detectada e os manifests lidos", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("deteccao");
      await painel.getByRole("button", { name: /detectar projecto/i }).click();

      await expect(page.getByTestId("deteccao-raiz")).toHaveText("/projecto-hades");
      const manifests = page.getByTestId("deteccao-lista-manifests");
      await expect(manifests).toContainText("go.mod");
      await expect(manifests).toContainText("package.json");
    });

    test("mostra os avisos do detector", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("deteccao");
      await painel.getByRole("button", { name: /detectar projecto/i }).click();

      const avisos = page.getByTestId("deteccao-avisos");
      await expect(avisos).toBeVisible();
      await expect(avisos).toContainText("varios go.mod encontrados");
    });

    test("mostra o aviso de fronteira do backend por inteiro", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("deteccao");
      await painel.getByRole("button", { name: /detectar projecto/i }).click();

      const aviso = page.getByTestId("deteccao-aviso-fronteira");
      await expect(aviso).toBeVisible();
      await expect(aviso).toContainText(/apenas LE'/);
      await expect(aviso).toContainText(/SUGESTAO/);
      await expect(aviso).toContainText(/toolchain.doctor/);
      await expect(aviso).toContainText(/nunca do pedido/);
    });
  });

  test.describe("modo BACKEND_REAL_AGENT_RUNTIME", () => {
    test.skip(MODO !== "BACKEND_REAL_AGENT_RUNTIME", "corre so' com o backend real");

    // Sem `HADES_RAIZ_PROJECTO` o servidor recusa com 412. O painel tem de mostrar isso como erro
    // accionavel nomeando a variavel -- nao como um resultado vazio.
    test("raiz nao configurada produz erro accionavel, nao um resultado", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("deteccao");
      await expect(painel).toBeVisible();
      await painel.getByRole("button", { name: /detectar projecto/i }).click();

      const erro = page.getByTestId("deteccao-erro");
      await expect(erro).toBeVisible();
      await expect(erro).toContainText(/HADES_RAIZ_PROJECTO/);
      await expect(page.getByTestId("deteccao-relatorio")).toHaveCount(0);
    });
  });
});
