ARG GO_BASE_IMAGE=golang:1.27.1-bookworm

FROM ${GO_BASE_IMAGE} AS builder

WORKDIR /src

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go run github.com/swaggo/swag/cmd/swag init -d cmd/server,server,server/internal/handler,vibe -g main.go -o swaggerdocs --parseInternal \
	&& CGO_ENABLED=0 go build -trimpath -ldflags '-w -s' -o /out/main cmd/server/main.go

RUN groupadd --system --gid 65532 appuser \
	&& useradd --system --uid 65532 --gid appuser --home-dir /app appuser \
	&& mkdir -p /out/data

FROM scratch AS prod

WORKDIR /app

COPY --from=builder /out/main /app/main
COPY --from=builder /src/static /app/static
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /zoneinfo.zip
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder --chown=65532:65532 /out/data /app/data

USER 65532:65532

ENV PORT=8080
ENV ZONEINFO=/zoneinfo.zip

EXPOSE 8080

ENTRYPOINT ["/app/main"]
