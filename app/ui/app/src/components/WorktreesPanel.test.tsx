import { act, create, type ReactTestInstance, type ReactTestRenderer } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WorktreesPanel } from "./WorktreesPanel";

// FASE 2 do adendo V3: o painel de WORKTREES dentro da pagina de Capacidades.
//
// O que estes testes travam:
//   1. o painel PEDE a raiz do repositorio em vez de adivinhar o checkout;
//   2. worktree externo aparece marcado como externo;
//   3. um estado FORA da lista fechada aparece como desconhecido -- o painel NAO inventa rotulo;
//   4. trabalho nao integrado aparece como alerta, nunca como sucesso;
//   5. `pode_remover` falso mostra o MOTIVO;
//   6. erro de rede e' anunciado;
//   7. NAO existe botao de remover:
//      *remover exige uma decisao sobre trabalho nao integrado, e essa decisao nao cabe num
//      clique que o painel nao sabe explicar.*

// Mockar o TRANSPORTE (`agentFetch`), nao o cliente: assim o `worktreeClient` real corre --
// incluindo `rotuloDoEstadoDoWorktree`, `corDoEstadoDoWorktree` e `nomeCurtoDoCaminho`.
//
//	*Um mock que substitui o modulo que se quer testar deixa de testar o modulo.*
const { agentFetchMock } = vi.hoisted(() => ({ agentFetchMock: vi.fn() }));

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: agentFetchMock,
}));

/**
 * Responde a DUAS perguntas diferentes com respostas diferentes.
 *
 *	*Um mock que devolve a mesma resposta para perguntas diferentes testa o mock, nao o painel.*
 */
function responderCom(inventario: unknown, diagnostico?: unknown) {
  agentFetchMock.mockImplementation((url: string) => {
    if (url.includes("/diagnose")) {
      if (diagnostico === undefined) return Promise.reject(new Error("diagnostico nao esperado"));
      return Promise.resolve(diagnostico);
    }
    return Promise.resolve(inventario);
  });
}

const INVENTARIO = {
  repo: "/repo",
  resumo: "3 worktree(s): 2 do HADES, 1 externo(s)",
  total: 3,
  do_hades: 2,
  externo: 1,
  lista: [
    {
      caminho_resolvido: "/repo",
      branch: "main",
      commit: "abc",
      detached: false,
      criado_pelo_hades: false,
      principal: true,
      bloqueado: false,
    },
    {
      caminho_resolvido: "/repo/.agent-worktrees/missao-a",
      branch: "agent/missao-a",
      commit: "def",
      detached: false,
      criado_pelo_hades: true,
      principal: false,
      bloqueado: false,
    },
    {
      caminho_resolvido: "/outro/externo",
      branch: "externo/um",
      commit: "ghi",
      detached: false,
      criado_pelo_hades: false,
      principal: false,
      bloqueado: false,
    },
  ],
  avisos: ["existem worktrees que o HADES nao criou"],
  somente_leitura: true,
  medido_em: "2026-10-18T12:00:00Z",
};

const DIAGNOSTICO_OK = {
  mission_id: "missao-a",
  estado: "preparado",
  estado_valido: true,
  estado_traduzido: false,
  estado_observado: "existe no inventario do git; o estado logico pertence ao processo",
  proximos: ["em_uso", "suspenso"],
  estado_do_registo: false,
  registo: "ausente: nenhuma entrada persistida neste data root",
  alvo_seguro: true,
  commits_nao_integrados: 0,
  ficheiros_modificados: 0,
  tem_trabalho_nao_integrado: false,
  pode_remover: true,
};

/** Todos os botoes da arvore, para provar ausencia de accoes destrutivas. */
function botoes(arvore: ReactTestRenderer): ReactTestInstance[] {
  return arvore.root.findAll(
    (n) => n.type === "button" && typeof n.props.onClick === "function",
  );
}

function texto(node: ReactTestInstance | string): string {
  if (typeof node === "string") return node;
  return node.children.map((c) => texto(c as ReactTestInstance | string)).join("");
}

async function montar(): Promise<ReactTestRenderer> {
  let arvore!: ReactTestRenderer;
  await act(async () => {
    arvore = create(<WorktreesPanel />);
  });
  return arvore;
}

async function esperar(arvore: ReactTestRenderer, contem: string, tentativas = 40): Promise<void> {
  for (let i = 0; i < tentativas; i++) {
    if (texto(arvore.root).includes(contem)) return;
    await act(async () => {
      await new Promise((r) => setTimeout(r, 10));
    });
  }
  throw new Error(`texto "${contem}" nao apareceu; obtido: ${texto(arvore.root).slice(0, 400)}`);
}

