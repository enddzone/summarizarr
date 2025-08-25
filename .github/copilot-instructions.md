# Copilot Instructions for Summarizarr

## Project Overview
Summarizarr is a Signal message summarizer that connects to Signal groups via WebSocket, stores messages in SQLite, and generates periodic AI summaries using multiple AI providers. Supports local AI (Ollama), cloud AI (OpenAI), and other OpenAI-compatible providers (Groq, Gemini via proxy, Claude via proxy). The application runs as a containerized service alongside signal-cli-rest-api.

## Architecture
- **Signal Integration**: Connects to `signal-cli-rest-api` via WebSocket (`internal/signal/client.go`) to receive real-time messages
- **Database Layer**: SQLCipher-encrypted SQLite with schema defined in `schema.sql` - stores users, groups, messages, summaries, and authentication data with mandatory encryption and automatic key management
    - Encryption at rest via SQLCipher; key rotation is not supported in-app
- **Authentication System**: Session-based web authentication with separate `auth_users` table (distinct from Signal users) and persistent login state stored in `sessions` table
- **AI Processing**: Unified AI client (`internal/ai/client.go`) with multi-provider support and configurable scheduling (`internal/ai/scheduler.go`)
- **Backend Abstraction**: Supports local AI (Ollama) and multiple OpenAI-compatible cloud providers with consistent prompt handling
- **API Server**: HTTP server (`internal/api/server.go`) on port 8081 with authentication middleware and protected routes for summaries, groups, export, and Signal configuration
    - No encryption key rotation endpoint
- **Frontend**: Next.js 15 application in `web/` directory with shadcn/ui components, date filtering (default: "Today"), responsive design, and built-in authentication with login/logout and protected routes
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
All configuration is managed via environment variables and a `.env` file for local development. For production or sensitive data, use `.env` and never commit secrets to version control. 

**Multi-Provider Support**: Use `AI_PROVIDER` to select between local, openai, groq, gemini, or claude.

Example configurations:

```bash
# Local AI (Ollama)
AI_PROVIDER=local
LOCAL_MODEL=llama3.2:1b
SIGNAL_PHONE_NUMBER=+18177392137
SUMMARIZATION_INTERVAL=1h

# OpenAI
AI_PROVIDER=openai
OPENAI_API_KEY=sk-proj-xxxxx
OPENAI_MODEL=gpt-4o

# Groq
AI_PROVIDER=groq
GROQ_API_KEY=gsk_xxxxx
GROQ_MODEL=llama3-8b-8192

# Gemini (requires proxy)
AI_PROVIDER=gemini
GEMINI_API_KEY=xxxxx
GEMINI_MODEL=gemini-2.0-flash
GEMINI_BASE_URL=http://localhost:8000/hf/v1

# Claude (requires proxy)
AI_PROVIDER=claude
CLAUDE_API_KEY=sk-ant-xxxxx
CLAUDE_MODEL=claude-3-sonnet
CLAUDE_BASE_URL=http://localhost:8000/openai/v1

# Encryption: always enabled
# Development: Key auto-generated at ./data/encryption.key (0600)
# Production: Provide key via Docker secret mounted at /run/secrets/encryption_key
```

For local development, copy `.env.example` to `.env` and fill in your values. The Makefile automatically loads `.env` for all local development targets. For OpenAI setup and testing, see `OPENAI_TESTING.md`.

### SQLCipher Database Encryption

Encryption is mandatory and handled automatically:
- Development: Key is generated on first run and saved to `./data/encryption.key` with `0600` permissions.
- Production: Provide the key via a Docker secret mounted at `/run/secrets/encryption_key`.
 - Rotation: Not supported. Keys are created/loaded automatically; rotate externally if required (service downtime recommended).

 

**Build Requirements**:
- CGO_ENABLED=1 must be set
- Use build tag: `-tags="sqlite_crypt"`
- SQLCipher library must be installed (Homebrew on macOS: `brew install sqlcipher`)

