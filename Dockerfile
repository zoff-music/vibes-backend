FROM golang:1.25.5 AS backend-dev

WORKDIR /go/src/github.com/zoff-music/vibes

COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

COPY . .

WORKDIR /go/src/github.com/zoff-music/vibes/backend
EXPOSE 8080
CMD ["go", "run", "cmd/server/main.go"]

FROM golang:1.25.5 AS migrator-dev

WORKDIR /go/src/github.com/zoff-music/vibes

COPY migrator/go.mod migrator/go.sum ./migrator/
RUN cd migrator && go mod download

COPY . .

WORKDIR /go/src/github.com/zoff-music/vibes/migrator
ENV CGO_ENABLED=1
CMD ["go", "run", "main.go", "-db", "/data/db/vibes.db"]

FROM golang:1.25.5 AS migrator-builder
RUN apt-get update && apt-get install -y build-essential
WORKDIR /go/src/github.com/zoff-music/vibes/migrator
COPY migrator/go.mod migrator/go.sum ./
RUN go mod download
COPY migrator .
RUN CGO_ENABLED=1 go build -a -ldflags '-w -s' -o migrator-bin main.go

FROM golang:1.25.5 AS backend-builder

# Install cross-compilation tools for CGO
RUN apt-get update && apt-get install -y build-essential

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

# Frontend builder stage
FROM oven/bun:1.2.0 AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/bun.lock ./
COPY frontend/apps ./apps
COPY frontend/packages ./packages
RUN bun install --frozen-lockfile

# Platform build
FROM frontend-builder AS platform-builder
ARG VITE_CAST_APP_ID
ARG VITE_CAST_RECEIVER_URL
ARG VITE_APP_TITLE
WORKDIR /app/apps/platform
COPY frontend/apps/platform .
# Build CSS, client, and server bundles
RUN VITE_CAST_APP_ID=$VITE_CAST_APP_ID \
    VITE_CAST_RECEIVER_URL=$VITE_CAST_RECEIVER_URL \
    VITE_APP_TITLE=$VITE_APP_TITLE \
    bun run build

# Cast build
FROM frontend-builder AS cast-builder
ARG VITE_CAST_APP_ID
ARG VITE_CAST_RECEIVER_URL
ARG VITE_API_URL
ARG VITE_APP_TITLE
WORKDIR /app/apps/cast
COPY frontend/apps/cast .
RUN VITE_CAST_APP_ID=$VITE_CAST_APP_ID \
    VITE_CAST_RECEIVER_URL=$VITE_CAST_RECEIVER_URL \
    VITE_API_URL=$VITE_API_URL \
    VITE_APP_TITLE=$VITE_APP_TITLE \
    bun run build

# Platform production image
FROM oven/bun:1.2.0-slim AS frontend-platform-prod
WORKDIR /app

# Copy workspace structure
COPY --from=frontend-builder /app/package.json .
COPY --from=frontend-builder /app/bun.lock .

# Copy workspace packages with their node_modules
COPY --from=frontend-builder /app/packages ./packages

# Copy the built platform app
COPY --from=platform-builder /app/apps/platform ./apps/platform

# Copy node_modules from builder (already has all dependencies)
COPY --from=frontend-builder /app/node_modules ./node_modules

# Set working directory to platform app
WORKDIR /app/apps/platform

EXPOSE 3000
CMD ["bun", "run", "start"]

# Cast production image - just provides static files
FROM debian:bookworm-slim AS frontend-cast-prod
WORKDIR /app
COPY --from=cast-builder /app/apps/cast/dist ./static
# Keep container running so assets can be accessed via volumes
CMD ["tail", "-f", "/dev/null"]

# Production image for migrator
FROM debian:bookworm-slim AS migrator-prod
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=migrator-builder /go/src/github.com/zoff-music/vibes/migrator/migrator-bin /app/migrator-bin
COPY --from=migrator-builder /go/src/github.com/zoff-music/vibes/migrator/migrations /app/migrations

RUN chmod +x /app/migrator-bin
RUN mkdir -p /data/db

ENTRYPOINT ["/app/migrator-bin"]

# Production image for backend application
FROM debian:bookworm-slim AS backend-prod
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=backend-builder /go/src/github.com/zoff-music/vibes/backend/main /app/main

RUN chmod +x /app/main
RUN mkdir -p /data/db

EXPOSE 8080
ENTRYPOINT ["/bin/sh", "-c", "/app/main"]

