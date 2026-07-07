ARG GO_BASE_IMAGE=golang:1.26.4-bookworm

FROM ${GO_BASE_IMAGE} AS backend-builder

WORKDIR /src

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates build-essential \
	&& rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags '-w -s' -o /out/main cmd/server/main.go \
	&& CGO_ENABLED=0 go build -ldflags '-w -s' -o /out/vibes-migrator cmd/migrator/main.go

FROM debian:bookworm-slim AS backend-prod

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& groupadd --system appuser \
	&& useradd --system --gid appuser --home-dir /app appuser

WORKDIR /app

COPY --from=backend-builder /out/main /app/main

RUN mkdir -p /app/data \
	&& chown -R appuser:appuser /app

USER appuser:appuser

ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/app/main"]

FROM debian:bookworm-slim AS migrator-prod

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& groupadd --system appuser \
	&& useradd --system --gid appuser --home-dir /app appuser

WORKDIR /app

COPY --from=backend-builder /out/vibes-migrator /app/vibes-migrator
COPY migrator/postgres /app/migrator/postgres

RUN chown -R appuser:appuser /app

USER appuser:appuser

ENTRYPOINT ["/app/vibes-migrator"]
