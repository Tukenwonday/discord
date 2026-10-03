# Cordis - Internal Monorepo Contract (AUTHORITATIVE)

All packages (`backend`, `desktop`, `desktop/src-tauri`) MUST conform to this document.
Do not invent alternative field names, envelope shapes, or event names.

## 1. Repository layout

```
/backend                     Go 1.23 API + WebSocket + LiveKit + uploads (module github.com/cordis/backend)
/desktop                     Tauri v2 + React 18 + TS + Vite + Tailwind + shadcn/ui
  /desktop/src-tauri         Rust shell (tray, notifications, updater, window, dialogs)
/docker-compose.yml
/.env.example
/.gitignore
/README.md
/CONTRACT.md                 (this file)
```

`/backend` contains no `.env`; only `.env.example` at repo root.

## 2. Backend configuration (env vars)

| Var | Default | Notes |
|---|---|---|
| `APP_ENV` | `development` | `development` or `production` |
| `PORT` | `8080` | HTTP listen port |
| `GIN_MODE` | `debug` | |
| `DATABASE_URL` | `postgres://cordis:cordis@localhost:5432/cordis?sslmode=disable` | |
| `REDIS_URL` | `redis://localhost:6379/0` | |
| `JWT_SECRET` | `change-me-in-production-min-32-chars` | HS256, min 32 chars enforced in prod |
| `ACCESS_TOKEN_TTL` | `15m` | Go duration |
| `REFRESH_TOKEN_TTL` | `720h` | 30 days |
| `CORS_ORIGINS` | `http://localhost:1420,http://localhost:5173` | comma separated |
| `PUBLIC_URL` | `http://localhost:8080` | base for generated links |
| `LIVEKIT_URL` | `ws://localhost:7880` | |
| `LIVEKIT_API_KEY` | `devkey` | |
| `LIVEKIT_API_SECRET` | `secret` | |
| `UPLOAD_DRIVER` | `local` | `local` or `s3` |
| `UPLOAD_DIR` | `./data/uploads` | local driver root |
| `UPLOAD_PUBLIC_BASE` | `http://localhost:8080/uploads` | |
| `MAX_UPLOAD_MB` | `25` | |
| `S3_ENDPOINT` | empty | e.g. `https://s3.example.com` |
| `S3_REGION` | `us-east-1` | |
| `S3_BUCKET` | `cordis` | |
| `S3_ACCESS_KEY` | empty | |
| `S3_SECRET_KEY` | empty | |
| `S3_FORCE_PATH_STYLE` | `true` | |
| `REDIS_CHANNEL` | `cordis:events` | pub/sub channel |
| `TRUSTED_PROXIES` | empty | comma separated CIDRs |
| `MAX_MESSAGE_LEN` | `4000` | |
| `SELF_HOSTED_DOMAIN` | `http://localhost:5173` | client URL for LiveKit |

## 3. HTTP conventions

* Base path `/api`; content type `application/json; charset=utf-8`.
* Auth: `Authorization: Bearer <access_token>`; WebSocket may use `?token=`.
* CORS allows configured origins, methods `GET,POST,PATCH,PUT,DELETE,OPTIONS`, headers
  `Authorization,Content-Type`, exposes `Content-Length`, max-age 86400, handles preflight.
* No response envelope: handlers return the resource object directly.
* Errors always exactly this shape:

```json
{ "error": { "code": "validation_error", "message": "content is required", "fields": { "content": "required" } } }
```

`fields` omitted when empty. Codes: `validation_error`, `unauthorized`, `forbidden`,
`not_found`, `conflict`, `internal_error`, `payload_too_large`, `unsupported_media_type`,
`rate_limited`.

* Status: 201 register, create server, create channel, create message, create reaction, create
  invite, upload, create friend; 200 login, refresh, updates, lists, join invite, livekit token;
  204 logout, delete message, delete reaction, delete friend, delete server, delete channel,
  delete invite, delete DM.
  400 validation_error, 401 unauthorized, 403 forbidden, 404 not_found, 409 conflict,
  413 payload_too_large, 415 unsupported_media_type, 429 rate_limited, 500 internal_error.
* Validation done manually per handler, returning a `fields` map.
* Structured JSON request logs with `request_id`, `method`, `path`, `status`, `latency_ms`,
  `user_id`; `X-Request-ID` response header echoing the incoming value or a new uuid.

