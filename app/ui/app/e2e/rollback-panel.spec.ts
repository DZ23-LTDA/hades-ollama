import { expect, test } from "@playwright/test";

/**
 * FASE 28.8 (E-1278): E2E do painel de ROLLBACK no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1277 provou o painel com `react-test-renderer` e o `agentFetch` MOCKADO. A E-1276 provou a rota
 * com `httptest`. **Os dois nunca falaram um com o outro.**
 *
 * As duas classificacoes:
 *
 *   `UI_FIXTURE` -- o fixture devolve um plano com candidato (WINDOWS_INSTALLER) ou 409 (Play).
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` -- atravessa o proxy ate' ao router de producao. **Sem
 *   `HADES_ARTEFACTOS`** o servidor responde **412**, e o spec exige que o painel mostre isso como
 *   erro accionavel nomeando a variavel -- nunca como um plano.
 *
 * O QUE ESTE SPEC GARANTE NOS DOIS MODOS
 *
 *   1. o painel **diz que so' planeia** antes de haver resultado;
 *   2. **NENHUM BOTAO DE RESTAURAR** -- o spec conta os botoes do painel e proibe os rotulos
 *      "restaurar", "aplicar" e "executar rollback";
 *   3. **o pedido HTTP sai de verdade** (`page.waitForRequest`) e leva **so' o `tipo`** -- nunca um
 *      `directorio`;
 *   4. **nenhum plano e' mostrado sem um pedido**.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

const ROTA = "/capabilities";

/** sha256 do candidato do fixture. */
const SHA_DO_CANDIDATO =
  "7d865e959b2466918c9863afca942d0fb89d7c9ac0c99bafc3749504ded97730";

test.describe(`painel de rollback (${MODO})`, () => {
  test("diz que so' planeia antes de haver resultado", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("rollback");
    await expect(painel).toBeVisible();
    await expect(painel.getByRole("heading", { name: "Plano de rollback" })).toBeVisible();
    await expect(painel).toContainText(/planeia/i);
    await expect(painel).toContainText(/Nada e' restaurado/i);
  });

  // NENHUM BOTAO DE RESTAURAR.
  test("nao existe botao de restaurar", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("rollback");
    await expect(painel).toBeVisible();
    const botoes = painel.getByRole("button");
    // So' um botao: "Planear rollback".
    await expect(botoes).toHaveCount(1);
    const textos = await botoes.allInnerTexts();
    for (const texto of textos) {
      const minusculas = texto.toLowerCase();
      expect(minusculas).not.toContain("restaurar");
      expect(minusculas).not.toContain("aplicar");
      expect(minusculas).not.toContain("executar rollback");
    }
  });

  test("nenhum plano e' mostrado sem um pedido", async ({ page }) => {
    await page.goto(ROTA);
    await expect(page.getByTestId("rollback")).toBeVisible();
    await expect(page.getByTestId("rollback-plano")).toHaveCount(0);
  });

  // O PEDIDO LEVA SO' O `tipo`.
  test("o pedido sai de verdade e leva apenas o tipo", async ({ page }) => {
    await page.goto(ROTA);
    const painel = page.getByTestId("rollback");
    await expect(painel).toBeVisible();

    const pedido = page.waitForRequest(
      (requisicao) =>
        requisicao.url().includes("/api/agent/v1/publish/rollback") &&
        requisicao.method() === "POST",
    );
    await painel.getByRole("button", { name: /planear rollback/i }).click();
    const capturado = await pedido;

    // O corpo nao pode ter `directorio` -- o directorio vem do AMBIENTE do servidor.
    const corpo = capturado.postDataJSON() as Record<string, unknown>;
    expect(Object.keys(corpo)).toEqual(["tipo"]);
    expect(corpo).not.toHaveProperty("directorio");
  });

  test.describe("modo UI_FIXTURE", () => {
    test.skip(MODO !== "UI_FIXTURE", "corre so' com o fixture");

    test("mostra o alvo, o sha256 e executado=nao", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("rollback");
      await painel.getByRole("button", { name: /planear rollback/i }).click();

      await expect(page.getByTestId("rollback-plano")).toBeVisible();
      await expect(page.getByTestId("rollback-executado")).toHaveText("nao");
      await expect(page.getByTestId("rollback-candidatos")).toHaveText("2");
      await expect(page.getByTestId("rollback-alvo")).toHaveText("HadesSetup-anterior.exe");
      await expect(page.getByTestId("rollback-plano")).toContainText(SHA_DO_CANDIDATO);
    });

    test("mostra o aviso de fronteira por inteiro", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("rollback");
      await painel.getByRole("button", { name: /planear rollback/i }).click();

      const aviso = page.getByTestId("rollback-aviso-fronteira");
      await expect(aviso).toBeVisible();
      await expect(aviso).toContainText(/nenhum ficheiro foi movido, copiado ou apagado/i);
      await expect(aviso).toContainText(/nunca do pedido/i);
    });

    test("um canal sem credencial mostra a explicacao, nao um alerta generico", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("rollback");

      // Trocar para Google Play -- o fixture responde 409 com a explicacao.
      await painel.locator("#rollback-canal").selectOption("GOOGLE_PLAY_ROLLOUT");
      await painel.getByRole("button", { name: /planear rollback/i }).click();

      // 409 e' um erro HTTP: o painel mostra-o em role=alert. Mas o texto tem de nomear o que falta.
      const erro = page.getByTestId("rollback-erro");
      await expect(erro).toBeVisible();
      await expect(erro).toContainText(/service account JSON/i);
      // E nenhum plano pode ter aparecido.
      await expect(page.getByTestId("rollback-plano")).toHaveCount(0);
    });
  });

  test.describe("modo BACKEND_REAL_AGENT_RUNTIME", () => {
    test.skip(MODO !== "BACKEND_REAL_AGENT_RUNTIME", "corre so' com o backend real");

    // Sem `HADES_ARTEFACTOS` o servidor recusa com 412. O painel tem de mostrar isso como erro
    // accionavel -- nao como um plano vazio.
    test("raiz nao configurada produz erro accionavel, nao um plano", async ({ page }) => {
      await page.goto(ROTA);
      const painel = page.getByTestId("rollback");
      await expect(painel).toBeVisible();
      await painel.getByRole("button", { name: /planear rollback/i }).click();

      const erro = page.getByTestId("rollback-erro");
      await expect(erro).toBeVisible();
      await expect(erro).toContainText(/HADES_ARTEFACTOS/);
      await expect(page.getByTestId("rollback-plano")).toHaveCount(0);
    });
  });
});
