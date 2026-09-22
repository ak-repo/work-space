.PHONY: help run dev build test vet fmt check clean \
	docker-up docker-up-build docker-down docker-down-volumes docker-logs docker-ps docker-reset \
	db-migrate migrate db-seed seed db-shell db-reset reset

APP_NAME ?= order-delivery-engine
DOCKER_COMPOSE ?= docker-compose

DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_NAME ?= order_delivery

PSQL := PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME)

help:
	@printf "\n$(APP_NAME) commands:\n\n"
	@printf "  make run                 Run API locally with go run\n"
	@printf "  make build               Build all Go packages\n"
	@printf "  make test                Run Go tests\n"
	@printf "  make vet                 Run go vet\n"
	@printf "  make fmt                 Format Go files\n"
	@printf "  make check               Format, test, vet, and build\n"
	@printf "  make docker-up           Start Docker services\n"
	@printf "  make docker-up-build     Build and start Docker services\n"
	@printf "  make docker-down         Stop Docker services\n"
	@printf "  make docker-down-volumes Stop Docker services and remove DB volume\n"
	@printf "  make docker-logs         Follow Docker logs\n"
	@printf "  make db-migrate          Apply SQL migrations to local DB\n"
	@printf "  make db-seed             Seed sample drivers\n"
	@printf "  make db-shell            Open psql shell\n"
	@printf "  make db-reset            Drop Docker DB volume, restart, migrate, seed\n"
	@printf "  make reset               Full Docker reset alias\n\n"

run:
	go run ./cmd/server

dev: run

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal

check: fmt test vet build

clean:
	rm -f $(APP_NAME)

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-up-build:
	$(DOCKER_COMPOSE) up --build -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-down-volumes:
	$(DOCKER_COMPOSE) down -v

docker-logs:
	$(DOCKER_COMPOSE) logs -f

docker-ps:
	$(DOCKER_COMPOSE) ps

docker-reset: docker-down-volumes docker-up-build

db-migrate:
	$(PSQL) -f migrations/001_init.sql

migrate: db-migrate

db-seed:
	$(PSQL) -f scripts/seed.sql

seed: db-seed

db-shell:
	$(PSQL)

db-reset:
	$(DOCKER_COMPOSE) down -v
	$(DOCKER_COMPOSE) up --build -d postgres
	$(DOCKER_COMPOSE) exec postgres sh -c 'until pg_isready -U $(DB_USER) -d $(DB_NAME); do sleep 1; done'
	$(PSQL) -f migrations/001_init.sql
	$(PSQL) -f scripts/seed.sql
	$(DOCKER_COMPOSE) up --build -d server

reset: db-reset
