# Variables
DOCKER_COMPOSE = docker compose
PNPM = pnpm
TURBO = pnpm turbo

.PHONY: install dev build test lint up down logs ps restart build-docker clean help

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install: ## Install all dependencies
	$(PNPM) install

dev: ## Start dev servers for all apps
	$(PNPM) run dev

build: ## Build all apps
	$(PNPM) run build

test: ## Run tests for all apps
	$(PNPM) run test

lint: ## Run lint for all apps
	$(PNPM) run lint

up: ## Start observability stack and apps in docker
	$(DOCKER_COMPOSE) up --build -d

down: ## Stop docker containers
	$(DOCKER_COMPOSE) down -v

logs: ## Follow docker logs
	$(DOCKER_COMPOSE) logs -f

ps: ## Show running docker containers
	$(DOCKER_COMPOSE) ps

restart: ## Restart docker containers
	$(DOCKER_COMPOSE) restart

build-docker: ## Build docker images
	$(DOCKER_COMPOSE) build

build-api: ## Build only the API
	$(PNPM) --filter @otel-monorepo/api build

build-web: ## Build only the Web app
	$(PNPM) --filter @otel-monorepo/web build

dev-api: ## Run only the API in dev mode
	$(PNPM) --filter @otel-monorepo/api dev

dev-web: ## Run only the Web app in dev mode
	$(PNPM) --filter @otel-monorepo/web dev

stress: ## Run TUI stress generator (requires stack up). Args: d=60s c=10
	cd scripts/stress && go run . -d $(or $(d),60s) -c $(or $(c),10)

clean: ## Remove node_modules, build artifacts and docker volumes
	$(TURBO) run clean || true
	rm -rf node_modules apps/*/node_modules packages/*/node_modules dist .turbo
	$(DOCKER_COMPOSE) down -v --remove-orphans
