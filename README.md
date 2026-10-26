# Livestream

A self-hosted live streaming site. Streamers go live from OBS on their own
channels; viewers watch in the browser and chat.

- **Many channels.** Each channel has its own page (`/c/<name>`), chat and stream key.
- **Accounts and roles.** Anyone can sign up as a *viewer* (watch and chat).
  An admin promotes users to *streamer* (can create channels).
- **Live chat.** Anyone can read it; only logged-in users can post. Channel
  owners and admins can delete messages.
- **Live viewer count**, pushed in real time over WebSocket.
- **Thumbnails** per channel, shown on cards, behind the offline player and
  in link previews.
- **Countdown:** schedule the next stream and viewers see a live countdown
  that turns into the video when you go live.
- **Two playback modes:** Standard (HLS, ~2–4 s delay, works everywhere) and
  Low latency (WebRTC, under 1 s).
- **Embeds:** put any channel's player on another website with an `<iframe>`.
- **Admin:** manage users (roles, disable) and channels (disable).

```
OBS ──RTMP / WHIP──▶ MediaMTX ──HLS / WebRTC──▶ browser
                        │  "is this stream key valid?"
                        ▼
browser ──▶ Caddy ──▶ Nuxt (pages, SSR) ──▶ Go API ──▶ MySQL
                 └──▶ Go API (REST, WebSocket)
```

Details: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) ·
[docs/API.md](docs/API.md) · [docs/SECURITY.md](docs/SECURITY.md) ·
[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)

## Run it locally

Requires Docker.

```sh
cp .env.example .env
# Edit .env: set DB_PASSWORD, DB_ROOT_PASSWORD, ADMIN_EMAIL and ADMIN_PASSWORD
# (openssl rand -hex 24 makes good secrets)
docker compose up -d --build
```

Open <http://localhost:8080> and log in with `ADMIN_EMAIL` / `ADMIN_PASSWORD`.
The admin account is created the first time the backend starts.

## Go live

1. **Get a streamer account.** Sign up, then as admin open **Admin → Users**
   and change that user's role to *Streamer*. Admins can stream too.
2. **Create a channel.** **Dashboard → New channel**. A dialog shows the OBS
   settings and the stream key, **once**. Lost it? Use *New stream key* on
   the channel page.
3. **Configure OBS** (*Settings → Stream*), using either option:

   | | RTMP (any OBS) | WHIP (OBS 30+, <1 s delay with sound) |
   |---|---|---|
   | Service | Custom… | WHIP |
   | Server | `rtmp://localhost/live` | `http://localhost:8080/live/<channel>/whip` |
   | Stream Key / Bearer Token | `<channel>?user=obs&pass=<key>` | `<key>` |

   The dashboard shows these exact values with copy buttons. Recommended
   output settings: x264 or Apple VT H264, CBR 2500–6000 Kbps, keyframe
   interval 1 s.
4. **Start Streaming.** Within about two seconds the channel page shows
   LIVE and viewers' players start automatically. The home page picks it up
   within 20 seconds.

With RTMP, OBS sends AAC audio, which WebRTC can't play: Low latency mode is
silent and the page says so. Standard mode has sound. Use WHIP if you want
sound in Low latency mode.

### Test without OBS

```sh
./scripts/test-stream.sh <channel> <key>   # test pattern + beep, sent like OBS does
```

## Tests

```sh
# Backend: unit + integration + security tests against a throwaway MySQL database
docker compose --profile test run --rm backend-test

# Frontend: types and unit tests
cd frontend && npm ci && npm run typecheck && npm test

# End to end in a real browser (needs the stack running and Google Chrome)
node frontend/tests/e2e/full-flow.mjs
```

CI runs the backend and frontend checks on every push
(`.github/workflows/ci.yml`).

## Deploy

On your own server or on Railway: see [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).
On Railway only RTMP streaming and Standard (HLS) playback work, because
Railway has no UDP for WebRTC.

## Project layout

```
backend/             Go API (Gin, GORM, MySQL)
  cmd/server/          entry point
  config/              env config, database, migrations runner, logger
  migrations/          versioned SQL schema (applied at startup)
  internal/
    routes/            every endpoint and who may call it  ← start here
    middleware/        auth, permissions, CSRF/origin, rate limits, logging
    controllers/       HTTP in/out only
    services/          business logic
    repositories/      all database access
    policies/          ownership rules (who may edit what)
    realtime/          WebSocket hub: viewers, live status, chat events
    mediamtx/          MediaMTX API client
  tests/integration/   API + security tests against MySQL
frontend/            Nuxt 3 (Vue 3, TypeScript, Tailwind, shadcn-vue, Pinia)
  pages/               routes; public pages are server-rendered
  components/          UI (components/ui = shadcn-vue)
  composables/         useAuth, usePermission, useChannelSocket, useChannelChat
  services/            the only place that calls the API
  stores/              Pinia auth store
  middleware/          route guards (UX only; the API enforces access)
mediamtx/            media server config
caddy/               reverse proxy: one origin for site, API, WebSocket, video
scripts/             test-stream.sh
docs/                architecture, API, security, deployment
```

## Common tasks

- **Logs:** `docker compose logs -f backend` (also `mediamtx`, `caddy`, `frontend`)
- **OBS can't connect:** the backend log shows `publish denied: wrong stream key`
  when the key is wrong. If nothing shows up, OBS isn't reaching port 1935.
- **After changing code:** `docker compose up -d --build`
- **Database shell:** `docker compose exec mysql mysql -uroot -p livestream`
