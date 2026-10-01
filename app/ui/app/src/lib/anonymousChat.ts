// Anonymous ("temporary") chats are kept only in the desktop server's memory
// and never written to history. This module tracks the toggle for new chats
// and which open chats are anonymous.
type Listener = () => void;

let enabled = false;
const anonymousIds = new Set<string>();
const listeners = new Set<Listener>();

function emit() {
  for (const listener of listeners) listener();
}

export function subscribeAnonymousChat(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function isAnonymousModeEnabled(): boolean {
  return enabled;
}

export function setAnonymousMode(value: boolean) {
  if (enabled === value) return;
  enabled = value;
  emit();
}

export function markAnonymousChat(chatId: string) {
  anonymousIds.add(chatId);
  emit();
}

export function isAnonymousChat(chatId: string | undefined): boolean {
  return !!chatId && anonymousIds.has(chatId);
}

// shouldSendTemporary decides the request flag: only a new chat started while
// anonymous mode is on is created as temporary; the server remembers it.
export function shouldSendTemporary(chatId: string): boolean {
  return chatId === "new" && enabled;
}
