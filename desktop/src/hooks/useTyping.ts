import { useCallback, useEffect, useRef } from "react";
import { socket } from "@/lib/ws/client";
import { usePresenceStore } from "@/stores/presence.store";
import type { User } from "@/types";

/** The contract asks for one `typing.start` at most every 3 seconds. */
const TYPING_THROTTLE = 3_000;
/** Idle for this long and the backend drops the typing indicator. */
const TYPING_IDLE = 5_000;

export interface UseTypingResult {
  typingUsers: (channelId: string | null) => User[];
  /** Call on every keystroke; throttling is handled internally. */
  notifyTyping: (channelId: string) => void;
  /** Call on send, blur or unmount so the indicator clears immediately. */
  notifyStopped: (channelId: string) => void;
}

export function useTyping(channelId: string | null): UseTypingResult {
  const lastSentAt = useRef(0);
  const idleTimer = useRef<number | null>(null);
  const active = useRef(false);

  const typingByChannelId = usePresenceStore((state) => state.typingByChannelId);

  const clearIdleTimer = useCallback(() => {
    if (idleTimer.current !== null) {
      window.clearTimeout(idleTimer.current);
      idleTimer.current = null;
    }
  }, []);

  const notifyStopped = useCallback(
    (target: string) => {
      clearIdleTimer();
      if (!active.current) return;
      active.current = false;
      lastSentAt.current = 0;
      socket.send("typing.stop", { channelId: target });
    },
    [clearIdleTimer],
  );

  const notifyTyping = useCallback(
    (target: string) => {
      const now = Date.now();
      // Re-announce every 3s so the server's 10s window never lapses.
      if (!active.current || now - lastSentAt.current >= TYPING_THROTTLE) {
        socket.send("typing.start", { channelId: target });
        lastSentAt.current = now;
        active.current = true;
      }

      clearIdleTimer();
      idleTimer.current = window.setTimeout(() => {
        notifyStopped(target);
      }, TYPING_IDLE);
    },
    [clearIdleTimer, notifyStopped],
  );

  // Switching conversations must clear the indicator on the previous one.
  useEffect(() => {
    return () => {
      clearIdleTimer();
      if (active.current) {
        socket.send("typing.stop", { channelId: channelId ?? "" });
        active.current = false;
      }
    };
  }, [channelId, clearIdleTimer]);

  const typingUsers = useCallback(
    (target: string | null): User[] => {
      const bucket = target ? typingByChannelId[target] : undefined;
      if (!bucket) return [];
      return Object.values(bucket)
        .filter((entry) => entry.expiresAt > Date.now())
        .map((entry) => entry.user);
    },
    [typingByChannelId],
  );

  return { typingUsers, notifyTyping, notifyStopped };
}