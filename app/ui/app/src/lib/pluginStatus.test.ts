import { describe, expect, it } from "vitest";

import {
  isReadyToUse,
  pluginExecutionSummary,
  pluginStateDescription,
  pluginStateLabel,
  pluginStateTone,
  type AgentPluginStatus,
} from "./pluginStatus";

function status(overrides: Partial<AgentPluginStatus> = {}): AgentPluginStatus {
  return {
    plugin_id: "runtime-plugin",
    version: "1.0.0",
    kind: "connector",
    state: "INSTALLED",
    enabled: true,
    trusted: true,
    content_verified: true,
    capabilities: ["sandbox:execute"],
    ready_to_use: false,
    last_execution: null,
    ...overrides,
  };
}

describe("status canônico de plugin (§11.3)", () => {
  it("rotula todos os estados canônicos em pt-BR", () => {
    expect(pluginStateLabel("INSTALLED")).toBe("Instalado");
    expect(pluginStateLabel("TESTED_LIVE")).toBe("Execução comprovada");
    expect(pluginStateLabel("CONFIG_REQUIRED")).toBe("Configuração pendente");
    expect(pluginStateLabel("UNAUTHORIZED")).toBe("Não autorizado");
    expect(pluginStateLabel("UNHEALTHY")).toBe("Conteúdo divergente");
    expect(pluginStateLabel("UNSUPPORTED")).toBe("Não executável");
    expect(pluginStateLabel("CONNECTED")).toBe("Conectado");
  });

  it("não inventa rótulo para estado desconhecido do servidor", () => {
    expect(pluginStateLabel("ALGO_NOVO")).toBe("Estado desconhecido");
    expect(pluginStateDescription("ALGO_NOVO")).toContain("indisponível");
    expect(pluginStateTone("ALGO_NOVO")).toBe("neutral");
  });

  it("nunca considera pronto sem ready_to_use do servidor", () => {
    // INSTALLED executável, com tudo verde, mas sem execução comprovada.
    expect(isReadyToUse(status())).toBe(false);
    // Servidor diz pronto, mas o estado não é TESTED_LIVE: recusa por segurança.
    expect(isReadyToUse(status({ ready_to_use: true, state: "INSTALLED" }))).toBe(false);
    // Estado certo, mas sem ready_to_use: também recusa.
    expect(isReadyToUse(status({ state: "TESTED_LIVE", ready_to_use: false }))).toBe(false);
    // Só a combinação real passa.
    expect(isReadyToUse(status({ state: "TESTED_LIVE", ready_to_use: true }))).toBe(true);
    expect(isReadyToUse(null)).toBe(false);
    expect(isReadyToUse(undefined)).toBe(false);
  });

  it("distingue execução comprovada de tentativa recusada", () => {
    expect(pluginExecutionSummary(status())).toBe("Nenhuma execução registrada.");
    expect(
      pluginExecutionSummary(
        status({
          state: "TESTED_LIVE",
          ready_to_use: true,
          last_execution: {
            step_id: "step_1",
            executed_at: "2026-10-10T20:00:00Z",
            succeeded: true,
            sandbox_mode: "strict",
          },
        }),
      ),
    ).toContain("bem-sucedida");
    expect(
      pluginExecutionSummary(
        status({
          last_execution: {
            step_id: "step_2",
            executed_at: "2026-10-10T20:05:00Z",
            succeeded: false,
            sandbox_mode: "strict",
          },
        }),
      ),
    ).toContain("recusada");
  });

  it("usa cor de alerta para estados que impedem uso", () => {
    expect(pluginStateTone("TESTED_LIVE")).toBe("green");
    expect(pluginStateTone("CONFIG_REQUIRED")).toBe("amber");
    expect(pluginStateTone("UNAUTHORIZED")).toBe("red");
    expect(pluginStateTone("UNHEALTHY")).toBe("red");
  });
});
