.PHONY: run test simulator fmt docker-up docker-down clean
run:
	go run ./cmd/server

test:
	go test ./...

simulator:
	go run ./cmd/simulator -count 20

fmt:
	gofmt -w ./cmd ./internal

docker-up:
	docker compose up --build

docker-down:
	docker compose down

clean:
	rm -rf data
