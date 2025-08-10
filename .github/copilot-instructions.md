# Copilot Instructions for Summarizarr

## Project Overview
Summarizarr is a Signal message summarizer that connects to Signal groups via WebSocket, stores messages in SQLite, and generates periodic AI summaries using local AI (Ollama) or cloud AI (OpenAI-compatible APIs). The application runs as a containerized service alongside signal-cli-rest-api.

## Architecture
- **Signal Integration**: Connects to `signal-cli-rest-api` via WebSocket (`internal/signal/client.go`) to receive real-time messages
- **Database Layer**: SQLite with schema defined in `schema.sql` - stores users, groups, messages, and summaries
- **AI Processing**: Unified AI client (`internal/ai/client.go`) with backend selection (Ollama/OpenAI) and configurable scheduling (`internal/ai/scheduler.go`)
- **Backend Abstraction**: Supports local AI (`internal/ollama/client.go`) and cloud AI (`internal/openai/client.go`) with consistent prompt handling
- **API Server**: Simple HTTP server (`internal/api/server.go`) exposing summaries endpoint on port 8081
- **Frontend**: Next.js 15 application in `web/` directory with shadcn/ui components, date filtering (default: "Today"), and responsive design
- **Docker Setup**: Multi-service compose with signal-cli-rest-api dependency and health checks

## Go 1.24+ Best Practices

### Structured Logging (slog)
- Use `slog.SetDefault()` for global logger (already implemented in `main.go`)
- Structured attributes: `slog.Info("message", "key", value)`
- Context-aware logging: `slog.InfoContext(ctx, "message")`

### Error Handling
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Use typed errors for different failure modes
- Defer error handling close to resource acquisition

### Context Usage
- Pass `context.Context` as first parameter to all async operations
- Use `context.WithCancel` for graceful shutdowns (implemented in `main.go`)
- Check `ctx.Done()` in long-running loops and goroutines

### Modern Go Patterns
- Interface segregation: small, focused interfaces (see `DB` interfaces)
- Embed `*sql.DB` in structs for type safety (`database.DB`)
- Use `time.Duration` for intervals (not strings or integers)
- Prefer `http.ServeMux` over third-party routers for simple APIs

## Key Patterns & Conventions

### Database Interface Segregation
Components use small, focused interfaces for testability:
```go
type DB interface {
    SaveMessage(msg *signal.Envelope) error
    GetMessagesForSummarization(groupID int64, start, end int64) ([]database.MessageForSummary, error)
}
```

### Message Processing Flow
1. Signal WebSocket → Enhanced `signal.Envelope` struct (with Quote/Reaction support) → Database storage
2. Scheduler runs on intervals → Fetches messages with context → AI summarization (enhanced with quote/reaction awareness) → Store summary
3. API serves summaries as JSON responses

### Prompt Management
- **Centralized Prompt**: Single `SummarizationPrompt` template in `internal/ai/client.go`
- **Message Formatting**: `FormatMessagesForLLM()` function handles all message types with anonymization
- **Backend Consistency**: Both Ollama and OpenAI backends receive identical prompts
- **Post-Processing**: User ID substitution with real names after LLM processing

### Environment Configuration
All configuration is managed via environment variables and a `.env` file for local development. For production or sensitive data, use `.env` and never commit secrets to version control. Example:

```
AI_BACKEND=openai
OPENAI_API_KEY=your_key_here
SIGNAL_PHONE_NUMBER=+18177392137
SUMMARIZATION_INTERVAL=1h
OPENAI_MODEL=gpt-4o
LOG_LEVEL=DEBUG
```

For local development, copy `.env.example` to `.env` and fill in your values. The Makefile automatically loads `.env` for all local development targets. For OpenAI setup and testing, see `OPENAI_TESTING.md`.

### Required Environment Variables
| Variable                | Required | Default   | Description                                      |
|-------------------------|----------|-----------|--------------------------------------------------|
| AI_BACKEND              | No       | local     | AI backend: 'local' (Ollama) or 'openai'        |
| OPENAI_API_KEY          | No*      | -         | OpenAI API key (*required when AI_BACKEND=openai)|
| SIGNAL_PHONE_NUMBER     | Yes      | -         | Signal phone number (e.g., +1234567890)          |
| SUMMARIZATION_INTERVAL  | No       | 12h       | How often to generate summaries (e.g., 1h, 12h)  |
| OPENAI_MODEL            | No       | gpt-4o    | OpenAI model to use                              |
| LOCAL_MODEL             | No       | llama3.2:1b | Local Ollama model to use                      |
| LOG_LEVEL               | No       | INFO      | Log level (DEBUG, INFO, WARN, ERROR)             |
| DATABASE_PATH           | No       | summarizarr.db | Database file path                          |

