import { expect, test } from "@playwright/test";

/**
 * FASE 28.7/28.8 + 28.5: E2E das duas superficies novas no browser real.
 *
 * DUAS CLASSIFICACOES, e o teste diz qual corre:
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` — quando `HADES_E2E_BACKEND_URL` esta' definido. O pedido sai
 *   do preview, atravessa o proxy `/api` e chega a um servidor Hades real com as rotas
 *   `toolchain/*` e `signing/*`. Neste modo o spec compara a UI com a ROTA REAL.
 *
 *   `UI_FIXTURE` — sem a variavel. O fixture `first-run-backend.mjs` serve dados marcados com
 *   `source: "e2e-fixture"` e nomes `fixture-*`. Neste modo o spec mede a UI: contagem,
 *   veredictos, rotulos e a regra de aprovacao. Nao mede o catalogo real, e di-lo.
 *
 * O QUE ESTE SPEC NAO FAZ: fingir que um modo e' o outro. *Um teste que dissesse "UI verificada
 * contra o servidor real" correndo contra um fixture seria pior do que nenhum.*
 *
 * PORQUE ESTE SPEC EXISTE
 *
 * As duas paginas tinham testes com `react-test-renderer` e a camada HTTP **mockada**. Isso
 * prova a logica do componente, nao a integracao. *Um E2E que intercepta toda a API e nunca
 * toca o backend real nao prova integracao* -- e as duas evidencias (`E-1213`, `E-1216`)
 * declararam exactamente essa lacuna. Este spec fecha-a onde o ambiente permite.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

test.describe(`capacidades e identidades de assinatura (${MODO})`, () => {
  // ---------------------------------------------------------------------------------------
  // CAPACIDADES (FASE 28.7/28.8)
  // ---------------------------------------------------------------------------------------

  test("o doctor mostra exactamente os alvos que o servidor devolveu", async ({ page, request }) => {
    // 1. A FONTE. Leio a rota que alimenta a pagina, seja real ou fixture.
    const resposta = await request.get("/api/agent/v1/toolchain/doctor", { timeout: 30_000 });
    expect(resposta.status(), "GET /toolchain/doctor tem de responder 200").toBe(200);

    const corpo = await resposta.json();
    expect(Array.isArray(corpo.alvos), "a rota tem de devolver uma lista").toBe(true);
    const alvos = corpo.alvos as Array<{ alvo: string }>;
    expect(alvos.length, "o fixture e o servidor tem de declarar alvos").toBeGreaterThan(0);

    // 2. A UI.
    await page.goto("/capabilities");

    // O resumo vem do backend, nao e' recalculado no cliente.
    // *Uma contagem calculada no cliente pode discordar do servidor e ninguem saber qual esta' certo.*
    await expect(page.getByText(corpo.resumo as string)).toBeVisible({ timeout: 15_000 });

    // 3. O DISCRIMINADOR: cada alvo do servidor aparece na UI, e a contagem bate.
    for (const alvo of alvos) {
      await expect(page.getByText(alvo.alvo, { exact: false })).toBeVisible();
    }
    const linhas = page.getByRole("list", { name: "Alvos de build" }).locator("li", {
      has: page.locator("button[aria-expanded]"),
    });
    await expect(linhas).toHaveCount(alvos.length);
  });

  test("o plano de um alvo mostra blockers E trabalho independente", async ({ page, request }) => {
    const servidor = await request.get("/api/agent/v1/toolchain/doctor", { timeout: 30_000 });
    const corpo = await servidor.json();
    const alvos = corpo.alvos as Array<{ alvo: string; veredicto: string }>;
    const bloqueado = alvos.find((a) => a.veredicto === "bloqueado");
    test.skip(!bloqueado, "nenhum alvo bloqueado neste ambiente; o caso nao existe para medir");

    const plano = await request.get(
      `/api/agent/v1/toolchain/plan/${encodeURIComponent(bloqueado!.alvo)}`,
      { timeout: 30_000 },
    );
    expect(plano.status()).toBe(200);
    const dados = await plano.json();
    expect(Array.isArray(dados.blockers)).toBe(true);
    expect(dados.blockers.length).toBeGreaterThan(0);

    await page.goto("/capabilities");
    await page.getByText(bloqueado!.alvo, { exact: false }).first().click();

    // O bloqueio vem ACOMPANHADO do que pode avancar.
    // *"Depende de conta externa" nao e' razao para parar o que nao depende.*
    await expect(page.getByText("Bloqueios")).toBeVisible({ timeout: 10_000 });
    for (const blocker of dados.blockers as string[]) {
      await expect(page.getByText(blocker, { exact: false })).toBeVisible();
    }
    if (Array.isArray(dados.trabalho_independente) && dados.trabalho_independente.length > 0) {
      await expect(page.getByText("Pode avancar sem esperar")).toBeVisible();
    }
  });

  // ---------------------------------------------------------------------------------------
  // IDENTIDADES DE ASSINATURA (FASE 28.5)
  // ---------------------------------------------------------------------------------------

  test("as identidades mostram os metadados que o servidor devolveu", async ({ page, request }) => {
    const resposta = await request.get("/api/agent/v1/signing/identities", { timeout: 30_000 });
    expect(resposta.status(), "GET /signing/identities tem de responder 200").toBe(200);

    const corpo = await resposta.json();
    const identidades = corpo.identidades as Array<{
      nome: string;
      plataforma: string;
      ambiente: string;
      fingerprint_publico_fim: string;
      exige_aprovacao: boolean;
    }>;

    await page.goto("/signing");

    // O aviso vem do servidor: o consumidor nao deve ter de adivinhar o contrato.
    await expect(page.getByText(corpo.aviso as string)).toBeVisible({ timeout: 15_000 });

    if (identidades.length === 0) {
      await expect(page.getByText("não declara nenhuma identidade")).toBeVisible();
      return;
    }

    for (const identidade of identidades) {
      await expect(page.getByText(identidade.nome, { exact: true })).toBeVisible();
      // O fingerprint vem TRUNCADO do servidor e aparece truncado na UI.
      await expect(page.getByText(`…${identidade.fingerprint_publico_fim}`)).toBeVisible();
    }
  });

  test("identidade de producao nunca diz 'pode assinar agora'", async ({ page, request }) => {
    const resposta = await request.get("/api/agent/v1/signing/identities", { timeout: 30_000 });
    const corpo = await resposta.json();
    const identidades = corpo.identidades as Array<{
      nome: string;
      exige_aprovacao: boolean;
    }>;
    const producao = identidades.find((i) => i.exige_aprovacao);
    test.skip(!producao, "nenhuma identidade que exija aprovacao neste ambiente");

    await page.goto("/signing");
    await expect(page.getByText(producao!.nome, { exact: true })).toBeVisible({ timeout: 15_000 });

    // A linha daquela identidade diz "requer aprovação", NAO "pode assinar agora".
    // *Uma UI que mostra verde numa identidade que ainda precisa de aprovacao ensina o
    // utilizador a carregar no botao antes de ler.*
    const linha = page.locator("tr", { hasText: producao!.nome });
    await expect(linha).toContainText("requer aprovação");
    await expect(linha).not.toContainText("pode assinar agora");
  });

  test("a pagina de identidades nao expoe material secreto", async ({ page }) => {
    await page.goto("/signing");
    // Espera o conteudo carregar (o aviso do servidor esta' sempre presente).
    await expect(page.getByText(/apenas metadados/i)).toBeVisible({ timeout: 15_000 });

    const html = (await page.content()).toLowerCase();
    for (const proibido of [
      "private_key",
      "-----begin",
      "storepassword",
      "keypassword",
      "password",
    ]) {
      expect(html, `a pagina contem ${proibido}`).not.toContain(proibido);
    }
  });

  test("as rotas de diagnostico e de identidades sao so' leitura", async ({ request }) => {
    // *Um preflight que muda o sistema nao e' preflight: e' instalacao disfarcada.*
    for (const caminho of [
      "/api/agent/v1/toolchain/targets",
      "/api/agent/v1/toolchain/doctor",
      "/api/agent/v1/signing/identities",
      "/api/agent/v1/signing/summary",
    ]) {
      const resposta = await request.post(caminho, { timeout: 30_000 });
      expect(resposta.status(), `POST ${caminho} nao pode devolver 200`).not.toBe(200);
    }
  });
});
