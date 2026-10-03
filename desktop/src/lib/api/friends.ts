import { del, get, patch, post } from "./client";
import type { Friend, FriendsResponse } from "@/types";

export function listFriends(): Promise<FriendsResponse> {
  return get<FriendsResponse>("/api/friends");
}

/** Answers 201 for a new row and 200 when the request already exists. */
export function addFriend(userId: string): Promise<Friend> {
  return post<Friend>("/api/friends", { userId });
}

export function acceptFriend(friendId: string): Promise<Friend> {
  return patch<Friend>(`/api/friends/${friendId}`, { status: "accepted" });
}

export function removeFriend(friendId: string): Promise<void> {
  return del<void>(`/api/friends/${friendId}`);
}