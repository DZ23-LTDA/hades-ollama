import { describe, expect, it } from "vitest";

import { Settings } from "@/gotypes";
import {
  applySettingsOverlay,
  __reiniciarEstadoPartilhadoDeSettings,
  definirGuardaDeEscritaDeSettings,
  guardaDeEscritaDeSettings,
  lerEstadoPartilhadoDeSettings,
  limparSobreposicaoDeSettings,
  marcarSettingsNaoPersistidas,
  settingsSnapshot,
  settingsWriteKey,
  shouldSkipSettingsWrite,
  sobreporSettings,
  type SettingsWriteGuard,
} from "./useSettings";

// Regressão do loop de /api/v1/settings: um backend que aceita a escrita mas
// devolve um payload sem o campo gravado transformava cada efeito "garanta X"
// em um ciclo grava -> invalida -> refetch -> grava. Medido na home com payload
// vazio: 2.387 requisições em 3s (1.206 POST + 1.181 GET).
describe("shouldSkipSettingsWrite", () => {
  const empty = new Settings({});
  const snapshot = settingsSnapshot(empty);

  it("libera a primeira escrita", () => {
    expect(shouldSkipSettingsWrite(null, "LastHomeView=\"chat\"", snapshot)).toBe(
      false,
    );
  });

  it("bloqueia a repetição enquanto a leitura não muda", () => {
    const guard: SettingsWriteGuard = {
      key: 'LastHomeView="chat"',
      snapshot,
    };

    // O efeito roda de novo a cada refetch porque LastHomeView continua ausente.
    let posts = 0;
    for (let i = 0; i < 100; i += 1) {
      if (!shouldSkipSettingsWrite(guard, 'LastHomeView="chat"', snapshot)) {
        posts += 1;
      }
    }
    expect(posts).toBe(0);
  });

  it("libera de novo quando o backend reflete alguma mudança", () => {
    const guard: SettingsWriteGuard = {
      key: 'LastHomeView="chat"',
      snapshot,
    };
    const reflected = settingsSnapshot(new Settings({ LastHomeView: "settings" }));

    expect(shouldSkipSettingsWrite(guard, 'LastHomeView="chat"', reflected)).toBe(
      false,
    );
  });

  it("libera quando a intenção de escrita é diferente", () => {
    const guard: SettingsWriteGuard = {
      key: 'LastHomeView="chat"',
      snapshot,
    };

    expect(shouldSkipSettingsWrite(guard, 'SelectedModel="llama3.2:3b"', snapshot)).toBe(
      false,
    );
  });

  it("nunca bloqueia uma escrita sem chave", () => {
    const guard: SettingsWriteGuard = { key: "x", snapshot };
    expect(shouldSkipSettingsWrite(guard, null, snapshot)).toBe(false);
  });
});

describe("settingsWriteKey", () => {
  it("é estável independente da ordem das propriedades", () => {
    expect(settingsWriteKey({ SelectedModel: "a", SidebarOpen: true })).toBe(
      settingsWriteKey({ SidebarOpen: true, SelectedModel: "a" }),
    );
  });

  it("distingue valores diferentes", () => {
    expect(settingsWriteKey({ SidebarOpen: true })).not.toBe(
      settingsWriteKey({ SidebarOpen: false }),
    );
  });

  it("retorna null para um patch vazio", () => {
    expect(settingsWriteKey({})).toBeNull();
  });
});

describe("settingsSnapshot", () => {
  it("é igual para payloads iguais e diferentes para payloads diferentes", () => {
    expect(settingsSnapshot(new Settings({ SelectedModel: "a" }))).toBe(
      settingsSnapshot(new Settings({ SelectedModel: "a" })),
    );
    expect(settingsSnapshot(new Settings({ SelectedModel: "a" }))).not.toBe(
      settingsSnapshot(new Settings({ SelectedModel: "b" })),
    );
  });
});

/**
 * 🚨 E-919 — **O TESTE DISCRIMINANTE DO «CHAT QUE NAO ENVIA».**
 *
 * O defeito medido (`E-917`/`E-918`): `useSettings` e' um hook simples, sem contexto, com **cinco**
 * consumidores. Cada um criava o **seu** `localOverlay`. O `ChatForm` escrevia no dele e **mostrava**
 * o modelo; o `useSelectedModel` - *de onde o `useSendMessage` le'* - via um overlay **vazio**, logo
 * `selectedModel` era `null` e o `mutationFn` fazia `throw new Error("No model selected")`
 * (`useChats.ts:302-304`) **antes** de qualquer `/api/chat`. *Medido: `0` pedidos a `/api/chat` e
 * `12` respostas `501`.*
 *
 * *Estes testes falhariam com o codigo anterior*, porque com estados locais nao havia nada para
 * partilhar: o segundo leitor veria sempre a sobreposicao inicial (vazia).
 */
