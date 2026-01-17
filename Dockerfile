FROM golang:1.25.5 AS backend-dev

WORKDIR /go/src/github.com/zoff-music/vibes

COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

COPY . .

WORKDIR /go/src/github.com/zoff-music/vibes/backend
EXPOSE 8000
CMD ["go", "run", "cmd/server/main.go"]

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
CMD ["bun", "run", "dev:web", "--", "--host", "0.0.0.0", "--port", "19006"]

# Create production image for application with needed files
FROM scratch

# Copy SSL certificates from builder
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy app user from builder
COPY --from=backend-builder /etc/passwd /etc/passwd
COPY --from=backend-builder /etc/group /etc/group

# Copy binary from builder
COPY --from=backend-builder /go/src/github.com/zoff-music/vibes/backend/main /app/main

USER appuser:appuser

EXPOSE 8000

WORKDIR /app
ENTRYPOINT [ "/app/main" ]
