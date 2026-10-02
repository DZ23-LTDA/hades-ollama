import { disconnectUser } from "@/api";
import { clearAgentSession } from "@/lib/agenticClient";

export async function performLogout(): Promise<void> {
  await disconnectUser();
  clearAgentSession();
}
