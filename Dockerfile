ARG GO_BASE_IMAGE=golang:1.26.5-bookworm

FROM ${GO_BASE_IMAGE} AS builder

WORKDIR /src

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates build-essential \
	&& rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags '-w -s' -o /out/main cmd/server/main.go

FROM debian:bookworm-slim AS prod

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/* \
	&& groupadd --system appuser \
	&& useradd --system --gid appuser --home-dir /app appuser

WORKDIR /app

COPY --from=builder /out/main /app/main

RUN mkdir -p /app/data \
	&& chown -R appuser:appuser /app

USER appuser:appuser

ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/app/main"]
