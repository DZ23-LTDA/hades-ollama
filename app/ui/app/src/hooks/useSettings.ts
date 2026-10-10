import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Settings } from "@/gotypes";
import { getSettings, updateSettings } from "@/api";
import { useMemo, useCallback, useRef } from "react";

// TODO(hoyyeva): remove turboEnabled when we remove Migration logic in useSelectedModel.ts
interface SettingsState {
  turboEnabled: boolean;
  webSearchEnabled: boolean;
  selectedModel: string;
  sidebarOpen: boolean;
  lastHomeView: string;
  onboardingVersion: number;
  thinkEnabled: boolean;
  thinkLevel: string;
}

// Type for partial settings updates
type SettingsUpdate = Partial<{
  TurboEnabled: boolean;
  WebSearchEnabled: boolean;
  ThinkEnabled: boolean;
  ThinkLevel: string;
  SelectedModel: string;
  SidebarOpen: boolean;
  LastHomeView: string;
  OnboardingVersion: number;
}>;

// Estabiliza a chave de uma escrita: a mesma intenção produz sempre a mesma
// string, independente da ordem das propriedades.
export function settingsWriteKey(updates: SettingsUpdate): string | null {
  const keys = Object.keys(updates).sort() as (keyof SettingsUpdate)[];
  if (keys.length === 0) return null;
  return keys.map((key) => `${key}=${JSON.stringify(updates[key])}`).join("&");
}

// Fotografia estável do que o backend devolveu; usada para detectar se uma
// escrita foi ou não refletida na leitura seguinte.
export function settingsSnapshot(settings: Settings): string {
  return JSON.stringify(settings);
}

export interface SettingsWriteGuard {
  key: string;
  snapshot: string;
}

// shouldSkipSettingsWrite evita repetir uma escrita idêntica enquanto o estado
// observado não mudou. Pure, para teste.
//
// Motivo (bug real): quando o /api/v1/settings aceita a escrita mas devolve um
// payload sem o campo gravado (parcial, proxy ou endpoint somente leitura),
// todo efeito "garanta X" — trocar LastHomeView para "chat", escolher o modelo
// padrão — reexecutava a cada invalidação da query: grava -> invalida ->
// refetch -> mesmo valor ausente -> grava de novo. Medido na home com payload
// vazio: 2.387 requisições a /api/v1/settings em 3s (~1.200 POST/s + ~1.180
// GET/s), saturando o renderer. Enquanto o snapshot observado não muda, a
// repetição é inútil por definição; assim que o backend reflete qualquer
// mudança, uma nova escrita é permitida.
export function shouldSkipSettingsWrite(
  guard: SettingsWriteGuard | null,
  key: string | null,
  snapshot: string,
): boolean {
  if (key === null || guard === null) return false;
  return guard.key === key && guard.snapshot === snapshot;
}

export function useSettings({
  refetchInterval,
}: { refetchInterval?: number } = {}) {
  const queryClient = useQueryClient();

  // Última escrita enviada e o snapshot observado quando ela foi enviada.
  const writeGuardRef = useRef<SettingsWriteGuard | null>(null);

  // Fetch settings with useQuery
  const { data: settingsData, error } = useQuery({
    queryKey: ["settings"],
    queryFn: getSettings,
    refetchInterval,
  });

  // Update settings with useMutation
  const updateSettingsMutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: () => {
      // Invalidate the query to ensure fresh data
      queryClient.invalidateQueries({ queryKey: ["settings"] });
    },
    onError: () => {
      // Uma escrita que falhou não foi aceita: libera a repetição para que o
      // usuário (ou a tela de onboarding) possa tentar de novo explicitamente.
      writeGuardRef.current = null;
    },
  });

  // Extract settings with defaults
  const settings: SettingsState = useMemo(
    () => ({
      turboEnabled: settingsData?.settings?.TurboEnabled ?? false,
      webSearchEnabled: settingsData?.settings?.WebSearchEnabled ?? false,
      thinkEnabled: settingsData?.settings?.ThinkEnabled ?? false,
      thinkLevel: settingsData?.settings?.ThinkLevel ?? "none",
      selectedModel: settingsData?.settings?.SelectedModel ?? "",
      sidebarOpen: settingsData?.settings?.SidebarOpen ?? false,
      lastHomeView: settingsData?.settings?.LastHomeView ?? "chat",
      onboardingVersion: settingsData?.settings?.OnboardingVersion ?? 0,
    }),
    [settingsData?.settings],
  );

  // Single function to update most settings
  const setSettings = useCallback(
    async (updates: SettingsUpdate) => {
      if (!settingsData?.settings) return;

      const key = settingsWriteKey(updates);
      const snapshot = settingsSnapshot(settingsData.settings);
      if (shouldSkipSettingsWrite(writeGuardRef.current, key, snapshot)) {
        return;
      }
      writeGuardRef.current = key === null ? null : { key, snapshot };

      const updatedSettings = new Settings({
        ...settingsData.settings,
        ...updates,
      });

      await updateSettingsMutation.mutateAsync(updatedSettings);
    },
    [settingsData?.settings, updateSettingsMutation],
  );

  return useMemo(
    () => ({
      settings,
      settingsData: settingsData?.settings,
      error,
      setSettings,
    }),
    [settings, settingsData?.settings, error, setSettings],
  );
}
