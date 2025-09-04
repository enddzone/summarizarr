# Repository Guidelines

## Project Structure & Module Organization
- `cmd/summarizarr/`: Go entrypoint (`main.go`) and integration tests.
- `internal/`: Backend packages (`api/`, `ai/`, `database/`, `auth/`, `frontend/` embed, provider clients).
- `web/`: Next.js + TypeScript UI (`src/components/`, `src/__tests__/`, `public/`).
- `data/`: Local dev database (do not commit secrets).
- `signal-cli-config/`: Local Signal CLI config for dev.
- Root: `Makefile`, `compose*.yaml`, `Dockerfile`, `.env.example`, `schema.sql`.

## Build, Test, and Development Commands
- `make all`: Start full dev stack (Signal, backend, web).
- `make backend`: Run backend only (SQLCipher, listens on `:8081`).
- `make frontend`: Start Next.js dev server (hot reload at `:3000`).
- `make test`: Run Go + Web tests; uses SQLCipher if available.
- `make test-backend` / `make test-frontend`: Run tests per side.
- `make build`: Production-like build; embeds UI (Go tags `sqlite_crypt libsqlite3`).
- `make docker` / `make prod`: Dev Docker stack / example prod stack.

## Coding Style & Naming Conventions
- Go: `gofmt` formatting; vet with `go vet` or `golangci-lint`. Exported `CamelCase`, unexported `camelCase`, packages lowercase (no underscores). Tests in same package, named `*_test.go`.
- TypeScript/React: ESLint config at `web/eslint.config.mjs`. Components `PascalCase` under `web/src/components/` with files like `my-component.tsx`. Hooks in `web/src/hooks/` as `use-*.ts`.
- Indentation: Go (tabs), TS/JS (2 spaces). Prefer small, pure modules.

## Testing Guidelines
- Go: standard `testing`; table-driven where possible. Prefer isolated DB tests using temp files. Run with `go test ./...` or `make test-backend`.
- Web: Jest + React Testing Library. Place tests in `web/src/__tests__/` or alongside components as `*.test.tsx`. Coverage: `npm run test:coverage` (from `web/`). Avoid flaky timers/network.

## Commit & Pull Request Guidelines
- Commits: concise, imperative subject (≤72 chars). Examples: `feat(api): add summaries filter`, `chore(npm)(deps): bump pkg in /web`.
- PRs: include summary, rationale, and testing notes. Link issues (e.g., `Closes #123`). For UI changes, attach before/after screenshots. Keep diffs focused.

## Security & Configuration Tips
- Never commit secrets. Copy `.env.example` to `.env` for local dev.
- SQLCipher is mandatory; run `make dev-setup` and follow README notes. In prod, provide 32-byte key via Docker secret.
- Required env: `SIGNAL_PHONE_NUMBER`, AI provider vars. Backend listens on `LISTEN_ADDR` (default `:8081`).

