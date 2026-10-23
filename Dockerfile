# syntax=docker/dockerfile:1

# ---- build stage ---------------------------------------------------------
FROM golang:1.23-alpine AS build
WORKDIR /src

# Cache module downloads first.
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/pageguard ./cmd/pageguard

# ---- runtime stage -------------------------------------------------------
# Node is included so out-of-process plugins and the Playwright renderer work.
FROM node:20-slim AS runtime
WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/pageguard /usr/local/bin/pageguard
COPY plugins /app/plugins
COPY migrations /app/migrations
COPY tools/browser-renderer /app/tools/browser-renderer

# Install the headless browser. This is the heavy step; comment it out (and
# unset PAGEGUARD_BROWSER_CMD) for a slim, HTTP-only image.
RUN cd /app/tools/browser-renderer \
    && npm install --omit=dev --no-audit --no-fund \
    && npx playwright install --with-deps chromium \
    && npm cache clean --force

ENV PAGEGUARD_ADDR=":8080" \
    PAGEGUARD_PLUGINS_DIR="/app/plugins" \
    PAGEGUARD_MIGRATIONS_DIR="/app/migrations" \
    PAGEGUARD_BROWSER_CMD="node /app/tools/browser-renderer/index.mjs" \
    PAGEGUARD_NODE_BIN="node"

EXPOSE 8080
USER node
ENTRYPOINT ["pageguard"]
