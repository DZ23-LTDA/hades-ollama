import { AppSidebar } from "@/components/AppSidebar";
import { SettingsTabs } from "@/components/SettingsTabs";
import { ConnectAppsScreen } from "@/components/Onboarding";
import { SidebarLayout } from "@/components/layout/layout";
import { createFileRoute } from "@tanstack/react-router";
import { useT } from "@/i18n";

export const Route = createFileRoute("/connect")({
  component: ConnectRoute,
});

function ConnectRoute() {
  const { t } = useT();
  return (
    <SidebarLayout
      title={t("appSidebar.nav.settings")}
      sidebar={<AppSidebar current="apps" />}
    >
      <SettingsTabs current="apps" />
      <ConnectAppsScreen />
    </SidebarLayout>
  );
}
