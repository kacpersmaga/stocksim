.PHONY: build up down test-unit test-integration lint tidy fmt help

APP_DIR := ./app
PORT    ?= 8080

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build Docker images (parallel)
	docker compose build --parallel

up: ## Start all services on PORT (default: 8080)
	PORT=$(PORT) ./start.sh $(PORT)

down: ## Stop all services
	docker compose down

logs: ## Tail logs for all services
	docker compose logs -f

test-unit: ## Run unit tests with race detector
	cd $(APP_DIR) && go test ./internal/... -race -count=1 -v

test-integration: ## Run integration tests (requires Docker)
	cd $(APP_DIR) && go test ./internal/store/... -tags=integration -race -v -timeout=120s

test-all: test-unit test-integration ## Run all tests

lint: ## Run golangci-lint
	cd $(APP_DIR) && golangci-lint run ./...

fmt: ## Format Go code
	cd $(APP_DIR) && gofmt -w .

tidy: ## Tidy Go module dependencies
	cd $(APP_DIR) && go mod tidy

cover: ## Run unit tests with coverage report
	cd $(APP_DIR) && go test ./internal/... -race -count=1 -coverprofile=coverage.out && go tool cover -html=coverage.out

ps: ## Show running containers
	docker compose ps
