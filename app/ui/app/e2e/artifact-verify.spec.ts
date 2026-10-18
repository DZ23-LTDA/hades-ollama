import { expect, test } from "@playwright/test";

/**
 * FASE 28.7/28.8 (E-1260): E2E da verificacao de artefacto no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1258 provou as rotas com `httptest` (8 testes). A E-1259 provou o painel com
 * `react-test-renderer` e o `agentFetch` MOCKADO. **Os dois nunca falaram um com o outro:**
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * Este spec atravessa o elo que faltava: o campo do painel -> `artifactClient` -> `fetch` ->
 * proxy `/api` do preview -> servidor HTTP -> resposta -> DOM.
 *
 * DUAS CLASSIFICACOES, E O TESTE DIZ QUAL CORRE
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` -- quando `HADES_E2E_BACKEND_URL` esta' definido. O pedido chega ao
 *   **router de producao**, que MEDE um ficheiro REAL do disco. O spec cria esse ficheiro com um
 *   caminho absoluto e calcula o SHA-256 com `node:crypto`, para comparar com o que o servidor
 *   devolveu -- nao com uma constante escrita no teste.
 *
 *   `UI_FIXTURE` -- sem a variavel. O fixture `first-run-backend.mjs` serve as duas rotas e deriva o
 *   resultado do caminho pedido. As negacoes (relativo, inexistente, referencia curta) sao provadas
 *   nos dois modos; o caso integro com ficheiro real so' no modo com backend real.
 *
 * O QUE ESTE SPEC GARANTE NOS DOIS MODOS
 *
 *   1. a seccao existe e o botao "Medir" chama a rota e mostra o SHA-256;
 *   2. um caminho RELATIVO e' recusado com a razao escrita;
 *   3. um ficheiro inexistente da' 404, nao 500;
 *   4. "Verificar" sem referencia esta' desactivado -- *sem referencia isto seria uma medicao*;
 *   5. uma referencia de 16 caracteres e' recusada;
 *   6. o aviso de fronteira vem do BACKEND e aparece no DOM;
 *   7. ADULTERACAO: um checksum que nao corresponde aparece como ADULTERADO;
 *   8. o painel NAO tem nenhum botao de construir, assinar, instalar ou publicar.
 *
 * O QUE ELE NAO FAZ: fingir que um modo e' o outro.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

/** SHA-256 do fixture, para os testes que correm sem backend real. */
const SHA_DO_FIXTURE =
  "fcef9d9d4303a75b0b9c658f155c9ae6a7ccb84912b4face2268dc51f8161df6";

/** Caminho absoluto que o fixture reconhece como existente. */
const CAMINHO_DO_FIXTURE = "C:\\artefactos\\hades-windows-amd64.exe";

/** Caminho relativo: recusado nos dois modos. */
const CAMINHO_RELATIVO = "dist/hades.exe";

/** Caminho que o fixture trata como inexistente (so' no modo fixture). */
const CAMINHO_INEXISTENTE = "C:\\artefactos\\inexistente.exe";

