// A generous default timeout for short request/response API calls. Without it, a
// backend that accepts the TCP connection but never responds (half-open socket,
// hung handler) leaves `await fetch(...)` pending forever and the UI stuck — for
// a route loader like getChat, the page never leaves its loading state.
//
// This is NOT for streaming endpoints (e.g. sendMessage), whose responses stay
// open by design.
export const DEFAULT_FETCH_TIMEOUT_MS = 15000;

// timeoutSignal returns an AbortSignal that fires after `ms`, or undefined when
// the runtime lacks AbortSignal.timeout (then the fetch simply has no timeout,
// as before — never worse than the previous behavior).
export function timeoutSignal(
  ms: number = DEFAULT_FETCH_TIMEOUT_MS,
): AbortSignal | undefined {
  if (
    typeof AbortSignal !== "undefined" &&
    typeof AbortSignal.timeout === "function"
  ) {
    return AbortSignal.timeout(ms);
  }
  return undefined;
}
