# Cordis

A Discord-style chat platform shipped as a **native Windows desktop application**.

| Layer | Technology |
|---|---|
| Desktop shell | Tauri v2 (Rust) — WebView2, system tray, native notifications, auto-update, custom titlebar |
| Frontend | React 18 + TypeScript + Vite + Tailwind CSS + shadcn/ui + Zustand + TanStack Query + React Router |
| Backend | Go 1.22 — Gin, Gorilla WebSocket, GORM, JWT, bcrypt |
| Realtime fan-out | Redis 7 Pub/Sub (multi-replica safe) |
| Voice / Video | LiveKit (self-hosted) |
| Database | PostgreSQL 16 |
| Storage | Local filesystem or any S3-compatible provider |

```
.
├── backend/                 Go API, WebSocket hub, LiveKit token minting, uploads
│   ├── cmd/server/          entrypoint
│   └── internal/            config, models, auth, handlers, services, ws, storage, livekit
├── desktop/                 Tauri v2 + React app
│   ├── src/                 React frontend
│   └── src-tauri/           Rust shell (tray, updater, window, notifications)
├── docker-compose.yml       backend + postgres + redis + livekit-server
├── .env.example             every backend variable, documented
└── CONTRACT.md              authoritative API / event / resource specification
```

---

## Table of contents