## 4. JSON resource shapes (camelCase keys, RFC3339 timestamps)

All IDs are UUIDv4 strings.

```ts
type Status = "online" | "idle" | "dnd" | "offline" | "invisible";

interface User {
  id: string; username: string; displayName: string; email: string;
  avatarUrl: string | null; bannerUrl: string | null; bio: string;
  status: Status; customStatus: string; createdAt: string; updatedAt: string;
}

interface Role {
  id: string; serverId: string; name: string; color: string;   // "#rrggbb"
  permissions: number;   // bitmask, see section 7
  hoist: boolean; position: number;
}

interface Member {
  id: string; serverId: string; user: User; nickname: string;
  roleIds: string[]; joinedAt: string;
}

interface Channel {
  id: string; serverId: string; parentId: string | null;
  name: string; topic: string; type: "text" | "voice" | "video" | "category";
  position: number; createdAt: string; updatedAt: string;
}

interface Attachment {
  id: string; url: string; filename: string; size: number;
  contentType: string; width: number | null; height: number | null;
}

interface Reaction {
  id: string; messageId: string; userId: string; emoji: string; createdAt: string;
}

interface Message {
  id: string; channelId: string; author: User; content: string;
  type: "text" | "image" | "file" | "system";
  replyTo: Message | null; attachments: Attachment[]; reactions: Reaction[];
  editedAt: string | null; createdAt: string;
}

interface Server {
  id: string; name: string; description: string; iconUrl: string | null;
  bannerUrl: string | null; ownerId: string; createdAt: string; updatedAt: string;
}

interface DMChannel {
  id: string; createdAt: string; updatedAt: string;
  recipients: User[]; lastMessage: Message | null;
}

interface Friend {
  id: string; status: "pending" | "accepted" | "blocked";
  user: User;              // the *other* participant
  createdAt: string;
}

interface Invite {
  code: string; server: Server; channelId: string | null;
  inviter: User; expiresAt: string | null; uses: number; maxUses: number; createdAt: string;
}

interface Paginated<T> { items: T[]; hasMore: boolean; nextCursor: string | null; }

interface ServerDetail {
  server: Server; channels: Channel[]; members: Member[]; roles: Role[];
  owner: User; unreadCount: number; invites: Invite[];
}
```

* `GET /servers` returns `Paginated<Server>` containing only servers the user belongs to.
* `GET /users/me`, `GET /users/:id`, `PATCH /users/me` return a `User` object.
* `GET /users?q=&limit=` returns `Paginated<User>`.
* Message pages are ordered oldest to newest inside `items`.
* Reactions are always included in every returned Message payload.
* Lists that are not paginated are bare JSON arrays: `Invite[]`, `Reaction[]`, `User[]`.

## 5. Endpoints (exact)

### Auth
| Method | Path | Body | Success |
|---|---|---|---|
| POST | `/api/auth/register` | `{username,email,password,displayName?}` | 201 `{user,accessToken,refreshToken,expiresIn}` |
| POST | `/api/auth/login` | `{login,password}` where login is username or email | 200 same as register |
| POST | `/api/auth/refresh` | `{refreshToken}` | 200 `{accessToken,refreshToken,expiresIn}` |
| POST | `/api/auth/logout` | `{refreshToken}` | 204 |

`expiresIn` is the access-token lifetime in seconds.
Validation: `username` 3 to 32 chars matching `^[a-z0-9_.]+$`, lowercased, unique; `email` valid,
max 254 chars, lowercased, unique; `password` 8 to 128 chars with at least one letter and one
digit; `displayName` optional 1 to 64 chars defaulting to the username. Duplicate username or
email yields 409.

### Users
| Method | Path | Notes |
|---|---|---|
| GET | `/api/users/me` | returns `User` |
| PATCH | `/api/users/me` | `{displayName?,bio?,avatarUrl?,bannerUrl?,status?,customStatus?}` returns `User` |
| GET | `/api/users/:id` | returns `User` |
| GET | `/api/users?q=&limit=` | returns `Paginated<User>` |


