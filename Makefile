.PHONY: build install server test help docker dev dev-down frontend local-dev gosec govulncheck ci-lint deploy deploy-dry-run deploy-rollback

PROJECT_NAME=$(shell basename $(CURDIR))
BACKEND_DIR=backend

## build: build the application
build:
	cd $(BACKEND_DIR) && \
	go mod download; \
	go mod vendor; \
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=amd64 \
	go build -ldflags '-w -s' -o main cmd/server/main.go

## build-migrator: builds the migrator
build-migrator:
	cd migrator && go build -a -ldflags '-w -s' -o ../backend/migrator-bin main.go

## migrate-up: Runs all up migrations
migrate-up:
	cd migrator && go run main.go -db ../backend/data/vibes.db

## migrate-down: Runs one down migration step by default (use STEPS=N for more)
migrate-down:
	cd migrator && go run main.go -db ../backend/data/vibes.db -down -steps $(if $(STEPS),$(STEPS),1)

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

## local-dev: Runs backend + frontend locally with env ports set and Caddy for SSL
local-dev:
	@echo "Stopping any existing dev processes..."
	@-lsof -ti :3000,3001,8080 | xargs kill -9 2>/dev/null || true
	@echo "Ensuring dependencies are up to date..."
	@cd frontend && bun install
	@echo "Starting local development services..."
	@sh -c 'trap "kill 0" INT TERM EXIT; \
	PORT=8080 sh -c "cd migrator && go run main.go -db ../backend/data/vibes.db && cd ../backend && exec go run cmd/server/main.go" & \
	PORT=3000 sh -c "cd frontend && exec bun dev" & \
	PORT=3001 sh -c "cd frontend && exec bun --filter @vibez/cast dev" & \
	exec caddy run --config Caddyfile & \
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