1. [Prerequisites](#1-prerequisites)
2. [Quick start with Docker](#2-quick-start-with-docker)
3. [Local development](#3-local-development)
4. [Building the Windows installer](#4-building-the-windows-installer)
5. [Code signing](#5-code-signing)
6. [Configuration reference](#6-configuration-reference)
7. [Auto-update setup](#7-auto-update-setup)
8. [Production deployment](#8-production-deployment)
9. [Architecture notes](#9-architecture-notes)
10. [Troubleshooting](#10-troubleshooting)
11. [License](#11-license)


## 1. Prerequisites

### For the full Docker stack
* **Docker Desktop for Windows** (Linux containers mode) — the only hard requirement.
* Ports free on your machine: `8080` (API), `5432` (Postgres), `6379` (Redis),
  `7880`/`7881`/`7882` (LiveKit).

### For local development
| Tool | Version | Purpose |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.22+ | backend |
| [Node.js](https://nodejs.org/) | 20 LTS or newer | frontend |
| [Rust](https://rustup.rs) | stable, `*-msvc` toolchain | Tauri shell |
| [Microsoft C++ Build Tools](https://visualstudio.microsoft.com/visual-cpp-build-tools/) | "Desktop development with C++" workload | required by Rust on Windows |
| [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) | preinstalled on Windows 10 21H2+ / Windows 11 | rendering |
| PostgreSQL | 16 | database |
| Redis | 7 | cache and pub/sub |

LiveKit is optional for pure text development; the voice and video UI degrades gracefully and the
`/api/livekit/token` endpoint returns `503` if LiveKit is unreachable. To run it locally without
Docker, download `livekit-server` and run `livekit-server --dev --bind 0.0.0.0`.

---

## 2. Quick start with Docker

```powershell
git clone <your-repo-url> cordis
cd cordis
Copy-Item .env.example .env      # PowerShell equivalent of `cp`
```

Edit `.env` and set at least:

```dotenv
JWT_SECRET=<64 random characters>          # openssl rand -base64 48
POSTGRES_PASSWORD=<a strong password>
LIVEKIT_API_KEY=devkey
LIVEKIT_API_SECRET=secret
```

Then:

```powershell
docker compose up -d --build
docker compose ps          # wait until backend reports healthy
Invoke-RestMethod http://localhost:8080/readyz
```

`/readyz` returns `{"status":"ok"}` only once Postgres **and** Redis are both reachable. The backend
runs GORM auto-migrations on boot, so the schema is created automatically on first start.

### Smoke test the API

```powershell
$body = @{
  username    = 'ada'
  email       = 'ada@example.com'
  password    = 'lovelace1843'
  displayName = 'Ada Lovelace'
} | ConvertTo-Json

$session = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/auth/register `
  -ContentType 'application/json' -Body $body

$headers = @{ Authorization = "Bearer $($session.accessToken)" }

# Create a server
$server = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/servers `
  -Headers $headers -ContentType 'application/json' `
  -Body (@{ name = 'Analytical Engines' } | ConvertTo-Json)

# Send a message to the default #general channel
$channel = $server.channels | Where-Object { $_.type -eq 'text' } | Select-Object -First 1

Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/channels/$($channel.id)/messages" `
  -Headers $headers -ContentType 'application/json' `
  -Body (@{ content = 'Hello from PowerShell!' } | ConvertTo-Json)
```

To stop everything, keeping the data:

```powershell
docker compose down          # keeps data
docker compose down -v       # deletes Postgres, Redis and uploaded files
```

---

## 3. Local development

### 3.1 Start PostgreSQL, Redis and LiveKit

The simplest option is to run only the stateful services in Docker and everything else natively:

```powershell
docker compose up -d postgres redis livekit-server
```

Or install them locally and point `DATABASE_URL` / `REDIS_URL` at `localhost`.

### 3.2 Backend

```powershell
cd backend
Copy-Item ..\.env.example .env          # then set DATABASE_URL/REDIS_URL to localhost
go mod download
go run ./cmd/server
```

The API listens on `http://127.0.0.1:8080`. Configuration comes from the environment, and on
Windows a `.env` file in the `backend` directory is loaded automatically when present.

```powershell
go run ./cmd/server                          # start
go build -o bin/cordis.exe ./cmd/server      # build a Windows binary
go vet ./...                                 # static analysis
gofmt -l .                                   # formatting check
```

Auto-migrations run on every start. To wipe local data, drop and recreate the schema:

```powershell
docker compose exec postgres psql -U cordis -d cordis -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
```

### 3.3 Desktop app

```powershell
cd desktop
Copy-Item .env.example .env.local
npm install
npm run tauri dev
```

`npm run tauri dev` starts Vite on `http://localhost:1420` and launches the Tauri window. Vite
proxies `/api` and `/uploads` to `http://127.0.0.1:8080` in development only, so the frontend can
use relative URLs during dev while production builds talk to a remote VPS.

For frontend-only iteration in a normal browser, run `npm run dev`. The app detects a non-Tauri
environment and falls back to browser notifications and a hidden `<input type="file">` for
attachments.

### 3.4 Pointing the app at a remote backend

Open **Settings → Connection** inside the app and set:

| Field | Example |
|---|---|
| API base URL | `http://34.165.143.31:8080` |
| WebSocket URL | `ws://34.165.143.31:8080/ws` |

These values are persisted locally and take effect immediately — no rebuild is required. Build-time
defaults come from `desktop/.env`:

```dotenv
VITE_API_BASE_URL=http://34.165.143.31:8080
VITE_WS_URL=ws://34.165.143.31:8080/ws
VITE_APP_NAME=Cordis
```

---

---

## 4. Building the Windows installer

Prerequisites: Node.js, Rust (`rustup default stable-msvc`), and the Visual Studio C++ build
tools. WebView2 is preinstalled on supported Windows versions, so no extra runtime is needed.

```powershell
cd desktop
npm install
npm run tauri build
```

Artifacts land in `desktop/src-tauri/target/release/bundle/`:

| Target | Output | Notes |
|---|---|---|
| MSI | `msi/Cordis_1.0.0_x64_en-US.msi` | enterprise deployment, `perMachine` install mode |
| NSIS | `nsis/Cordis_1.0.0_x64-setup.exe` | per-user friendly installer |

Useful variants:

```powershell
npm run tauri build -- --bundles nsis        # only the .exe installer
npm run tauri build -- --bundles msi         # only the MSI
npm run tauri build -- --target x86_64-pc-windows-msvc
npm run tauri build -- --no-bundle           # compile only, fastest for CI checks
```

Icons live in `desktop/src-tauri/icons/` and are referenced by `tauri.conf.json`. To use your own
artwork, drop in `32x32.png`, `128x128.png`, `128x128@2x.png`, `icon.ico` and `icon.png`, or
regenerate the full set with `npm run tauri icon path/to/logo.png`.

---

## 5. Code signing

Shipping an unsigned installer triggers SmartScreen warnings. Here is the intended workflow.

### 5.1 Obtain a certificate

Buy an **Authenticode** code-signing certificate (OV or EV). EV certificates require signing to
happen in the cloud (Azure Key Vault, DigiCert KeyLocker, or AWS KMS) via the `azure/trust` signer
tool. For self-signed distribution inside an organisation, generate one:

```powershell
New-SelfSignedCertificate `
  -Type CodeSigningCert `
  -Subject "CN=Cordis Internal, O=Cordis" `
  -CertStoreLocation "Cert:\CurrentUser\My" `
  -NotAfter (Get-Date).AddYears(3)

# Copy the thumbprint printed by the command above
$thumb = "YOUR_THUMBPRINT_HERE"

Export-PfxCertificate -Cert "Cert:\CurrentUser\My\$thumb" `
  -FilePath "$env:USERPROFILE\cordis-signing.pfx" `
  -Password (ConvertTo-SecureString -String "pfx-password" -AsPlainText -Force)
```

Keep the `.pfx` out of git.


### 5.2 Configure Tauri

Add a `bundle.windows.certificateThumbprint` entry to `desktop/src-tauri/tauri.conf.json`. The
values below are placeholders — replace them with your own:

```jsonc
{
  "bundle": {
    "windows": {
      "certificateThumbprint": "0000000000000000000000000000000000000000",
      "digestAlgorithm": "sha256",
      "timestampUrl": "http://timestamp.digicert.com"
    }
  }
}
```

When you use the Azure `trust` signer instead of a local certificate, keep the thumbprint entry out
of the config and drive signing from CI with a `.sign.ps1` step.

### 5.3 Sign manually (alternative)

If you prefer not to wire signing into Tauri, sign the built artifacts afterwards:

```powershell
$pfx = "$env:USERPROFILE\cordis-signing.pfx"
$password = ConvertTo-SecureString -String "pfx-password" -AsPlainText -Force
$thumb = "YOUR_THUMBPRINT_HERE"

Import-PfxCertificate -FilePath $pfx -CertStoreLocation Cert:\CurrentUser\My -Password $password

Get-ChildItem desktop\src-tauri\target\release\bundle -Recurse -Include *.msi,*.exe |
  ForEach-Object {
    Set-AuthenticodeSignature -FilePath $_.FullName `
      -Certificate (Get-ChildItem "Cert:\CurrentUser\My\$thumb") `
      -HashAlgorithm SHA256 `
      -TimestampServer http://timestamp.digicert.com
  }
```

Verify the result:

```powershell
Get-AuthenticodeSignature .\desktop\src-tauri\target\release\bundle\nsis\Cordis_1.0.0_x64-setup.exe |
  Format-List Status, StatusMessage, SignerCertificate
```

`Status` must read `Valid`.

### 5.4 CI signing notes

* Never commit the `.pfx` or its password. Use GitHub Actions secrets, or better, an Azure Key Vault
  with the `azure/trust` signer and federated credentials.
* Timestamp the signature (`timestampUrl`) or it becomes invalid once the certificate expires.
* Sign **after** bundling and **before** publishing release artifacts and updater manifests.

---

## 6. Configuration reference

Backend variables live in `.env.example` at the repo root. The most important ones:

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `DATABASE_URL` | postgres DSN | GORM connection string |
| `REDIS_URL` | `redis://redis:6379/0` | cache, refresh tokens, pub/sub |
| `JWT_SECRET` | dev placeholder | HS256 signing key, min 32 chars in production |
| `ACCESS_TOKEN_TTL` | `15m` | access token lifetime |
| `REFRESH_TOKEN_TTL` | `720h` | refresh token lifetime, rotating and revocable |
| `CORS_ORIGINS` | localhost:1420, localhost:5173 | allowed browser origins |
| `MAX_MESSAGE_LEN` | `4000` | hard message length cap |
| `LIVEKIT_URL` | `ws://localhost:7880` | LiveKit signalling URL the client connects to |
| `LIVEKIT_API_KEY` / `LIVEKIT_API_SECRET` | `devkey` / `secret` | server-side token minting |
| `SELF_HOSTED_DOMAIN` | `http://34.165.143.31:8080` | URL advertised to LiveKit clients |
| `UPLOAD_DRIVER` | `local` | `local` or `s3` |
| `MAX_UPLOAD_MB` | `25` | upload size limit |

Frontend variables live in `desktop/.env.example` and are inlined at build time by Vite:
`VITE_API_BASE_URL`, `VITE_WS_URL`, `VITE_APP_NAME`. Both can be overridden at runtime from
**Settings → Connection** inside the app.


### Data model

`User`, `Server`, `Channel`, `Message`, `Reaction`, `DirectMessage`, `Friend`, `Role`, `Member`,
`Invite`, plus `Attachment` and the `MemberRole` join table. Permissions use a 64-bit bitmask
(`Administrator`, `ManageChannels`, `SendMessages`, …) defined once in
`backend/internal/permissions`.

### API summary

Routes are prefixed with `/api` and require `Authorization: Bearer <access_token>` except register,
login, refresh, `GET /api/invites/:code` and the health checks.

```
POST   /auth/register           POST   /auth/login          POST /auth/refresh
POST   /auth/logout             GET    /users/me            PATCH /users/me
GET    /users/:id               GET    /users               GET  /servers
POST   /servers                 GET    /servers/:id         PATCH /servers/:id
DELETE /servers/:id             POST   /servers/:id/channels     PATCH /channels/:id
DELETE /channels/:id            GET    /channels/:id/messages    POST /channels/:id/messages
PATCH  /messages/:id            DELETE /messages/:id        POST /messages/:id/reactions
DELETE /messages/:id/reactions GET    /messages/:id/reactions
GET    /dms                     POST   /dms                 GET  /dms/:id/messages
POST   /dms/:id/messages        DELETE /dms/:id             GET  /dms/:id/recipients
GET    /friends                 POST   /friends             PATCH /friends/:id
DELETE /friends/:id             POST   /livekit/token       POST /upload
GET    /servers/:id/invites     POST   /servers/:id/invites DELETE /invites/:code
GET    /invites/:code           POST   /invites/:code/join
GET    /healthz                 GET    /readyz
```

Error responses always have the shape:

```json
{ "error": { "code": "validation_error", "message": "content is required", "fields": { "content": "required" } } }
```

### WebSocket

Connect to `/ws` (or `/api/ws`) with the access token:

```
ws://localhost:8080/ws?token=<access_token>
```

Every frame is `{ "event": "...", "data": { ... }, "nonce": "optional" }`. Client events:
`message.create`, `message.update`, `message.delete`, `typing.start`, `typing.stop`,
`presence.update`, `friend.request`, `friend.accepted`, `heartbeat`. Server events: `ready`,
`message.created`, `message.updated`, `message.deleted`, `typing.started`, `typing.stopped`,
`presence.updated`, `friend.request`, `friend.accepted`, `error`. Exact payloads live in
`CONTRACT.md`.

---

## 7. Auto-update setup

The desktop app uses the Tauri updater plugin. Two things must be configured before updates work.

1. **A public key** in `desktop/src-tauri/tauri.conf.json`:

   ```jsonc
   "plugins": {
     "updater": {
       "pubkey": "REPLACE_WITH_YOUR_TAURI_UPDATER_PUBLIC_KEY",
       "endpoints": [
         "https://releases.cordis.dev/desktop/{{target}}/{{arch}}/{{current_version}}"
       ]
     }
   }
   ```

   Generate the key pair once with `npm run tauri signer generate -w ~/.tauri/cordis.key`, and keep
   the private key in CI secrets only. **The value shipped in the repository is a placeholder** —
   generate your own before publishing, or signature verification will fail at runtime.

2. **A static file server** that exposes the artifacts and manifests produced by
   `npm run tauri build`:

   ```
   https://releases.cordis.desktop/
     1.0.0/Cordis_1.0.0_x64-setup.msi
     1.0.0/Cordis_1.0.0_x64-setup.msi.sig
     1.1.0/Cordis_1.1.0_x64-setup.msi
     1.1.0/Cordis_1.1.0_x64-setup.msi.sig
     latest.json
   ```

   The `latest.json` file describes the newest release and is generated by
   `tauri-action` or written by hand per the updater plugin schema.

Publishing from GitHub Actions:

```yaml
name: release-desktop
on:
  push:
    tags: ['v*']

jobs:
  build:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20 }
      - uses: dtolnay/rust-toolchain@stable
        with: { targets: x86_64-pc-windows-msvc }
      - run: npm ci
        working-directory: desktop
      - name: Build and publish
        uses: tauri-apps/tauri-action@v0
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAURI_SIGNING_PRIVATE_KEY: ${{ secrets.TAURI_SIGNING_PRIVATE_KEY }}
          TAURI_SIGNING_PRIVATE_KEY_PASSWORD: ${{ secrets.TAURI_SIGNING_PRIVATE_KEY_PASSWORD }}
        with:
          tagName: ${{ github.ref_name }}
          releaseName: 'Cordis ${{ github.ref_name }}'
          releaseDraft: true
          prerelease: false
          args: --bundles msi,nsis
```

In the app, **Settings → Account → Check for updates** runs the check, shows download progress, and
prompts to install and relaunch. The window title bar also shows a small update badge when a newer
version is available.

---


## 8. Production deployment

A 2 vCPU / 4 GB box comfortably handles a few thousand concurrent users.

### 8.1 Bare-IP deployment (current setup — no domain, no TLS)

This is the configuration this repository is currently tuned for: everything is served over plain
`http://` and `ws://` from a VPS IP address, with no domain name and no certificate.

```bash
git clone <your-repo-url> cordis && cd cordis
cp .env.example .env
```

`.env` for a bare-IP deployment:

```dotenv
APP_ENV=production
GIN_MODE=release
JWT_SECRET=<64+ random characters>
POSTGRES_PASSWORD=<strong password>

# IP-ONLY: TODO - replace the IP when it changes, or move to a domain + TLS
PUBLIC_URL=http://34.165.143.31:8080
CORS_ORIGINS=*
UPLOAD_PUBLIC_BASE=http://34.165.143.31:8080/uploads
LIVEKIT_URL=ws://34.165.143.31:7880
LIVEKIT_API_KEY=devkey
LIVEKIT_API_SECRET=secret
SELF_HOSTED_DOMAIN=http://34.165.143.31:8080
```

```bash
docker compose up -d --build
docker compose ps
curl -fsS http://34.165.143.31:8080/readyz
```

Only two services publish ports:

| Service | Published | Purpose |
|---|---|---|
| `backend` | `8080` | REST API, WebSocket, uploads |
| `livekit-server` | `7880`, `7881`, `50000-60000/udp` | signalling and RTC media |

`postgres` and `redis` are **not** published. They are reachable only from the backend container on
the internal `cordis` network, so neither database nor cache is exposed to the internet.

> **Limitation of IP-only deployment.** Without TLS, all traffic — including bearer tokens and
> message bodies — travels in clear text. Anyone on the network path can read it, and LiveKit will
> not negotiate `wss://`, so browser-based media may be restricted. This is acceptable for a private
> or trusted network. Add a domain and TLS before exposing the server publicly.

### 8.2 Domain + TLS deployment (optional)

Everything below is optional and **not used** by the current bare-IP setup. Reach it only when you
add a domain and a certificate.

```bash
git clone <your-repo-url> cordis && cd cordis
cp .env.example .env
```

Edit `.env` for a domain deployment:

```dotenv
APP_ENV=production
GIN_MODE=release
JWT_SECRET=<64+ random characters>
POSTGRES_PASSWORD=<strong password>
PUBLIC_URL=https://chat.example.com
CORS_ORIGINS=https://admin.example.com
LIVEKIT_URL=wss://chat.example.com/livekit
LIVEKIT_API_KEY=<from your livekit.yaml>
LIVEKIT_API_SECRET=<from your livekit.yaml>
SELF_HOSTED_DOMAIN=https://chat.example.com
UPLOAD_DRIVER=s3           # recommended, or use the local volume with a backup policy
S3_ENDPOINT=https://s3.example.com
S3_BUCKET=cordis
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
```

```bash
docker compose up -d --build
docker compose ps
curl -fsS https://chat.example.com/readyz
```

### 8.3 Put it behind a reverse proxy (Caddy — domain deployments only)

```caddyfile
chat.example.com {
    encode gzip zstd

    # WebSocket upgrade for /ws and /api/ws
    @ws {
        path /ws /api/ws
    }
    reverse_proxy @ws backend:8080

    # Everything else: API, uploads, health checks
    reverse_proxy backend:8080 {
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }
}
```

Nginx equivalent:

```nginx
server {
    listen 443 ssl http2;
    server_name chat.example.com;

    ssl_certificate     /etc/letsencrypt/live/chat.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/chat.example.com/privkey.pem;

    client_max_body_size 25m;

    location / {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;   # long-lived WebSocket connections
    }
}
```

When the proxy terminates TLS, set `TRUSTED_PROXIES` to the proxy network so rate limiting sees
real client IPs, for example `TRUSTED_PROXIES=172.16.0.0/12`.


### 8.4 LiveKit in production

`--dev` mode is for local work only. In production, run LiveKit with a real configuration:

```yaml
# livekit.yaml
port: 7880
rtc:
  tcp_port: 7881
  port_range_start: 50000
  port_range_end: 50100
  use_external_ip: true
keys:
  <your LIVEKIT_API_KEY>: <your LIVEKIT_API_SECRET>
```

```bash
docker run -d --name livekit \
  -p 7880:7880 -p 7881:7881 -p 50000-50100:50000-50100/udp \
  -v /etc/livekit.yaml:/etc/livekit.yaml \
  livekit/livekit-server --config /etc/livekit.yaml
```

Then point `LIVEKIT_URL` at the `wss://` endpoint and `SELF_HOSTED_DOMAIN` at the public HTTPS
origin, since LiveKit needs an ICE/TURN capable address to establish media paths.

### 8.5 Scaling the backend horizontally

The WebSocket hub is per-process, but every broadcast is also published to Redis Pub/Sub
(`REDIS_CHANNEL`), so replicas stay in sync and any client can connect to any replica.

```yaml
backend:
  deploy:
    replicas: 3
  environment:
    REDIS_URL: redis://redis:6379/0
```

Put a load balancer with WebSocket-aware routing in front. Sticky sessions are recommended so
reconnects land on the same replica and typing indicators stay warm.

### 8.6 Backups

```bash
# Postgres
docker compose exec postgres pg_dump -U cordis cordis | gzip > cordis-$(date +%F).sql.gz

# Local uploads
docker run --rm -v cordis_uploads:/data -v "$PWD":/backup alpine \
  tar czf /backup/uploads-$(date +%F).tar.gz -C /data .
```

Schedule both with cron and copy the results off the host. Redis holds only refresh tokens and
ephemeral presence, so it does not need backups — but a flush logs everyone out.

### 8.7 Hardening checklist

* [ ] `APP_ENV=production` and `GIN_MODE=release`
* [ ] `JWT_SECRET` is 32+ random characters and differs from the repository default
* [ ] `CORS_ORIGINS` lists only real origins — never `*`
* [ ] TLS terminated in front of both the API and LiveKit
* [ ] `TRUSTED_PROXIES` set to the proxy CIDR so rate limiting is accurate
* [ ] Strong `POSTGRES_PASSWORD`, and the DB port not exposed publicly
* [ ] Redis port not exposed publicly
* [ ] `UPLOAD_DRIVER=s3` or a mounted volume with backups for local uploads
* [ ] `MAX_UPLOAD_MB` set deliberately; Nginx `client_max_body_size` matches
* [ ] Code signing certificate and timestamping configured (section 5)
* [ ] Updater `pubkey` replaced with your real key pair (section 7)

---


## 9. Architecture notes

### 9.1 Request flow

```
React component
  └─ hooks (TanStack Query / Zustand)
      └─ lib/api/client.ts   ── fetch ──▶  Gin router
                                              └─ middleware (request id, logger,
                                                 recovery, CORS, rate limit, auth)
                                                  └─ handler  ──▶ service ──▶ GORM/Redis
                                                       └─ ws.Publisher ──▶ hub ──▶ sockets
                                                                              └─ Redis Pub/Sub
                                                                                   └─ other replicas
```

Handlers deal only with HTTP concerns: binding, validation and status codes. Services own the
business logic and never touch `*gin.Context`, which lets the WebSocket hub reuse exactly the same
service functions as the REST endpoints. A message sent over HTTP and one sent over the socket
therefore produce identical persisted state and identical outbound events.

### 9.2 Why a Publisher interface

HTTP handlers must be able to emit WebSocket events without depending on the hub package.
`service.Publisher` declares the publish methods, the WebSocket hub implements them, and a no-op
implementation is available for tests or when the WS server is disabled. This keeps the dependency
arrows pointing one way.

### 9.3 Authentication flow

1. `POST /auth/login` (or `/auth/register`) verifies the bcrypt hash and returns an **access token**
   (15 min, HS256 JWT) plus a **refresh token** (30 days, JWT carrying a `jti`).
2. The `jti` is stored in Redis at `cordis:refresh:<jti>` with the full refresh TTL.
3. `POST /auth/refresh` validates the signature and the token type, checks the `jti` still exists in
   Redis, **deletes** it, and issues a brand-new pair. Refresh tokens rotate on every use, so a
   stolen token is single-use and its reuse is detectable.
4. `POST /auth/logout` deletes the `jti`, revoking the session immediately.
5. The desktop client stores the pair locally and refreshes transparently on the first `401`,
   replaying the original request once.

### 9.4 Presence model

Presence is stored in Redis (`cordis:online:<userID>`, TTL 5 minutes) and refreshed by the client's
25-second heartbeat, so a crashed client disappears on its own without needing a disconnect event.
`presence.updated` frames are broadcast to everyone who shares a server with the user.

### 9.5 Voice and video

Voice channels are `Channel` rows with `type: "voice"` or `"video"`. Joining calls
`POST /api/livekit/token` with `roomName: "channel:<uuid>"`; the backend verifies channel membership
and mints a short-lived LiveKit access token. Text channels grant `canPublish: false`, which turns
them into watch-only rooms. The frontend keeps a single `livekit-client` `Room` instance in
`voice.store`, and screen share simply enables a screen-share track on that same room.

### 9.6 Windows integration

| Feature | Implementation |
|---|---|
| Rendering | WebView2, preinstalled on Windows 10/11 |
| Window | Frameless with an overlay title bar, custom drag/minimize/maximize/close |
| Tray | Mute, Deafen, Status submenu, Show, Quit; left click toggles visibility |
| Notifications | Native Tauri notifications for DMs and mentions, with a browser fallback |
| File dialogs | Native Windows picker; drag and drop also supported |
| Auto-update | Tauri updater plugin with signed manifests |
| Autostart | `autostart` plugin writing the per-user Run registry key |
| Background presence | The tray keeps the app resident; the window hides instead of closing |

---


## 10. Troubleshooting

**`readyz` returns `degraded`**
Check `docker compose ps` and `docker compose logs backend`. Postgres and Redis must both be
`healthy`. A `DATABASE_URL` pointing at `localhost` from inside a container is the most common
mistake — it must use the service hostname `postgres`.

**`401 unauthorized` immediately after login**
The access token expired (15 min default) and the client's refresh failed. Confirm `JWT_SECRET` did
not change between the login and the refresh, and that Redis is reachable — refresh token `jti`
records live there. If Redis was flushed, users must log in again.

**WebSocket connects then immediately closes**
* The token passed as `?token=` expired — the client must refresh before connecting.
* The proxy is not forwarding `Upgrade` and `Connection` headers (see the Nginx block above).
* `TRUSTED_PROXIES` is missing, so the origin check rejects the request.

**CORS errors in the browser during development**
Add the exact origin shown in the console to `CORS_ORIGINS`, comma separated. The bare-IP setup
already ships `CORS_ORIGINS=*`; if you pinned explicit origins, list every origin you use. Never
combine `*` with credentialed requests in a browser — the middleware omits
`Access-Control-Allow-Credentials` for the wildcard case, so use a bearer token instead of cookies.

**Voice connects but there is no audio**
* `LIVEKIT_URL` must be reachable from the client machine. Bare IP: `ws://34.165.143.31:7880`.
  Behind TLS: `wss://<domain>/livekit`.
* `SELF_HOSTED_DOMAIN` must be an address the client can resolve. Bare IP:
  `http://34.165.143.31:8080`. Note that browsers only permit media on a secure origin, so voice in
  a plain browser requires TLS — the Tauri client works over `ws://`.
* The UDP range published by compose is `50000-60000/udp`, but the `--dev` LiveKit server only
  allocates `50000-50100` by default. For the full range, add a `livekit.yaml` with
  `rtc.port_range_start: 50000` and `rtc.port_range_end: 60000` and mount it into the container.
* Windows firewall: allow `livekit-server.exe` on private networks.

**Uploads fail with 413**
Raise `MAX_UPLOAD_MB`, and in parallel raise `client_max_body_size` in Nginx or Caddy — the proxy
limit applies first.

**`npm run tauri build` fails with a linker error on Windows**
Install the Visual Studio Build Tools "Desktop development with C++" workload, then run
`rustup default stable-msvc`. Also confirm the WebView2 SDK is present (it ships with the build
tools) and that `rustup target add x86_64-pc-windows-msvc` succeeded.

**Installer build fails on icon files**
Icons must be real PNG and ICO files at the paths listed in `tauri.conf.json`. Regenerate them with
`npm run tauri icon path/to/logo.png`.

**Notifications never appear on Windows**
Windows Focus Assist suppresses them. Verify the app is allowed under **Settings → System →
Notifications**, and note that notifications require the installed app, not `tauri dev`.

---

## 11. License

Proprietary. All rights reserved.