### Servers and channels
| Method | Path | Body | Success |
|---|---|---|---|
| GET | `/api/servers` | none | 200 `Paginated<Server>` |
| POST | `/api/servers` | `{name,description?}` | 201 `ServerDetail`; creator becomes owner, an `@everyone` Role is created, plus text channel `general` and voice channel `voice` |
| GET | `/api/servers/:id` | none | 200 `ServerDetail`, 403 when not a member |
| PATCH | `/api/servers/:id` | `{name?,description?,iconUrl?,bannerUrl?}` | 200 `Server` |
| DELETE | `/api/servers/:id` | none | 204, owner only |
| POST | `/api/servers/:id/channels` | `{name,type?,topic?,parentId?}` | 201 `Channel` |
| PATCH | `/api/channels/:id` | `{name?,topic?,position?,parentId?}` | 200 `Channel` |
| DELETE | `/api/channels/:id` | none | 204 |
| GET | `/api/servers/:id/invites` | none | 200 `Invite[]`, owner only |
| POST | `/api/servers/:id/invites` | `{channelId?,maxUses?,expiresInHours?}` | 201 `Invite` |
| DELETE | `/api/invites/:code` | none | 204 |
| GET | `/api/invites/:code` | none, public | 200 `Invite` |
| POST | `/api/invites/:code/join` | none | 200 `ServerDetail`, requires auth |

`channel.type` is one of `text`, `voice`, `video`, `category`. `name` is 1 to 64 chars matching
`^[a-z0-9-_]{1,64}$`. Invite `code` is 10 lowercase alphanumeric characters.

### Messages
| Method | Path | Body or query | Success |
|---|---|---|---|
| GET | `/api/channels/:id/messages` | `?before=&after=&limit=` where limit is 1 to 100, default 50 | 200 `Paginated<Message>` |
| POST | `/api/channels/:id/messages` | `{content,replyToId?,attachments?}` | 201 `Message` |
| PATCH | `/api/messages/:id` | `{content}` | 200 `Message`, author only, sets `editedAt` |
| DELETE | `/api/messages/:id` | none | 204, author or ManageMessages |

`AttachmentInput` is `{url,filename,size,contentType,width?,height?}`.
Content is trimmed, 1 to MAX_MESSAGE_LEN. Empty content is allowed only when `attachments`
is non-empty.

### Reactions
| Method | Path | Body | Success |
|---|---|---|---|
| POST | `/api/messages/:id/reactions` | `{emoji}` | 201 `Reaction`; toggles, so an existing reaction is deleted and 204 returned |
| DELETE | `/api/messages/:id/reactions` | `?emoji=` or `?userId=` | 204 |
| GET | `/api/messages/:id/reactions` | none | 200 `Reaction[]` |

Emoji must be 1 to 16 unicode runes. Invalid UTF-8 is rejected with `validation_error`.

### Direct messages
| Method | Path | Body | Success |
|---|---|---|---|
| GET | `/api/dms` | none | 200 `Paginated<DMChannel>` |
| POST | `/api/dms` | `{recipientId}` | 200 `DMChannel`, idempotent, returns the existing DM |
| GET | `/api/dms/:id/messages` | `?before=&after=&limit=` | 200 `Paginated<Message>` |
| POST | `/api/dms/:id/messages` | `{content,attachments?}` | 201 `Message` |
| DELETE | `/api/dms/:id` | none | 204, removes the caller from the DM |
| GET | `/api/dms/:id/recipients` | none | 200 `User[]` |

### Friends
| Method | Path | Body | Success |
|---|---|---|---|
| GET | `/api/friends` | none | 200 `{friends, incoming, outgoing}`, all `Friend[]` |
| POST | `/api/friends` | `{userId}` | 201 `Friend` with status `pending`, 200 when the row already exists |
| PATCH | `/api/friends/:id` | `{status:"accepted"}` | 200 `Friend`, both sides updated |
| DELETE | `/api/friends/:id` | none | 204 |

Self request yields 400, unknown user 404. Friend rows are directional, requester to addressee.
Inside `friends` the `user` field is always the other party.

### LiveKit
`POST /api/livekit/token` with body `{roomName,roomType:"channel"|"dm",canPublish?,canSubscribe?}`
returns 200 `{token,url,roomName,expiresIn}`.

`roomName` must be `channel:<uuid>` or `dm:<uuid>` and membership is verified, otherwise 403.
Voice and video channels grant `canPublish:true`, text channels grant `canPublish:false`.
Video grants: `roomJoin`, `room` for create, `canPublish`, `canSubscribe`, `canPublishData`.
`expiresIn` is 3600 seconds.


