.PHONY: help build test test-unit vet fmt sqlc docker-up docker-down docker-build migrate-up migrate-down

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/tavrocred?sslmode=disable
MIGRATIONS_DIR ?= services/api/migrations

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build API and Worker binaries
	cd services/api && go build -o ../../bin/api ./cmd/api
	cd services/api && go build -o ../../bin/worker ./cmd/worker

test: ## Run all tests
	cd services/api && go test -v ./...

vet: ## Run go vet
	cd services/api && go vet ./...

fmt: ## Format Go source code
	cd services/api && gofmt -w .

sqlc: ## Generate database queries using sqlc
	cd services/api && sqlc generate

docker-up: ## Start Docker Compose containers
	docker compose up -d

docker-down: ## Stop Docker Compose containers
	docker compose down

docker-build: ## Build Docker Compose images
	docker compose build

migrate-up: ## Apply all database migrations
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down: ## Rollback the most recent migration
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

