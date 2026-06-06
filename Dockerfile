# syntax=docker/dockerfile:1.7
ARG GO_BASE_IMAGE=golang:1.26.4-bookworm

FROM ${GO_BASE_IMAGE} AS backend-builder

WORKDIR /src

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates build-essential \
	&& rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=1 go build -ldflags '-w -s' -o /out/main cmd/server/main.go \
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
