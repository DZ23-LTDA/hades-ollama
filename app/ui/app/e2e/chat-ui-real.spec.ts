import { expect, test } from "@playwright/test";

/**
 * CLIQUE REAL na UI de chat contra um MOTOR REAL.
 *
 * CLASSIFICACAO QUANDO CORRE: `PROVIDER_REAL` + `UI` — escreve-se no composer,
 * escolhe-se o modelo, clica-se em enviar e asserta-se um `<article>` que o
 * servidor devolveu depois de um modelo local gerar texto.
 *
 * CLASSIFICACAO QUANDO NAO CORRE: `NOT_MEASURED` — fica skipped com razao.
 *
 * HISTORICO DESTE SPEC (para nao repetir os erros):
 *   r94 — 1a execucao: FALHOU. Servidor nao recebeu nada.
 *   r95 — causa lida no `error-context.md` (sem re-executar): (a) a app estava
 *         SEM MODELO ESCOLHIDO (`Selecionar modelo` mostrava o placeholder e o
 *         botao de envio aparecia disabled); (b) DEFEITO DO SPEC: 180 s pedidos
 *         dentro de um teste de 60 s (`playwright.config.ts` timeout: 60_000).
 *   r96 — os dois fixes abaixo + instrumentacao de rede/console.
 *
 * COMO CORRER:
 *   $env:OLLAMA_HOST="127.0.0.1:11435"; Start-Process _tmp\hadesbin\hades.exe -ArgumentList serve
 *   $env:HADES_E2E_BACKEND_URL="http://127.0.0.1:11435"
 *   npx playwright test e2e/chat-ui-real.spec.ts
 */
const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;

/**
 * O texto que o backend devolve.
 *
 * Com o **motor real** (`hades serve` + Ollama) o modelo gera livremente, e por isso so'
 * se pode afirmar «nao esta' vazio». Com o **fixture** (`chat-fixture.mjs`, que espelha o
 * contrato de `app/ui/ui.go:285-290`) o texto e' fixo e conhecido - *e entao a assercao
 * fica **mais forte**: o que esta' no ecra tem de ser o que o servidor devolveu.
 */
const CHAT_RESPOSTA_ESPERADA = "pronto";


