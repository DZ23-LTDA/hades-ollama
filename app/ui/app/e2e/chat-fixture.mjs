/**
 * Fixture de E2E para o CHAT (familia `/api/v1/chat/...`).
 *
 * PORQUE E' QUE ISTO EXISTE
 * A cadeia `E-917` -> `E-926` mediu, em oito rondas, que o `chat-ui-real.spec.ts` falhava
 * com **`0`** pedidos ao chat e **`10`** respostas **`501`**. A causa final (`E-924`/`E-925`)
 * nao era um defeito de produto: o `hades serve` responde **`501` DE PROPOSITO** naquela
 * familia (`server/desktop_local_routes.go:141-145`: *«a persistencia real do chat vive na
 * webview (`app/ui/ui.go`) e **nao** se duplica aqui - **proibido sem ADR**»*), e o handler
 * real vive no **desktop**.
 *
 * ESTE FICHEIRO NAO SUBSTITUI O PRODUTO - ESPELHA-O
 * O §FASE-6 do prompt V3.0 exige: *«fixture deve espelhar o contrato e ser identificado como
 * fixture»*. As cinco pecas do contrato foram **medidas** em `E-926` e estao citadas abaixo
 * com `arquivo:linha`. Nenhuma foi inventada.
 *
 *   (1) rota      : POST /api/v1/chat/{id}      -> app/ui/ui.go:287
 *   (2) cabecalho : Content-Type: text/jsonl    -> app/ui/ui.go:718
 *                   (+ no-cache :719, keep-alive :720, chunked :721)
 *   (3) o `"new"` : id literal "new" CRIA o chat (uuid v7) e devolve o id
 *                                                 -> app/ui/ui.go:735-744
 *   (4) corpo     : responses.ChatRequest        -> app/ui/ui.go:746-747
 *   (5) resposta  : STREAM de eventos JSONL      -> app/ui/ui.go:718
 *                   o cliente le com `parseJsonlFromResponse` -> src/api.ts:459
 *                   e o evento e' `{ eventName: "chat", content }` -> ModelComparisonPage.test.tsx:140
 *
 * A CERCA (PORQUE E' QUE NAO USA `Referer` COMO OS OUTROS FIXTURES)
 * O `companion-fixture` e o `sala-fixture` isolam-se pela `mission_id` no `Referer`, porque
 * sao servidos **dentro de uma missao**. O `chat-ui-real.spec.ts` faz `page.goto("/")` **sem
 * missao** (`:50`) e preenche o composer da pagina inicial.
 * => a cerca aqui e' a **ROTA**: `POST /api/v1/chat/:id` e' exclusiva desta familia e nenhum
 * outro fixture a serve. Um fixture sem cerca devolveria `200` a rotas alheias; este
 * devolve `null` para tudo o que nao seja `/api/v1/chat/...`.
 */

/** O modelo que o `chat-ui-real.spec.ts` escolhe (`:74`) e cujo nome vai no `aria-label`. */
export const CHAT_E2E_MODELO = "llama3.2:3b";

/**
 * CATALOGO (`GET /api/tags`) — a pre-condicao MEDIDA do spec, e nao um extra.
 *
 * PORQUE E' QUE ESTA ROTA VIVE AQUI
 * O `chat-ui-real.spec.ts` falhava em `:73` (`abridor.click()`) **antes** de chegar ao chat,
 * com `waiting for getByRole('button', { name: /^[\w.\-]+:[\w.\-]+$/ })` - *o `ModelPicker`
 * ficava no placeholder porque **nao havia modelo nenhum**.
 *
 * A cadeia medida (`E-928`):
 *   `getModels` (src/api.ts:186) **nao** usa `fetch` - usa o **SDK do Ollama**
 *   (`await ollama.list()`, :188)
 *   -> `ollamaClient` (src/lib/ollama-client.ts:9) e' `new Ollama({ host: OLLAMA_HOST })`
 *   -> `OLLAMA_HOST` (src/lib/config.ts:15) em **producao** e' `window.location.origin`
 *      (o `preview` em :4173, NAO o backend)
 *   -> o `vite.config.ts:86-88` proxia `/api` para `OLLAMA_PROXY_TARGET`
 *   -> que o `playwright.config.ts:71` fixa em `BACKEND_URL`
 *   => logo `GET /api/tags` **chega a este processo**, e nenhum fixture o servia (404).
 *
 * O FORMATO NAO FOI INVENTADO: foi lido do Ollama real deste host
 * (`GET http://127.0.0.1:11434/api/tags`), cujos modelos tem os campos
 * `name, model, modified_at, size, digest, details, capabilities` - e o `api.ts` usa
 * `m.name` (:206), `m.details.families` (:192), `m.digest` (:225/:246) e `m.modified_at` (:247).
 */
