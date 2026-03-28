.PHONY: all setup run build test vet lint

all: vet lint test

setup:
	git config core.hooksPath .githooks

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...
