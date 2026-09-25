.PHONY: run test race bench vet fmt verify

run:
	go run ./cmd/server

test:
	go test ./...

race:
	go test -race ./...

bench:
	go test -bench=. -benchmem ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

verify:
	go test ./...
	go test -race ./...
	go vet ./...
