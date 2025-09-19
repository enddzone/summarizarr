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
	docker compose -f compose.dev.yaml up -d signal-cli
	@echo "$(GREEN)Signal container started on port 8080$(NC)"

backend: ## Run Go backend locally with SQLCipher (requires signal container)
	@echo "$(YELLOW)Starting Go backend locally on :8081 with SQLCipher...$(NC)"
	@if [ ! -f .env ]; then echo "$(RED)Warning: .env not found. Copy .env.example to .env$(NC)"; fi
	if command -v pkg-config >/dev/null 2>&1; then \
		CGO_ENABLED=1 \
		CGO_CFLAGS="$$(pkg-config --cflags sqlcipher) -DSQLITE_HAS_CODEC" \
		CGO_LDFLAGS="$$(pkg-config --libs sqlcipher) $$(pkg-config --libs openssl)" \
	LISTEN_ADDR=:8081 go run -tags="sqlite_crypt libsqlite3" cmd/summarizarr/main.go; \
	else \
		CGO_ENABLED=1 \
		CGO_CFLAGS="-I$$(brew --prefix sqlcipher)/include/sqlcipher -DSQLITE_HAS_CODEC" \
		CGO_LDFLAGS="-L$$(brew --prefix sqlcipher)/lib -lsqlcipher -L$$(brew --prefix openssl@3)/lib -lssl -lcrypto" \
	LISTEN_ADDR=:8081 go run -tags="sqlite_crypt libsqlite3" cmd/summarizarr/main.go; \
	fi

backend-bg: ## Run Go backend in background with SQLCipher and local config
	@echo "$(YELLOW)Starting Go backend in background on :8081 with SQLCipher...$(NC)"
	@if [ ! -f .env ]; then echo "$(RED)Warning: .env not found. Copy .env.example to .env$(NC)"; fi
	if command -v pkg-config >/dev/null 2>&1; then \
		export DATABASE_PATH=./data/summarizarr.db SIGNAL_URL=localhost:8080 LISTEN_ADDR=:8081 CGO_ENABLED=1 \
		CGO_CFLAGS="$$(pkg-config --cflags sqlcipher) -DSQLITE_HAS_CODEC" \
		CGO_LDFLAGS="$$(pkg-config --libs sqlcipher) $$(pkg-config --libs openssl)" && \
	nohup go run -tags="sqlite_crypt libsqlite3" cmd/summarizarr/main.go > backend.log 2>&1 & echo $$! > backend.pid; \
	else \
		export DATABASE_PATH=./data/summarizarr.db SIGNAL_URL=localhost:8080 LISTEN_ADDR=:8081 CGO_ENABLED=1 \
		CGO_CFLAGS="-I$$(brew --prefix sqlcipher)/include/sqlcipher -DSQLITE_HAS_CODEC" \
		CGO_LDFLAGS="-L$$(brew --prefix sqlcipher)/lib -lsqlcipher -L$$(brew --prefix openssl@3)/lib -lssl -lcrypto" && \
	nohup go run -tags="sqlite_crypt libsqlite3" cmd/summarizarr/main.go > backend.log 2>&1 & echo $$! > backend.pid; \
	fi
	@echo "$(GREEN)Backend started in background (PID: $$(cat backend.pid))$(NC)"

frontend: ## Run Next.js frontend locally with hot reload
	@echo "$(YELLOW)Starting Next.js frontend with hot reload...$(NC)"
	cd web && npm install && npm run dev

frontend-bg: ## Run Next.js frontend in background
	@echo "$(YELLOW)Starting Next.js frontend in background...$(NC)"
	@cd web && ( [ -d node_modules ] || npm install ) && ( nohup npm run dev > ../frontend.log 2>&1 & echo $$! > ../frontend.pid )
	@echo "$(GREEN)Frontend started in background (PID: $$(cat frontend.pid))$(NC)"

all: signal ## Start all services locally (signal container + Go backend + Next.js frontend)
	@echo "$(YELLOW)Starting all services for local development...$(NC)"
	@echo "$(GREEN)Signal container started, waiting for readiness...$(NC)"
	@sleep 5
	@echo "$(GREEN)Starting backend with local database...$(NC)"
	@$(MAKE) backend-bg
	@sleep 3
	@echo "$(GREEN)Starting frontend with hot reload...$(NC)"
	@$(MAKE) frontend-bg
	@sleep 3
	@echo "$(GREEN)✓ All services started successfully!$(NC)"
	@echo ""
	@echo "🔗 Service URLs:"
	@echo "   Backend:  http://localhost:8081"
	@echo "   Frontend: http://localhost:3000"
	@echo "   Signal:   http://localhost:8080"
	@echo ""
	@echo "📊 Database: Using existing data in ./data/summarizarr.db"
	@echo "📱 Signal:   Using config in ./signal-cli-config/"
	@echo ""
	@echo "💡 Commands:"
	@echo "   make status  - Check service health"
	@echo "   make stop    - Stop all services"
	@echo "   tail -f backend.log frontend.log - View logs"

