# ADR-003: Docker Homelab Deployment

## Status
Accepted

## Context

The fantasy hockey league is a private app for ~10 friends. It needs to be self-hosted on a homelab server (single machine, no Kubernetes). Requirements:

- Easy to deploy and update (`docker compose up --build -d`)
- SQLite data survives container restarts and image rebuilds
- WebSocket connections (live draft) must pass through cleanly
- No CORS complexity in production
- SSL handled at the host level (not inside the app containers)

## Decision

### Two-container Docker Compose stack

| Container | Image | Purpose |
|---|---|---|
| `backend` | Go binary (debian:bookworm-slim) | API + WebSocket server on :8080 |
| `frontend` | nginx:1.27-alpine + Vite dist | Serves static React app, proxies `/api` and `/ws` |

#### Why not a single combined container?

A single container (Go serving embedded static files via `embed.FS`) would be simpler, but two containers gives us:
- Independent rebuilds: changing a CSS file doesn't require recompiling Go
- Standard nginx features: gzip, asset caching headers, clean WebSocket proxying
- Easier to scale or replace either half independently in future

#### Why nginx instead of Caddy-as-proxy?

nginx runs inside the stack and is only responsible for the frontend + proxying. Caddy (or Traefik) runs at the homelab level and handles SSL termination and routing across multiple services. The separation of concerns keeps this stack self-contained.

#### No CORS in production

By having nginx proxy `/api` and `/ws` to the backend, the browser sees everything on the same origin. The CORS middleware in `main.go` is still needed for local development (Vite dev server on :5173 → Go on :8080) but is a no-op in production.

### SQLite on a named Docker volume

The database file lives at `/data/hockey_season.db` inside the backend container, backed by a named Docker volume (`db_data`). This survives:
- `docker compose up --build` (image rebuild)
- `docker compose down` (stack stop — **not** `down -v`)

To back up the database: `docker cp hockey_season-backend-1:/data/hockey_season.db ./backup.db`

### CGO + go-sqlite3 build strategy

`go-sqlite3` requires CGO. The backend Dockerfile uses:
- Builder: `golang:1.25-bookworm` (has gcc)
- Runtime: `debian:bookworm-slim` (has glibc, compatible with CGO-linked binary)

Alpine is avoided in the runtime stage because its musl libc is incompatible with a glibc-compiled CGO binary without extra configuration.

## Homelab setup (outside this repo)

**Recommended: Caddy as the outer reverse proxy** (handles SSL automatically via Let's Encrypt):

```
# /etc/caddy/Caddyfile
hockey.yourdomain.com {
    reverse_proxy localhost:3000
}
```

Then on the server:
```bash
cp .env.example .env
# edit .env — set DRAFT_SECRET
docker compose up -d --build
```

To update after a code push:
```bash
git pull
docker compose up -d --build
```

## Consequences

- Developers run `vite dev` + `go run .` locally as before — Docker is deploy-only
- `DRAFT_SECRET` must be set in `.env` before first run (server exits if missing)
- SQLite is appropriate for this league size (~10 concurrent users). If this ever expands significantly, the `store` package can be adapted for Postgres without changing handlers.
- `docker compose down -v` will **delete the database** — document this clearly for whoever operates the server
