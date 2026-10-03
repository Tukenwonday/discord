import { useCallback } from "react";
import { selectStatus, usePresenceStore } from "@/stores/presence.store";
import { useAuthStore } from "@/stores/auth.store";
import type { User, UserStatus } from "@/types";

export interface UsePresenceResult {
  statusOf: (userId: string | null) => UserStatus;
  customStatusOf: (userId: string | null) => string;
  typingUsers: (channelId: string | null) => User[];
  applyPresence: (userId: string, status: UserStatus, customStatus: string) => void;
}

/**
 * Reads presence from the store. Live updates arrive over the socket, which
 * writes into the same store via `presence.updated`.
 */
export function usePresence(): UsePresenceResult {
  const statusByUserId = usePresenceStore((state) => state.statusByUserId);
  const customStatusByUserId = usePresenceStore((state) => state.customStatusByUserId);
  const typingByChannelId = usePresenceStore((state) => state.typingByChannelId);
  const applyPresenceAction = usePresenceStore((state) => state.applyPresence);
  const user = useAuthStore((state) => state.user);

  const statusOf = useCallback(
    (userId: string | null): UserStatus => {
      if (!userId) return "offline";
      if (userId === user?.id) return user.status;
      return statusByUserId[userId] ?? "offline";
    },
    [statusByUserId, user],
  );

  const customStatusOf = useCallback(
    (userId: string | null): string => {
      if (!userId) return "";
      if (userId === user?.id) return user.customStatus;
      return customStatusByUserId[userId] ?? "";
    },
    [customStatusByUserId, user],
  );

  const typingUsers = useCallback(
    (channelId: string | null): User[] => {
      const bucket = channelId ? typingByChannelId[channelId] : undefined;
      if (!bucket) return [];
      return Object.values(bucket)
        .filter((entry) => entry.expiresAt > Date.now() && entry.user.id !== user?.id)
        .map((entry) => entry.user);
    },
    [typingByChannelId, user],
  );

  return {
    statusOf,
    customStatusOf,
    typingUsers,
    applyPresence: applyPresenceAction,
  };
}

/** Imperative helper for non-component callers such as the socket bridge. */
export function presenceActions() {
  const state = usePresenceStore.getState();
  return {
    applyPresence: (userId: string, status: UserStatus, customStatus: string) =>
      state.applyPresence(userId, status, customStatus),
    statusOf: (userId: string | null) =>
      selectStatus(usePresenceStore.getState(), userId),
  };
}