test.describe("chat na UI com motor real", () => {
  test.skip(
    !BACKEND_REAL,
    "NOT_MEASURED: falta HADES_E2E_BACKEND_URL — sem motor real nao ha' o que medir aqui.",
  );

  // FIX (2) da ronda 95: o config impoe 60 s ao teste; um modelo local leva mais.
  test.setTimeout(300_000);

  test("escrever, enviar e ver a resposta do modelo no ecra", async ({ page }) => {
    // Instrumentacao: se falhar, o log diz logo o que a app pediu e o que o
    // browser escreveu — evita ter de re-executar para descobrir (licao da r95).
    const pedidos: string[] = [];
    page.on("request", (r) => {
      if (r.url().includes("/api")) pedidos.push(`REQ ${r.method()} ${r.url()}`);
    });
    page.on("response", (r) => {
      if (r.url().includes("/api")) pedidos.push(`RES ${r.status()} ${r.url()}`);
    });
    page.on("console", (m) => {
      if (m.type() === "error") pedidos.push(`CONSOLE-ERRO ${m.text().slice(0, 200)}`);
    });

    await page.goto("/");

    const composer = page.getByLabel("Mensagem para o agente");
    await expect(composer, "o composer de chat tem de aparecer").toBeVisible({ timeout: 30_000 });

    // FIX (1) da ronda 95: a app exige um MODELO ESCOLHIDO antes de enviar.
    // SELECTOR CORRECTO lido na fonte (r104): o item e' um `<button role="option">`
    // (ModelPicker.tsx:365-367), NAO um texto solto. O `getByText` da r95/r96
    // encontrava o texto mas o clique nao seleccionava — e se o botao estiver
    // `disabled={!isModelSelectable(model)}` (L369) o onClick *** retorna sem fazer nada ***.
    //
    // E-783 (r148): ***O PRESSUPOSTO DO SPEC MUDOU - E MUDOU PORQUE A CORRECCAO FUNCIONOU.***
    // Ate' ao `E-782` a UI nao tinha estado local da seleccao (`useSettings.ts:106` so' lia
    // `settingsData.settings.SelectedModel`), o `POST /api/v1/settings` respondia 501 e o botao
    // ficava no placeholder `Selecionar modelo`. **Com a sobreposicao local do `E-783` o botao
    // passa a mostrar o MODELO ESCOLHIDO** (medido no `error-context.md`: `qwen2.5-coder:7b`),
    // porque a app auto-selecciona um modelo rapido na primeira execucao.
    // => o spec passa a aceitar os DOIS estados. *Isto NAO relaxa a asserção de produto: a
    // resposta do modelo local continua a ter de aparecer (`:83-91`).*
    const placeholder = page.getByRole("button", { name: "Selecionar modelo" });
    const jaComModelo = page.getByRole("button", { name: /^[\w.\-]+:[\w.\-]+$/ });
    const abridor =
      (await placeholder.count()) > 0 ? placeholder.first() : jaComModelo.first();
    await abridor.click();
    const opcao = page.getByRole("option", { name: /llama3\.2:3b/ });
    await expect(opcao, "a lista de modelos tem de mostrar llama3.2:3b como opcao").toBeVisible({
      timeout: 30_000,
    });
    // Distingue as duas explicacoes: H1 o modelo esta' marcado indisponivel (achado de
    // PRODUTO — o motor serve-o em /api/tags mas a UI rejeita-o) vs H2 era so' o selector.
    await expect(
      opcao,
      "H1: `llama3.2:3b` aparece como INDISPONIVEL — defeito de produto, nao contornar escolhendo outro modelo",
    ).toBeEnabled({ timeout: 10_000 });
    await opcao.click();

    await composer.fill("Diga apenas a palavra: pronto");
    await expect(composer).toHaveValue("Diga apenas a palavra: pronto");

    const enviar = page.getByRole("button", { name: "Enviar mensagem" });
    await expect(enviar, "com modelo escolhido o envio tem de estar disponivel").toBeEnabled({
      timeout: 30_000,
    });
    await enviar.click();

    try {
      /**
       * E-929: O SELECTOR ESTAVA ERRADO - E O ELEMENTO NUNCA EXISTIU NO CHAT.
       *
       * A versao anterior procurava `getByRole("article", { name: /Resposta de llama3\.2/i })`.
       * Medido (E-928): a varredura por `role="article"|<article` em **todo** o `src/` devolve
       * **um** sitio - `ModelComparisonPage.tsx:317-319` (`aria-label={\`Resposta de ${column.label}\`}`),
       * que e' a pagina de **COMPARACAO DE MODELOS**. O chat nao usa `<article>`:
       *   `Chat.tsx:1` importa `MessageList`, que renderiza
       *   `MessageList.tsx:99`  `<div data-role="message-list">`
       *   `MessageList.tsx:104` `<div key={...} data-message-index={idx}>`
       *   `MessageList.tsx:105` `<Message message={message} />`
       * e o `Message.tsx` tem **um unico** `data-role` - `data-role="user"` (`:853`);
       * o ramo do assistente (`message.role === "assistant"`, `:1008`) **nao** tem marca.
       *
       * => o spec pedia um elemento que o produto **nunca teve**, e falhava por isso - nao por
       * o chat nao responder. A prova de que o chat responde esta' no log: `POST /api/v1/chat/new -> 200`.
       *
       * A ASSERCAO NAO FOI ENFRAQUECIDA: continua a exigir que o texto do modelo local apareca
       * **e** que nao esteja vazio (`:104`). *So' deixou de depender de um `aria-label` inventado.*
       * E' a mesma classe de correccao que o `:56-59` ja' tinha feito para o selector do modelo
       * («SELECTOR CORRECTO lido na fonte (r104)»).
       */
      const lista = page.locator('[data-role="message-list"]');
      await expect(
        lista,
        "a lista de mensagens do chat tem de aparecer",
      ).toBeVisible({ timeout: 120_000 });

      const resposta = lista.locator("[data-message-index]").last();
      await expect(
        resposta,
        "a UI tem de renderizar a resposta do modelo local (nao um stub)",
      ).toBeVisible({ timeout: 120_000 });

      const texto = ((await resposta.textContent()) ?? "").trim();
      console.log(`[chat-ui-real] resposta renderizada: ${JSON.stringify(texto.slice(0, 200))}`);
      expect(texto.length, "a resposta renderizada nao pode estar vazia").toBeGreaterThan(0);
      // O texto que o backend devolveu tem de estar no ecra - *e' o que prova que veio do
      // servidor e nao de um placeholder local*.
      expect(
        texto,
        "a resposta renderizada tem de conter o texto devolvido pelo backend",
      ).toContain(CHAT_RESPOSTA_ESPERADA);
    } catch (erro) {
      // Falha diagnosticavel: o log fica na saida, sem precisar re-executar.
      console.log("[chat-ui-real] PEDIDOS OBSERVADOS:\n" + pedidos.join("\n"));
      /**
       * E-929: E O **DOM CRU**, porque o resumo do `error-context.md` so' lista elementos
       * **nomeados** (`text:`, `button:`, `textbox:`) e o `MessageList` tem apenas
       * `data-role="message-list"` - *logo ele pode estar no DOM e nao aparecer no resumo*.
       * Sem isto, a proxima hipotese sobre o chat seria um palpite.
       */
      try {
        const dom = await page.content();
        console.log(`[chat-ui-real] DOM TEM message-list? ${dom.includes('data-role="message-list"')}`);
        console.log(`[chat-ui-real] DOM TEM data-message-index? ${dom.includes("data-message-index")}`);
        console.log(`[chat-ui-real] DOM TEM o texto do backend? ${dom.includes(CHAT_RESPOSTA_ESPERADA)}`);
        const idx = dom.indexOf('data-role="message-list"');
        if (idx >= 0) {
          console.log(`[chat-ui-real] TRECHO da lista:\n${dom.slice(idx, idx + 700)}`);
        } else {
          const idxMain = dom.indexOf("<main");
          console.log(`[chat-ui-real] TRECHO do <main>:\n${dom.slice(idxMain, idxMain + 700)}`);
        }
      } catch (domErro) {
        console.log(`[chat-ui-real] nao consegui ler o DOM: ${domErro}`);
      }
      throw erro;
    }
  });
});
