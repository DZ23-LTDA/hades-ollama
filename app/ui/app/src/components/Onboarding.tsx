import { CodexDesktopRow } from "@/components/CodexDesktopRow";
import { confirmDialog } from "@/lib/confirmDialog";
import Logo from "@/components/Logo";
import type { OnboardingStep } from "@/lib/onboarding";
import { IntegrationConnectButton } from "@/components/IntegrationConnectButton";
import { Transition } from "@headlessui/react";
import {
  getIntegrationStatuses,
  pullModel,
  type IntegrationStatus,
  type IntegrationStatuses,
} from "@/api";
import { INTEGRATION_ICONS } from "@/lib/launchCommands";
import {
  CLAUDE_CONNECTION_TIMEOUT_MS,
  ClaudeConnectionTimeoutError,
  claudeDesktopRecoveryMessage,
  claudeDesktopRequestCountLabel,
  isClaudeConfigured,
  isClaudeConnectionComplete,
  optimisticClaudeConnectionState,
  scheduleClaudeInstallTimeout,
  withClaudeConnectionTimeout,
} from "@/lib/claudeDesktop";
import { isWindowsPlatform } from "@/lib/platform";
import { enableCloudModelsOptIn } from "@/lib/cloudSettings";
import type {
  ClaudeDesktopActionResult,
  ClaudeDesktopStatus,
  CodexDesktopStatus,
} from "@/types/webview";
import { shouldShowClaudeConnectedIntro } from "./onboardingUtils";
import { RECOMMENDED_FIRST_MODEL } from "@/lib/firstModel";
import { copyTextToClipboard } from "@/utils/clipboard";
import {
  ArrowsRightLeftIcon,
  CommandLineIcon,
  ShieldCheckIcon,
} from "@heroicons/react/24/outline";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useT } from "@/i18n";

type ClaudeConnectPhase =
  | "idle"
  | "installing"
  | "waiting-for-install"
  | "connecting"
  | "launching"
  | "disconnecting";

function setClaudeConnection(
  enabled: boolean,
  deferLaunch = false,
  restartConfirmed = false,
) {
  if (enabled && deferLaunch && window.prepareClaudeDesktopConnection) {
    return window.prepareClaudeDesktopConnection();
  }
  if (!window.setClaudeDesktopConnected) {
    throw new Error("A conexão com o Claude Desktop está indisponível");
  }
  return window.setClaudeDesktopConnected(enabled, restartConfirmed);
}

function getClaudeConnectionSummary() {
  const getStatus =
    window.getClaudeDesktopConnectionSummary ?? window.getClaudeDesktopStatus;
  return getStatus?.() ?? Promise.resolve(null);
}

interface ScreenProps {
  isSigningIn: boolean;
  signInError: string | null;
  onSignIn: () => void;
}

interface WelcomeScreenProps extends ScreenProps {
  isAuthenticated: boolean;
  isLeaving?: boolean;
  completionError?: string | null;
  onRetryCompletion?: () => void;
  onLocal: () => void;
  onSignUp: () => void;
  cloudOptIn?: boolean;
  onCloudOptInChange?: (enabled: boolean) => void;
}

interface RunOllamaScreenProps {
  completionError: string | null;
  onRetryCompletion: () => void;
  onContinue: () => void;
}

interface ConnectAppsScreenProps {
  initialIntegrations?: IntegrationStatuses;
  initialClaudeStatus?: ClaudeDesktopStatus;
  initialCodexStatus?: CodexDesktopStatus;
}

function TitleBar({ onSignIn }: { onSignIn?: () => void }) {
  const { t } = useT();
  const isMacOS =
    typeof navigator !== "undefined" &&
    navigator.platform.toLowerCase().includes("mac");

  return (
    <header
      className={`relative flex shrink-0 items-center justify-center bg-white ${isMacOS ? "h-[52px]" : "h-10"}`}
      onDoubleClick={() => window.doubleClick?.()}
      onMouseDown={() => window.drag?.()}
    >
      {onSignIn && (
        <button
          type="button"
          className="absolute inset-y-0 right-5 flex cursor-pointer items-center rounded-md px-2 text-sm font-normal leading-none text-neutral-500 hover:text-neutral-900 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
          onClick={onSignIn}
          onMouseDown={(event) => event.stopPropagation()}
        >
          {t("onboarding.signIn")}
        </button>
      )}
    </header>
  );
}

function OnboardingIcon({ compact = false }: { compact?: boolean }) {
  return (
    <div className="flex items-center justify-center">
      <Logo
        size={compact ? 42 : 54}
        containerClassName="mb-0"
        showBackground={false}
      />
    </div>
  );
}

function OnboardingCard({ children }: { children: ReactNode }) {
  return (
    <section className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto bg-white px-6 pb-10 pt-0">
      <div className="flex w-full max-w-[760px] flex-col items-center justify-center bg-white px-10 py-6 text-center">
        {children}
      </div>
    </section>
  );
}

const OLLAMA_FEATURES = [
  {
    titleKey: "onboarding.features.connectApps.title",
    descriptionKey: "onboarding.features.connectApps.description",
    icon: CommandLineIcon,
  },
  {
    titleKey: "onboarding.features.switchModel.title",
    descriptionKey: "onboarding.features.switchModel.description",
    icon: ArrowsRightLeftIcon,
  },
  {
    titleKey: "onboarding.features.yourData.title",
    descriptionKey: "onboarding.features.yourData.description",
    icon: ShieldCheckIcon,
  },
];

