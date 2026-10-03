import type * as React from "react";

import { Separator } from "@/components/ui/separator";
import { ToggleRow } from "@/components/settings/AppearanceSection";
import { useSettingsStore } from "@/stores/settings.store";

/** Desktop notification preferences for DMs, mentions and start-up. */
export function NotificationSection(): React.JSX.Element {
  const notificationsEnabled = useSettingsStore((state) => state.notificationsEnabled);
  const setNotificationsEnabled = useSettingsStore((state) => state.setNotificationsEnabled);
  const notifyOnMention = useSettingsStore((state) => state.notifyOnMention);
  const setNotifyOnMention = useSettingsStore((state) => state.setNotifyOnMention);
  const notifyOnDM = useSettingsStore((state) => state.notifyOnDM);
  const setNotifyOnDM = useSettingsStore((state) => state.setNotifyOnDM);
  const autostart = useSettingsStore((state) => state.autostart);
  const setAutostart = useSettingsStore((state) => state.setAutostart);

  return (
    <div className="space-y-6 py-4">
      <ToggleRow
        id="notifications-enabled"
        label="Enable notifications"
        description="Show a system notification for new activity."
        checked={notificationsEnabled}
        onCheckedChange={setNotificationsEnabled}
      />

      <ToggleRow
        id="notify-dm"
        label="Direct messages"
        description="Notify me when someone sends me a direct message."
        checked={notifyOnDM}
        onCheckedChange={setNotifyOnDM}
      />

      <ToggleRow
        id="notify-mention"
        label="Mentions"
        description="Notify me when someone @mentions me."
        checked={notifyOnMention}
        onCheckedChange={setNotifyOnMention}
      />

      <Separator />

      <ToggleRow
        id="autostart"
        label="Start Cordis when I sign in"
        description="Launch the app in the background at system start-up."
        checked={autostart}
        onCheckedChange={(value) => {
          setAutostart(value);
          void applyAutostart(value);
        }}
      />
    </div>
  );
}

/** Registers the app with the OS start-up list when packaged. */
async function applyAutostart(enabled: boolean): Promise<void> {
  const { setAutostart } = await import("@/lib/tauri/desktop");
  try {
    await setAutostart(enabled);
  } catch (error) {
    console.error("[settings] autostart failed", error);
  }
}