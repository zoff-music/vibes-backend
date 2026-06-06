ARG GO_BASE_IMAGE=golang:1.26.4-bookworm
ARG NODE_VERSION=26
ARG PNPM_VERSION=11.5.2

FROM ${GO_BASE_IMAGE} AS backend-builder

WORKDIR /src

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates build-essential \
	&& rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags '-w -s' -o /out/main cmd/server/main.go \
	&& CGO_ENABLED=0 go build -ldflags '-w -s' -o /out/migrator-main cmd/migrator/main.go

FROM debian:bookworm-slim AS backend-prod

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& groupadd --system appuser \
	&& useradd --system --gid appuser --home-dir /app appuser

WORKDIR /app

COPY --from=backend-builder /out/main /app/main
COPY --from=backend-builder /out/migrator-main /app/migrator-main
COPY migrator/migrations /app/migrator/migrations
COPY migrator/postgres /app/migrator/postgres
COPY docker-entrypoint.sh /app/docker-entrypoint.sh

RUN chmod +x /app/docker-entrypoint.sh \
	&& mkdir -p /app/data \
	&& chown -R appuser:appuser /app

USER appuser:appuser

ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/app/docker-entrypoint.sh"]

FROM node:${NODE_VERSION}-bookworm-slim AS frontend-deps

WORKDIR /src

COPY client/frontend/render/package.json client/frontend/render/pnpm-lock.yaml client/frontend/render/pnpm-workspace.yaml client/frontend/render/biome.json ./client/frontend/render/
COPY client/frontend/render/apps ./client/frontend/render/apps
COPY client/frontend/render/packages ./client/frontend/render/packages

RUN npm install -g pnpm@${PNPM_VERSION} \
	&& pnpm --dir client/frontend/render install --frozen-lockfile

FROM frontend-deps AS frontend-platform-builder

RUN pnpm --dir client/frontend/render --filter @vibez/platform build

FROM node:${NODE_VERSION}-bookworm-slim AS frontend-platform-prod

WORKDIR /app

COPY --from=frontend-platform-builder /src/client/frontend/render/package.json ./package.json
COPY --from=frontend-platform-builder /src/client/frontend/render/node_modules ./node_modules
COPY --from=frontend-platform-builder /src/client/frontend/render/apps/platform/package.json ./apps/platform/package.json
COPY --from=frontend-platform-builder /src/client/frontend/render/apps/platform/node_modules ./apps/platform/node_modules
COPY --from=frontend-platform-builder /src/client/frontend/render/apps/platform/dist ./apps/platform/dist
COPY --from=frontend-platform-builder /src/client/frontend/render/packages ./packages

ENV NODE_ENV=production
ENV VITE_API_URL_INTERNAL=http://backend:8080

EXPOSE 3000

WORKDIR /app/apps/platform

CMD ["./node_modules/.bin/react-router-serve", "./dist/server/index.js"]

FROM frontend-deps AS frontend-cast-builder

RUN pnpm --dir client/frontend/render --filter @vibez/cast build

FROM node:${NODE_VERSION}-bookworm-slim AS frontend-cast-prod

WORKDIR /app

COPY --from=frontend-cast-builder /src/client/frontend/render/package.json ./package.json
COPY --from=frontend-cast-builder /src/client/frontend/render/node_modules ./node_modules
COPY --from=frontend-cast-builder /src/client/frontend/render/apps/cast/package.json ./apps/cast/package.json
COPY --from=frontend-cast-builder /src/client/frontend/render/apps/cast/node_modules ./apps/cast/node_modules
COPY --from=frontend-cast-builder /src/client/frontend/render/apps/cast/dist ./apps/cast/dist

ENV NODE_ENV=production

EXPOSE 3001

WORKDIR /app/apps/cast

CMD ["./node_modules/.bin/vite", "preview", "--host", "0.0.0.0", "--port", "3001", "--outDir", "./dist"]
