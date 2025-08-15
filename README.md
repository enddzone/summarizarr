## Summarizarr

[![CI](https://github.com/enddzone/summarizarr/actions/workflows/ci.yml/badge.svg)](https://github.com/enddzone/summarizarr/actions/workflows/ci.yml)
[![Container](https://ghcr.io/enddzone/summarizarr/badge.svg)](https://ghcr.io/enddzone/summarizarr)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

AI-powered Signal message summarizer delivered as a lightweight, single-container application. Summarizarr ingests Signal group messages, stores them in SQLite, and produces periodic summaries using multiple AI providers including local LLMs (Ollama) and cloud services (OpenAI, Groq, Gemini, Claude).

## ✨ Key Highlights

- **Single Container**: ~57MB unified container with embedded frontend
- **Multi-Provider AI**: Local (Ollama), OpenAI, Groq, Gemini, Claude support
- **Signal Integration**: WebSocket connection to signal-cli-rest-api
- **Privacy-First**: Automatic data anonymization before AI processing
- **Modern UI**: Responsive Next.js interface with advanced filtering
- **Production Ready**: Health checks, semantic versioning, automated CI/CD

## 🚀 Features

### Container & Deployment
- **Lightweight**: Sub-60MB Alpine-based container
- **Single Port**: Unified HTTP server on port 8080
- **Health Checks**: Built-in endpoint monitoring
- **Multi-Arch**: Supports AMD64 and ARM64
- **Security**: Non-root user, vulnerability scanning

### AI & Processing
- **Multi-Provider Support**: 5 AI backends with unified interface
- **Smart Anonymization**: Names and data stripped before AI calls
- **Scheduled Processing**: Configurable summarization intervals
- **Local AI**: Self-hosted Ollama for privacy
- **Cloud AI**: OpenAI, Groq with native compatibility

### Web Interface
- **Embedded Frontend**: No separate frontend deployment needed
- **Advanced Filtering**: Multi-group, date range, text search
- **Export Options**: JSON, CSV, PDF formats
- **Responsive Design**: Mobile-optimized interface
- **Real-time Updates**: Live summary refresh

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────────────────────────┐
│   Signal CLI    │    │          Summarizarr                │
│   REST API      │◄───┤    (Single Container)               │
│   Port 8080     │    │                                     │
└─────────────────┘    │  ┌─────────────┐  ┌────────────────┐ │
                       │  │ Go Backend  │  │ Embedded       │ │
         ┌──────────────┼──┤ Port 8080   │  │ Next.js        │ │
         │              │  │ (API + UI)  │  │ Frontend       │ │
         │              │  └─────────────┘  └────────────────┘ │
         │              └──────────────────────────────────────┘
         │                       │                       
         │              ┌────────▼────────┐              
         │              │   AI Backends   │              
         │              │                 │              
         │              │ • Ollama (Local)│
         │              │ • OpenAI        │
         └──────────────┤ • Groq          │
                        │ • Gemini        │
                        │ • Claude        │
                        └─────────────────┘              
```

## 🚀 Quick Start

### Option 1: Docker Compose (Recommended)

```bash
# 1. Create docker-compose.yml
curl -O https://raw.githubusercontent.com/enddzone/summarizarr/main/compose.yaml

# 2. Create environment file
curl -O https://raw.githubusercontent.com/enddzone/summarizarr/main/.env.example
cp .env.example .env

# 3. Configure your Signal phone number
export SIGNAL_PHONE_NUMBER="+1234567890"  # Your number

# 4. Start services
docker compose up -d
```

### Option 2: Single Container

```bash
# Pull latest image
docker pull ghcr.io/enddzone/summarizarr:latest

# Run with minimal configuration
docker run -d \
  --name summarizarr \
  -p 8080:8080 \
  -e SIGNAL_PHONE_NUMBER="+1234567890" \
  -e AI_PROVIDER=local \
  -v summarizarr-data:/data \
  ghcr.io/enddzone/summarizarr:latest
```

### Access Points

After deployment, access your application:

- **Web Interface**: [http://localhost:8081](http://localhost:8081) (compose) or [http://localhost:8080](http://localhost:8080) (single)
- **Signal CLI**: [http://localhost:8080](http://localhost:8080) (compose only)
- **Health Check**: `/health` endpoint
- **API Documentation**: `/api/version` for version info

### Configure Signal

1. Open the web interface
2. Click "Setup Signal" to access the configuration wizard
3. Follow the QR code registration process
4. Verify your phone number

### AI Provider Setup

**Local AI (Default)**:
```bash
# No configuration needed - uses Ollama with llama3.2:1b
# Model downloads automatically on first use
```

**OpenAI**:
```bash
export AI_PROVIDER=openai
export OPENAI_API_KEY=sk-your-key-here
```

**Groq (Fast)**:
```bash
export AI_PROVIDER=groq
export GROQ_API_KEY=gsk-your-key-here
```

See [Configuration](#configuration) for all providers.

## 🔧 Configuration

### Environment Variables

#### Core Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `AI_PROVIDER` | `local` | AI provider: `local`, `openai`, `groq`, `gemini`, `claude` |
| `SIGNAL_PHONE_NUMBER` | - | Phone number for Signal registration |
| `DATABASE_PATH` | `/app/data/summarizarr.db` | SQLite database location |
| `SUMMARIZATION_INTERVAL` | `12h` | How often to generate summaries (e.g., 30m, 1h, 6h) |
| `LOG_LEVEL` | `INFO` | Logging level |

#### Local AI (Ollama) Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `LOCAL_MODEL` | `llama3.2:1b` | Ollama model name |
| `OLLAMA_HOST` | `127.0.0.1:11434` | Ollama server address |
| `MODELS_PATH` | `./models` | Directory for Ollama models |

#### OpenAI Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `OPENAI_API_KEY` | - | OpenAI API key (required when `AI_PROVIDER=openai`) |
| `OPENAI_MODEL` | `gpt-4o` | OpenAI model name |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | OpenAI API base URL |

#### Groq Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `GROQ_API_KEY` | - | Groq API key (required when `AI_PROVIDER=groq`) |
| `GROQ_MODEL` | `llama3-8b-8192` | Groq model name |
| `GROQ_BASE_URL` | `https://api.groq.com/openai/v1` | Groq API base URL |

#### Gemini Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `GEMINI_API_KEY` | - | Gemini API key (required when `AI_PROVIDER=gemini`) |
| `GEMINI_MODEL` | `gemini-2.0-flash` | Gemini model name |
| `GEMINI_BASE_URL` | `http://localhost:8000/hf/v1` | Gemini proxy base URL |

#### Claude Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `CLAUDE_API_KEY` | - | Claude API key (required when `AI_PROVIDER=claude`) |
| `CLAUDE_MODEL` | `claude-3-sonnet` | Claude model name |
| `CLAUDE_BASE_URL` | `http://localhost:8000/openai/v1` | Claude proxy base URL |

### Multi-Provider Setup Examples

#### OpenAI (Default Cloud Provider)
```env
AI_PROVIDER=openai
OPENAI_API_KEY=sk-proj-xxxxx
OPENAI_MODEL=gpt-4o
# OPENAI_BASE_URL defaults to https://api.openai.com/v1
```

#### Groq (Fast Inference)
```env
AI_PROVIDER=groq
GROQ_API_KEY=gsk_xxxxx
GROQ_MODEL=llama3-8b-8192
# GROQ_BASE_URL defaults to https://api.groq.com/openai/v1
```

#### Local Ollama (Default)
```env
AI_PROVIDER=local
LOCAL_MODEL=llama3.2:1b
OLLAMA_HOST=127.0.0.1:11434
```

#### Gemini (Requires Proxy)
```env
AI_PROVIDER=gemini
GEMINI_API_KEY=xxxxx
GEMINI_MODEL=gemini-2.0-flash
GEMINI_BASE_URL=http://localhost:8000/hf/v1  # Gemini Balance proxy
```

#### Claude (Requires Proxy)
```env
AI_PROVIDER=claude
CLAUDE_API_KEY=sk-ant-xxxxx
CLAUDE_MODEL=claude-3-sonnet
CLAUDE_BASE_URL=http://localhost:8000/openai/v1  # OpenAI-compatible proxy
```

#### Setting Up Proxy Services
For Gemini and Claude, you'll need OpenAI-compatible proxy services:

**Gemini Balance Proxy**:
```bash
# Install and run Gemini Balance (example)
npm install -g @google-ai/gemini-balance
gemini-balance --port 8000 --api-key YOUR_GEMINI_KEY
```

**Claude Proxy**:
```bash
# Use community proxy services or set up your own
# Example: anthropic-openai-bridge
docker run -p 8000:8000 -e ANTHROPIC_API_KEY=your_key anthropic-proxy
```

### Customizing Summarization
Edit `docker-compose.yml` to adjust:
- **Frequency**: Change `SUMMARIZATION_INTERVAL` (e.g., `30m`, `2h`, `1d`)
- **AI Provider**: Switch between different providers and models
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

## 📚 API Reference

### Core Endpoints
- `GET /` - Web interface (embedded frontend)
- `GET /health` - Health check for containers
- `GET /api/version` - Version and build information

### Data Endpoints
- `GET /api/summaries` - Fetch summaries with optional filters
- `DELETE /api/summaries/{id}` - Delete specific summary
- `GET /api/groups` - List available Signal groups
- `GET /api/export` - Export summaries in various formats
- `GET /api/signal/config` - Signal configuration status

### Query Parameters
- `groups`: Filter by group IDs (comma-separated)
- `start_time`: Start date (ISO 8601)
- `end_time`: End date (ISO 8601)
- `search`: Full-text search query
- `format`: Export format (`json`, `csv`)

### Health Check Response
```json
{
  "status": "healthy",
  "timestamp": 1703123456,
  "service": "summarizarr"
}
```

### Version Response
```json
{
  "version": "v0.1.0",
  "git_commit": "abc123def456",
  "build_time": "2024-01-01T12:00:00Z",
  "go_version": "go1.24.0"
}
```

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

### Container Registry
```bash
# Pull from GitHub Container Registry
docker pull ghcr.io/enddzone/summarizarr:latest

# Or specific version
docker pull ghcr.io/enddzone/summarizarr:v0.1.0
```

### Docker Compose (Recommended)
```bash
# Production deployment with all services
docker compose up -d

# With specific profiles
docker compose --profile production up -d
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: summarizarr
spec:
  replicas: 1
  selector:
    matchLabels:
      app: summarizarr
  template:
    metadata:
      labels:
        app: summarizarr
    spec:
      containers:
      - name: summarizarr
        image: ghcr.io/enddzone/summarizarr:latest
        ports:
        - containerPort: 8080
        env:
        - name: SIGNAL_PHONE_NUMBER
          value: "+1234567890"
        - name: AI_PROVIDER
          value: "openai"
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: summarizarr-secrets
              key: openai-api-key
        volumeMounts:
        - name: data
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: summarizarr-data
```

### Environment-Specific Configurations

**Development**:
```bash
docker compose -f compose.yaml -f compose.dev.yaml up
```

**Production with SSL**:
```bash
# Add reverse proxy (nginx, traefik, etc.)
# Handle SSL termination
# Configure monitoring
```

### Monitoring & Observability
```bash
# Health check endpoint
curl http://localhost:8080/health

# Version information
curl http://localhost:8080/api/version

# Container logs
docker logs summarizarr

# Resource usage
docker stats summarizarr
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

## 🚀 Release Information

### Versioning
Summarizarr follows [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features, backward compatible  
- **PATCH**: Bug fixes, improvements

### Container Images
- **Registry**: `ghcr.io/enddzone/summarizarr`
- **Tags**: `latest`, `v1.x.x`, `sha-<commit>`
- **Architectures**: linux/amd64, linux/arm64
- **Size**: ~57MB (Alpine-based)

### Automated Releases
- CI/CD via GitHub Actions
- Automatic image builds on version tags
- Security scanning with Trivy
- SLSA provenance attestation
- Multi-architecture builds

### Installation Methods
1. **Docker Compose**: Production-ready with all dependencies
2. **Single Container**: Minimal deployment
3. **Kubernetes**: Enterprise container orchestration
4. **Direct Download**: GitHub releases with assets

## 🙏 Acknowledgments

- [Signal CLI REST API](https://github.com/bbernhard/signal-cli-rest-api) for Signal integration
- [Ollama](https://ollama.ai/) for local AI capabilities
- [Next.js](https://nextjs.org/) for the modern web framework
- [Shadcn/ui](https://ui.shadcn.com/) for beautiful UI components
- [Docker](https://docker.com/) for containerization
- [GitHub Actions](https://github.com/features/actions) for CI/CD automation

---

**Container Distribution**: Get started with a single command  
**Multi-Platform Support**: Runs on Intel and ARM architectures  
**Production Ready**: Health checks, monitoring, and security built-in
