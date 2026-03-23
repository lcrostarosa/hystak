BINARY := hystak
GOFLAGS := -trimpath
LDFLAGS := -s -w
CGO_ENABLED := 0

.PHONY: build test test-race test-all test-update test-cover lint e2e snapshot clean

build:
	CGO_ENABLED=$(CGO_ENABLED) go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test -short ./...

test-race:
	go test -race ./...

test-all:
	go test ./...

test-update:
	UPDATE_GOLDEN=1 go test ./internal/tui/...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed on:"; gofmt -l .; exit 1)
	go vet ./...
	staticcheck ./... 2>/dev/null || true

e2e: build
	@PATH="$(PWD):$$PATH" bash e2e/run_vhs_tests.sh

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -f $(BINARY) coverage.out coverage.html
