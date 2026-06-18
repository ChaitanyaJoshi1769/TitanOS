.PHONY: help install dev dev-build dev-down dev-logs lint format test test-unit test-integration clean type-check build generate seed

# Color output
YELLOW := \033[0;33m
GREEN := \033[0;32m
BLUE := \033[0;34m
NC := \033[0m # No Color

help:
	@echo "$(BLUE)Titan Infrastructure OS - Development Commands$(NC)"
	@echo ""
	@echo "$(YELLOW)Setup:$(NC)"
	@echo "  make install            Install all dependencies"
	@echo ""
	@echo "$(YELLOW)Development:$(NC)"
	@echo "  make dev                Start development stack (docker-compose up)"
	@echo "  make dev-build          Start stack with rebuild (docker-compose up --build)"
	@echo "  make dev-down           Stop development stack"
	@echo "  make dev-logs           Tail logs from all services"
	@echo ""
	@echo "$(YELLOW)Code Quality:$(NC)"
	@echo "  make lint               Run linters (ESLint, Go fmt, etc.)"
	@echo "  make format             Format code (Prettier, gofmt, etc.)"
	@echo "  make type-check         Run TypeScript type checking"
	@echo ""
	@echo "$(YELLOW)Testing:$(NC)"
	@echo "  make test               Run all tests"
	@echo "  make test-unit          Run unit tests only"
	@echo "  make test-integration   Run integration tests only"
	@echo ""
	@echo "$(YELLOW)Build:$(NC)"
	@echo "  make build              Build all packages and services"
	@echo "  make generate           Run code generation"
	@echo ""
	@echo "$(YELLOW)Database:$(NC)"
	@echo "  make db-migrate         Run database migrations"
	@echo "  make db-seed            Seed database with demo data"
	@echo "  make db-reset           Reset database (schema + seed)"
	@echo ""
	@echo "$(YELLOW)Maintenance:$(NC)"
	@echo "  make clean              Remove all build artifacts and node_modules"
	@echo "  make docker-clean       Clean Docker images and volumes"
	@echo ""

install:
	@echo "$(GREEN)Installing dependencies...$(NC)"
	npm install --workspaces

dev:
	@echo "$(GREEN)Starting Titan OS development stack...$(NC)"
	docker-compose up

dev-build:
	@echo "$(GREEN)Building and starting Titan OS development stack...$(NC)"
	docker-compose up --build

dev-down:
	@echo "$(GREEN)Stopping Titan OS development stack...$(NC)"
	docker-compose down

dev-logs:
	@echo "$(GREEN)Tailing logs from all services...$(NC)"
	docker-compose logs -f

lint:
	@echo "$(GREEN)Running linters...$(NC)"
	npm run lint --workspaces --if-present

format:
	@echo "$(GREEN)Formatting code...$(NC)"
	npm run format --workspaces --if-present

type-check:
	@echo "$(GREEN)Running type checking...$(NC)"
	npm run type-check --workspaces --if-present

test:
	@echo "$(GREEN)Running all tests...$(NC)"
	npm run test --workspaces --if-present

test-unit:
	@echo "$(GREEN)Running unit tests...$(NC)"
	npm run test:unit --workspaces --if-present

test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	npm run test:integration --workspaces --if-present

build:
	@echo "$(GREEN)Building all packages and services...$(NC)"
	npm run build --workspaces --if-present

generate:
	@echo "$(GREEN)Running code generation...$(NC)"
	npm run generate --workspaces --if-present

db-migrate:
	@echo "$(GREEN)Running database migrations...$(NC)"
	@echo "PostgreSQL migrations can be run via services when ready"

db-seed:
	@echo "$(GREEN)Seeding database with demo data...$(NC)"
	@echo "Seed scripts will be created in services/scripts/"

db-reset: dev-down
	@echo "$(GREEN)Resetting database...$(NC)"
	docker volume rm $$(docker volume ls -q | grep postgres)
	make dev

clean:
	@echo "$(GREEN)Cleaning build artifacts...$(NC)"
	npm run clean --workspaces --if-present
	rm -rf dist build out coverage

docker-clean: dev-down
	@echo "$(GREEN)Cleaning Docker images and volumes...$(NC)"
	docker system prune -f
	docker volume prune -f

.DEFAULT_GOAL := help
