import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

/**
 * `E-1089` — a **catraca de performance do pet** (§FASE-3-A itens 5 e 9).
 *
 * ***PORQUE ESTE TESTE EXISTE.*** O prompt exige *«medir regressao de performance e vazamento:
 * listeners, timers, SSE/WebSocket, memoria, CPU, bateria e cleanup ao desmontar»*. ⇒ *o pet é um
 * componente que fica **montado** na página enquanto a missão está aberta: um `setInterval` ou um
 * `addEventListener` sem cleanup ali **acumula**, e não há teste de UI que o apanhe (o projecto não
 * tem `jsdom`).*
 *
 * ***O QUE ELE MEDE, E PORQUE ASSIM.*** Mede o **fonte** — e não o render — por uma razão medida no
 * `E-1086`: o `vitest.config.ts` corre em `environment: "node"`, o `renderToStaticMarkup` **não corre
 * `useEffect`**, e não há `@testing-library/react`. ⇒ *um teste que montasse o componente nunca
 * exercitaria os efeitos; o que **é** verificável é a **presença** do timer/listener e do cleanup.*
 *
 * ***E O QUE ELE NÃO PROVA.*** *Não mede CPU, RAM nem bateria — isso exigiria um browser com o
 * componente montado por minutos.* ⇒ *mede a **causa** que esses sintomas teriam: um timer, um
 * listener, ou um fetch sem cleanup.*
 */

const DIR = resolve(__dirname);
const COMPONENTES = ["CompanionMascote.tsx", "CompanionLigado.tsx"] as const;
const MODULOS = ["../lib/companionPreferencias.ts", "../lib/companionApi.ts"] as const;

function ler(nome: string): string {
  return readFileSync(resolve(DIR, nome), "utf8");
}

/** Conta ocorrências de um padrão, ignorando comentários de linha (`//`) e de bloco. */
function contarSemComentarios(fonte: string, padrao: RegExp): number {
  const semBloco = fonte.replace(/\/\*[\s\S]*?\*\//g, "");
  const semLinha = semBloco
    .split("\n")
    .map((l) => l.replace(/\/\/.*$/, ""))
    .join("\n");
  return (semLinha.match(padrao) ?? []).length;
}

describe("Companion — a catraca de performance (E-1089)", () => {
  it("NENHUM timer no pet -- um setInterval acumularia enquanto a missao esta' aberta", () => {
    for (const nome of COMPONENTES) {
      const fonte = ler(nome);
      expect(contarSemComentarios(fonte, /\bsetInterval\b/g), `${nome} tem setInterval`).toBe(0);
      expect(contarSemComentarios(fonte, /\bsetTimeout\b/g), `${nome} tem setTimeout`).toBe(0);
      expect(contarSemComentarios(fonte, /\brequestAnimationFrame\b/g), `${nome} tem rAF`).toBe(0);
    }
  });

  it("NENHUM listener global no pet -- um addEventListener sem cleanup vaza a cada remount", () => {
    for (const nome of COMPONENTES) {
      const fonte = ler(nome);
      expect(
        contarSemComentarios(fonte, /\baddEventListener\b/g),
        `${nome} tem addEventListener`,
      ).toBe(0);
      expect(contarSemComentarios(fonte, /\bwindow\./g), `${nome} toca em window`).toBe(0);
      expect(contarSemComentarios(fonte, /\bdocument\./g), `${nome} toca em document`).toBe(0);
    }
  });

  it("os modulos de preferencias/API sao PUROS -- nenhum efeito, nenhum timer", () => {
    for (const nome of MODULOS) {
      const fonte = ler(nome);
      expect(contarSemComentarios(fonte, /\bsetInterval\b|\bsetTimeout\b/g), nome).toBe(0);
      expect(
        contarSemComentarios(fonte, /\baddEventListener\b/g),
        nome,
      ).toBe(0);
    }
  });

  it("o fetch do estado tem CLEANUP -- a resposta tardia nao escreve num componente desmontado", () => {
    const fonte = ler("CompanionLigado.tsx");
    // *O padrão medido no ficheiro: `let vivo = true` ... `return () => { vivo = false; }`.*
    expect(fonte).toMatch(/let vivo = true/);
    expect(fonte).toMatch(/return \(\) => \{\s*vivo = false;/);
    // *E a guarda depois do `await`: sem ela, o `cleanup` seria decoração.*
    expect(fonte).toMatch(/if \(!vivo\) return/);
  });

  it("e o cleanup nao e' decoracao: existe a guarda DEPOIS do await", () => {
    const fonte = ler("CompanionLigado.tsx");
    const depoisDoAwait = fonte.split("carregarEstadoDoCompanion(missionID)")[1] ?? "";
    // *Sem o `if (vivo)`, o `setState` de uma resposta tardia dispara num componente desmontado.*
    expect(depoisDoAwait).toMatch(/if \(!vivo\) return/);
    expect(depoisDoAwait).toMatch(/if \(vivo\) setResultado/);
  });

  it("o pet NAO anima com movimento reduzido -- ja' provado no componente, repetido aqui como catraca", () => {
    const fonte = ler("CompanionMascote.tsx");
    // *A decisão de parar vem do `razaoParaNaoAnimar`, e nao de um estado interno do componente.*
    expect(fonte).toContain("razaoParaNaoAnimar");
    expect(fonte).toContain("companion-idle");
  });

  it("o pet e' DESLIGAVEL -- `visivel: false` devolve `null`, e nao um contentor vazio", () => {
    const fonte = ler("CompanionMascote.tsx");
    expect(fonte).toMatch(/if \(!podeDesenharCompanion\(preferencias, estado\)\) \{\s*return null;/);
  });
});
