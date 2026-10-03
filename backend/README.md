# Cordis backend

Go 1.23 API, WebSocket gateway, LiveKit token issuer and upload service for
Cordis. The module implements `CONTRACT.md` sections 2 to 9 exactly: the same
environment variables, JSON shapes, status codes, WebSocket events, permission
bitmask and Redis key layout.

## Layout

```
cmd/server/main.go          process wiring, middleware chain, route table
internal/config             env parsing and validation
internal/logger             zerolog plus request id helpers
internal/database           GORM open, retry, automigrate
internal/cache              Redis client, keys, pub/sub
internal/models             GORM entities with camelCase JSON
internal/auth               JWT issue, verify, refresh rotation, bcrypt
internal/middleware         request id, logging, recovery, CORS, auth, rate limit
internal/httpx              APIError, RespondJSON, RespondError, DecodeJSON
internal/permissions        bitmask, Compute, Can
internal/service            business logic, never touches gin
internal/handlers           gin transport layer
internal/ws                 hub, client, registry, dispatcher, presence
internal/storage            Driver interface with local and S3 implementations
internal/livekit            video access token minting
```

Layering is strict: `handlers` -> `service` -> `models`. Only `service` touches
GORM, and only `service` emits events, so an HTTP request and a WebSocket frame
produce byte identical `message.created` payloads.

## Environment variables

| Var | Default | Notes |
|---|---|---|
| `APP_ENV` | `development` | `development` or `production` |
| `PORT` | `8080` | HTTP listen port |
| `GIN_MODE` | `debug` | gin mode |
| `DATABASE_URL` | `postgres://cordis:cordis@localhost:5432/cordis?sslmode=disable` | Postgres DSN |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis DSN |
| `JWT_SECRET` | `change-me-in-production-min-32-chars` | HS256, 32 chars enforced in production |
| `ACCESS_TOKEN_TTL` | `15m` | Go duration |
| `REFRESH_TOKEN_TTL` | `720h` | refresh lifetime |
| `CORS_ORIGINS` | `http://localhost:1420,http://localhost:5173` | comma separated |
| `PUBLIC_URL` | `http://localhost:8080` | base for generated links |
| `LIVEKIT_URL` | `ws://localhost:7880` | |
| `LIVEKIT_API_KEY` | `devkey` | |
| `LIVEKIT_API_SECRET` | `secret` | |
| `UPLOAD_DRIVER` | `local` | `local` or `s3` |
| `UPLOAD_DIR` | `./data/uploads` | local driver root |
| `UPLOAD_PUBLIC_BASE` | `http://localhost:8080/uploads` | |
| `MAX_UPLOAD_MB` | `25` | |
| `S3_ENDPOINT` | empty | required when `UPLOAD_DRIVER=s3` |
| `S3_REGION` | `us-east-1` | |
| `S3_BUCKET` | `cordis` | |
| `S3_ACCESS_KEY` | empty | required when `UPLOAD_DRIVER=s3` |
| `S3_SECRET_KEY` | empty | required when `UPLOAD_DRIVER=s3` |
| `S3_FORCE_PATH_STYLE` | `true` | |
| `REDIS_CHANNEL` | `cordis:events` | WebSocket fan-out channel |
| `TRUSTED_PROXIES` | empty | comma separated CIDRs |
| `MAX_MESSAGE_LEN` | `4000` | |
| `SELF_HOSTED_DOMAIN` | `http://localhost:5173` | client URL for LiveKit |

## Running

With Docker Compose at the repository root:

```
docker compose up --build
```

Locally, with Go 1.23 installed and Postgres and Redis reachable:

```
cd backend
go mod tidy      # generates go.sum from go.mod on a fresh checkout
go run ./cmd/server
```

The schema is created automatically by `AutoMigrate` on start, so no migration
step is required. Startup blocks until both Postgres and Redis answer.

Health probes:

## Generating an invite

An invite is minted by the server owner, then shared as
`PUBLIC_URL + /invite/<code>`.

```
# 1. Sign in and keep the access token
curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"login":"owner","password":"secret123"}'

# 2. Create the server (the creator becomes owner)
curl -s -X POST http://localhost:8080/api/servers \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Cordis HQ","description":"Where the work happens"}'

# 3. Mint an invite for that server, optionally limited and expiring
curl -s -X POST http://localhost:8080/api/servers/$SERVER_ID/invites \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"maxUses":25,"expiresInHours":168}'
```

The response carries the `code` and the join link. Anyone signed in can redeem
it with `POST /api/invites/<code>/join`, which adds them to the server with the
`@everyone` role and answers with the `ServerDetail` payload. `GET
/api/invites/<code>` is public so the link can be previewed before signing in.

Invite codes are 10 lowercase alphanumeric characters generated from
`crypto/rand`.

## API examples

Register and authenticate:

```
curl -s -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"ada","email":"ada@example.com","password":"secret123"}'
```

Send a message over HTTP:

```
curl -s -X POST http://localhost:8080/api/channels/$CHANNEL_ID/messages \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"content":"Ship it"}'
```

Subscribe to the WebSocket, which is served at `/ws` and `/api/ws`:

```
websocat "ws://localhost:8080/ws?token=$ACCESS_TOKEN"
```

The server answers with `ready`, then accepts client frames:

```json
{"event":"message.create","data":{"channelId":"<uuid>","content":"hello"},"nonce":"n1"}
```

and emits server frames:

```json
{"event":"message.created","data":{"message":{},"channelId":"<uuid>","dmId":null}}
{"event":"error","data":{"code":"validation_error","message":"content is required","nonce":"n1"}}
```

A handler failure answers with an `error` event echoing the inbound `nonce`
rather than closing the socket. Typing events exclude the originating socket.

## Uploads

```
curl -s -X POST http://localhost:8080/api/upload \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -F "file=@screenshot.png"
```

The endpoint accepts `multipart/form-data` with the field `file` plus optional
`width` and `height`. A non multipart request yields 415 and an oversized file
yields 413. With the local driver the file is stored under `UPLOAD_DIR` with a
uuid name and served from `GET /uploads/<filename>`.

## Build

```
make build          # bin/cordis
make docker-build   # multi stage image
```

The Docker image produces a static binary, runs as a non-root `cordis` user and
ships a `HEALTHCHECK` against `/healthz`.
```
GET /healthz   200 {"status":"ok","uptimeSeconds":n,"version":"..."}
GET /readyz    200 when Postgres and Redis both ping, otherwise 503 with the checks map
```