export const CHAT_E2E_CATALOGO = [
  {
    name: CHAT_E2E_MODELO,
    model: CHAT_E2E_MODELO,
    modified_at: "2026-01-01T00:00:00Z",
    size: 2019393189,
    digest: "a80c4f17acd55265feec403c7aef86be0c25983ab279d83f3bcd3abbcb5b8b72",
    details: {
      parent_model: "",
      format: "gguf",
      family: "llama",
      families: ["llama"],
      parameter_size: "3.2B",
      quantization_level: "Q4_K_M",
    },
    capabilities: ["completion", "tools"],
  },
];

/** Texto que o spec escreve no composer (`:86`). */
export const CHAT_E2E_PROMPT = "Diga apenas a palavra: pronto";

/**
 * O que o fixture responde como conteudo do assistente.
 *
 * E' **texto fixo e rotulado**: o spec assere que o `<article>` existe e que o texto **nao
 * esta' vazio** (`:102-104`) - nao assere o valor gerado por um modelo. Um fixture nunca
 * pode alegar que um modelo gerou isto.
 */
export const CHAT_E2E_RESPOSTA = "pronto";

/**
 * @param {{ sendJSON?: Function, texto?: string }} [opcoes]
 * @returns {(pedido: { route: string, request: any, response: any }) => Promise<number|null>}
 */
