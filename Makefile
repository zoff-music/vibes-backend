.PHONY: build backend migrator dev install update test help gosec govulncheck docker integrationtest docs

PROJECT_NAME=$(shell basename $(CURDIR))
BACKEND_PORT ?= 8080
DEV_BACKEND_PORT ?= 8080

## build: builds the Go binaries
build:
	go mod download; \
	CGO_ENABLED=1 go build -ldflags '-w -s' -o main cmd/server/main.go; \
	CGO_ENABLED=0 go build -ldflags '-w -s' -o vibes-migrator cmd/migrator/main.go

## backend: runs the backend server only
backend:
	PORT=$(BACKEND_PORT) go run -race cmd/server/main.go

## migrator: runs all up migrations
migrator:
	go run -race cmd/migrator/main.go

## dev: runs migrator and backend
dev: migrator
	PORT=$(DEV_BACKEND_PORT) go run -race cmd/server/main.go

## install: fetches Go modules
install:
	go mod tidy; \
	go mod download

## update: updates Go dependencies
update:
	go get -u ./...; \
	go mod vendor; \
	go mod download; \
	go mod tidy

## test: runs backend tests
test:
	go test -race ./...

## integrationtest: runs migrator up and down against local Postgres
integrationtest:
	@set -e; \
	trap 'echo "Stopping postgres..." && docker compose down -v' EXIT INT TERM; \
	docker compose down -v 2>/dev/null || true; \
	echo "Starting postgres..."; \
	docker compose up -d shared-local; \
	echo "Waiting for postgres..."; \
	sleep 5; \
	echo "Running migrations up..."; \
	docker compose run --rm --build migrator up; \
	echo "Running migrations down..."; \
	docker compose run --rm migrator down

## docs: generates database table documentation using tbls
docs:
	@set -e; \
	trap 'echo "Stopping postgres..." && docker compose down -v' EXIT INT TERM; \
	docker compose down -v 2>/dev/null || true; \
	echo "Starting postgres..."; \
	docker compose up -d shared-local; \
	echo "Waiting for postgres..."; \
	sleep 5; \
	echo "Running migrations..."; \
	docker compose run --rm --build migrator up; \
	echo "Generating database documentation..."; \
	rm -rf docs/db; \
	mkdir -p docs/db; \
	docker compose run --rm tbls

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

## docker: builds the production backend image
docker:
	docker build --target backend-prod -t $(PROJECT_NAME)-backend .
	docker build --target migrator-prod -t $(PROJECT_NAME)-migrator .
