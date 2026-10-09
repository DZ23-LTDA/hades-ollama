import { useEffect, useState } from "react";

// In-app confirmation dialog. `window.confirm` is unusable inside the desktop
// webview (WebView2 suppresses script dialogs: confirm() returns false
// immediately, so every `if (!confirm()) return` silently aborts the action).
// This replaces it with an async, Promise-based dialog rendered by <ConfirmHost>.
// Call sites use `if (!(await confirmDialog(message))) return;`.

export interface ConfirmOptions {
  title?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
}

interface Pending extends ConfirmOptions {
  message: string;
  resolve: (value: boolean) => void;
  // Element focused when the dialog was requested, so focus can be restored to
  // it after the dialog closes (keyboard/screen-reader users keep their place).
  trigger: HTMLElement | null;
}

let notify: ((pending: Pending | null) => void) | null = null;
const queue: Pending[] = [];

function emitNext() {
  if (notify) notify(queue[0] ?? null);
}

export function confirmDialog(
  message: string,
  opts: ConfirmOptions = {},
): Promise<boolean> {
  if (typeof window === "undefined") return Promise.resolve(false);
  // No host mounted (should not happen in the app, but be safe in tests/SSR):
  // fall back to the native confirm when available, otherwise resolve false.
  if (!notify) {
    try {
      return Promise.resolve(Boolean(window.confirm(message)));
    } catch {
      return Promise.resolve(false);
    }
  }
  const trigger =
    document.activeElement instanceof HTMLElement ? document.activeElement : null;
  return new Promise<boolean>((resolve) => {
    queue.push({ message, resolve, trigger, ...opts });
    emitNext();
  });
}

export function ConfirmHost() {
  const [current, setCurrent] = useState<Pending | null>(null);

  useEffect(() => {
    notify = setCurrent;
    emitNext();
    return () => {
      notify = null;
    };
  }, []);

  useEffect(() => {
    if (!current) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") finish(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [current]);

  function finish(result: boolean) {
    const pending = queue.shift();
    pending?.resolve(result);
    emitNext();
    // Restore focus to whatever was focused when this dialog opened, once no
    // further dialog is queued, after the dialog has unmounted.
    const trigger = pending?.trigger;
    if (trigger && queue.length === 0) {
      window.setTimeout(() => {
        try {
          trigger.focus();
        } catch {
          // element may have been removed from the DOM; ignore.
        }
      }, 0);
    }
  }

  if (!current) return null;

  const confirmLabel = current.confirmLabel ?? "Confirmar";
  const cancelLabel = current.cancelLabel ?? "Cancelar";

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={current.title ?? "Confirmação"}
      className="fixed inset-0 z-[100] flex items-center justify-center bg-black/50 p-4"
      onClick={() => finish(false)}
    >
      <div
        className="w-full max-w-sm rounded-2xl bg-white p-5 shadow-xl dark:bg-neutral-900"
        onClick={(e) => e.stopPropagation()}
      >
        {current.title && (
          <h2 className="mb-2 text-base font-semibold text-neutral-900 dark:text-white">
            {current.title}
          </h2>
        )}
        <p className="text-sm text-neutral-700 dark:text-neutral-300">
          {current.message}
        </p>
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={() => finish(false)}
            className="rounded-lg px-3 py-1.5 text-sm font-medium text-neutral-700 hover:bg-neutral-100 dark:text-neutral-200 dark:hover:bg-neutral-800"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            autoFocus
            onClick={() => finish(true)}
            className={
              current.danger
                ? "rounded-lg bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700"
                : "rounded-lg bg-neutral-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-neutral-700 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
            }
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
