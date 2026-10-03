import { del, patch, post } from "./client";
import type { Channel, ChannelType, CreateChannelInput, UpdateChannelInput } from "@/types";

export function createChannel(
  serverId: string,
  input: CreateChannelInput & { type?: ChannelType },
): Promise<Channel> {
  return post<Channel>(`/api/servers/${serverId}/channels`, input);
}

export function updateChannel(id: string, input: UpdateChannelInput): Promise<Channel> {
  return patch<Channel>(`/api/channels/${id}`, input);
}

export function deleteChannel(id: string): Promise<void> {
  return del<void>(`/api/channels/${id}`);
}