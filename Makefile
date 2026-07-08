.PHONY: build serve dev install update test docs help gosec govulncheck docker

PROJECT_NAME=$(shell basename $(CURDIR))
PORT ?= 8080
DEV_PORT ?= 8080

## build: builds the Go binaries
build: docs
	go mod download; \
	CGO_ENABLED=1 GOFLAGS=-mod=mod go build -ldflags '-w -s' -o main cmd/server/main.go

## serve: runs the server
serve: docs
	PORT=$(PORT) GOFLAGS=-mod=mod go run -race cmd/server/main.go

## dev: runs the server
dev: docs
	PORT=$(DEV_PORT) GOFLAGS=-mod=mod go run -race cmd/server/main.go

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

## test: runs tests
test: docs
	GOFLAGS=-mod=mod go test -race ./...

## docs: generates Swagger documentation
docs:
	GOFLAGS=-mod=mod go run github.com/swaggo/swag/cmd/swag init -d cmd/server,server,server/internal/handler,vibe -g main.go -o swaggerdocs --parseInternal

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

## docker: builds the production image
docker:
	docker build --target prod -t $(PROJECT_NAME) .