### Upload
`POST /api/upload` accepts `multipart/form-data` with field `file` plus optional `width` and
`height` integers, and returns 201 `Attachment`.

The maximum size is MAX_UPLOAD_MB. Allowed content types are `image/png`, `image/jpeg`,
`image/gif`, `image/webp`, `application/pdf`, `text/plain`, `application/zip`, `video/mp4`,
`video/webm`, `application/x-msdownload`, `application/x-msi`,
`application/vnd.microsoft.portable-executable`, `application/x-7z-compressed`,
`application/vnd.rar`, `application/x-msdoc`,
`application/vnd.openxmlformats-officedocument.wordprocessingml.document`,
`application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` and
`application/vnd.openxmlformats-officedocument.presentationml.presentation`.
A non-multipart request yields 415, oversize yields 413. Stored files are served from
`GET /uploads/*filepath`. The local driver writes a uuid filename under UPLOAD_DIR preserving the
original extension, and the S3 driver uses minio-go and returns a public or presigned URL.

### Health
`GET /healthz` returns 200 `{"status":"ok","uptimeSeconds":n,"version":"..."}`.
`GET /readyz` returns 200 only when Postgres and Redis both ping successfully, otherwise 503
`{"status":"degraded","checks":{"postgres":"ok|down","redis":"ok|down"}}`.

## 6. WebSocket `/ws` also served at `/api/ws`

* Authenticated through `?token=<access>` or the `Authorization` header. An auth failure closes
  the connection before the upgrade completes.
* No subprotocol. Ping every 25s, pong deadline 60s, write deadline 10s, read limit 64 KiB.
* The envelope is identical in both directions:
  `{ "event": "message.create", "data": { ... }, "nonce": "optional" }`.
* On connect the server immediately sends:

```json
{"event":"ready","data":{"userId":"...","sessionId":"...","serverTime":"2024-01-01T00:00:00Z"}}
```

### Client to server
| event | data |
|---|---|
| `message.create` | `{channelId,content,replyToId?,attachments?}` |
| `message.update` | `{id,content}` |
| `message.delete` | `{id}` |
| `typing.start` | `{channelId}` |
| `typing.stop` | `{channelId}` |
| `presence.update` | `{status:"online"\|"idle"\|"dnd"\|"invisible",customStatus?}` |
| `friend.request` | `{userId}` |
| `friend.accepted` | `{friendId}` |
| `heartbeat` | `{}` |

### Server to client
| event | data |
|---|---|
| `ready` | `{userId,sessionId,serverTime}` |
| `message.created` | `{message,channelId,dmId}` |
| `message.updated` | `{message,channelId}` |
| `message.deleted` | `{id,channelId,deletedAt}` |
| `typing.started` | `{channelId,user}` |
| `typing.stopped` | `{channelId,userId}` |
| `presence.updated` | `{userId,status,customStatus,at}` |
| `friend.request` | `{friend}` |
| `friend.accepted` | `{friend}` |
| `error` | `{code,message,nonce}` |

* Handler failures produce an `error` event instead of a dropped connection.
* `message.*` and `typing.*` broadcast to every online socket belonging to the members of the
  channel's server, or to the DM participants. Typing events exclude the originating socket.
* The `nonce` of an inbound frame is echoed on the matching outbound `error` frame.

## 7. Permissions bitmask, `int64` in Go and `number` in TypeScript

```
1      CreateInstantInvite      1024   ViewChannel
2      KickMembers              2048   SendMessages
4      BanMembers               4096   SendTTSMessages
8      Administrator            8192   ManageMessages
16     ManageChannels           16384  EmbedLinks
32     ManageServer             32768  AttachFiles
64     AddReactions             65536  ReadMessageHistory
128    ViewAuditLog             131072 MentionEveryone
256    PrioritySpeaker          262144 UseExternalEmojis
512    Stream                   524288 ViewServerInsights
```

The `@everyone` role receives `ViewChannel|SendMessages|ReadMessageHistory|AddReactions|
AttachFiles|EmbedLinks` and can never hold `Administrator`. The server owner's member role has
`Administrator`. `Can(member, perm)` is true when the member owns the server, when any of the
member's roles including `@everyone` has the bit, or when the member has `Administrator`.

