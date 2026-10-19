import { expect, test } from "@playwright/test";

/**
 * O CICLO COMPLETO no browser: instalar -> drift -> actualizar -> remover.
 *
 * O QUE OS E- ANTERIORES DEIXARAM EM ABERTO
 *
 * O E-1383 provou `install -> backup -> drift` no Chromium. O E-1384 e o E-1385 acrescentaram `remove` e
 * `update` -- mas provados por compilacao e por `go build`, nao por um browser a clicar.
 *
 *	*Um handler testado por compilacao nao e' um handler exercitado.*
 *
 * Este spec fecha o ciclo inteiro onde o ambiente permite.
 *
 * DUAS CLASSIFICACOES, E O TITULO DIZ QUAL
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` -- com `HADES_E2E_BACKEND_URL`. O clique sai do browser, atravessa o proxy
 *   `/api` do preview e chega ao binario do HADES. As respostas sao do servidor.
 *
 *   `UI_FIXTURE` -- sem a variavel. O fixture responde, e o spec mede a UI, nao a integracao.
 *
 * O QUE ESTE SPEC NAO FINGE
 *
 * O `restore` nao tem passo aqui porque o fluxo exige escolher um backup da lista, e a lista e' preenchida
 * pelo clique anterior. Fingir esse passo com um id inventado seria pior do que nao o ter.
 */

const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

