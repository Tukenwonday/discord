import { get, patch } from "./client";
import type { Paginated, UpdateProfileInput, User } from "@/types";

export function me(): Promise<User> {
  return get<User>("/api/users/me");
}

export function updateMe(input: UpdateProfileInput): Promise<User> {
  return patch<User>("/api/users/me", input);
}

export function getUser(id: string): Promise<User> {
  return get<User>(`/api/users/${id}`);
}

export function searchUsers(query: string, limit = 20): Promise<Paginated<User>> {
  const params = new URLSearchParams({ q: query, limit: String(limit) });
  return get<Paginated<User>>(`/api/users?${params.toString()}`);
}