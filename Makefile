.PHONY: build install server test help docker godoc gosec govulncheck ci-lint

PROJECT_NAME=$(shell basename $(CURDIR))

## build: build the application
build:
	go mod download; \
	go mod vendor; \
	CGO_ENABLED=0 \
	GOOS=linux \
	GOARCH=amd64 \
	go build -a -ldflags '-w -s' -o main cmd/server/main.go

## install: fetches go modules
install:
	go mod tidy; \
	go mod download

## server: runs the server with -race
server:
	go run -race cmd/server/main.go

## test: runs tests
test:
	go test -race ./...

## help: prints help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## docker: Builds and runs the app via the project dockerfile, importing the .env-file as environment variables.
docker:
	docker build --progress=plain -t $(PROJECT_NAME) .
	docker run --rm --name $(PROJECT_NAME) -p 8000:8000 --env-file .env -it $(PROJECT_NAME)

## gosec: Runs gosec
gosec:
	@echo "Running gosec:" && gosec -conf gosec-config.json ./...

## govulncheck: Runs govulncheck
govulncheck:
	@echo "Running govulncheck:" && govulncheck ./...
