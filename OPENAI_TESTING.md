# OpenAI Backend Testing Guide

This guide shows how to test the OpenAI backend for Summarizarr.

## Prerequisites

1. OpenAI API key from https://platform.openai.com/api-keys
2. Go installed (for standalone testing)

## Configuration

### Environment Variables

Set these environment variables for OpenAI backend:

```bash
export AI_BACKEND=openai
export OPENAI_API_KEY=sk-your-actual-api-key-here
export OPENAI_MODEL=gpt-4o  # Optional, defaults to gpt-4o
```

### Alternative Models

You can use other OpenAI models:
- `gpt-4o` (recommended, latest and fastest)
- `gpt-4` (more capable but slower/expensive)
- `gpt-3.5-turbo` (fastest and cheapest)

## Testing Methods

### 1. Standalone Test Tool

Build and run the standalone test:

```bash
# Build the test tool
go build -o test-backend cmd/testing/main.go

# Test OpenAI backend
OPENAI_API_KEY=sk-your-key-here ./test-backend openai

# Test local backend (for comparison)
LOCAL_MODEL=llama3.2:1b ./test-backend local
```

### 2. Production Backend Test

Update your `.env` file or docker-compose environment:

```bash
AI_BACKEND=openai
OPENAI_API_KEY=sk-your-actual-api-key-here
OPENAI_MODEL=gpt-4o
```

Then restart the backend:

```bash
docker compose restart summarizarr-backend
```

Check the logs for OpenAI initialization:

```bash
docker compose logs -f summarizarr-backend
```

You should see:
- "Using OpenAI backend with model: gpt-4o"
- "Testing OpenAI backend connectivity..."
- "OpenAI backend is ready"

### 3. Web Interface Test

1. Open http://localhost:3000
2. Check that the default date filter is "Today" (not "All time")
3. Select different date ranges and verify filtering works
4. Create test summaries to ensure OpenAI integration works

## Troubleshooting

### Common Issues

1. **"OPENAI_API_KEY is required"**
   - Make sure you've set the environment variable
   - Verify the key starts with `sk-`

2. **"OpenAI backend test failed: unauthorized"**
   - Check your API key is valid and not expired
   - Ensure you have credits in your OpenAI account

3. **"OpenAI backend test failed: timeout"**
   - Check your internet connection
   - Try a faster model like `gpt-3.5-turbo`

4. **"backend is still looking to initialize ollama"**
   - This was the original bug - should be fixed now
   - Ensure `AI_BACKEND=openai` is properly set
   - Restart the backend service

### Debugging

Enable debug logging:

```bash
LOG_LEVEL=DEBUG docker compose restart summarizarr-backend
docker compose logs -f summarizarr-backend
```

## Expected Behavior

When working correctly:

1. **Startup**: Backend shows "Using OpenAI backend" instead of Ollama initialization
2. **Testing**: Simple test message gets summarized successfully
3. **Web Interface**: Default filter is "Today", date filtering works properly
4. **Performance**: OpenAI responses should be faster than local Ollama

## Cost Considerations

- GPT-4o: ~$2.50 per 1M input tokens, $10 per 1M output tokens
- GPT-3.5-turbo: ~$0.50 per 1M input tokens, $1.50 per 1M output tokens
- Typical summary: 100-500 tokens input, 50-200 tokens output
- Daily cost for active chat groups: $0.01-$0.10 depending on volume

## Switching Back to Local

To switch back to Ollama:

```bash
export AI_BACKEND=local
export LOCAL_MODEL=llama3.2:1b
docker compose restart summarizarr-backend
```
