# ── Stage 1: build ────────────────────────────────────────────────────────────
# go-sqlite3 requires CGO, so we need a real C compiler in the builder.
FROM golang:1.25-bookworm AS builder

WORKDIR /src

# Cache module downloads separately from source
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /out/server .

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
# bookworm-slim keeps glibc so the CGO binary links correctly.
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/server ./server

# Migrations are copied in via docker-compose volume mount at runtime.
# See docker-compose.yml: the db_data volume and ./data bind mount.

EXPOSE 8080

ENV DB_PATH=/data/hockey_season.db
ENV MIGRATIONS_DIR=/data/migrations

ENTRYPOINT ["./server"]
