# Security

The browser is untrusted. Every rule below is enforced by the Go API; the
frontend only hides what a user can't do.

## Authentication

- Opaque random session tokens (256-bit) in an `HttpOnly`, `SameSite=Lax`
  cookie, `Secure` when `APP_URL` is HTTPS. Only the SHA-256 hash is stored
  (`sessions.token_hash`).
- Sessions expire after `SESSION_TTL` (default 30 days). Logout deletes the
  session. Disabling a user deletes all their sessions, and every request
  re-checks that the user is active.
- Passwords: bcrypt, cost 12, 8–72 bytes. Login takes the same time for
  unknown emails as for wrong passwords, and both get the same message.

## Authorization (RBAC)

| Permission | viewer | streamer | admin |
|---|:-:|:-:|:-:|
| `comments.create` | ✓ | ✓ | ✓ |
| `channels.create`, `channels.manage_own` | | ✓ | ✓ |
| `comments.moderate` | | | ✓ |
| `users.view`, `users.update` | | | ✓ |
| `channels.view_all`, `channels.update_all` | | | ✓ |

- New accounts are viewers. Only an admin can grant streaming, so strangers
  can't use your server's bandwidth.
- Roles and permissions are seeded by migration; role changes take effect
  on the user's next request.
- Admins can't change their own role or status, so they can't lock themselves out.

### Resource policies (`internal/policies`)

- Channel management requires `channels.manage_own` **and** ownership.
  Someone else's channel returns 404 for every action, so IDs can't be probed.
- A comment can be deleted by its author, the channel owner, or a moderator.
- Disabled channels, and channels of disabled users, can't be watched or streamed to.

### Mass assignment

Request DTOs list only the fields a caller may set. `PUT /me/channels/:id`
accepts `title` and `description`; owner, slug, status and stream key in the
body are ignored. Repositories update explicit columns only.

## Stream keys

- 256-bit random, prefixed `sk_`, stored as SHA-256, shown to the owner only
  in the create/regenerate response.
- Compared in constant time. Each key works only for its own channel.
- Regenerating a key, disabling a channel or disabling its owner disconnects
  the current publisher through the MediaMTX API.
- The auth hook listens on a separate port that's never exposed.

## Web protections

- **CSRF:** `SameSite=Lax` cookies, plus the API rejects state-changing
  requests whose `Origin` isn't `APP_URL` (403) and any body an HTML form can
  send: urlencoded, multipart or text/plain (415). JSON and image uploads
  need a CORS preflight the API never approves.
- **Uploads:** thumbnails are size-limited (5 MB, 40 megapixels), decoded,
  cropped and re-encoded as JPEG before storing, so metadata (EXIF/GPS) and
  anything hidden in the file is dropped. Served as `image/jpeg` with
  `nosniff`.
- **XSS:** chat and all user text render through Vue interpolation (escaped);
  no `v-html`. Comment bodies are stripped of control characters.
- **WebSocket:** origin-checked, receive-only, 512-byte read limit, 10
  connections per IP, slow clients are dropped.
- **Visitor IP:** Caddy decides the client IP. It only believes a client-IP
  header from proxies listed in `TRUSTED_PROXIES` (default: none), then
  passes the result to the backend and frontend in `X-Real-IP`, overwriting
  anything the visitor sent. The backend only reads that header from its
  private-network proxies. Rate limits and audit logs depend on this.
- **Open redirects:** `?redirect=` only accepts same-site paths.
- **Headers:** `X-Content-Type-Options`, `Referrer-Policy`, and
  `X-Frame-Options: SAMEORIGIN` on everything except `/embed/*`.
  API responses are `Cache-Control: no-store`.
- **Rate limits (per IP unless noted):** API 20 req/s (burst 100),
  login/register 5/min, chat 1/s per user (burst 3), WebSocket opens 1 per 2 s
  (burst 10). They're in memory, so they reset on restart and aren't shared
  between instances.

## Audit log

`audit_logs` records: `register`, `login`, `failed_login`, `logout`,
`role_change`, `user_status_change`, `channel_create`, `channel_delete`,
`channel_status_change`, `stream_key_regenerate`, `stream_publish` (success
and failure), `comment_moderate`, with user, resource, IP, user agent and
time. Passwords, tokens and stream keys are never written to logs or the
audit table. Request logs omit query strings, and SQL logs omit values.

## Secrets

All secrets come from `.env` (never committed; see `.env.example`):
`DB_PASSWORD`, `DB_ROOT_PASSWORD`, `ADMIN_PASSWORD`. The frontend has none.
`ADMIN_PASSWORD` is only used to create the first admin; change it in the app
afterwards.

## Tested

`backend/tests/integration/security_test.go` calls the API directly,
bypassing the frontend: anonymous chat, forged and logged-out session
cookies, viewers creating channels, cross-user channel access (IDOR),
protected fields sent in updates, privilege escalation to admin, admin
self-lockout, disabled users and channels, cross-site requests and form
posts, WebSocket origin and unknown channels, and login rate limiting.

## Known limitations

- No email verification, password reset, or 2FA yet.
- No CAPTCHA on sign-up; rate limits are the only abuse control.
- Viewer counts count open connections, so one person with two tabs counts twice.
- RTMP from OBS is unencrypted (the stream key travels in clear text).
  For RTMPS see MediaMTX's docs; WHIP over HTTPS is encrypted.
