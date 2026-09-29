.PHONY: build run test test-race lint vet fixtures web test-scripts lint-scripts

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

# Host-side scripts (ADR-025): a fake docker stands in for the host.
test-scripts:
	bash deploy/hermes-dashboard/profile_test.sh

lint-scripts:
	shellcheck deploy/hermes-dashboard/*.sh

fixtures:
	./scripts/collect-fixtures.sh

web:
	cd web && npm ci && npm run build
