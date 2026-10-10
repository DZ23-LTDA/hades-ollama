import { describe, expect, it } from "vitest";

import { Settings } from "@/gotypes";
import {
  settingsSnapshot,
  settingsWriteKey,
  shouldSkipSettingsWrite,
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
