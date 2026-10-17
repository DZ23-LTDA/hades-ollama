import { expect, test } from "@playwright/test";

/**
 * Verifica o CAMINHO que a app usa, com um motor REAL por tras'.
 *
 * CLASSIFICACAO QUANDO CORRE: `PROVIDER_REAL` — o pedido sai do servidor de
 * preview (o mesmo que serve o SPA e faz proxy de `/api`), atravessa o proxy
 * definido em `vite.config.ts` e chega a um servidor Hades/Ollama verdadeiro a
 * servir um modelo local. Nao ha' stub nem fixture nesta rota.
 *
 * CLASSIFICACAO QUANDO NAO CORRE: `NOT_MEASURED` — o teste fica **skipped** e
 * di-lo na propria saida. Nao e' um PASS silencioso: a suite normal nao tem
 * motor, por isso nao mede isto, e um teste que fingisse medir seria pior do
 * que nenhum.
 *
 * COMO CORRER:
 *   # 1) Hades a servir um modelo local (runner ao lado do binario)
 *   $env:OLLAMA_HOST="127.0.0.1:11435"; Start-Process _tmp\hadesbin\hades.exe -ArgumentList serve
 *   # 2) apontar o E2E ao motor real (a variavel que o `playwright.config.ts` le)
 *   $env:HADES_E2E_BACKEND_URL="http://127.0.0.1:11435"
 *   npx playwright test e2e/chat-backend-real.spec.ts
 *
 * FRONTEIRA MEDIDA (ronda 90): o servidor Hades arranca em modo degradado neste
 * host (`agent runtime unavailable; serving core Ollama routes only`) e **nao
 * registra nenhuma rota `/api/agent/`**. Por isso este spec cobre CHAT — que
 * responde 200 — e NAO cobre projetos/missoes/agentes, que devolvem 404. Um spec
 * agentico aqui falharia por pre-condicao ambiental, nao por defeito do produto.
 */
const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;

test.describe("chat com backend REAL (motor local, sem fixture)", () => {
  test.skip(
    !BACKEND_REAL,
    "NOT_MEASURED: falta HADES_E2E_BACKEND_URL — a suite normal usa o fixture e nao mede um motor real.",
  );

  test("o proxy do SPA entrega uma resposta de modelo verdadeira", async ({
    request,
    baseURL,
  }) => {
    // O pedido sai pelo MESMO servidor (e portanto pelo mesmo proxy `/api`) que
    // serve a app. Nao tocamos no backend directamente: o que se mede aqui e' o
    // caminho app -> proxy -> motor.
    const resposta = await request.post("/api/chat", {
      data: {
        model: "llama3.2:3b",
        stream: false,
        messages: [{ role: "user", content: "Responda apenas com a palavra: pronto" }],
        options: { num_predict: 12 },
      },
      timeout: 180_000,
    });

    expect(resposta.status(), "o motor tem de responder 200").toBe(200);

    const corpo = await resposta.json();
    // Prova de que a resposta veio de um modelo e nao de um stub: tem de trazer
    // o modelo usado, a flag `done` e conteudo nao vazio.
    expect(corpo.model, "resposta tem de identificar o modelo").toContain("llama3.2");
    expect(corpo.done, "resposta tem de estar concluida").toBe(true);
    expect(
      typeof corpo.message?.content === "string" && corpo.message.content.trim().length > 0,
      "o modelo tem de devolver conteudo nao vazio",
    ).toBe(true);

    // O token gerado e' medido, nao presumido: se for 0, nao houve inferencia.
    expect(corpo.eval_count, "tem de haver tokens gerados").toBeGreaterThan(0);

    console.log(
      `[chat-backend-real] modelo=${corpo.model} tokens=${corpo.eval_count} ` +
        `conteudo=${JSON.stringify(corpo.message.content.trim())} origem=${baseURL}`,
    );
  });

  test("o motor real lista modelos (nao e' um servidor vazio)", async ({ request }) => {
    const resposta = await request.get("/api/tags", { timeout: 30_000 });
    expect(resposta.status()).toBe(200);
    const { models } = await resposta.json();
    expect(Array.isArray(models), "/api/tags tem de devolver uma lista").toBe(true);
    expect(models.length, "o motor tem de ter pelo menos um modelo").toBeGreaterThan(0);
    console.log(`[chat-backend-real] modelos vistos: ${models.length}`);
  });
});
