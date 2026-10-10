import { defineConfig, devices } from "@playwright/test";

// E2E local-first: sobe o build do shell via `vite preview` e valida a jornada
// de UI sem depender de provider externo (o shell renderiza offline).
//
// O segundo `webServer` é o backend mínimo da jornada de primeira execução
// (`e2e/first-run-backend.mjs`): o portão de onboarding é decidido pelo servidor
// via `GET/POST /api/v1/settings`, então essa jornada só é verificável com HTTP
// real. Como o build de preview tem `API_BASE = ""` (mesma origem) e o
// `vite.config.ts` faz proxy de `/api` para `OLLAMA_PROXY_TARGET`, basta apontar
// o proxy do preview para esse backend: nenhuma mudança em `src/`, nenhum
// segundo `vite build` e nenhuma interceptação de rede no navegador.
const PORT = 4173;
const BASE_URL = `http://127.0.0.1:${PORT}`;
const BACKEND_PORT = 43117;
const BACKEND_URL = `http://127.0.0.1:${BACKEND_PORT}`;

export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  // Um único worker: a jornada de primeira execução escreve estado GLOBAL no
  // backend (`OnboardingVersion`), e `shell`/`accessibility` esperam o shell já
  // configurado. Com vários workers os dois projetos rodariam ao mesmo tempo e a
  // instalação nova simulada por um derrubaria o outro.
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["list"]] : [["list"]],
  use: {
    baseURL: BASE_URL,
    trace: "on-first-retry",
  },
  // Projetos rodam em sequência (workers: 1) e o primeiro ignora a spec de
  // primeira execução, que é executada depois, sozinha, pelo projeto `first-run`.
  projects: [
    {
      name: "chromium",
      testIgnore: /firstRun\.spec\.ts/,
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "first-run",
      testMatch: /firstRun\.spec\.ts/,
      // Jornada longa (build do shell + 7 etapas de UI) com um único worker.
      timeout: 180_000,
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: [
    {
      command: `npm run build && npm run preview -- --port ${PORT} --host 127.0.0.1 --strictPort`,
      url: BASE_URL,
      timeout: 180_000,
      reuseExistingServer: !process.env.CI,
      env: { OLLAMA_PROXY_TARGET: BACKEND_URL },
    },
    {
      command: `node e2e/first-run-backend.mjs`,
      url: `${BACKEND_URL}/api/version`,
      timeout: 30_000,
      reuseExistingServer: !process.env.CI,
      stdout: "pipe",
      stderr: "pipe",
      env: { E2E_BACKEND_PORT: String(BACKEND_PORT) },
    },
  ],
});
