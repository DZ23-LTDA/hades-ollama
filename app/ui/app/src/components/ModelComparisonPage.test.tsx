import {
  act,
  create,
  type ReactTestInstance,
  type ReactTestRenderer,
} from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { sendMessage } from "@/api";
import { ChatEvent } from "@/gotypes";
import { useModels } from "@/hooks/useModels";
import { ModelComparisonPanel } from "./ModelComparisonPage";

vi.mock("@/hooks/useModels", () => ({ useModels: vi.fn() }));
vi.mock("@/api", () => ({ sendMessage: vi.fn() }));

const mockUseModels = vi.mocked(useModels);
const mockSendMessage = vi.mocked(sendMessage);

// Two local installs (with digests) are runnable; the cloud placeholder has no
// digest and must not be offered as a column.
function model(name: string, extra: Record<string, unknown> = {}) {
  return { model: name, digest: `sha256:${name}`, ...extra };
}

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

function findButton(renderer: ReactTestRenderer, label: string) {
  return renderer.root
    .findAllByType("button")
    .find((candidate) => textContent(candidate).includes(label));
}

function checkbox(renderer: ReactTestRenderer, label: string) {
  return renderer.root.findByProps({ "aria-label": `Comparar com ${label}` });
}

function streamOf(events: Array<{ eventName: string; content?: string; error?: string }>) {
  return (async function* () {
    for (const event of events) {
      yield new ChatEvent(event);
    }
  })();
}

async function flush(times = 8) {
  for (let i = 0; i < times; i++) {
    await Promise.resolve();
  }
}

async function render(): Promise<ReactTestRenderer> {
  let renderer!: ReactTestRenderer;
  await act(async () => {
    renderer = create(<ModelComparisonPanel />);
    await flush();
  });
  return renderer;
}

async function select(renderer: ReactTestRenderer, label: string) {
  await act(async () => {
    checkbox(renderer, label).props.onChange();
    await flush();
  });
}

async function type(renderer: ReactTestRenderer, value: string) {
  await act(async () => {
    renderer.root
      .findByProps({ "aria-label": "Pergunta para comparar" })
      .props.onChange({ target: { value } });
    await flush();
  });
}

async function submit(renderer: ReactTestRenderer) {
  await act(async () => {
    renderer.root
      .findByProps({ "aria-label": "Nova comparação" })
      .props.onSubmit({ preventDefault: () => undefined });
    await flush(20);
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
  mockUseModels.mockReturnValue({
    data: [
      model("llama3.2:3b"),
      model("qwen2.5:7b"),
      model("gpt-oss:120b-cloud"),
      // Not installed and not served by a provider: a column here could only fail.
      model("sugestao:1b", { digest: undefined, available: false }),
    ],
  } as never);
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("ModelComparisonPanel", () => {
  it("only enables the run once a question and two runnable models are picked", async () => {
    const renderer = await render();
    const run = () => findButton(renderer, "Comparar modelos")!.props.disabled;

    // A model that cannot answer right now is never offered as a column, while
    // a cloud model (served by the same backend) is.
    expect(
      renderer.root.findAllByProps({ "aria-label": "Comparar com sugestao:1b" }),
    ).toHaveLength(0);
    expect(
      renderer.root.findAllByProps({
        "aria-label": "Comparar com gpt-oss:120b-cloud",
      }),
    ).toHaveLength(1);
    expect(run()).toBe(true);

    await select(renderer, "llama3.2:3b");
    expect(run()).toBe(true);

    await select(renderer, "qwen2.5:7b");
    expect(run()).toBe(true);

    await type(renderer, "explique filas de mensagens mortas");
    expect(run()).toBe(false);
  });

  it("asks every selected model in its own temporary chat and shows both answers", async () => {
    mockSendMessage.mockImplementation(((_chatId: string, _message: string, model: { model: string }) =>
      streamOf([
        { eventName: "chat_created" },
        { eventName: "chat", content: `Resposta de ${model.model}` },
        { eventName: "done" },
      ])) as never);

    const renderer = await render();
    await select(renderer, "llama3.2:3b");
    await select(renderer, "qwen2.5:7b");
    await type(renderer, "explique filas de mensagens mortas");
    await submit(renderer);

    expect(mockSendMessage).toHaveBeenCalledTimes(2);
    const calls = mockSendMessage.mock.calls;
    expect(calls.map((call) => (call[2] as { model: string }).model).sort()).toEqual([
      "llama3.2:3b",
      "qwen2.5:7b",
    ]);
    // Every column is a brand new temporary chat, so nothing is persisted.
    for (const call of calls) {
      expect(call[0]).toBe("new");
      expect(call[1]).toBe("explique filas de mensagens mortas");
      expect(call[10]).toBe(true);
    }

    const text = textContent(renderer.root);
    expect(text).toContain("Resposta de llama3.2:3b");
    expect(text).toContain("Resposta de qwen2.5:7b");
    expect(text).toContain("2 de 2 responderam.");
    expect(
      renderer.root.findAllByProps({ "aria-label": "Resposta de llama3.2:3b" }),
    ).toHaveLength(1);
    expect(findButton(renderer, "Comparar modelos")!.props.disabled).toBe(false);
  });

  it("isolates a failing model instead of losing the other answers", async () => {
    mockSendMessage.mockImplementation(((_chatId: string, _message: string, model: { model: string }) => {
      if (model.model === "llama3.2:3b") {
        return (async function* () {
          yield new ChatEvent({
            eventName: "chat",
            content: "Parcial",
          });
          throw new Error("modelo indisponível");
        })();
      }
      return streamOf([
        { eventName: "chat", content: "Resposta boa" },
        { eventName: "done" },
      ]);
    }) as never);

    const renderer = await render();
    await select(renderer, "llama3.2:3b");
    await select(renderer, "qwen2.5:7b");
    await type(renderer, "pergunta");
    await submit(renderer);

    const alerts = renderer.root.findAllByProps({ role: "alert" });
    expect(alerts.map((alert) => textContent(alert)).join(" ")).toContain(
      "modelo indisponível",
    );
    const text = textContent(renderer.root);
    expect(text).toContain("Resposta boa");
    expect(text).toContain("Parcial");
    expect(text).toContain("1 de 2 responderam.");
  });

  it("keeps the interrupted answer visible and reports it as interrupted", async () => {
    mockSendMessage.mockImplementation(((_chatId: string, _message: string, _model: unknown, _attachments: unknown, signal?: AbortSignal) =>
      (async function* () {
        yield new ChatEvent({ eventName: "chat", content: "Trecho parcial" });
        await new Promise<void>((resolve) => {
          if (signal?.aborted) resolve();
          signal?.addEventListener("abort", () => resolve(), { once: true });
        });
      })()) as never);

    const renderer = await render();
    await select(renderer, "llama3.2:3b");
    await select(renderer, "qwen2.5:7b");
    await type(renderer, "pergunta");
    await act(async () => {
      renderer.root
        .findByProps({ "aria-label": "Nova comparação" })
        .props.onSubmit({ preventDefault: () => undefined });
      await flush(4);
    });

    expect(findButton(renderer, "Interromper")!.props.disabled).toBe(false);

    await act(async () => {
      findButton(renderer, "Interromper")!.props.onClick();
      await flush(20);
    });

    const text = textContent(renderer.root);
    expect(text).toContain("Trecho parcial");
    expect(text).toContain("Interrompido");
    expect(text).not.toContain("Nenhum modelo concluiu a resposta.");
    expect(findButton(renderer, "Comparar modelos")!.props.disabled).toBe(false);
  });
});
