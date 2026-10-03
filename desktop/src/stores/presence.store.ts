import { create } from "zustand";
import type { User, UserStatus } from "@/types";

/** A typing entry disappears on its own so a dropped `stop` cannot stick. */
export interface TypingEntry {
  user: User;
  expiresAt: number;
}

const TYPING_TTL = 8_000;

export interface PresenceState {
  statusByUserId: Record<string, UserStatus>;
  customStatusByUserId: Record<string, string>;
  /** channelId -> userId -> entry */
  typingByChannelId: Record<string, Record<string, TypingEntry>>;
  applyPresence: (userId: string, status: UserStatus, customStatus: string) => void;
  startTyping: (channelId: string, user: User) => void;
  stopTyping: (channelId: string, userId: string) => void;
  stopAllTyping: (channelId: string) => void;
  clear: () => void;
}

/** Timers are kept outside the store: they are not reactive state. */
const expiryTimers = new Map<string, number>();

function timerKey(channelId: string, userId: string): string {
  return `${channelId}:${userId}`;
}

export const usePresenceStore = create<PresenceState>((set, get) => ({
  statusByUserId: {},
  customStatusByUserId: {},
  typingByChannelId: {},

  applyPresence: (userId, status, customStatus) =>
    set((state) => ({
      statusByUserId: { ...state.statusByUserId, [userId]: status },
      customStatusByUserId: { ...state.customStatusByUserId, [userId]: customStatus },
    })),

  startTyping: (channelId, user) => {
    const key = timerKey(channelId, user.id);
    if (expiryTimers.has(key)) window.clearTimeout(expiryTimers.get(key));

    set((state) => ({
      typingByChannelId: {
        ...state.typingByChannelId,
        [channelId]: {
          ...(state.typingByChannelId[channelId] ?? {}),
          [user.id]: { user, expiresAt: Date.now() + TYPING_TTL },
        },
      },
    }));

    // Self-healing expiry: a missed `typing.stopped` frame is common.
    expiryTimers.set(
      key,
      window.setTimeout(() => {
        expiryTimers.delete(key);
        get().stopTyping(channelId, user.id);
      }, TYPING_TTL),
    );
  },

  stopTyping: (channelId, userId) => {
    const key = timerKey(channelId, userId);
    const timer = expiryTimers.get(key);
    if (timer !== undefined) {
      window.clearTimeout(timer);
      expiryTimers.delete(key);
    }

    set((state) => {
      const bucket = state.typingByChannelId[channelId];
      if (!bucket || !bucket[userId]) return state;
      const nextBucket = { ...bucket };
      delete nextBucket[userId];

      const next = { ...state.typingByChannelId };
      if (Object.keys(nextBucket).length === 0) {
        delete next[channelId];
      } else {
        next[channelId] = nextBucket;
      }
      return { typingByChannelId: next };
    });
  },

  stopAllTyping: (channelId) => {
    set((state) => {
      const bucket = state.typingByChannelId[channelId];
      if (!bucket) return state;
      for (const userId of Object.keys(bucket)) {
        const timer = expiryTimers.get(timerKey(channelId, userId));
        if (timer !== undefined) {
          window.clearTimeout(timer);
          expiryTimers.delete(timerKey(channelId, userId));
        }
      }
      const next = { ...state.typingByChannelId };
      delete next[channelId];
      return { typingByChannelId: next };
    });
  },

  clear: () => {
    for (const timer of expiryTimers.values()) window.clearTimeout(timer);
    expiryTimers.clear();
    set({ statusByUserId: {}, customStatusByUserId: {}, typingByChannelId: {} });
  },
}));

/** Live typists in a channel, excluding the current user. */
export function selectTypingUsers(
  state: PresenceState,
  channelId: string | null,
  selfId: string | null,
): User[] {
  if (!channelId) return [];
  const bucket = state.typingByChannelId[channelId] ?? {};
  return Object.values(bucket)
    .filter((entry) => entry.expiresAt > Date.now() && entry.user.id !== selfId)
    .map((entry) => entry.user);
}

export function selectStatus(
  state: PresenceState,
  userId: string | null,
  fallback: UserStatus = "offline",
): UserStatus {
  if (!userId) return fallback;
  return state.statusByUserId[userId] ?? fallback;
}