import { useCallback, useEffect, type React } from "react";
import type { Channel } from "@/types";
import { socket } from "@/lib/ws/client";
import { useMessagesStore } from "@/stores/messages.store";
import { usePresenceStore } from "@/stores/presence.store";
import { useFriendsRefetch } from "@/hooks/useFriends";

export interface UseHotkeysOptions {
  onToggleMembers: () => void;
  onToggleSettings: () => void;
  onSearch: () => void;
  onEscape: () => void;
  onQuickSwitcher: () => void;
}

/**
 * Global shortcuts. Bindings follow the Discord muscle memory: Ctrl/Cmd+K for
 * quick switcher, Ctrl+Shift+M for the member list, Ctrl+, for settings.
 */
export function useHotkeys(options: UseHotkeysOptions): void {
  const { onToggleMembers, onToggleSettings, onSearch, onEscape, onQuickSwitcher } = options;

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      const ctrl = event.ctrlKey || event.metaKey;

      if (event.key === "Escape") {
        onEscape();
        return;
      }

      if (!ctrl) return;

      const key = event.key.toLowerCase();

      if (key === "k" && !event.shiftKey) {
        event.preventDefault();
        onQuickSwitcher();
        return;
      }
      if (key === "k" && event.shiftKey) {
        event.preventDefault();
        onSearch();
        return;
      }
      if (key === "m" && event.shiftKey) {
        event.preventDefault();
        onToggleMembers();
        return;
      }
      if (key === ",") {
        event.preventDefault();
        onToggleSettings();
      }
    };

    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onToggleMembers, onToggleSettings, onSearch, onEscape, onQuickSwitcher]);
}

export interface UseMessageInputOptions {
  channel: Channel | null;
  sendOnEnter: boolean;
  onSend: (content: string) => void;
}

export interface MessageInputHandlers {
  onKeyDown: (event: React.KeyboardEvent<HTMLTextAreaElement>) => void;
}

/**
 * Enter sends and Shift+Enter inserts a newline, inverted when the user turns
 * `sendOnEnter` off in settings.
 */
export function useMessageInput(options: UseMessageInputOptions): MessageInputHandlers {
  const { channel, sendOnEnter, onSend } = options;

  const onKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (!channel) return;
      if (event.key !== "Enter") return;

      const shouldSend = event.shiftKey ? !sendOnEnter : sendOnEnter;
      if (!shouldSend) return;

      event.preventDefault();
      const target = event.currentTarget;
      onSend(target.value);
      target.value = "";
    },
    [channel, sendOnEnter, onSend],
  );

  return { onKeyDown };
}

/**
 * Owns the single WebSocket subscription for live messages, presence, typing
 * and friends, writing results into the stores.
 */
export function useSocketBridge(): void {
  const refetchFriends = useFriendsRefetch();

  useEffect(() => {
    const presence = usePresenceStore.getState();
    const messages = useMessagesStore.getState();

    const unsubscribes = [
      socket.on("message.created", (data) => {
        // The socket routes by conversation: a DM reports `dmId` instead.
        const bucket = data.dmId ?? data.channelId;
        if (!bucket) return;
        messages.reconcile(bucket, data.message);
      }),

      socket.on("message.updated", (data) => {
        messages.applyUpdate(data.channelId, data.message);
      }),

      socket.on("message.deleted", (data) => {
        messages.applyDeletion(data.channelId, data.id);
      }),

      socket.on("typing.started", (data) => {
        presence.startTyping(data.channelId, data.user);
      }),

      socket.on("typing.stopped", (data) => {
        presence.stopTyping(data.channelId, data.userId);
      }),

      socket.on("presence.updated", (data) => {
        presence.applyPresence(data.userId, data.status, data.customStatus);
      }),

      socket.on("friend.request", () => {
        refetchFriends();
      }),

      socket.on("friend.accepted", () => {
        refetchFriends();
      }),

      socket.on("error", (data) => {
        console.error(`[ws] ${data.code}: ${data.message}`);
      }),
    ];

    return () => {
      for (const unsubscribe of unsubscribes) unsubscribe();
    };
  }, [refetchFriends]);
}