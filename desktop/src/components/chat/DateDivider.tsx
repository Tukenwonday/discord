import type * as React from "react";

import { formatDayLabel } from "@/lib/utils";

export interface DateDividerProps {
  iso: string;
}

/** Horizontal rule with the day label, used between days in scrollback. */
export function DateDivider({ iso }: DateDividerProps): React.JSX.Element {
  return (
    <div className="my-4 flex items-center gap-3 px-2">
      <div className="h-px flex-1 bg-black/30" />
      <span className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        {formatDayLabel(iso)}
      </span>
      <div className="h-px flex-1 bg-black/30" />
    </div>
  );
}