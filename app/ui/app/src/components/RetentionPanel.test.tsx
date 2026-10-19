// Testes do painel de retenção.
//
// O que cada caso trava, e porque falharia com o defeito:
//
//   - "não configurada" tem de ser um ESTADO, não um erro. Se aparecer como
//     erro, o utilizador pensa que o produto está quebrado quando está apenas
//     à espera de uma decisão dele.
//   - `max_memories_per_project = 0` tem de ser dito como "sem tecto". Mostrar
//     "0 memórias por projecto" diz o contrário do que a política faz.
//   - gravar NÃO pode dizer que removeu nada: gravar e aplicar são passos
//     separados, e um aviso que os confunde ensina o utilizador a temer o botão
//     errado.
//   - o resultado da aplicação tem de mostrar os números REAIS; "sucesso" sem
//     números não é informação.
//   - o 400 do servidor tem de aparecer. Um painel que engole o erro deixa o
//     utilizador a adivinhar porque é que o botão não fez nada.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: vi.fn(),
}));

import { agentFetch } from "@/lib/agenticClient";
import { RetentionPanel } from "@/components/RetentionPanel";

const mockAgentFetch = vi.mocked(agentFetch);

// `create()` do react-test-renderer e' SINCRONO -- nao devolve Promise. Envolver
// em `act` e' o que deixa os efeitos correrem antes de a assercao olhar.
async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<RetentionPanel />);
  });
  return renderer;
}

/** Corre uma accao do utilizador dentro de `act` e deixa os efeitos terminarem. */
async function agir(accao: () => unknown) {
  await act(async () => {
    await accao();
  });
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll(
    (nodo) => nodo.props["data-testid"] === id,
  );
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("RetentionPanel", () => {
  it("diz que a organização não tem política em vez de mostrar erro", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      organization_id: "org-a",
      configured: false,
    });

    const renderer = await montar();

    expect(porTestId(renderer, "retention-por-configurar")).toHaveLength(1);
    expect(porTestId(renderer, "retention-erro")).toHaveLength(0);
  });

  it("mostra a política configurada e o resumo em palavras", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      organization_id: "org-a",
      configured: true,
      policy: {
        organization_id: "org-a",
        max_age_days: 30,
        max_memories_per_project: 500,
        updated_at: "2026-10-19T00:00:00Z",
      },
    });

    const renderer = await montar();

    const resumo = porTestId(renderer, "retention-resumo");
    expect(resumo).toHaveLength(1);
    expect(resumo[0].children.join("")).toContain("30 dias");
    expect(resumo[0].children.join("")).toContain("500 memórias por projecto");
  });

  it("diz 'sem tecto' quando o limite é zero, em vez de '0 memórias'", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      organization_id: "org-a",
      configured: true,
      policy: {
        organization_id: "org-a",
        max_age_days: 90,
        max_memories_per_project: 0,
      },
    });

    const renderer = await montar();

    const resumo = porTestId(renderer, "retention-resumo")[0].children.join("");
    expect(resumo).toContain("sem tecto");
    expect(resumo).not.toContain("0 memórias");
  });

  it("gravar diz que nada foi removido -- gravar não é aplicar", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({
        organization_id: "org-a",
        configured: false,
      })
      .mockResolvedValueOnce({
        organization_id: "org-a",
        max_age_days: 90,
        max_memories_per_project: 0,
      });

    const renderer = await montar();
    const botao = porTestId(renderer, "retention-gravar")[0];
    await agir(() => botao.props.onClick());

    const aviso = porTestId(renderer, "retention-aviso-gravacao");
    expect(aviso).toHaveLength(1);
    expect(aviso[0].children.join("")).toContain("Nada foi removido ainda");
  });

  it("aplicar mostra os números reais do que removeu", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({
        organization_id: "org-a",
        configured: true,
        policy: {
          organization_id: "org-a",
          max_age_days: 30,
          max_memories_per_project: 10,
        },
      })
      .mockResolvedValueOnce({
        organization_id: "org-a",
        projects_scanned: 3,
        removed_by_age: 7,
        removed_by_count: 2,
        applied_at: "2026-10-19T00:00:00Z",
      });

    const renderer = await montar();
    const botao = porTestId(renderer, "retention-aplicar")[0];
    await botao.props.onClick();
    await new Promise((resolver) => setTimeout(resolver, 0));

    const resultado = porTestId(renderer, "retention-resultado");
    expect(resultado).toHaveLength(1);
    const texto = resultado[0].children.join("");
    expect(texto).toContain("9 memórias");
    expect(texto).toContain("7 por idade");
    expect(texto).toContain("2 por excesso");
    expect(texto).toContain("3 projectos");
  });

  it("aplicar sem nada a remover diz isso em palavras, não '0'", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({
        organization_id: "org-a",
        configured: true,
        policy: {
          organization_id: "org-a",
          max_age_days: 30,
          max_memories_per_project: 10,
        },
      })
      .mockResolvedValueOnce({
        organization_id: "org-a",
        projects_scanned: 2,
        removed_by_age: 0,
        removed_by_count: 0,
      });

    const renderer = await montar();
    await agir(() => porTestId(renderer, "retention-aplicar")[0].props.onClick());

    const texto = porTestId(renderer, "retention-resultado")[0].children.join("");
    expect(texto).toContain("Nada a remover");
    expect(texto).toContain("2 projectos");
  });

  it("recusa idade fora do contrato sem chamar o servidor", async () => {
    mockAgentFetch.mockResolvedValueOnce({
      organization_id: "org-a",
      configured: false,
    });

    const renderer = await montar();
    const campo = porTestId(renderer, "retention-idade")[0];
    // Dois `act` separados: o primeiro aplica o estado E deixa re-renderizar; o
    // segundo clica ja' com a idade invalida no `gravar` novo.
    await agir(() => campo.props.onChange({ target: { value: "0" } }));
    await agir(() => porTestId(renderer, "retention-gravar")[0].props.onClick());

    // Só a leitura inicial; o PUT não pode ter acontecido.
    expect(mockAgentFetch).toHaveBeenCalledTimes(1);
    const erro = porTestId(renderer, "retention-erro");
    expect(erro).toHaveLength(1);
    expect(erro[0].children.join("")).toContain("entre 1 e 3650");
  });

  it("mostra o erro do servidor em vez de o engolir", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("retention policy is invalid"));

    const renderer = await montar();

    const erro = porTestId(renderer, "retention-erro");
    expect(erro).toHaveLength(1);
    expect(erro[0].children.join("")).toContain("retention policy is invalid");
  });
});
