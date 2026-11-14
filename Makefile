.PHONY: help build test test-unit vet fmt sqlc docker-up docker-down docker-build docker-run-dev docker-run-prod migrate-up migrate-down run run-dev run-prod run-worker run-worker-dev run-worker-prod web-dev web-build

-include .env
export

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/tavrocred?sslmode=disable
MIGRATIONS_DIR ?= services/api/migrations

# Branch and environment detection
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "development")

ifeq ($(filter $(GIT_BRANCH),prod production),$(GIT_BRANCH))
    DETECTED_ENV := production
else
    DETECTED_ENV := development
endif

# Allow overriding via BRANCH (e.g. make run BRANCH=production) or ENV
ifdef BRANCH
    ifeq ($(filter $(BRANCH),prod production),$(BRANCH))
        TARGET_ENV := production
    else
        TARGET_ENV := development
    endif
else ifdef ENV
    TARGET_ENV := $(ENV)
else
    TARGET_ENV := $(DETECTED_ENV)
endif

help: ## Show this help message
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

run: ## Run API server automatically matching current branch (dev/prod)
	@echo "==> Branch: $(GIT_BRANCH) | Target Mode: $(TARGET_ENV)"
	@if [ "$(TARGET_ENV)" = "production" ]; then \
		$(MAKE) run-prod; \
	else \
		$(MAKE) run-dev; \
	fi

run-dev: ## Run API server in development mode (hot go run)
	@echo "==> Starting API server in development mode..."
	cd services/api && ENV=development MIGRATIONS_DIR=migrations go run ./cmd/api

run-prod: ## Build and run API server in production mode
	@echo "==> Building optimized API binary for production..."
	cd services/api && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../../bin/api ./cmd/api
	@echo "==> Starting API server in production mode..."
	cd services/api && ENV=production MIGRATIONS_DIR=migrations ../../bin/api

run-worker: ## Run background worker automatically matching current branch (dev/prod)
	@echo "==> Branch: $(GIT_BRANCH) | Target Mode: $(TARGET_ENV)"
	@if [ "$(TARGET_ENV)" = "production" ]; then \
		$(MAKE) run-worker-prod; \
	else \
		$(MAKE) run-worker-dev; \
	fi

run-worker-dev: ## Run background worker in development mode
	@echo "==> Starting background worker in development mode..."
	cd services/api && ENV=development go run ./cmd/worker

run-worker-prod: ## Build and run background worker in production mode
	@echo "==> Building optimized worker binary for production..."
	cd services/api && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../../bin/worker ./cmd/worker
	@echo "==> Starting background worker in production mode..."
	cd services/api && ENV=production ../../bin/worker

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

docker-up: ## Start Docker Compose containers in background
	docker compose up -d

docker-down: ## Stop Docker Compose containers
	docker compose down

docker-build: ## Build Docker Compose images
	docker compose build

docker-run-dev: ## Start Docker Compose containers for development
	ENV=development docker compose up --build

docker-run-prod: ## Start Docker Compose containers for production
	ENV=production docker compose up --build -d

web-dev: ## Run Next.js web application in development mode (port 3000)
	cd apps/web && PORT=3000 npm run dev -- -p 3000

web-build: ## Build Next.js web application for production
	cd apps/web && npm run build

migrate-up: ## Apply all database migrations
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down: ## Rollback the most recent migration
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down
