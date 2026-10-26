# Architecture

## Components

| Service | Tech | Role |
|---|---|---|
| `caddy` | Caddy 2 | Only public entry point. Routes by path, terminates HTTPS in production. |
| `frontend` | Nuxt 3 / Vue 3 | Pages. Public pages are server-rendered; dashboard and admin render in the browser. |
| `backend` | Go, Gin, GORM | REST API, WebSocket, sessions, permissions, MediaMTX auth hook. |
| `mysql` | MySQL 8.4 | Users, roles, sessions, channels, comments, audit log. |
| `mediamtx` | MediaMTX 1.21 | Receives OBS streams (RTMP/WHIP), serves HLS and WebRTC. |

### Routing (caddy/Caddyfile)

Everything shares one origin, so the session cookie works for pages, API and
WebSocket, and no CORS is needed.

| Path | Goes to |
|---|---|
| `/api/*`, `/ws/*` | backend :8080 |
| `/live/<channel>/whip`, `/live/<channel>/whep` | mediamtx :8889 (WebRTC signalling) |
| `/live/*` | mediamtx :8888 (HLS) |
| everything else | frontend :3000 |

RTMP (1935/tcp) and WebRTC media (8189/udp+tcp) go straight to MediaMTX.
The backend's port 8081 (MediaMTX auth hook) is never published.

## Key flows

**Publishing.** OBS connects to MediaMTX at path `live/<slug>`. MediaMTX
POSTs the attempt to `backend:8081/mediamtx/auth`. The backend looks up the
channel, checks it and its owner are active, and compares the presented key
against the stored SHA-256 hash. 2xx allows the stream.

**Live status.** `LiveService` polls MediaMTX's API every 2 s. When a
channel goes live or offline it pushes a `status` event to that channel's
WebSocket room and updates `channels.last_live_at`.

**Watching.** The channel page is server-rendered with channel details and
recent chat. In the browser, `LivePlayer` plays HLS (hls.js, or native in
Safari) or WebRTC (WHEP), and `useChannelSocket` opens `/ws/channels/<slug>`.

**Viewer count.** Each open WebSocket on a channel page or embed counts as a
viewer (the streamer's dashboard polls instead, so it doesn't count). The hub
sends the count at most once per second per channel.

**Chat.** Posting is a REST call (`POST /channels/:slug/comments`), so it
goes through the same authentication, permission, validation and rate-limit
middleware as everything else. The service saves the comment and broadcasts
a `comment` event to the room. WebSocket connections are receive-only.

## Backend layers

```
Request → routes → middleware → controller → (DTO validation) → service → repository → MySQL
```

| Layer | Package | Rule |
|---|---|---|
| Routes | `internal/routes` | Endpoint list with auth/permission middleware. No logic. |
| Middleware | `internal/middleware` | Request ID, logging, recovery, security headers, origin check, JSON-only bodies, rate limits, session loading, `RequireAuth`, `RequirePermission`. |
| Controllers | `internal/controllers` | Bind/validate DTOs, call one service, write the response. |
| DTOs | `internal/dto` | Request and response contracts. Requests list only fields a caller may set. |
| Services | `internal/services` | Business rules, ownership checks, audit logging, realtime broadcasts. |
| Policies | `internal/policies` | Resource rules (owner of a channel, who may delete a comment). |
| Repositories | `internal/repositories` | All GORM/SQL. Return `ErrNotFound` / `ErrDuplicate`, never GORM errors. |
| Models | `internal/models` | Table mappings. Never serialized to clients. |

`internal/app` wires everything together; `cmd/server` and the integration
tests both use it.

Schema changes are SQL files in `backend/migrations`, applied in order at
startup and recorded in `schema_migrations` (GORM AutoMigrate is not used).

## Frontend

```
page → component → composable / store → service → $api (plugins/01.api.ts) → Go API
```

| Where | Runs on | Notes |
|---|---|---|
| `pages/index.vue`, `pages/c/[slug].vue`, `pages/embed/[slug].vue` | server + browser | SSR for SEO and fast first paint. |
| `pages/dashboard/**`, `pages/admin/**` | browser only | `routeRules: { ssr: false }`. Private, no SEO. |
| `plugins/01.api.ts` | both | One API client. On the server it calls `NUXT_API_INTERNAL_BASE` directly and forwards the cookie and client IP. |
| `plugins/02.auth.ts` | both | Loads `/auth/me` once; state is handed from server to browser through Pinia. |
| `LivePlayer`, `useChannelSocket` | browser only | Browser APIs are used only in `onMounted`/handlers. |
| `middleware/*` | both | Redirects for UX. The API enforces the same rules. |

No secrets live in the frontend. The session cookie is `HttpOnly`, so
JavaScript can't read it; SSR reads it from the request headers.

## Data model

```
roles ─< role_permissions >─ permissions
roles ─< users ─< sessions
              └─< channels ─< comments >─ users
audit_logs (user_id, no FK)
```

- `channels.stream_key_hash`: SHA-256 of the stream key. The key itself is never stored.
- `sessions.token_hash`: SHA-256 of the cookie token.
- `comments.deleted_at`: soft delete, for moderation history.

## Scaling notes

The API is a single instance by design: rate limits and the WebSocket hub
live in memory. To run several instances, move rate limiting and room
broadcasts to a shared store (for example Redis pub/sub). HLS scales well
behind a CDN; WebRTC load goes to the MediaMTX server.