## 8. Redis key layout

| Key | Type | TTL | Purpose |
|---|---|---|---|
| `cordis:refresh:<jti>` | string holding the userID | REFRESH_TOKEN_TTL | refresh rotation and revocation |
| `cordis:presence:<userID>` | hash of `status`, `customStatus`, `at` | 5m, refreshed by heartbeat | cross-replica presence |
| `cordis:online:<userID>` | string | 5m | online marker |
| `cordis:typing:<channelID>` | sorted set scored by expiry ms | 10s | typing dedupe |
| `cordis:events` | pub/sub | none | WS envelope fan-out across replicas |
| `cordis:rate:<scope>:<id>` | counter | 60s | rate limiting |


## 9. Backend Go layout, mandatory

```
backend/
  go.mod  Dockerfile  .dockerignore  Makefile  README.md
  cmd/server/main.go
  internal/
    config/config.go            env parsing and validation
    logger/logger.go            zerolog plus request id helpers
    database/database.go        GORM open, retry, automigrate
    cache/redis.go              client, pub/sub, presence
    models/                     user.go server.go channel.go message.go reaction.go
                                directmessage.go friend.go role.go member.go invite.go attachment.go
    auth/jwt.go                 access and refresh issue, verify, rotation
    auth/password.go            bcrypt
    middleware/                 auth.go cors.go logger.go recovery.go ratelimit.go requestid.go
    httpx/errors.go             APIError with RespondJSON, RespondError, BindJSON
    handlers/                   auth.go users.go servers.go channels.go messages.go reactions.go
                                dms.go friends.go livekit.go upload.go health.go invites.go
    permissions/permissions.go  bitmask, Compute, Can
    ws/                         hub.go client.go events.go registry.go presence.go handlers.go
    storage/storage.go          Driver interface with local.go and s3.go
    livekit/token.go
    service/                    auth.go user.go server.go channel.go message.go reaction.go
                                dm.go friend.go invite.go presence.go access.go
```

Rules: exported functions carry doc comments, handlers never call GORM directly because they use
services, services never touch `*gin.Context`, there is no `panic` outside recovery middleware,
and DB errors are wrapped with `fmt.Errorf("...: %w", err)`.

## 10. Desktop

### 10.1 Configuration files
* `package.json` with scripts `dev`, `build`, `preview`, `tauri`, `typecheck`, `lint`.
* `vite.config.ts` using the `@tauri-apps/api/vite` plugin, port 1420, `strictPort`,
  `clearScreen:false`, alias `@` to `./src`, and a dev-only proxy for `/api` and `/uploads`
  pointing at `http://127.0.0.1:8080`.
* `tsconfig.json` and `tsconfig.node.json` with `strict`, `noUnusedLocals`,
  `noUnusedParameters` and `paths` mapping `@/*` to `./src/*`.
* `tailwind.config.ts` with `darkMode:["class"]` and shadcn CSS variables,
  `postcss.config.js`, `components.json` with shadcn aliases, `index.html`, `.env.example`.
* `src-tauri/**` as described in section 11.


Dependencies: `react@^18.3.1`, `react-dom@^18.3.1`, `react-router-dom@^6.26.2`,
`@tanstack/react-query@^5.56.2`, `zustand@^4.5.5`, `date-fns@^3.6.0`, `clsx`, `tailwind-merge`,
`class-variance-authority`, `lucide-react`, `sonner`, `@radix-ui/react-slot`,
`@radix-ui/react-dialog`, `@radix-ui/react-dropdown-menu`, `@radix-ui/react-popover`,
`@radix-ui/react-tooltip`, `@radix-ui/react-avatar`, `@radix-ui/react-scroll-area`,
`@radix-ui/react-separator`, `@radix-ui/react-switch`, `@radix-ui/react-tabs`,
`@radix-ui/react-toast`, `@radix-ui/react-select`, `@radix-ui/react-checkbox`,
`@radix-ui/react-label`, `@radix-ui/react-progress`, `@radix-ui/react-slider`,
`@livekit/components-react@^2.5.0`, `@livekit/components-styles`, `livekit-client@^2.5.0`,
`@tauri-apps/api@^2`, `@tauri-apps/plugin-dialog`, `@tauri-apps/plugin-notification`,
`@tauri-apps/plugin-updater`, `@tauri-apps/plugin-process`, `@tauri-apps/plugin-os`,
`@tauri-apps/plugin-autostart`, `@tauri-apps/plugin-store`.
Dev dependencies: `vite@^5.4.8`, `@vitejs/plugin-react@^4.3.1`, `typescript@~5.5.4`,
`tailwindcss@^3.4.13`, `postcss`, `autoprefixer`, `tailwindcss-animate`, `@types/react`,
`@types/react-dom`, `@types/node`.