docker: ## Run development stack with docker compose (dev compose file)
	@echo "$(YELLOW)Starting development stack with Docker Compose (compose.dev.yaml)...$(NC)"
	docker compose -f compose.dev.yaml up --build -d
	@echo "$(GREEN)Development stack started. Backend: http://localhost:8081$(NC)"
	@echo "$(GREEN)Adminer (DB viewer): http://localhost:8083$(NC)"
	@echo "$(GREEN)Dozzle (logs viewer): http://localhost:8084$(NC)"

prod: ## Run production example stack with pre-built image
	@echo "$(YELLOW)Starting production example stack...$(NC)"
	docker compose -f compose.yaml up -d
	@echo "$(GREEN)Production stack started. Backend: http://localhost:8081$(NC)"

status: ## Show status of all services (dev compose only)
	@echo "$(YELLOW)Service Status:$(NC)"
	@echo "Signal Container:"
	@docker compose -f compose.dev.yaml ps signal-cli 2>/dev/null || echo "  $(RED)Not running$(NC)"
	@echo "Summarizarr Container:"
	@docker compose -f compose.dev.yaml ps summarizarr 2>/dev/null || echo "  $(RED)Not running$(NC)"
	@echo "Backend (Go):"
	@curl -s http://localhost:8081/api/summaries > /dev/null && echo "  $(GREEN)Running on :8081$(NC)" || echo "  $(RED)Not running$(NC)"
	@echo "Frontend (Next.js):"
	@curl -s http://localhost:3000 > /dev/null && echo "  $(GREEN)Running on :3000$(NC)" || echo "  $(RED)Not running$(NC)"
	@echo "Development Tools:"
	@docker compose -f compose.dev.yaml ps adminer 2>/dev/null | grep -q "Up" && echo "  $(GREEN)Adminer (DB viewer) on :8083$(NC)" || echo "  $(RED)Adminer not running$(NC)"
	@docker compose -f compose.dev.yaml ps dozzle 2>/dev/null | grep -q "Up" && echo "  $(GREEN)Dozzle (logs viewer) on :8084$(NC)" || echo "  $(RED)Dozzle not running$(NC)"

stop: ## Stop all local services and containers (dev compose only)
	@echo "$(YELLOW)Stopping all services...$(NC)"
	@# Stop background processes using PID files
	@if [ -f backend.pid ]; then \
		echo "Stopping backend (PID: $$(cat backend.pid))..."; \
		kill $$(cat backend.pid) 2>/dev/null || true; \
		rm -f backend.pid; \
	fi
	@if [ -f frontend.pid ]; then \
		echo "Stopping frontend (PID: $$(cat frontend.pid))..."; \
		kill $$(cat frontend.pid) 2>/dev/null || true; \
		rm -f frontend.pid; \
	fi
	@# Stop Docker containers
	@docker compose -f compose.dev.yaml down 2>/dev/null || true
	@# Fallback: kill processes by name
	@pkill -f "go run.*cmd/summarizarr/main.go" 2>/dev/null || true
	@pkill -f "npm run dev" 2>/dev/null || true
	@echo "$(GREEN)✓ All services stopped$(NC)"

clean: stop ## Remove build artifacts and stop containers
	@echo "$(YELLOW)Cleaning up build artifacts...$(NC)"
	@rm -rf web/node_modules web/.next
	@rm -f summarizarr backend.pid frontend.pid backend.log frontend.log
	@docker compose -f compose.dev.yaml down --volumes --remove-orphans 2>/dev/null || true
	@echo "$(GREEN)✓ Cleanup complete$(NC)"
	@echo "$(YELLOW)Note: Database (./data/) and Signal config (./signal-cli-config/) preserved$(NC)"

logs: ## Show logs for all docker services (dev compose only)
	@docker compose -f compose.dev.yaml logs -f

