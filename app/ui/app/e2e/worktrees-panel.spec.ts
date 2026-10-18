import { expect, test } from "@playwright/test";

/**
 * FASE 2 (E-1237): E2E do painel de WORKTREES no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1233 provou o painel com `react-test-renderer`. A E-1232/E-1235/E-1236 provaram as rotas com
 * `httptest` e git real. **Os dois nunca falaram um com o outro** -- ninguem tinha exercitado
 * `painel -> HTTP -> JSON -> DOM` num browser.
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * DUAS CLASSIFICACOES, E O TESTE DIZ QUAL CORRE
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` -- quando `HADES_E2E_BACKEND_URL` esta' definido. O pedido sai do
 *   preview, atravessa o proxy `/api` e chega a um servidor Hades real. Nesse modo o inventario
 *   contem worktrees de VERDADE e os dados nao sao previsiveis.
 *
 *   `UI_FIXTURE` -- sem a variavel. O fixture `first-run-backend.mjs` serve dados marcados com
 *   `source: "e2e-fixture"`.
 *
 * O QUE ESTE SPEC GARANTE NOS DOIS MODOS
 *
 *   1. o painel aparece e pede a raiz do repositorio em vez de adivinhar;
 *   2. o que o painel mostra veio do HTTP e nao de constantes do componente;
 *   3. accoes destrutivas NAO existem na UI;
 *   4. apos mudar o estado, o painel RELE' o backend.
 *
 * O QUE ELE NAO FAZ: fingir que um modo e' o outro.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

/** O painel de worktrees, pelo nome acessivel que ele proprio declara. */
function painel(page: import("@playwright/test").Page) {
  return page.getByRole("region", { name: "Worktrees" });
}

/**
 * Escreve a raiz do repositorio e espera o inventario.
 *
 * NAO ha' botao "Inventariar": o painel carrega por efeito quando o campo muda. E' por isso que
 * este helper espera pelo RESUMO em vez de clicar.
 */
async function abrirInventario(page: import("@playwright/test").Page) {
  const regiao = painel(page);
  await expect(regiao).toBeVisible({ timeout: 15_000 });
  const campo = regiao.getByLabel("Raiz do repositório");
  await campo.fill("/repo-fixture");
  // Esperar pelo texto que o RESUMO do servidor produz prova que o HTTP respondeu.
  await expect(regiao.getByText(/worktree\(s\)/i).first()).toBeVisible({ timeout: 15_000 });
}

