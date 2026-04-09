.PHONY: help build run dev dev-watch test clean

help:
	@echo "Go Backend - Makefile commands"
	@echo "  make build        - Build binary"
	@echo "  make run          - Run server"
	@echo "  make dev          - Run in debug mode"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"

build:
	mkdir -p ./bin
	go build -o ./bin/server ./cmd/server

run: build
	./bin/server

dev:
	DEV_MODE=true LOG_LEVEL=debug go run ./cmd/server/main.go

dev-watch:
	@command -v air >/dev/null 2>&1 || go install github.com/cosmtrek/air@latest
	air

test:
	go test -v ./...

clean:
	rm -rf ./bin
	go clean
