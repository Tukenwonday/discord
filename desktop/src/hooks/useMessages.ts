import { useCallback, useMemo } from "react";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import { listMessages } from "@/lib/api/messages";
import { listDmMessages } from "@/lib/api/dms";
import { useMessagesStore } from "@/stores/messages.store";
import type { Message } from "@/types";

const PAGE_SIZE = 50;

interface MessagePage {
  items: Message[];
  hasMore: boolean;
}

export interface UseMessagesResult {
  messages: Message[];
  isLoading: boolean;
  isFetching: boolean;
  hasMore: boolean;
  fetchOlder: () => void;
  isFetchingOlder: boolean;
  loadNewest: () => void;
}

/**
 * Paginates backwards: the cursor is the oldest id already held, matching the
 * contract's `?before=` semantics. Pages arrive oldest-first internally, so the
 * flattened list is reversed to read chronologically.
 */
export function useMessages(channelId: string | null): UseMessagesResult {
  const queryClient = useQueryClient();

  const query = useInfiniteQuery<MessagePage, Error>({
    queryKey: ["messages", channelId],
    enabled: Boolean(channelId),
    initialPageParam: null as string | null,
    queryFn: ({ pageParam }) =>
      listMessages(channelId as string, {
        before: pageParam ?? undefined,
        limit: PAGE_SIZE,
      }),
    getNextPageParam: (lastPage) => {
      if (!lastPage.hasMore || lastPage.items.length === 0) return undefined;
      return lastPage.items[0].id;
    },
  });

  const loadNewest = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["messages", channelId] });
  }, [queryClient, channelId]);

  const messages = useMemo(() => {
    const pages = query.data?.pages ?? [];
    const flattened: Message[] = [];
    for (const page of pages) flattened.push(...page.items);
    return flattened;
  }, [query.data]);

  const fetchOlder = useCallback(() => {
    if (query.hasNextPage && !query.isFetchingNextPage) {
      void query.fetchNextPage();
    }
  }, [query]);

  return {
    messages,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    hasMore: Boolean(query.hasNextPage),
    fetchOlder,
    isFetchingOlder: query.isFetchingNextPage,
    loadNewest,
  };
}

/** Same pagination, backed by the DM endpoints. */
export function useDmMessages(dmId: string | null): UseMessagesResult {
  const queryClient = useQueryClient();

  const query = useInfiniteQuery<MessagePage, Error>({
    queryKey: ["dm-messages", dmId],
    enabled: Boolean(dmId),
    initialPageParam: null as string | null,
    queryFn: ({ pageParam }) =>
      listDmMessages(dmId as string, {
        before: pageParam ?? undefined,
        limit: PAGE_SIZE,
      }),
    getNextPageParam: (lastPage) => {
      if (!lastPage.hasMore || lastPage.items.length === 0) return undefined;
      return lastPage.items[0].id;
    },
  });

  const messages = useMemo(() => {
    const pages = query.data?.pages ?? [];
    const flattened: Message[] = [];
    for (const page of pages) flattened.push(...page.items);
    return flattened;
  }, [query.data]);

  const fetchOlder = useCallback(() => {
    if (query.hasNextPage && !query.isFetchingNextPage) {
      void query.fetchNextPage();
    }
  }, [query]);

  const loadNewest = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["dm-messages", dmId] });
  }, [queryClient, dmId]);

  return {
    messages,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    hasMore: Boolean(query.hasNextPage),
    fetchOlder,
    isFetchingOlder: query.isFetchingNextPage,
    loadNewest,
  };
}

export interface UseMessageActions {
  send: (input: { content: string; replyToId?: string }) => Promise<Message | null>;
  sendDm: (dmId: string, content: string) => Promise<Message | null>;
  edit: (messageId: string, content: string) => Promise<boolean>;
  remove: (messageId: string) => Promise<boolean>;
  toggleReaction: (messageId: string, emoji: string) => Promise<boolean>;
}

export function useMessageActions(channelId: string | null): UseMessageActions {
  const optimisticSend = useMessagesStore((state) => state.optimisticSend);
  const optimisticDmSend = useMessagesStore((state) => state.optimisticDmSend);
  const edit = useMessagesStore((state) => state.edit);
  const remove = useMessagesStore((state) => state.remove);
  const toggleReaction = useMessagesStore((state) => state.toggleReaction);

  return useMemo(
    () => ({
      send: (input) =>
        channelId ? optimisticSend(channelId, input) : Promise.resolve(null),
      sendDm: (dmId, content) => optimisticDmSend(dmId, content),
      edit: (messageId, content) =>
        channelId ? edit(channelId, messageId, content) : Promise.resolve(false),
      remove: (messageId) =>
        channelId ? remove(channelId, messageId) : Promise.resolve(false),
      toggleReaction: (messageId, emoji) =>
        channelId
          ? toggleReaction(channelId, messageId, emoji)
          : Promise.resolve(false),
    }),
    [channelId, optimisticSend, optimisticDmSend, edit, remove, toggleReaction],
  );
}