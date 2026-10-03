import type * as React from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { usePresence } from "@/hooks/usePresence";
import { avatarHue, cn, initials, presenceLabel } from "@/lib/utils";
import type { Member } from "@/types";

export interface MemberItemProps {
  member: Member;
  customStatus: string;
}

/** One row in the member list: avatar, name and the presence dot. */
export function MemberItem({ member, customStatus }: MemberItemProps): React.JSX.Element {
  const { statusOf } = usePresence();
  const status = statusOf(member.user.id);
  const name = member.nickname || member.user.displayName || member.user.username;
  const isOffline = status === "offline" || status === "invisible";

  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded px-2 py-1 transition hover:bg-white/5",
        isOffline && "opacity-40",
      )}
      title={presenceLabel(status)}
    >
      <div className="relative shrink-0">
        <Avatar className="h-8 w-8">
          {member.user.avatarUrl ? <AvatarImage src={member.user.avatarUrl} alt="" /> : null}
          <AvatarFallback
            style={{ backgroundColor: `hsl(${avatarHue(member.user.id)} 45% 35%)` }}
            className="text-[10px] font-semibold text-white"
          >
            {initials(name)}
          </AvatarFallback>
        </Avatar>
        <span
          className={cn(
            "absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-cordis-panel",
            status === "online" && "bg-cordis-online",
            status === "idle" && "bg-cordis-idle",
            status === "dnd" && "bg-cordis-dnd",
            (status === "offline" || status === "invisible") && "bg-cordis-offline",
          )}
        />
      </div>

      <span className="min-w-0 flex-1 truncate text-sm text-foreground/90">{name}</span>

      {customStatus ? (
        <span className="max-w-[6rem] truncate text-[10px] text-muted-foreground">
          {customStatus}
        </span>
      ) : null}
    </div>
  );
}