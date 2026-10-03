import { useMemo, type React } from "react";
import { pluralize } from "@/lib/utils";
import type { User } from "@/types";

export interface TypingIndicatorProps {
  users: User[];
}

/** "Alex is typing…" with the three-dot animation; hidden when nobody is. */
export function TypingIndicator({ users }: TypingIndicatorProps): React.JSX.Element | null {
  const label = useMemo(() => describe(users), [users]);

  return (
    <div className="h-6 shrink-0 px-4 text-[11px] text-muted-foreground">
      {label ? (
        <span className="flex items-center gap-1.5">
          <span className="flex gap-0.5" aria-hidden="true">
            <span className="h-1.5 w-1.5 animate-typing-dot rounded-full bg-muted-foreground" />
            <span
              className="h-1.5 w-1.5 animate-typing-dot rounded-full bg-muted-foreground"
              style={{ animationDelay: "150ms" }}
            />
            <span
              className="h-1.5 w-1.5 animate-typing-dot rounded-full bg-muted-foreground"
              style={{ animationDelay: "300ms" }}
            />
          </span>
          <span className="truncate">{label}</span>
        </span>
      ) : null}
    </div>
  );
}

function describe(users: User[]): string {
  const names = users
    .map((user) => user.displayName || user.username)
    .filter((name) => name.length > 0);

  if (names.length === 0) return "";
  if (names.length === 1) return `${names[0]} is typing…`;
  if (names.length === 2) return `${names[0]} and ${names[1]} are typing…`;
  return `${names[0]}, ${names[1]} and ${pluralize(names.length - 2, "other")} are typing…`;
}