async function escreverRepo(arvore: ReactTestRenderer, valor: string): Promise<void> {
  const input = arvore.root.find((n) => n.type === "input");
  await act(async () => {
    input.props.onChange({ target: { value: valor } });
  });
}

async function clicarBotao(arvore: ReactTestRenderer, rotulo: string): Promise<void> {
  const botao = arvore.root.find(
    (n) => n.type === "button" && texto(n).trim() === rotulo,
  );
  await act(async () => {
    botao.props.onClick();
  });
}

describe("WorktreesPanel", () => {
  beforeEach(() => {
    agentFetchMock.mockReset();
  });

  it("pede a raiz do repositorio em vez de adivinhar o checkout", async () => {
    const arvore = await montar();
    const t = texto(arvore.root);
    expect(t).toContain("Raiz do repositório");
    expect(t).toContain("não adivinha o checkout");
    expect(agentFetchMock).not.toHaveBeenCalled();
  });

  it("mostra o inventario com externos marcados e avisos", async () => {
    responderCom(INVENTARIO);
    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "3 worktree(s): 2 do HADES, 1 externo(s)");

    const t = texto(arvore.root);
    expect(t).toContain("externo");
    expect(t).toContain("HADES");
    expect(t).toContain("principal");
    expect(t).toContain("worktrees que o HADES nao criou");

    // O aviso e' anunciado, nao texto decorativo.
    const alertas = arvore.root.findAll(
      (n) => n.props?.role === "alert" && texto(n).includes("worktrees que o HADES nao criou"),
    );
    expect(alertas.length).toBeGreaterThan(0);
  });

  it("mostra o diagnostico de uma missao", async () => {
    responderCom(INVENTARIO, DIAGNOSTICO_OK);
    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "missao-a");
    await clicarBotao(arvore, "missao-a");

    await esperar(arvore, "Preparado");
    const t = texto(arvore.root);
    expect(t).toContain("Pode ser removido sem perda");
    expect(t).toContain("A seguir:");
  });

  it("mostra trabalho nao integrado em alerta e explica por que nao pode remover", async () => {
    responderCom(INVENTARIO, {
      ...DIAGNOSTICO_OK,
      tem_trabalho_nao_integrado: true,
      commits_nao_integrados: 2,
      ficheiros_modificados: 1,
      pode_remover: false,
      motivo_remover: "2 commit(s) nao integrado(s)",
    });
    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "missao-a");
    await clicarBotao(arvore, "missao-a");

    await esperar(arvore, "Trabalho não integrado");
    const t = texto(arvore.root);
    expect(t).toContain("2 commit(s)");
    expect(t).toContain("Não pode ser removido: 2 commit(s) nao integrado(s)");
  });

  it("NAO inventa rotulo para um estado fora da lista fechada", async () => {
    responderCom(INVENTARIO, {
      ...DIAGNOSTICO_OK,
      estado: "quase_pronto",
      estado_valido: false,
    });
    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "missao-a");
    await clicarBotao(arvore, "missao-a");

    await esperar(arvore, "Estado desconhecido");
    const t = texto(arvore.root);
    expect(t).toContain("fora da lista fechada");
    //	*Um painel que inventa um rotulo para um estado que nao conhece ensina o utilizador a
    //	confiar no que nao foi verificado.*
    expect(t).not.toContain("quase_pronto");
  });

  it("mostra a inconsistencia entre estado e disco", async () => {
    responderCom(INVENTARIO, {
      ...DIAGNOSTICO_OK,
      inconsistente: "estado diz integrado mas o worktree tem trabalho por integrar",
    });
    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "missao-a");
    await clicarBotao(arvore, "missao-a");

    await esperar(arvore, "Inconsistência:");
  });

  it("anuncia erro de rede", async () => {
    agentFetchMock.mockImplementation(() => Promise.reject(new Error("falha de rede")));
    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "falha de rede");

    const alertas = arvore.root.findAll(
      (n) => n.props?.role === "alert" && texto(n).includes("falha de rede"),
    );
    expect(alertas.length).toBeGreaterThan(0);
  });

  it("NAO oferece botao de remover", async () => {
    responderCom(INVENTARIO, DIAGNOSTICO_OK);
    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "missao-a");
    await clicarBotao(arvore, "missao-a");
    await esperar(arvore, "Preparado");

    //	*Remover exige uma decisao sobre trabalho nao integrado, e essa decisao nao cabe num
    //	clique que o painel nao sabe explicar.*
    for (const b of arvore.root.findAll((n) => n.type === "button")) {
      expect(texto(b).toLowerCase()).not.toMatch(/remover|apagar|eliminar/);
    }
  });

  /**
   * O painel diz de ONDE veio o estado.
   *
   *     *Um estado que morre com o processo nao e' um estado: e' uma variavel.*
   *
   * Sem esta distincao "suspenso porque eu suspendi" e "a pasta existe" parecem a mesma coisa.
   */
  it("distingue estado persistido de estado lido do disco", async () => {
    responderCom(INVENTARIO, {
      ...DIAGNOSTICO_OK,
      estado: "suspenso",
      estado_do_registo: true,
      registo: "consultado",
    });
    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "missao-a");
    await clicarBotao(arvore, "missao-a");

    await esperar(arvore, "consultado (estado persistido)");
    // E nao mostra o alerta de registo ilegivel.
    if (texto(arvore.root).includes("não pôde ser lido")) {
      throw new Error("alerta de registo ilegivel mostrado num registo valido");
    }
  });

  /**
   * Registo ILEGIVEL aparece como ilegivel -- nunca como ausente.
   *
   *     *Um registo corrompido lido como vazio apagaria o estado de todos os worktrees.*
   */
  it("avisa quando o registo existe mas nao pode ser lido", async () => {
    responderCom(INVENTARIO, {
      ...DIAGNOSTICO_OK,
      estado_do_registo: false,
      registo: "ilegivel: registo de worktrees corrompido",
    });
    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "missao-a");
    await clicarBotao(arvore, "missao-a");

    await esperar(arvore, "não pôde ser lido");

  /**
   * As UNICAS accoes expostas sao' as reversiveis.
   *
   *     *Remover exige uma decisao sobre trabalho nao integrado, e essa decisao nao cabe num
   *     clique que o painel nao sabe explicar.*
   *
   * Este teste falha se alguem acrescentar um botao de remover.
   */
  });
  it("oferece suspender, retomar e arquivar -- e NAO oferece remover", async () => {
    responderCom(INVENTARIO, DIAGNOSTICO_OK);
    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "missao-a");
    await clicarBotao(arvore, "missao-a");
    await esperar(arvore, "Preparado");

    const rotulos = botoes(arvore).map((b) => texto(b).trim());
    for (const accao of ["Suspender", "Retomar", "Arquivar"]) {
      if (!rotulos.includes(accao)) {
        throw new Error(`botao "${accao}" ausente; encontrados: ${rotulos.join(", ")}`);
      }
    }
    const proibidos = rotulos.filter((r) => /remov|apag|exclu/i.test(r));
    if (proibidos.length > 0) {
      throw new Error(`botao destrutivo exposto no painel: ${proibidos.join(", ")}`);
    }
  });

  /**
   * O painel RELE' o backend depois de mudar o estado.
   *
   *     *Um painel que actualiza o ecra sem reler o backend mostra o que o programador achou que
   *     ia acontecer, nao o que aconteceu.*
   */
  it("rele o diagnostico depois de mudar o estado", async () => {
    // Primeira leitura: preparado. Depois do POST, a releitura devolve suspenso.
    let leituras = 0;
    agentFetchMock.mockImplementation((url: string) => {
      if (url.includes("/worktrees/state")) {
        return Promise.resolve({
          mission_id: "missao-a",
          de: "preparado",
          para: "suspenso",
          acao: "suspender",
          estado_do_registo: true,
          registado_em: "/repo/_state/worktree-state.json",
          avisos: ["o estado foi persistido"],
          somente_leitura: false,
        });
      }
      if (url.includes("/diagnose")) {
        leituras += 1;
        return Promise.resolve({
          ...DIAGNOSTICO_OK,
          estado: leituras >= 2 ? "suspenso" : "preparado",
          estado_do_registo: leituras >= 2,
        });
      }
      return Promise.resolve(INVENTARIO);
    });

    const arvore = await montar();
    await escreverRepo(arvore, "/repo");
    await esperar(arvore, "missao-a");
    await clicarBotao(arvore, "missao-a");
    await esperar(arvore, "Preparado");

    await clicarBotao(arvore, "Suspender");
    // A releitura tem de aparecer: o painel mostra "Suspenso", nao "Preparado".
    await esperar(arvore, "Suspenso");
    if (leituras < 2) {
      throw new Error(`o painel nao releu o diagnostico (leituras=${leituras})`);
    }
  });
});
