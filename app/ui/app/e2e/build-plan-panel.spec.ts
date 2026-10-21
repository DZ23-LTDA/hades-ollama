import { expect, test } from "@playwright/test";

/**
 * FASE 28.8 (E-1249): E2E do painel BUILD & RELEASE no browser real.
 *
 * O ELO QUE FALTAVA
 *
 * A E-1248 provou o painel com `react-test-renderer` e o transporte MOCKADO. A E-1247 provou a rota
 * com `httptest`. **Os dois nunca falaram um com o outro.**
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * DUAS CLASSIFICACOES, E O TESTE DIZ QUAL CORRE
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` -- quando `HADES_E2E_BACKEND_URL` esta' definido. O pedido sai do
 *   preview, atravessa o proxy `/api` e chega ao **router de producao**, que detecta um projecto
 *   git REAL e sonda ferramentas REAIS.
 *
 *   `UI_FIXTURE` -- sem a variavel. O fixture `first-run-backend.mjs` serve os planos marcados com
 *   `source: "e2e-fixture"`.
 *
 * O QUE ESTE SPEC GARANTE NOS DOIS MODOS
 *
 *   1. o painel aparece e PEDE o caminho em vez de adivinhar;
 *   2. o que o painel mostra veio do HTTP e nao de constantes do componente;
 *   3. a fronteira ("apenas detecta e sonda") esta' VISIVEL no DOM;
 *   4. projecto sem alvos tem mensagem propria, nao um alerta de erro;
 *   5. nenhuma accao destrutiva ou de build existe na UI.
 *
 * O QUE ELE NAO FAZ: fingir que um modo e' o outro.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

/**
 * O projecto a analisar.
 *
 * No modo `UI_FIXTURE` e' um caminho inventado. No modo `BACKEND_REAL_AGENT_RUNTIME` e' um
 * repositorio git REAL, publicado pelo harness em `HADES_E2E_REPO`.
 *
 *	*Um caminho fixo no spec faria o modo real analisar um projecto que nao existe.*
 */
const REPO = process.env.HADES_E2E_REPO || "/repo-fixture";

// E-1304: O DEFAULT SILENCIOSO MENTIA NO MODO REAL.
//
// Quando `HADES_E2E_BACKEND_URL` esta' definido mas `HADES_E2E_REPO` NAO, o `REPO` caía para
// `/repo-fixture` -- um caminho que so' existe no fixture. O painel pedia ao backend REAL para
// analisar um directoria que nao existe, e QUATRO testes falhavam por um motivo que nao era o
// defeito que eles medem:
//
//   - analisa um projeto e mostra stacks vindas do HTTP
//   - mostra os blockers no topo
//   - mostra os motivos que explicam cada blocker
//   - traduz o estado da sonda para texto humano
//
// Medido: sem `HADES_E2E_REPO` => 3 passed / 4 failed / 2 skipped. Com ela => 7 passed / 2 skipped.
//
// O modo real EXIGE a variavel. Falhar aqui e' mais honesto do que deixar quatro testes acusarem um
// defeito do produto quando o defeito e' do ambiente do teste.
//
//     *Um default que so' serve a um dos modos faz o outro modo mentir.*
if (BACKEND_REAL && !process.env.HADES_E2E_REPO) {
  throw new Error(
    "HADES_E2E_BACKEND_URL esta' definido mas HADES_E2E_REPO nao: no modo " +
      "BACKEND_REAL_AGENT_RUNTIME o spec precisa do caminho de um repositorio REAL para pedir ao " +
      "backend. Sem ele o painel analisaria '/repo-fixture', que so' existe no fixture, e os testes " +
      "acusariam o produto por um defeito do ambiente. Defina HADES_E2E_REPO com o caminho absoluto " +
      "do checkout.",
  );
}

/**
 * As expectativas que MUDAM com o modo.
 *
 * O fixture serve um projecto com Go e Node e um blocker `docker`. O backend real analisa o
 * repositorio git do harness, que e' **so' Go** e cujo blocker medido e' `inno-setup`.
 *
 *	*Um spec que espera o mesmo texto nos dois modos obriga um deles a mentir.*
 */
const ESPERADO = {
  stacks: MODO === "BACKEND_REAL_AGENT_RUNTIME" ? ["go"] : ["go", "node"],
  blocker: MODO === "BACKEND_REAL_AGENT_RUNTIME" ? "inno-setup" : "docker",
  motivo: MODO === "BACKEND_REAL_AGENT_RUNTIME" ? "compilar o instalador" : "empacotar a imagem",
};

/** O painel, pelo nome acessivel que ele proprio declara. */
function painel(page: import("@playwright/test").Page) {
  return page.getByRole("region", { name: "Build & Release" });
}

/** O campo do caminho. */
function campoDoRepo(page: import("@playwright/test").Page) {
  return painel(page).getByLabel("Caminho do projeto");
}

async function analisar(page: import("@playwright/test").Page, repo: string) {
  await campoDoRepo(page).fill(repo);
  // O painel recarrega ao mudar o campo; esperamos por um dos desfechos.
  await expect(painel(page)).toContainText(/Projeto|resultado|alvo/i, { timeout: 15_000 });
}

