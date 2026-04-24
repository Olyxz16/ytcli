.PHONY: build install test clean fmt lint

BINARY_NAME=ytcli
GO=go

build:
	$(GO) build -o $(BINARY_NAME) ./cmd/ytcli

install:
	$(GO) install ./cmd/ytcli

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY_NAME)
