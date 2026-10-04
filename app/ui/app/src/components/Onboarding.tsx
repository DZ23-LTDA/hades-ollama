import { CodexDesktopRow } from "@/components/CodexDesktopRow";
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
          Entrar
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
    title: "Conecte seus apps",
    description: "Turbine seus apps de código com modelos abertos",
    icon: CommandLineIcon,
  },
  {
    title: "Troque de modelo facilmente",
    description: "Alterne entre modelos de ponta em um clique.",
    icon: ArrowsRightLeftIcon,
  },
  {
    title: "Seus dados continuam seus",
    description: "Modelos locais processam seus pedidos neste computador. Modelos de nuvem só recebem pedidos quando você escolhe usá-los.",
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
  return (
    <main className="light-only flex h-screen w-full flex-col overflow-hidden bg-white text-neutral-950">
      <TitleBar />

      <section className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto px-6 pb-10">
        <div className="mx-auto flex min-h-full w-full max-w-[620px] flex-col items-center justify-center py-4 text-center">
          <div className="flex flex-col items-center justify-center gap-2">
            <img
              src="/hello.png"
              alt="Hades dando boas-vindas"
              className="h-[72px] w-[72px] select-none object-contain"
              draggable={false}
            />
            <h1 className="font-rounded text-2xl font-medium leading-8">
              Bem-vindo ao Hades!
            </h1>
          </div>
          <p className="mt-4 max-w-[380px] text-sm leading-6 text-neutral-500">
            Rode modelos abertos com seus agentes de código para gastar menos
            mantendo seus dados privados.
          </p>
          <div className="mx-auto mt-8 flex w-fit max-w-full flex-col gap-6 text-left">
            {OLLAMA_FEATURES.map((feature) => {
              const Icon = feature.icon;

              return (
                <div key={feature.title} className="flex items-start gap-4">
                  <Icon className="mt-0.5 h-7 w-7 shrink-0 stroke-[1.5] text-neutral-700" />
                  <div>
                    <h2 className="text-sm font-medium text-neutral-950">
                      {feature.title}
                    </h2>
                    <p className="mt-0.5 text-[13px] leading-5 text-neutral-500">
                      {feature.description}
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
            Continuar
          </button>
          <InlineError message={completionError} />
          {completionError && onRetryCompletion && (
            <button
              type="button"
              className="mt-2 cursor-pointer rounded-md px-3 py-1 text-sm font-normal text-neutral-600 underline decoration-neutral-300 underline-offset-4 hover:text-neutral-950 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
              onClick={onRetryCompletion}
            >
              Tentar de novo
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
  return (
    <main className="light-only flex min-h-screen w-full flex-col bg-white text-neutral-950">
      <TitleBar onSignIn={isAuthenticated ? undefined : onSignIn} />
      <OnboardingCard>
        <OnboardingIcon />
        <h1 className="mt-7 font-rounded text-2xl font-medium leading-8">
          Crie uma conta
        </h1>
        <p className="mt-3 max-w-[400px] text-sm leading-6 text-neutral-400">
          Crie sua conta para acessar modelos abertos maiores e mais rápidos.
        </p>
        <p className="mt-1 max-w-[400px] text-sm leading-6 text-neutral-400">
          Modelos locais processam seus pedidos neste computador. Se você ativar os modelos de nuvem, os pedidos são enviados ao provedor escolhido quando você os usa.
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
              ? "Abrindo os apps…"
              : isSigningIn
                ? "Conclua no seu navegador…"
                : "Criar conta"}
          </button>
          <label className="mt-4 flex items-center gap-2 text-xs text-neutral-600 select-none cursor-pointer">
            <input
              type="checkbox"
              checked={cloudOptIn ?? false}
              onChange={(e) => onCloudOptInChange?.(e.target.checked)}
              className="rounded border-neutral-300 text-neutral-900 focus:ring-neutral-500 cursor-pointer"
            />
            <span>Ativar modelos de nuvem grátis da minha conta</span>
          </label>
          <button
            type="button"
            className="mt-2 cursor-pointer rounded-md px-3 py-2 text-sm font-normal text-neutral-600 underline decoration-neutral-300 underline-offset-4 hover:text-neutral-950 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
            onClick={onLocal}
            disabled={isLeaving}
          >
            Não, obrigado — vou usar os modelos locais
          </button>
          <InlineError message={signInError ?? completionError} />
          {completionError && onRetryCompletion && (
            <button
              type="button"
              className="mt-2 cursor-pointer rounded-md px-3 py-1 text-sm font-normal text-neutral-600 underline decoration-neutral-300 underline-offset-4 hover:text-neutral-950 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
              onClick={onRetryCompletion}
            >
              Tentar de novo
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
    setPullStatus("Preparando download…");
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
      setPullStatus(`Modelo ${requestedModel} pronto para uso local.`);
      setModelReady(true);
    } catch (error) {
      setPullError(
        error instanceof Error
          ? error.message
          : "Não foi possível baixar o modelo neste computador.",
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
          Baixe seu primeiro modelo
        </h1>

        <p className="mt-3 max-w-xs text-[13px] leading-5 text-neutral-400">
          Escolha um modelo local para começar. O Hades baixa e roda o modelo no
          seu computador — sem precisar de terminal.
        </p>

        <div className="mt-5 flex w-full max-w-[360px] flex-col gap-2 text-left">
          <label htmlFor="onboarding-model" className="text-xs font-medium text-neutral-700">
            Modelo local
          </label>
          <div className="flex gap-2">
            <input
              id="onboarding-model"
              value={modelName}
              onChange={(event) => setModelName(event.target.value)}
              disabled={pulling}
              className="min-w-0 flex-1 rounded-xl border border-neutral-200 px-3 py-2 text-sm outline-none focus:border-neutral-500 focus:ring-2 focus:ring-neutral-200"
              placeholder={`ex.: ${RECOMMENDED_FIRST_MODEL}`}
            />
            <button
              type="button"
              onClick={() => void downloadModel()}
              disabled={pulling || !modelName.trim()}
              className="rounded-xl bg-neutral-900 px-3 py-2 text-xs font-medium text-white disabled:cursor-wait disabled:opacity-60"
              aria-busy={pulling}
            >
              {pulling ? "Baixando…" : "Baixar"}
            </button>
          </div>
          <p className="text-[11px] text-neutral-400">
            O arquivo pode ser grande; mantenha o Ollama em execução durante o processo.
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
            Continuar
          </button>
        )}
        {completionError && (
          <button
            type="button"
            className="mt-2 cursor-pointer rounded-md px-3 py-1 text-sm font-normal text-neutral-600 underline decoration-neutral-300 underline-offset-4 hover:text-neutral-950 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500"
            onClick={onRetryCompletion}
          >
            Tentar de novo
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
          alt="Exemplo de mapeamento de modelos do Claude nas configurações do Hades"
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
            Acesse os modelos do Hades no seu Claude
          </h2>
          <p
            id="claude-connected-description"
            className="mt-2 text-[13px] leading-5 text-neutral-500 dark:text-neutral-400"
          >
            O Hades roteia os modelos do Claude automaticamente. Abra as
            configurações do Hades para escolher quais modelos o Claude Desktop
            usa.
          </p>
          <div className="mt-11 flex justify-end">
            <button
              type="button"
              autoFocus
              className="rounded-full bg-neutral-100 px-6 py-2 text-sm font-normal text-neutral-950 transition-colors hover:bg-neutral-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-neutral-500 dark:bg-white dark:hover:bg-neutral-100"
              onClick={onDone}
            >
              Continuar
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
        setClaudeError("O Hades não conseguiu ler o status da conexão com o Claude.");
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
        setClaudeError("O Hades não conseguiu ler o status da conexão com o Claude.");
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
        return "O Hades conectou o Claude, mas não conseguiu abrir o app.";
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
                  "O Hades conectou o Claude, mas não conseguiu abrir o app.",
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
                ? "O Hades não conseguiu conectar ao Claude."
                : "O Hades não conseguiu desconectar do Claude.",
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
        restartConfirmed = window.confirm(
          "Reiniciar Claude Desktop para usar Ollama? Qualquer tarefa em execução será interrompida.",
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
          ? "O Claude está demorando demais para abrir. Verifique o Claude e tente de novo."
          : "O Hades conectou o Claude, mas não conseguiu abrir o app.",
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
            "O Claude está instalado. Ative o Conectar para reiniciá-lo com o Hades.",
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
          actionError = "O Hades não conseguiu conectar ao Claude.";
        }
        setClaudeError(actionError);
        setClaudePhase("idle");
      } catch (error) {
        if (!screenMounted.current) return;
        setClaudePhase("idle");
        setClaudeError(
          error instanceof ClaudeConnectionTimeoutError
            ? "O Claude está demorando demais para conectar. Verifique o Claude e tente de novo."
            : "O Hades não conseguiu concluir a conexão com o Claude.",
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
      setClaudeError("A instalação do Claude não foi detectada. Tente novamente.");
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
      setClaudeError("A conexão com o Claude está disponível no app do Hades para macOS.");
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
          ? `O Claude está demorando demais para ${enabling ? "conectar" : "desconectar"}. Tente de novo.`
          : "O Hades não conseguiu ler o status da conexão com o Claude.",
      );
      return;
    }
    if (!screenMounted.current) return;
    if (!status) {
      setClaudePhase("idle");
      setClaudeError("O Hades não conseguiu ler o status da conexão com o Claude.");
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
        setClaudeError("O Hades não conseguiu abrir o instalador do Claude.");
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
        setClaudeError("O Hades não conseguiu abrir o instalador do Claude.");
        return;
      }
      setClaudePhase("waiting-for-install");
      return;
    }

    let restartConfirmed = false;
    if (status.running) {
      restartConfirmed = window.confirm(
        enabling
          ? "Reiniciar Claude Desktop para usar Ollama? Qualquer tarefa em execução será interrompida."
          : "Reiniciar Claude Desktop para remover Ollama? Qualquer tarefa em execução será interrompida.",
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
          ? "O Hades não conseguiu conectar ao Claude."
          : "O Hades não conseguiu desconectar do Claude.";
      }
      setClaudeError(actionError);
    } catch (error) {
      setClaudeError(
        error instanceof ClaudeConnectionTimeoutError
          ? `O Claude está demorando demais para ${enabling ? "conectar" : "desconectar"}. Tente de novo.`
          : enabling
            ? "O Hades não conseguiu conectar ao Claude."
            : "O Hades não conseguiu desconectar do Claude.",
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
        description: "Use os modelos do Hades no ChatGPT",
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
      ? "Baixando…"
      : claudePhase === "waiting-for-install"
        ? "Conclua a instalação…"
        : claudePhase === "connecting"
          ? "Conectando…"
          : claudePhase === "launching"
            ? "Abrindo…"
            : claudePhase === "disconnecting"
              ? "Desconectando…"
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
            ? `Comando do ${item.name} copiado`
            : `Copiar comando do ${item.name}`
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
                ? `Conectado ao Hades · ${claudeDesktopRequestCountLabel(claudeStatus?.routedRequests ?? 0)}`
                : claudePhase === "installing"
                  ? "O Hades está baixando o instalador do Claude…"
                  : claudePhase === "waiting-for-install"
                    ? "Conclua a instalação do Claude. O Hades vai conectá-lo automaticamente."
                    : claudePhase === "connecting"
                      ? "Conectando o Claude ao Hades…"
                      : claudePhase === "launching"
                        ? "Abrindo o Claude…"
                        : claudePhase === "disconnecting"
                          ? "Restaurando a conexão normal do Claude…"
                          : !claudeInstalled
                            ? "Vamos baixar o Claude e conectá-lo ao Hades."
                            : "Use os modelos do Hades no seu Claude Code.")}
          </p>
        </div>
      </div>
      <IntegrationConnectButton
        connected={claudeToggleConfigured}
        busy={!initialClaudeStatusSettled || isConnectingClaude}
        progress={isConnectingClaude ? claudeStatusLabel : null}
        label={
          claudeConfigured
            ? "Desconectar Claude"
            : isConnectingClaude
              ? "Conectando Claude"
              : "Conectar Claude"
        }
        title={claudeConfigured ? "Desconectar" : "Conectar"}
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
                      Recomendados
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
                      {isWindows ? "Apps" : "Outros apps"}
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
                      Nenhum app encontrado.
                    </p>
                  )}
              </div>
            ) : statusError ? (
              <p role="alert" className="mt-8 text-sm text-red-600">
                Não foi possível carregar as integrações.
              </p>
            ) : (
              <p className="mt-8 text-sm text-neutral-400 dark:text-neutral-500">
                Verificando integrações…
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
                  ? "Comando de inicialização copiado."
                  : `Não foi possível copiar o comando do ${copyNotice.name}`}
              </p>
              <p className="mt-0.5 leading-5 text-neutral-600 dark:text-neutral-200">
                {copyNotice.copied
                  ? "Cole no seu terminal"
                  : "Selecione e copie o comando abaixo e cole no seu terminal."}
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
        setCloudError("Não foi possível salvar sua escolha de modelos de nuvem. A ativação não foi confirmada; tente novamente.");
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
        onRetryCompletion={() => void leave()}
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
      onRetryCompletion={() => void leave()}
      onLocal={() => {
        if (leavingRef.current) return;
        props.onUseLocal();
        setStep("run");
      }}
    />
  );
}
