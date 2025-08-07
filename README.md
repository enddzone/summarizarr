## Summarizarr

A modern AI-powered Signal message summarizer with a comprehensive web interface. Summarizarr automatically processes Signal group messages and generates intelligent summaries using local (Ollama) or cloud-based (OpenAI-compatible) AI models.

## 🚀 Features

### Core Capabilities
- **Signal Integration**: Direct integration with Signal Messenger via REST API
- **AI-Powered Summaries**: Support for both local (Ollama) and cloud (OpenAI-compatible) AI backends
- **Data Anonymization**: Automatic removal of sensitive information before AI processing
- **Automated Processing**: Configurable scheduled summarization with customizable intervals
- **Multi-Group Support**: Process messages from multiple Signal groups simultaneously

### Modern Web Interface
- **🎨 Modern UI/UX**: Built with Next.js 15, featuring dark/light mode and responsive design
- **📊 Timeline & Cards View**: Switch between detailed timeline and compact card layouts
- **🔍 Advanced Filtering**: Multi-select group filters, date ranges, and full-text search
- **📤 Export Options**: Export summaries in JSON, CSV, and PDF formats
- **🔧 Signal Setup Wizard**: Easy QR code-based Signal registration
- **⚡ Real-time Updates**: Live summary updates and connection monitoring

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Signal CLI    │    │   Summarizarr    │    │    Web UI       │
│   REST API      │◄───┤    Backend       │◄───┤   (Next.js)     │
│   Port 8080     │    │    Port 8081     │    │   Port 3000     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       
         │              ┌────────▼────────┐              
         │              │   AI Backends   │              
         │              │                 │              
         │              │  Ollama (Local) │              
         │              │       OR        │              
         └──────────────┤ OpenAI (Cloud)  │              
                        └─────────────────┘              
```

## 🚀 Quick Start

### Prerequisites
- Docker and Docker Compose
- Signal account for registration

### 1. Clone and Start
```bash
git clone <repository-url>
cd summarizarr
docker-compose up -d
```

### 2. Access the Application
- **Web Interface**: [http://localhost:3000](http://localhost:3000)
- **Backend API**: [http://localhost:8081](http://localhost:8081)
- **Signal CLI**: [http://localhost:8080](http://localhost:8080)

### 3. Configure Signal
1. Open the web interface at [http://localhost:3000](http://localhost:3000)
2. Click "Setup Signal" to access the configuration wizard
3. Follow the QR code registration process
4. Verify your phone number

### 4. Configure AI Backend

#### Option A: Local AI (Ollama)
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull a model (e.g., llama2)
ollama pull llama2

# Ensure Ollama is running
ollama serve
```

#### Option B: Cloud AI (OpenAI)
Set your OpenAI API key in `docker-compose.yml`:
```yaml
environment:
  - AI_BACKEND=openai
  - OPENAI_API_KEY=your_api_key_here
```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_BACKEND` | `local` | AI backend: `local` (Ollama) or `openai` |
| `OLLAMA_HOST` | `127.0.0.1:11434` | Ollama server address |
| `OPENAI_API_KEY` | - | OpenAI API key (required for cloud AI) |
| `SIGNAL_PHONE_NUMBER` | - | Phone number for Signal registration |
| `DATABASE_PATH` | `/app/data/summarizarr.db` | SQLite database location |
| `SUMMARIZATION_INTERVAL` | `1h` | How often to generate summaries |
| `LOG_LEVEL` | `INFO` | Logging level |

### Customizing Summarization
Edit `docker-compose.yml` to adjust:
- **Frequency**: Change `SUMMARIZATION_INTERVAL` (e.g., `30m`, `2h`, `1d`)
- **AI Model**: Specify different Ollama models or OpenAI models
- **Anonymization**: Toggle data anonymization features

## 🏃‍♂️ Development

### Backend Development
```bash
# Install Go dependencies
go mod download

# Run locally
go run ./cmd/summarizarr

# Build
go build -o summarizarr ./cmd/summarizarr
```

### Frontend Development
```bash
cd web

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

### Testing
```bash
# Backend tests
go test ./...

# Frontend tests
cd web && npm test

# Integration tests
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

## 📚 API Documentation

### Backend Endpoints
- `GET /api/summaries` - Fetch summaries with optional filters
- `GET /api/groups` - List available Signal groups
- `GET /api/export` - Export summaries in various formats
- `GET /api/signal/config` - Signal configuration status
- `POST /api/signal/register` - Register Signal account

### Query Parameters
- `groups`: Filter by group IDs (comma-separated)
- `start_time`: Start date (ISO 8601)
- `end_time`: End date (ISO 8601)
- `search`: Full-text search query
- `sort`: Sort order (`newest` or `oldest`)
- `format`: Export format (`json`, `csv`, `pdf`)

## 🔐 Privacy & Security

### Data Anonymization
Summarizarr automatically anonymizes data before sending to AI services:
- **Phone Numbers**: Replaced with generic identifiers
- **Names**: Replaced with role-based placeholders
- **Personal Information**: Stripped from message content
- **Group Names**: Anonymized while preserving context

### Local Data Storage
- Messages stored locally in SQLite database
- Configurable data retention policies
- No data sent to external services without anonymization

## 🐳 Production Deployment

### With Nginx (Recommended)
```bash
# Start with production profile
docker-compose --profile production up -d
```

This includes:
- Nginx reverse proxy
- SSL termination
- Static file serving
- Load balancing

### Manual Deployment
```bash
# Build and deploy individual services
docker build -t summarizarr-backend .
docker build -t summarizarr-frontend ./web

# Run with custom configuration
docker run -d 
  -p 8081:8081 
  -v $(pwd)/data:/app/data 
  -e AI_BACKEND=openai 
  -e OPENAI_API_KEY=your_key 
  summarizarr-backend
```

## 🛠️ Troubleshooting

### Common Issues

1. **Signal Connection Failed**
   ```bash
   # Check Signal CLI status
   docker logs summarizarr-signal-cli-rest-api-1
   
   # Restart Signal service
   docker-compose restart signal-cli-rest-api
   ```

2. **AI Backend Not Responding**
   ```bash
   # For Ollama
   curl http://localhost:11434/api/version
   
   # Check backend logs
   docker logs summarizarr-summarizarr-backend-1
   ```

3. **Frontend Build Issues**
   ```bash
   # Clear build cache
   cd web && rm -rf .next node_modules
   npm install && npm run build
   ```

4. **Database Issues**
   ```bash
   # Check database permissions
   ls -la data/
   
   # Reset database (⚠️ deletes all data)
   rm data/summarizarr.db
   docker-compose restart summarizarr-backend
   ```

### Performance Optimization

1. **Increase summarization frequency** for more responsive updates
2. **Use local AI** for faster processing and privacy
3. **Enable database indexes** for large message volumes
4. **Configure log rotation** to manage disk space

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit changes: `git commit -m 'Add amazing feature'`
4. Push to branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Development Guidelines
- Follow Go and TypeScript best practices
- Add tests for new features
- Update documentation
- Ensure Docker builds succeed

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Signal CLI REST API](https://github.com/bbernhard/signal-cli-rest-api) for Signal integration
- [Ollama](https://ollama.ai/) for local AI capabilities
- [Next.js](https://nextjs.org/) for the modern web framework
- [Shadcn/ui](https://ui.shadcn.com/) for beautiful UI components

---

**Made with ❤️ for the privacy-conscious community**Summarizarr
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
