import type { ClaudeDesktopStatus } from "@/types/webview";
import { RECOMMENDED_FIRST_MODEL } from "@/lib/firstModel";

export const FIRST_MODEL_COMMAND = `ollama run ${RECOMMENDED_FIRST_MODEL}`;

export function shouldShowClaudeConnectedIntro(status: ClaudeDesktopStatus) {
  return status.connected && !status.startFailed && !status.used;
}
