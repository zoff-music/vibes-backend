FROM golang:1.25.5 AS backend-dev

WORKDIR /go/src/github.com/zoff-music/vibes

COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

COPY . .

WORKDIR /go/src/github.com/zoff-music/vibes/backend
EXPOSE 8080
CMD ["go", "run", "cmd/server/main.go"]

FROM golang:1.25.5 AS migrator-builder
WORKDIR /go/src/github.com/zoff-music/vibes/migrator
COPY migrator/go.mod migrator/go.sum ./
RUN go mod download
COPY migrator .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -a -ldflags '-w -s' -o migrator-bin main.go

FROM golang:1.25.5 AS backend-builder

# See https://stackoverflow.com/a/55757473/12429735
# Create an app-user with no shell, no login, no home directory,
# a fixed UID (10001), and a fixed GUID (10001), then get ca-certificates.crt
ENV USER=appuser
ENV UID=10001

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}" \
    && apt-get update \
    && apt-get install -y ca-certificates --no-install-recommends

# Install govulncheck and gosec
# Tags:
# - https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck?tab=versions
# - https://github.com/securego/gosec/tags
RUN go install "golang.org/x/vuln/cmd/govulncheck@latest"
ENV GOSEC_VERSION="v2.22.10"
RUN curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(go env GOPATH)/bin "${GOSEC_VERSION}"

WORKDIR /go/src/github.com/zoff-music/vibes

COPY . .

# Run tests, then build the application
RUN make test \
 && make build

FROM oven/bun:1.2.0 AS frontend-dev

WORKDIR /app

COPY frontend/package.json frontend/bun.lock ./
COPY frontend/apps ./apps
COPY frontend/packages ./packages

RUN bun install

COPY frontend/. .

EXPOSE 19006
CMD ["bun", "run", "dev:web", "--", "--host", "localhost", "--port", "19006"]

# Frontend production build
FROM oven/bun:1.2.0 AS frontend-builder

WORKDIR /app

COPY frontend/package.json frontend/bun.lock ./
COPY frontend/apps ./apps
COPY frontend/packages ./packages

RUN bun install --frozen-lockfile

COPY frontend/. .

RUN bun run build

# Frontend production image
FROM oven/bun:1.2.0-slim AS frontend-prod

WORKDIR /app

COPY --from=frontend-builder /app/apps/platform/dist ./dist
COPY --from=frontend-builder /app/package.json ./

ENV PORT=3000
EXPOSE 3000

CMD ["bun", "run", "start"]

# Create production image for backend application with needed files
# Using alpine instead of scratch to have a shell for migrations
FROM alpine:3.21 AS backend-prod

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create app user
RUN adduser -D -g '' -u 10001 appuser

# Copy binaries
COPY --from=backend-builder /go/src/github.com/zoff-music/vibes/backend/main /app/main
COPY --from=migrator-builder /go/src/github.com/zoff-music/vibes/migrator/migrator-bin /app/migrator-bin
COPY --from=migrator-builder /go/src/github.com/zoff-music/vibes/migrator/migrations /app/migrations

# Create data directory
RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

WORKDIR /app

# Run migrations then start the app
ENTRYPOINT ["/bin/sh", "-c", "/app/migrator-bin -db /app/data/vibes.db && exec /app/main"]