### Required Environment Variables
| Variable                        | Required | Default   | Description                                      |
|---------------------------------|----------|-----------|--------------------------------------------------|
| AI_PROVIDER                     | No       | local     | AI provider: local, openai, groq, gemini, claude |
| SIGNAL_PHONE_NUMBER             | Yes      | -         | Signal phone number (e.g., +1234567890)          |
| SUMMARIZATION_INTERVAL          | No       | 12h       | How often to generate summaries (e.g., 1h, 12h)  |
| LOG_LEVEL                       | No       | INFO      | Log level (DEBUG, INFO, WARN, ERROR)             |
| DATABASE_PATH                   | No       | summarizarr.db | Database file path                          |
<!-- Rotation removed: ENCRYPTION_KEY_ROTATION_INTERVAL no longer used -->

#### Provider-Specific Variables
| Provider | API Key Variable | Model Variable | Base URL Variable |
|----------|------------------|----------------|-------------------|
| local    | -                | LOCAL_MODEL (llama3.2:1b) | OLLAMA_HOST |
| openai   | OPENAI_API_KEY   | OPENAI_MODEL (gpt-4o) | OPENAI_BASE_URL |
| groq     | GROQ_API_KEY     | GROQ_MODEL (llama3-8b-8192) | GROQ_BASE_URL |
| gemini   | GEMINI_API_KEY   | GEMINI_MODEL (gemini-2.0-flash) | GEMINI_BASE_URL |
| claude   | CLAUDE_API_KEY   | CLAUDE_MODEL (claude-3-sonnet) | CLAUDE_BASE_URL |

For local development, copy `.env.example` to `.env` and fill in your values. The Makefile automatically loads `.env` for all local development targets.

### Authentication System
The application includes a complete web authentication system:

**Backend Components**:
- **Session Management**: `internal/auth/sessions.go` - SQLite-backed session storage using SCS library
- **User Management**: `internal/auth/user.go` - User storage with bcrypt password hashing  
- **Middleware**: `internal/auth/middleware.go` - Authentication middleware for protected routes
- **API Endpoints**: `internal/api/auth.go` - Login, logout, registration, and user info endpoints

**Database Schema**:
- **auth_users table**: Stores web authentication users (separate from Signal users)
- **sessions table**: Stores session data for persistent login state
- **Indexes**: Email and session expiry indexes for performance

**Frontend Components**:
- **Auth Context**: `web/src/contexts/auth-context.tsx` - React Context for authentication state
- **Login Form**: `web/src/components/auth/login-form.tsx` - shadcn/ui login component
- **Protected Routes**: `web/src/components/auth/protected-route.tsx` - Route protection wrapper
- **Session Management**: Automatic session checking and credential inclusion

**API Endpoints**:
- `POST /api/auth/login` - User authentication
- `POST /api/auth/logout` - Session termination
- `GET /api/auth/me` - Current user information
- `POST /api/auth/register` - User registration

**Protected Routes**: All main API endpoints require authentication:
- `/api/summaries` - Summary data access
- `/api/groups` - Group information
- `/api/export` - Data export functionality

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

# Start all services locally (non-blocking with background processes)
make all          # Starts signal container + Go backend + Next.js frontend in parallel

# Or start services individually
make signal       # Start signal-cli-rest-api container only
make backend      # Run Go backend locally with SQLCipher (blocking)
make backend-bg   # Run Go backend in background with SQLCipher and PID management
make frontend     # Run Next.js frontend with hot reload (blocking)
make frontend-bg  # Run Next.js frontend in background with API proxying