For local development, copy `.env.example` to `.env` and fill in your values. The Makefile automatically loads `.env` for all local development targets.

### Modern Go Patterns in Use
- **Graceful Shutdown**: `signal.NotifyContext` in `main.go`
- **Embedded Structs**: `database.DB` embeds `*sql.DB`
- **Structured Logging**: `slog` with levels and structured attributes
- **Context Propagation**: All long-running operations accept `context.Context`

## Development Workflows

### Local Development (Fast Development with Make)
For rapid development with hot reload and fast builds, use the local tooling workflow:

```bash
# Initial setup
make dev-setup    # Creates .env from .env.example and installs npm deps

# Start all services locally (automatic)
make all          # Starts signal container + Go backend + Next.js frontend in parallel

# Or start services individually
make signal       # Start signal-cli-rest-api container only
make backend      # Run Go backend locally (requires signal container)
make frontend     # Run Next.js frontend with hot reload

# Check service status
make status       # Shows running status of all services

# Stop everything
make stop         # Stops all services and containers
```

**Environment Variables**: All local development uses `.env` for configuration. Copy `.env.example` to `.env` and fill in your values.

**Key Benefits**: 
- Frontend hot reload on code changes
- Faster Go compilation (no Docker build)
- Uses same environment variables from `.env`
- Signal container still runs in Docker for stability

### Legacy Docker Development
```bash
# Full stack with Docker (slower, for production-like testing)
make docker       # Equivalent to: docker compose up --build -d

# Copy and edit environment variables (if needed)
cp .env.example .env
# Edit .env and set your values
```

### Testing
### Testing & Debugging Scripts
- All custom test and debug scripts are located in `cmd/testing/`
- `cmd/testing/parse_sample.go`: Tests Signal message parsing with sample data
- Unit tests: `go test ./...` (requires full schema including `groups` table)
- Example Signal API message format in `internal/signal/message_test.go`
- Manual testing via Docker compose (no integration tests)

### Build & Deploy
```bash
# Local build with latest Go
go build -o summarizarr cmd/summarizarr/main.go

# Docker with health checks
make docker       # Equivalent to: docker compose up --build -d
```

## Signal CLI Integration
- Requires pre-configured signal-cli data in `signal-cli-config/` volume
- Phone number is configured via the `SIGNAL_PHONE_NUMBER` environment variable (see Environment Configuration above)
- WebSocket reconnection with exponential backoff (5 retries, 5s delay)
- Only processes group messages (ignores DMs)

## Database Schema Notes
- Foreign key relationships: messages → users/groups, summaries → groups
- Timestamps stored as Unix epoch integers
- Enhanced message support: quotes (quote_id, quote_author_uuid, quote_text), reactions (reaction_emoji, reaction_target_author), message types (regular, quote, reaction)
- Schema applied on startup via `db.Init()` reading `schema.sql`
- Automatic migration system adds missing columns to existing tables
- Database stored in mounted `./data` directory for persistence

## Common Tasks
- **Add new endpoints**: Extend `internal/api/server.go` with new handlers
- **Modify AI prompts**: Update summarization logic in `internal/ai/client.go`
- **Change message filtering**: Modify `SaveMessage` logic in `internal/database/db.go`
- **Adjust scheduling**: Update interval parsing and ticker logic in scheduler
- **Frontend changes**: Modify components in `web/src/components/` and pages in `web/src/app/`

## Frontend Development
The Next.js 15 frontend is located in the `web/` directory and features:
- **shadcn/ui components**: Modern UI components for date pickers, filters, and summaries
- **Default date filter**: "Today" preset for summary filtering
- **Responsive design**: Works on desktop and mobile devices
- **TypeScript**: Fully typed components and API interactions
- **Available scripts**: `npm run lint`, `npm test`, `npm run build` (conditional in CI)

## Dependencies
- `modernc.org/sqlite`: Pure Go SQLite driver
- `github.com/sashabaranov/go-openai`: OpenAI API client
- `github.com/coder/websocket`: WebSocket client for Signal API
