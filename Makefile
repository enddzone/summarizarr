# Makefile for Summarizarr Local Development
# Fast local development with Go, Next.js, and Docker

.DEFAULT_GOAL := help

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

# Load environment variables from .env if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

help: ## Show this help message
	@echo "$(GREEN)Summarizarr Local Development$(NC)"
	@echo ""
	@echo "$(YELLOW)Usage:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

signal: ## Start signal-cli-rest-api container
	@echo "$(YELLOW)Starting Signal CLI REST API container...$(NC)"
	docker compose up -d signal-cli-rest-api
	@echo "$(GREEN)Signal container started on port 8080$(NC)"

backend: ## Run Go backend locally (requires signal container)
	@echo "$(YELLOW)Starting Go backend locally...$(NC)"
	@if [ ! -f .env ]; then echo "$(RED)Warning: .env not found. Copy .env.example to .env$(NC)"; fi
	go run cmd/summarizarr/main.go

frontend: ## Run Next.js frontend locally with hot reload
	@echo "$(YELLOW)Starting Next.js frontend with hot reload...$(NC)"
	cd web && npm install && BACKEND_URL=http://localhost:8081 npm run dev

all: signal ## Start all services locally (signal container + Go backend + Next.js frontend)
	@echo "$(YELLOW)Starting all services for local development...$(NC)"
	@echo "$(GREEN)Signal container will start first...$(NC)"
	@sleep 3
	@echo "$(GREEN)Starting backend and frontend in parallel...$(NC)"
	@$(MAKE) -j2 backend frontend

docker: ## Run full stack with docker compose
	@echo "$(YELLOW)Starting full stack with Docker Compose...$(NC)"
	docker compose up --build -d
	@echo "$(GREEN)Full stack started. Frontend: http://localhost:3000, Backend: http://localhost:8081$(NC)"

status: ## Show status of all services
	@echo "$(YELLOW)Service Status:$(NC)"
	@echo "Signal Container:"
	@docker compose ps signal-cli-rest-api || echo "  $(RED)Not running$(NC)"
	@echo "Backend (Go):"
	@curl -s http://localhost:8081/api/summaries > /dev/null && echo "  $(GREEN)Running on :8081$(NC)" || echo "  $(RED)Not running$(NC)"
	@echo "Frontend (Next.js):"
	@curl -s http://localhost:3000 > /dev/null && echo "  $(GREEN)Running on :3000$(NC)" || echo "  $(RED)Not running$(NC)"

stop: ## Stop all local services and containers
	@echo "$(YELLOW)Stopping all services...$(NC)"
	docker compose down
	@pkill -f "go run main.go" || true
	@pkill -f "npm run dev" || true
	@echo "$(GREEN)All services stopped$(NC)"

clean: stop ## Remove build artifacts and stop containers
	@echo "$(YELLOW)Cleaning up build artifacts...$(NC)"
	rm -rf web/node_modules
	rm -rf web/.next
	rm -f summarizarr
	rm -f summarizarr.db
	rm -f data/summarizarr.db
	docker compose down --volumes --remove-orphans
	@echo "$(GREEN)Cleanup complete$(NC)"

logs: ## Show logs for all docker services
	docker compose logs -f

logs-signal: ## Show logs for signal-cli-rest-api
	docker compose logs -f signal-cli-rest-api

# Development helpers
dev-setup: ## Initial setup for development
	@echo "$(YELLOW)Setting up development environment...$(NC)"
	@if [ ! -f .env ]; then cp .env.example .env && echo "$(GREEN)Created .env from .env.example$(NC)"; fi
	cd web && npm install
	@echo "$(GREEN)Development setup complete. Edit .env with your values$(NC)"

test-backend: ## Test Go backend
	go test ./...

test-frontend: ## Test Next.js frontend
	cd web && npm test

.PHONY: help signal backend frontend all docker status stop clean logs logs-signal dev-setup test-backend test-frontend
