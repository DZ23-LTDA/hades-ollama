import { expect, test, type Page } from "@playwright/test";

// Jornada de primeira execução contra um backend HTTP real.
//
// Escopo do portão de primeira execução: `src/routes/index.tsx` compara
// `settings.OnboardingVersion` com `CURRENT_ONBOARDING_VERSION` (lib/onboarding.ts)
// a cada carga de `/`. Quando a leitura de `GET /api/v1/settings` falha, o shell
// segue utilizável e o onboarding NUNCA aparece; quando responde com uma versão
// anterior à atual, `/` redireciona para `/onboarding`. Ou seja: o portão de
// instalação/onboarding/persistência é do SERVIDOR, não do navegador.
//
// Este arquivo cobre, ponta a ponta e com requisições HTTP reais (sem
// `page.route`), as etapas que dependem apenas desse contrato:
//
//   1. instalação nova -> `/` abre o onboarding (gate lido do servidor);
//   2. onboarding local-first -> a escolha "usar os modelos locais" persiste
//      `OnboardingVersion` via `POST /api/v1/settings` (resposta real conferida);
//   3. download do primeiro modelo -> a UI só libera "Continuar" depois que o
//      stream de `POST /api/v1/models/pull` termina (o stream é stub do backend,
//      o portão da UI é o comportamento real);
//   4. conclusão -> `/connect`, com o estado relido do servidor;
//   5. reinício -> `/` volta ao shell sem repetir o onboarding (persistiu);
//   6. atualização -> uma versão de onboarding mais nova não reabre o fluxo;
//   7. rollback -> voltar a versão anterior reabre o onboarding (o gate é real,
//      não uma marcação local esquecida).
//
// Limite declarado e honesto: as etapas de missão, artefato, aprovação e
// cancelamento exigem o runtime de agente + um modelo servido, que não estão
// disponíveis no job de E2E (`dz23-e2e.yaml` instala apenas Node + Chromium).
// Essas etapas são cobertas hoje pelos testes de runtime em Go
// (`internal/agent/`) e por `e2e/scheduleFlow.spec.ts`. A build/instalação do
// binário é coberta por `.github/workflows/test-install.yaml`. Este spec assume
// o artefato JÁ construído (`dist/`) servido pelo preview do Vite, que faz proxy
// de `/api` para `OLLAMA_PROXY_TARGET` — ver `e2e/first-run-backend.mjs`.
//
// O backend é iniciado como segundo `webServer` em `playwright.config.ts`, e o
// preview aponta o proxy para ele. Como as specs `shell`/`accessibility` também
// rodam nessa origem, o estado padrão do backend é "já configurado"
// (`OnboardingVersion: 1`); cada teste deste arquivo simula uma instalação nova
// pela API pública e restaura o estado ao final.

const SETTINGS_PATH = "/api/v1/settings";
const LOCAL_FIRST_LABEL = "Não, obrigado — vou usar os modelos locais";

function isSettingsWrite(url: string, method: string): boolean {
  return method === "POST" && url.includes(SETTINGS_PATH);
}

async function setOnboardingVersion(
  request: import("@playwright/test").APIRequestContext,
  version: number,
): Promise<void> {
  const response = await request.post(SETTINGS_PATH, {
    data: { OnboardingVersion: version },
  });
  expect(response.ok()).toBeTruthy();
  expect((await response.json()).settings.OnboardingVersion).toBe(version);
}

// Reinício de verdade: fecha a aba e abre outra. `goto`/`reload` na MESMA URL
// não servem como reinício aqui — a rota `/` é mascarada (`mask: { to: "/" }` em
// `src/routes/index.tsx`) e o navegador pode restaurar o documento da memória,
// sem reexecutar o `beforeLoad` que lê o portão do servidor. Uma aba nova parte
// de um documento limpo e obriga a nova leitura de `GET /api/v1/settings`.
async function restart(page: Page): Promise<Page> {
  const context = page.context();
  await page.close();
  const fresh = await context.newPage();
  await fresh.goto("/");
  return fresh;
}

