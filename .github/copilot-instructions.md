# Copilot Instructions for Summarizarr

## Project Overview
Summarizarr is a Signal message summarizer that connects to Signal groups via WebSocket, stores messages in SQLite, and generates periodic AI summaries using OpenAI's API. The application runs as a containerized service alongside signal-cli-rest-api.

## Architecture
- **Signal Integration**: Connects to `signal-cli-rest-api` via WebSocket (`internal/signal/client.go`) to receive real-time messages
- **Database Layer**: SQLite with schema defined in `schema.sql` - stores users, groups, messages, and summaries
- **AI Processing**: OpenAI client (`internal/ai/client.go`) with configurable scheduling (`internal/ai/scheduler.go`)
- **API Server**: Simple HTTP server (`internal/api/server.go`) exposing summaries endpoint on port 8081
- **Docker Setup**: Two-service compose with signal-cli-rest-api dependency and health checks

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

### Environment Configuration
All configuration is managed via environment variables and a `.env` file for local development. Example:

```
OPENAI_API_KEY=your_key_here
SIGNAL_PHONE_NUMBER=+18177392137
SUMMARIZATION_INTERVAL=1h
OPENAI_MODEL=gpt-4o
LOG_LEVEL=DEBUG
```

### Required Environment Variables
| Variable                | Required | Default   | Description                                      |
|-------------------------|----------|-----------|--------------------------------------------------|
| OPENAI_API_KEY          | Yes      | -         | OpenAI API key for AI summarization              |
| SIGNAL_PHONE_NUMBER     | Yes      | -         | Signal phone number (e.g., +1234567890)          |
| SUMMARIZATION_INTERVAL  | No       | 12h       | How often to generate summaries (e.g., 1h, 12h)  |
| OPENAI_MODEL            | No       | gpt-4o    | OpenAI model to use                              |
| LOG_LEVEL               | No       | INFO      | Log level (DEBUG, INFO, WARN, ERROR)             |
| DATABASE_PATH           | No       | summarizarr.db | Database file path                          |

For local development, copy `.env.example` to `.env` and fill in your values.

### Modern Go Patterns in Use
- **Graceful Shutdown**: `signal.NotifyContext` in `main.go`
- **Embedded Structs**: `database.DB` embeds `*sql.DB`
- **Structured Logging**: `slog` with levels and structured attributes
- **Context Propagation**: All long-running operations accept `context.Context`

## Development Workflows

### Local Development
```bash
# Start signal-cli-rest-api dependency
docker-compose up signal-cli-rest-api

# Copy and edit environment variables
cp .env.example .env
# Edit .env and set your values

# Run with development settings (uses .env automatically)
go run cmd/summarizarr/main.go
```

### Testing
### Testing & Debugging Scripts
- All custom test and debug scripts are located in `cmd/testing/`
- `cmd/testing/parse_sample.go`: Tests Signal message parsing with sample data
- Unit tests: `go test ./...`
- Example Signal API message format in `internal/signal/message_test.go`
- Manual testing via Docker compose (no integration tests)

### Build & Deploy
```bash
# Local build with latest Go
go build -o summarizarr cmd/summarizarr/main.go

# Docker with health checks
docker-compose up --build
```

## Signal CLI Integration
- Requires pre-configured signal-cli data in `signal-cli-config/` volume
- Phone number is configured via the `SIGNAL_PHONE_NUMBER` environment variable (see Environment Configuration below)
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

## Dependencies
- `modernc.org/sqlite`: Pure Go SQLite driver
- `github.com/sashabaranov/go-openai`: OpenAI API client
- `github.com/coder/websocket`: WebSocket client for Signal API