# Process management and monitoring
make status       # Check service health and URLs
make stop         # Stop all services and clean up processes
```

**Environment Variables**: All local development uses `.env` for configuration. Copy `.env.example` to `.env` and fill in your values.

**Key Benefits**: 
- Frontend hot reload on code changes
- Faster Go compilation (no Docker build)
- Uses same environment variables from `.env`
- Signal container still runs in Docker for stability
- Background process management with PID files and log files

### Process Management
The Makefile includes sophisticated process management for local development:

**Background Processes**: 
- `make all` runs backend and frontend as background processes with PID files
- Backend PID stored in `backend.pid`, frontend PID in `frontend.pid`
- Log output redirected to `backend.log` and `frontend.log`

**Service Monitoring**:
- `make status` shows health of all services with URLs
- Checks signal-cli-rest-api container status
- Displays backend/frontend process status and service URLs

**Process Cleanup**:
- `make stop` properly terminates all processes using PID files
- Removes PID files after stopping processes
- Stops signal-cli container
- Data preservation: Database (`./data/`) and Signal config (`./signal-cli-config/`) preserved

**Service URLs**:
- **Frontend (Development)**: http://localhost:3000 - Next.js dev server with hot reload
- **Backend API**: http://localhost:8081 - Go backend with embedded frontend
- **Signal CLI**: http://localhost:8080 - Signal WebSocket service

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
- Rotation tests: located under `internal/encryption`, `internal/database`, and `internal/api` (integration)
- Example Signal API message format in `internal/signal/message_test.go`
- Manual testing via Docker compose (no integration tests)

### Build & Deploy
```bash
# Local build with SQLCipher support (requires CGO)
CGO_ENABLED=1 go build -tags="sqlite_crypt" -o summarizarr cmd/summarizarr/main.go

# Docker with health checks
make docker       # Equivalent to: docker compose up --build -d
```

## Signal CLI Integration
- Requires pre-configured signal-cli data in `signal-cli-config/` volume
- Phone number is configured via the `SIGNAL_PHONE_NUMBER` environment variable (see Environment Configuration above)
- WebSocket reconnection with exponential backoff (5 retries, 5s delay)
- Only processes group messages (ignores DMs)

## Database Schema Notes
- **Encryption**: SQLCipher with AES-256 encryption (mandatory)
- **Key Management**: Automatic — dev key at `./data/encryption.key`; production key via `/run/secrets/encryption_key`
 - **Rotation Metadata**: Not used; no rotation tables are created in new installs

- **Build Requirements**: CGO_ENABLED=1 and sqlite_crypt build tag for SQLCipher support
- Foreign key relationships: messages → users/groups, summaries → groups
- Timestamps stored as Unix epoch integers
- Enhanced message support: quotes (quote_id, quote_author_uuid, quote_text), reactions (reaction_emoji, reaction_target_author), message types (regular, quote, reaction)
- Authentication system: Separate `auth_users` table for web authentication (distinct from Signal users), `sessions` table for persistent login state
- Schema applied on startup via `db.Init()` reading `schema.sql`
- Automatic migration system adds missing columns to existing tables
- Database stored in mounted `./data` directory for persistence

## Common Tasks
- **Add new endpoints**: Extend `internal/api/server.go` with new handlers
- **Rotate encryption key**: Not supported by the service; perform offline rotation if needed
- **Modify AI prompts**: Update summarization logic in `internal/ai/client.go`
- **Change message filtering**: Modify `SaveMessage` logic in `internal/database/db.go`
- **Adjust scheduling**: Update interval parsing and ticker logic in scheduler
- **Frontend changes**: Modify components in `web/src/components/` and pages in `web/src/app/`
- **Authentication changes**: Modify auth handlers in `internal/api/auth.go` and auth components in `web/src/components/auth/`

## Frontend Development
The Next.js 15 frontend is located in the `web/` directory and features:
- **shadcn/ui components**: Modern UI components for date pickers, filters, and summaries
- **Authentication system**: Built-in login/logout functionality with protected routes using React Context
- **Default date filter**: "Today" preset for summary filtering
- **Responsive design**: Works on desktop and mobile devices
- **TypeScript**: Fully typed components and API interactions
- **Available scripts**: `npm run lint`, `npm test`, `npm run build` (conditional in CI)

## Dependencies
- `github.com/mattn/go-sqlite3`: SQLCipher-enabled SQLite driver (replaces modernc.org/sqlite for encryption support)
- `github.com/sashabaranov/go-openai`: OpenAI API client
- `github.com/coder/websocket`: WebSocket client for Signal API
- `github.com/alexedwards/scs/v2`: Session management library
- `github.com/alexedwards/scs/sqlite3store`: SQLite session store for SCS
- `golang.org/x/crypto`: Cryptographic functions including bcrypt for password hashing
- **SQLCipher library**: Native encryption library (installed via package manager)
