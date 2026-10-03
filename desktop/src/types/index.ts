export type UserStatus = "online" | "idle" | "dnd" | "offline" | "invisible";

export interface User {
  id: string;
  username: string;
  displayName: string;
  email: string;
  avatarUrl: string | null;
  bannerUrl: string | null;
  bio: string;
  status: UserStatus;
  customStatus: string;
  createdAt: string;
  updatedAt: string;
}

export interface Role {
  id: string;
  serverId: string;
  name: string;
  color: string;
  permissions: number;
  hoist: boolean;
  position: number;
}

export interface Member {
  id: string;
  serverId: string;
  user: User;
  nickname: string;
  roleIds: string[];
  joinedAt: string;
}

export type ChannelType = "text" | "voice" | "video" | "category";

export interface Channel {
  id: string;
  serverId: string;
  parentId: string | null;
  name: string;
  topic: string;
  type: ChannelType;
  position: number;
  createdAt: string;
  updatedAt: string;
}

export interface Attachment {
  id: string;
  url: string;
  filename: string;
  size: number;
  contentType: string;
  width: number | null;
  height: number | null;
}

export interface Reaction {
  id: string;
  messageId: string;
  userId: string;
  emoji: string;
  createdAt: string;
}

export type MessageType = "text" | "image" | "file" | "system";

export interface Message {
  id: string;
  channelId: string;
  author: User;
  content: string;
  type: MessageType;
  replyTo: Message | null;
  attachments: Attachment[];
  reactions: Reaction[];
  editedAt: string | null;
  createdAt: string;
}

export interface Server {
  id: string;
  name: string;
  description: string;
  iconUrl: string | null;
  bannerUrl: string | null;
  ownerId: string;
  createdAt: string;
  updatedAt: string;
}

export interface DMChannel {
  id: string;
  createdAt: string;
  updatedAt: string;
  recipients: User[];
  lastMessage: Message | null;
}

export type FriendStatus = "pending" | "accepted" | "blocked";

export interface Friend {
  id: string;
  status: FriendStatus;
  user: User;
  createdAt: string;
}

export interface Invite {
  code: string;
  server: Server;
  channelId: string | null;
  inviter: User;
  expiresAt: string | null;
  uses: number;
  maxUses: number;
  createdAt: string;
}

export interface Paginated<T> {
  items: T[];
  hasMore: boolean;
  nextCursor: string | null;
}

export interface ServerDetail {
  server: Server;
  channels: Channel[];
  members: Member[];
  roles: Role[];
  owner: User;
  unreadCount: number;
  invites: Invite[];
}

/** Body carried by every failing request, wrapped in `{ error: ... }`. */
export interface ApiErrorBody {
  code: ApiErrorCode;
  message: string;
  fields?: Record<string, string>;
}

export type ApiErrorCode =
  | "validation_error"
  | "unauthorized"
  | "forbidden"
  | "not_found"
  | "conflict"
  | "internal_error"
  | "payload_too_large"
  | "unsupported_media_type"
  | "rate_limited"
  | "network_error";

export interface ApiErrorEnvelope {
  error: ApiErrorBody;
}

/** Auth payload shared by register and login. */
export interface AuthResponse {
  user: User;
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

export interface RefreshResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

export interface LiveKitTokenResponse {
  token: string;
  url: string;
  roomName: string;
  expiresIn: number;
}

export interface FriendsResponse {
  friends: Friend[];
  incoming: Friend[];
  outgoing: Friend[];
}

export interface UpdateProfileInput {
  displayName?: string;
  bio?: string;
  avatarUrl?: string;
  bannerUrl?: string;
  status?: UserStatus;
  customStatus?: string;
}

export interface CreateServerInput {
  name: string;
  description?: string;
}

export interface UpdateServerInput {
  name?: string;
  description?: string;
  iconUrl?: string;
  bannerUrl?: string;
}

export interface CreateChannelInput {
  name: string;
  type?: ChannelType;
  topic?: string;
  parentId?: string | null;
}

export interface UpdateChannelInput {
  name?: string;
  topic?: string;
  position?: number;
  parentId?: string | null;
}

export interface AttachmentInput {
  url: string;
  filename: string;
  size: number;
  contentType: string;
  width?: number;
  height?: number;
}

export interface SendMessageInput {
  content: string;
  replyToId?: string;
  attachments?: AttachmentInput[];
}

export interface CreateInviteInput {
  channelId?: string | null;
  maxUses?: number;
  expiresInHours?: number;
}

export type LiveKitRoomType = "channel" | "dm";

export interface LiveKitTokenInput {
  roomName: string;
  roomType: LiveKitRoomType;
  canPublish?: boolean;
  canSubscribe?: boolean;
}

export interface MessageQuery {
  before?: string;
  after?: string;
  limit?: number;
}

export type VoiceRoomState = "disconnected" | "connecting" | "connected" | "error";

/** Permission bitmask, mirrored from contract section 7. */
export const Permissions = {
  CreateInstantInvite: 1,
  KickMembers: 2,
  BanMembers: 4,
  Administrator: 8,
  ManageChannels: 16,
  ManageServer: 32,
  AddReactions: 64,
  ViewAuditLog: 128,
  ViewChannel: 1024,
  SendMessages: 2048,
  SendTTSMessages: 4096,
  ManageMessages: 8192,
  EmbedLinks: 16384,
  AttachFiles: 32768,
  ReadMessageHistory: 65536,
  MentionEveryone: 131072,
  PrioritySpeaker: 262144,
  UseExternalEmojis: 524288,
  Stream: 1048576,
  ViewServerInsights: 2097152,
} as const;

/**
 * True when the member owns the server, holds Administrator, or one of their
 * roles (the caller is expected to pass @everyone too) carries the bit.
 */
export function can(
  member: Member | null,
  roles: Role[],
  server: Server | null,
  permission: number,
): boolean {
  if (!member || !server) return false;
  if (member.user.id === server.ownerId) return true;

  const owned = roles.filter((role) => member.roleIds.includes(role.id));
  if (owned.some((role) => (role.permissions & Permissions.Administrator) !== 0)) {
    return true;
  }
  return owned.some((role) => (role.permissions & permission) === permission);
}