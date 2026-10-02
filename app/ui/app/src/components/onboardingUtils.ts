import type { ClaudeDesktopStatus } from "@/types/webview";

export const FIRST_MODEL_COMMAND = "ollama run qwen2.5:0.5b";

export function shouldShowClaudeConnectedIntro(status: ClaudeDesktopStatus) {
  return status.connected && !status.startFailed && !status.used;
}
