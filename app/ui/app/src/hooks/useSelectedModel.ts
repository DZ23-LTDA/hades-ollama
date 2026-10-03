import { useEffect, useMemo, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useModels } from "./useModels";
import { useChat } from "./useChats";
import { useSettings } from "./useSettings.ts";
import { Model } from "@/gotypes";
import { getTotalVRAM } from "@/utils/vram.ts";
import { getInferenceCompute } from "@/api";
import { useCloudStatus } from "./useCloudStatus";

export function recommendDefaultModel(totalVRAM: number): string {
  const vram = Math.max(0, Number(totalVRAM) || 0);

  if (vram < 6) {
    return "gemma3:1b";
  } else if (vram < 16) {
    return "gemma3:4b";
  }
  return "gpt-oss:20b";
}

// The plain chat only supports local models and Ollama Cloud. The DZ23
// multi-provider router models (kind "router", e.g. auto/coding) and external
// provider models (kind "remote", e.g. groq/...) only work through the Agentic
// Console, and return 404 in the plain chat — so they must never be selected or
// defaulted to here.
export function isChatModel(model: Model): boolean {
  if (model.isCloud?.()) return true;
  const kind = model.kind;
  return kind === undefined || kind === null || kind === "local";
}

// isInstalledLocal is true for a local model whose weights are already on disk
// (has a digest), i.e. something the chat can run immediately offline.
function isInstalledLocal(model: Model): boolean {
  return (
    isChatModel(model) &&
    !model.isCloud?.() &&
    typeof model.digest === "string" &&
    model.digest !== ""
  );
}

// pickChatDefault chooses a model the plain chat can actually use, preferring an
// installed local model (works offline, no usage limits) over cloud.
export function pickChatDefault(
  models: Model[],
  recommendedModel: string,
  cloudDisabled: boolean,
): Model | null {
  const recommended = models.find(
    (m) => m.model === recommendedModel && isChatModel(m) && (isInstalledLocal(m) || m.isCloud?.()),
  );
  return (
    recommended ||
    models.find(isInstalledLocal) ||
    (cloudDisabled
      ? models.find((m) => isChatModel(m) && !m.isCloud?.())
      : models.find((m) => m.isCloud?.())) ||
    models.find(isChatModel) ||
    null
  );
}

