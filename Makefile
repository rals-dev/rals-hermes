.PHONY: build run test lint vet fixtures

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$$(git rev-parse --short HEAD)" -o bin/bff ./cmd/bff

run:
	go run ./cmd/bff

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

fixtures:
	./scripts/collect-fixtures.sh
