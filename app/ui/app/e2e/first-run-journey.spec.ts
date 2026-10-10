import { expect, test, type Page, type Route } from "@playwright/test";

/**
 * P0-1 - Jornada UI-first de primeira execucao.
 *
 * Cobre, numa unica jornada de ponta a ponta:
 *   1. primeira execucao abre o onboarding;
 *   2. escolha por modelos locais conclui o onboarding e persiste em
 *      `/api/v1/settings` (OnboardingVersion = 1);
 *   3. instalacao do primeiro modelo local (`/api/v1/models/pull`);
 *   4. criacao de missao no Console de Missoes (`/agentic`);
 *   5. aprovacao humana (justificativa obrigatoria antes de decidir);
 *   6. artefato entregue pela missao;
 *   7. cancelamento explicito ("Interromper");
 *   8. reinicio da interface com persistencia do estado cancelado.
 *
 * A etapa de atualizacao com rollback e exercitada no nivel de contrato pelo
 * teste Vitest `src/components/settingsUtils.rollback.test.ts`, que roda no
 * gate obrigatorio "Web and mobile quality"; aqui a jornada cobre a
 * persistencia real da escrita de configuracao.
 *
 * Escopo honesto: `.github/workflows/dz23-e2e.yaml` sobe apenas o bundle
 * estatico (`npm run build && npm run preview`) e nao ha backend Go servido
 * para o Playwright em CI. Por isso todo `/api/**` e atendido por um backend
 * deterministico em memoria registrado via `page.route`, e nao se afirma aqui
 * execucao contra o servidor real. O smoke com binario real e coberto pelos
 * workflows `test-install.yaml` e `dz23-linux-package.yaml`.
 */

const MISSION_ID = "mis_jornada_1";
const MISSION_OBJECTIVE = "Gerar o relatorio de abertura do workspace";
const PLAN_STEP_ID = "step_2";
const APPROVAL_ID = "app_1";
const ARTIFACT_NAME = "relatorio-jornada.md";
const CREATED_AT = "2026-10-09T12:00:00Z";

interface StubCall {
  method: string;
  pathname: string;
  body: string;
}

interface StubStep {
  id: string;
  title: string;
  kind: string;
  state: string;
  requires_approval: boolean;
}

interface StubApproval {
  id: string;
  step_id: string;
  status: string;
  policy?: string;
  nonce?: string;
}

interface StubArtifact {
  id: string;
  name: string;
  size: number;
  sha256: string;
}

interface StubMission {
  id: string;
  version: number;
  objective: string;
  provider?: string;
  model?: string;
  state: string;
  plan: StubStep[];
  approvals: StubApproval[];
  artifacts: StubArtifact[];
  created_at: string;
  updated_at: string;
}

function fulfillJson(route: Route, body: unknown, status = 200) {
  return route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  });
}

function parseBody(route: Route): Record<string, unknown> {
  const raw = route.request().postData();
  if (!raw) return {};
  try {
    return JSON.parse(raw) as Record<string, unknown>;
  } catch {
    return {};
  }
}

