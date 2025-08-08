## Summarizarr

AI-powered Signal message summarizer with a clean web UI. Summarizarr ingests Signal group messages, stores them in SQLite, and produces periodic summaries using either a local LLM (Ollama) or a cloud model (OpenAI).

## 🚀 Features

### Core Capabilities
- Signal integration via signal-cli-rest-api (WebSocket subscription)
- Local or cloud AI backends (Ollama or OpenAI)
- Anonymization before AI calls; names restored post-processing
- Scheduled summarization at configurable intervals
- Multiple Signal groups supported

### Modern Web Interface
- Built with Next.js 15
- Timeline and cards view
- Advanced filtering (multi-group, text, date range). Default date filter is "Today"
- Export (JSON/CSV/PDF)
- Signal setup wizard (QR-based)

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
docker compose up -d
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

By default, the backend uses the local Ollama path with model `llama3.2:1b`. To switch to OpenAI, set environment variables (see Configuration). A detailed OpenAI test guide is in `OPENAI_TESTING.md`.

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_BACKEND` | `local` | AI backend: `local` (Ollama) or `openai` |
| `LOCAL_MODEL` | `llama3.2:1b` | Ollama model name |
| `OLLAMA_HOST` | `127.0.0.1:11434` | Ollama server address |
| `OPENAI_API_KEY` | - | OpenAI API key (required when `AI_BACKEND=openai`) |
| `OPENAI_MODEL` | `gpt-4o` | OpenAI model name |
| `SIGNAL_PHONE_NUMBER` | - | Phone number for Signal registration |
| `DATABASE_PATH` | `/app/data/summarizarr.db` | SQLite database location |
| `SUMMARIZATION_INTERVAL` | `12h` | How often to generate summaries (e.g., 30m, 1h, 6h) |
| `LOG_LEVEL` | `INFO` | Logging level |
| `MODELS_PATH` | `./models` | Directory for Ollama models |

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

# Run locally (reads .env)
go run cmd/summarizarr/main.go

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

# Optional: Integration via Docker
docker compose up --build -d
```

## 📚 API (Backend)

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
docker compose --profile production up -d
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
   docker compose restart signal-cli-rest-api
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
   docker compose restart summarizarr-backend
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

See also: `OPENAI_TESTING.md` for OpenAI setup and validation.
