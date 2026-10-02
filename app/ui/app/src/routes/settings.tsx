import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { createFileRoute } from "@tanstack/react-router";
import Settings from "@/components/Settings";
import { SettingsTabs } from "@/components/SettingsTabs";
import { ModelsPanel } from "@/components/ModelsPanel";

export const Route = createFileRoute("/settings")({
  component: SettingsRoute,
});

function SettingsRoute() {
  return (
    <SidebarLayout
      title="Configurações"
      sidebar={<AppSidebar current="settings" />}
    >
      <SettingsTabs current="general" />
      <Settings />
      <div className="mt-6">
        <ModelsPanel />
      </div>
    </SidebarLayout>
  );
}