export function IntroScreen({
  completionError = null,
  onContinue,
  onRetryCompletion,
  isLeaving = false,
}: {
  completionError?: string | null;
  isLeaving?: boolean;
  onContinue: () => void;
  onRetryCompletion?: () => void;
}) {
  const { t } = useT();
  return (
    <main className="light-only flex h-screen w-full flex-col overflow-hidden bg-white text-neutral-950">
      <TitleBar />

      <section className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto px-6 pb-10">
        <div className="mx-auto flex min-h-full w-full max-w-[620px] flex-col items-center justify-center py-4 text-center">
          <div className="flex flex-col items-center justify-center gap-2">
            <img
              src="/hello.png"
              alt={t("onboarding.intro.alt")}
              className="h-[72px] w-[72px] select-none object-contain"
              draggable={false}
            />
            <h1 className="font-rounded text-2xl font-medium leading-8">
              {t("onboarding.intro.title")}
            </h1>
          </div>
          <p className="mt-4 max-w-[380px] text-sm leading-6 text-neutral-500">
            {t("onboarding.intro.tagline")}
          </p>
          <div className="mx-auto mt-8 flex w-fit max-w-full flex-col gap-6 text-left">
            {OLLAMA_FEATURES.map((feature) => {
              const Icon = feature.icon;

              return (
                <div key={feature.titleKey} className="flex items-start gap-4">
                  <Icon className="mt-0.5 h-7 w-7 shrink-0 stroke-[1.5] text-neutral-700" />
                  <div>
                    <h2 className="text-sm font-medium text-neutral-950">
                      {t(feature.titleKey)}
                    </h2>
                    <p className="mt-0.5 text-[13px] leading-5 text-neutral-500">
                      {t(feature.descriptionKey)}
                    </p>
                  </div>
                </div>
              );
            })}
          </div>

          <button
            type="button"
            className="mt-8 flex h-11 w-full max-w-[240px] cursor-pointer items-center justify-center rounded-full bg-neutral-900 px-5 font-sans text-sm font-normal text-white transition-colors hover:bg-neutral-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
            onClick={onContinue}
            disabled={isLeaving}
            aria-busy={isLeaving || undefined}
          >
            {t("onboarding.continue")}
          </button>
          <InlineError message={completionError} />
          {completionError && onRetryCompletion && (
            <button
              type="button"
              className="mt-2 cursor-pointer rounded-md px-3 py-1 text-sm font-normal text-neutral-600 underline decoration-neutral-300 underline-offset-4 hover:text-neutral-950 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
              onClick={onRetryCompletion}
            >
              {t("onboarding.retry")}
            </button>
          )}
        </div>
      </section>
    </main>
  );
}

function InlineError({
  message,
  className = "mt-3 text-sm",
}: {
  message: string | null;
  className?: string;
}) {
  if (!message) return null;

  return (
    <p role="alert" className={`${className} text-red-600`}>
      {message}
    </p>
  );
}

export function WelcomeScreen({
  isAuthenticated,
  isSigningIn,
  isLeaving = false,
  signInError,
  completionError = null,
  onSignIn,
  onSignUp,
  onLocal,
  onRetryCompletion,
  cloudOptIn,
  onCloudOptInChange,
}: WelcomeScreenProps) {
  const { t } = useT();
  return (
    <main className="light-only flex min-h-screen w-full flex-col bg-white text-neutral-950">
      <TitleBar onSignIn={isAuthenticated ? undefined : onSignIn} />
      <OnboardingCard>
        <OnboardingIcon />
        <h1 className="mt-7 font-rounded text-2xl font-medium leading-8">
          {t("onboarding.welcome.title")}
        </h1>
        <p className="mt-3 max-w-[400px] text-sm leading-6 text-neutral-400">
          {t("onboarding.welcome.subtitle")}
        </p>
        <p className="mt-1 max-w-[400px] text-sm leading-6 text-neutral-400">
          {t("onboarding.welcome.privacy")}
        </p>

        <div className="mt-7 flex w-full max-w-[240px] flex-col items-center">
          <button
            type="button"
            className="flex h-11 w-full cursor-pointer items-center justify-center rounded-full bg-neutral-900 px-5 font-sans text-sm font-normal text-white transition-colors hover:bg-neutral-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500 disabled:cursor-wait disabled:opacity-70"
            onClick={onSignUp}
            disabled={isSigningIn || isLeaving}
            aria-busy={isSigningIn || isLeaving}
          >
            {isLeaving
              ? t("onboarding.welcome.openingApps")
              : isSigningIn
                ? t("onboarding.welcome.finishInBrowser")
                : t("onboarding.welcome.createAccount")}
          </button>
          <label className="mt-4 flex items-center gap-2 text-xs text-neutral-600 select-none cursor-pointer">
            <input
              type="checkbox"
              checked={cloudOptIn ?? false}
              onChange={(e) => onCloudOptInChange?.(e.target.checked)}
              className="rounded border-neutral-300 text-neutral-900 focus:ring-neutral-500 cursor-pointer"
            />
            <span>{t("onboarding.welcome.cloudOptIn")}</span>
          </label>
          <button
            type="button"
            className="mt-2 cursor-pointer rounded-md px-3 py-2 text-sm font-normal text-neutral-600 underline decoration-neutral-300 underline-offset-4 hover:text-neutral-950 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
            onClick={onLocal}
            disabled={isLeaving}
          >
            {t("onboarding.welcome.useLocal")}
          </button>
          <InlineError message={signInError ?? completionError} />
          {completionError && onRetryCompletion && (
            <button
              type="button"
              className="mt-2 cursor-pointer rounded-md px-3 py-1 text-sm font-normal text-neutral-600 underline decoration-neutral-300 underline-offset-4 hover:text-neutral-950 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
              onClick={onRetryCompletion}
            >
              {t("onboarding.retry")}
            </button>
          )}
        </div>
      </OnboardingCard>
    </main>
  );
}

