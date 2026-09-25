.PHONY: run test vet fmt build

run:
	go run ./cmd/server

build:
	go build -o bin/server-monitor ./cmd/server

fmt:
	gofmt -w ./cmd ./internal

test:
	go test ./...

vet:
	go vet ./...
