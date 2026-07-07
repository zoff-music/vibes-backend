.PHONY: build backend dev install update test help gosec govulncheck docker

PROJECT_NAME=$(shell basename $(CURDIR))
BACKEND_PORT ?= 8080
DEV_BACKEND_PORT ?= 8080

## build: builds the Go binaries
build:
	go mod download; \
	CGO_ENABLED=1 go build -ldflags '-w -s' -o main cmd/server/main.go

## backend: runs the backend server only
backend:
	PORT=$(BACKEND_PORT) go run -race cmd/server/main.go

## dev: runs backend
dev:
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
