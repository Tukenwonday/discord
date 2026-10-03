import { create } from "zustand";
import type { AttachmentInput, Message, Reaction, User } from "@/types";
import * as messagesApi from "@/lib/api/messages";
import * as reactionsApi from "@/lib/api/reactions";
import * as dmsApi from "@/lib/api/dms";
import { tempId } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth.store";

/** Optimistic rows live alongside confirmed ones until `message.created` lands. */
export interface OptimisticMeta {
  failed?: boolean;
  pendingReplyId?: string;
}

export interface MessagesState {
  byChannelId: Record<string, Message[]>;
  byDmId: Record<string, Message[]>;
  metaById: Record<string, OptimisticMeta>;
  loading: boolean;
  error: string | null;
  setLoading: (value: boolean) => void;
  setError: (error: string | null) => void;
  prependPage: (channelId: string, messages: Message[]) => void;
  appendMessage: (channelId: string, message: Message) => void;
  upsertMessage: (channelId: string, message: Message) => void;
  removeMessage: (channelId: string, messageId: string) => void;
  /** Replaces a `tmp_` row with the confirmed message from the socket. */
  reconcile: (channelId: string, message: Message) => void;
  applyUpdate: (channelId: string, message: Message) => void;
  applyDeletion: (channelId: string, messageId: string) => void;
  optimisticSend: (channelId: string, input: {
    content: string;
    replyToId?: string;
    attachments?: AttachmentInput[];
  }) => Promise<Message | null>;
  optimisticDmSend: (dmId: string, content: string) => Promise<Message | null>;
  edit: (channelId: string, messageId: string, content: string) => Promise<boolean>;
  remove: (channelId: string, messageId: string) => Promise<boolean>;
  toggleReaction: (channelId: string, messageId: string, emoji: string) => Promise<boolean>;
  clear: () => void;
}

