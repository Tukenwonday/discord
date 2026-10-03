import type * as React from "react";

import { MemberItem } from "@/components/members/MemberItem";
import type { MemberGroup } from "@/components/layout/MembersPanel";

export interface MemberListProps {
  groups: MemberGroup[];
  customStatusOf: (userId: string | null) => string;
}

/** Renders the role and presence groups produced by the members panel. */
export function MemberList({ groups, customStatusOf }: MemberListProps): React.JSX.Element {
  if (groups.length === 0) {
    return (
      <p className="px-2 py-4 text-center text-xs text-muted-foreground">
        No members to show
      </p>
    );
  }

  return (
    <div className="space-y-4">
      {groups.map((group) => (
        <section key={group.label}>
          <h3 className="px-2 pb-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
            {group.label}
          </h3>
          <div className="space-y-px">
            {group.members.map((member) => (
              <MemberItem
                key={member.id}
                member={member}
                customStatus={customStatusOf(member.user.id)}
              />
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}