.PHONY: build install test clean fmt lint

BINARY_NAME=ytcli
GO=go
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS=-ldflags "-X github.com/Olyxz16/ytcli/internal/cmdx.Version=$(VERSION) -X github.com/Olyxz16/ytcli/internal/cmdx.Commit=$(COMMIT)"

build:
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/ytcli

install:
	$(GO) install $(LDFLAGS) ./cmd/ytcli

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY_NAME)
