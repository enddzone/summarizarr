# Summarizarr
Signal message summarizer that connects to Signal groups via WebSocket, stores messages in SQLite, and generates periodic AI summaries using local AI models via Ollama or cloud models via OpenAI-compatible APIs.

## Features

- **Privacy-First**: Default local AI processing using Ollama - no data sent to external APIs
- **Cloud AI Support**: Optional OpenAI-compatible API support for cloud models
- **User Anonymization**: User and group IDs are anonymized in prompts sent to LLMs, with real names substituted in final summaries
- **Automatic Setup**: Downloads and manages Ollama and AI models automatically
- **Signal Integration**: Connects to Signal groups via signal-cli-rest-api
- **Periodic Summaries**: Configurable interval for generating conversation summaries
- **REST API**: Query generated summaries via HTTP API

## Setup

1. Copy the environment variables example:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and set your values:
   - `SIGNAL_PHONE_NUMBER`: Your Signal phone number with country code, e.g., `+1234567890` (required)
   - `SUMMARIZATION_INTERVAL`: How often to generate summaries (default: 12h)
   - `LOCAL_MODEL`: AI model to use (default: phi3)
   - `LOG_LEVEL`: Log level (default: INFO)

3. Configure Signal CLI data in `signal-cli-config/` volume

4. Run with Docker Compose:
   ```bash
   docker-compose up --build
   ```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SIGNAL_PHONE_NUMBER` | Yes | - | Signal phone number with country code (e.g., +1234567890) |
| `SUMMARIZATION_INTERVAL` | No | 12h | How often to generate summaries (e.g., 30m, 1h, 6h, 12h, 24h) |
| `AI_BACKEND` | No | local | AI backend to use: 'local' (Ollama) or 'openai' (OpenAI-compatible API) |
| `LOCAL_MODEL` | No | llama3.2:1b | Local AI model name (for Ollama backend) |
| `OPENAI_API_KEY` | No | - | API key for OpenAI-compatible services (required when AI_BACKEND=openai) |
| `OPENAI_MODEL` | No | gpt-4o | Model name for OpenAI-compatible APIs |
| `OLLAMA_AUTO_DOWNLOAD` | No | true | Automatically download and start Ollama |
| `OLLAMA_HOST` | No | 127.0.0.1:11434 | Ollama server host and port |
| `OLLAMA_KEEP_ALIVE` | No | 5m | How long to keep models loaded in memory |
| `MODELS_PATH` | No | ./models | Directory to store downloaded models |
| `LOG_LEVEL` | No | INFO | Log level (DEBUG, INFO, WARN, ERROR) |

## AI Backend Options

### Local AI (Default - Recommended for Privacy)
Summarizarr uses [Ollama](https://ollama.ai/) to run AI models locally for maximum privacy. The default model is **llama3.2:1b** (1.3GB), which provides good summarization quality while being lightweight.

#### Supported Local Models
- `llama3.2:1b` - Meta Llama 3.2 1B (1.3GB, recommended)
- `phi3` - Microsoft Phi-3-Mini (2.3GB)
- `llama2` - Meta Llama 2 (3.8GB)
- `mistral` - Mistral 7B (4.1GB)
- `codellama` - Code Llama (3.8GB)

#### Model Management
- Models are automatically downloaded on first use
- Downloaded models are cached in the `./models` directory
- Models stay loaded in memory for the duration specified by `OLLAMA_KEEP_ALIVE`

### Cloud AI (OpenAI-Compatible APIs)
For users who prefer cloud models, Summarizarr supports any OpenAI-compatible API, including:
- OpenAI (ChatGPT, GPT-4, etc.)
- Anthropic Claude (via OpenAI-compatible endpoints)
- Local LLM servers (text-generation-webui, LocalAI, etc.)

#### Privacy Features for Cloud AI
When using cloud models, Summarizarr implements anonymization:
- **User Anonymization**: User names are replaced with `user_123` IDs in prompts sent to the LLM
- **Group Anonymization**: Group names are not included in prompts sent to the LLM
- **Post-Processing**: Real names are substituted back into the final summary after processing
- **No Metadata**: Only conversation text is sent, no phone numbers or personal identifiers

#### Configuration for Cloud AI
```bash
AI_BACKEND=openai
OPENAI_API_KEY=your_api_key_here
OPENAI_MODEL=gpt-4o
```

## API Endpoints

- `GET /summaries` - Get all generated summaries

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Signal CLI    │────│   Summarizarr    │────│     Ollama      │
│   REST API      │    │     Server       │    │   (Local AI)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
        │                        │                        │
        │                        │                        │
        ▼                        ▼                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Signal        │    │     SQLite       │    │    AI Models    │
│   Messages      │    │    Database      │    │   (phi3, etc.)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Privacy

- **No External APIs**: All AI processing happens locally using Ollama
- **No Data Transmission**: Signal messages never leave your infrastructure
- **Self-Contained**: Complete solution runs in Docker containers
- **Model Storage**: AI models are downloaded and cached locally
