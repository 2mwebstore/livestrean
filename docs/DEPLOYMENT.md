# Deployment

Two ways to run it:

- **A Linux server with Docker** (below), e.g. a DigitalOcean Droplet with
  2 vCPU / 4 GB. Everything works, including Low latency (WebRTC).
- **Railway** (see [Railway](#railway) at the end). Quick to set up, but
  Railway has no UDP, so only RTMP streaming and Standard (HLS) playback work.

Video goes through your server either way, so bandwidth is the main cost:
roughly stream bitrate × viewers.

## 1. Server

```sh
# Ubuntu 24.04
curl -fsSL https://get.docker.com | sh
git clone <your repo> livestream && cd livestream
cp .env.example .env
```

## 2. DNS

| Record | Points to | Cloudflare proxy |
|---|---|---|
| `stream.example.com` | server IP | on or off |
| `ingest.example.com` | server IP | **off** (DNS only) |

Cloudflare's proxy only carries HTTP(S). RTMP (1935) and WebRTC media (UDP
8189) must reach the server directly, so OBS uses the DNS-only name. If you
use the Cloudflare proxy, set SSL/TLS mode to **Full (strict)**, and set
`TRUSTED_PROXIES` to Cloudflare's IP ranges and `CLIENT_IP_HEADERS=CF-Connecting-IP`
so the app sees visitors' real IPs (see `.env.example`).

## 3. `.env` for production

```ini
APP_ENV=production
APP_URL=https://stream.example.com
SITE_ADDRESS=stream.example.com     # Caddy gets a certificate automatically
HTTP_PORT=80
HTTPS_PORT=443
PUBLIC_RTMP_URL=rtmp://ingest.example.com/live
PUBLIC_HOST=203.0.113.10            # the server's public IP (WebRTC)
DB_PASSWORD=<openssl rand -hex 24>
DB_ROOT_PASSWORD=<openssl rand -hex 24>
ADMIN_EMAIL=you@example.com
ADMIN_PASSWORD=<long random password>
```

With an `https://` `APP_URL`, session cookies are `Secure` automatically.

## 4. Firewall

| Port | Protocol | For |
|---|---|---|
| 22 | tcp | SSH |
| 80, 443 | tcp | website, API, HLS, WHIP/WHEP |
| 1935 | tcp | RTMP from OBS |
| 8189 | udp + tcp | WebRTC media |

Nothing else needs to be open. MySQL, the MediaMTX API and the backend's
internal port aren't published.

```sh
ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp
ufw allow 1935/tcp && ufw allow 8189
ufw enable
```

## 5. Start

```sh
docker compose up -d --build
docker compose logs -f backend    # migrations, admin creation
```

Log in at `https://stream.example.com` as the admin. In OBS, WHIP uses
`https://stream.example.com/live/<channel>/whip`.

## Updates

```sh
git pull
docker compose up -d --build
```

Migrations run automatically on backend start. Check
`docker compose logs backend` after updating.

## Backups

Everything worth keeping is in MySQL:

```sh
# daily, e.g. from cron
docker compose exec -T mysql sh -c 'mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction livestream' | gzip > backup-$(date +%F).sql.gz

# restore
gunzip -c backup-2026-01-01.sql.gz | docker compose exec -T mysql sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" livestream'
```

Copy backups off the server, for example to DigitalOcean Spaces.

## Monitoring

- `GET /api/v1/health` returns 200 when the API and database are up. Point an
  uptime checker at it.
- `docker compose logs` per service. In production the backend writes JSON
  logs with a request ID on every line.

## Growing

- **More viewers:** put a CDN in front of `/live/*` (HLS is plain HTTP files),
  or move MediaMTX to its own server.
- **More API instances:** move rate limits and WebSocket broadcasting to a
  shared store such as Redis first (see ARCHITECTURE.md).

## Railway

Each part runs as its own Railway service, talking over Railway's private
network (`<service>.railway.internal`).

| Service | Source | Public access |
|---|---|---|
| `MySQL` | Railway MySQL database | none |
| `backend` | `backend/` | none (reached through `web`) |
| `frontend` | `frontend/` | none (reached through `web`) |
| `mediamtx` | `mediamtx/` | TCP proxy → port 1935, the RTMP address for OBS |
| `web` | `caddy/` | domain → port 8080, the site everyone opens |

### What works on Railway

Railway has no UDP, so WebRTC can't carry video:

- OBS must stream over **RTMP**; WHIP doesn't work.
- Viewers get **Standard (HLS)** playback. Choosing Low latency falls back to
  Standard automatically.
- The RTMP address is Railway's TCP proxy (`rtmp://<host>.proxy.rlwy.net:<port>/live`),
  not port 1935. The dashboard shows the right one.

### Set up

```sh
railway init --name livestream
railway add --database mysql
for s in backend frontend mediamtx web; do railway add --service $s; done
railway domain --service web --port 8080                  # → https://web-….up.railway.app
railway tcp-proxy create --service mediamtx --port 1935   # → host:port for OBS
```

Variables (`railway variable set --service <name> KEY=value …`):

| Service | Variables |
|---|---|
| `backend` | `APP_ENV=production`, `APP_URL=https://<web domain>`, `DB_HOST=${{MySQL.MYSQLHOST}}`, `DB_PORT=${{MySQL.MYSQLPORT}}`, `DB_NAME=${{MySQL.MYSQLDATABASE}}`, `DB_USER=${{MySQL.MYSQLUSER}}`, `DB_PASSWORD=${{MySQL.MYSQLPASSWORD}}`, `MEDIAMTX_API=http://mediamtx.railway.internal:9997`, `PUBLIC_RTMP_URL=rtmp://<tcp proxy host:port>/live`, `ADMIN_NAME`, `ADMIN_EMAIL`, `ADMIN_PASSWORD` |
| `frontend` | `NUXT_API_INTERNAL_BASE=http://backend.railway.internal:8080/api/v1`, `HOST=::`, `PORT=3000` |
| `mediamtx` | `MTX_AUTHHTTPADDRESS=http://backend.railway.internal:8081/mediamtx/auth` |
| `web` | `SITE_ADDRESS=:8080`, `PORT=8080`, `BACKEND_UPSTREAM=backend.railway.internal:8080`, `FRONTEND_UPSTREAM=frontend.railway.internal:3000`, `MEDIAMTX_HLS_UPSTREAM=mediamtx.railway.internal:8888`, `MEDIAMTX_WEBRTC_UPSTREAM=mediamtx.railway.internal:8889`, `TRUSTED_PROXIES=100.64.0.0/10`, `CLIENT_IP_HEADERS=X-Real-IP` |

Quote values containing `${{…}}` in single quotes so your shell leaves them alone.

The last two `web` variables let rate limits and audit logs see each
visitor's real IP. Railway's proxy connects from `100.64.0.0/10` and
replaces `X-Real-IP` with the real client address, so visitors can't fake
it. Without them every visitor appears as Railway's proxy and shares one
rate limit.

### Deploy

```sh
railway up backend  --path-as-root --service backend
railway up frontend --path-as-root --service frontend
railway up mediamtx --path-as-root --service mediamtx
railway up caddy    --path-as-root --service web
```

To deploy automatically on every `git push`, connect each service to the
GitHub repo in its Railway settings and set its **Root Directory**
(`backend`, `frontend`, `mediamtx`, `caddy`).