describe("E-919: a sobreposicao de settings e' PARTILHADA entre consumidores", () => {
  it("uma escrita num consumidor e' vista por OUTRO (era isto que faltava)", () => {
    __reiniciarEstadoPartilhadoDeSettings();

    // Consumidor A (o `ChatForm`) escreve a escolha.
    sobreporSettings(() => ({ SelectedModel: "llama3.2:3b" }));

    // Consumidor B (o `useSelectedModel`, de onde o `useSendMessage` le') tem de a ver.
    const visto = lerEstadoPartilhadoDeSettings().overlay;
    expect(visto.SelectedModel).toBe("llama3.2:3b");
  });

  it("o `selectedModel` que o envio usa deixa de ser vazio (a causa do `throw`)", () => {
    __reiniciarEstadoPartilhadoDeSettings();

    // O servidor local responde 501, logo `settingsData.settings.SelectedModel` e' "".
    const doServidor = new Settings({ SelectedModel: "" });
    const antes = applySettingsOverlay(
      doServidor,
      lerEstadoPartilhadoDeSettings().overlay,
    );
    expect(antes.SelectedModel).toBe("");

    // O `ChatForm` escolhe o modelo.
    sobreporSettings(() => ({ SelectedModel: "llama3.2:3b" }));

    // *Agora o `useSelectedModel` ve' o mesmo* - e o `useSendMessage` deixa de lancar.
    const depois = applySettingsOverlay(
      doServidor,
      lerEstadoPartilhadoDeSettings().overlay,
    );
    expect(depois.SelectedModel).toBe("llama3.2:3b");
  });

  it("a bandeira de nao-persistencia tambem e' partilhada", () => {
    __reiniciarEstadoPartilhadoDeSettings();
    expect(lerEstadoPartilhadoDeSettings().persistFailed).toBe(false);

    marcarSettingsNaoPersistidas(true);
    expect(lerEstadoPartilhadoDeSettings().persistFailed).toBe(true);

    limparSobreposicaoDeSettings();
    expect(lerEstadoPartilhadoDeSettings().persistFailed).toBe(false);
  });

  it("a guarda de escrita e' UNICA (era uma por consumidor - os 11 POST)", () => {
    __reiniciarEstadoPartilhadoDeSettings();
    expect(guardaDeEscritaDeSettings()).toBeNull();

    const guarda: SettingsWriteGuard = {
      key: 'SelectedModel="llama3.2:3b"',
      snapshot: settingsSnapshot(new Settings({})),
    };
    definirGuardaDeEscritaDeSettings(guarda);
    expect(guardaDeEscritaDeSettings()).toBe(guarda);
  });

  it("a referencia so' muda quando o valor muda (a licao do E-782: 42 651 POST)", () => {
    __reiniciarEstadoPartilhadoDeSettings();

    sobreporSettings(() => ({ SelectedModel: "llama3.2:3b" }));
    const primeira = lerEstadoPartilhadoDeSettings().overlay;

    // Repetir a MESMA escrita devolve a MESMA referencia - *e' o que impede os efeitos de
    // re-executarem e o ciclo de POST de voltar*.
    sobreporSettings((anterior) => {
      const proximo = { ...anterior, SelectedModel: "llama3.2:3b" };
      const mudou = Object.keys(proximo).some(
        (c) => (anterior as Record<string, unknown>)[c] !== (proximo as Record<string, unknown>)[c],
      );
      return mudou ? proximo : anterior;
    });
    expect(lerEstadoPartilhadoDeSettings().overlay).toBe(primeira);

    // Uma intencao NOVA muda a referencia (nao fica travada).
    sobreporSettings((anterior) => ({ ...anterior, SelectedModel: "qwen2.5:7b" }));
    expect(lerEstadoPartilhadoDeSettings().overlay).not.toBe(primeira);
  });

  it("limpar a sobreposicao repoe o estado partilhado inicial", () => {
    __reiniciarEstadoPartilhadoDeSettings();
    sobreporSettings(() => ({ SelectedModel: "llama3.2:3b" }));
    marcarSettingsNaoPersistidas(true);

    limparSobreposicaoDeSettings();

    const estado = lerEstadoPartilhadoDeSettings();
    expect(estado.overlay).toEqual({});
    expect(estado.persistFailed).toBe(false);
  });
});