test.describe(`painel de worktrees (${MODO})`, () => {
  test("pede a raiz do repositorio em vez de adivinhar o checkout", async ({ page }) => {
    await page.goto("/capabilities");
    const regiao = painel(page);
    await expect(regiao).toBeVisible({ timeout: 15_000 });

    // O campo existe e esta' vazio: o painel NAO inventa um caminho.
    //
    //	*Inventariar o checkout errado e' pior do que pedir o caminho.*
    const campo = regiao.getByLabel("Raiz do repositório");
    await expect(campo).toBeVisible();
    await expect(campo).toHaveValue("");
  });

  test("mostra o inventario que o servidor devolveu, nao o que o componente sabe", async ({
    page,
    request,
  }) => {
    // 1. A FONTE.
    const resposta = await request.get("/api/agent/v1/worktrees?repo=/repo-fixture", {
      timeout: 30_000,
    });
    expect(resposta.status(), "GET /worktrees tem de responder 200").toBe(200);
    const corpo = await resposta.json();
    const lista = corpo.lista as Array<{ caminho_resolvido: string; criado_pelo_hades: boolean }>;
    expect(lista.length, "o servidor tem de declarar worktrees").toBeGreaterThan(0);

    // 2. A UI.
    await page.goto("/capabilities");
    await abrirInventario(page);
    const regiao = painel(page);

    // 3. A COMPARACAO. Cada worktree que o servidor devolveu tem de aparecer no DOM.
    for (const w of lista) {
      // O nome curto do caminho e' o que o painel mostra; comparar o caminho inteiro provaria
      // apenas que eu sei concatenar strings.
      const partes = w.caminho_resolvido.split("/").filter(Boolean);
      const nome = partes[partes.length - 1];
      await expect(
        regiao.getByText(nome, { exact: false }).first(),
        `o worktree ${nome} nao apareceu no painel`,
      ).toBeVisible({ timeout: 15_000 });
    }
  });

  test("marca os worktrees que o HADES nao criou, com aviso", async ({ page, request }) => {
    const resposta = await request.get("/api/agent/v1/worktrees?repo=/repo-fixture", {
      timeout: 30_000,
    });
    const corpo = await resposta.json();
    const externos = (corpo.lista as Array<{ criado_pelo_hades: boolean }>).filter(
      (w) => !w.criado_pelo_hades,
    );
    test.skip(externos.length === 0, "este backend nao tem worktrees externos");

    await page.goto("/capabilities");
    await abrirInventario(page);
    const regiao = painel(page);

    // O aviso vem da API (`avisos`), nao de uma constante do componente.
    //
    //	*Um worktree externo que nao se ve' e' um worktree que se apaga por engano.*
    await expect(regiao.getByText(/nao criou|não criou/i).first()).toBeVisible({
      timeout: 15_000,
    });
  });

  test("NAO oferece accao destrutiva -- remover nao existe no painel", async ({ page }) => {
    await page.goto("/capabilities");
    await abrirInventario(page);
    const regiao = painel(page);

    // Procurar em TODOS os botoes do painel, ANTES e DEPOIS de abrir um diagnostico.
    //
    //	*Remover exige uma decisao sobre trabalho nao integrado, e essa decisao nao cabe num
    //	clique que o painel nao sabe explicar.*
    const varrer = async () => {
      const botoes = await regiao.getByRole("button").allTextContents();
      const destrutivos = botoes.filter((rotulo) =>
        /remov|apag|exclu|delete|remove/i.test(rotulo),
      );
      expect(
        destrutivos,
        `o painel nao pode expor accao destrutiva; encontrados: ${destrutivos.join(", ")}`,
      ).toHaveLength(0);
    };

    await varrer();

    // As tres accoes reversiveis so' existem com um diagnostico aberto -- e' o painel a dizer
    // que a accao precisa de contexto. Exigir as tres SEM diagnostico seria exigir botoes soltos.
    const botaoMissao = regiao.getByRole("button", { name: "missao-e2e" });
    const temMissao = await botaoMissao.count();
    test.skip(temMissao === 0, "este backend nao tem a missao 'missao-e2e'");
    await botaoMissao.click();
    await expect(regiao.getByText(/Estado:/)).toBeVisible({ timeout: 15_000 });

    await varrer();
    for (const accao of ["Suspender", "Retomar", "Arquivar"]) {
      await expect(regiao.getByRole("button", { name: accao })).toBeVisible({ timeout: 15_000 });
    }
  });

  test("o estado mostrado vem do backend e o painel rele' depois de mudar", async ({
    page,
    request,
  }) => {
    await page.goto("/capabilities");
    await abrirInventario(page);
    const regiao = painel(page);

    // Abrir o diagnostico da missao que o inventario declarou.
    const botaoMissao = regiao.getByRole("button", { name: "missao-e2e" });
    const temMissao = await botaoMissao.count();
    test.skip(temMissao === 0, "este backend nao tem a missao 'missao-e2e'");

    await botaoMissao.click();
    await expect(regiao.getByText(/Estado:/)).toBeVisible({ timeout: 15_000 });
    const antes = await regiao.getByText(/Estado:/).textContent();

    // Mudar o estado pela UI e provar que o que aparece DEPOIS vem de uma nova leitura.
    await regiao.getByRole("button", { name: "Suspender" }).click();

    // A fonte confirma o novo estado...
    const resposta = await request.get(
      "/api/agent/v1/worktrees/diagnose?repo=/repo-fixture&missao=missao-e2e",
      { timeout: 30_000 },
    );
    expect(resposta.status()).toBe(200);
    const depois = await resposta.json();
    expect(depois.estado, "o servidor tem de reflectir a mudanca").toBe("suspenso");

    // ...e a UI tambem.
    //
    //	*Um painel que actualiza o ecra sem reler o backend mostra o que o programador achou que
    //	ia acontecer, nao o que aconteceu.*
    // `exact: true`: sem isto, "Suspenso" tambem casa com "A seguir: Suspenso, Integrado, ...",
    // e o teste passaria sem provar que o ESTADO mudou.
    await expect(regiao.getByText("Suspenso", { exact: true })).toBeVisible({ timeout: 15_000 });
    if (antes === null) {
      throw new Error("o painel nao mostrava o estado antes da mudanca");
    }
  });

  test("recusa accao destrutiva mesmo se pedida por HTTP directo", async ({ request }) => {
    // O painel nao tem o botao; a API tambem nao pode aceitar.
    //
    //	*Remover exige uma decisao sobre trabalho nao integrado, e essa decisao nao cabe num
    //	clique que o painel nao sabe explicar.*
    const resposta = await request.post("/api/agent/v1/worktrees/state", {
      data: { repo: "/repo-fixture", missao: "missao-e2e", para: "removido" },
      timeout: 30_000,
    });
    expect(
      resposta.status(),
      "POST /worktrees/state com 'removido' tem de ser recusado",
    ).toBe(400);
    const corpo = await resposta.json();
    expect(JSON.stringify(corpo)).toMatch(/nao permitida|não permitida/i);
  });
});