### 10.2 Mandatory files under `desktop/src`

```
src/main.tsx  src/App.tsx  src/index.css  src/vite-env.d.ts
src/types/index.ts
src/lib/utils.ts
src/lib/api/client.ts  auth.ts users.ts servers.ts channels.ts messages.ts reactions.ts
                       dms.ts friends.ts livekit.ts upload.ts
src/lib/ws/client.ts  events.ts
src/lib/tauri/desktop.ts  window.ts
src/stores/auth.store.ts settings.store.ts servers.store.ts messages.store.ts
       presence.store.ts voice.store.ts
src/components/ui/button.tsx input.tsx label.tsx textarea.tsx dialog.tsx dropdown-menu.tsx
       popover.tsx tooltip.tsx avatar.tsx scroll-area.tsx separator.tsx switch.tsx tabs.tsx
       toast.tsx toaster.tsx select.tsx checkbox.tsx progress.tsx slider.tsx badge.tsx
       skeleton.tsx alert.tsx
src/components/layout/AppShell.tsx Titlebar.tsx ServerRail.tsx ChannelSidebar.tsx
       ChatView.tsx MembersPanel.tsx
src/components/auth/LoginScreen.tsx RegisterScreen.tsx
src/components/servers/ServerList.tsx CreateServerDialog.tsx
src/components/channels/ChannelList.tsx ChannelHeader.tsx CreateChannelDialog.tsx
src/components/chat/MessageList.tsx MessageItem.tsx MessageInput.tsx ReplyPreview.tsx
       AttachmentList.tsx ReactionBar.tsx EditMessageDialog.tsx EmojiPicker.tsx
       TypingIndicator.tsx DateDivider.tsx JumpToBottomButton.tsx
src/components/members/MemberList.tsx MemberItem.tsx
src/components/dm/DmList.tsx DmView.tsx NewDmDialog.tsx
src/components/voice/VoicePanel.tsx JoinVoiceDialog.tsx VideoCallGrid.tsx CallControls.tsx
src/components/settings/SettingsDialog.tsx ProfileSection.tsx AppearanceSection.tsx
       ConnectionSection.tsx NotificationSection.tsx AccountSection.tsx
src/components/common/ContextMenu.tsx ConfirmDialog.tsx
src/hooks/useAuth.ts useServers.ts useMessages.ts usePresence.ts useFriends.ts useTyping.ts
       useVoice.ts useHotkeys.ts useTheme.ts useLiveKitToken.ts
src/routes/AuthRoute.tsx AppRoute.tsx
src/pages/LoginPage.tsx RegisterPage.tsx AppPage.tsx
```

Router paths are `/login`, `/register`, protected `/channels/:channelId?`, protected `/dm/:dmId`
and protected `/settings`. `AuthRoute` redirects to `/login` when unauthenticated and
`AppRoute` redirects to `/login` when authenticated.

### 10.3 Frontend behaviour
* Infinite scroll uses `useInfiniteQuery` per channel with `getNextPageParam` equal to the oldest
  `items[0].id`, a 300px threshold, preserved scroll height on prepend, a sticky jump-to-latest
  button, and auto-stick only when already at the bottom.
* The message input sends on Enter and inserts a newline on Shift+Enter, driven by settings.
  Typing events throttle to one `typing.start` every 3s plus a matching stop. Reply, edit and
  delete use optimistic updates with rollback. Uploads go through the Tauri file dialog into
  `POST /api/upload`, and drag and drop is also supported.
* Optimistic sends use a temporary id of `tmp_<uuid>` replaced by `message.created`.
* Presence derives from `presence.updated`, and the WebSocket heartbeats every 25s.
* Native Tauri notifications fire for direct messages and mentions, never for own messages.
* Voice uses `useLiveKitToken` then `livekit-client` with a `Room`, offering mic, camera and
  screen-share toggles. `VideoCallGrid` uses `@livekit/components-react` and reconnect is automatic.
