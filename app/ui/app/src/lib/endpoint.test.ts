import { describe, expect, it } from "vitest";
import { EndpointPage } from "@/components/EndpointPage";
import { Route } from "@/routes/endpoint";
import {
  endpointHealthLabel,
  endpoints,
  gatewayRotationLabel,
  gatewaySetups,
  toolSetups,
} from "./endpoint";

describe("endpoint health", () => {
  it("does not claim online while the backend is unavailable", () => {
    expect(endpointHealthLabel("offline")).toBe("Offline");
    expect(endpointHealthLabel("checking")).toBe("Verificando…");
    expect(endpointHealthLabel("online")).toBe("Online");
  });
});

describe("endpoint page data", () => {
  it("exposes native, OpenAI and Anthropic base addresses", () => {
    expect(endpoints().map((e) => e.url)).toEqual([
      "http://localhost:11434",
      "http://localhost:11434/v1",
      "http://localhost:11434",
    ]);
  });

  it("builds launch commands for the selected model", () => {
    const setups = toolSetups("qwen3:8b");
    expect(setups.find((s) => s.id === "claude")?.code).toBe(
      "ollama launch claude --model qwen3:8b",
    );
    expect(setups.find((s) => s.id === "codex")?.code).toBe(
      "ollama launch codex --model qwen3:8b",
    );
    expect(setups.find((s) => s.id === "claude-env")?.code).toContain(
      'ANTHROPIC_BASE_URL = "http://localhost:11434"',
    );
  });

  it("quotes model names that are not shell-safe", () => {
    expect(toolSetups('odd "name"').find((s) => s.id === "codex")?.code).toBe(
      'ollama launch codex --model "odd \\"name\\""',
    );
  });

  it("routes /endpoint to the page", () => {
    expect(Route.options.component).toBe(EndpointPage);
  });
});

describe("gateway de inferência de terceiros", () => {
  it("describes the rotation already resolved by the backend", () => {
    expect(
      gatewayRotationLabel({
        enabled: true,
        max_attempts: 3,
        cross_provider: false,
      }),
    ).toBe("Rotação ligada · 3 tentativas · provedores reservas");
    expect(
      gatewayRotationLabel({
        enabled: true,
        max_attempts: 1,
        cross_provider: true,
      }),
    ).toBe("Rotação ligada · 1 tentativa · entre provedores diferentes");
    expect(
      gatewayRotationLabel({
        enabled: false,
        max_attempts: 3,
        cross_provider: false,
      }),
    ).toBe("Rotação desligada");
  });

  it("points Claude and Codex at the gateway with the gateway key", () => {
    const setups = gatewaySetups(
      {
        base_url: "http://localhost:11434/",
        gateway_key: "gw-secret",
        gateway_key_env: "OLLAMA_DZ23_GATEWAY_KEY",
      },
      "auto",
    );
    const claude = setups.find((s) => s.id === "gateway-claude")?.code ?? "";
    expect(claude).toContain('ANTHROPIC_BASE_URL = "http://localhost:11434"');
    expect(claude).toContain('ANTHROPIC_AUTH_TOKEN = "gw-secret"');
    expect(claude).toContain('ANTHROPIC_API_KEY = ""');
    const codex = setups.find((s) => s.id === "gateway-codex")?.code ?? "";
    expect(codex).toContain('OPENAI_BASE_URL = "http://localhost:11434/v1"');
    expect(codex).toContain('OPENAI_API_KEY = "gw-secret"');
    expect(setups.find((s) => s.id === "gateway-curl")?.code).toBe(
      'curl -H "Authorization: Bearer gw-secret" http://localhost:11434/v1/models',
    );
  });

  it("falls back to the environment variable while the key is not on screen", () => {
    const setups = gatewaySetups(
      {
        base_url: "http://localhost:11434",
        gateway_key_env: "OLLAMA_DZ23_GATEWAY_KEY",
      },
      "qwen3:8b",
    );
    expect(setups.find((s) => s.id === "gateway-claude")?.code).toContain(
      'ANTHROPIC_AUTH_TOKEN = "<OLLAMA_DZ23_GATEWAY_KEY>"',
    );
  });
});
