.PHONY: build install server test help docker dev dev-down frontend local-dev gosec govulncheck ci-lint deploy deploy-dry-run deploy-rollback

PROJECT_NAME=$(shell basename $(CURDIR))
BACKEND_DIR=backend

## build: build the application
build:
	cd $(BACKEND_DIR) && \
	go mod download; \
	go mod vendor; \
	CGO_ENABLED=1 \
	go build -ldflags '-w -s' -o main cmd/server/main.go

## build-migrator: builds the migrator
build-migrator:
	cd migrator && go build -a -ldflags '-w -s' -o migrator-bin main.go

## migrate-up: Runs all up migrations
migrate-up:
	mkdir -p data/db
	cd migrator && go run main.go -db ../data/db/vibes.db

## migrate-down: Runs one down migration step by default (use STEPS=N for more)
migrate-down:
	mkdir -p data/db
	cd migrator && go run main.go -db ../data/db/vibes.db -down -steps $(if $(STEPS),$(STEPS),1)


## install: fetches go modules
install:
	cd $(BACKEND_DIR) && \
	go mod tidy; \
	go mod download

## server: runs the server with -race
server:
	cd $(BACKEND_DIR) && \
	go run -race cmd/server/main.go

## test: runs tests
test:
	cd $(BACKEND_DIR) && \
	go test -race ./...

## help: prints help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## docker: Builds and runs backend + frontend via docker compose.
docker:
	docker compose up --build

## dev: Runs backend + frontend via docker compose
dev:
	docker compose up --build

## dev-down: Stops docker compose services
dev-down:
	docker compose down

## frontend: Runs frontend dev server locally
frontend:
	cd frontend && bun dev

## setup-caddy: Installs Caddy and trusts local CA (MacOS only)
setup-caddy:
	@command -v caddy >/dev/null 2>&1 || (echo "Caddy not found. Installing via Homebrew..." && brew install caddy)
	@echo "Ensuring local Caddy CA is trusted (may require password)..."
	@echo "Validating Caddyfile..."
	@caddy validate --config Caddyfile || (echo "Caddyfile validation failed" && exit 1)
	@echo "Starting Caddy temporarily to fetch CA info..."
	@caddy start --config Caddyfile >/dev/null 2>&1 || true
	@sleep 2
	@sudo caddy trust || echo "Failed to trust CA, continuing anyway..."
	@caddy stop >/dev/null 2>&1 || true


## local-dev: Runs backend + frontend locally with env ports set and Caddy for SSL
local-dev: setup-caddy
	@echo "Stopping any existing dev processes..."
	@-lsof -ti :3000,3001,8080 | xargs kill -9 2>/dev/null || true
	@echo "Ensuring dependencies are up to date..."
	@cd frontend && bun install
	@echo "Ensuring database directory exists..."
	@mkdir -p data/db
	@echo "Starting local development services..."
	@sh -c 'trap "kill 0" INT TERM EXIT; \
	PORT=8080 sh -c "cd migrator && go run main.go -db ../data/db/vibes.db && cd ../backend && DATABASE_PATH=../data/db/vibes.db exec go run cmd/server/main.go" & \
	VITE_API_URL=http://localhost:8080 sh -c "cd frontend/apps/platform && FORCE_COLOR=1 exec bun run dev" & \
	VITE_API_URL=https://localhost sh -c "cd frontend/apps/cast && FORCE_COLOR=1 exec bun run dev" & \
	CAST_DEV_MODE=true exec caddy run --config Caddyfile & \
	wait'



## gosec: Runs gosec
gosec:
	@echo "Running gosec:" && cd $(BACKEND_DIR) && gosec -conf ../gosec-config.json ./...

## govulncheck: Runs govulncheck
govulncheck:
	@echo "Running govulncheck:" && cd $(BACKEND_DIR) && govulncheck ./...

## deploy: Run blue-green deployment to production
deploy:
	.github/deploy/deploy.sh

## deploy-dry-run: Preview deployment without executing
deploy-dry-run:
	.github/deploy/deploy.sh --dry-run

## deploy-rollback: Rollback to previous deployment
deploy-rollback:
	.github/deploy/deploy.sh --rollback
