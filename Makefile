.PHONY: build serve dev install update test docs help security gosec govulncheck docker

PROJECT_NAME=$(shell basename $(CURDIR))
PORT ?= 8080
DEV_PORT ?= 8080
GOSEC_VERSION ?= 71577268350e1e0f2f1491de1ef53f1227b43c1f
GOVULNCHECK_VERSION ?= latest

## build: builds the Go binaries
build: security
	go mod download
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
	GOFLAGS=-mod=mod go get -u ./...
	go mod tidy
	go mod vendor
	go mod download

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

## security: runs both security checks before building
security: gosec govulncheck

## gosec: runs gosec at the configured upstream commit
gosec: docs
	GOFLAGS=-mod=mod go run github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION) -conf gosec-config.json ./...

## govulncheck: Runs govulncheck
govulncheck: docs
	GOFLAGS=-mod=mod go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

## docker: builds the production image
docker:
	docker build --target prod -t $(PROJECT_NAME) .
