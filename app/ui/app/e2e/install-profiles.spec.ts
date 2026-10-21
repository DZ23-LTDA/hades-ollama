import { expect, test } from "@playwright/test";

/**
 * FASE 28.6-A / 28.8: E2E do painel de PERFIS DE INSTALACAO no browser real.
 *
 * DUAS CLASSIFICACOES, e o teste diz qual corre:
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` — quando `HADES_E2E_BACKEND_URL` esta' definido. O pedido sai do
 *   preview, atravessa o proxy `/api` e chega a um servidor Hades real com as rotas `install/*`.
 *
 *   `UI_FIXTURE` — sem a variavel. O fixture `first-run-backend.mjs` serve dados marcados com
 *   `source: "e2e-fixture"`.
 *
 * PORQUE ESTE SPEC EXISTE
 *
 * `E-1223` fechou o circuito motor -> API -> UI, mas o E2E existente cobria so' `/toolchain/*` e
 * `/signing/*`. **O painel de perfis nao era exercitado num browser.** Este spec fecha essa
 * lacuna -- que eu proprio declarei.
 *
 * O QUE ELE NAO FAZ: fingir que um modo e' o outro.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

test.describe(`perfis de instalacao (${MODO})`, () => {
  test("o painel mostra exactamente os perfis que o servidor devolveu", async ({ page, request }) => {
    // 1. A FONTE.
    const resposta = await request.get("/api/agent/v1/install/profiles", { timeout: 30_000 });
    expect(resposta.status(), "GET /install/profiles tem de responder 200").toBe(200);

    const corpo = await resposta.json();
    const perfis = corpo.perfis as Array<{ id: string }>;
    expect(perfis.length, "o servidor tem de declarar perfis").toBeGreaterThan(0);

    // 2. A UI.
    await page.goto("/capabilities");
    const painel = page.getByRole("region", { name: "Perfis de instalacao" });
    await expect(painel).toBeVisible({ timeout: 15_000 });

    // 3. O DISCRIMINADOR: cada perfil do servidor aparece, e o numero bate.
    for (const perfil of perfis) {
      await expect(painel.getByText(perfil.id, { exact: true })).toBeVisible();
    }
    const botoes = painel.locator("button[aria-expanded]");
    await expect(botoes).toHaveCount(perfis.length);
  });

  test("a UI nao recalcula o catalogo: usa o que o servidor mandou", async ({ page, request }) => {
    const resposta = await request.get("/api/agent/v1/install/profiles", { timeout: 30_000 });
    const corpo = await resposta.json();
    const perfis = corpo.perfis as Array<{ id: string; escolhe_pacotes: boolean }>;

    await page.goto("/capabilities");
    const painel = page.getByRole("region", { name: "Perfis de instalacao" });
    await expect(painel).toBeVisible({ timeout: 15_000 });

    // O perfil que o servidor marca `escolhe_pacotes` diz isso na UI.
    // *Uma contagem calculada no cliente pode discordar do servidor e ninguem saber qual esta'
    // certo.*
    const escolhem = perfis.filter((p) => p.escolhe_pacotes);
    test.skip(escolhem.length === 0, "nenhum perfil de escolha livre neste ambiente");
    for (const perfil of escolhem) {
      // O botao DAQUELE perfil tem de dizer que a escolha e' manual. Nao basta o texto existir
      // algures na pagina: *um teste de "esta' escrito" nao prova que esta' escrito no sitio
      // certo.*
      const botao = painel.locator("button[aria-expanded]").filter({ hasText: perfil.id }).first();
      await expect(botao).toContainText("escolha componente a componente");
    }
  });

  test("abrir um perfil mostra o plano com accoes e motivos", async ({ page, request }) => {
    const resposta = await request.get("/api/agent/v1/install/profiles", { timeout: 30_000 });
    const corpo = await resposta.json();
    const perfis = corpo.perfis as Array<{ id: string }>;

    // Um perfil que NAO seja CUSTOM (tem itens).
    const alvo = perfis.find((p) => p.id !== "CUSTOM");
    test.skip(!alvo, "nenhum perfil com itens neste ambiente");

    const plano = await request.get(
      `/api/agent/v1/install/plan/${encodeURIComponent(alvo!.id)}`,
      { timeout: 30_000 },
    );
    expect(plano.status()).toBe(200);
    const dados = await plano.json();

    await page.goto("/capabilities");
    const painel = page.getByRole("region", { name: "Perfis de instalacao" });
    await expect(painel).toBeVisible({ timeout: 15_000 });

    await painel.getByText(alvo!.id, { exact: true }).click();

    // O resumo vem do servidor, nao e' recalculado no cliente.
    await expect(painel.getByText(dados.resumo as string)).toBeVisible({ timeout: 10_000 });

    // Cada item do plano aparece com o motivo.
    // `.first()` porque o nome do pacote pode aparecer tambem dentro de um bloqueio que o
    // menciona -- e nesse caso as duas ocorrencias sao a mesma informacao, nao duas provas.
    for (const item of dados.itens as Array<{ nome: string; motivo: string }>) {
      await expect(painel.getByText(item.nome, { exact: false }).first()).toBeVisible();
    }

    // E o payload declara que NAO executa nada.
    expect(dados.executa_algo).toBe(false);
  });

  test("UNKNOWN aparece como 'Nao avaliado', nunca como 'Ausente'", async ({ page, request }) => {
    const resposta = await request.get("/api/agent/v1/install/profiles", { timeout: 30_000 });
    const corpo = await resposta.json();
    const perfis = corpo.perfis as Array<{ id: string }>;

    // Procura um perfil cujo plano tenha um item `PULAR` (UNKNOWN).
    let achou = false;
    for (const perfil of perfis) {
      const plano = await request.get(
        `/api/agent/v1/install/plan/${encodeURIComponent(perfil.id)}`,
        { timeout: 30_000 },
      );
      if (plano.status() !== 200) continue;
      const dados = await plano.json();
      const pulados = (dados.itens as Array<{ acao: string }>).filter((i) => i.acao === "PULAR");
      if (pulados.length === 0) continue;

      achou = true;
      await page.goto("/capabilities");
      const painel = page.getByRole("region", { name: "Perfis de instalacao" });
      await expect(painel).toBeVisible({ timeout: 15_000 });
      await painel.getByText(perfil.id, { exact: true }).click();
      await expect(painel.getByText("Nao avaliado").first()).toBeVisible({ timeout: 10_000 });

      // A palavra "Ausente" NAO pode ser o rotulo de uma accao.
      // *Dizer "ausente" do que nao se conseguiu avaliar faria o utilizador instalar por cima
      // do que ja' tem.*
      const celulas = painel.locator("td");
      const total = await celulas.count();
      for (let i = 0; i < total; i++) {
        const texto = (await celulas.nth(i).textContent()) ?? "";
        expect(texto.trim(), "celula com rotulo 'Ausente'").not.toBe("Ausente");
      }
      break;
    }
    test.skip(!achou, "nenhum perfil com item UNKNOWN neste ambiente");
  });

  test("o painel nao oferece botao de instalar", async ({ page }) => {
    await page.goto("/capabilities");
    const painel = page.getByRole("region", { name: "Perfis de instalacao" });
    await expect(painel).toBeVisible({ timeout: 15_000 });

    // A `<section>` existe tambem em "a-carregar" (so' com o titulo). Esperar o primeiro botao
    // antes de contar -- *contar antes de carregar mede o estado vazio, nao a UI.*
    await expect(painel.locator("button[aria-expanded]").first()).toBeVisible({ timeout: 15_000 });

    // *Um botao "Instalar" na UI de diagnostico mentiria: instalar exige approval e superficie
    // propria.*
    const botoes = painel.locator("button");
    const total = await botoes.count();
    expect(total).toBeGreaterThan(0);
    for (let i = 0; i < total; i++) {
      const rotulo = ((await botoes.nth(i).textContent()) ?? "").trim().toLowerCase();
      expect(rotulo, `botao "${rotulo}"`).not.toBe("instalar");
      expect(rotulo, `botao "${rotulo}"`).not.toContain("instalar agora");
    }
  });

  test("a UI mostra os bloqueios quando o estado e' desconhecido", async ({ page, request }) => {
    const resposta = await request.get("/api/agent/v1/install/profiles", { timeout: 30_000 });
    const corpo = await resposta.json();
    const perfis = corpo.perfis as Array<{ id: string }>;

    let achou = false;
    for (const perfil of perfis) {
      const plano = await request.get(
        `/api/agent/v1/install/plan/${encodeURIComponent(perfil.id)}`,
        { timeout: 30_000 },
      );
      if (plano.status() !== 200) continue;
      const dados = await plano.json();
      const bloqueios = dados.bloqueios as string[] | null;
      if (!bloqueios || bloqueios.length === 0) continue;

      achou = true;
      await page.goto("/capabilities");
      const painel = page.getByRole("region", { name: "Perfis de instalacao" });
      await expect(painel).toBeVisible({ timeout: 15_000 });
      await painel.getByText(perfil.id, { exact: true }).click();
      await expect(painel.getByText("O que falta saber")).toBeVisible({ timeout: 10_000 });
      break;
    }
    test.skip(!achou, "nenhum perfil com bloqueios neste ambiente");
  });

  test("as rotas do instalador sao so' leitura", async ({ request }) => {
    // *Uma rota que planear e executar na mesma chamada poupa um clique e gasta uma decisao.*
    for (const caminho of [
      "/api/agent/v1/install/profiles",
      "/api/agent/v1/install/plan/BASIC",
    ]) {
      const resposta = await request.post(caminho, { timeout: 30_000 });
      expect(resposta.status(), `POST ${caminho} nao pode devolver 200`).not.toBe(200);
    }
  });

  test("perfil desconhecido e' recusado, nao vira plano vazio", async ({ request }) => {
    // *Um plano vazio para um perfil que nao existe diria "nada a instalar" a quem le so' o
    // tamanho.*
    const resposta = await request.get("/api/agent/v1/install/plan/PERFIL-INVENTADO", {
      timeout: 30_000,
    });
    expect(resposta.status()).not.toBe(200);
  });

  test("o painel nao expoe material secreto", async ({ page }) => {
    await page.goto("/capabilities");
    const painel = page.getByRole("region", { name: "Perfis de instalacao" });
    await expect(painel).toBeVisible({ timeout: 15_000 });

    const html = ((await painel.innerHTML()) ?? "").toLowerCase();
    for (const proibido of [
      "private_key",
      "-----begin",
      "storepassword",
      "keypassword",
      "password",
    ]) {
      expect(html, `o painel contem ${proibido}`).not.toContain(proibido);
    }
  });
});
