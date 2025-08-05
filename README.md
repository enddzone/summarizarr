# Summarizarr
Signal message summarizer that connects to Signal groups via WebSocket, stores messages in SQLite, and generates periodic AI summaries using OpenAI's API.

## Setup

1. Copy the environment variables example:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and set your values:
   - `OPENAI_API_KEY`: Your OpenAI API key (required)
   - `SIGNAL_PHONE_NUMBER`: Your Signal phone number with country code, e.g., `+1234567890` (required)
   - `SUMMARIZATION_INTERVAL`: How often to generate summaries (default: 12h)
   - `OPENAI_MODEL`: OpenAI model to use (default: gpt-4o)
   - `LOG_LEVEL`: Log level (default: INFO)

3. Configure Signal CLI data in `signal-cli-config/` volume

4. Run with Docker Compose:
   ```bash
   docker-compose up --build
   ```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OPENAI_API_KEY` | Yes | - | OpenAI API key for generating summaries |
| `SIGNAL_PHONE_NUMBER` | Yes | - | Signal phone number with country code (e.g., +1234567890) |
| `SUMMARIZATION_INTERVAL` | No | 12h | How often to generate summaries (e.g., 30m, 1h, 6h, 12h, 24h) |
| `OPENAI_MODEL` | No | gpt-4o | OpenAI model to use for summaries |
| `LOG_LEVEL` | No | INFO | Log level (DEBUG, INFO, WARN, ERROR) |

## API Endpoints

- `GET /summaries` - Get all generated summaries
