import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import { createFileRoute } from "@tanstack/react-router";
import Settings from "@/components/Settings";
import { SettingsTabs } from "@/components/SettingsTabs";
import { ModelsPanel } from "@/components/ModelsPanel";
import { DataToolsPanel } from "@/components/DataToolsPanel";
import { useT } from "@/i18n";

export const Route = createFileRoute("/settings")({
  component: SettingsRoute,
});

function SettingsRoute() {
  const { t } = useT();
  return (
    <SidebarLayout
      title={t("appSidebar.nav.settings")}
      sidebar={<AppSidebar current="settings" />}
    >
      <SettingsTabs current="general" />
      {/* Single scroll column: Settings sizes to its content so the panels
          below it are never overlapped (previously Settings used flex-1 +
          its own scroll, leaving 0 height for the sibling panels). */}
      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain pb-12">
        <Settings />
        <div className="mx-auto mt-6 w-full max-w-4xl px-6">
          <ModelsPanel />
        </div>
        <div className="mx-auto mt-6 w-full max-w-4xl px-6">
          <DataToolsPanel />
        </div>
      </div>
    </SidebarLayout>
  );
}
