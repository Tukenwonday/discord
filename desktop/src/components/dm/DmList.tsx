import { useState, type React } from "react";
import { useQuery } from "@tanstack/react-query";
import { MessageCircle, Plus } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { NewDmDialog } from "@/components/dm/NewDmDialog";
import { listDms } from "@/lib/api/dms";
import { useAuth } from "@/hooks/useAuth";
import { avatarHue, cn, initials, presenceClass } from "@/lib/utils";

export interface DmListProps {
  onOpenDm: (dmId: string) => void;
}

/** The "home" button in the server rail; also the new-DM launcher. */
export function DmIconButton({
  onOpenDm,
  active,
}: {
  onOpenDm: (dmId: string) => void;
  active: boolean;
}): React.JSX.Element {
  const [dialogOpen, setDialogOpen] = useState(false);

  return (
    <>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            onClick={() => setDialogOpen(true)}
            aria-label="Direct messages"
            className={cn(
              "relative flex h-12 w-12 items-center justify-center overflow-hidden transition-all duration-200",
              active ? "rounded-2xl" : "rounded-3xl hover:rounded-2xl",
            )}
            style={{ backgroundColor: "hsl(var(--cordis-brand))" }}
          >
            <MessageCircle className="h-6 w-6 text-white" />
            <span
              className={cn(
                "absolute -left-2 top-1/2 h-8 w-1 -translate-y-1/2 rounded-r bg-foreground transition-all",
                active ? "opacity-100" : "opacity-0",
              )}
            />
          </button>
        </TooltipTrigger>
        <TooltipContent side="right">Direct messages</TooltipContent>
      </Tooltip>

      <NewDmDialog open={dialogOpen} onOpenChange={setDialogOpen} onOpenDm={onOpenDm} />
    </>
  );
}

/** Conversation list shown under the channel list in the sidebar. */
export function DmList({ onOpenDm }: DmListProps): React.JSX.Element {
  const { user } = useAuth();
  const [newDmOpen, setNewDmOpen] = useState(false);

  const query = useQuery({
    queryKey: ["dms"],
    queryFn: () => listDms(),
    staleTime: 30_000,
  });

  const conversations = query.data?.items ?? [];

  return (
    <section>
      <div className="flex items-center justify-between px-1 py-1">
        <span className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
          Direct messages
        </span>
        <button
          type="button"
          aria-label="Start a conversation"
          onClick={() => setNewDmOpen(true)}
          className="text-muted-foreground transition hover:text-foreground"
        >
          <Plus className="h-3.5 w-3.5" />
        </button>
      </div>

      <div className="space-y-px">
        {conversations.map((dm) => {
          // The other participant is the one that is not the signed-in user.
          const other = dm.recipients.find((recipient) => recipient.id !== user?.id);
          const name = other?.displayName || other?.username || "Unknown";
          const preview = dm.lastMessage?.content ?? "No messages yet";

          return (
            <button
              key={dm.id}
              type="button"
              onClick={() => onOpenDm(dm.id)}
              className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm text-muted-foreground transition hover:bg-white/5 hover:text-foreground"
            >
              <div className="relative shrink-0">
                <Avatar className="h-6 w-6">
                  {other?.avatarUrl ? <AvatarImage src={other.avatarUrl} alt="" /> : null}
                  <AvatarFallback
                    style={{ backgroundColor: `hsl(${avatarHue(other?.id ?? dm.id)} 45% 35%)` }}
                    className="text-[9px] font-semibold text-white"
                  >
                    {initials(name)}
                  </AvatarFallback>
                </Avatar>
                {other ? (
                  <span
                    className={cn(
                      "absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border-2 border-cordis-sidebar",
                      presenceClass(other.status),
                    )}
                  />
                ) : null}
              </div>

              <span className="min-w-0 flex-1">
                <span className="block truncate text-xs font-medium text-foreground/90">
                  {name}
                </span>
                <span className="block truncate text-[10px] text-muted-foreground">
                  {preview}
                </span>
              </span>
            </button>
          );
        })}

        {conversations.length === 0 && !query.isLoading ? (
          <p className="px-2 py-1 text-[11px] text-muted-foreground">
            No conversations yet
          </p>
        ) : null}
      </div>

      <NewDmDialog open={newDmOpen} onOpenChange={setNewDmOpen} onOpenDm={onOpenDm} />
    </section>
  );
}