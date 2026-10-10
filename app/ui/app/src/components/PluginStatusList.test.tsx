import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, describe, expect, it } from "vitest";

import { PluginStatusList, type PluginStatusListProps } from "./PluginStatusList";
import type { AgentPluginStatus } from "@/lib/pluginStatus";

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

// React 19 + react-test-renderer só materializa a árvore dentro de act().
async function render(props: Partial<PluginStatusListProps> = {}): Promise<ReactTestRenderer> {
  let renderer: ReactTestRenderer | null = null;
  await act(async () => {
    renderer = create(
      <PluginStatusList statuses={[]} loading={false} error={null} {...props} />,
    );
  });
  return renderer as unknown as ReactTestRenderer;
}

function textOf(renderer: ReactTestRenderer): string {
  return JSON.stringify(renderer.toJSON());
}

describe("PluginStatusList", () => {
  let renderer: ReactTestRenderer | null = null;

  afterEach(() => {
    renderer?.unmount();
    renderer = null;
  });

  it("não anuncia 'pronto para usar' para plugin sem execução comprovada", async () => {
    renderer = await render({ statuses: [status()] });
    const text = textOf(renderer);
    expect(text).toContain("Instalado");
    expect(text).toContain("sem execução comprovada");
    expect(text).not.toContain("Pronto para usar");
  });

  it("anuncia pronto apenas quando o servidor comprova execução", async () => {
    renderer = await render({
      statuses: [
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
      ],
    });
    const text = textOf(renderer);
    expect(text).toContain("Execução comprovada");
    expect(text).toContain("Pronto para usar");
  });

  it("explica o motivo quando o estado é de recusa", async () => {
    renderer = await render({
      statuses: [
        status({
          state: "UNAUTHORIZED",
          trusted: false,
          reasons: ["assinatura não aceita pelo servidor: promoção a confiável pendente"],
        }),
      ],
    });
    const text = textOf(renderer);
    expect(text).toContain("Não autorizado");
    expect(text).toContain("assinatura não aceita pelo servidor");
    expect(text).not.toContain("Pronto para usar");
  });

  it("mostra as capacidades concedidas, que são os escopos reais", async () => {
    renderer = await render({
      statuses: [status({ capabilities: ["sandbox:execute", "workspace:read"] })],
    });
    const text = textOf(renderer);
    expect(text).toContain("sandbox:execute");
    expect(text).toContain("workspace:read");
  });

  it("trata lista vazia, carregando e erro de forma honesta", async () => {
    renderer = await render();
    expect(textOf(renderer)).toContain("Nenhum plugin instalado");
    renderer.unmount();
    renderer = await render({ loading: true });
    expect(textOf(renderer)).toContain("Carregando");
    renderer.unmount();
    renderer = await render({ error: "organização não autenticada" });
    expect(textOf(renderer)).toContain("organização não autenticada");
  });
});
