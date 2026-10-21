import { CodexConnectedIntro } from "./CodexConnectedIntro";
import { confirmDialog } from "@/lib/confirmDialog";
import type { IntegrationStatus } from "@/api";
import { INTEGRATION_ICONS } from "@/lib/launchCommands";
import type {
  CodexDesktopActionResult,
  CodexDesktopStatus,
} from "@/types/webview";
import { CommandLineIcon } from "@heroicons/react/24/outline";
import { IntegrationConnectButton } from "@/components/IntegrationConnectButton";
import {
  useMutation,
  useMutationState,
  useQueryClient,
} from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { useT, type Translator } from "@/i18n";

export const CODEX_DESKTOP_INSTALL_TIMEOUT_MS = 120_000;
const acknowledgmentKey = ["codex-desktop-acknowledgment"];

// Guardamos CHAVES (nao textos): o literal so' e' resolvido no render, para que
// a troca de idioma repinte a faixa sem recarregar o modulo.
const connectionProgress = {
  idle: null,
  installing: {
    label: "codexDesktop.progress.installing.label",
    description: "codexDesktop.progress.installing.description",
  },
  "waiting-for-install": {
    label: "codexDesktop.progress.waitingForInstall.label",
    description: "codexDesktop.progress.waitingForInstall.description",
  },
  connecting: {
    label: "codexDesktop.progress.connecting.label",
    description: "codexDesktop.progress.connecting.description",
  },
  saving: {
    label: "codexDesktop.progress.saving.label",
    description: "codexDesktop.progress.saving.description",
  },
  disconnecting: {
    label: "codexDesktop.progress.disconnecting.label",
    description: "codexDesktop.progress.disconnecting.description",
  },
} as const;

type CodexConnectPhase = keyof typeof connectionProgress;

interface CodexDesktopRowProps {
  integration: IntegrationStatus;
  initialStatus?: CodexDesktopStatus;
}

function CodexIcon({ integration }: { integration: IntegrationStatus }) {
  const icon = INTEGRATION_ICONS[integration.id];
  return (
    <div className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-transparent">
      {icon ? (
        <>
          <img
            src={icon.src}
            alt=""
            className={`${icon.className ?? "h-7 w-7"} rounded-sm object-contain ${icon.darkSrc ? "dark:hidden" : ""}`}
          />
          {icon.darkSrc && (
            <img
              src={icon.darkSrc}
              alt=""
              className={`${icon.className ?? "h-7 w-7"} hidden rounded-sm object-contain dark:block`}
            />
          )}
        </>
      ) : (
        <CommandLineIcon className="h-6 w-6 stroke-[1.5] text-neutral-700 dark:text-neutral-300" />
      )}
    </div>
  );
}

function codexDesktopDescription(
  t: Translator,
  status: CodexDesktopStatus | null,
  defaultDescription: string,
): string {
  if (!status?.connected) return defaultDescription;
  const requestCount = status.requests ?? 0;
  // Plural por duas chaves: `requisição` (1) e `requisições` (resto).
  const key =
    requestCount === 1
      ? "codexDesktop.description.connectedOne"
      : "codexDesktop.description.connectedMany";
  return t(key, { count: requestCount });
}