function sortByCreatedAt(messages: Message[]): Message[] {
  return [...messages].sort(
    (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
  );
}

/** Applies `patch` to the matching message and returns a new array. */
function mapMessage(
  list: Message[],
  messageId: string,
  patch: (message: Message) => Message,
): Message[] {
  let touched = false;
  const next = list.map((message) => {
    if (message.id !== messageId) return message;
    touched = true;
    return patch(message);
  });
  return touched ? next : list;
}

/** Builds the placeholder row shown between send and server confirmation. */
function buildOptimistic(
  user: User,
  content: string,
  attachments: AttachmentInput[],
): Message {
  return {
    id: tempId(),
    channelId: "",
    author: user,
    content,
    type: attachments.length > 0 ? "file" : "text",
    replyTo: null,
    attachments: [],
    reactions: [],
    editedAt: null,
    createdAt: new Date().toISOString(),
  };
}

/** Drops a confirmed message into a channel cache, replacing its temp row. */
function withMessage(list: Message[], message: Message): Message[] {
  const withoutTemp = list.filter(
    (existing) => existing.id !== message.id && !isSameOptimistic(existing, message),
  );
  return sortByCreatedAt([...withoutTemp, message]);
}

/**
 * A temp row and the confirmed message share an author and a near-identical
 * timestamp; matching on a 10s window keeps the reconcile step from duplicating
 * rows when the socket arrives before the HTTP response.
 */
function isSameOptimistic(candidate: Message, confirmed: Message): boolean {
  if (!candidate.id.startsWith("tmp_")) return false;
  if (candidate.author.id !== confirmed.author.id) return false;
  if (candidate.content.trim() !== confirmed.content.trim()) return false;
  const delta =
    Math.abs(new Date(candidate.createdAt).getTime() - new Date(confirmed.createdAt).getTime());
  return delta < 10_000;
}
export const useMessagesStore = create<MessagesState>((set, get) => ({
  byChannelId: {},
  byDmId: {},
  metaById: {},
  loading: false,
  error: null,

  setLoading: (value) => set({ loading: value }),
  setError: (error) => set({ error }),

  prependPage: (channelId, messages) =>
    set((state) => {
      const existing = state.byChannelId[channelId] ?? [];
      const known = new Set(existing.map((message) => message.id));
      const fresh = messages.filter((message) => !known.has(message.id));
      if (fresh.length === 0) return state;
      return {
        byChannelId: {
          ...state.byChannelId,
          [channelId]: sortByCreatedAt([...fresh, ...existing]),
        },
      };
    }),

  upsertMessage: (channelId, message) =>
    set((state) => {
      // A DM bucket takes precedence: an id cannot be both at once.
      if (state.byDmId[channelId]) {
        return {
          byDmId: { ...state.byDmId, [channelId]: withMessage(state.byDmId[channelId], message) },
        };
      }
      const existing = state.byChannelId[channelId] ?? [];
      return {
        byChannelId: { ...state.byChannelId, [channelId]: withMessage(existing, message) },
      };
    }),

  appendMessage: (channelId, message) => get().upsertMessage(channelId, message),

  reconcile: (channelId, message) => get().upsertMessage(channelId, message),

  removeMessage: (channelId, messageId) =>
    set((state) => {
      const list = state.byChannelId[channelId];
      if (!list) return state;
      return {
        byChannelId: {
          ...state.byChannelId,
          [channelId]: list.filter((message) => message.id !== messageId),
        },
      };
    }),

  /** Replaces a message in whichever bucket holds the conversation. */
  applyUpdate: (channelId, message) =>
    set((state) => {
      const list = state.byChannelId[channelId] ?? state.byDmId[channelId];
      if (!list) return state;
      const next = mapMessage(list, message.id, () => message);
      if (state.byChannelId[channelId]) {
        return { byChannelId: { ...state.byChannelId, [channelId]: next } };
      }
      return { byDmId: { ...state.byDmId, [channelId]: next } };
    }),

  applyDeletion: (channelId, messageId) =>
    set((state) => {
      const inChannel = state.byChannelId[channelId];
      if (inChannel) {
        return {
          byChannelId: {
            ...state.byChannelId,
            [channelId]: inChannel.filter((message) => message.id !== messageId),
          },
        };
      }
      const inDm = state.byDmId[channelId];
      if (inDm) {
        return {
          byDmId: {
            ...state.byDmId,
            [channelId]: inDm.filter((message) => message.id !== messageId),
          },
optimisticSend: async (channelId, input) => {
    const user = useAuthStore.getState().user;
    if (!user) return null;

    const attachments = input.attachments ?? [];
    const placeholder = buildOptimistic(user, input.content, attachments);
    set((state) => ({
      byChannelId: {
        ...state.byChannelId,
        [channelId]: [...(state.byChannelId[channelId] ?? []), placeholder],
      },
      metaById: input.replyToId
        ? { ...state.metaById, [placeholder.id]: { pendingReplyId: input.replyToId } }
        : state.metaById,
    }));

    try {
      const confirmed = await messagesApi.sendMessage(channelId, {
        content: input.content,
        ...(input.replyToId ? { replyToId: input.replyToId } : {}),
        ...(attachments.length > 0 ? { attachments } : {}),
      });
      get().reconcile(channelId, confirmed);
      set((state) => {
        const meta = { ...state.metaById };
        delete meta[placeholder.id];
        return { metaById: meta };
      });
      return confirmed;
    } catch (error) {
      // Keep the row but flag it so the UI can offer a retry.
      set((state) => ({
        metaById: { ...state.metaById, [placeholder.id]: { failed: true } },
        error: error instanceof Error ? error.message : "Message could not be sent",
      }));
      return null;
    }
  },

  optimisticDmSend: async (dmId, content) => {
    const user = useAuthStore.getState().user;
    if (!user) return null;

    const placeholder = buildOptimistic(user, content, []);
    set((state) => ({
      byDmId: { ...state.byDmId, [dmId]: [...(state.byDmId[dmId] ?? []), placeholder] },
    }));

    try {
      const confirmed = await dmsApi.sendDmMessage(dmId, content);
      get().reconcile(dmId, confirmed);
      return confirmed;
    } catch (error) {
      set((state) => ({
        metaById: { ...state.metaById, [placeholder.id]: { failed: true } },
        error: error instanceof Error ? error.message : "Message could not be sent",
      }));
      return null;
    }
  },

  edit: async (channelId, messageId, content) => {
    const current =
      get().byChannelId[channelId]?.find((message) => message.id === messageId) ??
      get().byDmId[channelId]?.find((message) => message.id === messageId);
    if (!current) return false;

    get().applyUpdate(channelId, {
      ...current,
      content,
      editedAt: new Date().toISOString(),
    });

    try {
      const confirmed = await messagesApi.editMessage(messageId, content);
      get().applyUpdate(channelId, confirmed);
      return true;
    } catch (error) {
      get().applyUpdate(channelId, current);
      set({ error: error instanceof Error ? error.message : "Could not edit message" });
      return false;
    }
  },

  remove: async (channelId, messageId) => {
    const previous =
      get().byChannelId[channelId]?.find((message) => message.id === messageId) ??
      get().byDmId[channelId]?.find((message) => message.id === messageId);
    if (!previous) return false;

    get().applyDeletion(channelId, messageId);
    try {
      await messagesApi.deleteMessage(messageId);
      return true;
    } catch (error) {
      // Rollback: put the message back where it was.
      get().reconcile(channelId, previous);
      set({ error: error instanceof Error ? error.message : "Could not delete message" });
      return false;
    }
  },

  toggleReaction: async (channelId, messageId, emoji) => {
    const message =
      get().byChannelId[channelId]?.find((item) => item.id === messageId) ??
      get().byDmId[channelId]?.find((item) => item.id === messageId);
    if (!message) return false;

    const me = useAuthStore.getState().user;
    const userId = me?.id ?? "";
    const alreadyMine =
      me !== null && message.reactions.some((r) => r.emoji === emoji && r.userId === userId);
/* ------------------------------------------------------------------ */
/* Derived selectors                                                   */
/* ------------------------------------------------------------------ */

/** Live rows for a conversation, whether it is a server channel or a DM. */
export function selectChannelMessages(
  state: MessagesState,
  conversationId: string | null,
): Message[] {
  if (!conversationId) return [];
  return state.byChannelId[conversationId] ?? state.byDmId[conversationId] ?? [];
}

export function selectDmMessages(state: MessagesState, dmId: string | null): Message[] {
  if (!dmId) return [];
  return state.byDmId[dmId] ?? [];
}

    const optimistic: Reaction[] = alreadyMine
      ? message.reactions.filter((r) => !(r.emoji === emoji && r.userId === userId))
      : [
          ...message.reactions,
          {
            id: `tmp_reaction_${messageId}_${emoji}`,
            messageId,
            userId,
            emoji,
            createdAt: new Date().toISOString(),
          },
        ];

    get().applyUpdate(channelId, { ...message, reactions: optimistic });

    try {
      await reactionsApi.toggleReaction(messageId, emoji);
      // The endpoint toggles, so re-read instead of guessing the end state.
      const fresh = await reactionsApi.listReactions(messageId);
      get().applyUpdate(channelId, { ...message, reactions: fresh });
      return true;
    } catch (error) {
      get().applyUpdate(channelId, message);
      set({ error: error instanceof Error ? error.message : "Could not update reaction" });
      return false;
    }
  },

  clear: () => set({ byChannelId: {}, byDmId: {}, metaById: {}, error: null }),
}));
        };
      }
      return state;
    }),