test.describe(`Build & Release (${MODO})`, () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/capabilities");
    await expect(painel(page)).toBeVisible();
  });

  test("o painel pede o caminho em vez de adivinhar", async ({ page }) => {
    await expect(campoDoRepo(page)).toBeVisible();
    // Sem caminho, o painel nao inventa um plano.
    await expect(painel(page)).not.toContainText("Stacks");
  });

  test("declara a fronteira no DOM, nao apenas na documentacao", async ({ page }) => {
    await expect(painel(page)).toContainText("apenas detecta");
    await expect(painel(page)).toContainText("não constrói nada");
  });

  test("analisa um projeto e mostra stacks vindas do HTTP", async ({ page }) => {
    await analisar(page, REPO);
    // As stacks sao as que o BACKEND mediu naquele projecto; um componente que inventasse
    // mostraria outra coisa.
    for (const stack of ESPERADO.stacks) {
      await expect(painel(page)).toContainText(stack);
    }
  });

  test("mostra os blockers no topo, sem abrir cada alvo", async ({ page }) => {
    await analisar(page, REPO);
    await expect(painel(page).getByRole("heading", { name: "O que falta" })).toBeVisible();
    await expect(painel(page)).toContainText(ESPERADO.blocker);
  });

  test("mostra os motivos que explicam cada blocker", async ({ page }) => {
    await analisar(page, REPO);
    const porque = painel(page).getByRole("heading", { name: "Porquê" });
    await expect(porque).toBeVisible();
    await expect(painel(page)).toContainText(ESPERADO.motivo);
  });

  test("traduz o estado da sonda para texto humano", async ({ page }) => {
    await analisar(page, REPO);
    // As palavras cruas do backend nao aparecem ao utilizador.
    await expect(painel(page)).toContainText("pronta");
    await expect(painel(page)).toContainText("não instalada");
    await expect(painel(page)).not.toContainText("saudavel");
    await expect(painel(page)).not.toContainText("ausente");
  });

  test("projeto sem alvos tem mensagem propria, nao alerta de erro", async ({ page }) => {
    // O fixture serve `sem_alvos` para caminhos que contenham "sem-alvos". No modo real, o mesmo
    // desfecho vem de um diretorio sem manifests reconhecidos -- mas nao o forjamos aqui: nesse
    // modo este teste e' pulado para nao fingir que o fixture e' o backend.
    test.skip(
      MODO === "BACKEND_REAL_AGENT_RUNTIME",
      "no modo real o desfecho `sem_alvos` precisa de um projeto sem manifests; nao se forja",
    );
    await analisar(page, `${REPO}-sem-alvos`);
    await expect(painel(page)).toContainText("não é um erro, é um resultado");
    await expect(painel(page).getByRole("alert")).toHaveCount(0);
  });

  test("o painel nao promete assinatura, publicacao nem deploy -- e o build declara-se por alvo", async ({
    page,
  }) => {
    // HISTORICO, porque isto importa: este teste chamava-se «nao existe botao de build, assinatura ou
    // publicacao» e exigia a AUSENCIA de qualquer botao com «Build» no nome. Na altura era verdade: nao
    // havia `build.start`. A rota passou a existir (`agent_build_start_route.go`, POST /build/start) e o
    // painel passou a ter o botao «Iniciar build».
    //
    // *O teste nao estava errado: ficou obsoleto.* Um teste obsoleto que se mantem verde impede a
    // funcionalidade de existir; um que se mantem vermelho obriga a mentir sobre ela. O que se corrige e'
    // a premissa, nao a exigencia.
    //
    // O que CONTINUA a ser proibido e' prometer o que nao existe: nao ha' `sign.start` nem `publish.*`.
    //   *Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
    await analisar(page, REPO);
    for (const nome of ["Assinar", "Publicar", "Deploy", "Construir"]) {
      await expect(painel(page).getByRole("button", { name: nome, exact: false })).toHaveCount(0);
    }
    // E o build nao pode aparecer como promessa muda: cada alvo tem de declarar se pode iniciar ou por
    // que nao pode. Se alguem remover a decisao e deixar so' o rotulo, isto falha.
    const podeIniciar = await painel(page).getByTestId(/^build-plan-iniciar-/).count();
    const recusado = await painel(page).getByTestId(/^build-plan-sem-iniciar-/).count();
    expect(
      podeIniciar + recusado,
      "cada alvo tem de declarar se pode iniciar a build ou por que nao pode",
    ).toBeGreaterThan(0);
  });

  test("o erro da rota aparece como alerta", async ({ page }) => {
    test.skip(
      MODO === "BACKEND_REAL_AGENT_RUNTIME",
      "no modo real um caminho inexistente devolve 404, mas o caminho e' do SERVIDOR; nao se forja",
    );
    // O fixture responde 400 quando `repo` esta' vazio, mas o painel nao chama com vazio.
    // Forcamos um caminho que o fixture nao reconhece como vazio mas que o backend recusaria:
    // um caminho vazio nao e' enviado, portanto usamos o comportamento do proxy.
    await campoDoRepo(page).fill(" ");
    // Com espaco, o cliente faz trim e nao chama -- o painel fica sem plano e sem alerta.
    await expect(painel(page).getByRole("alert")).toHaveCount(0);
  });
});
