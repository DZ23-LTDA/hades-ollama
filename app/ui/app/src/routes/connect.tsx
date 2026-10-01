import { AppSidebar } from "@/components/AppSidebar";
import { SettingsTabs } from "@/components/SettingsTabs";
import { ConnectAppsScreen } from "@/components/Onboarding";
import { SidebarLayout } from "@/components/layout/layout";
import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/connect")({
  component: ConnectRoute,
});

function ConnectRoute() {
  return (
    <SidebarLayout
      title="Configurações"
      sidebar={<AppSidebar current="apps" />}
    >
      <SettingsTabs current="apps" />
      <ConnectAppsScreen />
    </SidebarLayout>
  );
}
