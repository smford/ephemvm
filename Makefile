# ephemvm Makefile

BIN_DIR := bin
BINARY := $(BIN_DIR)/ephemvm
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X github.com/smford/ephemvm/internal/version.Version=$(VERSION) \
	-X github.com/smford/ephemvm/internal/version.Commit=$(COMMIT) \
	-X github.com/smford/ephemvm/internal/version.Date=$(DATE)

.PHONY: all build build-all build-linux build-linux-amd64 build-linux-arm64 build-darwin-arm64 test test-integration lint vet clean

all: vet test build

build:
	@mkdir -p $(BIN_DIR)
	go build -v -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/ephemvm

build-all: build-linux-amd64 build-linux-arm64 build-darwin-arm64

build-linux-amd64:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ephemvm-linux-amd64 ./cmd/ephemvm

build-linux-arm64:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ephemvm-linux-arm64 ./cmd/ephemvm

build-darwin-arm64:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/ephemvm-darwin-arm64 ./cmd/ephemvm

build-linux: build-linux-amd64 build-linux-arm64

test:
	go test -v -race ./internal/...

test-integration:
	go test -v -tags=integration ./...

vet:
	go vet ./...
	GOOS=linux go vet ./...

lint: vet

clean:
	rm -rf $(BIN_DIR)
