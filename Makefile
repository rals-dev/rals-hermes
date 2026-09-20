.PHONY: build run test test-race lint vet fixtures web

# Builds the frontend first so it is embedded (ADR-011).
build: web
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$$(git rev-parse --short HEAD)" -o bin/bff ./cmd/bff

run:
	CGO_ENABLED=0 go run ./cmd/bff

# Pure-Go tests run everywhere; the race detector needs cgo and a working
# C toolchain, so it has its own target (CI runs it on Linux).
test:
	CGO_ENABLED=0 go test -count=1 ./...

test-race:
	go test -race -count=1 ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

fixtures:
	./scripts/collect-fixtures.sh

web:
	cd web && npm ci && npm run build
