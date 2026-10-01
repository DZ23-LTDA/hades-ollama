import { useSyncExternalStore } from "react";
import { EyeSlashIcon } from "@heroicons/react/24/outline";
import {
  isAnonymousChat,
  isAnonymousModeEnabled,
  setAnonymousMode,
  subscribeAnonymousChat,
} from "@/lib/anonymousChat";

// Toggle shown on a new chat, and a notice while an anonymous chat is open.
export function AnonymousChatToggle({ chatId }: { chatId: string }) {
  const enabled = useSyncExternalStore(
    subscribeAnonymousChat,
    isAnonymousModeEnabled,
    isAnonymousModeEnabled,
  );
  const anonymousOpen = useSyncExternalStore(
    subscribeAnonymousChat,
    () => isAnonymousChat(chatId),
    () => false,
  );

  if (chatId !== "new") {
    if (!anonymousOpen) return null;
    return (
      <p
        role="status"
        className="mx-auto mb-2 flex w-fit items-center gap-2 rounded-full bg-neutral-200 px-3 py-1 text-xs text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300"
      >
        <EyeSlashIcon className="h-4 w-4" />
        Chat anônimo: não é salvo no histórico e some ao fechar o app.
      </p>
    );
  }

  return (
    <div className="mx-auto mb-2 flex w-fit">
      <button
        type="button"
        aria-pressed={enabled}
        onClick={() => setAnonymousMode(!enabled)}
        title="Conversas anônimas não são salvas no histórico"
        className={`flex items-center gap-2 rounded-full border px-3 py-1 text-xs transition-colors ${
          enabled
            ? "border-neutral-900 bg-neutral-900 text-white dark:border-white dark:bg-white dark:text-neutral-900"
            : "border-neutral-300 text-neutral-600 hover:bg-neutral-100 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-800"
        }`}
      >
        <EyeSlashIcon className="h-4 w-4" />
        {enabled ? "Chat anônimo ativado" : "Chat anônimo"}
      </button>
    </div>
  );
}