* Theme supports `dark`, `light` and `system` by toggling the class on `document.documentElement`
  and persists the choice.
* Settings under Connection expose editable `apiBaseUrl` and `wsUrl`, persisted, defaulting to
  `import.meta.env.VITE_API_BASE_URL` and `import.meta.env.VITE_WS_URL`.

`desktop/.env.example` contains `VITE_API_BASE_URL=http://127.0.0.1:8080`,
`VITE_WS_URL=ws://127.0.0.1:8080/ws` and `VITE_APP_NAME=Cordis`.


## 11. Tauri v2 under `desktop/src-tauri`

```
Cargo.toml  build.rs  tauri.conf.json  capabilities/default.json  icons/*
src/main.rs  lib.rs  tray.rs  notifications.rs  updater.rs  window.rs  commands.rs
```

* `tauri.conf.json` uses `productName:"Cordis"`, `identifier:"com.cordis.desktop"`,
  `frontendDist:"../dist"`, `devUrl:"http://localhost:1420"`, `beforeDevCommand:"npm run dev"`,
  `beforeBuildCommand:"npm run build"` and `decorations:false`. The window uses
  `titleBarStyle:"Overlay"`, `hiddenTitle:true`, `width:1280`, `height:800`, `minWidth:900`,
  `minHeight:600`, `resizable:true`, `center:true` and `dragDropEnabled:true`.
  The bundle has `active:true` and `targets:["msi","nsis"]`, `nsis.installMode:"perMachine"`,
  `wix.language:["en-US"]` and `webviewInstallMode:{type:"downloadBootstrapper"}`.
  The updater plugin endpoint is
  `https://releases.cordis.dev/desktop/{{target}}/{{arch}}/{{current_version}}`
  with a documented `pubkey` placeholder.
* `capabilities/default.json` grants `core:default`, window drag, minimize, maximize, close,
  hide and set-focus, `core:event:default`, `core:tray:default`, `core:webview:default`,
  `core:menu:default`, `dialog:default`, `notification:default`, `updater:default`,
  `os:default`, `process:default`, `autostart:default` and `store:default`.
* `lib.rs` exposes `run()` which registers a tray with Mute, Deafen, a Status submenu containing
  Online, Idle, Do Not Disturb and Invisible, plus Show Cordis and Quit. Tray actions emit a
  `tray://action` event carrying `{action,value}` and a left click toggles window visibility.
* `commands.rs` provides `app_version()`, `set_launch_on_boot(bool)`, `open_external(url)`,
  `copy_to_clipboard(text)` and `device_id()`.
* Real icon PNG and ICO files must exist and be referenced by `tauri.conf.json`.
* No `unwrap()` outside `main` and no `TODO` comments.

## 12. Docker

`docker-compose.yml` defines `postgres` on 16-alpine with a `pg_isready` healthcheck, volume
`cordis_pgdata` and port 5432; `redis` on 7-alpine with a `redis-cli ping` healthcheck, volume
`cordis_redisdata` and port 6379; `livekit-server` running
`livekit/livekit-server --dev --bind 0.0.0.0` on ports 7880, 7881 and 7882; and `backend`
built from the multi-stage Dockerfile, depending on the healthy services, published on 8080 and
configured from `.env`.

`backend/Dockerfile` builds with `golang:1.23-alpine` running
`CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cordis ./cmd/server`, and
runs on `alpine:3.20` with `ca-certificates` and `tzdata`, a non-root `cordis` user owning
`/app/data/uploads`, `EXPOSE 8080`, a `HEALTHCHECK` using
`wget -qO- http://127.0.0.1:8080/healthz` and `ENTRYPOINT ["/usr/local/bin/cordis"]`.

`.env.example` at the repo root lists every variable from section 2 plus `POSTGRES_USER`,
`POSTGRES_PASSWORD`, `POSTGRES_DB` and `TZ`.

## 13. Quality bar

No `TODO`, no `FIXME`, no placeholder ellipses and no commented-out code. Comments appear only
where they explain why something is done. Paths must be Windows-safe. Every package stays
internally consistent with this document.