async function installBackendStub(page: Page) {
  const calls: StubCall[] = [];
  const settings: Record<string, unknown> = {
    OnboardingVersion: 0,
    Expose: false,
    Browser: false,
    Models: "",
    Agent: false,
    Tools: false,
    ContextLength: 8192,
    AutoUpdateEnabled: true,
  };
  const missions: StubMission[] = [];

  const currentMission = (): StubMission | undefined =>
    missions[missions.length - 1];

  const record = (route: Route) => {
    const request = route.request();
    const { pathname } = new URL(request.url());
    calls.push({
      method: request.method().toUpperCase(),
      pathname,
      body: request.postData() ?? "",
    });
  };

  await page.route(
    (url) => url.pathname.startsWith("/api/"),
    async (route) => {
      record(route);
      const request = route.request();
      const method = request.method().toUpperCase();
      const { pathname } = new URL(request.url());

      // --- configuracao -------------------------------------------------
      if (pathname === "/api/v1/settings") {
        if (method === "GET") return fulfillJson(route, { settings });
        const parsed = parseBody(route);
        const patch = (parsed.settings ?? parsed) as Record<string, unknown>;
        Object.assign(settings, patch);
        return fulfillJson(route, { settings });
      }

      // --- instalacao do primeiro modelo --------------------------------
      if (pathname === "/api/v1/models/pull") {
        return route.fulfill({
          status: 200,
          contentType: "application/x-ndjson",
          body:
            '{"status":"pulling manifest"}\n' +
            '{"status":"success","completed":1,"total":1}\n',
        });
      }

      if (pathname === "/api/tags") {
        return fulfillJson(route, {
          models: [
            {
              name: "qwen2.5:7b",
              model: "qwen2.5:7b",
              size: 4000000000,
              digest: "sha256:jornada",
              details: {
                family: "qwen2",
                families: ["qwen2"],
                parameter_size: "7B",
                quantization_level: "Q4_K_M",
              },
            },
          ],
        });
      }

      // --- superficies auxiliares do shell ------------------------------
      if (pathname === "/api/v1/integrations") {
        return fulfillJson(route, { integrations: [] });
      }
      if (pathname === "/api/agent/v1/notifications") {
        return fulfillJson(route, { notifications: [] });
      }
      if (pathname === "/api/agent/v1/metrics") {
        return fulfillJson(route, {
          missions_created: missions.length,
          missions_completed: 0,
          missions_running: missions.filter((m) => m.state === "RUNNING").length,
          missions_failed: 0,
          steps_completed: 0,
          approvals_pending: 0,
        });
      }
      if (pathname === "/api/agent/v1/projects") {
        return fulfillJson(route, { projects: [] });
      }

      // --- missoes ------------------------------------------------------
      if (pathname === "/api/agent/v1/missions") {
        if (method === "POST") {
          const payload = parseBody(route);
          const created: StubMission = {
            id: MISSION_ID,
            version: 1,
            objective:
              typeof payload.objective === "string" && payload.objective
                ? payload.objective
                : MISSION_OBJECTIVE,
            provider: "ollama-local",
            model: "qwen2.5:7b",
            state: "AWAITING_APPROVAL",
            plan: [
              {
                id: "step_1",
                title: "Inspecionar o workspace do projeto",
                kind: "workspace.read",
                state: "COMPLETED",
                requires_approval: false,
              },
              {
                id: PLAN_STEP_ID,
                title: "Escrever o relatorio de abertura",
                kind: "workspace.write",
                state: "AWAITING_APPROVAL",
                requires_approval: true,
              },
            ],
            approvals: [
              {
                id: APPROVAL_ID,
                step_id: PLAN_STEP_ID,
                status: "PENDING",
                policy: "workspace:write",
                nonce: "nonce_jornada",
              },
            ],
            artifacts: [],
            created_at: CREATED_AT,
            updated_at: CREATED_AT,
          };
          missions.push(created);
          return fulfillJson(route, created);
        }
        return fulfillJson(route, { missions });
      }

      const eventsMatch = pathname.match(
        /^\/api\/agent\/v1\/missions\/([^/]+)\/events$/,
      );
      if (eventsMatch) {
        return fulfillJson(route, {
          events: [
            { id: "ev_1", type: "mission.created", created_at: CREATED_AT },
            { id: "ev_2", type: "step.succeeded", created_at: CREATED_AT },
          ],
        });
      }

      const approvalMatch = pathname.match(
        /^\/api\/agent\/v1\/missions\/([^/]+)\/approvals\/([^/]+)$/,
      );
      if (approvalMatch && method === "POST") {
        const target = currentMission();
        if (target) {
          const decision = String(parseBody(route).decision ?? "");
          target.state = decision === "reject" ? "PAUSED" : "RUNNING";
          target.approvals = target.approvals.map((entry) =>
            entry.id === approvalMatch[2]
              ? { ...entry, status: decision === "reject" ? "REJECTED" : "APPROVED" }
              : entry,
          );
          target.plan = target.plan.map((step) =>
            step.id === PLAN_STEP_ID ? { ...step, state: "COMPLETED" } : step,
          );
          if (decision !== "reject") {
            target.artifacts = [
              {
                id: "art_jornada",
                name: ARTIFACT_NAME,
                size: 2048,
                sha256:
                  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
              },
            ];
          }
          return fulfillJson(route, target);
        }
        return fulfillJson(route, {});
      }

      const cancelMatch = pathname.match(
        /^\/api\/agent\/v1\/missions\/([^/]+)\/cancel$/,
      );
      if (cancelMatch && method === "POST") {
        const target = currentMission();
        if (target) {
          target.state = "CANCELLED";
          return fulfillJson(route, target);
        }
        return fulfillJson(route, {});
      }

      const runMatch = pathname.match(
        /^\/api\/agent\/v1\/missions\/([^/]+)\/run$/,
      );
      if (runMatch && method === "POST") {
        const target = currentMission();
        if (target) {
          target.state = "RUNNING";
          return fulfillJson(route, target);
        }
        return fulfillJson(route, {});
      }

      const artifactMatch = pathname.match(
        /^\/api\/agent\/v1\/missions\/([^/]+)\/artifacts\/([^/]+)$/,
      );
      if (artifactMatch) {
        return route.fulfill({
          status: 200,
          contentType: "text/plain",
          body: "conteudo do relatorio de abertura\n",
        });
      }

      const missionMatch = pathname.match(
        /^\/api\/agent\/v1\/missions\/([^/]+)$/,
      );
      if (missionMatch && method === "GET") {
        const found = missions.find((entry) => entry.id === missionMatch[1]);
        return fulfillJson(route, found ?? currentMission() ?? {});
      }

      return fulfillJson(route, {});
    },
  );

  return {
    calls,
    settings,
    missions,
    settingsValue(key: string) {
      return settings[key];
    },
    callsTo(pathname: string, method?: string) {
      return calls.filter(
        (call) =>
          call.pathname === pathname && (!method || call.method === method),
      );
    },
  };
}

