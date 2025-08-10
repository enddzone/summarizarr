# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Summarizarr is an AI-powered Signal message summarizer that connects to Signal groups via WebSocket, stores messages in SQLite, and generates periodic AI summaries using local AI (Ollama) or cloud AI (OpenAI). The application consists of a Go backend, Next.js frontend, and Signal CLI integration running as containerized services.

## Architecture

**Signal Integration**: 
- WebSocket connection to `signal-cli-rest-api` via `internal/signal/client.go`
- Only processes group messages (ignores DMs)
- Enhanced message support: quotes, reactions, and different message types

**Database Layer**: 
- SQLite with schema in `schema.sql` 
- Stores users, groups, messages, and summaries
- Foreign key relationships and automatic migration system

**AI Processing**: 
- Unified AI client in `internal/ai/client.go` with backend selection
- Supports Ollama (`internal/ollama/`) and OpenAI (`internal/openai/`)
- Configurable scheduling via `internal/ai/scheduler.go`
- Centralized prompt management and anonymization

**API Server**: 
- HTTP server in `internal/api/server.go` on port 8081
- RESTful endpoints for summaries, groups, export, and Signal configuration

**Frontend**: 
- Next.js 15 application in `web/` directory
- shadcn/ui components with responsive design
- Default "Today" date filter for summaries

## Development Commands

### Local Development (Recommended)
```bash
# Initial setup
make dev-setup

# Start all services locally
make all          # Signal container + Go backend + Next.js frontend

# Individual services  
make signal       # Start signal-cli-rest-api container only
make backend      # Run Go backend locally
make frontend     # Run Next.js frontend with hot reload

# Utilities
make status       # Check service status
make stop         # Stop all services
make clean        # Remove build artifacts
```

### Testing
```bash
# Backend tests
go test ./...
make test-backend

# Frontend tests  
cd web && npm test
make test-frontend

# Custom testing scripts in cmd/testing/
go run cmd/testing/main.go
```

### Build & Deploy
```bash
# Local Go build
go build -o summarizarr cmd/summarizarr/main.go

# Docker development
make docker       # Full stack with docker-compose

# Frontend build
cd web && npm run build
cd web && npm run lint
cd web && npm run type-check
```

## Configuration

All configuration uses environment variables. For local development:

1. Copy `.env.example` to `.env`
2. Set required variables:
   - `SIGNAL_PHONE_NUMBER` (required)
   - `AI_BACKEND` (local/openai)
   - `OPENAI_API_KEY` (if using OpenAI)
   - `SUMMARIZATION_INTERVAL` (e.g., 1h, 12h)

The Makefile automatically loads `.env` for local development.

## Key Go Patterns

**Modern Go 1.24+ practices**:
- Structured logging with `slog` 
- Context propagation for cancellation
- Interface segregation for testability
- Graceful shutdown with `signal.NotifyContext`

**Database interfaces**: Small, focused interfaces like:
```go
type DB interface {
    SaveMessage(msg *signal.Envelope) error
    GetMessagesForSummary(groupID int64, start, end int64) ([]MessageForSummary, error)
}
```

**Message processing flow**:
1. WebSocket → Enhanced `signal.Envelope` → Database
2. Scheduler → AI summarization with anonymization → Store summary
3. API serves summaries with name substitution

## Dependencies

- `modernc.org/sqlite`: Pure Go SQLite driver
- `github.com/sashabaranov/go-openai`: OpenAI API client  
- `github.com/coder/websocket`: WebSocket client for Signal
- Next.js 15 with TypeScript and shadcn/ui components

## Database Schema

- Foreign keys: messages → users/groups, summaries → groups
- Unix timestamps for all time fields
- Enhanced message fields: quotes, reactions, message types
- Automatic schema migration on startup