import { del, get, post } from "./client";
import type {
  AttachmentInput,
  DMChannel,
  Message,
  MessageQuery,
  Paginated,
  User,
} from "@/types";
import { DEFAULT_MESSAGE_LIMIT, MAX_MESSAGE_LIMIT } from "./messages";

export function listDms(): Promise<Paginated<DMChannel>> {
  return get<Paginated<DMChannel>>("/api/dms");
}

/** Idempotent: returns the existing conversation when one already exists. */
export function createDm(recipientId: string): Promise<DMChannel> {
  return post<DMChannel>("/api/dms", { recipientId });
}

export function listDmMessages(dmId: string, query: MessageQuery = {}): Promise<Paginated<Message>> {
  const params = new URLSearchParams();
  if (query.before) params.set("before", query.before);
  if (query.after) params.set("after", query.after);
  const limit = Math.min(Math.max(query.limit ?? DEFAULT_MESSAGE_LIMIT, 1), MAX_MESSAGE_LIMIT);
  params.set("limit", String(limit));
  return get<Paginated<Message>>(`/api/dms/${dmId}/messages?${params.toString()}`);
}

export function sendDmMessage(
  dmId: string,
  content: string,
  attachments: AttachmentInput[] = [],
): Promise<Message> {
  return post<Message>(`/api/dms/${dmId}/messages`, { content, attachments });
}

export function leaveDm(dmId: string): Promise<void> {
  return del<void>(`/api/dms/${dmId}`);
}

export function listDmRecipients(dmId: string): Promise<User[]> {
  return get<User[]>(`/api/dms/${dmId}/recipients`);
}