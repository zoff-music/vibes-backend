.PHONY: build frontend-build frontend-check backend frontend migrator dev install update test help gosec govulncheck docker deploy

PROJECT_NAME=$(shell basename $(CURDIR))
BACKEND_PORT ?= 8080
DEV_BACKEND_PORT ?= 8080
FRONTEND_PORT ?= 3001
FRONTEND_DIR := client/frontend/render
PNPM_VERSION := 11.5.2
PNPM := npx --yes pnpm@$(PNPM_VERSION)

## build: builds the frontend and Go binaries
build: frontend-build
	go mod download; \
	CGO_ENABLED=1 go build -ldflags '-w -s' -o main cmd/server/main.go; \
	CGO_ENABLED=0 go build -ldflags '-w -s' -o migrator-main cmd/migrator/main.go

## frontend-build: builds the frontend
frontend-build:
	$(PNPM) --dir $(FRONTEND_DIR) install
	$(PNPM) --dir $(FRONTEND_DIR) build

## frontend-check: type checks and lints the frontend
frontend-check:
	$(PNPM) --dir $(FRONTEND_DIR) install
	$(PNPM) --dir $(FRONTEND_DIR) fix
	$(PNPM) --dir $(FRONTEND_DIR) typecheck

## backend: runs the backend server only
backend:
	PORT=$(BACKEND_PORT) go run -race cmd/server/main.go

## frontend: runs the frontend dev server
frontend:
	$(PNPM) --dir $(FRONTEND_DIR) install
	$(PNPM) --dir $(FRONTEND_DIR) --filter @vibez/platform dev --host 127.0.0.1 --port $(FRONTEND_PORT)

## migrator: runs all up migrations
migrator:
	go run -race cmd/migrator/main.go

## dev: runs migrator, backend, and frontend
dev: migrator
	@set -e; \
	PORT=$(DEV_BACKEND_PORT) go run -race cmd/server/main.go & \
	BACKEND_PID=$$!; \
	trap 'kill $$BACKEND_PID 2>/dev/null || true' EXIT INT TERM; \
	$(PNPM) --dir $(FRONTEND_DIR) install; \
	VITE_API_URL=http://127.0.0.1:$(DEV_BACKEND_PORT) $(PNPM) --dir $(FRONTEND_DIR) --filter @vibez/platform dev --host 127.0.0.1 --port $(FRONTEND_PORT)

## install: fetches Go and frontend modules
install:
	go mod tidy; \
	go mod download; \
	$(PNPM) --dir $(FRONTEND_DIR) install

## update: updates Go and frontend dependencies
update:
	go get -u ./...; \
	go mod vendor; \
	go mod download; \
	go mod tidy; \
	$(PNPM) --dir $(FRONTEND_DIR) install; \
	$(PNPM) --dir $(FRONTEND_DIR) update --latest

## test: runs backend tests
test:
	go test -race ./...

## help: prints help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## gosec: Runs gosec
gosec:
	@echo "Running gosec:" && gosec -conf gosec-config.json ./...

## govulncheck: Runs govulncheck
govulncheck:
	@echo "Running govulncheck:" && govulncheck ./...

## docker: builds the production backend and frontend images
docker:
	docker build --target backend-prod -t $(PROJECT_NAME)-backend .
	docker build --target frontend-platform-prod -t $(PROJECT_NAME)-frontend-platform .

## deploy: deploys production manifests to k3s
deploy:
	.build/scripts/deploy-vibes.sh
