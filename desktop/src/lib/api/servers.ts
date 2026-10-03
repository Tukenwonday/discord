import { del, get, patch, post } from "./client";
import type {
  CreateInviteInput,
  CreateServerInput,
  Invite,
  Paginated,
  Server,
  ServerDetail,
  UpdateServerInput,
} from "@/types";

export function listServers(): Promise<Paginated<Server>> {
  return get<Paginated<Server>>("/api/servers");
}

export function createServer(input: CreateServerInput): Promise<ServerDetail> {
  return post<ServerDetail>("/api/servers", input);
}

export function getServer(id: string): Promise<ServerDetail> {
  return get<ServerDetail>(`/api/servers/${id}`);
}

export function updateServer(id: string, input: UpdateServerInput): Promise<Server> {
  return patch<Server>(`/api/servers/${id}`, input);
}

export function deleteServer(id: string): Promise<void> {
  return del<void>(`/api/servers/${id}`);
}

export function listInvites(serverId: string): Promise<Invite[]> {
  return get<Invite[]>(`/api/servers/${serverId}/invites`);
}

export function createInvite(serverId: string, input: CreateInviteInput = {}): Promise<Invite> {
  return post<Invite>(`/api/servers/${serverId}/invites`, input);
}

export function deleteInvite(code: string): Promise<void> {
  return del<void>(`/api/invites/${code}`);
}

/** Public preview used by the invite dialog before joining. */
export function getInvite(code: string): Promise<Invite> {
  return get<Invite>(`/api/invites/${code}`, { anonymous: true });
}

export function joinInvite(code: string): Promise<ServerDetail> {
  return post<ServerDetail>(`/api/invites/${code}/join`);
}