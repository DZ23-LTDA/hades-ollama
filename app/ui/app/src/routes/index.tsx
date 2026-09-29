import { createFileRoute, redirect } from "@tanstack/react-router";
import { getSettings } from "@/api";
import { CURRENT_ONBOARDING_VERSION } from "@/lib/onboarding";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { HomePage } from "@/components/HomePage";

export const Route = createFileRoute("/")({
  beforeLoad: async ({ context }) => {
    let settingsData: Awaited<ReturnType<typeof getSettings>> | undefined;
    try {
      settingsData = await context.queryClient.fetchQuery({
        queryKey: ["settings"],
        queryFn: getSettings,
        staleTime: 0,
      });
    } catch {
      // The local-first shell must remain usable when the optional preferences
      // API is offline. Chat/agent requests will show their own safe states.
    }
    if (
      settingsData &&
      settingsData.settings.OnboardingVersion < CURRENT_ONBOARDING_VERSION
    ) {
      throw redirect({ to: "/onboarding" });
    }
  },
  component: HomeRoute,
});

function HomeRoute() {
  return (
    <SidebarLayout title="Início" sidebar={<AppSidebar current="chat" />}>
      <HomePage />
    </SidebarLayout>
  );
}
