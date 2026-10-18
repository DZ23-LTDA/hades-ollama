/**
 * FASE 28.8: testes do painel de ROLLBACK.
 *
 * O que estes testes travam:
 *
 *  1. o painel diz que **só planeia** antes de haver resultado;
 *  2. **não existe botão de restaurar** — a rota não oferece, o cliente não tem, o painel não inventa;
 *  3. um plano com candidatos mostra o alvo, o SHA-256 e `executado: nao`;
 *  4. o aviso de fronteira do backend é mostrado por inteiro;
 *  5. **canal não suportado mostra a explicação**, não um alerta vermelho;
 *  6. uma falha da rota aparece em `role="alert"`;
 *  7. o pedido leva **só o `tipo`** — nunca um `directorio`.
 *
 * Mocka-se **apenas o `agentFetch`**, não o cliente.
 */

import { act, create, type ReactTestRenderer } from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockAgentFetch = vi.fn();

vi.mock("@/lib/agenticClient", () => ({
  agentFetch: (...args: unknown[]) => mockAgentFetch(...args),
}));

import { RollbackPanel } from "@/components/RollbackPanel";

/** Plano de exemplo com um candidato. */
function planoDeExemplo(sobrescrever: Record<string, unknown> = {}) {
  return {
    executado: false,
    suportado: true,
    tipo: "WINDOWS_INSTALLER",
    alvo: {
      caminho: "C:/artefactos/HadesSetup-novo.exe",
      nome: "HadesSetup-novo.exe",
      tamanho_bytes: 1024,
      sha256: "a".repeat(64),
      modificado_em: "2026-10-18T12:00:00Z",
      parece_instalador: true,
    },
    total_de_candidatos: 2,
    integro: true,
    requer_aprovacao: "escolher o alvo e autorizar a restauracao",
    explicacao: "2 candidatos; o mais recente e' HadesSetup-novo.exe",
    aviso: "este plano NAO executou o rollback",
    aviso_de_fronteira:
      "esta rota apenas PLANEIA o rollback: executado e' sempre false -- nenhum ficheiro foi " +
      "movido, copiado ou apagado. O directorio vem da variavel de ambiente, nunca do pedido.",
    ...sobrescrever,
  };
}

/** Texto de toda a arvore. */
function textoDosNos(no: ReactTestRenderer): string {
  const partes: string[] = [];
  const visitar = (valor: unknown): void => {
    if (valor === null || valor === undefined) return;
    if (typeof valor === "string" || typeof valor === "number") {
      partes.push(String(valor));
      return;
    }
    if (Array.isArray(valor)) {
      valor.forEach(visitar);
      return;
    }
    if (typeof valor === "object" && "children" in valor) {
      visitar((valor as { children: unknown }).children);
    }
  };
  visitar(no.toJSON());
  return partes.join(" ");
}

async function montar(): Promise<ReactTestRenderer> {
  let arvore!: ReactTestRenderer;
  await act(async () => {
    arvore = create(<RollbackPanel />);
  });
  return arvore;
}

function botao(arvore: ReactTestRenderer) {
  return arvore.root.find(
    (no) => no.type === "button" && String(no.props.children).includes("Planear rollback"),
  );
}

beforeEach(() => {
  mockAgentFetch.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("RollbackPanel", () => {
  it("diz que so' planeia antes de haver resultado", async () => {
    const arvore = await montar();
    const texto = textoDosNos(arvore);
    expect(texto).toContain("planeia");
    expect(texto).toContain("Nada e' restaurado");
    arvore.unmount();
  });

  // NAO EXISTE BOTAO DE RESTAURAR.
  it("nao oferece nenhuma accao de restaurar", async () => {
    mockAgentFetch.mockResolvedValue(planoDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const botoes = arvore.root.findAll((no) => no.type === "button");
    // So' um botao: "Planear rollback". Nenhum "Restaurar", "Aplicar" ou "Executar".
    expect(botoes).toHaveLength(1);
    const rotulos = botoes.map((no) => String(no.props.children).toLowerCase());
    for (const rotulo of rotulos) {
      expect(rotulo).not.toContain("restaurar");
      expect(rotulo).not.toContain("aplicar");
      expect(rotulo).not.toContain("executar rollback");
    }
    arvore.unmount();
  });

  it("mostra o alvo, o sha256 e executado=nao", async () => {
    mockAgentFetch.mockResolvedValue(planoDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("HadesSetup-novo.exe");
    expect(texto).toContain("a".repeat(64));
    expect(arvore.root.findByProps({ "data-testid": "rollback-executado" }).props.children)
      .toBe("nao");
    arvore.unmount();
  });

  // O PEDIDO NAO LEVA DIRECTORIO.
  it("envia apenas o tipo, nunca um directorio", async () => {
    mockAgentFetch.mockResolvedValue(planoDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(mockAgentFetch).toHaveBeenCalledTimes(1);
    const [caminho, init] = mockAgentFetch.mock.calls[0] as [string, { body: string }];
    expect(caminho).toBe("/api/agent/v1/publish/rollback");
    const corpo = JSON.parse(init.body) as Record<string, unknown>;
    expect(Object.keys(corpo)).toEqual(["tipo"]);
    expect(corpo).not.toHaveProperty("directorio");
    arvore.unmount();
  });

  it("mostra o aviso de fronteira do backend por inteiro", async () => {
    mockAgentFetch.mockResolvedValue(planoDeExemplo());
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("nenhum ficheiro foi movido, copiado ou apagado");
    expect(texto).toContain("nunca do pedido");
    arvore.unmount();
  });

  // Canal nao suportado mostra a explicacao -- nao um alerta generico.
  it("mostra a explicacao de um canal nao suportado", async () => {
    mockAgentFetch.mockResolvedValue(
      planoDeExemplo({
        suportado: false,
        alvo: undefined,
        total_de_candidatos: 0,
        integro: false,
        requer_aprovacao: undefined,
        explicacao: "exige uma service account JSON com acesso ao Play Console",
      }),
    );
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("nao suportado neste ambiente");
    expect(texto).toContain("service account JSON");
    arvore.unmount();
  });

  it("mostra uma falha em role=alert", async () => {
    mockAgentFetch.mockRejectedValue(new Error("412 raiz de artefactos nao configurada"));
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    expect(arvore.root.findByProps({ role: "alert" })).toBeDefined();
    expect(textoDosNos(arvore)).toContain("412");
    arvore.unmount();
  });

  // Um servidor que declarasse `executado: true` nao pode ser escondido pelo painel.
  it("grita se o servidor declarar que executou", async () => {
    mockAgentFetch.mockResolvedValue(planoDeExemplo({ executado: true }));
    const arvore = await montar();
    await act(async () => {
      botao(arvore).props.onClick();
    });
    const texto = textoDosNos(arvore);
    expect(texto).toContain("ATENCAO");
    expect(texto).toContain("EXECUTOU");
    expect(arvore.root.findByProps({ "data-testid": "rollback-executado" }).props.children)
      .toBe("SIM");
    arvore.unmount();
  });
});
