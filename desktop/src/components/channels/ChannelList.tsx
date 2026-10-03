import { useState, type React } from "react";
import { ChevronDown, Plus, Volume2, VolumeX } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ChannelIcon } from "@/components/layout/ChannelSidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { Channel } from "@/types";

export interface ChannelListProps {
  channels: Channel[];
  activeChannelId: string | null;
  onSelectChannel: (channelId: string) => void;
  onJoinVoice: (channel: Channel) => void;
  connectedVoiceChannelId: string | null;
  onCreateChannel: () => void;
  canCreate: boolean;
}

/**
 * Channels grouped under their category, with loose channels in a trailing
 * group, mirroring Discord's sidebar structure.
 */
export function ChannelList({
  channels,
  activeChannelId,
  onSelectChannel,
  onJoinVoice,
  connectedVoiceChannelId,
  onCreateChannel,
  canCreate,
}: ChannelListProps): React.JSX.Element {
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  const categories = channels.filter((channel) => channel.type === "category");
  const loose = channels.filter((channel) => channel.parentId === null && channel.type !== "category");

  return (
    <div className="space-y-4">
      {categories.map((category) => {
        const children = channels.filter((channel) => channel.parentId === category.id);
        if (children.length === 0) return null;
        const isCollapsed = collapsed[category.id] ?? false;

        return (
          <section key={category.id}>
            <button
              type="button"
              onClick={() =>
                setCollapsed((state) => ({ ...state, [category.id]: !isCollapsed }))
              }
              aria-expanded={!isCollapsed}
              className="flex w-full items-center gap-0.5 px-1 py-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground hover:text-foreground"
            >
              <ChevronDown
                className={cn("h-3 w-3 transition-transform", isCollapsed && "-rotate-90")}
              />
              <span className="truncate">{category.name}</span>
            </button>

            {!isCollapsed ? (
              <div className="space-y-px">
                {children.map((channel) => (
                  <ChannelRow
                    key={channel.id}
                    channel={channel}
                    active={channel.id === activeChannelId}
                    muted={channel.id === connectedVoiceChannelId}
                    onSelect={() => onSelectChannel(channel.id)}
                    onJoinVoice={() => onJoinVoice(channel)}
                  />
                ))}
              </div>
            ) : null}
          </section>
        );
      })}

      <section>
        {canCreate ? (
          <div className="flex items-center justify-between px-1 py-1">
            <span className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
              Channels
            </span>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label="Create channel"
                  className="h-5 w-5 text-muted-foreground hover:text-foreground"
                  onClick={onCreateChannel}
                >
                  <Plus className="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="right">Create channel</TooltipContent>
            </Tooltip>
          </div>
        ) : null}

        <div className="space-y-px">
          {loose.map((channel) => (
            <ChannelRow
                channel={channel}
                active={channel.id === activeChannelId}
                muted={channel.id === connectedVoiceChannelId}
                onSelect={() => onSelectChannel(channel.id)}
                onJoinVoice={() => onJoinVoice(channel)}
              />
          ))}
        </div>
      </section>
    </div>
  );
}

interface ChannelRowProps {
  channel: Channel;
  active: boolean;
  muted: boolean;
  onSelect: () => void;
  onJoinVoice: () => void;
}

function ChannelRow({
  channel,
  active,
  muted,
  onSelect,
  onJoinVoice,
}: ChannelRowProps): React.JSX.Element {
  const isVoice = channel.type === "voice" || channel.type === "video";

  const button = (
    <button
      type="button"
      onClick={onSelect}
      aria-current={active}
      className={cn(
        "group flex w-full items-center gap-1.5 rounded px-2 py-1.5 text-left text-sm transition-colors",
        active
          ? "bg-white/10 font-semibold text-foreground"
          : "text-muted-foreground hover:bg-white/5 hover:text-foreground",
      )}
    >
      <ChannelIcon channel={channel} className="opacity-70" />
      <span className="truncate">{channel.name}</span>
      {channel.topic ? (
        <span className="ml-auto truncate text-[10px] text-muted-foreground/70">
          {channel.topic}
        </span>
      ) : null}
    </button>
  );

  return (
    <div className="group/row relative">
      {button}
      {isVoice ? (
        <button
          type="button"
          onClick={onJoinVoice}
          aria-label={muted ? `Disconnect from ${channel.name}` : `Join ${channel.name}`}
          className={cn(
            "absolute right-1 top-1/2 -translate-y-1/2 rounded p-1 opacity-0 transition-opacity",
            "hover:bg-black/30 group-hover/row:opacity-100",
            muted ? "text-cordis-dnd opacity-100" : "text-muted-foreground",
          )}
        >
          {muted ? <VolumeX className="h-3.5 w-3.5" /> : <Volume2 className="h-3.5 w-3.5" />}
        </button>
      ) : null}
    </div>
  );
}
