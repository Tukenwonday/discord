import { useState, type React } from "react";
import { Plus, Compass } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ServerList } from "@/components/servers/ServerList";
import { CreateServerDialog } from "@/components/servers/CreateServerDialog";
import { useAuth } from "@/hooks/useAuth";
import { avatarHue, cn, initials, presenceClass } from "@/lib/utils";
import type { Server } from "@/types";

export interface ServerRailProps {
  servers: Server[];
  activeServerId: string | null;
  onSelectServer: (serverId: string) => void;
  onOpenSettings: () => void;
  onOpenDm: (dmId: string) => void;
}

/** The leftmost icon column: home, one icon per server, plus add. */
export function ServerRail({
  servers,
  activeServerId,
  onSelectServer,
  onOpenSettings,
  onOpenDm,
}: ServerRailProps): React.JSX.Element {
  const { user } = useAuth();
  const [createOpen, setCreateOpen] = useState(false);

  return (
    <nav
      className="cordis-scroll flex h-full w-[72px] shrink-0 flex-col items-center gap-2 overflow-y-auto py-3"
      style={{ backgroundColor: "hsl(var(--cordis-rail))" }}
      aria-label="Servers"
    >
      <ServerList
        servers={servers}
        activeServerId={activeServerId}
        onSelectServer={onSelectServer}
        onOpenDm={onOpenDm}
      />

      <RailSeparator />

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            size="icon"
            aria-label="Create a server"
            onClick={() => setCreateOpen(true)}
            className="h-12 w-12 rounded-full bg-cordis-brand text-white transition hover:rounded-2xl hover:bg-cordis-brand-hover"
          >
            <Plus className="h-5 w-5" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="right">Create a server</TooltipContent>
      </Tooltip>

      <RailSeparator />

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Explore servers"
            className="h-12 w-12 rounded-full bg-secondary text-secondary-foreground transition hover:rounded-2xl hover:bg-cordis-brand hover:text-white"
          >
            <Compass className="h-5 w-5" />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="right">Explore servers</TooltipContent>
      </Tooltip>

      {/* The account avatar doubles as the settings entry point. */}
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            onClick={onOpenSettings}
            aria-label="Open settings"
            className="relative mt-auto h-12 w-12 overflow-hidden rounded-full ring-2 ring-transparent transition hover:ring-cordis-brand"
          >
            <Avatar className="h-12 w-12">
              {user?.avatarUrl ? <AvatarImage src={user.avatarUrl} alt="" /> : null}
              <AvatarFallback
                style={{ backgroundColor: `hsl(${avatarHue(user?.id ?? "me")} 45% 35%)` }}
                className="text-sm font-semibold text-white"
              >
                {initials(user?.displayName ?? "?")}
              </AvatarFallback>
            </Avatar>
            {user ? (
              <span
                className={cn(
                  "absolute bottom-0 right-0 h-3.5 w-3.5 rounded-full border-2 border-cordis-rail",
                  presenceClass(user.status),
                )}
              />
            ) : null}
          </button>
        </TooltipTrigger>
        <TooltipContent side="right">Settings</TooltipContent>
      </Tooltip>

      <CreateServerDialog open={createOpen} onOpenChange={setCreateOpen} />
    </nav>
  );
}

/** A server icon, rendered from its image or two-letter initials. */
export function ServerIconButton({
  server,
  active,
  onClick,
}: {
  server: Server;
  active: boolean;
  onClick: () => void;
}): React.JSX.Element {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={onClick}
          aria-label={server.name}
          aria-current={active}
          className={cn(
            "group relative flex h-12 w-12 items-center justify-center overflow-hidden",
            "transition-all duration-200",
            active ? "rounded-2xl" : "rounded-3xl hover:rounded-2xl",
          )}
          style={{ backgroundColor: `hsl(${avatarHue(server.id)} 45% 35%)` }}
        >
          {server.iconUrl ? (
            <img src={server.iconUrl} alt="" className="h-full w-full object-cover" />
          ) : (
            <span className="text-sm font-semibold text-white">{initials(server.name)}</span>
          )}

          {/* The active pill echoes Discord's unread/selected indicator. */}
          <span
            className={cn(
              "absolute -left-2 top-1/2 h-8 w-1 -translate-y-1/2 rounded-r bg-foreground transition-all",
              active ? "opacity-100" : "opacity-0 group-hover:opacity-60",
            )}
          />
        </button>
      </TooltipTrigger>
      <TooltipContent side="right">{server.name}</TooltipContent>
    </Tooltip>
  );
}

function RailSeparator(): React.JSX.Element {
  return <div className="h-0.5 w-8 rounded-full bg-white/10" aria-hidden="true" />;
}