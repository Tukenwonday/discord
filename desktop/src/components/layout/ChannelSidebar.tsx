import { useState, type React } from "react";
import { Hash, Headphones, Settings, Video, ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChannelList } from "@/components/channels/ChannelList";
import { CreateChannelDialog } from "@/components/channels/CreateChannelDialog";
import { DmList } from "@/components/dm/DmList";
import { VoicePanel } from "@/components/voice/VoicePanel";
import { useServers } from "@/hooks/useServers";
import { useVoice } from "@/hooks/useVoice";
import { cn } from "@/lib/utils";
import type { Channel } from "@/types";

export interface ChannelSidebarProps {
  onOpenSettings: () => void;
  onOpenDm: (dmId: string) => void;
}

/** Channel and DM navigation for the active server. */
export function ChannelSidebar({
  onOpenSettings,
  onOpenDm,
}: ChannelSidebarProps): React.JSX.Element {
  const { detail, channels, activeChannelId, selectChannel, activeServerId } = useServers();
  const voice = useVoice();
  const [filter, setFilter] = useState("");
  const [createOpen, setCreateOpen] = useState(false);

  const visible = filter.trim()
    ? channels.filter((channel) =>
        channel.name.toLowerCase().includes(filter.trim().toLowerCase()),
      )
    : channels;

  return (
    <aside
      className="flex h-full w-60 shrink-0 flex-col"
      style={{ backgroundColor: "hsl(var(--cordis-sidebar))" }}
      aria-label="Channels"
    >
      <div className="flex h-12 shrink-0 items-center justify-between border-b border-black/20 px-4">
        <h2 className="truncate text-sm font-bold text-foreground">
          {detail?.server.name ?? "Cordis"}
        </h2>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Server settings"
              className="h-7 w-7 text-muted-foreground hover:text-foreground"
              onClick={onOpenSettings}
            >
              <Settings className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">Settings</TooltipContent>
        </Tooltip>
      </div>

      <div className="flex items-center gap-1 px-2 pb-2 pt-3">
        <Button
          variant="secondary"
          size="sm"
          className="flex-1 justify-between bg-black/20 text-xs text-muted-foreground hover:bg-black/40 hover:text-foreground"
          onClick={() => setFilter(filter ? "" : " ")}
        >
          <span>{filter ? filter.trim() || "Jump to…" : "Jump to…"}</span>
          <ChevronDown className="h-3 w-3" />
        </Button>
      </div>

      {filter.trim() ? (
        <div className="px-2 pb-2">
          <Input
            autoFocus
            value={filter}
            placeholder="Filter channels"
            onChange={(event) => setFilter(event.target.value)}
            className="h-7 bg-black/20 text-xs"
          />
        </div>
      ) : null}

      <div className="cordis-scroll flex-1 overflow-y-auto px-2 pb-4">
        <ChannelList
          channels={visible}
          activeChannelId={activeChannelId}
          onSelectChannel={selectChannel}
          onJoinVoice={(channel) => void voice.join(channel)}
          connectedVoiceChannelId={voice.channelId}
          onCreateChannel={() => setCreateOpen(true)}
          canCreate={Boolean(activeServerId)}
        />

        <div className="mt-4">
          <DmList onOpenDm={onOpenDm} />
        </div>
      </div>

      <VoicePanel />

      <CreateChannelDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        serverId={activeServerId ?? ""}
      />
    </aside>
  );
}

/** Icon for a channel row, chosen by the channel's type. */
export function ChannelIcon({
  channel,
  className,
}: {
  channel: Channel;
  className?: string;
}): React.JSX.Element {
  const shared = cn("h-4 w-4 shrink-0", className);
  switch (channel.type) {
    case "voice":
      return <Headphones className={shared} />;
    case "video":
      return <Video className={shared} />;
    default:
      return <Hash className={shared} />;
  }
}
