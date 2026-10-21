import { expect, test } from "@playwright/test";

/**
 * Prova E2E da SALA (complemento 48 — Cowork/Team Chat).
 *
 * **PORQUE E' QUE ESTE FICHEIRO EXISTE**: o `SalaLigada` declara, no proprio cabecalho, o limite que
 * este spec fecha:
 *
 *   «LIMITE DECLARADO (medido): o arnes de testes deste pacote **nao tem ambiente DOM** nem corre
 *    efeitos (`renderToStaticMarkup`), por isso **esta unidade nao tem teste local** — e nao a declaro
 *    `TESTED`. *O que a pode provar e' um **E2E de browser, que ainda nao existe**.»*
 *
 * E o §03 do `/goal` pede exactamente isto: *«Provar, **em navegador real**, a fiacao do componente ->
 * API -> runtime/store»*.
 *
 * CLASSIFICACAO: `LOCAL_BACKEND_REAL` — HTTP de verdade contra o stub do repositorio
 * (`e2e/sala-fixture.mjs`, registado em `e2e/first-run-backend.mjs`). ***Nao prova o backend Go em
 * producao***; prova a fiacao **UI -> HTTP -> contrato** ponta a ponta no browser.
 *
 * O QUE ESTE SPEC **NAO** PROVA, e fica declarado: *mensagem, solicitacao ao agente, tarefa e
 * actualizacao SSE*. O fixture serve `members` e `rooms`; estas quatro nao estao montadas na UI (a
 * `SalaLigada` -> `SalaPainel` -> `SalaMembros` mostra **membros**). *Chamar-lhes "provado" seria a
 * classe que esta missao derruba.*
 */

const MISSAO = "mis_sala_e2e";
const PROJECTO = "proj_sala_e2e";
const SALA = "sala_e2e";

/**
 * A Sala so' e' renderizada quando a aba de canvas e' `"artifacts"` **e** o URL traz `project_id` +
 * `room_id`:
 *   `AgenticSplitShell.tsx`  `) : activeCanvasTab === "artifacts" ? (`
 *                            `{projectID && roomID ? <SalaLigada ... /> : null}`
 * *Sem o clique na aba, a rota `/members` NUNCA e' pedida* — a mesma medicao que isolou a causa no
 * `companion.spec.ts`.
 *
 * O `Referer` da pagina passa a ser este URL, e e' isso que satisfaz o isolamento do fixture
 * (`mission_id=mis_sala_e2e`).
 */
async function abrirSala(page: import("@playwright/test").Page) {
  await page.goto(
    `/agentic?mission_id=${MISSAO}&project_id=${PROJECTO}&room_id=${SALA}`,
  );
  const aba = page.getByRole("button", { name: /Artefatos/i }).first();
  await aba.waitFor({ state: "visible", timeout: 20000 });
  await aba.click();
}

test.describe("Sala (complemento 48 / Cowork-Team Chat)", () => {
  test("monta e mostra os MEMBROS reais que o servidor devolveu", async ({ page }) => {
    await abrirSala(page);

    // O fixture devolve `SALA_MEMBROS_PADRAO` = Ana (humano) e Revisor (agente).
    // Se a UI inventasse a lista, estes nomes nao apareceriam.
    //
    // *Regex, e nao string exacta*: medido no diagnostico — o `SalaMembros` renderiza o nome
    // **dentro de um rotulo** (`pessoa Ana (ana)` / `agente Revisor (agente-revisor)`), e o
    // `getByText("Ana")` exige correspondencia do elemento inteiro. *O primeiro `getByText("Ana")`
    // falhou por isso, e nao por o produto nao mostrar a pessoa.*
    await expect(page.getByText(/Ana/).first()).toBeVisible({ timeout: 20000 });
    await expect(page.getByText(/Revisor/).first()).toBeVisible({ timeout: 20000 });
  });

  test("distingue PESSOA de AGENTE (o caso que o complemento existe para cobrir)", async ({
    page,
  }) => {
    await abrirSala(page);

    // `SalaMembros.tsx` rotula os dois tipos de forma diferente — *um agente nao pode passar por
    // pessoa*. Prova que o tipo vindo do contrato chega a' vista, e nao e' so' um nome numa lista.
    // *Medido no diagnostico*: o texto e' `pessoa Ana (ana)` e `agente Revisor (agente-revisor)`.
    await expect(page.getByText(/pessoa\s+Ana/).first()).toBeVisible({ timeout: 20000 });
    await expect(page.getByText(/agente\s+Revisor/).first()).toBeVisible({ timeout: 20000 });
  });

  test("SEM `room_id` nao monta painel nenhum (nao ha sala vazia a fingir)", async ({ page }) => {
    // O condicional e' `projectID && roomID`. Sem a sala, **nada** e' renderizado: nem um painel
    // vazio, nem um "a carregar" eterno. *Este teste e' o que impede a montagem de degradar para
    // um placeholder que mente.*
    await page.goto(`/agentic?mission_id=${MISSAO}&project_id=${PROJECTO}`);
    const aba = page.getByRole("button", { name: /Artefatos/i }).first();
    await aba.waitFor({ state: "visible", timeout: 20000 });
    await aba.click();

    // ⚠️ **`/pessoa Ana/` e nao `/Ana/`** — medido: `/Ana/` faz match com **«Analisar»**
    // (`Analisar imagem (visao) (image.analyze)` esta' na mesma pagina!), e o teste passava a medir
    // outra coisa. *E' a licao do `E-883` («um padrao largo pergunta se alguem **fala** disto») na sua
    // forma mais directa: `/Ana/` responde "existe este texto?", nao "existe a pessoa Ana?".*
    // O rotulo completo, esse, so' a Sala o produz.
    await expect(page.getByText(/pessoa\s+Ana/)).toHaveCount(0);
    await expect(page.getByText(/agente\s+Revisor/)).toHaveCount(0);
  });
});
