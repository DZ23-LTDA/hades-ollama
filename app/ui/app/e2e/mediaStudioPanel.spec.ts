import { test, expect } from "@playwright/test";

// PROVA EM BROWSER DO ESTUDIO MULTIMODAL (item 45).
//
// *Porque NAO ha' `page.route` aqui:* medido em 2026-10-15 (commit e899c5f77) — um spec
// com `page.route` foi reprovado porque as chamadas do browser foram ao backend-stub da
// suite (`[e2e-backend] GET /api/agent/v1/missions -> 404`) e nao foram interceptadas.
// O caminho do repositorio e' o fixture ligado ao stub: `e2e/media-studio-fixture.mjs`
// (mesmo contrato do `studio-builder-fixture.mjs`), servido em `first-run-backend.mjs`.
//
// `?mission_id=` e' lido pelo proprio console (`AgenticConsole.tsx:160`), portanto nao e'
// preciso simular cliques numa lista de missoes.
//
// O que se prova: que o painel MONTA e mostra o estado HONESTO do provider. Sem provider
// medido (`providerConfigurado === undefined`) o resumo tem de dizer "a confirmar" — nao
// pode prometer disponibilidade que nao mediu (exigencia explicita do item 45).

const OPERACOES = [
  "tone.generate",
  "audio.transcribe",
  "image.analyze",
  "image.generate",
  "video.generate",
  "speech.generate",
];

test.describe("Estudio multimodal (item 45)", () => {
  test("o painel monta e declara o estado do provider sem prometer disponibilidade", async ({
    page,
  }) => {
    await page.goto("/agentic?mission_id=mis_e2e");

    // A aba de artefactos pode ou nao estar activa por omissao.
    const abaArtefatos = page.getByRole("button", { name: /^Artefatos/ });
    if ((await abaArtefatos.count()) > 0) {
      await abaArtefatos.first().click();
    }

    const resumo = page.locator("[data-estudio-resumo]");
    await expect(resumo).toBeVisible();

    // Honestidade: sem medida de provider, o texto NAO afirma disponibilidade.
    await expect(resumo).toContainText(/confirmar/i);

    // As 6 operacoes do catalogo estao lancaveis na UI (o servidor decide ao executar).
    for (const operacao of OPERACOES) {
      await expect(
        page.getByText(operacao, { exact: false }).first(),
      ).toBeVisible();
    }
  });
});
