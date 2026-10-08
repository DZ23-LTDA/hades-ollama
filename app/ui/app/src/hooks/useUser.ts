import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchUser, fetchConnectUrl, disconnectUser } from "@/api";

// userRetryDelay backs off between /api/me retries. attemptIndex is 0-based, so
// we add 1 — otherwise the first retry fires at 0ms and hammers a backend that
// just failed before any backoff applies. Pure, for testing.
export function userRetryDelay(attemptIndex: number): number {
  return Math.min(500 * (attemptIndex + 1), 2000);
}

export function useUser() {
  const queryClient = useQueryClient();

  const userQuery = useQuery({
    queryKey: ["user"],
    queryFn: async () => {
      const result = await fetchUser();
      return result;
    },
    staleTime: 5 * 60 * 1000, // Consider data stale after 5 minutes
    gcTime: 10 * 60 * 1000, // Keep in cache for 10 minutes
    retry: 10,
    retryDelay: userRetryDelay,
    refetchOnMount: true, // Always fetch when component mounts
  });

  // Mutation to refresh user data
  const refreshUser = useMutation({
    mutationFn: () => fetchUser(),
    onSuccess: (data) => {
      queryClient.setQueryData(["user"], data);
    },
  });

  // Query for connect URL (only fetched when needed)
  const connectUrlQuery = useQuery({
    queryKey: ["connectUrl"],
    queryFn: fetchConnectUrl,
    enabled: false, // Don't fetch automatically
    staleTime: Infinity, // Connect URL doesn't change
  });

  const disconnectMutation = useMutation({
    mutationFn: disconnectUser,
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: ["user"] });
      const previousUser = queryClient.getQueryData(["user"]);
      queryClient.setQueryData(["user"], null);

      return { previousUser };
    },
    onError: (_error, _variables, context) => {
      queryClient.setQueryData(["user"], context?.previousUser);
    },
  });

  const isLoading = userQuery.isLoading || userQuery.isFetching;
  const isLocalOnly = userQuery.data?.local_only === true;
  // The backend always returns a placeholder "Local Operator" identity when no
  // Ollama account is connected (local_only=true). Treating that as signed-in
  // hid the "Entrar" affordances and let cloud requests fail silently, so a
  // real account requires a name AND local_only to be false.
  const isAuthenticated = Boolean(userQuery.data?.name) && !isLocalOnly;

  return {
    user: userQuery.data,
    isLoading,
    isError: userQuery.isError,
    error: userQuery.error,
    isAuthenticated,
    isLocalOnly,
    refreshUser: refreshUser.mutate,
    isRefreshing: refreshUser.isPending,
    refetchUser: userQuery.refetch,
    fetchConnectUrl: connectUrlQuery.refetch,
    connectUrl: connectUrlQuery.data,
    disconnectUser: disconnectMutation.mutateAsync,
  };
}
