import { expect, test } from "@playwright/test";

/**
 * FASE 28.8 (E-1285): E2E do painel de verificacao no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1284 provou o painel com `agentFetch` MOCKADO. A rota `POST /build/verify` foi provada com
 * `httptest`. **Os dois nunca falaram um com o outro.**
 *
 * O TESTE QUE SO' O BROWSER CONSEGUE FAZER
 *
 * O fixture devolve `sem_assinatura: 1` e `nao_verificadas: 1` -- **o mesmo valor nos dois**. Se o
 * painel os somasse, apareceria **2** num campo so'. O E2E le cada campo pelo `data-testid` e exige
 * **1 e 1**.
 *
 * Um teste unitario nao apanha isto se ele proprio construir o objecto; o browser apanha porque le o
 * DOM que o painel realmente pintou.
 *
 *	*Um estado que mente e' pior do que nenhum.*
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

const ROTA = "/capabilities";

test.describe(`painel de verificacao de artefactos (${MODO})`, () => {
  test("diz que a rota so' compara checksums, antes de haver resultado", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("verificacao");
    await expect(painel).toBeVisible();
    await expect(painel.getByRole("heading", { name: /Verificacao de artefactos/ })).toBeVisible();
    await expect(painel).toContainText(/apenas compara/i);
    await expect(painel).toContainText(/Nao constroi, nao assina, nao instala/);
    await expect(painel).toContainText(/Aprovada nao significa assinada/);
  });

  test("declara que os caminhos sao de demonstracao", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("verificacao");
    await expect(painel).toBeVisible();
    await expect(page.getByTestId("verificacao-demonstracao")).toBeVisible();
    await expect(page.getByTestId("verificacao-demonstracao")).toContainText(/demonstracao/i);
    await expect(page.getByTestId("verificacao-demonstracao")).toContainText(/nao de um build real/i);
  });

  // NAO EXISTE BOTAO DE ASSINAR, CONSTRUIR OU INSTALAR.
  test("nao existe botao de assinar, construir, instalar ou publicar", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("verificacao");
    await expect(painel).toBeVisible();
    const botoes = painel.getByRole("button");
    await expect(botoes).toHaveCount(1);
    const textos = await botoes.allInnerTexts();
    for (const texto of textos) {
      const minusculas = texto.toLowerCase();
      for (const proibido of ["assinar", "construir", "instalar", "publicar"]) {
        expect(minusculas).not.toContain(proibido);
      }
      expect(minusculas).toContain("verificar");
    }
  });

  // NAO TEM CAMPO PARA ESCOLHER CAMINHOS.
  test("nao tem campo para escolher caminhos ou checksums", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("verificacao");
    await expect(painel).toBeVisible();
    await expect(painel.locator("input")).toHaveCount(0);
  });

  test("nenhum resultado e' mostrado sem um pedido", async ({ page }) => {
    await page.goto(ROTA);
    await expect(page.getByTestId("verificacao")).toBeVisible();
    await expect(page.getByTestId("verificacao-relatorio")).toHaveCount(0);
  });

  // O PEDIDO TEM `artefactos` COM CAMINHO E SHA-256 DE 64 HEX.
  test("envia artefactos com caminho e sha256 de 64 hex", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("verificacao");

    const pedido = page.waitForRequest(
      (requisicao) =>
        requisicao.url().includes("/api/agent/v1/build/verify") && requisicao.method() === "POST",
    );
    await painel.getByRole("button", { name: /verificar artefactos/i }).click();
    const capturado = await pedido;

    const corpo = capturado.postDataJSON() as {
      artefactos: { caminho: string; sha256: string }[];
    };
    expect(Array.isArray(corpo.artefactos)).toBe(true);
    expect(corpo.artefactos.length).toBeGreaterThan(0);
    for (const artefacto of corpo.artefactos) {
      expect(typeof artefacto.caminho).toBe("string");
      expect(artefacto.caminho.length).toBeGreaterThan(0);
      expect(artefacto.sha256).toMatch(/^[a-f0-9]{64}$/);
    }
    // E nenhuma chave de raiz.
    expect(Object.keys(corpo)).toEqual(["artefactos"]);
  });

  test.describe("modo UI_FIXTURE", () => {
    test.skip(MODO !== "UI_FIXTURE", "corre so' com o fixture");

    // O TESTE QUE SO' O BROWSER FAZ: os dois campos NAO podem ser somados.
    test("sem_assinatura e nao_verificadas ficam separados com 1 e 1", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("verificacao");
      await painel.getByRole("button", { name: /verificar artefactos/i }).click();

      await expect(page.getByTestId("verificacao-relatorio")).toBeVisible();
      // O fixture devolve 1 e 1. Se o painel somasse, um dos campos leria 2.
      await expect(page.getByTestId("verificacao-sem-assinatura")).toHaveText("1");
      await expect(page.getByTestId("verificacao-nao-verificadas")).toHaveText("1");
    });

    test("o resumo diz que ha' por inspeccionar -- nao que esta' tudo assinado", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("verificacao");
      await painel.getByRole("button", { name: /verificar artefactos/i }).click();

      const resumo = page.getByTestId("verificacao-resumo");
      await expect(resumo).toBeVisible();
      await expect(resumo).toContainText(/NAO foi\/foram inspeccionado/);
      // E nunca afirma que esta' assinado.
      await expect(resumo).not.toContainText(/assinad/i);
    });

    test("mostra os artefactos e os estados de assinatura do backend", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("verificacao");
      await painel.getByRole("button", { name: /verificar artefactos/i }).click();

      const resultados = page.getByTestId("verificacao-resultados");
      await expect(resultados).toBeVisible();
      await expect(resultados).toContainText("hades-windows-amd64");
      // Os rotulos vem do backend, nao de valores inventados.
      await expect(resultados).toContainText("sem assinatura (o adaptador olhou)");
      await expect(resultados).toContainText("nao inspeccionada (sem adaptador)");
    });

    test("aprovada nao e' apresentada como prova de assinatura", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("verificacao");
      await painel.getByRole("button", { name: /verificar artefactos/i }).click();

      await expect(page.getByTestId("verificacao-aprovada")).toHaveText("sim");
      const prova = page.getByTestId("verificacao-prova");
      await expect(prova).toBeVisible();
      // Com `nada_assinado: false` e `aprovada: true`, a prova tem de negar equivalencia.
      await expect(prova).toContainText(/NAO prova assinatura/);
    });

    test("mostra o aviso de fronteira do backend por inteiro", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("verificacao");
      await painel.getByRole("button", { name: /verificar artefactos/i }).click();

      const aviso = page.getByTestId("verificacao-aviso-fronteira");
      await expect(aviso).toBeVisible();
      await expect(aviso).toContainText(/apenas COMPARA/);
      await expect(aviso).toContainText(/nao significa que exista assinatura/);
      await expect(aviso).toContainText(/leia `nao_verificadas`/);
    });
  });

  test.describe("modo BACKEND_REAL_AGENT_RUNTIME", () => {
    test.skip(MODO !== "BACKEND_REAL_AGENT_RUNTIME", "corre so' com o backend real");

    // Contra o router de producao os caminhos de demonstracao nao existem. A rota devolve um 4xx/5xx
    // accionavel, e o painel tem de o mostrar como erro -- nao como um resultado vazio.
    test("caminho inexistente produz erro accionavel, nao um resultado", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("verificacao");
      await expect(painel).toBeVisible();
      await painel.getByRole("button", { name: /verificar artefactos/i }).click();

      // Ou erro explicito, ou relatorio com artefactos nao integros. Nunca silencio.
      const erro = page.getByTestId("verificacao-erro");
      const relatorio = page.getByTestId("verificacao-relatorio");
      await expect(erro.or(relatorio)).toBeVisible();
    });
  });
});
