import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import { getSettings } from "@/api";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCloudStatus } from "@/hooks/useCloudStatus";
import { preloadChatData } from "@/lib/chatPreload";
import { preventPageSelectAll } from "@/lib/keyboard";
import { useEffect } from "react";
import { BackendStatusBanner } from "@/components/BackendStatusBanner";
import { RouteErrorFallback } from "@/components/RouteErrorFallback";

function RootComponent() {
  const queryClient = useQueryClient();

  useEffect(() => {
    document.addEventListener("keydown", preventPageSelectAll);
    return () => document.removeEventListener("keydown", preventPageSelectAll);
  }, []);

  useEffect(() => {
    void preloadChatData(queryClient);
  }, [queryClient]);

  // This hook ensures settings are fetched on app startup
  useQuery({
    queryKey: ["settings"],
    queryFn: getSettings,
  });
  // Fetch cloud status on startup (best-effort)
  useCloudStatus();

  return (
    <div>
      <BackendStatusBanner />
      <Outlet />
    </div>
  );
}

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient;
}>()({
  component: RootComponent,
  // Friendly route-level fallback so a failing page never traps the user with
  // the bare default error component.
  errorComponent: RouteErrorFallback,
});