logs-signal: ## Show logs for signal-cli-rest-api (dev compose only)
	@docker compose -f compose.dev.yaml logs -f signal-cli

# Development helpers
dev-setup: ## Initial setup for development
	@echo "$(YELLOW)Setting up development environment...$(NC)"
	@if [ ! -f .env ]; then cp .env.example .env && echo "$(GREEN)Created .env from .env.example$(NC)"; fi
	cd web && npm install
	@echo "$(GREEN)Development setup complete. Edit .env with your values$(NC)"

# SQLCipher encryption key management
install-sqlcipher: ## Install SQLCipher for development (macOS)
	@if command -v brew >/dev/null 2>&1; then \
		brew install sqlcipher; \
		echo "$(GREEN)SQLCipher installed via Homebrew$(NC)"; \
	else \
		echo "$(RED)Please install SQLCipher manually for your platform$(NC)"; \
	fi

dev-key: ## Generate development encryption key
	@if [ ! -f .dev_encryption_key ]; then \
		openssl rand -hex 32 > .dev_encryption_key; \
		echo "$(GREEN)Generated development encryption key in .dev_encryption_key$(NC)"; \
	fi
	@if ! grep -q "SQLCIPHER_ENCRYPTION_KEY=" .env 2>/dev/null; then \
		echo "SQLCIPHER_ENCRYPTION_KEY=$$(cat .dev_encryption_key)" >> .env; \
		echo "$(GREEN)Added SQLCIPHER_ENCRYPTION_KEY to .env$(NC)"; \
	fi

dev-setup-encrypted: dev-key ## Setup development environment with encryption enabled
	@echo "SQLCIPHER_ENCRYPTION_ENABLED=true" >> .env
	@echo "CGO_ENABLED=1" >> .env
	@echo "$(GREEN)Development environment setup with encryption enabled$(NC)"

prod-key: ## Generate production encryption key
	@mkdir -p docker/secrets
	@openssl rand -hex 32 > docker/secrets/encryption_key
	@chmod 600 docker/secrets/encryption_key
	@echo "$(GREEN)Generated production encryption key in docker/secrets/encryption_key$(NC)"
	@echo "$(RED)IMPORTANT: Back up this key securely!$(NC)"

test-backend: ## Test Go backend with SQLCipher
	@echo "$(YELLOW)Running backend tests with SQLCipher support...$(NC)"
	TEST_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
	CGO_ENABLED=1 go test -tags="sqlite_crypt" ./...

test-frontend: ## Test Next.js frontend
	cd web && npm test

build-frontend: ## Build Next.js frontend and copy to internal/frontend/static/
	@echo "$(YELLOW)Building Next.js frontend...$(NC)"
	cd web && npm install
	@echo "$(YELLOW)Building with production config for static export...$(NC)"
	cd web && cp next.config.mjs next.config.mjs.bak
	cd web && cp next.config.prod.mjs next.config.mjs
	cd web && npm run build
	cd web && mv next.config.mjs.bak next.config.mjs
	@echo "$(YELLOW)Copying frontend build to internal/frontend/static/...$(NC)"
	rm -rf internal/frontend/static/
	cp -r web/out/ internal/frontend/static/
	@echo "$(GREEN)Frontend build complete$(NC)"

# SQLCipher builds
build-encrypted: build-frontend ## Build with SQLCipher support (CGO enabled)
	@echo "$(YELLOW)Building Go backend with SQLCipher support...$(NC)"
	if command -v pkg-config >/dev/null 2>&1; then \
		CGO_ENABLED=1 \
		CGO_CFLAGS="$$(pkg-config --cflags sqlcipher) -DSQLITE_HAS_CODEC" \
		CGO_LDFLAGS="$$(pkg-config --libs sqlcipher) $$(pkg-config --libs openssl)" \
	go build -tags="sqlite_crypt libsqlite3" -o summarizarr cmd/summarizarr/main.go; \
	else \
		echo "$(RED)pkg-config not found. Trying with Homebrew paths...$(NC)"; \
		CGO_ENABLED=1 \
		CGO_CFLAGS="-I$$(brew --prefix sqlcipher)/include/sqlcipher -DSQLITE_HAS_CODEC" \
		CGO_LDFLAGS="-L$$(brew --prefix sqlcipher)/lib -lsqlcipher -L$$(brew --prefix openssl@3)/lib -lssl -lcrypto" \
	go build -tags="sqlite_crypt libsqlite3" -o summarizarr cmd/summarizarr/main.go; \
	fi
	@echo "$(GREEN)Build complete with SQLCipher: ./summarizarr$(NC)"

