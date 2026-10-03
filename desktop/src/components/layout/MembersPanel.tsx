import { useMemo, type React } from "react";
import { MemberList } from "@/components/members/MemberList";
import { useServers } from "@/hooks/useServers";
import { usePresence } from "@/hooks/usePresence";
import type { Member, Role, UserStatus } from "@/types";

export interface MembersPanelProps {
  visible: boolean;
}

/** Right-hand column grouping members by hoisted role, then online state. */
export function MembersPanel({ visible }: MembersPanelProps): React.JSX.Element | null {
  const { members, roles } = useServers();
  const { statusOf, customStatusOf } = usePresence();

  const grouped = useMemo(
    () => groupMembers(members, roles, statusOf),
    [members, roles, statusOf],
  );

  if (!visible) return null;

  return (
    <aside
      className="cordis-scroll hidden w-60 shrink-0 overflow-y-auto px-2 py-4 lg:block"
      style={{ backgroundColor: "hsl(var(--cordis-panel))" }}
      aria-label="Members"
    >
      <MemberList groups={grouped} customStatusOf={customStatusOf} />
    </aside>
  );
}

export interface MemberGroup {
  label: string;
  color?: string;
  members: Member[];
}

const ONLINE_ORDER: UserStatus[] = ["online", "idle", "dnd"];

function isOnline(status: UserStatus): boolean {
  return status !== "offline" && status !== "invisible";
}

function sortByPresence(members: Member[], statusOf: (id: string | null) => UserStatus): Member[] {
  return [...members].sort((a, b) => {
    const rankA = ONLINE_ORDER.indexOf(statusOf(a.user.id));
    const rankB = ONLINE_ORDER.indexOf(statusOf(b.user.id));
    const normalizedA = rankA === -1 ? ONLINE_ORDER.length : rankA;
    const normalizedB = rankB === -1 ? ONLINE_ORDER.length : rankB;
    if (normalizedA !== normalizedB) return normalizedA - normalizedB;

    const nameA = a.nickname || a.user.displayName || a.user.username;
    const nameB = b.nickname || b.user.displayName || b.user.username;
    return nameA.localeCompare(nameB);
  });
}

/**
 * Hoisted roles first (highest position first), then Online, then Offline.
 * @everyone is hidden because it carries no useful information.
 */
export function groupMembers(
  members: Member[],
  roles: Role[],
  statusOf: (id: string | null) => UserStatus,
): MemberGroup[] {
  const byId = new Map(roles.map((role) => [role.id, role]));
  const hoisted = new Map<string, Member[]>();

  for (const member of members) {
    const memberRoles = member.roleIds
      .map((id) => byId.get(id))
      .filter((role): role is Role => Boolean(role) && role.hoist && role.name !== "@everyone");

    if (memberRoles.length === 0) continue;

    const top = memberRoles.sort((a, b) => b.position - a.position)[0];
    const bucket = hoisted.get(top.id) ?? [];
    bucket.push(member);
    hoisted.set(top.id, bucket);
  }

  const groups: MemberGroup[] = [];
  const groupedIds = new Set<string>();

  for (const role of [...roles].sort((a, b) => b.position - a.position)) {
    const bucket = hoisted.get(role.id);
    if (!bucket || bucket.length === 0) continue;
    groups.push({
      label: role.name,
      color: role.color,
      members: sortByPresence(bucket, statusOf),
    });
    for (const member of bucket) groupedIds.add(member.id);
  }

  const ungrouped = members.filter((member) => !groupedIds.has(member.id));
  const online = ungrouped.filter((member) => isOnline(statusOf(member.user.id)));
  const offline = ungrouped.filter((member) => !isOnline(statusOf(member.user.id)));

  if (online.length > 0) {
    groups.push({ label: `Online — ${online.length}`, members: sortByPresence(online, statusOf) });
  }
  if (offline.length > 0) {
    groups.push({ label: `Offline — ${offline.length}`, members: offline });
  }

  return groups;
}