export function CodexDesktopRow({
  integration,
  initialStatus,
}: CodexDesktopRowProps) {
  const { t } = useT();
  const [status, setStatus] = useState<CodexDesktopStatus | null>(
    initialStatus ?? null,
  );
  const [phase, setPhase] = useState<CodexConnectPhase>("idle");
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [showIntro, setShowIntro] = useState(false);
  const introRestartConfirmed = useRef(false);
  const used = useRef(false);
  const mounted = useRef(true);
  const operationInFlight = useRef(false);
  const statusRequest = useRef(0);
  const queryClient = useQueryClient();
  const acknowledgment = useMutation({
    mutationKey: acknowledgmentKey,
    // Keep the save and its retry available across Apps page navigation.
    gcTime: Infinity,
    retry: false,
    networkMode: "always",
    mutationFn: async () => {
      if (!window.markCodexDesktopIntegrationUsed)
        throw new Error("Acknowledgment is unavailable");
      const saveError = await window.markCodexDesktopIntegrationUsed();
      if (saveError) throw new Error(saveError);
    },
  });
  const acknowledgmentStates = useMutationState({
    filters: { mutationKey: acknowledgmentKey, exact: true },
    select: (mutation) => mutation.state.status,
  });
  const acknowledgmentStatus =
    acknowledgmentStates[acknowledgmentStates.length - 1];
  const savingAcknowledgment = acknowledgmentStates.includes("pending");
  const acknowledgmentFailed =
    acknowledgmentStates.includes("error") &&
    acknowledgmentStatus !== "success" &&
    !status?.used &&
    !used.current;

  const beginOperation = useCallback(
    (nextPhase: CodexConnectPhase) => {
      if (
        !mounted.current ||
        operationInFlight.current ||
        queryClient.isMutating({ mutationKey: acknowledgmentKey })
      )
        return false;
      operationInFlight.current = true;
      ++statusRequest.current;
      setPhase(nextPhase);
      setError(null);
      setNotice(null);
      return true;
    },
    [queryClient],
  );

  const finishOperation = useCallback(
    (nextPhase: CodexConnectPhase = "idle") => {
      operationInFlight.current = false;
      if (mounted.current) setPhase(nextPhase);
    },
    [],
  );

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  useEffect(() => {
    if (acknowledgmentStatus !== "success") return;
    used.current = true;
    setStatus((current) =>
      current && !current.used ? { ...current, used: true } : current,
    );
  }, [acknowledgmentStatus]);

  const refreshStatus = useCallback(async () => {
    if (operationInFlight.current || !window.getCodexDesktopStatus) return;
    const request = ++statusRequest.current;
    const isCurrent = () =>
      mounted.current &&
      request === statusRequest.current &&
      !operationInFlight.current;
    try {
      const next = await window.getCodexDesktopStatus();
      if (!isCurrent()) return;
      setStatus(next);
      if (next.used) {
        used.current = true;
      }
      setError(null);
      setNotice(null);
    } catch {
      if (isCurrent()) setError(t("codexDesktop.error.readStatus"));
    }
  }, [t]);

  useEffect(() => {
    if (!initialStatus) void refreshStatus();
    const onFocus = () => void refreshStatus();
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, [initialStatus, refreshStatus]);

  useEffect(() => {
    if (!status?.connected || !window.getCodexDesktopRequestCount) return;

    let active = true;
    let checking = false;
    const refreshRequestCount = async () => {
      if (!active || checking || document.visibilityState === "hidden") return;
      checking = true;
      try {
        const requests = await window.getCodexDesktopRequestCount?.();
        if (!active || requests === undefined) return;
        setStatus((current) => {
          if (!current || current.requests === requests) return current;
          return { ...current, requests };
        });
      } catch {
        // The next interval or window-focus refresh can recover the count.
      } finally {
        checking = false;
      }
    };

    void refreshRequestCount();
    const interval = window.setInterval(refreshRequestCount, 1000);
    return () => {
      active = false;
      window.clearInterval(interval);
    };
  }, [status?.connected]);

  useEffect(() => {
    if (phase !== "waiting-for-install") return;

    let active = true;
    let checking = false;
    let completing = false;
    const checkForInstall = async () => {
      if (
        !active ||
        checking ||
        completing ||
        operationInFlight.current ||
        !window.getCodexDesktopStatus ||
        !window.setCodexDesktopConnected
      ) {
        return;
      }
      checking = true;
      try {
        const next = await window.getCodexDesktopStatus();
        if (!active || !mounted.current) return;
        setStatus(next);
        if (!next.installed) return;
        if (!beginOperation("connecting")) return;
        completing = true;

        if (next.running) {
          setError(t("codexDesktop.error.restartToConnect"));
          return;
        }

        if (!next.used && !used.current) {
          introRestartConfirmed.current = false;
          setShowIntro(true);
          return;
        }

        const result = await window.setCodexDesktopConnected(true, false);
        if (!mounted.current) return;
        setStatus(result.status);
        if (result.restartConfirmationRequired) {
          setError(t("codexDesktop.error.restartToConnect"));
        } else if (result.error || !result.status.connected) {
          setError(result.error || t("codexDesktop.error.addModels"));
        } else {
          setNotice(t("codexDesktop.notice.modelsAdded"));
        }
      } catch {
        if (!mounted.current || (!active && !completing)) return;
        setPhase("idle");
        setError(t("codexDesktop.error.completeConnection"));
      } finally {
        checking = false;
        if (completing) finishOperation();
      }
    };

    void checkForInstall();
    const interval = window.setInterval(checkForInstall, 1000);
    const timeout = window.setTimeout(() => {
      if (!active || completing) return;
      active = false;
      setPhase("idle");
      setError(t("codexDesktop.error.installNotDetected"));
    }, CODEX_DESKTOP_INSTALL_TIMEOUT_MS);
    return () => {
      active = false;
      window.clearInterval(interval);
      window.clearTimeout(timeout);
    };
  }, [phase, beginOperation, finishOperation, t]);

  const connected = status?.connected ?? false;
  const installed = status?.installed ?? integration.installed ?? false;
  const pending = phase !== "idle" || savingAcknowledgment;
  const displayedConnected =
    phase === "disconnecting"
      ? false
      : connected ||
        showIntro ||
        phase === "installing" ||
        phase === "waiting-for-install" ||
        phase === "connecting";
  const progress = connectionProgress[savingAcknowledgment ? "saving" : phase];
  const statusLabel = progress ? t(progress.label) : null;
  const actionError =
    error ??
    (acknowledgmentFailed ? t("codexDesktop.error.saveProgress") : null);
  const description =
    actionError ??
    notice ??
    (progress ? t(progress.description) : null) ??
    codexDesktopDescription(
      t,
      status,
      installed
        ? t("codexDesktop.description.useCodexModels")
        : t("codexDesktop.description.downloadIntro"),
    );

  const saveAcknowledgment = async (): Promise<boolean> => {
    if (queryClient.isMutating({ mutationKey: acknowledgmentKey }))
      return false;
    try {
      await acknowledgment.mutateAsync();
      used.current = true;
      if (mounted.current) {
        setStatus((current) =>
          current ? { ...current, used: true } : current,
        );
      }
      return true;
    } catch {
      return false;
    }
  };

  const retryAcknowledgment = async () => {
    if (pending || !acknowledgmentFailed || !beginOperation("saving")) return;
    try {
      await saveAcknowledgment();
    } finally {
      finishOperation();
      if (mounted.current) void refreshStatus();
    }
  };

  const toggleConnection = async (fromIntro = false) => {
    const enabled = fromIntro || !connected;
    const nextPhase = enabled
      ? installed
        ? "connecting"
        : "installing"
      : "disconnecting";
    if (pending || (showIntro && !fromIntro) || !beginOperation(nextPhase))
      return;
    let finalPhase: CodexConnectPhase = "idle";
    let restartConfirmed = fromIntro && introRestartConfirmed.current;
    if (fromIntro) {
      setShowIntro(false);
      introRestartConfirmed.current = false;
    }
    try {
      if (!window.setCodexDesktopConnected) {
        setError(t("codexDesktop.error.integrationUnavailable"));
        return;
      }
      if (enabled && !installed) {
        if (!window.installCodexDesktop || !window.getCodexDesktopStatus) {
          setError(t("codexDesktop.error.installFailed"));
          return;
        }
        const installResult = await window.installCodexDesktop();
        if (!mounted.current) return;
        if (installResult === "opened") finalPhase = "waiting-for-install";
        else if (installResult !== "cancelled")
          setError(t("codexDesktop.error.installFailed"));
        return;
      }
      if (fromIntro || (enabled && !status?.used && !used.current)) {
        if (!window.getCodexDesktopStatus) {
          setError(t("codexDesktop.error.readStatus"));
          return;
        }
        const liveStatus = await window.getCodexDesktopStatus();
        if (!mounted.current) return;
        setStatus(liveStatus);
        if (liveStatus.running && !restartConfirmed) {
          restartConfirmed = await confirmDialog(
            t("codexDesktop.confirm.restartToAdd"),
            { confirmLabel: t("codexDesktop.confirm.restartLabel") },
          );
          if (!restartConfirmed) return;
        }

        if (!fromIntro && !liveStatus.used && !used.current) {
          introRestartConfirmed.current = restartConfirmed;
          setShowIntro(true);
          return;
        }
      }

      let result: CodexDesktopActionResult =
        await window.setCodexDesktopConnected(enabled, restartConfirmed);

      if (!mounted.current) return;
      setStatus(result.status);
      if (result.restartConfirmationRequired) {
        // Keep focus-driven status refreshes from discarding this operation
        // while the native confirmation dialog temporarily owns focus.
        if (
          !(await confirmDialog(
            enabled
              ? t("codexDesktop.confirm.restartToAdd")
              : t("codexDesktop.confirm.restartToRemove"),
            { confirmLabel: t("codexDesktop.confirm.restartLabel") },
          )) ||
          !mounted.current
        ) {
          return;
        }
        result = await window.setCodexDesktopConnected(enabled, true);
        if (!mounted.current) return;
        setStatus(result.status);
      }

      if (result.restartConfirmationRequired) return;
      if (result.error) {
        setError(result.error);
        return;
      }
      if (result.status.connected !== enabled) {
        setError(
          enabled
            ? t("codexDesktop.error.addModels")
            : t("codexDesktop.error.removeModels"),
        );
        return;
      }
      if (fromIntro) {
        setPhase("saving");
        if (!(await saveAcknowledgment())) return;
      }
      if (enabled) {
        setNotice(t("codexDesktop.notice.modelsAdded"));
      } else {
        setNotice(t("codexDesktop.notice.modelsRemoved"));
      }
    } catch {
      setError(
        nextPhase === "installing"
          ? t("codexDesktop.error.installFailed")
          : enabled
            ? t("codexDesktop.error.addModels")
            : t("codexDesktop.error.removeModels"),
      );
    } finally {
      finishOperation(finalPhase);
    }
  };

  return (
    <div
      id="integration-chatgpt"
      className="flex items-center gap-4 rounded-2xl border border-neutral-200 bg-neutral-50 px-5 py-3 dark:border-neutral-700 dark:bg-neutral-800/50"
    >
      <div className="flex min-w-0 flex-1 items-center gap-4">
        <CodexIcon integration={integration} />
        <div className="min-w-0">
          <p className="text-base font-medium text-neutral-950 dark:text-neutral-100">
            ChatGPT
          </p>
          <p
            role={actionError ? "alert" : notice ? "status" : undefined}
            className="mt-1 text-[13px] leading-5 text-neutral-500 dark:text-neutral-400"
          >
            {description}
          </p>
        </div>
      </div>
      <div className="ml-auto flex shrink-0 items-center gap-2.5">
        {acknowledgmentFailed && (
          <button
            type="button"
            aria-label={t("codexDesktop.action.retryAria")}
            disabled={pending}
            onClick={() => void retryAcknowledgment()}
            className="text-xs font-medium text-neutral-700 hover:underline disabled:cursor-wait disabled:opacity-50 dark:text-neutral-300"
          >
            {t("codexDesktop.action.retry")}
          </button>
        )}
        <IntegrationConnectButton
          connected={displayedConnected}
          busy={pending}
          progress={statusLabel}
          label={
            showIntro
              ? t("codexDesktop.action.finishConnection")
              : connected
                ? t("codexDesktop.action.removeModels")
                : pending
                  ? t("codexDesktop.action.connecting")
                  : t("codexDesktop.action.addModels")
          }
          title={
            connected
              ? t("codexDesktop.action.removeModelsTitle")
              : installed
                ? t("codexDesktop.action.addModelsTitle")
                : t("codexDesktop.action.installTitle")
          }
          disabled={pending || showIntro}
          onClick={() => void toggleConnection()}
        />
      </div>
      {showIntro && (
        <CodexConnectedIntro onDone={() => void toggleConnection(true)} />
      )}
    </div>
  );
}
