import { del, get, patch, post } from "./client";
import type { Message, MessageQuery, Paginated, SendMessageInput } from "@/types";

export const DEFAULT_MESSAGE_LIMIT = 50;
export const MAX_MESSAGE_LIMIT = 100;

/** `before`/`after` are message ids; `limit` is clamped to the documented range. */
export function listMessages(
  channelId: string,
  query: MessageQuery = {},
): Promise<Paginated<Message>> {
  const params = new URLSearchParams();
  if (query.before) params.set("before", query.before);
  if (query.after) params.set("after", query.after);
  const limit = Math.min(Math.max(query.limit ?? DEFAULT_MESSAGE_LIMIT, 1), MAX_MESSAGE_LIMIT);
  params.set("limit", String(limit));

  const search = params.toString();
  return get<Paginated<Message>>(`/api/channels/${channelId}/messages?${search}`);
}

export function sendMessage(channelId: string, input: SendMessageInput): Promise<Message> {
  return post<Message>(`/api/channels/${channelId}/messages`, input);
}

export function editMessage(id: string, content: string): Promise<Message> {
  return patch<Message>(`/api/messages/${id}`, { content });
}

export function deleteMessage(id: string): Promise<void> {
  return del<void>(`/api/messages/${id}`);
}