export function RunOllamaScreen({
  completionError,
  onRetryCompletion,
  onContinue,
}: RunOllamaScreenProps) {
  const { t } = useT();
  const [modelName, setModelName] = useState(RECOMMENDED_FIRST_MODEL);
  const [pulling, setPulling] = useState(false);
  const [modelReady, setModelReady] = useState(false);
  const [pullStatus, setPullStatus] = useState<string | null>(null);
  const [pullError, setPullError] = useState<string | null>(null);

  const downloadModel = async () => {
    const requestedModel = modelName.trim();
    if (!requestedModel || pulling) return;
    setPulling(true);
    setPullError(null);
    setPullStatus(t("onboarding.run.preparing"));
    try {
      for await (const event of pullModel(requestedModel)) {
        const completed = event.completed ?? 0;
        const total = event.total ?? 0;
        setPullStatus(
          total > 0
            ? `${event.status} (${Math.round((completed / total) * 100)}%)`
            : event.status,
        );
      }
      setPullStatus(t("onboarding.run.modelReady", { model: requestedModel }));
      setModelReady(true);
    } catch (error) {
      setPullError(
        error instanceof Error
          ? error.message
          : t("onboarding.run.downloadFailed"),
      );
      setPullStatus(null);
    } finally {
      setPulling(false);
    }
  };

  return (
    <main className="light-only flex min-h-screen w-full flex-col bg-white text-neutral-950">
      <TitleBar />
      <OnboardingCard>
        <OnboardingIcon compact />
        <h1 className="mt-6 font-rounded text-[22px] font-medium leading-7">
          {t("onboarding.run.title")}
        </h1>

        <p className="mt-3 max-w-xs text-[13px] leading-5 text-neutral-400">
          {t("onboarding.run.description")}
        </p>

        <div className="mt-5 flex w-full max-w-[360px] flex-col gap-2 text-left">
          <label htmlFor="onboarding-model" className="text-xs font-medium text-neutral-700">
            {t("onboarding.run.modelLabel")}
          </label>
          <div className="flex gap-2">
            <input
              id="onboarding-model"
              value={modelName}
              onChange={(event) => setModelName(event.target.value)}
              disabled={pulling}
              className="min-w-0 flex-1 rounded-xl border border-neutral-200 px-3 py-2 text-sm outline-none focus:border-neutral-500 focus:ring-2 focus:ring-neutral-200"
              placeholder={t("onboarding.run.modelPlaceholder", { model: RECOMMENDED_FIRST_MODEL })}
            />
            <button
              type="button"
              onClick={() => void downloadModel()}
              disabled={pulling || !modelName.trim()}
              className="rounded-xl bg-neutral-900 px-3 py-2 text-xs font-medium text-white disabled:cursor-wait disabled:opacity-60"
              aria-busy={pulling}
            >
              {pulling ? t("onboarding.run.downloading") : t("onboarding.run.download")}
            </button>
          </div>
          <p className="text-[11px] text-neutral-400">
            {t("onboarding.run.fileNote")}
          </p>
        </div>

        {pullStatus && (
          <p className="mt-3 text-xs text-emerald-700" role="status" aria-live="polite">
            {pullStatus}
          </p>
        )}
        <InlineError message={pullError} />

        <InlineError message={completionError} />
        {modelReady && (
          <button
            type="button"
            onClick={onContinue}
            className="mt-5 h-11 w-full max-w-[240px] rounded-full bg-neutral-900 px-5 text-sm font-normal text-white transition-colors hover:bg-neutral-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
          >
            {t("onboarding.continue")}
          </button>
        )}
        {completionError && (
          <button
            type="button"
            className="mt-2 cursor-pointer rounded-md px-3 py-1 text-sm font-normal text-neutral-600 underline decoration-neutral-300 underline-offset-4 hover:text-neutral-950 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
            onClick={onRetryCompletion}
          >
            {t("onboarding.retry")}
          </button>
        )}
      </OnboardingCard>
    </main>
  );
}

