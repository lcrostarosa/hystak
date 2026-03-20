VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build test test-cover cover-html lint snapshot clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o hystak .

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

cover-html: test-cover
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	golangci-lint run

snapshot:
	goreleaser build --snapshot --clean

clean:
	rm -f hystak coverage.out coverage.html
	rm -rf dist/
