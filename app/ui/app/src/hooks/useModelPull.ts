import { useRef, useState } from "react";

import { pullModel } from "@/api";
import { useRefetchModels } from "./useModels";

export interface ModelPullProgress {
  completed: number;
  total: number;
}

export interface ModelPull {
  pulling: boolean;
  progress: ModelPullProgress;
  status: string | null;
  error: string | null;
  // Downloads the model and refreshes the local model list. Resolves to true on
  // success, false on cancel/error (the caller then decides what to do, e.g.
  // select the model). Safe to call again after it settles.
  pull: (name: string) => Promise<boolean>;
  cancel: () => void;
  reset: () => void;
}

// useModelPull encapsulates the model download state machine shared by the
// first-run card and the models panel: streaming progress, cancel via
// AbortController, a localized error, and a refetch of the local model list on
// success. It never selects the model itself — that stays a caller concern.
export function useModelPull(errorMessage: string): ModelPull {
  const refetchModels = useRefetchModels();
  const [pulling, setPulling] = useState(false);
  const [progress, setProgress] = useState<ModelPullProgress>({
    completed: 0,
    total: 0,
  });
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const pull = async (name: string): Promise<boolean> => {
    const requested = name.trim();
    if (!requested || pulling) return false;
    const controller = new AbortController();
    abortRef.current = controller;
    setPulling(true);
    setError(null);
    setStatus("Preparando download…");
    setProgress({ completed: 0, total: 0 });
    try {
      for await (const event of pullModel(requested, controller.signal)) {
        setProgress({
          completed: event.completed ?? 0,
          total: event.total ?? 0,
        });
        if (event.status) setStatus(event.status);
      }
      setStatus(`Modelo ${requested} pronto.`);
      await refetchModels();
      return true;
    } catch (err) {
      if (controller.signal.aborted) {
        setStatus(null);
        return false;
      }
      setError(
        err instanceof Error && err.message
          ? `${errorMessage} (${err.message})`
          : errorMessage,
      );
      setStatus(null);
      return false;
    } finally {
      setPulling(false);
      abortRef.current = null;
    }
  };

  const cancel = () => abortRef.current?.abort();

  const reset = () => {
    setStatus(null);
    setError(null);
    setProgress({ completed: 0, total: 0 });
  };

  return { pulling, progress, status, error, pull, cancel, reset };
}
