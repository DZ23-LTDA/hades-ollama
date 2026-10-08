import { useQuery } from "@tanstack/react-query";
import { fetchHealth } from "@/api";

// How often to re-check a backend that is currently down or still starting.
// The previous value (10ms) hammered the local backend ~100x/second while it was
// unavailable; a couple of seconds is responsive enough for a backend that is
// still booting without thrashing CPU/network.
export const HEALTH_RECHECK_INTERVAL_MS = 2000;

// healthRefetchInterval keeps re-checking only while the backend is known to be
// down; it stops once healthy (or still unknown). Pure, for testing.
export function healthRefetchInterval(
  data: boolean | undefined,
): number | false {
  return data === false ? HEALTH_RECHECK_INTERVAL_MS : false;
}

export function useHealth() {
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: (query) => healthRefetchInterval(query.state.data),
    refetchIntervalInBackground: true,
    retry: false, // Don't retry, just return false
    staleTime: 0, // Always consider stale so we keep polling
  });

  return {
    isHealthy: healthQuery.data ?? false,
    isChecking: healthQuery.isLoading,
  };
}