test.describe("primeira execução — gate de onboarding servido pelo backend", () => {
  test.beforeEach(async ({ request }) => {
    // Instalação nova: o servidor informa que o onboarding ainda não aconteceu.
    await setOnboardingVersion(request, 0);
  });

  test.afterEach(async ({ request }) => {
    // Devolve o ambiente ao estado já configurado usado pelas outras specs.
    await setOnboardingVersion(request, 1);
  });

  test("a API de settings persiste OnboardingVersion e preserva quando omitido", async ({
    request,
  }) => {
    await setOnboardingVersion(request, 7);

    // Mesmo contrato de `TestSettingsPreservesOnboardingVersionWhenOmitted`
    // (app/ui/ui_test.go): campo omitido preserva o valor gravado.
    const omitted = await request.post(SETTINGS_PATH, { data: {} });
    expect(omitted.ok()).toBeTruthy();
    expect((await omitted.json()).settings.OnboardingVersion).toBe(7);

    const readBack = await request.get(SETTINGS_PATH);
    expect(readBack.ok()).toBeTruthy();
    expect((await readBack.json()).settings.OnboardingVersion).toBe(7);
  });

  test("instalação, onboarding, persistência, reinício, atualização e rollback", async ({
    page: firstPage,
    request,
  }) => {
    let page = firstPage;
    await test.step("instalação nova abre o onboarding a partir do estado do servidor", async () => {
      await page.goto("/");
      await expect(page).toHaveURL(/\/onboarding$/);
      await expect(
        page.getByRole("heading", { name: "Bem-vindo ao Hades!", level: 1 }),
      ).toBeVisible();
    });

    await test.step("a escolha local-first é gravada por POST /api/v1/settings", async () => {
      await page
        .getByRole("button", { name: "Continuar", exact: true })
        .click();
      await expect(
        page.getByRole("heading", { name: "Crie uma conta", level: 1 }),
      ).toBeVisible();

      const saved = page.waitForResponse((response) =>
        isSettingsWrite(response.url(), response.request().method()),
      );
      await page.getByRole("button", { name: LOCAL_FIRST_LABEL }).click();
      const savedResponse = await saved;
      expect(savedResponse.ok()).toBeTruthy();
      expect((await savedResponse.json()).settings.OnboardingVersion).toBe(1);

      await expect(
        page.getByRole("heading", {
          name: "Baixe seu primeiro modelo",
          level: 1,
        }),
      ).toBeVisible();

      // O passo de download é stub do backend (ver first-run-backend.mjs), mas o
      // portão da UI é real: "Continuar" só existe depois do stream de pull.
      await expect(
        page.getByRole("button", { name: "Continuar", exact: true }),
      ).toHaveCount(0);
    });

    await test.step("o download do primeiro modelo destrava a conclusão do onboarding", async () => {
      await page.getByRole("button", { name: "Baixar", exact: true }).click();
      await expect(
        page.getByRole("button", { name: "Continuar", exact: true }),
      ).toBeVisible();
    });

    await test.step("concluir o onboarding leva ao shell e o estado fica no servidor", async () => {
      const saved = page.waitForResponse((response) =>
        isSettingsWrite(response.url(), response.request().method()),
      );
      await page
        .getByRole("button", { name: "Continuar", exact: true })
        .click();
      expect((await saved).ok()).toBeTruthy();
      await expect(page).toHaveURL(/\/connect$/);

      const persisted = await request.get(SETTINGS_PATH);
      expect(persisted.ok()).toBeTruthy();
      expect((await persisted.json()).settings.OnboardingVersion).toBe(1);
    });

    await test.step("reinício: o shell volta sem repetir o onboarding", async () => {
      page = await restart(page);
      await expect(page).not.toHaveURL(/\/onboarding/);
      await expect(
        page.getByRole("heading", {
          name: "O que posso fazer por você?",
          level: 1,
        }),
      ).toBeVisible();
    });

    await test.step("atualização: uma versão mais nova não reabre o onboarding", async () => {
      await setOnboardingVersion(request, 2);
      page = await restart(page);
      await expect(page).not.toHaveURL(/\/onboarding/);
      await expect(
        page.getByRole("heading", {
          name: "O que posso fazer por você?",
          level: 1,
        }),
      ).toBeVisible();
    });

    await test.step("rollback: voltar a versão anterior reabre o onboarding", async () => {
      await setOnboardingVersion(request, 0);
      page = await restart(page);
      await expect(page).toHaveURL(/\/onboarding$/);
      await expect(
        page.getByRole("heading", { name: "Bem-vindo ao Hades!", level: 1 }),
      ).toBeVisible();
    });
  });
});
