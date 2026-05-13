.PHONY: build install test clean fmt lint

BINARY_NAME=tkt
GO=go
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS=-ldflags "-X github.com/Olyxz16/tkt/internal/cmdx.Version=$(VERSION) -X github.com/Olyxz16/tkt/internal/cmdx.Commit=$(COMMIT)"

build:
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/tkt

install:
	$(GO) install $(LDFLAGS) ./cmd/tkt

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY_NAME)
