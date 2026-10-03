import { useEffect, type React } from "react";
import { AppShell } from "@/components/layout/AppShell";
import { useSocketBridge } from "@/hooks/useHotkeys";
import { useAuthStore } from "@/stores/auth.store";
import { useSettingsStore } from "@/stores/settings.store";
import { socket, resolveWsUrl } from "@/lib/ws/client";
import { onNotifyRequest, notifyMention } from "@/lib/tauri/desktop";

/**
 * The authenticated landing screen. Owns the socket lifecycle, the single
 * event bridge and the native notification channel.
 */
export function AppPage(): React.JSX.Element {
  const accessToken = useAuthStore((state) => state.accessToken);
  const status = useSettingsStore((state) => state.status);

  useSocketBridge();

  // Connect whenever a token appears; the bridge is registered by a hook above.
  useEffect(() => {
    if (!accessToken) return;

    const url = resolveWsUrl();
    if (url) socket.connect(accessToken, url);

    return () => {
      socket.close();
    };
  }, [accessToken]);

  // Announce presence on connect and whenever the user changes their status.
  useEffect(() => {
    const announce = () => {
      if (socket.getState() !== "open") return;
      // The backend rejects "offline"; an explicit sign-out is handled by logout.
      socket.send("presence.update", {
        status: status === "offline" ? "invisible" : status,
      });
    };

    announce();
    return socket.onStatus(announce);
  }, [status]);

  useEffect(() => {
    let dispose: (() => void) | undefined;
    void onNotifyRequest((request) => {
      notifyMention(request);
    }).then((unlisten) => {
      dispose = unlisten;
    });
    return () => dispose?.();
  }, []);

  return <AppShell />;
}