.PHONY: build build-server install test clean run run-server fmt tidy

VERSION := $(shell cat VERSION)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o cg ./cmd/chitra

build-server:
	go build -ldflags "$(LDFLAGS)" -o cg-server ./cmd/chitra-server

build-all: build build-server

install:
	go install ./cmd/chitra

test:
	go test ./...

clean:
	rm -f cg cg-server
	go clean

run:
	go run ./cmd/chitra

run-server:
	DB_TYPE=sqlite DB_DSN=chitragupta.db STORAGE_PATH=./storage/packages PORT=8080 \
	go run ./cmd/chitra-server

fmt:
	go fmt ./...

tidy:
	go mod tidy