test.describe("Jornada de primeira execucao", () => {
  test("instalacao, onboarding, missao, artefato, aprovacao, cancelamento e reinicio", async ({
    page,
  }) => {
    const backend = await installBackendStub(page);
    const pageErrors: string[] = [];
    page.on("pageerror", (error) => pageErrors.push(String(error)));

    await test.step("1. primeira execucao abre o onboarding", async () => {
      await page.goto("/onboarding", { waitUntil: "domcontentloaded" });
      await expect(
        page.getByRole("heading", { name: "Bem-vindo ao Hades!" }),
      ).toBeVisible();
      await expect(
        page.getByText("Rode modelos abertos com seus agentes de código"),
      ).toBeVisible();
      await expect(
        page.getByRole("button", { name: "Continuar", exact: true }),
      ).toBeVisible();
    });

    await test.step("2. escolha local persiste o onboarding", async () => {
      await page
        .getByRole("button", { name: "Continuar", exact: true })
        .click();
      await page
        .getByRole("button", { name: /vou usar os modelos locais/i })
        .click();
      await expect
        .poll(() => backend.settingsValue("OnboardingVersion"))
        .toBe(1);
      expect(
        backend.callsTo("/api/v1/settings").filter((c) => c.method !== "GET")
          .length,
      ).toBeGreaterThan(0);
    });

    await test.step("3. instalacao do primeiro modelo local", async () => {
      await page.getByRole("button", { name: "Baixar", exact: true }).click();
      await expect
        .poll(() =>
          backend.callsTo("/api/v1/models/pull", "POST").length,
        )
        .toBeGreaterThan(0);
    });

    await test.step("4. missao criada no console agentic", async () => {
      await page.goto("/agentic", { waitUntil: "domcontentloaded" });
      await expect(
        page.getByRole("heading", { name: "Console de Missões" }),
      ).toBeVisible();
      await page
        .getByPlaceholder(
          "Descreva o que você quer construir, pesquisar, revisar ou automatizar",
        )
        .fill(MISSION_OBJECTIVE);
      await page
        .getByRole("button", { name: "Criar missão", exact: true })
        .first()
        .click();
      await expect
        .poll(() => backend.callsTo("/api/agent/v1/missions", "POST").length)
        .toBeGreaterThan(0);
      await expect(page.getByText(MISSION_OBJECTIVE).first()).toBeVisible();
      await expect(
        page.getByText("Aguardando aprovação").first(),
      ).toBeVisible();
    });

    await test.step("5. aprovacao humana com justificativa", async () => {
      const reason = page.getByPlaceholder(
        "Explique por que esta ação deve ser aprovada ou rejeitada",
      );
      await expect(reason).toBeVisible();
      const approve = page
        .getByRole("button", { name: "Aprovar", exact: true })
        .first();
      await expect(approve).toBeDisabled();
      await reason.fill("Aprovado na jornada automatizada de primeira execucao");
      await expect(approve).toBeEnabled();
      await approve.click();
      await expect
        .poll(() =>
          backend.calls.filter((call) =>
            /\/approvals\//.test(call.pathname),
          ).length,
        )
        .toBeGreaterThan(0);
    });

    await test.step("6. artefato entregue e persistido", async () => {
      await expect(page.getByText(ARTIFACT_NAME).first()).toBeVisible();
      expect(backend.missions[backend.missions.length - 1]?.artifacts.length).toBe(1);
    });

    await test.step("7. cancelamento explicito da missao", async () => {
      await page.goto(`/agentic?missionId=${MISSION_ID}`, {
        waitUntil: "domcontentloaded",
      });
      const stop = page
        .getByRole("button", { name: "Interromper", exact: true })
        .first();
      await expect(stop).toBeVisible();
      await stop.click();
      await expect
        .poll(() => backend.callsTo(`/api/agent/v1/missions/${MISSION_ID}/cancel`, "POST").length)
        .toBeGreaterThan(0);
      await expect(page.getByText("Cancelada").first()).toBeVisible();
    });

    await test.step("8. reinicio preserva o estado persistido", async () => {
      await page.reload({ waitUntil: "domcontentloaded" });
      await expect(
        page.getByRole("heading", { name: "Console de Missões" }),
      ).toBeVisible();
      await expect(page.getByText("Cancelada").first()).toBeVisible();
      expect(backend.settingsValue("OnboardingVersion")).toBe(1);
    });

    expect(pageErrors).toEqual([]);
  });
});
