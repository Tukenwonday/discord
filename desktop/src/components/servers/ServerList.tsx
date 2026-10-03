import type * as React from "react";

import { MessageSquarePlus } from "lucide-react";
import { ServerIconButton } from "@/components/layout/ServerRail";
import { DmIconButton } from "@/components/dm/DmList";
import { cn } from "@/lib/utils";
import type { Server } from "@/types";

export interface ServerListProps {
  servers: Server[];
  activeServerId: string | null;
  onSelectServer: (serverId: string) => void;
  onOpenDm: (dmId: string) => void;
}

/** Home entry plus the scrollable list of servers the user belongs to. */
export function ServerList({
  servers,
  activeServerId,
  onSelectServer,
  onOpenDm,
}: ServerListProps): React.JSX.Element {
  return (
    <div className="flex w-full flex-col items-center gap-2">
      <DmIconButton onOpenDm={onOpenDm} active={false} />

      {servers.map((server) => (
        <ServerIconButton
          key={server.id}
          server={server}
          active={server.id === activeServerId}
          onClick={() => onSelectServer(server.id)}
        />
      ))}

      {servers.length === 0 ? (
        <p
          className={cn(
            "mt-1 whitespace-nowrap px-1 text-center text-[10px] leading-tight",
            "text-muted-foreground",
          )}
        >
          <MessageSquarePlus className="mx-auto mb-1 h-4 w-4" />
          No servers
        </p>
      ) : null}
    </div>
  );
}