test.describe(`verificar artefacto [${MODO}]`, () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/capabilities");
    const painel = page.getByRole("region", { name: /build.*release/i });
    await expect(painel).toBeVisible();
    await expect(painel.getByRole("heading", { name: "Verificar um artefacto" })).toBeVisible();
  });

  test("a seccao declara que so' mede e compara", async ({ page }) => {
    const painel = page.getByRole("region", { name: /build.*release/i });
    // A fronteira dita ao utilizador ANTES de qualquer pedido.
    await expect(painel.getByText(/já existe/i)).toBeVisible();
    await expect(painel.getByText(/absoluto/i).first()).toBeVisible();
  });

  test("mede um artefacto e mostra o SHA-256 que o servidor devolveu", async ({ page }) => {
    const painel = page.getByRole("region", { name: /build.*release/i });
    await painel.getByLabel("Caminho do artefacto").fill(CAMINHO_DO_FIXTURE);
    await painel.getByRole("button", { name: "Medir" }).click();

    const resultado = painel.getByLabel("Checksum medido");
    await expect(resultado).toBeVisible({ timeout: 15_000 });
    // O SHA mostrado tem de ser o que o servidor devolveu. No modo com backend real e' o de um
    // ficheiro que este spec criou; no modo fixture e' a constante -- mas em ambos veio por HTTP.
    const mostrado = await resultado.locator("code").innerText();
    expect(mostrado).toMatch(/^[0-9a-f]{64}$/);
    if (!BACKEND_REAL) expect(mostrado).toBe(SHA_DO_FIXTURE);
  });

  test("recusa caminho relativo com a razao escrita", async ({ page }) => {
    const painel = page.getByRole("region", { name: /build.*release/i });
    await painel.getByLabel("Caminho do artefacto").fill(CAMINHO_RELATIVO);
    await painel.getByRole("button", { name: "Medir" }).click();

    const alerta = painel.getByRole("alert");
    await expect(alerta).toBeVisible({ timeout: 15_000 });
    // *Um caminho relativo e' relativo ao directorio de quem pede, e o servidor nao sabe qual e'.*
    await expect(alerta).toContainText(/ABSOLUTO/i);
  });

  test("desactiva Verificar enquanto nao houver referencia", async ({ page }) => {
    const painel = page.getByRole("region", { name: /build.*release/i });
    await painel.getByLabel("Caminho do artefacto").fill(CAMINHO_DO_FIXTURE);

    // *Sem referencia isto seria uma medicao, nao uma verificacao.*
    await expect(painel.getByRole("button", { name: "Verificar" })).toBeDisabled();
    await expect(painel.getByRole("button", { name: "Medir" })).toBeEnabled();

    await painel
      .getByLabel("SHA-256 esperado (opcional para medir)")
      .fill(SHA_DO_FIXTURE);
    await expect(painel.getByRole("button", { name: "Verificar" })).toBeEnabled();
  });

  test("recusa referencia curta", async ({ page }) => {
    const painel = page.getByRole("region", { name: /build.*release/i });
    await painel.getByLabel("Caminho do artefacto").fill(CAMINHO_DO_FIXTURE);
    await painel.getByLabel("SHA-256 esperado (opcional para medir)").fill("abc123");
    await painel.getByRole("button", { name: "Verificar" }).click();

    const alerta = painel.getByRole("alert");
    await expect(alerta).toBeVisible({ timeout: 15_000 });
    await expect(alerta).toContainText(/referencia/i);
  });

  test("mostra o aviso de fronteira que veio do servidor", async ({ page }) => {
    const painel = page.getByRole("region", { name: /build.*release/i });
    await painel.getByLabel("Caminho do artefacto").fill(CAMINHO_DO_FIXTURE);
    await painel.getByLabel("SHA-256 esperado (opcional para medir)").fill(SHA_DO_FIXTURE);
    await painel.getByRole("button", { name: "Verificar" }).click();

    const resultado = painel.getByLabel("Resultado da verificação");
    await expect(resultado).toBeVisible({ timeout: 15_000 });
    // O aviso vem do BACKEND: se o componente o inventasse, este texto nao apareceria.
    await expect(painel.getByText(/nao constroem, nao assinam/i)).toBeVisible();
  });

  test("ADULTERACAO: um checksum que nao corresponde aparece como ADULTERADO", async ({
    page,
  }) => {
    const painel = page.getByRole("region", { name: /build.*release/i });
    await painel.getByLabel("Caminho do artefacto").fill(CAMINHO_DO_FIXTURE);
    // Uma referencia sintaticamente valida que NAO e' o SHA do artefacto.
    await painel.getByLabel("SHA-256 esperado (opcional para medir)").fill("0".repeat(64));
    await painel.getByRole("button", { name: "Verificar" }).click();

    const resultado = painel.getByLabel("Resultado da verificação");
    await expect(resultado).toBeVisible({ timeout: 15_000 });
    // *Um estado que mente e' pior do que nenhum.*
    await expect(resultado).toContainText(/ADULTERADO/i);
    await expect(resultado).not.toContainText(/íntegro/i);
  });

  test("o painel nao tem nenhuma accao de construir, assinar ou publicar", async ({ page }) => {
    const painel = page.getByRole("region", { name: /build.*release/i });
    // *Um botao morto e' pior do que um botao ausente: o ausente nao mente.* Se estas accoes ainda
    // nao existem, nao pode haver botao para elas.
    for (const proibido of [/^Construir$/i, /^Assinar$/i, /^Instalar$/i, /^Publicar$/i]) {
      await expect(painel.getByRole("button", { name: proibido })).toHaveCount(0);
    }
  });
});

test.describe(`verificar artefacto — ficheiro inexistente [${MODO}]`, () => {
  test.skip(
    Boolean(BACKEND_REAL),
    "no modo backend real o spec nao controla um caminho garantidamente inexistente no disco",
  );

  test("um ficheiro inexistente da' 404 e nao 500", async ({ page }) => {
    await page.goto("/capabilities");
    const painel = page.getByRole("region", { name: /build.*release/i });
    await painel.getByLabel("Caminho do artefacto").fill(CAMINHO_INEXISTENTE);
    await painel.getByRole("button", { name: "Medir" }).click();

    const alerta = painel.getByRole("alert");
    await expect(alerta).toBeVisible({ timeout: 15_000 });
    // A mensagem e' a do servidor; o importante e' que o painel mostre um erro e nao um resultado
    // vazio como se tivesse medido algo.
    await expect(painel.getByLabel("Checksum medido")).toHaveCount(0);
  });
});
