import { useState, type React } from "react";
import { Hash, Users, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ConfirmDialog } from "@/components/common/ConfirmDialog";
import { useServersStore } from "@/stores/servers.store";
import { deleteChannel } from "@/lib/api/channels";
import { ChannelIcon } from "@/components/layout/ChannelSidebar";
import type { Channel } from "@/types";

export interface ChannelHeaderProps {
  channel: Channel;
}

/** Channel name, topic and the per-channel admin actions. */
export function ChannelHeader({ channel }: ChannelHeaderProps): React.JSX.Element {
  const activeServerId = useServersStore((state) => state.activeServerId);
  const loadServer = useServersStore((state) => state.loadServer);
  const [confirmOpen, setConfirmOpen] = useState(false);

  async function onDelete(): Promise<void> {
    setConfirmOpen(false);
    await deleteChannel(channel.id);
    if (activeServerId) await loadServer(activeServerId);
  }

  return (
    <header className="flex h-12 shrink-0 items-center justify-between border-b border-black/20 px-4">
      <div className="flex min-w-0 items-center gap-2">
        <ChannelIcon channel={channel} className="text-muted-foreground" />
        <h1 className="truncate text-sm font-bold">{channel.name}</h1>
        {channel.topic ? (
          <>
            <span className="h-4 w-px shrink-0 bg-border" />
            <p className="truncate text-xs text-muted-foreground">{channel.topic}</p>
          </>
        ) : null}
      </div>

      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="icon"
          aria-label="Invite people"
          className="h-8 w-8 text-muted-foreground hover:text-foreground"
        >
          <Users className="h-4 w-4" />
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Channel actions"
              className="h-8 w-8 text-muted-foreground hover:text-foreground"
            >
              <Hash className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-52">
            <DropdownMenuItem onSelect={() => setConfirmOpen(true)} className="text-destructive">
              <Trash2 className="h-4 w-4" />
              Delete channel
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem disabled className="text-xs text-muted-foreground">
              Channel settings are managed by server owners
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={`Delete #${channel.name}?`}
        description="Messages in this channel are removed for everyone. This cannot be undone."
        confirmLabel="Delete channel"
        destructive
        onConfirm={() => void onDelete()}
      />
    </header>
  );
}

/** Small helper so both the flag and its setter stay in sync. */
function useConfirmState(): [boolean, (open: boolean) => void] {
  const [open, setOpen] = useState(false);
  return [open, setOpen];
}
