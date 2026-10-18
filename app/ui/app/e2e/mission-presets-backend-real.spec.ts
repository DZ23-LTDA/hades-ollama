import { expect, test } from "@playwright/test";

/**
 * FASE 23 do adendo V3: o catalogo de presets de missao no picker.
 *
 * DUAS CLASSIFICACOES, e o teste diz qual corre:
 *
 *   `BACKEND_REAL_AGENT_RUNTIME` — quando `HADES_E2E_BACKEND_URL` esta' definido. O
 *   pedido sai do preview, atravessa o proxy `/api` e chega a um servidor Hades com o
 *   runtime de agente inicializado. Neste modo o spec compara a UI com a ROTA REAL.
 *
 *   `UI_FIXTURE` — sem a variavel. O fixture `first-run-backend.mjs` serve dois presets
 *   com ids `fixture-*`, declaradamente falsos. Neste modo o spec mede a UI: contagem,
 *   renderizacao, seleccao e a opcao "Sem preset". Nao mede o catalogo real, e di-lo.
 *
 * O QUE ESTE SPEC NAO FAZ: fingir que um modo e' o outro. Um teste que dissesse
 * "catalogo verificado" correndo contra um fixture seria pior do que nenhum.
 *
 * O DISCRIMINADOR (igual nos dois modos)
 *
 * O numero de radios tem de ser EXACTAMENTE o numero de presets devolvidos pelo
 * servidor + 1 (o "Sem preset"). Se a UI passar a inventar, omitir ou duplicar
 * presets, falha aqui. *Um teste de strings nao e' um teste de comportamento; este
 * compara a UI com a fonte que a alimenta.*
 */
const BACKEND_REAL = process.env.HADES_E2E_BACKEND_URL;
const MODO = BACKEND_REAL ? "BACKEND_REAL_AGENT_RUNTIME" : "UI_FIXTURE";

type PresetDoServidor = { id: string; name?: string };

test.describe(`presets de missao (${MODO})`, () => {
  test("o picker mostra exactamente os presets que o servidor devolveu", async ({
    page,
    request,
  }) => {
    // 1. A FONTE. Leio a rota que alimenta o picker, seja real ou fixture.
    const resposta = await request.get("/api/agent/v1/presets", { timeout: 30_000 });
    expect(resposta.status(), "GET /api/agent/v1/presets tem de responder 200").toBe(200);

    const corpo = await resposta.json();
    expect(Array.isArray(corpo.presets), "a rota tem de devolver uma lista").toBe(true);
    const presets = corpo.presets as PresetDoServidor[];
    expect(presets.length, "o catalogo tem de ter presets").toBeGreaterThan(0);

    // No modo fixture, a rota tem de se identificar como fixture. *Um fixture que nao
    // se declara passa a ser confundido com o produto.*
    if (MODO === "UI_FIXTURE") {
      expect(
        corpo.source,
        "o fixture tem de se identificar como `e2e-fixture` para nao ser confundido com o catalogo real",
      ).toBe("e2e-fixture");
      for (const preset of presets) {
        expect(
          preset.id.startsWith("fixture-"),
          `no modo fixture os ids tem de ser fixture-*, veio ${preset.id}`,
        ).toBe(true);
      }
    } else {
      // No modo real o catalogo embutido nao tem ids de fixture.
      for (const preset of presets) {
        expect(
          preset.id.startsWith("fixture-"),
          `o catalogo REAL nao pode ter ids de fixture, veio ${preset.id}`,
        ).toBe(false);
      }
    }

    // Nenhum preset sem id: um preset sem id nao e' aplicavel.
    for (const preset of presets) {
      expect(typeof preset.id).toBe("string");
      expect(preset.id.length).toBeGreaterThan(0);
    }

    // 2. A UI.
    await page.goto("/agentic");

    const grupo = page.getByRole("group", { name: "Preset de missão" });
    await expect(
      grupo,
      "o picker tem de aparecer quando o runtime publica presets",
    ).toBeVisible();

    // 3. O DISCRIMINADOR: mesmo numero (+1 pelo "Sem preset").
    const opcoes = grupo.getByRole("radio");
    await expect(
      opcoes,
      `a UI tem de mostrar ${presets.length} presets + a opcao "Sem preset"`,
    ).toHaveCount(presets.length + 1);

    // 4. Cada preset do servidor tem de aparecer com o seu nome.
    for (const preset of presets) {
      const nome = (preset.name || preset.id).trim();
      if (!nome) continue;
      await expect(
        grupo.getByText(nome, { exact: false }).first(),
        `o preset ${preset.id} tem de aparecer no picker`,
      ).toBeVisible();
    }

    // 5. A escolha tem de ser registada, nao apenas desenhada. Uso o indice 1 porque
    //    o indice 0 e' sempre "Sem preset".
    const primeiroPreset = grupo.getByRole("radio").nth(1);
    await primeiroPreset.check();
    await expect(primeiroPreset).toBeChecked();

    // 6. Recusar todos os presets tem de ser possivel: o utilizador nao pode ficar
    //    preso a um preset.
    const semPreset = grupo.getByRole("radio").first();
    await semPreset.check();
    await expect(semPreset).toBeChecked();

    console.log(
      `[mission-presets] modo=${MODO} presets=${presets.length} ` +
        `ids=${presets.map((p) => p.id).join(",")}`,
    );
  });

  test("o catalogo e' estavel entre duas leituras", async ({ request }) => {
    const ler = async () => {
      const r = await request.get("/api/agent/v1/presets", { timeout: 30_000 });
      expect(r.status()).toBe(200);
      return (await r.json()) as { presets: PresetDoServidor[] };
    };
    const ids = (c: { presets: PresetDoServidor[] }) =>
      c.presets.map((p) => p.id).sort().join(",");
    const a = await ler();
    const b = await ler();
    expect(
      ids(a),
      "presets sao configuracao versionada, nao estado mutavel por pedido",
    ).toBe(ids(b));
  });
});