import { useState, type React } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Titlebar } from "./Titlebar";
import { ServerRail } from "./ServerRail";
import { ChannelSidebar } from "./ChannelSidebar";
import { ChatView } from "./ChatView";
import { MembersPanel } from "./MembersPanel";
import { DmView } from "@/components/dm/DmView";
import { SettingsDialog } from "@/components/settings/SettingsDialog";
import { useServers } from "@/hooks/useServers";
import { useHotkeys } from "@/hooks/useHotkeys";

/**
 * The authenticated application frame. Members and the chat column swap when the
 * route moves to a DM, keeping the left navigation stable.
 */
export function AppShell(): React.JSX.Element {
  const navigate = useNavigate();
  const params = useParams<{ channelId?: string; dmId?: string }>();
  const { servers, activeServerId, selectServer, detail, activeChannel } = useServers();

  const [settingsOpen, setSettingsOpen] = useState(false);
  const [membersVisible, setMembersVisible] = useState(true);

  useHotkeys({
    onToggleMembers: () => setMembersVisible((value) => !value),
    onToggleSettings: () => setSettingsOpen((value) => !value),
    onSearch: () => setSettingsOpen(true),
    onEscape: () => setSettingsOpen(false),
    onQuickSwitcher: () => navigate(`/channels/${params.channelId ?? ""}`),
  });

  const isDm = Boolean(params.dmId);

  return (
    <div className="flex h-full w-full flex-col overflow-hidden bg-background">
      <Titlebar
        title={detail?.server.name ?? "Cordis"}
        subtitle={activeChannel?.name}
      />

      <div className="flex min-h-0 flex-1">
        <ServerRail
          servers={servers}
          activeServerId={activeServerId}
          onSelectServer={selectServer}
          onOpenSettings={() => setSettingsOpen(true)}
          onOpenDm={(dmId) => navigate(`/dm/${dmId}`)}
        />

        <ChannelSidebar
          onOpenSettings={() => setSettingsOpen(true)}
          onOpenDm={(dmId) => navigate(`/dm/${dmId}`)}
        />

        {isDm ? <DmView dmId={params.dmId ?? null} /> : <ChatView channel={activeChannel} />}

        <MembersPanel visible={membersVisible} />
      </div>

      <SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
    </div>
  );
}