# API

Base path: `/api/v1`. JSON in and out. Authentication is the `ls_session`
cookie set by register/login (`HttpOnly`, `SameSite=Lax`, `Secure` on HTTPS).
State-changing requests must send JSON (or an image, for thumbnail uploads),
never a form body, and, from browsers, an `Origin` matching `APP_URL`.

## Response format

```json
{ "success": true, "message": "Channel created", "data": { } }
{ "success": true, "data": [ ], "meta": { "page": 1, "limit": 20, "total": 42, "total_pages": 3 } }
{ "success": false, "message": "Please check the highlighted fields", "errors": { "email": "Enter a valid email address" } }
```

| Status | Meaning |
|---|---|
| 401 | Not logged in, or the session expired |
| 403 | Logged in but not allowed, or cross-site request blocked |
| 404 | Not found, including another user's channel |
| 409 | Conflict (email or channel URL taken) |
| 413 | Upload too large |
| 415 | Body type an HTML form could send (form, multipart, text/plain) |
| 422 | Validation failed; see `errors` |
| 429 | Rate limited; see `Retry-After` |

## Endpoints

Permission names are defined in `backend/internal/auth/principal.go` and
mirrored in `frontend/types/auth.ts`.

### Auth

| Method | Path | Access | Body / notes |
|---|---|---|---|
| POST | `/auth/register` | public, 5/min/IP | `{name, email, password}`: new **viewer**, logs in |
| POST | `/auth/login` | public, 5/min/IP | `{email, password}` |
| POST | `/auth/logout` | logged in | Deletes the session |
| GET | `/auth/me` | logged in | `{id, name, email, role, permissions[]}` |

### Channels and chat (public)

| Method | Path | Access | Notes |
|---|---|---|---|
| GET | `/channels?live=&page=&limit=` | public | Active channels, most recently live first. `live=true`: only live; `live=false`: only offline |
| GET | `/channels/:slug` | public | `{slug, title, description, owner{id,name}, live, live_started_at, viewers, last_live_at, scheduled_at, thumbnail_url, hls_url, whep_url}` |
| GET | `/channels/:slug/thumbnail?v=` | public | JPEG. Cached for a year when `v` matches the current version (`thumbnail_url` includes it) |
| GET | `/channels/:slug/comments?before_id=&limit=` | public | Oldest first, up to 100 |
| POST | `/channels/:slug/comments` | `comments.create`, 3 burst then 1/s per user | `{body}`, 1–500 chars, one line |
| DELETE | `/comments/:id` | author, channel owner, or `comments.moderate` | |

### Your channels (streamers)

| Method | Path | Access | Notes |
|---|---|---|---|
| GET | `/me/channels` | `channels.manage_own` | Own channels, with OBS settings |
| POST | `/me/channels` | `channels.create` | `{slug, title, description}` → **stream key returned once** |
| GET | `/me/channels/:id` | owner | |
| PUT | `/me/channels/:id` | owner | `{title, description}`; other fields are ignored |
| POST | `/me/channels/:id/stream-key` | owner | New key (returned once); old key stops, live stream is cut |
| DELETE | `/me/channels/:id` | owner | |
| PUT | `/me/channels/:id/schedule` | owner | `{scheduled_at: "<RFC 3339>" \| null}`: next stream, future and within a year; `null` clears. Viewers see a countdown; going live within an hour of it clears it |
| PUT | `/me/channels/:id/thumbnail` | owner, 5 then 1 per 10 s per user | Raw image body (`Content-Type: image/jpeg\|png\|webp\|gif`, max 5 MB, min 160×90). Stored as a 16:9 JPEG of at most 1280×720 |
| DELETE | `/me/channels/:id/thumbnail` | owner | |

Another user's channel ID returns 404, not 403. Slugs are 3–40 characters
of `a-z 0-9 -` and can't be changed. Default limit: 5 channels per user.

Create and regenerate return:

```json
{
  "channel": { "id": 1, "slug": "my-show", "obs": { "rtmp_server": "rtmp://host/live", "whip_url": "https://host/live/my-show/whip" } },
  "stream_key": "sk_…",
  "rtmp_stream_key": "my-show?user=obs&pass=sk_…"
}
```

### Admin

| Method | Path | Access |
|---|---|---|
| GET | `/admin/users?q=&page=&limit=` | `users.view` |
| GET | `/admin/roles` | `users.view` |
| PATCH | `/admin/users/:id/role` `{role: admin\|streamer\|viewer}` | `users.update` (not yourself) |
| PATCH | `/admin/users/:id/status` `{status: active\|disabled}` | `users.update` (not yourself). Disabling signs the user out and stops their streams. |
| GET | `/admin/channels?q=&page=&limit=` | `channels.view_all` |
| PATCH | `/admin/channels/:id/status` `{status}` | `channels.update_all`. Disabling stops the stream and hides the channel. |

### Other

| Method | Path | Notes |
|---|---|---|
| GET | `/health` | Database check |

## WebSocket: `GET /ws/channels/:slug`

Receive-only. Each connection counts as one viewer. Max 10 connections per
IP. Events:

```json
{ "type": "status",          "data": { "live": true, "started_at": "2026-…" } }
{ "type": "viewers",         "data": { "count": 12 } }
{ "type": "comment",         "data": { "id": 7, "body": "hi", "created_at": "…", "user": { "id": 3, "name": "Dara" } } }
{ "type": "comment_deleted", "data": { "id": 7 } }
{ "type": "schedule",        "data": { "scheduled_at": "2026-11-01T13:00:00Z" } }
```

On connect you get `status`, then `viewers`.

## Internal: MediaMTX auth hook

`POST http://backend:8081/mediamtx/auth`, reachable only inside the Docker
network. MediaMTX sends `{user, password, token, ip, action, path, protocol, id, query}`.
`read` on an active channel is allowed. `publish` is allowed when the
password, Bearer token, or `pass`/`key`/`token` query value matches the
channel's stream key. Everything else is denied.