export function useSelectedModel(currentChatId?: string, searchQuery?: string) {
  const { settings, setSettings } = useSettings();
  const { data: models = [], isLoading } = useModels(searchQuery || "");
  const { cloudDisabled } = useCloudStatus();
  const { data: chatData, isLoading: isChatLoading } = useChat(
    currentChatId && currentChatId !== "new" ? currentChatId : "",
  );

  const { data: inferenceComputeResponse } = useQuery({
    queryKey: ["inferenceCompute"],
    queryFn: getInferenceCompute,
    enabled: !settings.selectedModel, // Only fetch if no model is selected
  });

  const inferenceComputes = useMemo(
    () => inferenceComputeResponse?.inferenceComputes || [],
    [inferenceComputeResponse?.inferenceComputes],
  );

  const totalVRAM = useMemo(
    () => getTotalVRAM(inferenceComputes),
    [inferenceComputes],
  );

  const recommendedModel = useMemo(
    () => recommendDefaultModel(totalVRAM),
    [totalVRAM],
  );

  // Track which chat we've already restored the model for
  const restoredChatRef = useRef<string | null>(null);

  const selectedModel: Model | null = useMemo(() => {
    // If cloud is disabled and selected model ends with cloud, switch to a local default.
    if (cloudDisabled && settings.selectedModel?.endsWith("cloud")) {
      return (
        models.find((m) => m.model === recommendedModel) ||
        models.find((m) => !m.isCloud()) ||
        models.find((m) => m.digest === undefined || m.digest === "") ||
        models[0] ||
        null
      );
    }

    // Migration logic: if turboEnabled is true and selectedModel is a base model,
    // migrate to the cloud version and disable turboEnabled permanently
    // TODO: remove this logic in a future release
    const baseModelsToMigrate = [
      "gpt-oss:20b",
      "gpt-oss:120b",
      "deepseek-v3.1:671b",
      "qwen3-coder:480b",
    ];
    const shouldMigrate =
      !cloudDisabled &&
      settings.turboEnabled &&
      baseModelsToMigrate.includes(settings.selectedModel);

    if (shouldMigrate) {
      const cloudModel = `${settings.selectedModel}cloud`;
      return (
        models.find((m) => m.model === cloudModel) ||
        new Model({
          model: cloudModel,
          cloud: true,
          ollama_host: false,
        })
      );
    }

    const found = models.find((m) => m.model === settings.selectedModel);
    // If the stored selection is a known multi-provider router/remote model, the
    // plain chat can't use it (404). Fall back to a chat-compatible model so a
    // leftover "auto/coding" never breaks the composer.
    if (found && !isChatModel(found)) {
      return pickChatDefault(models, recommendedModel, cloudDisabled) || found;
    }
    return (
      found ||
      (settings.selectedModel &&
        new Model({
          model: settings.selectedModel,
          cloud: settings.selectedModel.endsWith("cloud"),
          ollama_host: false,
        })) ||
      null
    );
  }, [
    models,
    settings.selectedModel,
    settings.turboEnabled,
    cloudDisabled,
    recommendedModel,
  ]);

  useEffect(() => {
    if (!selectedModel) return;

    if (
      cloudDisabled &&
      settings.selectedModel?.endsWith("cloud") &&
      selectedModel.model !== settings.selectedModel
    ) {
      setSettings({ SelectedModel: selectedModel.model });
    }

    if (
      !cloudDisabled &&
      settings.turboEnabled &&
      selectedModel.model !== settings.selectedModel
    ) {
      setSettings({ SelectedModel: selectedModel.model, TurboEnabled: false });
    }
  }, [
    selectedModel,
    cloudDisabled,
    settings.selectedModel,
    settings.turboEnabled,
    setSettings,
  ]);

  // Set model from chat history when chat data loads
  useEffect(() => {
    // Only run this effect if we have a valid currentChatId
    if (!currentChatId || currentChatId === "new") {
      return;
    }

    if (
      chatData?.chat?.messages &&
      !isChatLoading &&
      restoredChatRef.current !== currentChatId
    ) {
      // Find the most recent model used in this chat
      const messages = [...chatData.chat.messages].reverse();
      for (const message of messages) {
        if (message.model) {
          const chatModelName = message.model;

          if (chatModelName !== settings.selectedModel) {
            setSettings({ SelectedModel: chatModelName });
          }

          // Mark this chat as restored
          restoredChatRef.current = currentChatId;
          return;
        }
      }
      // Mark this chat as processed even if no model was found
      restoredChatRef.current = currentChatId;
    }
  }, [
    currentChatId,
    chatData,
    isChatLoading,
    settings.selectedModel,
    setSettings,
  ]);

  // Set a chat-compatible default when nothing is selected, or replace a
  // leftover multi-provider router/remote selection (e.g. "auto/coding") that
  // the plain chat cannot use.
  useEffect(() => {
    if (isLoading || models.length === 0) return;

    const current = settings.selectedModel
      ? models.find((m) => m.model === settings.selectedModel)
      : null;

    // A valid chat selection stays as-is.
    if (current && isChatModel(current)) return;
    // An unknown selection (e.g. a cloud name synthesized on the fly that is not
    // in the local list yet) is left alone; only known router/remote is reset.
    if (settings.selectedModel && !current) return;

    const defaultModel = pickChatDefault(models, recommendedModel, cloudDisabled);
    if (defaultModel && defaultModel.model !== settings.selectedModel) {
      setSettings({ SelectedModel: defaultModel.model });
    }
  }, [
    isLoading,
    models,
    settings.selectedModel,
    cloudDisabled,
    recommendedModel,
    setSettings,
  ]);

  // Add the selected model to the models list if it's not already there
  const allModels = useMemo(() => {
    if (!selectedModel || models.find((m) => m.model === selectedModel.model)) {
      return models;
    }

    return [...models, selectedModel];
  }, [models, selectedModel]);

  return {
    selectedModel,
    setSettings,
    models: allModels,
    loading: isLoading,
  };
}
