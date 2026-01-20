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
# Build cast app explicitly if not covered by root build (it might be if root build is workspace aware, but simpler to be explicit or ensure it is)
# "bun run build" at root likely runs "turbo run build" or similar if configured, or just recursive.
# Let's assume root build script covers it OR run it explicitly. 
# Checking root package.json would be good, but safe to run explicit build for cast here.
RUN cd apps/cast && bun run build

# Frontend production image - serve static files
# Frontend production image - serve static files with Caddy
FROM caddy:2.9.1-alpine AS frontend-prod

WORKDIR /srv

# Copy Caddy configuration
COPY frontend/Caddyfile /etc/caddy/Caddyfile

# Copy built static files
COPY --from=frontend-builder /app/apps/platform/dist ./app
COPY --from=frontend-builder /app/apps/cast/dist ./cast-app

EXPOSE 3000
# Default command for caddy image is to run with /etc/caddy/Caddyfile

# Create production image for backend application with needed files
# Using Debian slim for glibc compatibility (Go CGO binaries need glibc)
FROM debian:bookworm-slim AS backend-prod

# Install ca-certificates for HTTPS and curl for healthcheck
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy binaries
COPY --from=backend-builder /go/src/github.com/zoff-music/vibes/backend/main /app/main
COPY --from=migrator-builder /go/src/github.com/zoff-music/vibes/migrator/migrator-bin /app/migrator-bin
COPY --from=migrator-builder /go/src/github.com/zoff-music/vibes/migrator/migrations /app/migrations

# Ensure binaries are executable
RUN chmod +x /app/main /app/migrator-bin

# Create data directory
RUN mkdir -p /app/data

EXPOSE 8080

WORKDIR /app

# Run migrations then start the app
ENTRYPOINT ["/bin/sh", "-c", "/app/migrator-bin -db /app/data/vibes.db && exec /app/main"]