export function createChatFixture({
  sendJSON,
  texto = CHAT_E2E_RESPOSTA,
  catalogo = CHAT_E2E_CATALOGO,
  conversas = [],
} = {}) {
  /**
   * A GUARDA QUE EU NAO PUS NA 1.a VERSAO, e que o fixture ao lado ja' tinha.
   *
   * O `sala-fixture.mjs:68-72` escreve: *«createSalaFixture exige `sendJSON`. Sem ele o
   * fixture nao consegue responder e **falharia em silencio**.»*
   *
   * E foi exactamente isso que aconteceu: sem `sendJSON` no destructuring, a referencia
   * `sendJSON` no corpo **nao existe**, e `typeof variavelInexistente` devolve `"undefined"`
   * **sem lancar** (a unica excepcao do `typeof`). Logo o `if` era sempre falso e o handler
   * devolvia `null` - *o pedido seguia na cadeia e o stub respondia 404, sem uma unica pista*.
   *
   * Medido: `GET /api/tags -> status=null` e `[e2e-backend] GET /api/tags -> 404` **sete** vezes.
   */
  if (typeof sendJSON !== "function") {
    throw new Error(
      "createChatFixture exige `sendJSON`. Sem ele o fixture nao consegue responder ao catalogo e falharia em silencio.",
    );
  }

  /**
   * O servidor de E2E (`first-run-backend.mjs`) ja' tem um `sendJSON`, mas o chat precisa de
   * escrever um **stream** - e o `sendJSON` fecha com `response.end()` depois de um unico
   * `JSON.stringify`. Por isso o stream e' escrito aqui, com os MESMOS cabecalhos.
   */
  const escreverStream = (response, status, eventos) => {
    response.writeHead(status, {
      // `app/ui/ui.go:718-721`, na mesma ordem.
      "Content-Type": "text/jsonl",
      "Cache-Control": "no-cache",
      Connection: "keep-alive",
      "Transfer-Encoding": "chunked",
    });
    for (const evento of eventos) {
      response.write(`${JSON.stringify(evento)}\n`);
    }
    response.end();
  };

  return async function handleChatRequest({ route, request, response }) {
    /**
     * O METODO E O CAMINHO VEEM DO `route`, NAO DO `request`.
     * E' o contrato documentado do dispatcher (`first-run-backend.mjs:150` monta
     * `` `${request.method} ${url.pathname}` ``, com o **pathname** - logo sem query string)
     * e e' o que o `sala-fixture.mjs:96` e o `studio-builder-fixture.mjs` ja' fazem.
     */
    const [metodo, ...partesDoCaminho] = String(route || "").split(" ");
    const caminho = partesDoCaminho.join(" ");

    // --- CATALOGO (`GET /api/tags`): a pre-condicao do spec, medida em `E-928` ----------------
    // Vem **antes** da cerca `/api/v1/chat` de proposito: nao e' uma rota do chat, e' a rota
    // que faz o `ModelPicker` ter opcoes. A cerca continua a valer para tudo o resto.
    if (route === "GET /api/tags") {
      if (typeof sendJSON === "function") {
        sendJSON(response, 200, { models: catalogo });
        return 200;
      }
      return null;
    }

    // --- LISTA DE CONVERSAS (`GET /api/v1/chats`): sem ela o Chat NAO MONTA ------------------
    // Medido no `error-context.md` da execucao com a cadeia ja' ligada:
    //   `[e2e-backend] GET /api/v1/chats -> 404` **quatro** vezes,
    //   e o DOM mostrava **`Erro ao carregar as conversas`**
    //   => o `Chat` aborta a renderizacao e o `<article>` **nunca** aparece, mesmo com o
    //      `POST /api/v1/chat/new` a ter respondido **200**.
    // Contrato real: `app/ui/types.go:20-21` - `ChatsResponse { ChatInfos []ChatInfo `json:"chatInfos"` }`.
    // (E' o plural: o `api.ts:165` chama `/api/v1/chats`, e o `:176` chama `/api/v1/chat/{id}`.)
    if (route === "GET /api/v1/chats") {
      sendJSON(response, 200, { chatInfos: conversas });
      return 200;
    }

    // --- CERCA: so' a familia do chat. Qualquer outra rota segue na cadeia ------------------
    if (!caminho.startsWith("/api/v1/chat")) return null;

    // --- (3) o `new`: id literal que CRIA o chat (`app/ui/ui.go:735-744`) ---------------------
    // O `api.ts:423` envia `chatId === "new"` no primeiro turno. O produto troca-o por um UUID
    // v7; aqui usa-se um id fixo e rotulado, para o log dizer de onde veio.
    if (metodo === "POST" && caminho === "/api/v1/chat/new") {
      const chatId = "chat-e2e-fixture";
      /**
       * CONTRA PROVA (`E-930`): o `chatId` do cliente **nunca** saia de `"new"`, e a hipotese
       * era o nome do campo. O Go declara `ChatID *string \`json:"chatId,omitempty"\``
       * (`app/ui/types.go:81`), mas o **tipo gerado** do cliente pode expor outro nome.
       * Em vez de adivinhar, emito **os dois**: `chatId` (a tag JSON) e `chatID` (o nome Go).
       * *Se o `chatId` da app passar a mudar, o nome era um deles; se nao mudar, nao e' o nome.*
       * Ambos rotulados como parte da mesma contraprova - *nao e' contrato do produto*.
       */
      /**
       * E-931: O PRIMEIRO EVENTO TEM DE SER `chat_created` - *nao `chat`*.
       *
       * Medido em `Chat.tsx:165-174`:
       *   onChatEvent: (event) => {
       *     if (event.eventName === "chat_created" && event.chatId) {
       *       navigate({ to: "/c/$chatId", params: { chatId: event.chatId } });
       *     }
       *   }
       * => **so'** esse evento faz a app sair de `/c/new`. Sem ele o `chatId` fica `"new"`,
       * o `useChat` esta' desligado (`useChats.ts:67 enabled: chatId !== "new"`), o
       * `Chat.tsx:206` mostra o **empty state** e o `MessageList` (`:305`) **nunca** monta.
       * *Era exactamente isso que o `E-930` media no DOM cru.*
       *
       * O `api.ts:459-470` faz `switch (event.eventName)` com `download`/`error`/`default`,
       * logo o `chat_created` passa como `ChatEvent` e chega ao `onChatEvent` sem tratamento
       * especial. *O `chatId` e' o campo `ChatID` de `app/ui/types.go:80`.*
       */
      escreverStream(response, 200, [
        { eventName: "chat_created", chatId, chatID: chatId },
        { eventName: "chat", chatId, chatID: chatId, content: "" },
        { eventName: "chat", chatId, chatID: chatId, content: texto },
        { eventName: "chat", chatId, chatID: chatId, content: "", done: true },
      ]);
      return 200;
    }
    if (metodo === "POST" && /^\/api\/v1\/chat\/[^/]+$/.test(caminho)) {
      const chatId = caminho.slice("/api/v1/chat/".length);
      escreverStream(response, 200, [
        { eventName: "chat", chatId, chatID: chatId, content: "" },
        { eventName: "chat", chatId, chatID: chatId, content: texto },
        { eventName: "chat", chatId, chatID: chatId, content: "", done: true },
      ]);
      return 200;
    }

    // --- `GET /api/v1/chat/{id}`: leitura do chat (`api.ts:176`) -----------------------------
    // O contrato real devolve `ChatResponse { chat: Chat }` (`app/ui/types.go:24-25`).
    if (metodo === "GET" && /^\/api\/v1\/chat\/[^/]+$/.test(caminho)) {
      const chatId = caminho.slice("/api/v1/chat/".length);
      if (typeof sendJSON === "function") {
        sendJSON(response, 200, {
          chat: {
            id: chatId,
            title: "Chat E2E (fixture)",
            messages: [
              { role: "user", content: CHAT_E2E_PROMPT },
              { role: "assistant", content: texto },
            ],
          },
        });
        return 200;
      }
      return null;
    }

    // --- `PUT/POST .../rename` e `DELETE /api/v1/chat/{id}` (`api.ts:530`/`:544`) --------------
    if (/^\/api\/v1\/chat\/[^/]+(\/rename)?$/.test(caminho)) {
      if (typeof sendJSON === "function") {
        sendJSON(response, 200, {});
        return 200;
      }
      return null;
    }

    return null;
  };
}
