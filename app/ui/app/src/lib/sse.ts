// Shared Server-Sent-Events reconnect policy.
//
// A native EventSource auto-reconnects forever when the endpoint errors (~every
// few seconds), so a down, 404 or erroring stream endpoint turns into a
// reconnect storm against the backend — the same "hammer a down backend"
// pattern as a tight poll. Callers count consecutive failed reconnects (reset on
// a successful open) and stop once this cap is reached, closing the EventSource.
export const MAX_SSE_RETRIES = 5;

export function shouldStopSseReconnect(
  consecutiveErrors: number,
  max: number = MAX_SSE_RETRIES,
): boolean {
  return consecutiveErrors >= max;
}
