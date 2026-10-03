import { del, get, post } from "./client";
import type { Reaction } from "@/types";

/**
 * Toggling is server side: adding an existing reaction removes it and answers
 * 204, so both outcomes are represented by a void return here.
 */
export async function toggleReaction(messageId: string, emoji: string): Promise<void> {
  await post<void>(`/api/messages/${messageId}/reactions`, { emoji });
}

export function removeReaction(
  messageId: string,
  selector: { emoji?: string; userId?: string },
): Promise<void> {
  const params = new URLSearchParams();
  if (selector.emoji) params.set("emoji", selector.emoji);
  if (selector.userId) params.set("userId", selector.userId);
  const search = params.toString();
  return del<void>(`/api/messages/${messageId}/reactions?${search}`);
}

export function listReactions(messageId: string): Promise<Reaction[]> {
  return get<Reaction[]>(`/api/messages/${messageId}/reactions`);
}