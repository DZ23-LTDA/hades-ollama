import { describe, expect, it, vi } from "vitest";
import { Settings as SettingsType } from "@/gotypes";
import { applySettingsDefaults } from "./settingsUtils";

/**
 * Contrato transacional do reset de configuracao: a etapa de
 * "atualizacao -> rollback" da jornada de primeira execucao. Se uma escrita
 * falha, tudo o que ja foi aplicado precisa ser revertido e o erro propagado,
 * sem chamar `onSaved`.
 */

function makeActions(overrides: Partial<Parameters<typeof applySettingsDefaults>[0]> = {}) {
  const currentSettings = new SettingsType({ ContextLength: 4096 });
  const updateSettings = vi.fn().mockResolvedValue(undefined);
  const updateCloud = vi.fn().mockResolvedValue(undefined);
  const updateShowAppsInMenu = vi.fn().mockResolvedValue(undefined);
  const resetChatGPTModels = vi.fn().mockResolvedValue(true);
  const resetClaudeMappings = vi.fn().mockResolvedValue(true);
  const onSaved = vi.fn();

  return {
    currentSettings,
    updateSettings,
    updateCloud,
    updateShowAppsInMenu,
    resetChatGPTModels,
    resetClaudeMappings,
    onSaved,
    actions: {
      updateSettings,
      updateCloud,
      updateShowAppsInMenu,
      resetChatGPTModels,
      resetClaudeMappings,
      currentSettings,
      currentShowAppsInMenu: true,
      cloudSource: "config" as const,
      onSaved,
      ...overrides,
    },
  };
}

describe("applySettingsDefaults", () => {
  it("aplica os padroes e confirma quando tudo funciona", async () => {
    const { actions, onSaved, updateSettings, updateCloud } = makeActions();

    await applySettingsDefaults(actions);

    expect(updateCloud).toHaveBeenCalledWith(true);
    expect(updateSettings).toHaveBeenCalledTimes(1);
    const applied = updateSettings.mock.calls[0][0] as SettingsType;
    expect(applied.Expose).toBe(false);
    expect(applied.Browser).toBe(false);
    expect(applied.Agent).toBe(false);
    expect(applied.Tools).toBe(false);
    expect(applied.ContextLength).toBe(4096);
    expect(applied.AutoUpdateEnabled).toBe(true);
    expect(onSaved).toHaveBeenCalledTimes(1);
  });

  it("reverte o que ja foi escrito quando o reset final falha", async () => {
    const { actions, currentSettings, onSaved, updateSettings, updateCloud } =
      makeActions();
    actions.resetClaudeMappings = vi.fn().mockResolvedValue(false);

    await expect(applySettingsDefaults(actions)).rejects.toThrow(
      "Claude model mappings could not be reset",
    );

    expect(onSaved).not.toHaveBeenCalled();
    // Rollback em ordem inversa: primeiro os apps no menu, depois as
    // configuracoes anteriores e por fim o estado de nuvem.
    expect(updateSettings).toHaveBeenCalledTimes(2);
    expect(updateSettings.mock.calls[1][0]).toBe(currentSettings);
    expect(updateCloud).toHaveBeenCalledTimes(2);
    expect(updateCloud.mock.calls[1][0]).toBe(false);
  });

  it("reverte quando uma escrita intermediaria falha", async () => {
    const { actions, onSaved, updateSettings, updateCloud, updateShowAppsInMenu } =
      makeActions();
    updateShowAppsInMenu.mockRejectedValueOnce(new Error("falha ao salvar menu"));

    await expect(applySettingsDefaults(actions)).rejects.toThrow(
      "falha ao salvar menu",
    );

    expect(onSaved).not.toHaveBeenCalled();
    expect(updateSettings).toHaveBeenCalledTimes(2);
    expect(updateCloud).toHaveBeenCalledTimes(2);
  });
});