function LaunchCommandIcon({ id }: { id: string }) {
  const icon = INTEGRATION_ICONS[id];

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

export function ClaudeConnectedIntro({ onDone }: { onDone: () => void }) {
  const { t } = useT();
  return (
    <div className="claude-connected-backdrop fixed inset-0 z-50 flex items-center justify-center bg-black/20 p-6 dark:bg-black/50">
      <section
        role="dialog"
        aria-modal="true"
        aria-labelledby="claude-connected-title"
        aria-describedby="claude-connected-description"
        className="claude-connected-dialog relative w-full max-w-md overflow-hidden rounded-2xl bg-white font-sans shadow-2xl ring-1 ring-black/10 dark:bg-neutral-800 dark:ring-white/10"
      >
        <img
          src="/claude-connected.png"
          alt={t("onboarding.claudeIntro.alt")}
          width={896}
          height={768}
          className="h-auto w-full object-contain"
          draggable={false}
        />
        <div className="p-6 pt-8">
          <h2
            id="claude-connected-title"
            className="font-rounded text-lg font-medium leading-6 text-neutral-950 dark:text-neutral-100"
          >
            {t("onboarding.claudeIntro.title")}
          </h2>
          <p
            id="claude-connected-description"
            className="mt-2 text-[13px] leading-5 text-neutral-500 dark:text-neutral-400"
          >
            {t("onboarding.claudeIntro.description")}
          </p>
          <div className="mt-11 flex justify-end">
            <button
              type="button"
              autoFocus
              className="rounded-full bg-neutral-100 px-6 py-2 text-sm font-normal text-neutral-950 transition-colors hover:bg-neutral-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500 dark:bg-white dark:hover:bg-neutral-100"
              onClick={onDone}
            >
              {t("onboarding.continue")}
            </button>
          </div>
        </div>
      </section>
    </div>
  );
}

export function ConnectAppsScreen({
  initialIntegrations,
  initialClaudeStatus,
  initialCodexStatus,
}: ConnectAppsScreenProps) {
  const { t } = useT();
  const isWindows = isWindowsPlatform();
  const safeInitialIntegrations = Array.isArray(initialIntegrations)
    ? initialIntegrations
    : undefined;
  const [copyNotice, setCopyNotice] = useState<{
    sequence: number;
    id: string;
    name: string;
    command: string;
    copied: boolean;
    visible: boolean;
  } | null>(null);
  const copyInFlight = useRef(false);
  const copySequence = useRef(0);
  const copyNoticeRef = useRef<HTMLDivElement>(null);
  const [initialClaudeStatusSettled, setInitialClaudeStatusSettled] = useState(
    Boolean(initialClaudeStatus),
  );
  const [claudeError, setClaudeError] = useState<string | null>(null);
  const [claudeStatus, setClaudeStatus] = useState<ClaudeDesktopStatus | null>(
    initialClaudeStatus ?? null,
  );
  const [claudePhase, setClaudePhase] = useState<ClaudeConnectPhase>("idle");
  const [showClaudeConnectedIntro, setShowClaudeConnectedIntro] =
    useState(false);
  const claudeConnectedIntroPending = useRef(false);
  const claudeRestartConfirmed = useRef(false);
  const screenMounted = useRef(true);
  const [integrationStatuses, setIntegrationStatuses] =
    useState<IntegrationStatuses | null>(safeInitialIntegrations ?? null);
  const [statusError, setStatusError] = useState(false);

  useEffect(() => {
    screenMounted.current = true;
    return () => {
      screenMounted.current = false;
    };
  }, []);

  useEffect(() => {
    if (!copyNotice?.copied) return;
    const timeout = window.setTimeout(() => {
      setCopyNotice((current) => current && { ...current, visible: false });
    }, 6000);
    return () => window.clearTimeout(timeout);
  }, [copyNotice?.sequence, copyNotice?.copied]);

  useEffect(() => {
    if (!copyNotice?.visible || copyNotice.copied) return;

    const dismiss = () => {
      setCopyNotice((current) =>
        current && !current.copied ? { ...current, visible: false } : current,
      );
    };
    const onPointerDown = (event: PointerEvent) => {
      if (!copyNoticeRef.current?.contains(event.target as Node)) dismiss();
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !event.defaultPrevented) dismiss();
    };
    window.addEventListener("pointerdown", onPointerDown);
    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("pointerdown", onPointerDown);
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [copyNotice?.copied, copyNotice?.visible]);

  const refreshClaudeStatus = useCallback(async () => {
    if (isWindows) return null;
    if (
      !window.getClaudeDesktopConnectionSummary &&
      !window.getClaudeDesktopStatus
    ) {
      return null;
    }
    try {
      const status = await getClaudeConnectionSummary();
      if (!status || !screenMounted.current) return null;
      setClaudeStatus(status);
      setClaudeError(null);
      return status;
    } catch {
      if (screenMounted.current) {
        setClaudeError(t("onboarding.claude.readStatusFailed"));
      }
      return null;
    }
  }, [isWindows]);

  useEffect(() => {
    let active = true;
    const integrations = safeInitialIntegrations
      ? Promise.resolve(safeInitialIntegrations)
      : getIntegrationStatuses();

    void integrations.then(
      (statuses) => {
        if (!active) return;
        setIntegrationStatuses(statuses);
      },
      () => {
        if (!active) return;
        setStatusError(true);
      },
    );

    return () => {
      active = false;
    };
  }, [safeInitialIntegrations]);

  useEffect(() => {
    let active = true;
    const claude = initialClaudeStatus
      ? Promise.resolve(initialClaudeStatus)
      : getClaudeConnectionSummary();

    void claude.then(
      (status) => {
        if (!active) return;
        setClaudeStatus(status);
        setInitialClaudeStatusSettled(true);
      },
      () => {
        if (!active) return;
        setClaudeError(t("onboarding.claude.readStatusFailed"));
        setInitialClaudeStatusSettled(true);
      },
    );

    return () => {
      active = false;
    };
  }, [initialClaudeStatus]);

  const openConnectedClaude = useCallback(
    async (status: ClaudeDesktopStatus) => {
      if (!status.connected || status.running || !window.openClaudeDesktop) {
        return null;
      }
      try {
        return (await window.openClaudeDesktop()) || null;
      } catch {
        return t("onboarding.claude.openFailed");
      }
    },
    [],
  );

  const finishClaudeConnection = useCallback(
    async (status: ClaudeDesktopStatus) => {
      if (claudeConnectedIntroPending.current) return null;
      if (shouldShowClaudeConnectedIntro(status)) {
        claudeConnectedIntroPending.current = true;
        setShowClaudeConnectedIntro(true);
        window.activateOllama?.();
        return null;
      }

      return openConnectedClaude(status);
    },
    [openConnectedClaude],
  );

  const reconcileLateClaudeAction = useCallback(
    (enabled: boolean) =>
      (settled: PromiseSettledResult<ClaudeDesktopActionResult>) => {
        if (!screenMounted.current) return;

        if (settled.status === "fulfilled") {
          const result = settled.value;
          setClaudeStatus(result.status);
          setClaudeError(result.error || null);
          if (result.error || !enabled) return;

          void finishClaudeConnection(result.status).then(
            (completionError) => {
              if (completionError && screenMounted.current) {
                setClaudeError(completionError);
              }
            },
            () => {
              if (screenMounted.current) {
                setClaudeError(
                  t("onboarding.claude.openFailed"),
                );
              }
            },
          );
          return;
        }

        void refreshClaudeStatus().then(() => {
          if (screenMounted.current) {
            setClaudeError(
              enabled
                ? t("onboarding.claude.connectFailed")
                : t("onboarding.claude.disconnectFailed"),
            );
          }
        });
      },
    [finishClaudeConnection, refreshClaudeStatus],
  );

  const dismissClaudeConnectedIntro = async () => {
    if (!window.setClaudeDesktopConnected || claudePhase !== "idle") return;
    setClaudePhase("launching");
    try {
      const liveStatus = await withClaudeConnectionTimeout(
        getClaudeConnectionSummary(),
      );
      if (!screenMounted.current) return;
      if (!liveStatus) {
        throw new Error("O status da conexão com o Claude Desktop está indisponível");
      }
      setClaudeStatus(liveStatus);

      let restartConfirmed = claudeRestartConfirmed.current;
      if (liveStatus.running && !restartConfirmed) {
        restartConfirmed = await confirmDialog(
          t("onboarding.claude.restartToUse"),
          { confirmLabel: t("onboarding.claude.restartLabel") },
        );
        if (!screenMounted.current) return;
        if (!restartConfirmed) {
          setClaudePhase("idle");
          return;
        }
      }

      claudeConnectedIntroPending.current = false;
      claudeRestartConfirmed.current = false;
      setShowClaudeConnectedIntro(false);
      const result = await withClaudeConnectionTimeout(
        window.setClaudeDesktopConnected(true, restartConfirmed),
        CLAUDE_CONNECTION_TIMEOUT_MS,
        reconcileLateClaudeAction(true),
      );
      setClaudeStatus(result.status);
      setClaudeError(result.error || null);
    } catch (error) {
      setClaudeError(
        error instanceof ClaudeConnectionTimeoutError
          ? t("onboarding.claude.timeoutOpen")
          : t("onboarding.claude.openFailed"),
      );
    } finally {
      setClaudePhase("idle");
    }
  };

  useEffect(() => {
    const handleFocus = () => void refreshClaudeStatus();
    window.addEventListener("focus", handleFocus);
    return () => window.removeEventListener("focus", handleFocus);
  }, [refreshClaudeStatus]);

  useEffect(() => {
    if (!claudeStatus?.connected || !window.getClaudeDesktopRequestCount) {
      return;
    }

    let active = true;
    let checking = false;
    const refreshRequestCount = async () => {
      if (!active || checking || document.visibilityState === "hidden") return;
      checking = true;
      try {
        const routedRequests = await window.getClaudeDesktopRequestCount?.();
        if (!active || routedRequests === undefined) return;
        setClaudeStatus((current) => {
          if (!current || current.routedRequests === routedRequests) {
            return current;
          }
          return { ...current, routedRequests };
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
  }, [claudeStatus?.connected]);

  useEffect(() => {
    if (claudePhase !== "waiting-for-install") return;

    let active = true;
    let completing = false;
    let checking = false;
    const checkForInstall = async () => {
      if (checking) return;
      if (
        (!window.getClaudeDesktopConnectionSummary &&
          !window.getClaudeDesktopStatus) ||
        !window.setClaudeDesktopConnected
      ) {
        return;
      }
      checking = true;
      try {
        const status = await withClaudeConnectionTimeout(
          getClaudeConnectionSummary(),
        );
        if (!status) return;
        if (!active) return;
        setClaudeStatus(status);
        if (!status.installed || completing) return;
        completing = true;

        if (status.running) {
          setClaudePhase("idle");
          setClaudeError(
            t("onboarding.claude.installedRestart"),
          );
          return;
        }

        setClaudePhase("connecting");
        claudeRestartConfirmed.current = false;
        const result = await withClaudeConnectionTimeout(
          setClaudeConnection(true, !status.used, false),
          CLAUDE_CONNECTION_TIMEOUT_MS,
          reconcileLateClaudeAction(true),
        );
        if (!screenMounted.current) return;
        setClaudeStatus(result.status);
        let actionError = result.error || null;
        if (!actionError && result.status.connected) {
          actionError = await finishClaudeConnection(result.status);
        } else if (!actionError) {
          actionError = t("onboarding.claude.connectFailed");
        }
        setClaudeError(actionError);
        setClaudePhase("idle");
      } catch (error) {
        if (!screenMounted.current) return;
        setClaudePhase("idle");
        setClaudeError(
          error instanceof ClaudeConnectionTimeoutError
            ? t("onboarding.claude.timeoutConnect")
            : t("onboarding.claude.completeFailed"),
        );
      } finally {
        checking = false;
      }
    };

    void checkForInstall();
    const interval = window.setInterval(checkForInstall, 1000);
    const timeout = scheduleClaudeInstallTimeout(() => {
      if (!active) return;
      setClaudePhase("idle");
      setClaudeError(t("onboarding.claude.installNotDetected"));
    });
    return () => {
      active = false;
      window.clearInterval(interval);
      window.clearTimeout(timeout);
    };
  }, [claudePhase, finishClaudeConnection, reconcileLateClaudeAction]);

  const copyLaunchCommand = async (item: IntegrationStatus) => {
    if (!item.command || copyInFlight.current) return;
    copyInFlight.current = true;
    let copied = false;
    try {
      copied = await copyTextToClipboard(item.command);
    } catch {
      // Keep the command available for manual copying when clipboard access fails.
    } finally {
      copyInFlight.current = false;
    }
    if (!screenMounted.current) return;
    setCopyNotice({
      sequence: ++copySequence.current,
      id: item.id,
      name: item.name,
      command: item.command,
      copied,
      visible: true,
    });
  };

  const connectClaude = async () => {
    if (claudePhase !== "idle") return;
    if (
      (!window.getClaudeDesktopConnectionSummary &&
        !window.getClaudeDesktopStatus) ||
      !window.setClaudeDesktopConnected
    ) {
      setClaudeError(t("onboarding.claude.macosOnly"));
      return;
    }

    setCopyNotice((current) => current && { ...current, visible: false });
    setClaudeError(null);
    const enabling = claudeStatus ? !isClaudeConfigured(claudeStatus) : true;
    setClaudePhase(enabling ? "connecting" : "disconnecting");

    let status: ClaudeDesktopStatus | null;
    try {
      status = await withClaudeConnectionTimeout(getClaudeConnectionSummary());
    } catch (error) {
      setClaudePhase("idle");
      setClaudeError(
        error instanceof ClaudeConnectionTimeoutError
          ? t("onboarding.claude.timeoutAction", { action: enabling ? t("onboarding.claude.actionConnect") : t("onboarding.claude.actionDisconnect") })
          : t("onboarding.claude.readStatusFailed"),
      );
      return;
    }
    if (!screenMounted.current) return;
    if (!status) {
      setClaudePhase("idle");
      setClaudeError(t("onboarding.claude.readStatusFailed"));
      return;
    }
    setClaudeStatus(status);

    // The menu-bar control may have reached this target since the app last
    // refreshed. Sync the row without repeating the profile change or
    // restarting Claude, while preserving first-use completion behavior.
    if (isClaudeConnectionComplete(enabling, status)) {
      claudeRestartConfirmed.current = false;
      const completionError = enabling
        ? await finishClaudeConnection(status)
        : null;
      setClaudeError(completionError);
      setClaudePhase("idle");
      return;
    }

    if (enabling && !status.installed) {
      if (!window.installClaudeDesktop) {
        setClaudePhase("idle");
        setClaudeError(t("onboarding.claude.installerFailed"));
        return;
      }
      setClaudePhase("installing");
      let installResult: "opened" | "cancelled" | "failed" = "failed";
      try {
        installResult = await window.installClaudeDesktop();
      } catch {
        // The error below is shared with a rejected native install request.
      }
      if (installResult === "cancelled") {
        setClaudePhase("idle");
        return;
      }
      if (installResult !== "opened") {
        setClaudePhase("idle");
        setClaudeError(t("onboarding.claude.installerFailed"));
        return;
      }
      setClaudePhase("waiting-for-install");
      return;
    }

    let restartConfirmed = false;
    if (status.running) {
      restartConfirmed = await confirmDialog(
        enabling
          ? t("onboarding.claude.restartToUse")
          : t("onboarding.claude.restartToRemove"),
        { confirmLabel: t("onboarding.claude.restartLabel") },
      );
      if (!screenMounted.current) return;
      if (!restartConfirmed) {
        setClaudePhase("idle");
        return;
      }
    }

    claudeRestartConfirmed.current = restartConfirmed;
    try {
      const result = await withClaudeConnectionTimeout(
        setClaudeConnection(
          enabling,
          enabling && !status.used,
          restartConfirmed,
        ),
        CLAUDE_CONNECTION_TIMEOUT_MS,
        reconcileLateClaudeAction(enabling),
      );
      setClaudeStatus(result.status);
      let actionError = result.error || null;
      if (!actionError && enabling && result.status.connected) {
        const openError = await finishClaudeConnection(result.status);
        actionError = openError;
      } else if (
        !actionError &&
        (enabling ? !result.status.connected : result.status.configured)
      ) {
        actionError = enabling
          ? t("onboarding.claude.connectFailed")
          : t("onboarding.claude.disconnectFailed");
      }
      setClaudeError(actionError);
    } catch (error) {
      setClaudeError(
        error instanceof ClaudeConnectionTimeoutError
          ? t("onboarding.claude.timeoutAction", { action: enabling ? t("onboarding.claude.actionConnect") : t("onboarding.claude.actionDisconnect") })
          : enabling
            ? t("onboarding.claude.connectFailed")
            : t("onboarding.claude.disconnectFailed"),
      );
    } finally {
      if (!claudeConnectedIntroPending.current) {
        claudeRestartConfirmed.current = false;
      }
      setClaudePhase("idle");
    }
  };

  const claudeIntegration = isWindows
    ? undefined
    : integrationStatuses?.find((item) => item.id === "claude-desktop");
  const codexIntegration = isWindows
    ? undefined
    : (integrationStatuses?.find((item) => item.id === "chatgpt") ?? {
        id: "chatgpt",
        name: "ChatGPT (Desktop)",
        description: t("onboarding.chatgpt.description"),
        installed: false,
      });
  const launchIntegrations =
    integrationStatuses?.filter(
      (item) =>
        item.id !== "claude-desktop" && item.id !== "chatgpt" && item.command,
    ) ?? [];
  const claudeConnected = claudeStatus?.connected ?? false;
  const claudeConfigured = claudeStatus
    ? isClaudeConfigured(claudeStatus)
    : false;
  const pendingClaudeConnection =
    claudePhase === "installing" ||
    claudePhase === "waiting-for-install" ||
    claudePhase === "connecting" ||
    claudePhase === "launching"
      ? true
      : claudePhase === "disconnecting"
        ? false
        : null;
  const claudeToggleConfigured = optimisticClaudeConnectionState(
    claudeConfigured,
    pendingClaudeConnection,
  );
  const claudeInstalled =
    claudeStatus?.installed ?? claudeIntegration?.installed ?? false;
  const isConnectingClaude = claudePhase !== "idle";
  const claudeStatusLabel =
    claudePhase === "installing"
      ? t("onboarding.claude.statusDownloading")
      : claudePhase === "waiting-for-install"
        ? t("onboarding.claude.statusFinishingInstall")
        : claudePhase === "connecting"
          ? t("onboarding.claude.statusConnecting")
          : claudePhase === "launching"
            ? t("onboarding.claude.statusLaunching")
            : claudePhase === "disconnecting"
              ? t("onboarding.claude.statusDisconnecting")
              : null;
  const claudeGuidance = claudeDesktopRecoveryMessage(
    claudeStatus?.error,
    claudeError,
  );
  const launchIntegrationCard = (item: IntegrationStatus) => {
    const copied =
      copyNotice?.id === item.id && copyNotice.copied && copyNotice.visible;
    return (
      <button
        key={item.id}
        id={`integration-${item.id}`}
        type="button"
        onClick={() => copyLaunchCommand(item)}
        aria-label={
          copied
            ? t("onboarding.apps.copiedAria", { name: item.name })
            : t("onboarding.apps.copyAria", { name: item.name })
        }
        title={item.description}
        className="relative isolate flex min-w-0 items-center gap-3 rounded-2xl border border-neutral-200 bg-white px-4 py-3 text-left transition-colors duration-700 hover:bg-neutral-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500 motion-reduce:transition-none dark:border-neutral-700 dark:bg-neutral-900 dark:hover:bg-neutral-800"
      >
        {copied && (
          <span
            key={copyNotice?.sequence}
            aria-hidden="true"
            className="apps-copy-card-feedback pointer-events-none absolute inset-0 -z-10 rounded-[inherit] bg-neutral-100 ring-1 ring-inset ring-neutral-300/70 dark:bg-neutral-700/60 dark:ring-neutral-500/50"
          />
        )}
        <LaunchCommandIcon id={item.id} />
        <span className="min-w-0 flex-1 truncate text-sm font-medium text-neutral-950 dark:text-neutral-100">
          {item.name}
        </span>
      </button>
    );
  };
  const claudeRow = claudeIntegration ? (
    <div
      id={`integration-${claudeIntegration.id}`}
      className="flex items-center gap-4 rounded-2xl border border-neutral-200 bg-neutral-50 px-5 py-3 transition-colors duration-700 motion-reduce:transition-none dark:border-neutral-700 dark:bg-neutral-800/50"
    >
      <div className="flex min-w-0 flex-1 items-center gap-4">
        <LaunchCommandIcon id={claudeIntegration.id} />
        <div className="min-w-0">
          <p className="text-base font-medium text-neutral-950 dark:text-neutral-100">
            Claude Code
          </p>
          <p
            role={claudeGuidance ? "alert" : undefined}
            className="mt-1 text-[13px] leading-5 text-neutral-500 dark:text-neutral-400"
          >
            {claudeGuidance ??
              (claudeConnected
                ? t("onboarding.claude.connected", { count: claudeDesktopRequestCountLabel(claudeStatus?.routedRequests ?? 0) })
                : claudePhase === "installing"
                  ? t("onboarding.claude.installing")
                  : claudePhase === "waiting-for-install"
                    ? t("onboarding.claude.waitingInstall")
                    : claudePhase === "connecting"
                      ? t("onboarding.claude.connectingDetail")
                      : claudePhase === "launching"
                        ? t("onboarding.claude.launchingDetail")
                        : claudePhase === "disconnecting"
                          ? t("onboarding.claude.disconnectingDetail")
                          : !claudeInstalled
                            ? t("onboarding.claude.notInstalled")
                            : t("onboarding.claude.ready"))}
          </p>
        </div>
      </div>
      <IntegrationConnectButton
        connected={claudeToggleConfigured}
        busy={!initialClaudeStatusSettled || isConnectingClaude}
        progress={isConnectingClaude ? claudeStatusLabel : null}
        label={
          claudeConfigured
            ? t("onboarding.claude.disconnect")
            : isConnectingClaude
              ? t("onboarding.claude.connecting")
              : t("onboarding.claude.connect")
        }
        title={claudeConfigured ? t("onboarding.claude.disconnectTitle") : t("onboarding.claude.connectTitle")}
        disabled={!initialClaudeStatusSettled || isConnectingClaude}
        onClick={connectClaude}
      />
    </div>
  ) : null;

  return (
    <div className="relative flex min-h-0 w-full flex-1 flex-col overflow-hidden bg-white text-neutral-950 dark:bg-neutral-900 dark:text-neutral-100">
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain px-6 pb-6 pt-4">
        <section className="min-h-0 flex-1">
          <div className="mx-auto w-full max-w-[620px] text-left">
            {integrationStatuses ? (
              <div className="space-y-8 pb-4 pt-2">
                {(claudeIntegration || codexIntegration) && (
                  <section aria-labelledby="recommended-heading">
                    <h2
                      id="recommended-heading"
                      className="text-xs font-medium uppercase tracking-wider text-neutral-400 dark:text-neutral-500"
                    >
                      {t("onboarding.apps.recommended")}
                    </h2>
                    <div className="mt-2 space-y-2 bg-white dark:bg-neutral-900">
                      {claudeRow}
                      {codexIntegration && (
                        <CodexDesktopRow
                          integration={codexIntegration}
                          initialStatus={initialCodexStatus}
                        />
                      )}
                    </div>
                  </section>
                )}

                {launchIntegrations.length > 0 && (
                  <section aria-labelledby="terminal-heading">
                    <h2
                      id="terminal-heading"
                      className="text-xs font-medium uppercase tracking-wider text-neutral-400 dark:text-neutral-500"
                    >
                      {isWindows ? t("onboarding.apps.titleApps") : t("onboarding.apps.titleOthers")}
                    </h2>
                    <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
                      {launchIntegrations.map(launchIntegrationCard)}
                    </div>
                  </section>
                )}

                {!claudeIntegration &&
                  !codexIntegration &&
                  launchIntegrations.length === 0 && (
                    <p className="py-12 text-center text-sm text-neutral-400 dark:text-neutral-500">
                      {t("onboarding.apps.empty")}
                    </p>
                  )}
              </div>
            ) : statusError ? (
              <p role="alert" className="mt-8 text-sm text-red-600">
                {t("onboarding.apps.loadFailed")}
              </p>
            ) : (
              <p className="mt-8 text-sm text-neutral-400 dark:text-neutral-500">
                {t("onboarding.apps.checking")}
              </p>
            )}
          </div>
        </section>
      </div>
      {copyNotice && (
        <Transition
          appear
          show={copyNotice.visible}
          as="div"
          className="apps-copy-hint pointer-events-none absolute right-4 top-4 z-30 w-[360px] max-w-[calc(100%-2rem)]"
        >
          <div
            ref={copyNoticeRef}
            role={copyNotice.copied ? "status" : "alert"}
            aria-live={copyNotice.copied ? "polite" : "assertive"}
            aria-atomic="true"
            className={`flex items-start gap-3 rounded-2xl border border-neutral-200/80 bg-neutral-100/95 p-4 text-[13px] shadow-lg shadow-black/10 backdrop-blur-xl dark:border-white/10 dark:bg-neutral-700/90 dark:shadow-black/30 ${!copyNotice.copied && copyNotice.visible ? "pointer-events-auto" : ""}`}
          >
            <div aria-hidden="true" className="shrink-0">
              <LaunchCommandIcon id={copyNotice.id} />
            </div>
            <div className="min-w-0 flex-1">
              <p className="font-semibold leading-5 text-neutral-900 dark:text-neutral-100">
                {copyNotice.copied
                  ? t("onboarding.apps.copiedTitle")
                  : t("onboarding.apps.copyFailedTitle", { name: copyNotice.name })}
              </p>
              <p className="mt-0.5 leading-5 text-neutral-600 dark:text-neutral-200">
                {copyNotice.copied
                  ? t("onboarding.apps.pasteHint")
                  : t("onboarding.apps.selectHint")}
              </p>
              {!copyNotice.copied && (
                <code className="mt-2 block select-all break-all rounded-md bg-white/70 px-3 py-2 text-xs dark:bg-neutral-900/60">
                  {copyNotice.command}
                </code>
              )}
            </div>
          </div>
        </Transition>
      )}
      {showClaudeConnectedIntro && (
        <ClaudeConnectedIntro
          onDone={() => void dismissClaudeConnectedIntro()}
        />
      )}
    </div>
  );
}

interface OnboardingProps extends ScreenProps {
  completionError: string | null;
  isAuthenticated: boolean;
  isLocalOnly: boolean;
  onOpenApps: () => Promise<boolean>;
  onRetryCompletion: () => void;
  onSignUp: () => void;
  onUseLocal: () => void;
}

export default function Onboarding(props: OnboardingProps) {
  const { t } = useT();
  const [step, setStep] = useState<OnboardingStep>("intro");
  const [isLeaving, setIsLeaving] = useState(false);
  const [cloudOptIn, setCloudOptIn] = useState(false);
  const [cloudError, setCloudError] = useState<string | null>(null);
  const leavingRef = useRef(false);
  const authenticationHandoffStarted = useRef(false);
  const { onOpenApps } = props;

  const leave = useCallback(async () => {
    if (leavingRef.current) return;
    leavingRef.current = true;
    setIsLeaving(true);
    if (cloudOptIn) {
      try {
        await enableCloudModelsOptIn();
        setCloudError(null);
      } catch {
        setCloudError(t("onboarding.welcome.saveCloudFailed"));
        leavingRef.current = false;
        setIsLeaving(false);
        return;
      }
    }
    const opened = await onOpenApps();
    if (!opened) {
      leavingRef.current = false;
      setIsLeaving(false);
    }
  }, [onOpenApps, cloudOptIn]);

  useEffect(() => {
    window.setOnboardingWindow?.(true);
    return () => window.setOnboardingWindow?.(false);
  }, []);

  useEffect(() => {
    if (!props.isAuthenticated) {
      authenticationHandoffStarted.current = false;
      return;
    }
    if (step !== "welcome" || authenticationHandoffStarted.current) return;
    // A failed save waits for an explicit retry, even if query updates replace
    // the callback. StrictMode must not start a second completion either.
    authenticationHandoffStarted.current = true;
    void leave();
  }, [step, props.isAuthenticated, leave]);

  if (step === "run") {
    return (
      <RunOllamaScreen
        completionError={props.completionError}
        onRetryCompletion={props.onRetryCompletion}
        onContinue={() => void onOpenApps()}
      />
    );
  }

  if (step === "intro") {
    return (
      <IntroScreen
        completionError={props.completionError}
        isLeaving={isLeaving}
        /**
         * E-936 (cura do `E-818`): **«Tentar de novo» TEM DE VOLTAR A GRAVAR.**
         *
         * Estava `() => void leave()`, e o `E-935` mediu a consequencia: *o botao ao lado de
         * «Unable to save setup. Please try again.» **sai da pagina em vez de repetir a
         * escrita*** - e' o `E-818` do dono: «Tentar de novo apaga o erro e nao grava».
         *
         * `props.onRetryCompletion` e' o `retryCompletion` de `routes/onboarding.tsx:113-117`,
         * que chama `completeOnboarding(true)` - *com `forcar`, que e' o que faz a escrita
         * repetida passar a guarda (`useSettings.ts:278`)*.
         *
         * *O `leave()` continua nos `onContinue`, que e' onde ele faz sentido: sair quando a
         * gravacao **correu bem**.*
         */
        onRetryCompletion={props.onRetryCompletion}
        onContinue={() => {
          if (props.isAuthenticated && props.isLocalOnly) setStep("run");
          else if (props.isAuthenticated) void leave();
          else setStep("welcome");
        }}
      />
    );
  }

  return (
    <WelcomeScreen
      {...props}
      completionError={cloudError ?? props.completionError}
      cloudOptIn={cloudOptIn}
      onCloudOptInChange={(enabled) => { setCloudOptIn(enabled); setCloudError(null); }}
      isLeaving={isLeaving}
      // E-936: mesma cura do `IntroScreen` - *ver o commentario acima.* O `E-935` mediu que
      // estes dois botoes chamavam `leave()` em vez de `retryCompletion()`.
      onRetryCompletion={props.onRetryCompletion}
      onLocal={() => {
        if (leavingRef.current) return;
        props.onUseLocal();
        setStep("run");
      }}
    />
  );
}