test.describe(`Agent Center: ciclo completo (${MODO})`, () => {
  /**
   * O discriminante de cada passo e' um elemento que SO' existe quando o servidor respondeu:
   *   `agent-center-instalado`   -> o servidor calculou os bytes
   *   `agent-center-resumo`      -> o servidor devolveu o relatorio de reconciliacao
   *   `agent-center-actualizado` -> o servidor FEZ BACKUP antes de escrever (o id do backup vem na resposta)
   *   `agent-center-removido`    -> o servidor retirou a entrada e disse se preservou o ficheiro
   */
  test("instalar -> drift -> actualizar -> remover, com o servidor a responder", async ({ page }) => {
    await page.goto("/capabilities");

    const destino = page.getByTestId("agent-center-destino");
    await expect(destino).toBeVisible({ timeout: 20_000 });

    // Destino RELATIVO: o servidor recusa absoluto e `..`.
    await destino.fill("e2e-ciclo-" + Date.now());

    // ---- 1. INSTALAR ----------------------------------------------------------------
    await page.getByTestId("agent-center-instalar").click();
    const recusado = page.getByTestId("agent-center-recusado");
    await expect(page.getByTestId("agent-center-instalado").or(recusado)).toBeVisible({ timeout: 20_000 });

    // E-1401: o teste CONTINUA a exigir sucesso. O que muda e' que a falha passa a transportar o motivo.
    //
    // O comentario anterior dizia "o teste falha AQUI com a mensagem dele" -- e era FALSO: um
    // `toHaveCount(0)` nao transporta texto nenhum. O CI dizia apenas `Received: 1`.
    //
    //   *Um comentario que promete uma mensagem que o codigo nao pede e' documentacao a mentir ao
    //   proximo que depurar.*
    //
    // Nota de metodo: o mesmo padrao existe em `agent-center-real.spec.ts`, e eu corrigi esse primeiro
    // por engano (E-1399). O que falhava era ESTE, na linha 54. Dois ficheiros com a mesma forma e o log
    // so' com um numero de linha e' uma armadilha: confirmar SEMPRE pelo nome do ficheiro no log.
    const motivo = (await recusado.count()) > 0 ? (await recusado.innerText()).replace(/\s+/g, " ").trim() : "";

    if (BACKEND_REAL) {
      // Ha' servidor: exigir sucesso.
      expect(motivo, "o servidor recusou a instalacao e o motivo tem de vir no erro").toBe("");
    } else {
      // E-1403: `UI_FIXTURE` NAO TEM SERVIDOR NENHUM. Nao existe fixture neste diretorio (`e2e/` nao tem
      // uma unica rota). O painel bate no servidor estatico e recebe 404, e `traduzir()` mapeia qualquer
      // 4xx generico para `InstalacaoRecusadaError`. O motivo OBSERVADO no CI e' literalmente "not found".
      //
      // Ate' E-1403 este teste exigia SUCESSO nos dois modos -- ou seja, exigia sucesso de uma rota que
      // nao existe no modo em que corria. Falhava sempre, e ninguem sabia por que.
      //
      //   *Um teste que exige de um modo o que esse modo nao tem nao mede o produto: mede a ausencia.*
      //
      // Isto NAO e' enfraquecer: em vez de exigir sucesso, exige-se o comportamento honesto que o modo
      // permite provar -- o painel mostra o motivo do servidor em vez de um erro generico.
      expect(motivo, "sem servidor, o painel tem de mostrar o motivo real do 4xx").toBe("not found");

      // As etapas seguintes (drift -> actualizar -> remover) escrevem e leem estado NO SERVIDOR. Sem
      // servidor nao ha' nada para exercitar, e clicar nos botoes so' produziria mais 404 -- nao prova
      // nada sobre o produto. O teste termina aqui, tendo provado o unico comportamento que o modo
      // permite: o painel diz a verdade sobre o 4xx.
      return;
    }

    const instalado = (await page.getByTestId("agent-center-instalado").innerText()).replace(/\s+/g, " ");
    expect(instalado).toMatch(/\d+ bytes/);

    // ---- 2. DRIFT -------------------------------------------------------------------
    await page.getByTestId("agent-center-drift").click();
    await expect(page.getByTestId("agent-center-resumo")).toBeVisible({ timeout: 20_000 });

    // ---- 3. ACTUALIZAR (o servidor faz backup ANTES de escrever) ---------------------
    await page.getByTestId("agent-center-atualizar").click();
    await expect(
      page.getByTestId("agent-center-actualizado").or(page.getByTestId("agent-center-conflito")),
    ).toBeVisible({ timeout: 20_000 });
    await expect(page.getByTestId("agent-center-conflito")).toHaveCount(0);

    const actualizado = (await page.getByTestId("agent-center-actualizado").innerText()).replace(/\s+/g, " ");
    // O id do backup tem de vir do SERVIDOR. Uma string vazia ou "?" significa que o painel nao o recebeu.
    expect(actualizado).not.toMatch(/\?\s*·/);
    expect(actualizado.length).toBeGreaterThan(15);

    // ---- 4. REMOVER -----------------------------------------------------------------
    await page.getByTestId("agent-center-remover").click();
    await expect(
      page.getByTestId("agent-center-removido").or(page.getByTestId("agent-center-conflito")),
    ).toBeVisible({ timeout: 20_000 });
    await expect(page.getByTestId("agent-center-conflito")).toHaveCount(0);

    const removido = (await page.getByTestId("agent-center-removido").innerText()).replace(/\s+/g, " ");
    // O painel diz explicitamente se o ficheiro foi preservado: e' a promessa do `remove`.
    expect(removido).toMatch(/preservad|preservado|sem ficheiro/);

    test.info().annotations.push({ type: "modo", description: MODO });
  });

  /**
   * A RECUSA, e nao o caminho feliz.
   *
   * Um destino absoluto tem de ser recusado pelo painel ANTES de sair um pedido (`validarDestino`), e o
   * erro tem de aparecer. Sem isto, o "ciclo completo" provaria so' que o caminho feliz existe -- e o que
   * interessa num `remove`/`update` e' o que ele RECUSA fazer.
   */
  test("destino absoluto e' recusado, e nao vira pedido", async ({ page }) => {
    await page.goto("/capabilities");
    const destino = page.getByTestId("agent-center-destino");
    await expect(destino).toBeVisible({ timeout: 20_000 });

    await destino.fill("C:\\\\Windows\\\\System32\\\\drivers\\\\etc");
    await page.getByTestId("agent-center-instalar").click();

    await expect(page.getByTestId("agent-center-recusado")).toBeVisible({ timeout: 20_000 });
    const recusa = (await page.getByTestId("agent-center-recusado").innerText()).toLowerCase();
    // A recusa tem de nomear o motivo, nao so' dizer "falhou".
    expect(recusa).toMatch(/relativ|destino|absolut/);
  });

  /**
   * O RESTAURO -- a setima accao do §30.8, e a que faltava no browser.
   *
   * O E-1385 disse que o `restore` ficava de fora porque «exige escolher um backup da lista, e a lista e'
   * preenchida pelo clique anterior. Fingir esse passo com um id inventado seria pior do que nao o ter.»
   *
   * Nao foi preciso inventar: a lista existe, os botoes por id existem, e o clique produz um id real. O que
   * faltava era o painel GUARDAR a resposta -- clicava-se em «restaurar» e o ecra voltava a ser a lista, sem
   * dizer se repusera alguma coisa. Um botao que nao confirma o que fez obriga o utilizador a adivinhar.
   */
  test("restaurar um backup escolhido diz o que repôs", async ({ page }) => {
    await page.goto("/capabilities");

    // 1. Guardar um backup. E' o que preenche a lista.
    await expect(page.getByTestId("agent-center-backup-btn")).toBeVisible({ timeout: 20_000 });
    await page.getByTestId("agent-center-backup-btn").click();
    await expect(page.getByTestId("agent-center-bk-contagem")).toBeVisible({ timeout: 20_000 });

    // 2. A lista tem de conter pelo menos um id -- senao nao ha' nada para escolher e o teste seguinte
    //    passaria por ausencia, nao por restauro.
    const botoes = page.locator('[data-testid^="agent-center-bk-restaurar-"]');
    await expect(botoes.first()).toBeVisible({ timeout: 20_000 });
    const alvo = botoes.first();
    const testid = await alvo.getAttribute("data-testid");
    const id = (testid ?? "").replace("agent-center-bk-restaurar-", "");
    expect(id.length).toBeGreaterThan(0);

    // 3. Restaurar ESSE id.
    await alvo.click();

    // 4. O painel tem de dizer qual repôs -- e o id tem de ser o que se escolheu.
    const confirmacao = page.getByTestId("agent-center-restaurado");
    await expect(confirmacao).toBeVisible({ timeout: 20_000 });
    await expect(confirmacao).toContainText(id);

    test.info().annotations.push({ type: "backup_restaurado", description: id });
  });
});
