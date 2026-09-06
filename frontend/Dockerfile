# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# VITE_API_BASE_URL is intentionally left unset here — the frontend always
# uses relative paths (/api, /ws) so nginx can proxy them without CORS.
RUN npm run build

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM nginx:1.27-alpine

COPY --from=builder /app/dist /usr/share/nginx/html

# nginx config is injected via docker-compose volume mount (see ../nginx.conf)
# so we don't bake it into the image — easier to tweak without a rebuild.

EXPOSE 80
