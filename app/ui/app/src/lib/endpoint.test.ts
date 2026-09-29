import { describe, expect, it } from "vitest";
import { EndpointPage } from "@/components/EndpointPage";
import { Route } from "@/routes/endpoint";
import { endpoints, toolSetups } from "./endpoint";

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
