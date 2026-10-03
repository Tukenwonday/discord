import type {
  ApiErrorCode,
  AttachmentInput,
  Friend,
  Message,
  User,
  UserStatus,
} from "@/types";

/** Envelope shape is identical in both directions. */
export interface WsEnvelope<K extends string, D> {
  event: K;
  data: D;
  nonce?: string;
}

export interface ReadyData {
  userId: string;
  sessionId: string;
  serverTime: string;
}

export interface MessageCreateData {
  channelId: string;
  content: string;
  replyToId?: string;
  attachments?: AttachmentInput[];
}

export interface MessageUpdateData {
  id: string;
  content: string;
}

export interface MessageDeleteData {
  id: string;
}

export interface TypingData {
  channelId: string;
}

export interface PresenceUpdateData {
  status: Exclude<UserStatus, "offline">;
  customStatus?: string;
}

export interface FriendRequestData {
  userId: string;
}

export interface FriendAcceptedData {
  friendId: string;
}

/** Frames the frontend is allowed to put on the wire. */
export interface ClientEventMap {
  "message.create": MessageCreateData;
  "message.update": MessageUpdateData;
  "message.delete": MessageDeleteData;
  "typing.start": TypingData;
  "typing.stop": TypingData;
  "presence.update": PresenceUpdateData;
  "friend.request": FriendRequestData;
  "friend.accepted": FriendAcceptedData;
  heartbeat: Record<string, never>;
}

export interface MessageCreatedData {
  message: Message;
  channelId: string;
  dmId: string;
}

export interface MessageUpdatedData {
  message: Message;
  channelId: string;
}

export interface MessageDeletedData {
  id: string;
  channelId: string;
  deletedAt: string;
}

export interface TypingStartedData {
  channelId: string;
  user: User;
}

export interface TypingStoppedData {
  channelId: string;
  userId: string;
}

export interface PresenceUpdatedData {
  userId: string;
  status: UserStatus;
  customStatus: string;
  at: string;
}

export interface FriendEventData {
  friend: Friend;
}

export interface WsErrorData {
  code: ApiErrorCode;
  message: string;
  nonce: string;
}

/** Frames the backend sends to the client. */
export interface ServerEventMap {
  ready: ReadyData;
  "message.created": MessageCreatedData;
  "message.updated": MessageUpdatedData;
  "message.deleted": MessageDeletedData;
  "typing.started": TypingStartedData;
  "typing.stopped": TypingStoppedData;
  "presence.updated": PresenceUpdatedData;
  "friend.request": FriendEventData;
  "friend.accepted": FriendEventData;
  error: WsErrorData;
}

export type ClientEvent = keyof ClientEventMap;
export type ServerEvent = keyof ServerEventMap;

export type ClientHandler<K extends ServerEvent> = (
  data: ServerEventMap[K],
  envelope: WsEnvelope<K, ServerEventMap[K]>,
) => void;

export type ConnectionState =
  | "idle"
  | "connecting"
  | "open"
  | "reconnecting"
  | "closed";

export type StatusListener = (state: ConnectionState) => void;

export type Unsubscribe = () => void;