build: build-frontend ## Build the entire application (frontend + backend with SQLCipher)
	@echo "$(YELLOW)Building Go backend with embedded frontend and SQLCipher support...$(NC)"
	if command -v pkg-config >/dev/null 2>&1; then \
		CGO_ENABLED=1 \
		CGO_CFLAGS="$$(pkg-config --cflags sqlcipher) -DSQLITE_HAS_CODEC" \
		CGO_LDFLAGS="$$(pkg-config --libs sqlcipher) $$(pkg-config --libs openssl)" \
	go build -tags="sqlite_crypt libsqlite3" -o summarizarr cmd/summarizarr/main.go; \
	else \
		echo "$(YELLOW)pkg-config not found. Trying with Homebrew paths...$(NC)"; \
		CGO_ENABLED=1 \
		CGO_CFLAGS="-I$$(brew --prefix sqlcipher)/include/sqlcipher -DSQLITE_HAS_CODEC" \
		CGO_LDFLAGS="-L$$(brew --prefix sqlcipher)/lib -lsqlcipher -L$$(brew --prefix openssl@3)/lib -lssl -lcrypto" \
	go build -tags="sqlite_crypt libsqlite3" -o summarizarr cmd/summarizarr/main.go; \
	fi
	@echo "$(GREEN)Build complete: ./summarizarr$(NC)"

# Claude Code hooks integration
lint: ## Lint code (supports FILE= for specific files)
	@if [ -n "$(FILE)" ]; then \
		echo "Linting specific file: $(FILE)" >&2; \
		case "$(FILE)" in \
			*.go) \
				if command -v golangci-lint >/dev/null 2>&1; then \
					golangci-lint run $(FILE); \
				else \
					gofmt -w $(FILE) && go vet $(FILE); \
				fi \
				;; \
			web/*.ts|web/*.tsx|web/*.js|web/*.jsx) \
				cd web && npm run lint -- --fix $(patsubst web/%,%,$(FILE)) 2>/dev/null || npm run lint; \
				;; \
			*.ts|*.tsx|*.js|*.jsx) \
				cd web && npm run lint -- --fix ../$(FILE) 2>/dev/null || npm run lint; \
				;; \
			*) \
				echo "No linter configured for $(FILE)" >&2; \
				;; \
		esac \
	else \
		echo "Linting all files" >&2; \
		if command -v golangci-lint >/dev/null 2>&1; then \
			golangci-lint run ./...; \
		else \
			gofmt -w . && go vet ./...; \
		fi; \
		cd web && npm run lint; \
	fi

test: ## Run tests (supports FILE= for specific files)
	@if [ -n "$(FILE)" ]; then \
		echo "Testing specific file: $(FILE)" >&2; \
		case "$(FILE)" in \
			*.go) \
				go test -v -race $(dir $(FILE)); \
				;; \
			web/*.test.ts|web/*.test.tsx|web/*.test.js|web/*.test.jsx|web/*.spec.ts|web/*.spec.tsx|web/*.spec.js|web/*.spec.jsx) \
				cd web && npm test -- $(patsubst web/%,%,$(FILE)); \
				;; \
			*.test.ts|*.test.tsx|*.test.js|*.test.jsx|*.spec.ts|*.spec.tsx|*.spec.js|*.spec.jsx) \
				cd web && npm test -- ../$(FILE); \
				;; \
			web/*) \
				cd web && npm test; \
				;; \
			*) \
				go test -v -race ./...; \
				cd web && npm test; \
				;; \
		esac \
	else \
		echo "Running all tests" >&2; \
		if command -v pkg-config >/dev/null 2>&1; then \
			CGO_ENABLED=1 CGO_CFLAGS="$$(pkg-config --cflags sqlcipher) -DSQLITE_HAS_CODEC" CGO_LDFLAGS="$$(pkg-config --libs sqlcipher) $$(pkg-config --libs openssl)" \
			TEST_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
			go test -v -race -tags="sqlite_crypt" ./...; \
		else \
			go test -v -race ./...; \
		fi; \
			cd web && ( [ -d node_modules ] || npm install ) && npm test; \
	fi

.PHONY: help signal backend frontend all docker prod status stop clean logs logs-signal dev-setup install-sqlcipher dev-key dev-setup-encrypted prod-key test-backend test-frontend build-encrypted build-frontend build lint test
