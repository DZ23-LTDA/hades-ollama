import {
  act,
  create,
  type ReactTestInstance,
  type ReactTestRenderer,
} from "react-test-renderer";
import { describe, expect, it, vi } from "vitest";

import { PROMPT_STORAGE_KEY } from "@/lib/promptLibrary";
import { PromptLibraryPanel } from "./PromptLibraryPanel";

function memoryStorage(initial: Record<string, string> = {}) {
  const data = new Map(Object.entries(initial));
  return {
    getItem: (key: string) => data.get(key) ?? null,
    setItem: (key: string, value: string) => {
      data.set(key, value);
    },
    raw: (key: string) => data.get(key) ?? null,
  };
}

type Storage = ReturnType<typeof memoryStorage>;

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

function renderer(props: { onUse?: (text: string) => void; storage?: Storage } = {}) {
  let result!: ReactTestRenderer;
  act(() => {
    result = create(<PromptLibraryPanel {...props} />);
  });
  return result;
}

function items(result: ReactTestRenderer): ReactTestInstance[] {
  return result.root
    .findAllByType("li")
    .filter((node) => typeof node.props["data-prompt-id"] === "string");
}

function button(result: ReactTestRenderer, label: string): ReactTestInstance {
  const found = result.root.findAllByType("button").find((candidate) => {
    const aria = String(candidate.props["aria-label"] ?? "");
    return textContent(candidate).includes(label) || aria.includes(label);
  });
  if (!found) throw new Error(`botão não encontrado: ${label}`);
  return found;
}

function field(result: ReactTestRenderer, id: string): ReactTestInstance {
  return result.root.findByProps({ id });
}

function fill(result: ReactTestRenderer, id: string, value: string) {
  act(() => {
    field(result, id).props.onChange({ target: { value } });
  });
}

function selectValue(result: ReactTestRenderer, id: string, value: string) {
  act(() => {
    field(result, id).props.onChange({ target: { value } });
  });
}

function click(result: ReactTestRenderer, label: string) {
  const target = button(result, label);
  act(() => {
    target.props.onClick();
  });
}

function submit(result: ReactTestRenderer) {
  const form = result.root.findAllByType("form")[0];
  act(() => {
    form.props.onSubmit({ preventDefault: () => undefined });
  });
}

function variableFields(result: ReactTestRenderer): ReactTestInstance[] {
  return result.root
    .findAllByType("input")
    .filter((node) => String(node.props.id).startsWith("prompt-value-"));
}

describe("PromptLibraryPanel", () => {
  it("lista os prompts iniciais e o contador", () => {
    const result = renderer();
    expect(items(result)).toHaveLength(4);
    expect(
      textContent(result.root.findByProps({ "aria-label": "Biblioteca de prompts" })),
    ).toContain("4 prompt(s) salvos neste navegador.");
  });

  it("cria um prompt, persiste e recarrega do armazenamento injetado", () => {
    const storage = memoryStorage();
    const first = renderer({ storage });
    fill(first, "prompt-title", "Resumo executivo");
    fill(first, "prompt-body", "Resuma o contexto {{projeto}} em cinco pontos.");
    fill(first, "prompt-folder", "Gestão");
    fill(first, "prompt-tags", "resumo, executivo");

    submit(first);

    expect(items(first)).toHaveLength(5);
    expect(textContent(first.root.findByProps({ role: "status" }))).toBe(
      "Prompt salvo na biblioteca.",
    );
    expect(storage.raw(PROMPT_STORAGE_KEY)).toContain("Resumo executivo");

    const reloaded = renderer({ storage });
    expect(items(reloaded)).toHaveLength(5);
    expect(textContent(reloaded.root)).toContain("Resumo executivo");
  });

  it("recusa prompt sem título e destaca o erro", () => {
    const storage = memoryStorage();
    const result = renderer({ storage });
    submit(result);
    expect(textContent(result.root.findByProps({ role: "alert" }))).toBe(
      "Informe um título para o prompt.",
    );
    expect(storage.raw(PROMPT_STORAGE_KEY)).toBeNull();
  });

  it("usa um prompt sem variáveis e entrega o texto ao consumidor", () => {
    const storage = memoryStorage();
    const onUse = vi.fn();
    const result = renderer({ onUse, storage });

    fill(result, "prompt-title", "Resumo simples");
    fill(result, "prompt-body", "Resuma em três pontos objetivos.");
    submit(result);

    click(result, "Usar o prompt Resumo simples");

    expect(onUse).toHaveBeenCalledTimes(1);
    expect(onUse.mock.calls[0][0]).toBe("Resuma em três pontos objetivos.");
    expect(textContent(result.root.findByProps({ role: "status" }))).toBe(
      "Prompt enviado para o campo de mensagem.",
    );
    expect(storage.raw(PROMPT_STORAGE_KEY)).toContain('"uses": 1');
  });

  it("preenche variáveis antes de usar e bloqueia enquanto faltar valor", () => {
    const onUse = vi.fn();
    const result = renderer({ onUse, storage: memoryStorage() });

    click(result, "Usar o prompt /goal");
    expect(onUse).not.toHaveBeenCalled();

    const fields = variableFields(result);
    expect(fields).toHaveLength(2);

    act(() => {
      fields[0].props.onChange({ target: { value: "reduzir o tempo de build" } });
    });
    click(result, "Usar o prompt /goal");
    expect(onUse).not.toHaveBeenCalled();

    const remaining = variableFields(result);
    act(() => {
      remaining[1].props.onChange({ target: { value: "monorepo Go + React" } });
    });

    click(result, "Gerar e usar");
    expect(onUse).toHaveBeenCalledTimes(1);
    expect(onUse.mock.calls[0][0]).toContain("reduzir o tempo de build");
    expect(onUse.mock.calls[0][0]).not.toContain("{{");
  });

  it("filtra por busca, pasta e favoritos", () => {
    const result = renderer({ storage: memoryStorage() });

    fill(result, "prompt-library-search", "rodar os testes");
    expect(items(result)).toHaveLength(1);

    fill(result, "prompt-library-search", "");
    selectValue(result, "prompt-library-folder", "Comandos");
    expect(items(result)).toHaveLength(4);

    click(result, "Alternar favorito de /test");

    act(() => {
      result.root.findByProps({ type: "checkbox" }).props.onChange({
        target: { checked: true },
      });
    });
    expect(items(result)).toHaveLength(2);
    expect(textContent(result.root)).toContain("/test");
  });

  it("edita, duplica e exclui um prompt", () => {
    const storage = memoryStorage();
    const result = renderer({ storage });

    click(result, "Editar /plan");
    fill(result, "prompt-title", "Plano revisado");
    submit(result);
    expect(textContent(result.root)).toContain("Prompt atualizado.");
    expect(textContent(result.root)).toContain("Plano revisado");

    click(result, "Duplicar Plano revisado");
    expect(items(result)).toHaveLength(5);

    click(result, "Excluir Plano revisado");
    expect(textContent(result.root)).toContain("Prompt excluído.");
    expect(items(result)).toHaveLength(4);
  });

  it("explica quando a importação não traz prompt válido", () => {
    const result = renderer({ storage: memoryStorage() });
    click(result, "Importar");
    fill(result, "prompt-import", "isto não é json");
    click(result, "Confirmar importação");
    expect(textContent(result.root.findByProps({ role: "alert" }))).toBe(
      "Nenhum prompt válido encontrado.",
    );
  });
});
